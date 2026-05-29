package job

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/internal"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
)

// JiraSyncIssuesJob syncs all Jira issues that have worklogs or have been
// updated since the last successful run, so the local DB has supporting data
// for every worklog and dashboard query.
type JiraSyncIssuesJob struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	repo   repository.Repo
	logger *slog.Logger
}

func NewJJiraSyncIssuesJob(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo) *JiraSyncIssuesJob {
	return &JiraSyncIssuesJob{
		cfg:  cfg,
		jira: jira,
		repo: repo,
	}
}

func (j *JiraSyncIssuesJob) GetName() string {
	return "JiraSyncIssuesJob"
}

func (j *JiraSyncIssuesJob) GetInterval() time.Duration {
	return time.Minute * 5
}

// Maximum amount of Jira-update history fetched in a single Run. After a long
// outage we walk forward this much per run (one window per 5-minute tick)
// until the catch-up reaches `now`. Prevents a multi-week JQL response.
const syncMaxQueryWindow = 24 * time.Hour

// Backstop overlap on every query — clock skew between this host and Jira,
// and the minute-precision JQL filter, can leave sub-minute updates straddling
// the boundary. BulkInsertChangelogs uses ON CONFLICT DO NOTHING and
// UpdateIssue is an upsert, so re-fetching is harmless.
const syncQueryOverlap = 1 * time.Minute

// First-run / cold-start cap. If the on-disk lastTimestamp is missing or
// absurdly old (treat anything older than this as suspect), we sync only
// the trailing 30 days rather than asking Jira for years of history.
const syncMaxCatchup = 30 * 24 * time.Hour

func (j *JiraSyncIssuesJob) Run() error {
	// Get tz first — bulk fetch fails silently (200 + empty) on bad creds,
	// so this call doubles as a permission probe.
	tz, err := j.jira.GetTimezone()
	if err != nil {
		return errors.Wrap(err, "error getting jira timezone")
	}

	// Step 1: backfill issues referenced by worklogs but missing from the
	// issue table.
	if err := j.processMissingIssues(); err != nil {
		return err
	}

	// Step 2: pull the next chunk of updated issues from Jira. One simple
	// time window — no date-only juggling, no clock-time gating. Job
	// interval is the throttle.
	startTs := time.Unix(j.cfg.IssueLastTimestamp, 0)
	if j.cfg.IssueLastTimestamp == 0 || time.Since(startTs) > syncMaxCatchup {
		startTs = time.Now().Add(-syncMaxCatchup)
	}
	endTs := time.Now()
	if endTs.Sub(startTs) > syncMaxQueryWindow {
		endTs = startTs.Add(syncMaxQueryWindow)
	}
	startStr := internal.JiraDateString(startTs.Add(-syncQueryOverlap).Unix(), tz)
	endStr := internal.JiraDateString(endTs.Unix(), tz)
	slog.Info("syncing updated jira issues", "start", startStr, "end", endStr)
	if err := j.syncUpdatedIssuesByString(startStr, endStr); err != nil {
		return err
	}
	j.cfg.IssueLastTimestamp = endTs.Unix()
	if err := j.cfg.Save(); err != nil {
		return errors.Wrap(err, "error saving config")
	}

	// Step 3: keep project_charge table populated with any new charges
	// seen on the issues we just synced.
	if err := j.repo.SyncProjectCharges(); err != nil {
		return errors.Wrap(err, "error syncing project charges")
	}

	// Step 4: detect issues deleted in Jira (touched older than 60 days).
	return j.processDeletes(time.Hour * 1440)
}

// processMissingIssues fetches issues whose ID appears in worklog rows but
// has no corresponding row in the issue table. Batches of 50 so we don't
// send absurdly large bulk-fetch payloads.
func (j *JiraSyncIssuesJob) processMissingIssues() error {
	missingIssues, err := j.repo.MissingIssues()
	if err != nil {
		return errors.Wrap(err, "error fetching missing jira issues")
	}
	if len(missingIssues) == 0 {
		return nil
	}
	slog.Info("fetching missing jira issues", "count", len(missingIssues))

	jiraIds := make([]string, 0, 50)
	for _, id := range missingIssues {
		jiraIds = append(jiraIds, strconv.Itoa(id))
		if len(jiraIds) == 50 {
			if err := j.fetchAndSaveIssues(jiraIds); err != nil {
				return err
			}
			jiraIds = jiraIds[:0]
		}
	}
	if len(jiraIds) > 0 {
		if err := j.fetchAndSaveIssues(jiraIds); err != nil {
			return err
		}
	}
	return nil
}

func (j *JiraSyncIssuesJob) issueOK(issue types.StoredIssue) bool {
	for _, proj := range j.cfg.ExcludedProjects {
		if strings.EqualFold(proj, issue.Project) {
			return false
		}
	}
	return true
}

// This does a lot for issues. It will update the issue table, fetch the changelog for the issue and save that
func (j *JiraSyncIssuesJob) fetchAndSaveIssues(jiraIds []string) error {
	const batchSize = 50
	var issues []jiraworklog.Issue
	for i := 0; i < len(jiraIds); i += batchSize {
		end := min(i+batchSize, len(jiraIds))
		batch := jiraIds[i:end]
		fetched, err := jiraworklog.Retry(3, time.Second*10, func() ([]jiraworklog.Issue, error) {
			return j.jira.BulkFetchIssues(batch)
		})
		if err != nil {
			return errors.Wrap(err, "unknown error bulk fetching issues from jira")
		}
		issues = append(issues, fetched...)
	}
	for _, i := range issues {
		issue := types.ToDomain(i)
		if !j.issueOK(issue) {
			continue
		}
		//fmt.Printf("Issue: %d - %s issue %s    --  %s\n", issue.ID, issue.Key, issue.UpdateDate.String(), i.Fields.Updated)
		err := j.repo.UpdateIssue(&issue)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+issue.Key)
		}

		if issue.IsParent() {
			//sync chnagelog ONLY for Parent issues (only sub-tasks excluded..for now)
			changelog, err := jiraworklog.Retry(3, time.Second*10, func() ([]types.ChangelogStatus, error) {
				return j.fetchIssueChangelog(issue.ID)
			})
			if err != nil {
				return errors.Wrap(err, "error fetching changelogs for issue "+issue.Key)
			}
			//slog.Info("jira issue changelog", slog.String("key", issue.Key), slog.Int("records", len(changelog)))
			err = j.repo.BulkInsertChangelogs(changelog)
			if err != nil {
				return errors.Wrap(err, "error bulk inserting changelogs for issue "+issue.Key)
			}

			err = j.repo.RefreshStatusStints(issue.ID)
			if err != nil {
				return errors.Wrap(err, "error refreshing status stints for issue "+issue.Key)
			}
		}

	}
	return nil
}

// syncUpdatedIssuesByString queries Jira for every issue whose `updated`
// timestamp lies in [start, end) — both strings already formatted in Jira's
// tz by the caller — and refreshes the local DB for each. Paginates via
// nextPageToken.
func (j *JiraSyncIssuesJob) syncUpdatedIssuesByString(start, end string) error {
	jiraIds := []string{}
	nextPageToken := ""
	for {
		updatedIssues, err := j.jira.IssuesUpdated(start, end, nextPageToken)
		if err != nil {
			return errors.Wrap(err, "error fetching updated jira issues")
		}
		nextPageToken = updatedIssues.NextPageToken
		for _, issue := range updatedIssues.Issues {
			jiraIds = append(jiraIds, issue.ID)
		}
		if updatedIssues.IsLast {
			break
		}
	}
	if len(jiraIds) == 0 {
		return nil
	}
	slog.Info("found updated jira issues", "start", start, "end", end, "count", len(jiraIds))
	return j.fetchAndSaveIssues(jiraIds)
}

func (j *JiraSyncIssuesJob) fetchIssueChangelog(issueId int) ([]types.ChangelogStatus, error) {
	return fetchIssueChangelogPaged(j.jira, issueId)
}

func (j *JiraSyncIssuesJob) processDeletes(threshold time.Duration) error {
	lastSeenIssues, err := j.repo.LastSeenIssues(threshold)
	if err != nil {
		return errors.Wrap(err, "error fetching last seen jira issues")
	}
	slog.Info("processing potentially deleted issues", slog.Int("cnt", len(lastSeenIssues)))
	for _, id := range lastSeenIssues {
		_, err := j.jira.Issue(strconv.Itoa(id))
		if errors.Is(err, jiraworklog.ErrIssueNotFound) {
			err = j.repo.DeleteIssue(id)
			if err != nil {
				slog.Error("error deleting issue", slog.Int("id", id))
			}
		}
		if err == nil {
			err = j.repo.UpdateIssueLastSeen(id)
			if err != nil {
				slog.Error("error updating last seen for issue", slog.Int("id", id))
			}
		}
	}
	return nil
}
