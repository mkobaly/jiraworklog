package repository

import (
	"github.com/mkobaly/jiraworklog/types"
	"time"
)

//Repo is the interface that handles writing Jira Worklog items
type Repo interface {
	Write(w types.WorklogItem) error
	NonResolvedIssues() ([]string, error)
	UpdateResolutionDate(issueKey string, resolvedDate time.Time) error
	Close()

	AllWorkLogs() ([]types.WorklogItem, error)
	AllIssues() ([]types.ParentIssue, error)
}
