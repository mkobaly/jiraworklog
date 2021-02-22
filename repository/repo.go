package repository

import (
	"time"

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
	IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error)
	IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error)

	AllWorkLogs() ([]types.WorklogItem, error)
	WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error)
	WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error)
	WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error)

	//WorklogsPerDay() ([]types.WorklogsPerDay, error)
	//WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)

}
