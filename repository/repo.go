package repository

import (
	"context"
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

	IssuesMissingProjectCharge(excludedProjects []string) ([]types.IssueMissingCharge, error)
	IssuesMismatchedProjectCharge(excludedProjects []string) ([]types.IssueMismatchedCharge, error)

	CustomerBugCounts(project string) ([]types.CustomerBugCount, error)
	CustomerBugTrends(project string) ([]types.CustomerBugTrend, error)
	CustomerBugProjects() ([]string, error)

	ProjectCharges() ([]types.ProjectCharge, error)
	UpdateProjectCharge(name string, visible bool, label string) error
	SyncProjectCharges() error

	// Weekly hours by author
	DailyHoursByRole(roles []string, startDate, endDate time.Time) ([]types.DailyHours, error)

	// Timesheet hours by author, grouped by raw project charge and year-month.
	// from is inclusive, to is exclusive (first day of the month after the last
	// month to include).
	TimesheetHours(roles []string, from, to time.Time) ([]types.TimesheetHours, error)

	// Project charge hours reporting (fromMonth/toMonth are YYYY-MM, empty = default range)
	ProjectChargeHours(fromMonth, toMonth string, excludedProjects []string) ([]types.ProjectChargeHours, error)

	// Project time tracking
	ProjectTimeTracking(fixedVersion string) ([]types.ProjectTimeTracking, error)
	ProjectName(epicOrVersion string) (string, error)

	BulkInsertChangelogs(transitions []types.ChangelogStatus) error
	RefreshStatusStints(issueID int) error

	// IssuesWithStaleStints returns the IDs of issues whose `issue.status`
	// disagrees with the status of their most recent `status_stints` row.
	// Excludes issues updated within `notUpdatedSince` (to avoid racing
	// with the regular sync job, which would also be trying to reconcile
	// recently-touched issues). Capped at `limit` to keep each backfill
	// pass bounded.
	IssuesWithStaleStints(notUpdatedSince time.Duration, limit int) ([]int, error)
	ProjectKPIs(epicOrVersion string) (types.ProjectKPIData, error)

	// MonthlyTeamMetrics returns one row per (team, year_month) for the leadership
	// dashboard's all-teams comparison grid. `teams` is the list of Jira project
	// prefixes to include (e.g. ["IDM","SYM","ESG"]); `fromMonth` and `toMonth`
	// are YYYY-MM strings; empty strings default to last-13-months trailing.
	// `ctx` is honored: when the caller (e.g. the HTTP request) is cancelled,
	// the in-flight query is cancelled in Postgres mid-execution.
	MonthlyTeamMetrics(ctx context.Context, teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error)

	// ManagerMetrics returns the full single-team payload for the manager
	// dashboard: the leadership-tier monthly KPIs filtered to `team`, plus
	// seven manager-only sub-metrics (Time in Status, WIP series, Aging WIP,
	// Rework Cycles, Status Bounce, QA vs Eng Hours, Bug vs Forward-Work Hours).
	// `fromMonth` and `toMonth` are YYYY-MM strings; empty defaults to last 13.
	// Sub-queries run concurrently and share `ctx`: when the caller cancels,
	// all in-flight queries cancel too.
	ManagerMetrics(ctx context.Context, team string, fromMonth, toMonth string) (types.ManagerMetricsData, error)

	//WorklogsPerDay() ([]types.WorklogsPerDay, error)
	//WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error)

}
