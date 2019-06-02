package job

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/writers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

func RunResolution(cfg *jiraworklog.Config, writer writers.Writer, logger *log.Entry) error {

	logger.Info("starting runResolution")
	jira := jiraworklog.NewJira(cfg)

	unresolvedIssues, err := writer.NonResolvedIssues()
	if err != nil {
		return errors.Wrap(err, "error fetching non resolved issues")
	}
	logger.Info("fetching issues")

	for _, issueKey := range unresolvedIssues {
		time.Sleep(400 * time.Millisecond)
		issue, err := jira.Issue(issueKey)
		if err != nil {
			return errors.Wrap(err, "unknown error getting issue details from jira. id="+issueKey)
		}

		logger.Info("have issue")

		if !isIssueResolved(cfg, issue) {
			continue
		}

		logger.Info("issue resolved")

		resolved := time.Time{}
		if issue.Fields.ResolutionDate != nil {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.ResolutionDate)
		} else {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *issue.Fields.StatusCategoryChangeDate)
		}

		logger.WithField("resolved", resolved).Info("resolution date")

		err = writer.UpdateResolutionDate(issueKey, resolved)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+issueKey)
		}
		logger.WithField("IssueKey", issueKey).Info("updated resolution date for jira issue")
	}

	//logger.WithField("lasttimestamp", lastTimestamp).WithField("maxworklogID", maxWorklogID).Info("finished processing batch")
	return nil
}

func isIssueResolved(cfg *jiraworklog.Config, issue jiraworklog.Issue) bool {
	for _, status := range cfg.DoneStatus {
		if strings.ToLower(status) == strings.ToLower(issue.Fields.Status.Name) {
			return true
		}
	}
	return false
}
