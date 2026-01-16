package job

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
)

// JiraSyncIssuesJob will sync all jira issues that have worklogs or have been updated so we have the supporting data
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
	return time.Second * 20
}

func (j *JiraSyncIssuesJob) Run() error {
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
	err = j.fetchAndSaveIssues(jiraIds)
	return err
}

func (j *JiraSyncIssuesJob) fetchAndSaveIssues(jiraIds []string) error {

	slog.Info("bulk fetching jira issues", slog.String("ids", strings.Join(jiraIds, ",")))
	issues, err := jiraworklog.Retry(3, time.Second*10, func() ([]jiraworklog.Issue, error) {
		return j.jira.BulkFetchIssues(jiraIds)
	})
	if err != nil {
		return errors.Wrap(err, "unknown error bulk fetching issues from jira")
	}
	for _, i := range issues {
		issue := types.ToDomain(i)
		err := j.repo.UpdateIssue(&issue)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+issue.Key)
		}
	}
	return nil
}
