# Monthly Leadership Dashboard — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the `/dashboard/leadership` page — an all-teams comparison grid of 5 monthly KPIs (Cycle Time, Throughput, Flow Efficiency, Failed QA Ratio, Defect Escape Rate) plus a per-team Stability customer-bug trend strip, scoped to the IDM/SYM/ESG Jira projects.

**Architecture:** Two query entry points (`MonthlyTeamMetrics` for the cross-team aggregate, `CustomerBugTrends` already exists for Stability), one templ page, one Echo handler under `/dashboard/leadership`. All SQL reuses existing `issue`, `issue_transition`, `status_stints`, `worklog` tables — no schema changes. Shared SQL constants (status bucketing, issue-type filters) are lifted out of the existing per-epic queries so both code paths can't drift.

**Tech Stack:** Go, Echo v4, sqlx + pgx/v5, templ, Chart.js (via CDN), Tailwind, Hotwire Turbo, PostgreSQL.

**Reference:** Spec at `docs/superpowers/specs/2026-05-29-monthly-leadership-dashboard-design.md`.

---

## File Structure

**Files created:**
- `types/monthly_metrics.go` — type definitions for MonthlyTeamMetrics and helpers
- `templates/pages/leadership.templ` — page template (Phase 1 only — manager.templ is Phase 2)

**Files modified:**
- `repository/repo.go` — add `MonthlyTeamMetrics` to the `Repo` interface
- `repository/posgres.go` — add `MonthlyTeamMetrics()` implementation + lift shared SQL constants
- `cmd/jiraworklog/handlers.go` — add `GetLeadershipDashboard` handler
- `cmd/jiraworklog/main.go` — register `GET /dashboard/leadership` route
- `templates/layouts/base.templ` — add "Leadership" link to desktop + mobile nav
- `test/repo_test.go` — add integration tests for `MonthlyTeamMetrics`

**Out of scope (Phase 2, separate plan):**
- `/dashboard/manager` page
- Manager-only metrics: Time-in-Status, WIP series, Aging WIP, Rework Cycles, QA-vs-Eng hours, Bug-vs-Forward-Work hours, Status Bounce Rate

---

## Conventions used in this plan

- The project uses **integration tests against a real Postgres** (see `test/repo_test.go`). Tests load `bin/config.yaml` via `GetTestConfig()`. Existing pattern is loose smoke tests (`require.Greater(t, len(rows), 0)`). New tests follow the same pattern.
- Templ files are compiled with `templ generate` before `go build` / `go test`. The build script runs both. If you edit a `.templ` file, run `templ generate` (or `templ generate --watch` in a second terminal) to regenerate the `_templ.go` files.
- Commits are small and frequent. Each task ends with a commit. Sign-off line is whatever Git is configured to add automatically — do not add a `Co-Authored-By` line.

---

## Task 1: Lift shared SQL constants out of ProjectKPIs

Lifting the bucket and population filters into named Go constants so the new `MonthlyTeamMetrics` query and the existing `ProjectKPIs` query share one source of truth.

**Files:**
- Modify: `repository/posgres.go` (around lines 588-602 and around lines 633, 649-651, 687-692, 805)

- [ ] **Step 1: Add the two new SQL constants near `projectIssuesCTE` (currently at posgres.go:590)**

Insert immediately after the existing `projectIssuesCTE` declaration:

```go
// closeableTypes is the set of issue types that flow through Dev → QA and so
// participate in cycle-time/throughput/flow-efficiency/defect-escape metrics.
// Excludes Task (skips QA), Epic/Release Candidate (containers), and sub-tasks.
const closeableTypes = `('Story','Bug','Hardware Bug','Customer Bug','HW / FW Customer Bug')`

// bucketCase maps a status-stints `status` column to its flow bucket.
// Identical to the inline CASE used in ProjectKPIs — both code paths reference
// this constant so they cannot drift.
const bucketCase = `
    CASE
      WHEN status IN ('In Development','In Progress','Code Complete','Code Merged','In Review') THEN 'Dev'
      WHEN status IN ('In QA','Failed QA') THEN 'QA'
      WHEN status IN ('On Hold','QA Backlog') THEN 'Waiting'
      ELSE 'Other'
    END`

// devQAStatuses lists the statuses that count toward "active" cycle time —
// Dev bucket + QA bucket. Used by cycle-time and flow-efficiency queries.
const devQAStatuses = `('In Development','In Progress','Code Complete','Code Merged','In Review','In QA','Failed QA')`

// doneStatuses lists the statuses that mark an issue as closed.
const doneStatuses = `('Done','Closed','Cancelled','Awaiting Release to Customer')`
```

- [ ] **Step 2: Update `ProjectKPIs` cycle-time query at posgres.go:633 to reference `devQAStatuses`**

Find the existing line:

```go
AND status IN ('In Development','In Progress','Code Complete', 'Code Merged', 'In Review', 'In QA', 'Failed QA')
```

Replace with:

```go
AND status IN `+devQAStatuses+`
```

(Same change at posgres.go:756 inside the flow-efficiency query's `FILTER (WHERE status IN (...))` clause — replace the inline status list with `+devQAStatuses+`.)

- [ ] **Step 3: Update `ProjectKPIs` WIPHistory inner JOIN at posgres.go:805 to reference `devQAStatuses`**

Find:

```go
ON  ss.status IN ('In Development','In Progress','Code Complete', 'Code Merged' ,'In Review', 'In QA', 'Failed QA')
```

Replace with:

```go
ON  ss.status IN `+devQAStatuses+`
```

- [ ] **Step 4: Verify the file still builds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build succeeds with no errors.

- [ ] **Step 5: Run the existing ProjectKPIs integration test (if any) and confirm no regression**

Check if a `TestProjectKPIs` test exists: `grep -n "TestProjectKPIs" test/*.go`.

If it exists: `go test ./test/ -run TestProjectKPIs -v` — expected PASS.

If it doesn't exist: write a minimal smoke test in `test/repo_test.go`:

```go
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
```

Run it: `go test ./test/ -run TestProjectKPIsSmoke -v` — expected PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "refactor(repo): lift shared SQL constants for status/issue-type filtering"
```

---

## Task 2: Add MonthlyTeamMetrics type definitions

Create the type that drives the leadership page payload.

**Files:**
- Create: `types/monthly_metrics.go`

- [ ] **Step 1: Create `types/monthly_metrics.go`**

```go
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
```

- [ ] **Step 2: Verify the file builds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build succeeds.

- [ ] **Step 3: Commit**

```bash
git add types/monthly_metrics.go
git commit -m "feat(types): add MonthlyTeamMetrics for leadership dashboard payload"
```

---

## Task 3: Extend the Repo interface with MonthlyTeamMetrics

**Files:**
- Modify: `repository/repo.go`

- [ ] **Step 1: Add the interface method**

Find the line containing `ProjectKPIs(epicOrVersion string) (types.ProjectKPIData, error)` in `repository/repo.go`.

Add immediately after it:

```go
// MonthlyTeamMetrics returns one row per (team, year_month) for the leadership
// dashboard's all-teams comparison grid. `teams` is the list of Jira project
// prefixes to include (e.g. ["IDM","SYM","ESG"]); `fromMonth` and `toMonth`
// are YYYY-MM strings; empty strings default to last-13-months trailing.
MonthlyTeamMetrics(teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error)
```

- [ ] **Step 2: Verify the file builds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build fails with `*Postgres does not implement Repo (missing method MonthlyTeamMetrics)`. That's expected — Task 4 implements it. **Do not commit until Task 4 is complete.**

---

## Task 4: Implement MonthlyTeamMetrics — skeleton (months × teams grid only)

Build the (team × month) skeleton first with all metrics zero. Once the skeleton works, subsequent tasks add one metric at a time. This is intentionally TDD-friendly: each metric gets its own red→green cycle.

**Files:**
- Modify: `repository/posgres.go`

- [ ] **Step 1: Write a failing integration test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsSkeleton(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `cd /home/developer/code/github/jiraworklog && go test ./test/ -run TestMonthlyTeamMetricsSkeleton -v`
Expected: FAIL (compile error — method doesn't exist yet).

- [ ] **Step 3: Implement the skeleton in `repository/posgres.go`**

Add this method near the existing `ProjectKPIs` method:

```go
// MonthlyTeamMetrics — see repository/repo.go for contract.
func (s *Postgres) MonthlyTeamMetrics(teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error) {
	result := []types.MonthlyTeamMetrics{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	// Skeleton query: cross-join teams × months, all metrics zero.
	// Subsequent tasks add LEFT JOIN sub-queries to populate each metric.
	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		team_list AS (
			SELECT unnest($1::text[]) AS team
		)
		SELECT
			tl.team,
			m.year_month,
			0::float8 AS median_cycle_time_secs,
			0         AS closed_issue_count,
			0::float8 AS flow_efficiency_pct,
			0::float8 AS failed_qa_ratio_pct,
			0::float8 AS defect_escape_rate_pct,
			0         AS stability_new_count,
			0         AS stability_closed_count,
			0         AS stability_open_count
		FROM team_list tl
		CROSS JOIN months m
		ORDER BY tl.team, m.year_month`

	err := s.DB.Select(&result, query, teams, from, to)
	return result, err
}
```

- [ ] **Step 4: Verify the build succeeds**

Run: `cd /home/developer/code/github/jiraworklog && go build ./...`
Expected: build succeeds.

- [ ] **Step 5: Run the test to confirm it passes**

Run: `go test ./test/ -run TestMonthlyTeamMetricsSkeleton -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/repo.go repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add MonthlyTeamMetrics skeleton (teams × months grid)"
```

---

## Task 5: Add Throughput (ClosedIssueCount) to MonthlyTeamMetrics

Throughput = COUNT(DISTINCT issueid) where the issue entered a `doneStatuses` state in the month, filtered to `closeableTypes`.

**Files:**
- Modify: `repository/posgres.go` (the `MonthlyTeamMetrics` query)
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsThroughput(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./test/ -run TestMonthlyTeamMetricsThroughput -v`
Expected: FAIL — total is 0 because the skeleton returns zeros.

- [ ] **Step 3: Add the throughput sub-query**

In `repository/posgres.go`, modify the `MonthlyTeamMetrics` query. Add a new CTE after `team_list` and a LEFT JOIN against the result. Replace the entire query body:

```go
	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		team_list AS (
			SELECT unnest($1::text[]) AS team
		),
		issue_close_month AS (
			-- First time each closeable-type issue entered a done status.
			SELECT
				ss.issueid,
				i.project,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN `+doneStatuses+`
			AND i.type IN `+closeableTypes+`
			AND i.project = ANY($1::text[])
			GROUP BY ss.issueid, i.project
		),
		throughput AS (
			SELECT project AS team, year_month, COUNT(DISTINCT issueid) AS closed_issue_count
			FROM issue_close_month
			GROUP BY project, year_month
		)
		SELECT
			tl.team,
			m.year_month,
			0::float8                            AS median_cycle_time_secs,
			COALESCE(th.closed_issue_count, 0)   AS closed_issue_count,
			0::float8                            AS flow_efficiency_pct,
			0::float8                            AS failed_qa_ratio_pct,
			0::float8                            AS defect_escape_rate_pct,
			0                                    AS stability_new_count,
			0                                    AS stability_closed_count,
			0                                    AS stability_open_count
		FROM team_list tl
		CROSS JOIN months m
		LEFT JOIN throughput th
		  ON th.team = tl.team AND th.year_month = m.year_month
		ORDER BY tl.team, m.year_month`
```

- [ ] **Step 4: Run the throughput test to confirm it passes**

Run: `go test ./test/ -run TestMonthlyTeamMetricsThroughput -v`
Expected: PASS.

- [ ] **Step 5: Re-run the skeleton test to confirm we didn't regress row counts**

Run: `go test ./test/ -run TestMonthlyTeamMetricsSkeleton -v`
Expected: PASS (still 39 rows).

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add throughput to MonthlyTeamMetrics"
```

---

## Task 6: Add Median Cycle Time to MonthlyTeamMetrics

Median Cycle Time = `percentile_cont(0.5)` over per-issue SUM(Dev+QA stint seconds), grouped by (project, close month).

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsCycleTime(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run the test to confirm it fails**

Run: `go test ./test/ -run TestMonthlyTeamMetricsCycleTime -v`
Expected: FAIL.

- [ ] **Step 3: Add the cycle-time sub-query**

In `repository/posgres.go`, add a `cycle_time` CTE before the final SELECT, and replace the placeholder `0::float8 AS median_cycle_time_secs`:

Add to the CTE chain (after `throughput`):

```sql
,
		issue_cycle_secs AS (
			-- Per-issue total active (Dev+QA) seconds, joined to its close month.
			SELECT
				icm.project   AS team,
				icm.year_month,
				ss.issueid,
				SUM(ss.durationseconds)::float8 AS total_secs
			FROM status_stints ss
			JOIN issue_close_month icm ON icm.issueid = ss.issueid
			WHERE ss.status IN `+devQAStatuses+`
			GROUP BY icm.project, icm.year_month, ss.issueid
		),
		cycle_time AS (
			SELECT
				team,
				year_month,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY total_secs) AS median_cycle_time_secs
			FROM issue_cycle_secs
			GROUP BY team, year_month
		)
```

Then change the placeholder in the final SELECT from:

```sql
0::float8                            AS median_cycle_time_secs,
```

to:

```sql
COALESCE(ct.median_cycle_time_secs, 0)::float8 AS median_cycle_time_secs,
```

And add the LEFT JOIN to the FROM clause (after the throughput join):

```sql
LEFT JOIN cycle_time ct
  ON ct.team = tl.team AND ct.year_month = m.year_month
```

- [ ] **Step 4: Run the cycle-time test to confirm it passes**

Run: `go test ./test/ -run TestMonthlyTeamMetricsCycleTime -v`
Expected: PASS.

- [ ] **Step 5: Re-run prior tests for no regression**

Run: `go test ./test/ -run TestMonthlyTeamMetrics -v`
Expected: all three TestMonthlyTeamMetrics* tests PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add median cycle time to MonthlyTeamMetrics"
```

---

## Task 7: Add Flow Efficiency to MonthlyTeamMetrics

Flow Efficiency = `SUM(Dev+QA seconds) / SUM(Dev+QA+Waiting+Done seconds)` across closed issues in the month, expressed as a percentage.

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsFlowEfficiency(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run it to confirm failure**

Run: `go test ./test/ -run TestMonthlyTeamMetricsFlowEfficiency -v`
Expected: FAIL.

- [ ] **Step 3: Add the flow-efficiency CTE and JOIN**

Add after the `cycle_time` CTE:

```sql
,
		flow_efficiency AS (
			SELECT
				icm.project   AS team,
				icm.year_month,
				CASE
					WHEN SUM(CASE WHEN ss.status IN `+devQAStatuses+` OR ss.status IN ('On Hold','QA Backlog') OR ss.status IN `+doneStatuses+` THEN ss.durationseconds ELSE 0 END) > 0
					THEN
						100.0 * SUM(CASE WHEN ss.status IN `+devQAStatuses+` THEN ss.durationseconds ELSE 0 END)::float8
						/ SUM(CASE WHEN ss.status IN `+devQAStatuses+` OR ss.status IN ('On Hold','QA Backlog') OR ss.status IN `+doneStatuses+` THEN ss.durationseconds ELSE 0 END)::float8
					ELSE 0
				END AS flow_efficiency_pct
			FROM status_stints ss
			JOIN issue_close_month icm ON icm.issueid = ss.issueid
			GROUP BY icm.project, icm.year_month
		)
```

Change the placeholder in the final SELECT from:

```sql
0::float8                            AS flow_efficiency_pct,
```

to:

```sql
COALESCE(fe.flow_efficiency_pct, 0)::float8 AS flow_efficiency_pct,
```

And add to the FROM clause:

```sql
LEFT JOIN flow_efficiency fe
  ON fe.team = tl.team AND fe.year_month = m.year_month
```

- [ ] **Step 4: Run the flow-efficiency test**

Run: `go test ./test/ -run TestMonthlyTeamMetricsFlowEfficiency -v`
Expected: PASS.

- [ ] **Step 5: Re-run all MonthlyTeamMetrics tests**

Run: `go test ./test/ -run TestMonthlyTeamMetrics -v`
Expected: 4 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add flow efficiency to MonthlyTeamMetrics"
```

---

## Task 8: Add Failed QA Ratio to MonthlyTeamMetrics

Failed QA Ratio = `COUNT(stories that hit 'Failed QA' status at least once) / COUNT(closed stories)`, expressed as percentage. Stories only (`issue.type = 'Story'`).

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsFailedQARatio(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)

	for _, r := range rows {
		require.GreaterOrEqual(t, r.FailedQARatioPct, 0.0)
		require.LessOrEqual(t, r.FailedQARatioPct, 100.0)
	}
}
```

The looser assertion is intentional — Failed QA Ratio can legitimately be 0 in a month if no stories failed QA, so we just bound-check.

- [ ] **Step 2: Run to confirm it (currently passes because all zero)**

Run: `go test ./test/ -run TestMonthlyTeamMetricsFailedQARatio -v`
Expected: PASS, but the metric is uninformative until implemented. To get a real "fail then green" cycle, change the assertion to require a *non-zero* value somewhere on the grid before implementing, then revert after green:

Temporarily add at the end of the test:

```go
foundNonZero := false
for _, r := range rows {
	if r.FailedQARatioPct > 0 {
		foundNonZero = true
		break
	}
}
require.True(t, foundNonZero, "expected at least one month with a failed QA story")
```

Run again: expected FAIL.

- [ ] **Step 3: Implement Failed QA Ratio**

Add a CTE after `flow_efficiency`:

```sql
,
		closed_stories AS (
			SELECT
				icm.project   AS team,
				icm.year_month,
				icm.issueid
			FROM issue_close_month icm
			JOIN issue i ON i.id = icm.issueid
			WHERE i.type = 'Story'
		),
		failed_qa_stories AS (
			SELECT DISTINCT cs.team, cs.year_month, cs.issueid
			FROM closed_stories cs
			JOIN issue_transition it ON it.issueid = cs.issueid
			WHERE it.tostatus = 'Failed QA'
		),
		failed_qa_ratio AS (
			SELECT
				cs.team,
				cs.year_month,
				CASE
					WHEN COUNT(DISTINCT cs.issueid) > 0
					THEN 100.0 * COUNT(DISTINCT fqs.issueid)::float8 / COUNT(DISTINCT cs.issueid)::float8
					ELSE 0
				END AS failed_qa_ratio_pct
			FROM closed_stories cs
			LEFT JOIN failed_qa_stories fqs
			  ON fqs.team = cs.team AND fqs.year_month = cs.year_month AND fqs.issueid = cs.issueid
			GROUP BY cs.team, cs.year_month
		)
```

Replace placeholder:

```sql
0::float8                            AS failed_qa_ratio_pct,
```

with:

```sql
COALESCE(fqr.failed_qa_ratio_pct, 0)::float8 AS failed_qa_ratio_pct,
```

Add JOIN:

```sql
LEFT JOIN failed_qa_ratio fqr
  ON fqr.team = tl.team AND fqr.year_month = m.year_month
```

- [ ] **Step 4: Run test, confirm it passes**

Run: `go test ./test/ -run TestMonthlyTeamMetricsFailedQARatio -v`
Expected: PASS.

- [ ] **Step 5: Remove the temporary "foundNonZero" assertion (it served its red→green purpose)**

Revert the test back to only the bounds checks (the version from Step 1, before the temporary block). Re-run — still PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add Failed QA Ratio to MonthlyTeamMetrics"
```

---

## Task 9: Add Defect Escape Rate to MonthlyTeamMetrics

Defect Escape Rate = `COUNT(customer bugs) / COUNT(all bugs)` per (project, createdate month), expressed as percentage.

- Customer bugs = `Customer Bug` + `HW / FW Customer Bug`
- Internal bugs = `Bug` + `Hardware Bug`

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsDefectEscapeRate(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run to confirm failure**

Run: `go test ./test/ -run TestMonthlyTeamMetricsDefectEscapeRate -v`
Expected: FAIL on `foundNonZero`.

- [ ] **Step 3: Add the defect-escape-rate CTE**

Add after the `failed_qa_ratio` CTE:

```sql
,
		defect_escape AS (
			SELECT
				project AS team,
				to_char(createdate, 'YYYY-MM') AS year_month,
				CASE
					WHEN COUNT(*) FILTER (WHERE type IN ('Customer Bug','HW / FW Customer Bug','Bug','Hardware Bug')) > 0
					THEN 100.0 *
						COUNT(*) FILTER (WHERE type IN ('Customer Bug','HW / FW Customer Bug'))::float8
						/ COUNT(*) FILTER (WHERE type IN ('Customer Bug','HW / FW Customer Bug','Bug','Hardware Bug'))::float8
					ELSE 0
				END AS defect_escape_rate_pct
			FROM issue
			WHERE project = ANY($1::text[])
			AND createdate >= $2::timestamptz
			AND createdate <  $3::timestamptz
			AND type IN ('Customer Bug','HW / FW Customer Bug','Bug','Hardware Bug')
			GROUP BY project, to_char(createdate, 'YYYY-MM')
		)
```

Replace placeholder:

```sql
0::float8                            AS defect_escape_rate_pct,
```

with:

```sql
COALESCE(de.defect_escape_rate_pct, 0)::float8 AS defect_escape_rate_pct,
```

Add JOIN:

```sql
LEFT JOIN defect_escape de
  ON de.team = tl.team AND de.year_month = m.year_month
```

- [ ] **Step 4: Run test, confirm pass**

Run: `go test ./test/ -run TestMonthlyTeamMetricsDefectEscapeRate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add defect escape rate to MonthlyTeamMetrics"
```

---

## Task 10: Add Stability counts (new / closed / open) to MonthlyTeamMetrics

These mirror the logic already in `CustomerBugTrends()` (`posgres.go:144`) but the existing function returns one shape per team — here we inline the same logic into the multi-team grid query so the leadership page renders from a single payload.

**Note on spec deviation:** The spec called for reusing `CustomerBugTrends()` once per team. This task inlines the same logic into the multi-team grid query instead — semantically identical, one DB round-trip instead of three, and gives the template a single ordered payload. `CustomerBugTrends()` itself is left untouched (still used by the existing `/reports/customer-bugs` page).

**Files:**
- Modify: `repository/posgres.go`
- Modify: `test/repo_test.go`

- [ ] **Step 1: Write a failing test**

Add to `test/repo_test.go`:

```go
func TestMonthlyTeamMetricsStability(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
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
```

- [ ] **Step 2: Run, confirm FAIL**

Run: `go test ./test/ -run TestMonthlyTeamMetricsStability -v`
Expected: FAIL.

- [ ] **Step 3: Add the stability CTEs**

Add after `defect_escape`:

```sql
,
		stability_new AS (
			SELECT
				project AS team,
				to_char(createdate, 'YYYY-MM') AS year_month,
				COUNT(*) AS new_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND createdate >= $2::timestamptz
			AND createdate <  $3::timestamptz
			GROUP BY project, to_char(createdate, 'YYYY-MM')
		),
		stability_closed AS (
			SELECT
				project AS team,
				to_char(COALESCE(resolveddate, updatedate), 'YYYY-MM') AS year_month,
				COUNT(*) AS closed_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND (
				(resolveddate >= $2::timestamptz AND resolveddate < $3::timestamptz)
				OR (status LIKE 'Awaiting Release%' AND updatedate >= $2::timestamptz AND updatedate < $3::timestamptz)
			)
			GROUP BY project, to_char(COALESCE(resolveddate, updatedate), 'YYYY-MM')
		),
		stability_baseline AS (
			-- Open customer bugs at the start of the window, per team.
			SELECT
				project AS team,
				COUNT(*) AS open_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND createdate < $2::timestamptz
			AND (resolveddate IS NULL OR resolveddate >= $2::timestamptz)
			GROUP BY project
		),
		stability AS (
			SELECT
				tl.team,
				m.year_month,
				COALESCE(sn.new_count, 0)    AS stability_new_count,
				COALESCE(sc.closed_count, 0) AS stability_closed_count,
				SUM(COALESCE(sn.new_count, 0) - COALESCE(sc.closed_count, 0)) OVER (
					PARTITION BY tl.team
					ORDER BY m.year_month
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) + COALESCE(sb.open_count, 0) AS stability_open_count
			FROM team_list tl
			CROSS JOIN months m
			LEFT JOIN stability_new    sn ON sn.team = tl.team AND sn.year_month = m.year_month
			LEFT JOIN stability_closed sc ON sc.team = tl.team AND sc.year_month = m.year_month
			LEFT JOIN stability_baseline sb ON sb.team = tl.team
		)
```

Replace the three placeholders:

```sql
0                                    AS stability_new_count,
0                                    AS stability_closed_count,
0                                    AS stability_open_count
```

with:

```sql
COALESCE(st.stability_new_count, 0)    AS stability_new_count,
COALESCE(st.stability_closed_count, 0) AS stability_closed_count,
COALESCE(st.stability_open_count, 0)   AS stability_open_count
```

Add LEFT JOIN:

```sql
LEFT JOIN stability st
  ON st.team = tl.team AND st.year_month = m.year_month
```

- [ ] **Step 4: Run test, confirm pass**

Run: `go test ./test/ -run TestMonthlyTeamMetricsStability -v`
Expected: PASS.

- [ ] **Step 5: Run ALL MonthlyTeamMetrics tests to confirm no regression**

Run: `go test ./test/ -run TestMonthlyTeamMetrics -v`
Expected: 6 tests PASS.

- [ ] **Step 6: Commit**

```bash
git add repository/posgres.go test/repo_test.go
git commit -m "feat(repo): add stability new/closed/open counts to MonthlyTeamMetrics"
```

---

## Task 11: Create leadership.templ — page skeleton with month selector + footer

Build the page incrementally. This task: outer shell, month-selector form, footer caveat, no data yet.

**Files:**
- Create: `templates/pages/leadership.templ`

- [ ] **Step 1: Create the template**

```go
package pages

import (
	"fmt"
	"time"

	"github.com/mkobaly/jiraworklog/templates/layouts"
	"github.com/mkobaly/jiraworklog/types"
)

// LeadershipDashboard renders the all-teams monthly KPI comparison grid.
// `rows` is expected to contain (team × year_month) rows for a 13-month
// window, ordered by (team, year_month). `selectedMonth` is the YYYY-MM the
// user picked in the dropdown; empty string means "default to last complete
// month."
templ LeadershipDashboard(rows []types.MonthlyTeamMetrics, teams []string, selectedMonth string, monthOptions []string) {
	@layouts.Base("Leadership Dashboard") {
		<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>

		<div class="space-y-6">
			<!-- Header -->
			<div class="bg-white rounded-lg shadow p-6 flex items-center justify-between flex-wrap gap-3">
				<h1 class="text-2xl font-bold text-gray-800">Leadership Dashboard</h1>
				<form method="GET" action="/dashboard/leadership" class="flex items-center gap-2">
					<label for="month" class="text-sm font-medium text-gray-700">Month</label>
					<select id="month" name="month" onchange="this.form.submit()" class="border border-gray-300 rounded-md px-3 py-1.5 text-sm">
						for _, opt := range monthOptions {
							<option value={ opt } if opt == selectedMonth { selected }>{ formatMonthOption(opt) }</option>
						}
					</select>
				</form>
			</div>

			if len(rows) == 0 {
				<div class="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center text-yellow-700">
					No monthly data yet. Sync must have run at least once.
				</div>
			} else {
				<!-- KPI grid placeholder (Task 12 replaces this) -->
				<div class="bg-white rounded-lg shadow p-6">
					<p class="text-sm text-gray-500">{ fmt.Sprintf("Loaded %d rows for %d teams", len(rows), len(teams)) }</p>
				</div>

				<!-- Stability strip placeholder (Task 15 replaces this) -->
				<div class="bg-white rounded-lg shadow p-6">
					<h2 class="text-lg font-semibold text-gray-700 mb-3">Stability (Customer Bugs)</h2>
					<p class="text-sm text-gray-500">Per-team stability charts go here.</p>
				</div>
			}

			<!-- Footer caveat -->
			<p class="text-xs text-gray-400 italic text-center">
				Each project includes both project-team and stability-team work; v2 will split these.
			</p>
		</div>
	}
}

// formatMonthOption turns "2026-04" into "April 2026" for the dropdown.
func formatMonthOption(yyyymm string) string {
	t, err := time.Parse("2006-01", yyyymm)
	if err != nil {
		return yyyymm
	}
	return t.Format("January 2006")
}
```

- [ ] **Step 2: Generate templ Go code**

Run: `cd /home/developer/code/github/jiraworklog && templ generate`
Expected: `templates/pages/leadership_templ.go` is created with no errors.

- [ ] **Step 3: Verify the package builds**

Run: `go build ./...`
Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add templates/pages/leadership.templ templates/pages/leadership_templ.go
git commit -m "feat(templates): add leadership dashboard page skeleton"
```

---

## Task 12: Add KPI grid to leadership.templ

Replace the placeholder with the actual 5-row × N-team grid. Each cell shows the current month's value + a direction arrow vs. the prior month. Sparklines are added in Task 13.

**Files:**
- Modify: `templates/pages/leadership.templ`

- [ ] **Step 1: Add helper functions and the grid**

Replace the "KPI grid placeholder" block with a full grid. Insert the helper functions at the bottom of `leadership.templ` (after `formatMonthOption`):

```go
// pickMonthRows returns the rows whose YearMonth equals selectedMonth, in the
// same team order as `teams`.
func pickMonthRows(rows []types.MonthlyTeamMetrics, teams []string, selectedMonth string) []types.MonthlyTeamMetrics {
	byTeam := map[string]types.MonthlyTeamMetrics{}
	for _, r := range rows {
		if r.YearMonth == selectedMonth {
			byTeam[r.Team] = r
		}
	}
	out := make([]types.MonthlyTeamMetrics, 0, len(teams))
	for _, t := range teams {
		out = append(out, byTeam[t])
	}
	return out
}

// priorMonthRows returns the rows for the month immediately preceding
// selectedMonth, ordered by `teams`. Used for direction-arrow comparison.
func priorMonthRows(rows []types.MonthlyTeamMetrics, teams []string, selectedMonth string) []types.MonthlyTeamMetrics {
	prior := priorYearMonth(selectedMonth)
	byTeam := map[string]types.MonthlyTeamMetrics{}
	for _, r := range rows {
		if r.YearMonth == prior {
			byTeam[r.Team] = r
		}
	}
	out := make([]types.MonthlyTeamMetrics, 0, len(teams))
	for _, t := range teams {
		out = append(out, byTeam[t])
	}
	return out
}

func priorYearMonth(yyyymm string) string {
	t, err := time.Parse("2006-01", yyyymm)
	if err != nil {
		return yyyymm
	}
	return t.AddDate(0, -1, 0).Format("2006-01")
}

// directionArrow returns ("▲"|"▼"|"▬", tailwind-class) comparing current vs
// prior, where `lowerIsBetter` indicates whether a decrease is good (e.g.
// cycle time). Threshold of 5% relative change is treated as flat.
func directionArrow(current, prior float64, lowerIsBetter bool) (string, string) {
	if prior == 0 && current == 0 {
		return "▬", "text-gray-400"
	}
	var rel float64
	if prior == 0 {
		rel = 1.0
	} else {
		rel = (current - prior) / prior
	}
	if rel > -0.05 && rel < 0.05 {
		return "▬", "text-gray-400"
	}
	improving := (rel < 0 && lowerIsBetter) || (rel > 0 && !lowerIsBetter)
	if rel > 0 {
		if improving {
			return "▲", "text-green-600"
		}
		return "▲", "text-red-600"
	}
	if improving {
		return "▼", "text-green-600"
	}
	return "▼", "text-red-600"
}

func fmtDays(secs float64) string {
	if secs <= 0 {
		return "—"
	}
	days := secs / 86400
	return fmt.Sprintf("%.1fd", days)
}

func fmtPct(pct float64) string {
	return fmt.Sprintf("%.0f%%", pct)
}

func fmtCount(n int) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

// sparsenessNote returns a "* n=3" caveat when the closed-issue count is sparse.
func sparsenessNote(r types.MonthlyTeamMetrics) string {
	if r.IsSparseSample() {
		return fmt.Sprintf("* n=%d", r.ClosedIssueCount)
	}
	return ""
}
```

Now replace the placeholder block in the main template:

```go
<!-- KPI grid placeholder (Task 12 replaces this) -->
<div class="bg-white rounded-lg shadow p-6">
	<p class="text-sm text-gray-500">{ fmt.Sprintf("Loaded %d rows for %d teams", len(rows), len(teams)) }</p>
</div>
```

with:

```go
<div class="bg-white rounded-lg shadow p-6 overflow-x-auto">
	<table class="min-w-full text-sm">
		<thead>
			<tr class="text-left text-gray-500 uppercase tracking-wide text-xs">
				<th class="py-2 pr-4">Metric</th>
				for _, t := range teams {
					<th class="py-2 px-4 text-center">
						<a href={ templ.SafeURL("/dashboard/manager?team=" + t) } class="text-blue-600 hover:underline">{ t }</a>
					</th>
				}
			</tr>
		</thead>
		<tbody class="divide-y divide-gray-100">
			@kpiRow("Cycle Time (median)", pickMonthRows(rows, teams, selectedMonth), priorMonthRows(rows, teams, selectedMonth), "cycle", true)
			@kpiRow("Throughput", pickMonthRows(rows, teams, selectedMonth), priorMonthRows(rows, teams, selectedMonth), "throughput", false)
			@kpiRow("Flow Efficiency", pickMonthRows(rows, teams, selectedMonth), priorMonthRows(rows, teams, selectedMonth), "flow", false)
			@kpiRow("Failed QA Ratio", pickMonthRows(rows, teams, selectedMonth), priorMonthRows(rows, teams, selectedMonth), "failedqa", true)
			@kpiRow("Defect Escape Rate", pickMonthRows(rows, teams, selectedMonth), priorMonthRows(rows, teams, selectedMonth), "escape", true)
		</tbody>
	</table>
</div>
```

Add the `kpiRow` component above `LeadershipDashboard`:

```go
// kpiRow renders one metric row across all teams. metricKey controls
// formatting (e.g. "cycle" → days, others → %/count). lowerIsBetter is
// passed to directionArrow.
templ kpiRow(label string, current []types.MonthlyTeamMetrics, prior []types.MonthlyTeamMetrics, metricKey string, lowerIsBetter bool) {
	<tr>
		<td class="py-3 pr-4 font-medium text-gray-700">{ label }</td>
		for i, cur := range current {
			<td class="py-3 px-4 text-center align-top">
				if !cur.HasClosedIssues() && (metricKey == "cycle" || metricKey == "throughput" || metricKey == "flow" || metricKey == "failedqa") {
					<span class="text-gray-300">—</span>
				} else {
					<div class="flex items-center justify-center gap-1.5">
						<span class="text-lg font-semibold text-gray-800">
							switch metricKey {
								case "cycle":
									{ fmtDays(cur.MedianCycleTimeSecs) }
								case "throughput":
									{ fmtCount(cur.ClosedIssueCount) }
								case "flow":
									{ fmtPct(cur.FlowEfficiencyPct) }
								case "failedqa":
									{ fmtPct(cur.FailedQARatioPct) }
								case "escape":
									{ fmtPct(cur.DefectEscapeRatePct) }
							}
						</span>
						@arrowCell(cur, priorOrEmpty(prior, i), metricKey, lowerIsBetter)
					</div>
					if metricKey == "cycle" {
						<div class="text-[10px] text-gray-400">{ sparsenessNote(cur) }</div>
					}
				}
			</td>
		}
	</tr>
}

templ arrowCell(cur types.MonthlyTeamMetrics, prior types.MonthlyTeamMetrics, metricKey string, lowerIsBetter bool) {
	{{
		var c, p float64
		switch metricKey {
		case "cycle":
			c, p = cur.MedianCycleTimeSecs, prior.MedianCycleTimeSecs
		case "throughput":
			c, p = float64(cur.ClosedIssueCount), float64(prior.ClosedIssueCount)
		case "flow":
			c, p = cur.FlowEfficiencyPct, prior.FlowEfficiencyPct
		case "failedqa":
			c, p = cur.FailedQARatioPct, prior.FailedQARatioPct
		case "escape":
			c, p = cur.DefectEscapeRatePct, prior.DefectEscapeRatePct
		}
		arrow, color := directionArrow(c, p, lowerIsBetter)
	}}
	<span class={ "text-sm font-bold", color }>{ arrow }</span>
}

func priorOrEmpty(prior []types.MonthlyTeamMetrics, i int) types.MonthlyTeamMetrics {
	if i < len(prior) {
		return prior[i]
	}
	return types.MonthlyTeamMetrics{}
}
```

- [ ] **Step 2: Generate templ**

Run: `templ generate`
Expected: regenerates `_templ.go` files with no errors.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add templates/pages/leadership.templ templates/pages/leadership_templ.go
git commit -m "feat(templates): add KPI grid + direction arrows to leadership page"
```

---

## Task 13: Add 12-month sparklines under each KPI cell

Each cell gets a tiny Chart.js sparkline showing the metric's trailing 12 months. To keep JS light, all sparkline data is rendered into a hidden JSON `<script>` block and a single bootstrap JS block iterates and renders.

**Files:**
- Modify: `templates/pages/leadership.templ`

- [ ] **Step 1: Add a JSON serializer helper**

At the bottom of `leadership.templ`, add:

```go
// sparklineSeries pivots all 13-month rows into per-(team,metric) numeric
// series for client-side rendering. Returns a JSON blob.
func sparklineSeries(rows []types.MonthlyTeamMetrics, teams []string) string {
	// outer keys: metric, inner: team → []float64 of 12 trailing values (oldest first)
	metrics := []string{"cycle", "throughput", "flow", "failedqa", "escape"}
	out := map[string]map[string][]float64{}
	for _, m := range metrics {
		out[m] = map[string][]float64{}
		for _, t := range teams {
			out[m][t] = []float64{}
		}
	}
	// rows are ordered (team, year_month asc). Walk and append.
	for _, r := range rows {
		out["cycle"][r.Team]      = append(out["cycle"][r.Team], r.MedianCycleTimeSecs/86400.0)
		out["throughput"][r.Team] = append(out["throughput"][r.Team], float64(r.ClosedIssueCount))
		out["flow"][r.Team]       = append(out["flow"][r.Team], r.FlowEfficiencyPct)
		out["failedqa"][r.Team]   = append(out["failedqa"][r.Team], r.FailedQARatioPct)
		out["escape"][r.Team]     = append(out["escape"][r.Team], r.DefectEscapeRatePct)
	}
	b, _ := json.Marshal(out)
	return string(b)
}
```

And add `"encoding/json"` to the import block at the top of the file.

- [ ] **Step 2: Add a sparkline `<canvas>` to each KPI cell**

In `kpiRow`, after the `arrowCell` and the sparsenessNote (still inside the `else` branch), insert:

```go
<canvas
	data-spark-team={ teamFor(current, i) }
	data-spark-metric={ metricKey }
	class="mt-1 inline-block"
	width="80"
	height="20"
></canvas>
```

Add the helper at the bottom of `leadership.templ`:

```go
func teamFor(current []types.MonthlyTeamMetrics, i int) string {
	if i < len(current) {
		return current[i].Team
	}
	return ""
}
```

- [ ] **Step 3: Add the bootstrap `<script>` block at the bottom of the page (above the footer)**

Just above `<!-- Footer caveat -->`:

```go
<div id="sparklineData" data-series={ sparklineSeries(rows, teams) } style="display:none;"></div>
<script>
	(function() {
		const blob = JSON.parse(document.getElementById('sparklineData').dataset.series || '{}');
		document.querySelectorAll('canvas[data-spark-team]').forEach(function(canvas) {
			const team = canvas.dataset.sparkTeam;
			const metric = canvas.dataset.sparkMetric;
			const series = (blob[metric] || {})[team] || [];
			new Chart(canvas, {
				type: 'line',
				data: {
					labels: series.map((_, i) => i),
					datasets: [{
						data: series,
						borderColor: '#3b82f6',
						borderWidth: 1.5,
						pointRadius: 0,
						tension: 0.35,
						fill: false,
					}],
				},
				options: {
					responsive: false,
					animation: false,
					plugins: { legend: { display: false }, tooltip: { enabled: false } },
					scales: { x: { display: false }, y: { display: false } },
				},
			});
		});
	})();
</script>
```

- [ ] **Step 4: Generate templ**

Run: `templ generate`

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add templates/pages/leadership.templ templates/pages/leadership_templ.go
git commit -m "feat(templates): add 12-month sparklines under each KPI cell"
```

---

## Task 14: Add per-team Stability strip (customer-bug trend charts)

The leadership page renders one customer-bug trend chart per team using the `StabilityNewCount` / `StabilityClosedCount` / `StabilityOpenCount` fields already in the payload.

**Files:**
- Modify: `templates/pages/leadership.templ`

- [ ] **Step 1: Add a serializer for stability series**

At the bottom of `leadership.templ`:

```go
// stabilitySeries returns a JSON blob keyed by team → {labels, new, closed, open}
// for client-side stability chart rendering.
func stabilitySeries(rows []types.MonthlyTeamMetrics, teams []string) string {
	type teamSeries struct {
		Labels []string `json:"labels"`
		New    []int    `json:"new"`
		Closed []int    `json:"closed"`
		Open   []int    `json:"open"`
	}
	out := map[string]*teamSeries{}
	for _, t := range teams {
		out[t] = &teamSeries{}
	}
	for _, r := range rows {
		s := out[r.Team]
		if s == nil {
			continue
		}
		s.Labels = append(s.Labels, r.YearMonth)
		s.New    = append(s.New, r.StabilityNewCount)
		s.Closed = append(s.Closed, r.StabilityClosedCount)
		s.Open   = append(s.Open, r.StabilityOpenCount)
	}
	b, _ := json.Marshal(out)
	return string(b)
}
```

- [ ] **Step 2: Replace the stability placeholder**

Find:

```go
<!-- Stability strip placeholder (Task 15 replaces this) -->
<div class="bg-white rounded-lg shadow p-6">
	<h2 class="text-lg font-semibold text-gray-700 mb-3">Stability (Customer Bugs)</h2>
	<p class="text-sm text-gray-500">Per-team stability charts go here.</p>
</div>
```

Replace with:

```go
<div class="bg-white rounded-lg shadow p-6">
	<h2 class="text-lg font-semibold text-gray-700 mb-1">Stability (Customer Bugs)</h2>
	<p class="text-xs text-gray-400 mb-4">New bugs opened, bugs closed, total open bugs per month.</p>
	<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
		for _, t := range teams {
			<div>
				<p class="text-sm font-medium text-gray-700 text-center mb-2">{ t }</p>
				<div class="h-48">
					<canvas data-stability-team={ t }></canvas>
				</div>
			</div>
		}
	</div>
	<div id="stabilityData" data-series={ stabilitySeries(rows, teams) } style="display:none;"></div>
</div>
```

- [ ] **Step 3: Add the stability bootstrap script next to the sparkline script**

Inside the existing `<script>` block (or in a sibling block), add:

```javascript
(function() {
	const blob = JSON.parse(document.getElementById('stabilityData').dataset.series || '{}');
	document.querySelectorAll('canvas[data-stability-team]').forEach(function(canvas) {
		const team = canvas.dataset.stabilityTeam;
		const s = blob[team] || { labels: [], new: [], closed: [], open: [] };
		new Chart(canvas, {
			type: 'line',
			data: {
				labels: s.labels,
				datasets: [
					{ label: 'New',    data: s.new,    borderColor: '#ef4444', backgroundColor: 'rgba(239,68,68,0.1)', tension: 0.3 },
					{ label: 'Closed', data: s.closed, borderColor: '#10b981', backgroundColor: 'rgba(16,185,129,0.1)', tension: 0.3 },
					{ label: 'Open',   data: s.open,   borderColor: '#f59e0b', backgroundColor: 'rgba(245,158,11,0.1)', tension: 0.3 },
				],
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: { legend: { position: 'bottom', labels: { font: { size: 10 } } } },
				scales: {
					x: { ticks: { font: { size: 9 }, maxRotation: 45, minRotation: 45 } },
					y: { beginAtZero: true, ticks: { font: { size: 10 } } },
				},
			},
		});
	});
})();
```

- [ ] **Step 4: Generate templ**

Run: `templ generate`

- [ ] **Step 5: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add templates/pages/leadership.templ templates/pages/leadership_templ.go
git commit -m "feat(templates): add per-team stability strip to leadership page"
```

---

## Task 15: Add GetLeadershipDashboard handler

The handler picks teams (hard-coded IDM/SYM/ESG for v1), calls the repo, computes the month-options dropdown, and renders the page.

**Files:**
- Modify: `cmd/jiraworklog/handlers.go`

- [ ] **Step 1: Add the handler method**

Insert near the existing `GetProjectKPIs` method:

```go
// GET /dashboard/leadership
func (h *Handler) GetLeadershipDashboard(c echo.Context) error {
	// v1: hard-coded team list. v2 will pull from a teams table or projectCharge.
	teams := []string{"IDM", "SYM", "ESG"}

	rows, err := h.repo.MonthlyTeamMetrics(teams, "", "")
	if err != nil {
		h.logger.Error("error fetching monthly team metrics", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch monthly team metrics")
	}

	// Build month-options dropdown from the rows we received (sorted unique
	// year_month values, descending so the most recent is on top).
	monthSet := map[string]struct{}{}
	for _, r := range rows {
		monthSet[r.YearMonth] = struct{}{}
	}
	monthOptions := make([]string, 0, len(monthSet))
	for m := range monthSet {
		monthOptions = append(monthOptions, m)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(monthOptions)))

	// Default to last complete month (highest year_month in the result).
	selectedMonth := c.QueryParam("month")
	if selectedMonth == "" && len(monthOptions) > 0 {
		selectedMonth = monthOptions[0]
	}

	if wantsHTML(c) {
		return pages.LeadershipDashboard(rows, teams, selectedMonth, monthOptions).
			Render(c.Request().Context(), c.Response().Writer)
	}
	return c.JSON(http.StatusOK, rows)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/jiraworklog/handlers.go
git commit -m "feat(handler): add GET /dashboard/leadership handler"
```

---

## Task 16: Register the `/dashboard/leadership` route

**Files:**
- Modify: `cmd/jiraworklog/main.go`

- [ ] **Step 1: Add the route inside the existing dashboard block**

Find:

```go
	// Dashboard routes
	e.GET("/", handler.Dashboard, handler.AuthMiddleware)
	e.GET("/dashboard", handler.Dashboard, handler.AuthMiddleware)
```

Append after these lines (still inside the dashboard block):

```go
	e.GET("/dashboard/leadership", handler.GetLeadershipDashboard, handler.AuthMiddleware)
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/jiraworklog/main.go
git commit -m "feat(routes): register /dashboard/leadership"
```

---

## Task 17: Add "Leadership" link to the base layout navigation

**Files:**
- Modify: `templates/layouts/base.templ`

- [ ] **Step 1: Add the link to both desktop and mobile nav blocks**

In the desktop block (around line 98), immediately after the `<a href="/" ...>Dashboard</a>` link, insert:

```go
<a href="/dashboard/leadership" class="px-3 py-2 rounded-md hover:bg-blue-700 transition">Leadership</a>
```

Do the same in the mobile block (around line 117), in the same position immediately after the Dashboard link.

- [ ] **Step 2: Generate templ**

Run: `templ generate`

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add templates/layouts/base.templ templates/layouts/base_templ.go
git commit -m "feat(nav): add Leadership link to base layout navigation"
```

---

## Task 18: Manual smoke test

Bring up the app and confirm the page renders end-to-end with real data.

- [ ] **Step 1: Build everything**

Run: `cd /home/developer/code/github/jiraworklog && ./build.sh`
Expected: build completes; `bin/jiraworklog` exists.

- [ ] **Step 2: Start the server**

Run: `./bin/jiraworklog -v -c bin/config.yaml`
Expected: server starts on the configured port (default 8380).

- [ ] **Step 3: In a browser, navigate to `http://localhost:8380/dashboard/leadership`**

Expected:
- Page renders with a header that says "Leadership Dashboard"
- Month dropdown shows the last 13 months, defaulted to the most recent complete month
- KPI grid shows 5 rows × 3 columns (IDM, SYM, ESG)
- Each non-empty cell shows a value + direction arrow + sparkline
- Empty cells show "—" not "0"
- Stability strip shows 3 line charts (one per team) with new/closed/open lines
- Footer caveat is visible
- "Leadership" link in the nav highlights this page

- [ ] **Step 4: Test the month selector**

Change the month in the dropdown to a different month. Page should reload with that month selected and the KPI grid showing that month's values (sparklines still show the same 13-month trailing series).

- [ ] **Step 5: Test the team-header link**

Click on the IDM, SYM, or ESG column header. Expect a 404 — Phase 2 will add the manager page. Note this as expected behavior for this manual test.

- [ ] **Step 6: Stop the server (Ctrl-C)**

- [ ] **Step 7: Final commit (only if anything changed during smoke testing)**

If you noticed any visual issues and fixed them during the smoke test, commit them now:

```bash
git status
# review any changes
git add <files>
git commit -m "fix: smoke-test corrections to leadership dashboard"
```

If nothing changed: no commit needed.

---

## Done

At this point Phase 1 is complete and shippable. The leadership dashboard renders the 5 monthly KPIs (Cycle Time, Throughput, Flow Efficiency, Failed QA Ratio, Defect Escape Rate) plus a per-team Stability strip, for the IDM/SYM/ESG Jira projects, with month selection and sparklines.

**Next:** ship Phase 1 → collect feedback from one monthly leadership review → write Phase 2 plan for the `/dashboard/manager` deep-dive page (Time-in-Status, WIP, Aging WIP, Rework Cycles, hour ratios, Status Bounce Rate).
