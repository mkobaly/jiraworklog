package repository

import (
	"time"

	"github.com/mkobaly/jiraworklog/types"
)

// Repo is the interface that handles writing Jira Worklog items
type Repo interface {
	//Write(w *types.WorklogItem, pi *types.ParentIssue) error
	NonResolvedIssues() ([]types.ParentIssue, error)
	//UpdateResolutionDate(issueKey string, resolvedDate time.Time) error

	SaveWorklog(w *types.Worklog) error
	DeleteWorklog(id int) error
	UpdateIssue(*types.StoredIssue) error
	Close()

	MissingIssues() ([]int, error)

	AllIssues() ([]types.ParentIssue, error)
	IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error)
	IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error)

	AllWorkLogs() ([]types.WorklogItem, error)
	WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error)
	WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error)
	WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error)

	MaitenanceRatio(roles []string) ([]types.MaitenanceRatio, error)

	People() ([]types.People, error)
	UpdatePersonRole(personId int, role string) error

	AllRoles() ([]string, error)

	IssuesMissingProjectCharge() ([]types.IssueMissingCharge, error)

	CustomerBugCounts(project string) ([]types.CustomerBugCount, error)
	CustomerBugProjects() ([]string, error)

	// Project management
	AllProjects() ([]types.Project, error)
	CreateProject(name string, projectCharges []string) error
	UpdateProject(id int, name string, visible bool, projectCharges []string) error
	DeleteProject(id int) error
	AllProjectCharges() ([]string, error)

	// Project charge hours reporting
	ProjectChargeHours() ([]types.ProjectChargeHours, error)

	// Weekly hours by author
	DailyHoursByRole(roles []string, startDate, endDate time.Time) ([]types.DailyHours, error)

	//WorklogsPerDay() ([]types.WorklogsPerDay, error)
	//WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)

}
