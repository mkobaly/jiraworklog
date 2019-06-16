package repository

import (
	"github.com/mkobaly/jiraworklog/types"
)

//Repo is the interface that handles writing Jira Worklog items
type Repo interface {
	Write(w *types.WorklogItem, pi *types.ParentIssue) error
	NonResolvedIssues() ([]types.ParentIssue, error)
	//UpdateResolutionDate(issueKey string, resolvedDate time.Time) error
	UpdateIssue(*types.ParentIssue) error
	Close()

	AllWorkLogs() ([]types.WorklogItem, error)
	AllIssues() ([]types.ParentIssue, error)
	IssuesGroupedBy(groupBy string, daysBack int) ([]types.IssueChartData, error)
}
