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
	CycleTimeByStatus []CycleTimeByStatus
	WIPHistory        []WIPDataPoint
	BurndownHistory   []BurndownDataPoint
}

type CycleTimeByStatus struct {
	Status     string  `db:"status"`
	AvgSecs    float64 `db:"avg_seconds"`
	IssueCount int     `db:"issue_count"`
}

type WIPDataPoint struct {
	Day      time.Time `db:"day"`
	WIPCount int       `db:"wip_count"`
}

type BurndownDataPoint struct {
	Day             time.Time `db:"day"`
	CompletedIssues int       `db:"completed_issues"`
}
