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

// JiraSyncIssuesJob will sync all jira issues that have worklogs or have been updated so we have the supporting data
type JiraSyncIssuesJob struct {
	cfg        *jiraworklog.Config
	jira       jiraworklog.JiraReader
	repo       repository.Repo
	logger     *slog.Logger
	todaysHour int
}

func NewJJiraSyncIssuesJob(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo) *JiraSyncIssuesJob {
	return &JiraSyncIssuesJob{
		cfg:        cfg,
		jira:       jira,
		repo:       repo,
		todaysHour: 99,
	}
}

func (j *JiraSyncIssuesJob) GetName() string {
	return "JiraSyncIssuesJob"
}

func (j *JiraSyncIssuesJob) GetInterval() time.Duration {
	return time.Second * 31
}

func (j *JiraSyncIssuesJob) Run() error {
	//Grab all missing issues that have worklog ids
	missingIssues, err := j.repo.MissingIssues()
	if err != nil {
		return errors.Wrap(err, "error fetching missing jira issues")
	}
	slog.Info("fetching missing jira issues")

	jiraIds := []string{}
	cnt := 0
	for _, id := range missingIssues {

		jiraIds = append(jiraIds, strconv.Itoa(id))
		cnt++
		if cnt%50 == 0 {

			err = j.fetchAndSaveIssues(jiraIds)
			if err != nil {
				return err
			}
			cnt = 0
			jiraIds = jiraIds[:0]
		}
	}
	if len(jiraIds) > 0 {
		err = j.fetchAndSaveIssues(jiraIds)
		if err != nil {
			return err
		}
	}

	tz := j.jira.GetTimezone()
	lastUpdated := j.cfg.IssueLastTimestamp
	lastUpdatedDateOnly := internal.DateOnly(time.Unix(lastUpdated, 0)).In(tz).Unix()

	today := internal.DateOnly(time.Now().In(tz)).Unix()
	if lastUpdatedDateOnly < today {
		ts, te := internal.GetDateRange(lastUpdated, time.Now().UTC().Unix(), tz)
		//fmt.Println("callinng syncUpdatedIssues less than today")
		err = j.syncUpdatedIssues(ts, te, tz)
		if err != nil {
			return err
		}
		//fmt.Println("saving lasttimestamp")
		j.cfg.IssueLastTimestamp = te
		if err := j.cfg.Save(); err != nil {
			return errors.Wrap(err, "error saving config")
		}
		//lastUpdated = lastUpdated + 86400 //add day
	}

	//if caught up, for today sync once every 10 mins
	if lastUpdatedDateOnly == today {
		hour := time.Now().Hour()
		minute := time.Now().Minute()
		if j.todaysHour != hour || minute%10 == 0 {
			ts, te := internal.GetDateRange(lastUpdated, time.Now().UTC().Unix(), tz)
			err = j.syncUpdatedIssues(ts, te, tz)
			if err != nil {
				return err
			}
			j.todaysHour = hour
			err = j.repo.SyncProjectCharges()
			if err != nil {
				return err
			}
			j.cfg.IssueLastTimestamp = te
			if err := j.cfg.Save(); err != nil {
				return errors.Wrap(err, "error saving config")
			}
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

func (j *JiraSyncIssuesJob) syncUpdatedIssues(startDate, endDate int64, tz *time.Location) error {
	jiraIds := []string{}
	nextPageToken := ""
	start := internal.JiraDateString(startDate, tz)
	end := internal.JiraDateString(endDate, tz)
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

	slog.Warn("fetching updated jira issues", slog.String("start", start), slog.Int("count", len(jiraIds)), slog.String("end", end))
	if len(jiraIds) > 0 {
		err := j.fetchAndSaveIssues(jiraIds)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *JiraSyncIssuesJob) fetchIssueChangelog(issueId int) ([]types.ChangelogStatus, error) {
	changelog := []types.ChangelogStatus{}
	startAt := 0
	for {
		cl, err := j.jira.Changelog(issueId, startAt)
		if err != nil {
			return nil, err
		}
		startAt = cl.Total + startAt - 1
		changelog = append(changelog, types.ToChangelogStatus(cl, issueId)...)
		if cl.IsLast {
			break
		}
	}
	return changelog, nil
}
