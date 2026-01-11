package job

import (
	"log/slog"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
)

// JiraWorklogsDownloader is a background job that will download all worklogs from Jira
type JiraWorklogsDownloader struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	repo   repository.Repo
	logger *slog.Logger
}

func NewJiraDownloadWorklogs(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo, logger *slog.Logger) *JiraWorklogsDownloader {
	return &JiraWorklogsDownloader{
		cfg:    cfg,
		jira:   jira,
		repo:   repo,
		logger: logger,
	}
}

func (j *JiraWorklogsDownloader) GetName() string {
	return "JiraWorklogsDownloader"
}

func (j *JiraWorklogsDownloader) GetInterval() time.Duration {
	return time.Second * 20
}

func (j *JiraWorklogsDownloader) Run() error {
	lastTimestamp := j.cfg.LastTimestamp
	if lastTimestamp == 0 {
		// default going back 4 months
		lastTimestamp = time.Now().AddDate(0, -4, 0).UnixNano() / 1e6
	}
	maxWorklogID := j.cfg.MaxWorklogID
	j.logger.Info("last timestamp", "timestamp", lastTimestamp)

	//Fetch worklogs updated since last timestamp check
	wl, err := j.jira.WorklogsUpdated(lastTimestamp)
	if err != nil {
		return errors.Wrap(err, "failed to fetch updated worklogs from jira")
	}
	j.logger.Info("worklogs updated stats",
		"since", wl.Since,
		"until", wl.Until,
		"lastPage", wl.LastPage,
		"count", len(wl.Values))

	//For given worklog Ids we now need to get the worklog details
	var ids []int
	for _, w := range wl.Values {
		if w.WorklogID > maxWorklogID {
			ids = append(ids, w.WorklogID)
		}
	}

	if len(ids) == 0 {
		return nil //nothing to do
	}

	details, err := j.jira.WorklogDetails(ids)
	if err != nil {
		return errors.Wrap(err, "failed to fetch worklog details")
	}

	for _, wd := range details {
		if !j.okToProcess(wd, j.cfg.UserList) {
			continue
		}

		issue, err := jiraworklog.Retry(3, time.Second*10, func() (jiraworklog.Issue, error) {
			return j.jira.Issue(wd.IssueID)
		})
		if err != nil {
			switch err {
			case jiraworklog.ErrIssueNotFound:
				continue
			default:
				return errors.Wrap(err, "unknown error getting issue details from jira. id="+wd.ID)
			}
		}

		issueParent := jiraworklog.Issue{}
		if issue.HasParent() {
			issueParent, err = j.jira.Issue(issue.ParentID())
		}

		workItem, parentIssue := types.ConvertToModels(wd, issue, issueParent)
		err = j.repo.Write(workItem, parentIssue)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+workItem.IssueKey)
		}
		maxWorklogID = workItem.ID
		j.logger.Info("inserted jira issue",
			"IssueKey", workItem.IssueKey,
			"Date", workItem.Date)
		time.Sleep(200 * time.Millisecond)
	}
	lastTimestamp = wl.Until
	j.cfg.MaxWorklogID = maxWorklogID
	j.cfg.LastTimestamp = lastTimestamp
	j.cfg.Save()
	j.logger.Info("finished processing batch",
		"lasttimestamp", lastTimestamp,
		"maxworklogID", maxWorklogID)
	return nil
}

func (j *JiraWorklogsDownloader) okToProcess(w jiraworklog.Worklog, userNames []string) bool {
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
