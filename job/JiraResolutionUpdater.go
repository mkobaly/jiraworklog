package job

import (
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
)

//JiraResolutionUpdater is a job that runs in the background and scans for any jira issues that are not resolved yet
//It will query for those issues until they are resolved
type JiraResolutionUpdater struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	repo   repository.Repo
	logger *slog.Logger
}

func NewJiraCheckResolution(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo, logger *slog.Logger) *JiraResolutionUpdater {
	return &JiraResolutionUpdater{
		cfg:    cfg,
		jira:   jira,
		repo:   repo,
		logger: logger,
	}
}

func (j *JiraResolutionUpdater) GetName() string {
	return "JiraResolutionUpdater"
}

func (j *JiraResolutionUpdater) GetInterval() time.Duration {
	return time.Second * 600
}

func (j *JiraResolutionUpdater) Run() error {
	unresolvedIssues, err := j.repo.NonResolvedIssues()
	if err != nil {
		return errors.Wrap(err, "error fetching non resolved issues")
	}
	j.logger.Info("fetching all non resolved issues")

	for _, ui := range unresolvedIssues {
		delay := getDelay()
		time.Sleep(time.Duration(delay) * time.Millisecond)
		issue, err := j.jira.Issue(ui.Key)
		if err != nil {
			return errors.Wrap(err, "unknown error getting issue details from jira. id="+ui.Key)
		}

		if !j.isIssueResolved(j.cfg, issue) {
			continue
		}

		j.logger.Info("issue resolved")

		types.MergeIssue(&ui, issue)

		err = j.repo.UpdateIssue(&ui)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+ui.Key)
		}
		j.logger.Info("updated resolution date for jira issue", "IssueKey", ui.Key)
	}

	return nil
}

func (j *JiraResolutionUpdater) isIssueResolved(cfg *jiraworklog.Config, issue jiraworklog.Issue) bool {
	for _, status := range cfg.DoneStatus {
		if strings.ToLower(status) == strings.ToLower(issue.Fields.Status.Name) {
			return true
		}
	}
	return false
}

func getDelay() int {
	rand.Seed(time.Now().UnixNano())
	min := 100
	max := 900
	return rand.Intn(max-min) + min
}
