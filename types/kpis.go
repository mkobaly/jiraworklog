package types

import "time"

// ProjectKPIData holds all computed KPI metrics for a project epic or fixed version.
type ProjectKPIData struct {
	IssueCount int

	// Cycle time (active development statuses only)
	AvgCycleTimeSecs    float64
	MedianCycleTimeSecs float64
	P85CycleTimeSecs    float64
	P95CycleTimeSecs    float64

	// Blocked time
	AvgBlockedTimeSecs float64
	BlockedIssueCount  int

	// Status bounce / thrash
	TotalBounces     int
	BounceIssueCount int

	// Flow efficiency: active_secs / total_elapsed_secs
	FlowEfficiencyPct float64

	// QA rework
	AvgQACycles          float64
	IssuesReturnedFromQA int

	// Breakdowns
	FlowBuckets       []FlowBucket
	CycleTimeByStatus []CycleTimeByStatus
	WIPHistory        []WIPDataPoint
	BurndownHistory   []BurndownDataPoint
	BurnupHistory     []BurnupDataPoint
}

// FlowBucket aggregates time spent across a named group of statuses.
type FlowBucket struct {
	Bucket     string  `db:"bucket"`
	AvgSecs    float64 `db:"avg_seconds"`
	TotalSecs  float64 `db:"total_seconds"`
	IssueCount int     `db:"issue_count"`
}

type CycleTimeByStatus struct {
	Status     string  `db:"status"`
	Bucket     string  `db:"bucket"`
	AvgSecs    float64 `db:"avg_seconds"`
	IssueCount int     `db:"issue_count"`
}

type WIPDataPoint struct {
	Day      time.Time `db:"day"`
	WIPCount int       `db:"wip_count"`
}

type BurndownDataPoint struct {
	Day              time.Time `db:"day"`
	DevCompleteCount int       `db:"dev_complete_count"` // cumulative issues dev is done with (in QA or beyond)
	FullyDoneCount   int       `db:"fully_done_count"`   // cumulative issues fully shipped
}

type BurnupDataPoint struct {
	Day                  time.Time `db:"day"`
	CompletedSeconds     float64   `db:"completed_seconds"`       // cumulative hours logged on project issues
	TotalScopeSeconds    float64   `db:"total_scope_seconds"`     // cumulative originalestimate for all issues (inc. bugs)
	PlannedScopeSeconds  float64   `db:"planned_scope_seconds"`   // cumulative originalestimate excluding Bug / Hardware Bug
}
