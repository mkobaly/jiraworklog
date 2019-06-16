package job

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strings"
	"time"
)

//JiraResolutionUpdater is a job that runs in the background and scans for any jira issues that are not resolved yet
//It will query for those issues until they are resolved
type JiraResolutionUpdater struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	repo   repository.Repo
	logger *log.Entry
}

func NewJiraCheckResolution(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo, logger *log.Entry) *JiraResolutionUpdater {
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
	//jira := jiraworklog.NewJira(j.cfg)

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

		//j.logger.Info("have issue")

		if !j.isIssueResolved(j.cfg, issue) {
			continue
		}

		j.logger.Info("issue resolved")

		types.MergeIssue(&ui, issue)
		//resolvedIssue := types.NewResolvedParentIssue(issue)

		// resolved := time.Time{}
		// if issue.Fields.ResolutionDate != nil {
		// 	resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.ResolutionDate)
		// } else {
		// 	resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.StatusCategoryChangeDate)
		// }
		//j.logger.WithField("resolved", resolved).Info("resolution date")

		err = j.repo.UpdateIssue(&ui)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+ui.Key)
		}
		j.logger.WithField("IssueKey", ui.Key).Info("updated resolution date for jira issue")
	}

	//logger.WithField("lasttimestamp", lastTimestamp).WithField("maxworklogID", maxWorklogID).Info("finished processing batch")
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
