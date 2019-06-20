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

	AllIssues() ([]types.ParentIssue, error)
	IssuesGroupedBy(groupBy string, weeksBack int) ([]types.IssueChartData, error)

	AllWorkLogs() ([]types.WorklogItem, error)
	WorklogsPerDay() ([]types.WorklogsPerDay, error)
	WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)
	WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error)
}
