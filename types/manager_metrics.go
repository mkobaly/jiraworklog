package types

// ManagerMetricsData is the full payload for the manager-page render. It
// bundles the existing leadership-tier monthly metrics (filtered to one team)
// with seven new manager-only metrics.
type ManagerMetricsData struct {
	Team               string
	LeadershipMonthly  []MonthlyTeamMetrics // 13 rows, one per month, single team
	TimeInStatus       []TimeInStatusPoint  // monthly avg seconds by bucket
	WIPSeries          []WIPPoint           // daily WIP count, last 13 months
	AgingWIPItems      []AgingWIPItem       // current snapshot, top 20 stale issues
	ReworkCycles       []ReworkCyclesPoint  // monthly
	StatusBounce       []BouncePoint        // monthly
	QAvsEngHours       []HoursRatioPoint    // monthly
	BugvsForwardHours  []HoursRatioPoint    // monthly
}

// TimeInStatusPoint is one (month, bucket) entry for the stacked-bar chart.
// Bucket values: 'Dev', 'QA', 'Waiting'.
type TimeInStatusPoint struct {
	YearMonth string  `db:"year_month"`
	Bucket    string  `db:"bucket"`
	AvgSecs   float64 `db:"avg_seconds"`
}

// WIPPoint is one daily WIP count (issues currently in Dev or QA).
type WIPPoint struct {
	Day      string `db:"day"`       // YYYY-MM-DD
	WIPCount int    `db:"wip_count"`
}

// AgingWIPItem is one open issue currently in Dev or QA whose time-in-bucket
// exceeds the team's 85th-percentile cycle time.
type AgingWIPItem struct {
	Key          string `db:"key"`
	Status       string `db:"status"`
	DaysInStatus int    `db:"days_in_status"`
}

// ReworkCyclesPoint is the average QA→Dev rework cycle count per closed
// issue in a given (team, month).
type ReworkCyclesPoint struct {
	YearMonth  string  `db:"year_month"`
	AvgCycles  float64 `db:"avg_cycles"`
	IssueCount int     `db:"issue_count"`
}

// BouncePoint is monthly status-bounce rate (status re-entries / total
// status entries) for closed issues.
type BouncePoint struct {
	YearMonth   string  `db:"year_month"`
	BouncePct   float64 `db:"bounce_pct"`
	TotalIssues int     `db:"total_issues"`
}

// HoursRatioPoint is one month's numerator / denominator / ratio for hour
// comparisons (QA vs Eng, Bug vs Forward-Work).
type HoursRatioPoint struct {
	YearMonth   string  `db:"year_month"`
	Numerator   float64 `db:"numerator_hours"`
	Denominator float64 `db:"denominator_hours"`
	Ratio       float64 `db:"ratio"`
}
