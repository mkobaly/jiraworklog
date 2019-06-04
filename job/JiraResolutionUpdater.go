package job

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/writers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

//JiraResolutionUpdater is a job that runs in the background and scans for any jira issues that are not resolved yet
//It will query for those issues until they are resolved
type JiraResolutionUpdater struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	writer writers.Writer
	logger *log.Entry
}

func NewJiraCheckResolution(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, writer writers.Writer, logger *log.Entry) *JiraResolutionUpdater {
	return &JiraResolutionUpdater{
		cfg:    cfg,
		jira:   jira,
		writer: writer,
		logger: logger,
	}
}

func (j *JiraResolutionUpdater) GetName() string {
	return "JiraCheckResolution"
}

func (j *JiraResolutionUpdater) GetInterval() time.Duration {
	return time.Second * 90
}

func (j *JiraResolutionUpdater) Run() error {
	//jira := jiraworklog.NewJira(j.cfg)

	unresolvedIssues, err := j.writer.NonResolvedIssues()
	if err != nil {
		return errors.Wrap(err, "error fetching non resolved issues")
	}
	j.logger.Info("fetching all non resolved issues")

	for _, issueKey := range unresolvedIssues {
		time.Sleep(400 * time.Millisecond)
		issue, err := j.jira.Issue(issueKey)
		if err != nil {
			return errors.Wrap(err, "unknown error getting issue details from jira. id="+issueKey)
		}

		//j.logger.Info("have issue")

		if !j.isIssueResolved(j.cfg, issue) {
			continue
		}

		j.logger.Info("issue resolved")

		resolved := time.Time{}
		if issue.Fields.ResolutionDate != nil {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.ResolutionDate)
		} else {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.StatusCategoryChangeDate)
		}

		j.logger.WithField("resolved", resolved).Info("resolution date")

		err = j.writer.UpdateResolutionDate(issueKey, resolved)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+issueKey)
		}
		j.logger.WithField("IssueKey", issueKey).Info("updated resolution date for jira issue")
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
