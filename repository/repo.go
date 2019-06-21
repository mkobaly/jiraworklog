package repository

import (
	"github.com/mkobaly/jiraworklog/types"
	"time"
)

//Repo is the interface that handles writing Jira Worklog items
type Repo interface {
	Write(w *types.WorklogItem, pi *types.ParentIssue) error
	NonResolvedIssues() ([]types.ParentIssue, error)
	//UpdateResolutionDate(issueKey string, resolvedDate time.Time) error
	UpdateIssue(*types.ParentIssue) error
	Close()

	AllIssues() ([]types.ParentIssue, error)
	IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error)
	IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error)

	AllWorkLogs() ([]types.WorklogItem, error)
	WorklogsGroupBy(groupBy string) ([]types.WorklogGroupByChart, error)
	WorklogsPerDay() ([]types.WorklogsPerDay, error)
	WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)
	WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error)
}
