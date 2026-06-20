package test

import (
	"context"
	"testing"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
)

var cnnString = "Server=192.168.0.2;Database=jira_new;User Id=sa;Password=Kobaly!123"

// func Init() *Repository {

// }

func GetTestConfig() (*jiraworklog.Config, error) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestMaintenanceRatio(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}
	foo, err := repo.MaitenanceRatio([]string{"dev"})
	require.Greater(t, len(foo), 1)
	if err != nil {
		t.Fatal("Error executing repository.Fetch()", err.Error())
	}
}

func TestNullDates(t *testing.T) {
	//var createDate *mssql.DateTime1

}

func TestInsert(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	worklog := &types.Worklog{
		ID:               -1,
		Author:           "bob.smith",
		TimeSpentSeconds: 456,
		Date:             time.Now(),
		IssueId:          1,
		WeekNumber:       23,
		WeekDay:          "Friday",
		TimeSpentHours:   3,
	}

	repo.SaveWorklog(worklog)
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}

func TestBulkInsertChangelog(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	jira := jiraworklog.NewJira(cfg)
	id := 97039
	changelog, err := jira.Changelog(id, 0)
	require.NoError(t, err)
	cs := types.ToChangelogStatus(changelog, id)

	err = repo.BulkInsertChangelogs(cs)
	if err != nil {
		t.Error("Error bulk inserting changlogs", err.Error())
	}
}

func TestRefreshOfStatus(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	id := 97039

	err = repo.RefreshStatusStints(id)
	if err != nil {
		t.Error("Error bulk inserting changlogs", err.Error())
	}
}

func TestProjectKPIsSmoke(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	// Pull any real epic key from the DB so the test doesn't depend on a
	// hard-coded key that may not exist.
	var key string
	err = repo.DB.Get(&key, `SELECT key FROM issue WHERE type = 'Epic' LIMIT 1`)
	if err != nil {
		t.Skip("no epic in DB; skipping ProjectKPIs smoke test")
	}

	data, err := repo.ProjectKPIs(key)
	require.NoError(t, err)
	require.GreaterOrEqual(t, data.IssueCount, 0)
}

func TestMonthlyTeamMetricsThroughput(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	// At least one row across the whole grid must have a positive throughput.
	// (If this fails, either no issues have closed in the last 13 months for
	// these teams, or the close-month bucketing logic is broken.)
	total := 0
	for _, r := range rows {
		total += r.ClosedIssueCount
	}
	require.Greater(t, total, 0, "expected at least one closed issue across all teams/months")
}

func TestMonthlyTeamMetricsSkeleton(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)
	// 3 teams * 13 months = 39 rows
	require.Equal(t, 39, len(rows))

	// Each team should appear with 13 distinct year_months
	teamCounts := map[string]int{}
	for _, r := range rows {
		teamCounts[r.Team]++
		require.Regexp(t, `^\d{4}-\d{2}$`, r.YearMonth)
	}
	require.Equal(t, 13, teamCounts["IDM"])
	require.Equal(t, 13, teamCounts["SYM"])
	require.Equal(t, 13, teamCounts["ESG"])
}

func TestMonthlyTeamMetricsCycleTime(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	// For any (team, month) row where throughput > 0, median cycle time must
	// also be > 0. Cycle time = 0 with throughput > 0 means the cycle-time
	// query failed to populate.
	foundPositive := false
	for _, r := range rows {
		if r.ClosedIssueCount > 0 {
			require.Greaterf(t, r.MedianCycleTimeSecs, 0.0,
				"team=%s month=%s has %d closed issues but median cycle time is 0",
				r.Team, r.YearMonth, r.ClosedIssueCount)
			foundPositive = true
		}
	}
	require.True(t, foundPositive, "no closed-issue rows found; can't validate cycle time")
}

func TestMonthlyTeamMetricsFlowEfficiency(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	for _, r := range rows {
		// Flow efficiency must be in [0, 100]
		require.GreaterOrEqual(t, r.FlowEfficiencyPct, 0.0)
		require.LessOrEqual(t, r.FlowEfficiencyPct, 100.0)
		// If issues closed this month, flow efficiency must be > 0
		if r.ClosedIssueCount > 0 {
			require.Greaterf(t, r.FlowEfficiencyPct, 0.0,
				"team=%s month=%s has %d closed issues but flow efficiency is 0",
				r.Team, r.YearMonth, r.ClosedIssueCount)
		}
	}
}

func TestMonthlyTeamMetricsFailedQARatio(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	for _, r := range rows {
		require.GreaterOrEqual(t, r.FailedQARatioPct, 0.0)
		require.LessOrEqual(t, r.FailedQARatioPct, 100.0)
	}
}

func TestMonthlyTeamMetricsDefectEscapeRate(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	for _, r := range rows {
		require.GreaterOrEqual(t, r.DefectEscapeRatePct, 0.0)
		require.LessOrEqual(t, r.DefectEscapeRatePct, 100.0)
	}

	// Customer bugs exist for all three teams in recent history — the metric
	// should be > 0 at least once across the whole grid.
	foundNonZero := false
	for _, r := range rows {
		if r.DefectEscapeRatePct > 0 {
			foundNonZero = true
			break
		}
	}
	require.True(t, foundNonZero, "expected defect escape rate > 0 somewhere on the grid")
}

func TestManagerMetricsSkeleton(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.Equal(t, "SYM", data.Team)
	// 13 months of leadership rows for the single team
	require.Equal(t, 13, len(data.LeadershipMonthly))
	for _, r := range data.LeadershipMonthly {
		require.Equal(t, "SYM", r.Team)
		require.Regexp(t, `^\d{4}-\d{2}$`, r.YearMonth)
	}
}

func TestManagerMetricsTimeInStatus(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.Greater(t, len(data.TimeInStatus), 0,
		"expected at least one TimeInStatus row across 13 months × 3 buckets")
	for _, p := range data.TimeInStatus {
		require.Regexp(t, `^\d{4}-\d{2}$`, p.YearMonth)
		require.Contains(t, []string{"Dev", "QA", "Waiting"}, p.Bucket)
		require.GreaterOrEqual(t, p.AvgSecs, 0.0)
	}
}

func TestManagerMetricsWIPSeries(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.Greater(t, len(data.WIPSeries), 300,
		"expected ~390 daily WIP points across 13 months")
	for _, p := range data.WIPSeries {
		require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, p.Day)
		require.GreaterOrEqual(t, p.WIPCount, 0)
	}
}

func TestManagerMetricsAgingWIP(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	// AgingWIPItems may legitimately be empty (healthy team), but the field
	// must be initialized (non-nil) after the query runs. Pre-implementation
	// it will be nil.
	require.NotNil(t, data.AgingWIPItems, "AgingWIPItems must be initialized")
	require.LessOrEqual(t, len(data.AgingWIPItems), 20, "AgingWIP capped at top 20")
	for _, item := range data.AgingWIPItems {
		require.NotEmpty(t, item.Key)
		require.NotEmpty(t, item.Status)
		require.GreaterOrEqual(t, item.DaysInStatus, 0)
	}
	// Items must be sorted descending by DaysInStatus
	for i := 1; i < len(data.AgingWIPItems); i++ {
		require.GreaterOrEqual(t,
			data.AgingWIPItems[i-1].DaysInStatus,
			data.AgingWIPItems[i].DaysInStatus,
			"AgingWIP must be sorted by DaysInStatus desc")
	}
}

func TestManagerMetricsReworkCycles(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.ReworkCycles)
	for _, p := range data.ReworkCycles {
		require.Regexp(t, `^\d{4}-\d{2}$`, p.YearMonth)
		require.GreaterOrEqual(t, p.AvgCycles, 0.0)
		require.GreaterOrEqual(t, p.ReworkedIssueCount, 0)
	}
}

func TestManagerMetricsStatusBounce(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.StatusBounce)
	for _, p := range data.StatusBounce {
		require.GreaterOrEqual(t, p.BouncePct, 0.0)
		require.LessOrEqual(t, p.BouncePct, 100.0)
		require.GreaterOrEqual(t, p.TotalIssues, 0)
	}
}

func TestManagerMetricsQAvsEngHours(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.QAvsEngHours)
	for _, p := range data.QAvsEngHours {
		require.GreaterOrEqual(t, p.Numerator, 0.0)
		require.GreaterOrEqual(t, p.Denominator, 0.0)
		require.GreaterOrEqual(t, p.Ratio, 0.0)
	}
}

func TestManagerMetricsBugVsForwardHours(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics(context.Background(), "SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.BugvsForwardHours)
	for _, p := range data.BugvsForwardHours {
		require.GreaterOrEqual(t, p.Numerator, 0.0)
		require.GreaterOrEqual(t, p.Denominator, 0.0)
		require.GreaterOrEqual(t, p.Ratio, 0.0)
	}
}

func TestMonthlyTeamMetricsStability(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics(context.Background(), []string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	// All counts must be non-negative
	for _, r := range rows {
		require.GreaterOrEqual(t, r.StabilityNewCount, 0)
		require.GreaterOrEqual(t, r.StabilityClosedCount, 0)
		require.GreaterOrEqual(t, r.StabilityOpenCount, 0)
	}
	// At least one positive open count across the grid (customer bugs always
	// have nonzero open backlog).
	totalOpen := 0
	for _, r := range rows {
		totalOpen += r.StabilityOpenCount
	}
	require.Greater(t, totalOpen, 0, "expected non-zero customer bug open count somewhere")
}

func TestTimesheetHours(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	// A recent, completed 3-month window: Jan–Mar 2026.
	// Use America/New_York so the bounds align with how the SQL buckets months.
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)
	to := time.Date(2026, 4, 1, 0, 0, 0, 0, loc) // exclusive upper bound

	rows, err := repo.TimesheetHours([]string{"dev"}, from, to)
	require.NoError(t, err)
	require.Greater(t, len(rows), 0)

	for _, r := range rows {
		require.NotEmpty(t, r.Author)
		require.NotEmpty(t, r.ProjectCharge)
		// YearMonth must fall inside the requested inclusive month range.
		require.GreaterOrEqual(t, r.YearMonth, "2026-01")
		require.LessOrEqual(t, r.YearMonth, "2026-03")
		require.Greater(t, r.Hours, 0.0)
	}
}
