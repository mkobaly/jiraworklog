package job

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
)

// JiraSyncWorklogsJob is a background job that will download all worklogs from Jira (updated or deleted)
type JiraSyncWorklogsJob struct {
	cfg  *jiraworklog.Config
	jira jiraworklog.JiraReader
	repo repository.Repo
	tz   *time.Location
	//logger *slog.Logger
}

func NewJiraSyncWorklogsJob(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo) *JiraSyncWorklogsJob {
	nyc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	return &JiraSyncWorklogsJob{
		cfg:  cfg,
		jira: jira,
		repo: repo,
		tz:   nyc,
		//logger: logger,
	}
}

func (j *JiraSyncWorklogsJob) GetName() string {
	return "JiraWorklogsDownloader"
}

func (j *JiraSyncWorklogsJob) GetInterval() time.Duration {
	return time.Second * 60
}

func (j *JiraSyncWorklogsJob) Run() error {
	lastUpdateTimestamp := j.cfg.WorklogUpdatedLastTimestamp
	lastDeletedTimestamp := j.cfg.WorklogDeletedLastTimestamp

	if lastUpdateTimestamp == 0 {
		// default going back 4 months
		lastUpdateTimestamp = time.Now().AddDate(0, -4, 0).UnixNano() / 1e6
	}
	if lastDeletedTimestamp == 0 {
		// default going back 4 months
		lastDeletedTimestamp = time.Now().AddDate(0, -4, 0).UnixNano() / 1e6
	}
	//maxWorklogID := j.cfg.MaxWorklogID
	slog.Info("last timestamp", slog.Int64("updated", lastUpdateTimestamp), slog.Int64("deleted", lastDeletedTimestamp))

	//Fetch worklogs updated since last timestamp check
	worklogsUpdated, err := j.jira.WorklogsUpdated(lastUpdateTimestamp)
	if err != nil {
		return errors.Wrap(err, "failed to fetch updated worklogs from jira")
	}
	slog.Info("worklogs updated stats", slog.Int64("since", worklogsUpdated.Since), slog.Int64("until", worklogsUpdated.Until),
		slog.Bool("lastPage", worklogsUpdated.LastPage), slog.Int("count", len(worklogsUpdated.Values)))

	worklogsDeleted, err := j.jira.WorklogsDeleted(lastDeletedTimestamp)
	if err != nil {
		return errors.Wrap(err, "failed to fetch deleted worklogs from jira")
	}
	slog.Info("worklogs deleted stats", slog.Int64("since", worklogsDeleted.Since), slog.Int64("until", worklogsDeleted.Until),
		slog.Bool("lastPage", worklogsDeleted.LastPage), slog.Int("count", len(worklogsDeleted.Values)))

	//For given worklog Ids we now need to get the worklog details
	var ids []int
	for _, w := range worklogsUpdated.Values {
		ids = append(ids, w.WorklogID)
	}

	if len(ids) == 0 && len(worklogsDeleted.Values) == 0 {
		return nil //nothing to do
	}

	if len(ids) > 0 {
		details, err := j.jira.WorklogDetails(ids)
		if err != nil {
			return errors.Wrap(err, "failed to fetch worklog details")
		}

		for _, wd := range details {
			if !j.okToProcess(wd, j.cfg.UserList) {
				continue
			}
			worklog := types.ToModel(wd, j.tz)

			err = j.repo.SaveWorklog(worklog)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("error saving worklog %d", worklog.IssueId))
			}
		}
	}

	for _, item := range worklogsDeleted.Values {
		err = j.repo.DeleteWorklog(item.WorklogID)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("error deleting worklog %d", item.WorklogID))
		}
	}

	lastUpdateTimestamp = worklogsUpdated.Until
	lastDeletedTimestamp = worklogsDeleted.Until
	j.cfg.WorklogUpdatedLastTimestamp = lastUpdateTimestamp
	j.cfg.WorklogDeletedLastTimestamp = lastDeletedTimestamp
	j.cfg.Save()
	slog.Info("finished processing batch with timestamps", slog.Int64("update", lastUpdateTimestamp), slog.Int64("deleted", lastDeletedTimestamp))
	return nil
}

func (j *JiraSyncWorklogsJob) okToProcess(w jiraworklog.Worklog, userNames []string) bool {
	if len(userNames) == 0 {
		return true
	}

	for _, u := range userNames {
		if strings.ToLower(w.Author.DisplayName) == strings.ToLower(u) {
			return true
		}
	}
	return false
}
