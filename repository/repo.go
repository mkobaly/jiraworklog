package repository

import (
	"time"

	"github.com/mkobaly/jiraworklog/types"
)

// Repo is the interface that handles writing Jira Worklog items
type Repo interface {
	//Write(w *types.WorklogItem, pi *types.ParentIssue) error
	//NonResolvedIssues() ([]types.ParentIssue, error)
	//UpdateResolutionDate(issueKey string, resolvedDate time.Time) error

	SaveWorklog(w *types.Worklog) error
	DeleteWorklog(id int) error
	UpdateIssue(*types.StoredIssue) error
	Close()

	MissingIssues() ([]int, error)

	LastSeenIssues(threshold time.Duration) ([]int, error)
	DeleteIssue(id int) error
	UpdateIssueLastSeen(id int) error

	//AllIssues() ([]types.ParentIssue, error)
	IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error)
	IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error)

	//AllWorkLogs() ([]types.WorklogItem, error)
	WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error)
	WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error)
	WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error)

	MaitenanceRatio(roles []string) ([]types.MaitenanceRatio, error)

	People() ([]types.People, error)
	SyncPeople() error
	UpdatePerson(personId int, role string, isEmployee bool, location string) error

	AllRoles() ([]string, error)

	IssuesMissingProjectCharge() ([]types.IssueMissingCharge, error)
	IssuesMismatchedProjectCharge() ([]types.IssueMismatchedCharge, error)

	CustomerBugCounts(project string) ([]types.CustomerBugCount, error)
	CustomerBugTrends(project string) ([]types.CustomerBugTrend, error)
	CustomerBugProjects() ([]string, error)

	ProjectCharges() ([]types.ProjectCharge, error)
	UpdateProjectCharge(name string, visible bool, label string) error
	SyncProjectCharges() error

	// Weekly hours by author
	DailyHoursByRole(roles []string, startDate, endDate time.Time) ([]types.DailyHours, error)

	// Project charge hours reporting (fromMonth/toMonth are YYYY-MM, empty = default range)
	ProjectChargeHours(fromMonth, toMonth string) ([]types.ProjectChargeHours, error)

	// Project time tracking
	ProjectTimeTracking(fixedVersion string) ([]types.ProjectTimeTracking, error)
	ProjectName(epicOrVersion string) (string, error)

	BulkInsertChangelogs(transitions []types.ChangelogStatus) error
	RefreshStatusStints(issueID int) error
	ProjectKPIs(epicOrVersion string) (types.ProjectKPIData, error)

	// MonthlyTeamMetrics returns one row per (team, year_month) for the leadership
	// dashboard's all-teams comparison grid. `teams` is the list of Jira project
	// prefixes to include (e.g. ["IDM","SYM","ESG"]); `fromMonth` and `toMonth`
	// are YYYY-MM strings; empty strings default to last-13-months trailing.
	MonthlyTeamMetrics(teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error)

	//WorklogsPerDay() ([]types.WorklogsPerDay, error)
	//WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)

}
