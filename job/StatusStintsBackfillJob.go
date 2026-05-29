package job

import (
	"log/slog"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/pkg/errors"
)

// StatusStintsBackfillJob periodically reconciles issue.status against the
// status of each issue's most recent status_stints row. If they disagree
// AND the issue hasn't been touched in the last 10 minutes (a window
// chosen so the regular JiraSyncIssuesJob has had a fair chance to
// reconcile it first), we re-fetch the issue's changelog from Jira and
// rebuild its stints.
//
// This is the self-healing path for drift introduced by sync-time issues —
// network errors, pagination bugs (the changelog one was fixed but new
// ones could appear), Jira API hiccups, partial syncs, etc. The job is
// safe to run repeatedly: BulkInsertChangelogs uses ON CONFLICT DO NOTHING
// and RefreshStatusStints is idempotent (delete + reinsert per issue).
type StatusStintsBackfillJob struct {
	cfg  *jiraworklog.Config
	jira jiraworklog.JiraReader
	repo repository.Repo
}

const (
	// Issues updated within this window are excluded — the regular sync
	// is likely still working on them.
	statusStintsBackfillNotUpdatedSince = 10 * time.Minute

	// How often the job runs.
	statusStintsBackfillInterval = 10 * time.Minute

	// Per-run cap. Keeps Jira API load bounded if a large drift backlog
	// has accumulated (e.g., after a long outage). Subsequent runs will
	// chip away at the rest.
	statusStintsBackfillBatchSize = 50
)

func NewStatusStintsBackfillJob(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo) *StatusStintsBackfillJob {
	return &StatusStintsBackfillJob{
		cfg:  cfg,
		jira: jira,
		repo: repo,
	}
}

func (j *StatusStintsBackfillJob) GetName() string {
	return "StatusStintsBackfillJob"
}

func (j *StatusStintsBackfillJob) GetInterval() time.Duration {
	return statusStintsBackfillInterval
}

func (j *StatusStintsBackfillJob) Run() error {
	ids, err := j.repo.IssuesWithStaleStints(statusStintsBackfillNotUpdatedSince, statusStintsBackfillBatchSize)
	if err != nil {
		return errors.Wrap(err, "error finding issues with stale stints")
	}
	if len(ids) == 0 {
		return nil
	}

	slog.Info("backfilling stints for drifted issues", "count", len(ids))

	succeeded, failed := 0, 0
	for _, id := range ids {
		if err := j.refetchAndRefresh(id); err != nil {
			// Log and continue — one bad issue (deleted in Jira, permissions
			// changed, transient API error) shouldn't stop the rest.
			slog.Error("backfill failed for issue", "issueId", id, "error", err)
			failed++
			continue
		}
		succeeded++
	}
	slog.Info("backfill pass complete", "succeeded", succeeded, "failed", failed)
	return nil
}

func (j *StatusStintsBackfillJob) refetchAndRefresh(issueId int) error {
	transitions, err := fetchIssueChangelogPaged(j.jira, issueId)
	if err != nil {
		return errors.Wrap(err, "fetch changelog")
	}
	if err := j.repo.BulkInsertChangelogs(transitions); err != nil {
		return errors.Wrap(err, "bulk insert changelogs")
	}
	if err := j.repo.RefreshStatusStints(issueId); err != nil {
		return errors.Wrap(err, "refresh stints")
	}
	return nil
}
