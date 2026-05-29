package types

// MonthlyTeamMetrics is one row per (team, year_month) in the leadership
// dashboard's KPI grid. Returned by repository.MonthlyTeamMetrics.
//
// Cycle Time / Throughput / Flow Efficiency / Failed QA Ratio are bucketed by
// the close month (first month the issue entered Done/Closed/Cancelled/
// Awaiting Release to Customer). Defect Escape Rate is bucketed by createdate
// month. Stability counts are sourced from the existing CustomerBugTrends
// query and embedded here so the page renders from a single payload.
type MonthlyTeamMetrics struct {
	Team      string `db:"team"`       // Jira project prefix: IDM, SYM, ESG
	YearMonth string `db:"year_month"` // YYYY-MM

	// Close-month bucketed
	MedianCycleTimeSecs  float64 `db:"median_cycle_time_secs"`
	ClosedIssueCount     int     `db:"closed_issue_count"` // = throughput
	FlowEfficiencyPct    float64 `db:"flow_efficiency_pct"`
	FailedQARatioPct     float64 `db:"failed_qa_ratio_pct"`

	// Createdate-month bucketed
	DefectEscapeRatePct float64 `db:"defect_escape_rate_pct"`
	// TotalBugsCount = customer + internal bugs created in this month.
	// Drives HasDefectData(): a month with zero bugs at all should render
	// "—" instead of "0%" with a misleading direction arrow.
	TotalBugsCount int `db:"total_bugs_count"`

	// Stability — populated from CustomerBugTrends, joined in MonthlyTeamMetrics
	StabilityNewCount    int `db:"stability_new_count"`
	StabilityClosedCount int `db:"stability_closed_count"`
	StabilityOpenCount   int `db:"stability_open_count"`
}

// HasClosedIssues reports whether the team closed any issues this month.
// Used by the template to decide between rendering "—" (no data) vs the value.
func (m MonthlyTeamMetrics) HasClosedIssues() bool {
	return m.ClosedIssueCount > 0
}

// IsSparseSample reports whether the closed-issue count is too small for a
// reliable median (n < 5). Drives the "* n=3" caveat in the template.
func (m MonthlyTeamMetrics) IsSparseSample() bool {
	return m.ClosedIssueCount > 0 && m.ClosedIssueCount < 5
}

// HasDefectData reports whether any bugs (customer + internal) were created
// in this month. Used by the leadership-page Defect Escape Rate cell so it
// renders "—" instead of "0% ▼" when there is simply no bug data.
func (m MonthlyTeamMetrics) HasDefectData() bool {
	return m.TotalBugsCount > 0
}
