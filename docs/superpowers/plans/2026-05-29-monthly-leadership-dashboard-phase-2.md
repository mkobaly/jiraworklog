# Monthly Leadership Dashboard — Phase 2 Implementation Plan (Manager Page)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `/dashboard/manager?team=SYM` page — a single-team drill-down showing the four Leadership predictability KPIs (with bigger 13-month charts), four new Quality metrics (Rework Cycles, Status Bounce, QA-vs-Eng Hours, Bug-vs-Forward-Work Hours), Time-in-Status stacked bar, WIP trend, Aging WIP list, and the team's Stability chart plus Defect Escape Rate trend.

**Architecture:** New `ManagerMetrics(team, fromMonth, toMonth)` repo method that bundles seven new sub-queries with the existing `MonthlyTeamMetrics` filtered to a single team. New `templates/pages/manager.templ` page (three sections: Delivery Predictability, Quality, Stability). New `GET /dashboard/manager` handler. Team-header links on the Leadership page already point here (Phase 1 wired the href).

**Tech Stack:** Go, Echo v4, sqlx + pgx/v5, templ, Chart.js (via CDN), Tailwind, Hotwire Turbo, PostgreSQL.

**Reference:** Spec at `docs/superpowers/specs/2026-05-29-monthly-leadership-dashboard-design.md`. Phase 1 plan at `docs/superpowers/plans/2026-05-29-monthly-leadership-dashboard-phase-1.md`.

---

## File Structure

**Files created:**
- `types/manager_metrics.go` — type definitions for the manager-page payload
- `templates/pages/manager.templ` — page template

**Files modified:**
- `repository/repo.go` — add `ManagerMetrics` to the `Repo` interface
- `repository/posgres.go` — add `ManagerMetrics()` implementation (calls `MonthlyTeamMetrics` filtered to one team, plus seven new sub-queries)
- `cmd/jiraworklog/handlers.go` — add `GetManagerDashboard` handler
- `cmd/jiraworklog/main.go` — register `GET /dashboard/manager` route
- `test/repo_test.go` — add integration tests for `ManagerMetrics`

---

## Conventions

- Tests follow the same loose-smoke-test convention used in Phase 1 (real DB via `GetTestConfig()`, assert returned data shape and non-trivial population).
- Templ regeneration via `templ generate` after every `.templ` edit.
- One commit per task. Sign-off line is whatever Git is configured to add automatically — do not add `Co-Authored-By` manually.
- All new SQL reuses the constants lifted in Phase 1 (`closeableTypes`, `bucketCase`, `devQAStatuses`, `doneStatuses`). One new constant is added in Task 4 (`waitingStatuses`).

---

## Task 1: Add manager-metrics type definitions

Create the type that drives the manager-page payload.

**Files:**
- Create: `types/manager_metrics.go`

- [ ] **Step 1: Create the file with all new types**

```go
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
```

- [ ] **Step 2: Verify the file builds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build succeeds (no consumers yet — types compile in isolation).

- [ ] **Step 3: Commit**

```bash
git add types/manager_metrics.go
git commit -m "feat(types): add ManagerMetricsData and supporting types"
```

---

## Task 2: Add waitingStatuses SQL constant

The Time-in-Status query needs to identify the "Waiting" bucket statuses separately. Lift them to a named constant alongside the Phase 1 constants.

**Files:**
- Modify: `repository/posgres.go`

- [ ] **Step 1: Add the constant inside the existing `const ( ... )` block**

Find the existing `const ( ... )` block (added in Phase 1 Task 1, around line 605-625) that defines `closeableTypes`, `bucketCase`, `devQAStatuses`, `doneStatuses`. Add a new constant inside it:

```go
// waitingStatuses lists the statuses that represent waiting time
// (issue is open but not actively progressing in Dev or QA).
waitingStatuses = `('On Hold','QA Backlog')`
```

The full block should now contain five constants.

- [ ] **Step 2: Verify the file builds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build succeeds (the constant is unused right now — Task 5 consumes it).

- [ ] **Step 3: Commit**

```bash
git add repository/posgres.go
git commit -m "refactor(repo): add waitingStatuses SQL constant for Time-in-Status query"
```

---

## Task 3: Extend the Repo interface with ManagerMetrics

**Files:**
- Modify: `repository/repo.go`

- [ ] **Step 1: Add the interface method**

Find the line `MonthlyTeamMetrics(teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error)` in `repository/repo.go`. Add immediately after it:

```go
// ManagerMetrics returns the full single-team payload for the manager
// dashboard: the leadership-tier monthly KPIs filtered to `team`, plus
// seven manager-only sub-metrics (Time in Status, WIP series, Aging WIP,
// Rework Cycles, Status Bounce, QA vs Eng Hours, Bug vs Forward-Work Hours).
// `fromMonth` and `toMonth` are YYYY-MM strings; empty defaults to last 13.
ManagerMetrics(team string, fromMonth, toMonth string) (types.ManagerMetricsData, error)
```

- [ ] **Step 2: Verify (expected to fail)**

Run: `go build ./...`
Expected: FAIL with `*Postgres does not implement Repo (missing method ManagerMetrics)`.

- [ ] **Step 3: Do NOT commit yet.**

Task 4 commits both the interface change and the skeleton implementation together (mirrors Phase 1's Task 3/4 pattern).

---

## Task 4: Implement ManagerMetrics skeleton (single-team leadership reuse)

The skeleton returns the team plus 13 months of leadership-tier metrics (`MonthlyTeamMetrics` filtered to the single team). The seven new sub-queries are added one per task.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing integration test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsSkeleton(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.Equal(t, "SYM", data.Team)
	// 13 months of leadership rows for the single team
	require.Equal(t, 13, len(data.LeadershipMonthly))
	for _, r := range data.LeadershipMonthly {
		require.Equal(t, "SYM", r.Team)
		require.Regexp(t, `^\d{4}-\d{2}$`, r.YearMonth)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsSkeleton -v
```

Expected: FAIL (method doesn't exist).

- [ ] **Step 3: Implement the skeleton**

Add this method near `MonthlyTeamMetrics` (just after it) in `repository/posgres.go`:

```go
// ManagerMetrics — see repository/repo.go for contract.
func (s *Postgres) ManagerMetrics(team string, fromMonth, toMonth string) (types.ManagerMetricsData, error) {
	data := types.ManagerMetricsData{Team: team}

	// Reuse the leadership-tier monthly query, filtered to one team.
	rows, err := s.MonthlyTeamMetrics([]string{team}, fromMonth, toMonth)
	if err != nil {
		return data, err
	}
	data.LeadershipMonthly = rows

	// Subsequent tasks populate the manager-only sub-fields here.

	return data, nil
}
```

- [ ] **Step 4: Build and run the test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsSkeleton -v
```

Expected: build succeeds, test PASS.

- [ ] **Step 5: Commit (includes both the interface change from Task 3 and the skeleton)**

```bash
git add repository/repo.go repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add ManagerMetrics skeleton wrapping single-team MonthlyTeamMetrics"
```

---

## Task 5: Add Time in Status sub-query

Time in Status = `AVG(durationseconds)` per (month, bucket) for `status_stints` belonging to the team's closeable-type issues, where bucket is Dev / QA / Waiting. One row per (month, bucket) so the chart can stack them.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsTimeInStatus(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.Greater(t, len(data.TimeInStatus), 0,
		"expected at least one TimeInStatus row across 13 months × 3 buckets")
	for _, p := range data.TimeInStatus {
		require.Regexp(t, `^\d{4}-\d{2}$`, p.YearMonth)
		require.Contains(t, []string{"Dev", "QA", "Waiting"}, p.Bucket)
		require.GreaterOrEqual(t, p.AvgSecs, 0.0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsTimeInStatus -v
```

Expected: FAIL (TimeInStatus is empty).

- [ ] **Step 3: Add the query and call it from `ManagerMetrics`**

In `repository/posgres.go`, add a helper method below `ManagerMetrics`:

```go
// timeInStatus returns one row per (month status was exited, bucket) for the
// team's closeable-type issues. Used by the Time-in-Status stacked-bar chart.
func (s *Postgres) timeInStatus(team string, fromMonth, toMonth string) ([]types.TimeInStatusPoint, error) {
	result := []types.TimeInStatusPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		SELECT
			to_char(COALESCE(ss.dateended, NOW()), 'YYYY-MM') AS year_month,
			CASE
				WHEN ss.status IN ('In Development','In Progress','Code Complete','Code Merged','In Review') THEN 'Dev'
				WHEN ss.status IN ('In QA','Failed QA') THEN 'QA'
				WHEN ss.status IN `+waitingStatuses+` THEN 'Waiting'
				ELSE NULL
			END AS bucket,
			AVG(ss.durationseconds)::float8 AS avg_seconds
		FROM status_stints ss
		JOIN issue i ON i.id = ss.issueid
		WHERE i.project = $1
		AND i.type IN `+closeableTypes+`
		AND COALESCE(ss.dateended, NOW()) >= $2::timestamptz
		AND COALESCE(ss.dateended, NOW()) <  $3::timestamptz
		AND (
			ss.status IN `+devQAStatuses+`
			OR ss.status IN `+waitingStatuses+`
		)
		GROUP BY year_month, bucket
		HAVING bucket IS NOT NULL
		ORDER BY year_month, bucket`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, just before `return data, nil`, add:

```go
data.TimeInStatus, err = s.timeInStatus(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run the test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsTimeInStatus -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Time-in-Status sub-query to ManagerMetrics"
```

---

## Task 6: Add WIP series sub-query

WIP series = daily count of the team's open issues currently in a Dev or QA status. Daily granularity, last 13 months.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsWIPSeries(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.Greater(t, len(data.WIPSeries), 300,
		"expected ~390 daily WIP points across 13 months")
	for _, p := range data.WIPSeries {
		require.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, p.Day)
		require.GreaterOrEqual(t, p.WIPCount, 0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsWIPSeries -v
```

Expected: FAIL.

- [ ] **Step 3: Add the query and call it**

Add helper below `timeInStatus`:

```go
// wipSeries returns one row per day across the 13-month window — the count
// of the team's closeable-type issues currently in a Dev or QA status on
// that day. Used by the WIP-trend line chart.
func (s *Postgres) wipSeries(team string, fromMonth, toMonth string) ([]types.WIPPoint, error) {
	result := []types.WIPPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH days AS (
			SELECT generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 day'::interval
			)::date AS day
		),
		team_issues AS (
			SELECT id FROM issue
			WHERE project = $1
			AND type IN `+closeableTypes+`
		)
		SELECT
			to_char(d.day, 'YYYY-MM-DD') AS day,
			COUNT(DISTINCT ss.issueid) AS wip_count
		FROM days d
		LEFT JOIN status_stints ss
			ON ss.issueid IN (SELECT id FROM team_issues)
			AND ss.status IN `+devQAStatuses+`
			AND ss.datestarted::date <= d.day
			AND (ss.dateended IS NULL OR ss.dateended::date > d.day)
		GROUP BY d.day
		ORDER BY d.day`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the TimeInStatus call:

```go
data.WIPSeries, err = s.wipSeries(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsWIPSeries -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add WIP series sub-query to ManagerMetrics"
```

---

## Task 7: Add Aging WIP sub-query (current snapshot)

Aging WIP = current snapshot. List of the team's issues currently in Dev or QA where time-in-current-bucket exceeds the team's 85th-percentile cycle time over the last 13 months. Returns top 20, sorted by days-in-status descending.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsAgingWIP(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
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
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsAgingWIP -v
```

Expected: FAIL (`data.AgingWIPItems` is nil before implementation, so `require.NotNil` triggers).

- [ ] **Step 3: Add the query and call it**

Add helper below `wipSeries`:

```go
// agingWIP returns the team's currently open issues in Dev or QA whose
// time-in-current-status exceeds the team's 85th-percentile cycle time over
// the last 13 months. Used by the Aging WIP list on the manager page.
// Returns at most 20 rows, sorted by days_in_status DESC.
func (s *Postgres) agingWIP(team string, fromMonth, toMonth string) ([]types.AgingWIPItem, error) {
	result := []types.AgingWIPItem{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH p85_cycle AS (
			-- 85th-percentile cycle time for the team over the last 13 months
			SELECT COALESCE(
				PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY ct.total_secs),
				14 * 86400  -- fallback if no closed issues: 14 days
			) AS p85_secs
			FROM (
				SELECT
					ss.issueid,
					SUM(ss.durationseconds)::float8 AS total_secs
				FROM status_stints ss
				JOIN issue i ON i.id = ss.issueid
				JOIN (
					SELECT ss2.issueid, MIN(ss2.datestarted) AS close_at
					FROM status_stints ss2
					JOIN issue i2 ON i2.id = ss2.issueid
					WHERE ss2.status IN `+doneStatuses+`
					AND i2.project = $1
					AND i2.type IN `+closeableTypes+`
					GROUP BY ss2.issueid
				) closed ON closed.issueid = ss.issueid
				WHERE ss.status IN `+devQAStatuses+`
				AND i.project = $1
				AND i.type IN `+closeableTypes+`
				AND closed.close_at >= $2::timestamptz
				AND closed.close_at <  $3::timestamptz
				GROUP BY ss.issueid
			) ct
		),
		current_stints AS (
			-- Issues that are currently in a Dev/QA status (dateended IS NULL)
			SELECT
				i.key,
				ss.status,
				EXTRACT(EPOCH FROM (NOW() - ss.datestarted))::bigint AS current_secs
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.dateended IS NULL
			AND ss.status IN `+devQAStatuses+`
			AND i.project = $1
			AND i.type IN `+closeableTypes+`
		)
		SELECT
			cs.key,
			cs.status,
			(cs.current_secs / 86400)::int AS days_in_status
		FROM current_stints cs
		CROSS JOIN p85_cycle p
		WHERE cs.current_secs > p.p85_secs
		ORDER BY cs.current_secs DESC
		LIMIT 20`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the WIPSeries call:

```go
data.AgingWIPItems, err = s.agingWIP(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsAgingWIP -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Aging WIP sub-query to ManagerMetrics"
```

---

## Task 8: Add Rework Cycles sub-query

Rework Cycles = average QA→Dev transition count per closed issue, by month. Reuses the pattern from `ProjectKPIs`'s existing rework query but bucketed by close month.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsReworkCycles(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.ReworkCycles)
	for _, p := range data.ReworkCycles {
		require.Regexp(t, `^\d{4}-\d{2}$`, p.YearMonth)
		require.GreaterOrEqual(t, p.AvgCycles, 0.0)
		require.GreaterOrEqual(t, p.IssueCount, 0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsReworkCycles -v
```

Expected: FAIL.

- [ ] **Step 3: Add the query**

Add helper below `agingWIP`:

```go
// reworkCycles returns the average number of QA→Dev transitions per closed
// issue, bucketed by close month. Reuses the pattern from ProjectKPIs.
func (s *Postgres) reworkCycles(team string, fromMonth, toMonth string) ([]types.ReworkCyclesPoint, error) {
	result := []types.ReworkCyclesPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH issue_close_month AS (
			SELECT
				ss.issueid,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN `+doneStatuses+`
			AND i.project = $1
			AND i.type IN `+closeableTypes+`
			AND ss.datestarted >= $2::timestamptz
			AND ss.datestarted <  $3::timestamptz
			GROUP BY ss.issueid
		),
		qa_to_dev AS (
			SELECT
				icm.year_month,
				icm.issueid,
				COUNT(*) AS cycles
			FROM issue_close_month icm
			JOIN issue_transition it ON it.issueid = icm.issueid
			WHERE (it.fromstatus ILIKE '%qa%' OR it.fromstatus ILIKE '%test%')
			AND (it.tostatus ILIKE '%dev%' OR it.tostatus = 'In Progress' OR it.tostatus = 'In Development')
			GROUP BY icm.year_month, icm.issueid
		)
		SELECT
			icm.year_month,
			COALESCE(AVG(qtd.cycles), 0)::float8 AS avg_cycles,
			COUNT(DISTINCT qtd.issueid) AS issue_count
		FROM issue_close_month icm
		LEFT JOIN qa_to_dev qtd ON qtd.issueid = icm.issueid
		GROUP BY icm.year_month
		ORDER BY icm.year_month`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the AgingWIP call:

```go
data.ReworkCycles, err = s.reworkCycles(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsReworkCycles -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Rework Cycles sub-query to ManagerMetrics"
```

---

## Task 9: Add Status Bounce Rate sub-query

Status Bounce Rate = re-entries / total transitions, by month, for closed issues. Re-entries means the same status was entered more than once on the same issue.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsStatusBounce(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.StatusBounce)
	for _, p := range data.StatusBounce {
		require.GreaterOrEqual(t, p.BouncePct, 0.0)
		require.LessOrEqual(t, p.BouncePct, 100.0)
		require.GreaterOrEqual(t, p.TotalIssues, 0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsStatusBounce -v
```

Expected: FAIL.

- [ ] **Step 3: Add the query**

Add helper below `reworkCycles`:

```go
// statusBounce returns monthly bounce rate (status re-entries / total
// transitions) for the team's closed issues.
func (s *Postgres) statusBounce(team string, fromMonth, toMonth string) ([]types.BouncePoint, error) {
	result := []types.BouncePoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH issue_close_month AS (
			SELECT
				ss.issueid,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN `+doneStatuses+`
			AND i.project = $1
			AND i.type IN `+closeableTypes+`
			AND ss.datestarted >= $2::timestamptz
			AND ss.datestarted <  $3::timestamptz
			GROUP BY ss.issueid
		),
		per_month AS (
			SELECT
				icm.year_month,
				COUNT(*) AS total_entries,
				SUM(CASE WHEN status_count > 1 THEN status_count - 1 ELSE 0 END) AS re_entries,
				COUNT(DISTINCT icm.issueid) AS total_issues
			FROM issue_close_month icm
			JOIN (
				SELECT issueid, tostatus, COUNT(*) AS status_count
				FROM issue_transition
				GROUP BY issueid, tostatus
			) tx ON tx.issueid = icm.issueid
			GROUP BY icm.year_month
		)
		SELECT
			year_month,
			CASE
				WHEN total_entries > 0
				THEN 100.0 * re_entries::float8 / total_entries::float8
				ELSE 0
			END AS bounce_pct,
			total_issues
		FROM per_month
		ORDER BY year_month`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the ReworkCycles call:

```go
data.StatusBounce, err = s.statusBounce(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsStatusBounce -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Status Bounce Rate sub-query to ManagerMetrics"
```

---

## Task 10: Add QA vs Eng Hours sub-query

QA vs Eng Hours = sum of `worklog.timespenthours` where `people.role='qa'`, divided by sum where `people.role='dev'`, per (worklog.date month).

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsQAvsEngHours(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.QAvsEngHours)
	for _, p := range data.QAvsEngHours {
		require.GreaterOrEqual(t, p.Numerator, 0.0)
		require.GreaterOrEqual(t, p.Denominator, 0.0)
		require.GreaterOrEqual(t, p.Ratio, 0.0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsQAvsEngHours -v
```

Expected: FAIL.

- [ ] **Step 3: Add the query**

Add helper below `statusBounce`:

```go
// qaVsEngHours returns the monthly ratio of QA hours to Eng hours for the
// team's issues (worklog.date month bucketing). Numerator = QA-role hours,
// Denominator = Dev-role hours.
func (s *Postgres) qaVsEngHours(team string, fromMonth, toMonth string) ([]types.HoursRatioPoint, error) {
	result := []types.HoursRatioPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		hours AS (
			SELECT
				to_char(w.date, 'YYYY-MM') AS year_month,
				SUM(CASE WHEN p.role = 'qa'  THEN w.timespenthours ELSE 0 END)::float8 AS qa_hours,
				SUM(CASE WHEN p.role = 'dev' THEN w.timespenthours ELSE 0 END)::float8 AS dev_hours
			FROM worklog w
			JOIN issue i  ON i.id = w.issueid
			LEFT JOIN people p ON p.name = w.author
			WHERE i.project = $1
			AND w.date >= $2::timestamptz
			AND w.date <  $3::timestamptz
			GROUP BY to_char(w.date, 'YYYY-MM')
		)
		SELECT
			m.year_month,
			COALESCE(h.qa_hours, 0)  AS numerator_hours,
			COALESCE(h.dev_hours, 0) AS denominator_hours,
			CASE
				WHEN COALESCE(h.dev_hours, 0) > 0
				THEN h.qa_hours / h.dev_hours
				ELSE 0
			END AS ratio
		FROM months m
		LEFT JOIN hours h ON h.year_month = m.year_month
		ORDER BY m.year_month`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the StatusBounce call:

```go
data.QAvsEngHours, err = s.qaVsEngHours(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsQAvsEngHours -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add QA-vs-Eng Hours sub-query to ManagerMetrics"
```

---

## Task 11: Add Bug Hours vs Forward-Work Hours sub-query

Bug Hours vs Forward-Work Hours = sum of hours on `Bug` + `Hardware Bug` issues, divided by sum of hours on `Story` + `Task` issues, per worklog month.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestManagerMetricsBugVsForwardHours(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	data, err := repo.ManagerMetrics("SYM", "", "")
	require.NoError(t, err)
	require.NotNil(t, data.BugvsForwardHours)
	for _, p := range data.BugvsForwardHours {
		require.GreaterOrEqual(t, p.Numerator, 0.0)
		require.GreaterOrEqual(t, p.Denominator, 0.0)
		require.GreaterOrEqual(t, p.Ratio, 0.0)
	}
}
```

- [ ] **Step 2: Run, confirm FAIL**

```bash
go test ./test/ -run TestManagerMetricsBugVsForwardHours -v
```

Expected: FAIL.

- [ ] **Step 3: Add the query**

Add helper below `qaVsEngHours`:

```go
// bugVsForwardHours returns the monthly ratio of bug-fix hours to forward-
// work (Story + Task) hours for the team. Bucketed by worklog.date month.
// Numerator = Bug + Hardware Bug hours. Denominator = Story + Task hours.
func (s *Postgres) bugVsForwardHours(team string, fromMonth, toMonth string) ([]types.HoursRatioPoint, error) {
	result := []types.HoursRatioPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		hours AS (
			SELECT
				to_char(w.date, 'YYYY-MM') AS year_month,
				SUM(CASE WHEN i.type IN ('Bug','Hardware Bug')  THEN w.timespenthours ELSE 0 END)::float8 AS bug_hours,
				SUM(CASE WHEN i.type IN ('Story','Task')        THEN w.timespenthours ELSE 0 END)::float8 AS fwd_hours
			FROM worklog w
			JOIN issue i ON i.id = w.issueid
			WHERE i.project = $1
			AND w.date >= $2::timestamptz
			AND w.date <  $3::timestamptz
			GROUP BY to_char(w.date, 'YYYY-MM')
		)
		SELECT
			m.year_month,
			COALESCE(h.bug_hours, 0) AS numerator_hours,
			COALESCE(h.fwd_hours, 0) AS denominator_hours,
			CASE
				WHEN COALESCE(h.fwd_hours, 0) > 0
				THEN h.bug_hours / h.fwd_hours
				ELSE 0
			END AS ratio
		FROM months m
		LEFT JOIN hours h ON h.year_month = m.year_month
		ORDER BY m.year_month`

	err := s.DB.Select(&result, query, team, from, to)
	return result, err
}
```

In `ManagerMetrics`, add after the QAvsEngHours call:

```go
data.BugvsForwardHours, err = s.bugVsForwardHours(team, fromMonth, toMonth)
if err != nil {
	return data, err
}
```

- [ ] **Step 4: Run test**

```bash
go build ./... && go test ./test/ -run TestManagerMetricsBugVsForwardHours -v
```

Expected: PASS.

- [ ] **Step 5: Run ALL ManagerMetrics tests for regression check**

```bash
go test ./test/ -run TestManagerMetrics -v
```

Expected: 8 tests PASS (Skeleton + TimeInStatus + WIPSeries + AgingWIP + ReworkCycles + StatusBounce + QAvsEngHours + BugVsForwardHours).

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Bug-vs-Forward-Work Hours sub-query to ManagerMetrics"
```

---

## Task 12: Create manager.templ page skeleton

Create the outer page shell: team selector, three section headers, back-to-leadership link. Sections are filled in later tasks.

**Files:**
- Create: `templates/pages/manager.templ`

- [ ] **Step 1: Create the file**

```go
package pages

import (
	"encoding/json"
	"fmt"

	"github.com/mkobaly/jiraworklog/templates/layouts"
	"github.com/mkobaly/jiraworklog/types"
)

// ManagerDashboard renders the single-team detail page. `data.Team` is the
// currently selected team; `teams` is the list of all teams for the
// dropdown selector.
templ ManagerDashboard(data types.ManagerMetricsData, teams []string) {
	@layouts.Base("Manager Dashboard") {
		<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>

		<div class="space-y-6">
			<!-- Header -->
			<div class="bg-white rounded-lg shadow p-6 flex items-center justify-between flex-wrap gap-3">
				<div class="flex items-center gap-3">
					<h1 class="text-2xl font-bold text-gray-800">Manager Dashboard</h1>
					<span class="text-lg text-gray-500">— { data.Team }</span>
					<button
						type="button"
						onclick="document.getElementById('mgrHelpModal').classList.remove('hidden')"
						class="w-7 h-7 rounded-full border border-gray-300 text-gray-500 hover:bg-gray-100 hover:text-gray-700 transition flex items-center justify-center text-sm font-semibold"
						aria-label="Help: what these metrics mean"
						title="What do these metrics mean?"
					>?</button>
				</div>
				<form method="GET" action="/dashboard/manager" class="flex items-center gap-2">
					<label for="team" class="text-sm font-medium text-gray-700">Team</label>
					<select id="team" name="team" onchange="this.form.submit()" class="border border-gray-300 rounded-md px-3 py-1.5 text-sm">
						for _, t := range teams {
							<option value={ t } if t == data.Team { selected }>{ t }</option>
						}
					</select>
				</form>
			</div>

			<!-- Delivery Predictability section (KPI cards + Time-in-Status + WIP + Aging WIP) -->
			<section>
				<h2 class="text-lg font-bold text-gray-800 mb-3">Delivery Predictability</h2>
				<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
					{ fmt.Sprintf("Predictability section — %d months of leadership data, %d time-in-status points, %d WIP points, %d aging WIP items",
						len(data.LeadershipMonthly), len(data.TimeInStatus), len(data.WIPSeries), len(data.AgingWIPItems)) }
				</div>
			</section>

			<!-- Quality section (Rework, Bounce, QA/Eng, Bug/Fwd cards) -->
			<section>
				<h2 class="text-lg font-bold text-gray-800 mb-3">Quality</h2>
				<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
					{ fmt.Sprintf("Quality section — %d rework points, %d bounce points, %d QA/Eng points, %d Bug/Fwd points",
						len(data.ReworkCycles), len(data.StatusBounce), len(data.QAvsEngHours), len(data.BugvsForwardHours)) }
				</div>
			</section>

			<!-- Stability section -->
			<section>
				<h2 class="text-lg font-bold text-gray-800 mb-3">Stability</h2>
				<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
					Stability section placeholder (Task 16 replaces this).
				</div>
			</section>

			<!-- Back link -->
			<div>
				<a href="/dashboard/leadership" class="text-sm text-blue-600 hover:underline">← Back to Leadership Dashboard</a>
			</div>

			@mgrHelpModal()
		</div>
	}
}

// mgrHelpModal is the manager-page help glossary. Phase 2 Task 18 adds the
// metric cards. For now this is a hidden placeholder so the button has
// somewhere to open.
templ mgrHelpModal() {
	<div
		id="mgrHelpModal"
		class="hidden fixed inset-0 z-50 flex items-center justify-center p-4"
		onclick="if (event.target === this) this.classList.add('hidden')"
	>
		<div class="fixed inset-0 bg-black bg-opacity-50"></div>
		<div class="relative bg-white rounded-lg shadow-2xl max-w-3xl w-full max-h-[85vh] overflow-y-auto">
			<div class="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
				<h2 class="text-xl font-bold text-gray-800">What these metrics mean</h2>
				<button type="button" onclick="document.getElementById('mgrHelpModal').classList.add('hidden')" class="text-gray-400 hover:text-gray-700 text-2xl leading-none px-2" aria-label="Close">&times;</button>
			</div>
			<div class="px-6 py-5 space-y-6 text-sm text-gray-700">
				Help content goes here (Task 18).
			</div>
		</div>
	</div>
}

// mgrFmtDays / mgrFmtPct / mgrFmtCount are local formatters mirroring the
// leadership-page helpers but renamed to avoid collisions across .templ files
// in the same package.
func mgrFmtDays(secs float64) string {
	if secs <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.1fd", secs/86400)
}

func mgrFmtPct(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

func mgrFmtCount(n int) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

func mgrFmtRatio(r float64) string {
	return fmt.Sprintf("%.2f", r)
}

// mgrSerialize returns a JSON blob of the full ManagerMetricsData for client-
// side chart bootstrapping. Charts pick out the slices they need.
func mgrSerialize(data types.ManagerMetricsData) string {
	b, _ := json.Marshal(data)
	return string(b)
}
```

- [ ] **Step 2: Generate templ**

```bash
templ generate
```

Expected: regenerates; `templates/pages/manager_templ.go` created.

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add manager dashboard page skeleton"
```

---

## Task 13: Add Delivery Predictability KPI cards (4 metrics with 13-month bar charts)

Replace the placeholder for the Delivery Predictability section. Show the 4 leadership KPIs as cards (Cycle Time, Throughput, Flow Efficiency, Failed QA Ratio) each with a 13-month bar/line chart bigger than the leadership-page sparklines.

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Replace the placeholder block**

Find this block in `templates/pages/manager.templ`:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Delivery Predictability</h2>
	<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
		{ fmt.Sprintf("Predictability section — %d months of leadership data, %d time-in-status points, %d WIP points, %d aging WIP items",
			len(data.LeadershipMonthly), len(data.TimeInStatus), len(data.WIPSeries), len(data.AgingWIPItems)) }
	</div>
</section>
```

Replace with:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Delivery Predictability</h2>
	<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
		@mgrKpiCard("Cycle Time (median)", mgrFmtDays(currentMgrCycle(data)), "mgr-cycle")
		@mgrKpiCard("Throughput", mgrFmtCount(currentMgrThroughput(data)), "mgr-throughput")
		@mgrKpiCard("Flow Efficiency", mgrFmtPct(currentMgrFlow(data)), "mgr-flow")
		@mgrKpiCard("Failed QA Ratio", mgrFmtPct(currentMgrFailedQA(data)), "mgr-failedqa")
	</div>
</section>
```

- [ ] **Step 2: Add the `mgrKpiCard` templ component above `ManagerDashboard`**

Insert before `templ ManagerDashboard(...) {`:

```go
// mgrKpiCard renders a single KPI card with a current value and a 13-month
// chart canvas. chartID is the DOM id the bootstrap script looks for.
templ mgrKpiCard(label, value, chartID string) {
	<div class="bg-white rounded-lg shadow p-5">
		<p class="text-xs text-gray-500 uppercase tracking-wide">{ label }</p>
		<p class="text-2xl font-bold text-gray-800 mt-1">{ value }</p>
		<div class="h-20 mt-3">
			<canvas id={ chartID }></canvas>
		</div>
	</div>
}
```

- [ ] **Step 3: Add the current-value helpers at the bottom of the file**

```go
// currentMgrCycle returns the median cycle time (seconds) for the most
// recent month in the LeadershipMonthly slice.
func currentMgrCycle(data types.ManagerMetricsData) float64 {
	if len(data.LeadershipMonthly) == 0 {
		return 0
	}
	return data.LeadershipMonthly[len(data.LeadershipMonthly)-1].MedianCycleTimeSecs
}

func currentMgrThroughput(data types.ManagerMetricsData) int {
	if len(data.LeadershipMonthly) == 0 {
		return 0
	}
	return data.LeadershipMonthly[len(data.LeadershipMonthly)-1].ClosedIssueCount
}

func currentMgrFlow(data types.ManagerMetricsData) float64 {
	if len(data.LeadershipMonthly) == 0 {
		return 0
	}
	return data.LeadershipMonthly[len(data.LeadershipMonthly)-1].FlowEfficiencyPct
}

func currentMgrFailedQA(data types.ManagerMetricsData) float64 {
	if len(data.LeadershipMonthly) == 0 {
		return 0
	}
	return data.LeadershipMonthly[len(data.LeadershipMonthly)-1].FailedQARatioPct
}
```

- [ ] **Step 4: Add the bootstrap script for the 4 KPI cards**

Inside `ManagerDashboard`, just before the closing `</div>` that closes the outermost `space-y-6` div (so it runs after all chart canvases exist), add:

```go
<div id="mgrData" data-payload={ mgrSerialize(data) } style="display:none;"></div>
@mgrLeadershipChartsScript()
```

Then add a new templ component at the bottom of the file:

```go
templ mgrLeadershipChartsScript() {
	<script>
		(function() {
			const data = JSON.parse(document.getElementById('mgrData').dataset.payload || '{}');
			const lm = data.LeadershipMonthly || [];
			const labels = lm.map(function(r) { return r.YearMonth; });
			const series = {
				'mgr-cycle':      lm.map(function(r) { return r.MedianCycleTimeSecs / 86400; }),
				'mgr-throughput': lm.map(function(r) { return r.ClosedIssueCount; }),
				'mgr-flow':       lm.map(function(r) { return r.FlowEfficiencyPct; }),
				'mgr-failedqa':   lm.map(function(r) { return r.FailedQARatioPct; }),
			};
			Object.keys(series).forEach(function(id) {
				const canvas = document.getElementById(id);
				if (!canvas) return;
				new Chart(canvas, {
					type: 'bar',
					data: {
						labels: labels,
						datasets: [{
							data: series[id],
							backgroundColor: '#3b82f6',
							borderColor: '#1d4ed8',
							borderWidth: 1,
						}],
					},
					options: {
						responsive: true,
						maintainAspectRatio: false,
						plugins: { legend: { display: false }, tooltip: { enabled: true } },
						scales: {
							x: { ticks: { font: { size: 8 }, maxRotation: 45, minRotation: 45 } },
							y: { beginAtZero: true, ticks: { font: { size: 9 } } },
						},
					},
				});
			});
		})();
	</script>
}
```

- [ ] **Step 5: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 6: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add Delivery Predictability KPI cards to manager page"
```

---

## Task 14: Add Time in Status stacked bar chart

Adds a stacked-bar chart below the Predictability KPI cards showing avg time-in-status per (month, bucket).

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Add the chart canvas inside the Predictability section**

In `ManagerDashboard`, find the closing tag of the Predictability section (the `</section>` after the KPI card grid). BEFORE that `</section>`, add:

```go
<div class="bg-white rounded-lg shadow p-6 mt-4">
	<h3 class="text-base font-semibold text-gray-700 mb-1">Time in Status</h3>
	<p class="text-xs text-gray-400 mb-3">Average time issues spent in each bucket per month (Dev / QA / Waiting).</p>
	<div class="h-72">
		<canvas id="mgr-tis"></canvas>
	</div>
</div>
```

- [ ] **Step 2: Add the bootstrap script for the stacked bar**

Add a new templ component at the bottom of the file:

```go
templ mgrTimeInStatusScript() {
	<script>
		(function() {
			const data = JSON.parse(document.getElementById('mgrData').dataset.payload || '{}');
			const tis = data.TimeInStatus || [];
			// Pivot tis into { months: [...], Dev: [...], QA: [...], Waiting: [...] }
			const monthSet = new Set();
			tis.forEach(function(p) { monthSet.add(p.YearMonth); });
			const months = Array.from(monthSet).sort();
			const buckets = { Dev: {}, QA: {}, Waiting: {} };
			tis.forEach(function(p) { (buckets[p.Bucket] || {})[p.YearMonth] = p.AvgSecs / 86400; });
			const dataFor = function(bucket) {
				return months.map(function(m) { return buckets[bucket][m] || 0; });
			};
			const canvas = document.getElementById('mgr-tis');
			if (!canvas) return;
			new Chart(canvas, {
				type: 'bar',
				data: {
					labels: months,
					datasets: [
						{ label: 'Dev',     data: dataFor('Dev'),     backgroundColor: '#3b82f6' },
						{ label: 'QA',      data: dataFor('QA'),      backgroundColor: '#10b981' },
						{ label: 'Waiting', data: dataFor('Waiting'), backgroundColor: '#f59e0b' },
					],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: {
						legend: { position: 'bottom', labels: { font: { size: 11 } } },
						tooltip: { callbacks: { label: function(ctx) { return ctx.dataset.label + ': ' + ctx.parsed.y.toFixed(1) + 'd'; } } },
					},
					scales: {
						x: { stacked: true, ticks: { font: { size: 10 } } },
						y: { stacked: true, beginAtZero: true, ticks: { font: { size: 10 }, callback: function(v) { return v + 'd'; } } },
					},
				},
			});
		})();
	</script>
}
```

- [ ] **Step 3: Invoke `@mgrTimeInStatusScript()` from `ManagerDashboard`**

Just after `@mgrLeadershipChartsScript()`, add:

```go
@mgrTimeInStatusScript()
```

- [ ] **Step 4: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 5: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add Time-in-Status stacked bar to manager page"
```

---

## Task 15: Add WIP trend chart and Aging WIP list

Two parallel UI elements inside the Predictability section: a WIP trend line chart and a list of stale issues.

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Add the WIP + Aging block inside the Predictability section**

In `ManagerDashboard`, inside the Predictability `<section>`, AFTER the Time in Status block and BEFORE the closing `</section>`, add:

```go
<div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mt-4">
	<div class="bg-white rounded-lg shadow p-6 lg:col-span-2">
		<h3 class="text-base font-semibold text-gray-700 mb-1">WIP Trend</h3>
		<p class="text-xs text-gray-400 mb-3">Open issues in Dev or QA on each day, last 13 months.</p>
		<div class="h-56">
			<canvas id="mgr-wip"></canvas>
		</div>
	</div>
	<div class="bg-white rounded-lg shadow p-6">
		<h3 class="text-base font-semibold text-gray-700 mb-1">Aging WIP</h3>
		<p class="text-xs text-gray-400 mb-3">Currently open issues exceeding the team's 85th-percentile cycle time. Top 20.</p>
		if len(data.AgingWIPItems) == 0 {
			<p class="text-sm text-gray-400 italic">No aging items — team is healthy.</p>
		} else {
			<ul class="divide-y divide-gray-100 text-sm">
				for _, item := range data.AgingWIPItems {
					<li class="py-2 flex items-center justify-between gap-2">
						<span class="font-mono text-blue-600">{ item.Key }</span>
						<span class="text-gray-500 text-xs">{ item.Status }</span>
						<span class="font-medium text-gray-700">{ fmt.Sprintf("%dd", item.DaysInStatus) }</span>
					</li>
				}
			</ul>
		}
	</div>
</div>
```

- [ ] **Step 2: Add the WIP bootstrap script**

Add a new templ component at the bottom of the file:

```go
templ mgrWIPScript() {
	<script>
		(function() {
			const data = JSON.parse(document.getElementById('mgrData').dataset.payload || '{}');
			const wip = data.WIPSeries || [];
			const canvas = document.getElementById('mgr-wip');
			if (!canvas) return;
			new Chart(canvas, {
				type: 'line',
				data: {
					labels: wip.map(function(p) { return p.Day; }),
					datasets: [{
						data: wip.map(function(p) { return p.WIPCount; }),
						borderColor: '#6366f1',
						backgroundColor: 'rgba(99,102,241,0.1)',
						fill: true,
						pointRadius: 0,
						tension: 0.25,
						borderWidth: 1.5,
					}],
				},
				options: {
					responsive: true,
					maintainAspectRatio: false,
					plugins: { legend: { display: false }, tooltip: { enabled: true } },
					scales: {
						x: { ticks: { font: { size: 9 }, maxTicksLimit: 13 } },
						y: { beginAtZero: true, ticks: { font: { size: 10 } } },
					},
				},
			});
		})();
	</script>
}
```

- [ ] **Step 3: Invoke the new script from `ManagerDashboard`**

After `@mgrTimeInStatusScript()`, add:

```go
@mgrWIPScript()
```

- [ ] **Step 4: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 5: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add WIP trend chart and Aging WIP list to manager page"
```

---

## Task 16: Add Quality section KPI cards

Replace the Quality section placeholder with four KPI cards: Rework Cycles, Status Bounce, QA-vs-Eng Hours, Bug-vs-Forward-Work Hours. Each has a 13-month line chart.

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Replace the Quality section placeholder**

Find:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Quality</h2>
	<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
		{ fmt.Sprintf("Quality section — %d rework points, %d bounce points, %d QA/Eng points, %d Bug/Fwd points",
			len(data.ReworkCycles), len(data.StatusBounce), len(data.QAvsEngHours), len(data.BugvsForwardHours)) }
	</div>
</section>
```

Replace with:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Quality</h2>
	<div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
		@mgrKpiCard("Rework Cycles", mgrFmtRatio(currentMgrRework(data)), "mgr-rework")
		@mgrKpiCard("Status Bounce", mgrFmtPct(currentMgrBounce(data)), "mgr-bounce")
		@mgrKpiCard("QA hrs / Eng hrs", mgrFmtRatio(currentMgrQAvsEng(data)), "mgr-qaeng")
		@mgrKpiCard("Bug hrs / Fwd-work hrs", mgrFmtRatio(currentMgrBugFwd(data)), "mgr-bugfwd")
	</div>
</section>
```

- [ ] **Step 2: Add the current-value helpers at the bottom of the file**

```go
func currentMgrRework(data types.ManagerMetricsData) float64 {
	if len(data.ReworkCycles) == 0 {
		return 0
	}
	return data.ReworkCycles[len(data.ReworkCycles)-1].AvgCycles
}

func currentMgrBounce(data types.ManagerMetricsData) float64 {
	if len(data.StatusBounce) == 0 {
		return 0
	}
	return data.StatusBounce[len(data.StatusBounce)-1].BouncePct
}

func currentMgrQAvsEng(data types.ManagerMetricsData) float64 {
	if len(data.QAvsEngHours) == 0 {
		return 0
	}
	return data.QAvsEngHours[len(data.QAvsEngHours)-1].Ratio
}

func currentMgrBugFwd(data types.ManagerMetricsData) float64 {
	if len(data.BugvsForwardHours) == 0 {
		return 0
	}
	return data.BugvsForwardHours[len(data.BugvsForwardHours)-1].Ratio
}
```

- [ ] **Step 3: Add the Quality charts bootstrap script**

Add a new templ component at the bottom of the file:

```go
templ mgrQualityChartsScript() {
	<script>
		(function() {
			const data = JSON.parse(document.getElementById('mgrData').dataset.payload || '{}');
			const series = {
				'mgr-rework': {
					labels: (data.ReworkCycles || []).map(function(p) { return p.YearMonth; }),
					values: (data.ReworkCycles || []).map(function(p) { return p.AvgCycles; }),
				},
				'mgr-bounce': {
					labels: (data.StatusBounce || []).map(function(p) { return p.YearMonth; }),
					values: (data.StatusBounce || []).map(function(p) { return p.BouncePct; }),
				},
				'mgr-qaeng': {
					labels: (data.QAvsEngHours || []).map(function(p) { return p.YearMonth; }),
					values: (data.QAvsEngHours || []).map(function(p) { return p.Ratio; }),
				},
				'mgr-bugfwd': {
					labels: (data.BugvsForwardHours || []).map(function(p) { return p.YearMonth; }),
					values: (data.BugvsForwardHours || []).map(function(p) { return p.Ratio; }),
				},
			};
			Object.keys(series).forEach(function(id) {
				const canvas = document.getElementById(id);
				if (!canvas) return;
				new Chart(canvas, {
					type: 'line',
					data: {
						labels: series[id].labels,
						datasets: [{
							data: series[id].values,
							borderColor: '#8b5cf6',
							backgroundColor: 'rgba(139,92,246,0.1)',
							fill: true,
							pointRadius: 0,
							tension: 0.3,
							borderWidth: 1.5,
						}],
					},
					options: {
						responsive: true,
						maintainAspectRatio: false,
						plugins: { legend: { display: false }, tooltip: { enabled: true } },
						scales: {
							x: { ticks: { font: { size: 8 }, maxRotation: 45, minRotation: 45 } },
							y: { beginAtZero: true, ticks: { font: { size: 9 } } },
						},
					},
				});
			});
		})();
	</script>
}
```

- [ ] **Step 4: Invoke the new script**

After `@mgrWIPScript()`, add:

```go
@mgrQualityChartsScript()
```

- [ ] **Step 5: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 6: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add Quality KPI cards to manager page"
```

---

## Task 17: Add Stability section (customer bug chart for selected team)

Replace the Stability section placeholder with a single customer-bug trend chart for the selected team plus a Defect Escape Rate trend chart.

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Replace the Stability section placeholder**

Find:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Stability</h2>
	<div class="bg-white rounded-lg shadow p-6 text-sm text-gray-500">
		Stability section placeholder (Task 16 replaces this).
	</div>
</section>
```

Replace with:

```go
<section>
	<h2 class="text-lg font-bold text-gray-800 mb-3">Stability</h2>
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
		<div class="bg-white rounded-lg shadow p-6">
			<h3 class="text-base font-semibold text-gray-700 mb-1">Customer Bugs</h3>
			<p class="text-xs text-gray-400 mb-3">New bugs opened, bugs closed, total open bugs per month.</p>
			<div class="h-72">
				<canvas id="mgr-stability"></canvas>
			</div>
		</div>
		<div class="bg-white rounded-lg shadow p-6">
			<h3 class="text-base font-semibold text-gray-700 mb-1">Defect Escape Rate</h3>
			<p class="text-xs text-gray-400 mb-3">% of bugs found this month that came from customers vs were found internally.</p>
			<div class="h-72">
				<canvas id="mgr-escape"></canvas>
			</div>
		</div>
	</div>
</section>
```

- [ ] **Step 2: Add the stability charts bootstrap script**

Add a new templ component at the bottom of the file:

```go
templ mgrStabilityChartsScript() {
	<script>
		(function() {
			const data = JSON.parse(document.getElementById('mgrData').dataset.payload || '{}');
			const lm = data.LeadershipMonthly || [];
			const labels = lm.map(function(r) { return r.YearMonth; });

			// Customer bug trend
			const stab = document.getElementById('mgr-stability');
			if (stab) {
				new Chart(stab, {
					type: 'line',
					data: {
						labels: labels,
						datasets: [
							{ label: 'New',    data: lm.map(function(r) { return r.StabilityNewCount; }),    borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.1)', tension: 0.3 },
							{ label: 'Closed', data: lm.map(function(r) { return r.StabilityClosedCount; }), borderColor: '#10b981', backgroundColor: 'rgba(16,185,129,0.1)', tension: 0.3 },
							{ label: 'Open',   data: lm.map(function(r) { return r.StabilityOpenCount; }),   borderColor: '#f59e0b', backgroundColor: 'rgba(245,158,11,0.1)', tension: 0.3 },
						],
					},
					options: {
						responsive: true,
						maintainAspectRatio: false,
						plugins: { legend: { position: 'bottom', labels: { font: { size: 11 } } } },
						scales: {
							x: { ticks: { font: { size: 10 }, maxRotation: 45, minRotation: 45 } },
							y: { beginAtZero: true, ticks: { font: { size: 10 } } },
						},
					},
				});
			}

			// Defect Escape Rate trend
			const esc = document.getElementById('mgr-escape');
			if (esc) {
				new Chart(esc, {
					type: 'bar',
					data: {
						labels: labels,
						datasets: [{
							data: lm.map(function(r) { return r.DefectEscapeRatePct; }),
							backgroundColor: '#ef4444',
							borderColor: '#b91c1c',
							borderWidth: 1,
						}],
					},
					options: {
						responsive: true,
						maintainAspectRatio: false,
						plugins: { legend: { display: false }, tooltip: { enabled: true } },
						scales: {
							x: { ticks: { font: { size: 10 }, maxRotation: 45, minRotation: 45 } },
							y: { beginAtZero: true, max: 100, ticks: { font: { size: 10 }, callback: function(v) { return v + '%'; } } },
						},
					},
				});
			}
		})();
	</script>
}
```

- [ ] **Step 3: Invoke the new script**

After `@mgrQualityChartsScript()`, add:

```go
@mgrStabilityChartsScript()
```

- [ ] **Step 4: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 5: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): add Stability section (customer bug + escape rate) to manager page"
```

---

## Task 18: Fill in the help modal glossary

Replace the placeholder text in `mgrHelpModal` with metric cards for all eight metrics shown on the manager page (4 Predictability + Time-in-Status + 4 Quality + Defect Escape + Stability), each with What / How / So what.

**Files:**
- Modify: `templates/pages/manager.templ`

- [ ] **Step 1: Replace the modal body**

Find:

```go
<div class="px-6 py-5 space-y-6 text-sm text-gray-700">
	Help content goes here (Task 18).
</div>
```

Replace with:

```go
<div class="px-6 py-5 space-y-6 text-sm text-gray-700">
	@mgrHelpCard(
		"Cycle Time (median)",
		"Median time issues spend actively moving through Development and QA — the \"typical\" elapsed time from dev start to QA done.",
		"Sum of time in Dev statuses plus QA statuses, per issue. Median across all issues closed in the month.",
		"Lower = faster delivery. A rising trend is an early warning of bottlenecks (more bugs, more rework, harder problems, distractions).",
	)
	@mgrHelpCard(
		"Throughput",
		"Count of issues closed in the month for this team.",
		"Issues of type Story / Bug / Hardware Bug / Customer Bug / HW-FW Customer Bug that entered Done / Closed / Cancelled / Awaiting Release for the first time in the month.",
		"Steady throughput + steady cycle time = predictable team. Erratic throughput = lumpy delivery, hard to plan around.",
	)
	@mgrHelpCard(
		"Flow Efficiency",
		"Percent of an issue's elapsed time spent actively progressing vs waiting.",
		"Active (Dev + QA) seconds / total (Dev + QA + Waiting + Done) seconds, summed across closed issues in the month.",
		"Low flow efficiency means most cycle time is waiting in queues. High = clean handoffs and low queue time.",
	)
	@mgrHelpCard(
		"Failed QA Ratio",
		"Percent of Stories closed this month that failed QA at least once.",
		"Numerator = stories closed in month that hit Failed QA status. Denominator = all stories closed in month. Bugs and Tasks excluded.",
		"Your \"first-time-right\" rate. High = stories aren't actually ready when devs hand them off.",
	)
	@mgrHelpCard(
		"Time in Status",
		"Average time issues spent in each flow bucket (Dev / QA / Waiting) per month.",
		"Avg of `status_stints.durationseconds`, grouped by the month the stint ended and the bucket. Stacked bars show the composition over time.",
		"Surfaces where the team's time is actually going. Big QA stack = QA bottleneck. Big Waiting stack = handoff or blocker problem.",
	)
	@mgrHelpCard(
		"WIP & Aging WIP",
		"WIP = count of open issues currently in Dev or QA. Aging WIP = the subset that have been stuck longer than the team's typical cycle time.",
		"WIP series sampled daily. Aging WIP threshold = team's 85th-percentile cycle time over the last 13 months. List capped at 20.",
		"Rising WIP without rising throughput = work piling up. Aging WIP items are the specific tickets stuck — usually the place to start digging when delivery slows.",
	)
	@mgrHelpCard(
		"Rework Cycles",
		"Average QA→Dev transitions per closed issue in the month.",
		"For issues closed in the month, count the number of times the issue transitioned from a QA/test status back to a dev status. Average across those issues.",
		"When stuff fails QA, how badly does it thrash? Rising rework cycles is the deeper-dive signal behind Failed QA Ratio.",
	)
	@mgrHelpCard(
		"Status Bounce Rate",
		"How often issues re-enter the same status (status thrash).",
		"Re-entries = times an issue's status was entered more than once. Bounce rate = re-entries / total transitions per closed issue, averaged for the month.",
		"High bounce = unclear workflow, requirement churn, or thrash. Useful for catching process problems separately from QA-specific rework.",
	)
	@mgrHelpCard(
		"QA hrs / Eng hrs",
		"Ratio of hours logged by people with role=qa to hours logged by people with role=dev, per month.",
		"Sum of worklog.timespenthours where author has role 'qa', divided by the same sum for role 'dev'. Bucketed by worklog date.",
		"Indicates QA capacity relative to dev capacity. A persistent low ratio (e.g. <0.2) may mean QA is undersized and creates a downstream bottleneck.",
	)
	@mgrHelpCard(
		"Bug hrs / Forward-work hrs",
		"Ratio of hours spent fixing internal bugs to hours spent on Story + Task work, per month.",
		"Sum of hours on type Bug + Hardware Bug, divided by sum of hours on type Story + Task. Bucketed by worklog date.",
		"High = team is firefighting more than delivering. Rising trend is an early warning of accumulating quality debt.",
	)
	@mgrHelpCard(
		"Stability (Customer Bugs) & Defect Escape Rate",
		"Customer-reported bug volume over time, plus the % of bugs found this month that came from customers vs internally.",
		"Customer Bug + HW/FW Customer Bug counts (new, closed, running open) per month. Escape Rate = customer bugs / (customer + internal bugs), by createdate month.",
		"The customer-pain signal. Open trend rising = customers filing faster than the team fixes. High escape rate = team isn't catching defects before release.",
	)

	<p class="text-xs text-gray-400 italic pt-2 border-t border-gray-100">
		"Forward work" = Story + Task. Tasks don't go through QA, so they're excluded from Failed QA Ratio but counted in throughput and the hours ratio.
	</p>
</div>
```

- [ ] **Step 2: Add the `mgrHelpCard` templ component**

Insert near `mgrHelpModal`:

```go
templ mgrHelpCard(name, what, how, soWhat string) {
	<div>
		<h3 class="text-base font-semibold text-gray-800 mb-2">{ name }</h3>
		<dl class="space-y-1.5">
			<div class="flex gap-2">
				<dt class="font-medium text-gray-600 whitespace-nowrap">What:</dt>
				<dd>{ what }</dd>
			</div>
			<div class="flex gap-2">
				<dt class="font-medium text-gray-600 whitespace-nowrap">How:</dt>
				<dd>{ how }</dd>
			</div>
			<div class="flex gap-2">
				<dt class="font-medium text-gray-600 whitespace-nowrap">So what:</dt>
				<dd>{ soWhat }</dd>
			</div>
		</dl>
	</div>
}
```

- [ ] **Step 3: Generate and build**

```bash
templ generate && go build ./...
```

Expected: succeeds.

- [ ] **Step 4: Commit**

```bash
git add templates/pages/manager.templ templates/pages/manager_templ.go
git commit -m "feat(templates): fill in manager-page help modal glossary"
```

---

## Task 19: Add GetManagerDashboard handler

**Files:**
- Modify: `cmd/jiraworklog/handlers.go`

- [ ] **Step 1: Add the handler method**

Open `cmd/jiraworklog/handlers.go`. Find the existing `GetLeadershipDashboard` method (added in Phase 1). Insert the new handler RIGHT AFTER it:

```go
// GET /dashboard/manager
func (h *Handler) GetManagerDashboard(c echo.Context) error {
	// v1: hard-coded team list matches the leadership page.
	teams := []string{"IDM", "SYM", "ESG"}

	team := c.QueryParam("team")
	if team == "" {
		team = teams[0] // default to first team alphabetically
	}

	data, err := h.repo.ManagerMetrics(team, "", "")
	if err != nil {
		h.logger.Error("error fetching manager metrics", "error", err, "team", team)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch manager metrics")
	}

	if wantsHTML(c) {
		return pages.ManagerDashboard(data, teams).
			Render(c.Request().Context(), c.Response().Writer)
	}
	return c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/jiraworklog/handlers.go
git commit -m "feat(handler): add GET /dashboard/manager handler"
```

---

## Task 20: Register the `/dashboard/manager` route

**Files:**
- Modify: `cmd/jiraworklog/main.go`

- [ ] **Step 1: Add the route inside the dashboard block**

Find:

```go
	e.GET("/dashboard/leadership", handler.GetLeadershipDashboard, handler.AuthMiddleware)
```

Add immediately after:

```go
	e.GET("/dashboard/manager", handler.GetManagerDashboard, handler.AuthMiddleware)
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/jiraworklog/main.go
git commit -m "feat(routes): register /dashboard/manager"
```

---

## Task 21: Smoke test

Bring up the app and confirm the manager page renders end-to-end for each team.

- [ ] **Step 1: Build**

```bash
cd /home/developer/code/github/jiraworklog && ./build.sh
```

Expected: succeeds.

- [ ] **Step 2: Start the server in the background**

```bash
./bin/jiraworklog -d -c bin/config.yaml > /tmp/manager_smoke.log 2>&1 &
SERVER_PID=$!
sleep 3
```

- [ ] **Step 3: Test default (no `?team=`)**

```bash
curl -s -o /tmp/mgr_default.html -w "%{http_code}\n" http://localhost:8380/dashboard/manager
```

Expected: 200 OK. Body should contain "Manager Dashboard", "IDM" (the default selected team), all four section headers (Delivery Predictability, Quality, Stability), and "Back to Leadership Dashboard".

- [ ] **Step 4: Test each team explicitly**

```bash
for t in IDM SYM ESG; do
  echo -n "Team $t: "
  curl -s -w "%{http_code}\n" -o /tmp/mgr_$t.html "http://localhost:8380/dashboard/manager?team=$t"
done
```

Expected: each prints 200.

- [ ] **Step 5: Verify markers in one response**

```bash
for marker in "Manager Dashboard" "Delivery Predictability" "Quality" "Stability" "Cycle Time" "Throughput" "Time in Status" "Rework Cycles" "Status Bounce" "QA hrs" "Bug hrs" "Customer Bugs" "Defect Escape" "Aging WIP" "WIP Trend" "Back to Leadership"; do
  count=$(grep -c "$marker" /tmp/mgr_SYM.html)
  echo "$marker: $count"
done
```

Expected: every marker count >= 1.

- [ ] **Step 6: Verify JSON variant**

```bash
curl -s -H "Accept: application/json" -w "\nstatus=%{http_code}\n" "http://localhost:8380/dashboard/manager?team=SYM" | head -3
```

Expected: 200 + JSON object (starts with `{`) containing `"Team":"SYM"`, `"LeadershipMonthly"`, `"TimeInStatus"`, etc.

- [ ] **Step 7: Verify the leadership page's team-header links work**

```bash
curl -s http://localhost:8380/dashboard/leadership | grep -o 'href="/dashboard/manager?team=[A-Z]*"' | sort -u
```

Expected: three hrefs printed — `href="/dashboard/manager?team=ESG"`, `href="/dashboard/manager?team=IDM"`, `href="/dashboard/manager?team=SYM"`. (These were added by Phase 1; this confirms they still resolve to the new route.)

- [ ] **Step 8: Stop server**

```bash
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null
```

- [ ] **Step 9: Verify no errors in the server log**

```bash
echo "=== Server log ==="
grep -E "ERROR|panic|500" /tmp/manager_smoke.log || echo "No errors found."
```

Expected: "No errors found." (or only known-unrelated background-sync errors).

- [ ] **Step 10: Cleanup**

```bash
rm -f /tmp/manager_smoke.log /tmp/mgr_*.html
```

- [ ] **Step 11: No commit needed if nothing changed**

`git status` should show no source changes outside `.claude/settings.local.json`.

---

## Done

Phase 2 complete. Manager page is fully wired: team selector, three sections, all seven new metrics rendered, help modal, back link to leadership. The leadership grid's team-header links now resolve to a real page.

**What's left for v2 (separate spec, not in this plan):**

- Split project-team vs stability-team within each Jira project via `projectCharge` code `3`
- Commitment Reliability (deferred from v1)
- Planned vs Unplanned Ratio (deferred from v1)
- Caching layer
- Per-issue drill-down page (currently links go to Jira)
