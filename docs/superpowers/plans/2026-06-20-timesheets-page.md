# Timesheets Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `/reports/timesheets` page that shows, per author, hours grouped by raw project charge with one column per month over a selectable From/To range, with subtle red/green tints when a month deviates ±10% from the prior month.

**Architecture:** Mirrors the existing Weekly Hours and Project Charge Hours pages. A new `Postgres.TimesheetHours` repo method runs a grouped SQL aggregation; a new `GetTimesheets` handler parses/clamps the month range and pivots the flat rows into an author-centric view model; a new `templ` page renders a wide table with trend coloring. Pure date/pivot/trend helpers are unit-tested; the repo method has an integration test.

**Tech Stack:** Go, Echo v4, sqlx + pgx/v5 (PostgreSQL), templ, Tailwind CSS, testify.

## Global Constraints

- Go module path: `github.com/mkobaly/jiraworklog`.
- Timezone for bucketing worklog dates into months: `America/New_York` (matches `DailyHoursByRole`).
- Month strings are `YYYY-MM` (Go layout `"2006-01"`). Date boundaries are UTC `time.Time`; upper bound is **exclusive** (first day of the month after To).
- Roles come from `repo.AllRoles()` (distinct non-null `people.role`). When no roles are selected, default to all roles.
- The current (incomplete) month must never be selectable or queryable — the latest allowed month is the previous completed month.
- After modifying any `.templ` file, regenerate Go with `templ generate` (or `./build.sh`). Generated `_templ.go` files must be committed.
- Repo tests are integration tests needing `bin/config.yaml`; run with `go test ./test/...`. Pure-function unit tests run with `go test ./...` and need no DB.
- Build the whole project with `./build.sh` (runs `templ generate` then `go build`). Quick compile check: `go build ./...`.

---

### Task 1: `TimesheetHours` type and repository method

**Files:**
- Modify: `types/parentIssue.go` (add type near `DailyHours` at line ~437)
- Modify: `repository/repo.go` (add method to `Repo` interface near line 56)
- Modify: `repository/posgres.go` (add `TimesheetHours` implementation near `DailyHoursByRole` at line ~330)
- Test: `test/repo_test.go` (append a new test function)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces:
  - `types.TimesheetHours` struct with fields `Role, Author, ProjectCharge, YearMonth string` and `Hours float64`.
  - `Postgres.TimesheetHours(roles []string, from, to time.Time) ([]types.TimesheetHours, error)` — `from` inclusive, `to` exclusive.

- [ ] **Step 1: Write the failing integration test**

Append to `test/repo_test.go`:

```go
func TestTimesheetHours(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	// A recent, completed 3-month window: Jan–Mar 2026. Bounds are built in
	// America/New_York so they align exactly with the query's NY month bucketing
	// (this mirrors how the handler calls the method via monthRangeToTimes).
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./test/ -run TestTimesheetHours -v`
Expected: compile failure — `repo.TimesheetHours undefined` and `types.TimesheetHours undefined`.

- [ ] **Step 3: Add the `TimesheetHours` type**

In `types/parentIssue.go`, immediately after the `DailyHours` block (after its `ProductiveRatio` method, ~line 459), add:

```go
// TimesheetHours represents hours an author logged to a single raw project
// charge in a single year-month. Used by the Timesheets report.
type TimesheetHours struct {
	Role          string  `db:"role"`
	Author        string  `db:"author"`
	ProjectCharge string  `db:"projectcharge"`
	YearMonth     string  `db:"yearmonth"`
	Hours         float64 `db:"hours"`
}
```

- [ ] **Step 4: Add the method to the `Repo` interface**

In `repository/repo.go`, after the `DailyHoursByRole` line (line 56), add:

```go
	// Timesheet hours by author, grouped by raw project charge and year-month.
	// from is inclusive, to is exclusive (first day of the month after the last
	// month to include).
	TimesheetHours(roles []string, from, to time.Time) ([]types.TimesheetHours, error)
```

- [ ] **Step 5: Implement `TimesheetHours` on `Postgres`**

In `repository/posgres.go`, after the `DailyHoursByRole` method (after line 330), add:

```go
// TimesheetHours returns hours per author grouped by raw project charge and
// year-month for the given role set and date range. startDate is inclusive,
// endDate is exclusive.
func (s *Postgres) TimesheetHours(roles []string, startDate, endDate time.Time) ([]types.TimesheetHours, error) {
	result := []types.TimesheetHours{}

	query := `
		SELECT
			p.role,
			w.author,
			i.projectcharge,
			to_char((w.date AT TIME ZONE 'America/New_York'), 'YYYY-MM') AS yearmonth,
			SUM(w.timespenthours) AS hours
		FROM worklog w
		JOIN issue i ON w.issueid = i.id
		JOIN people p ON w.author = p.name
		WHERE w.date >= ($2 AT TIME ZONE 'UTC')
		AND w.date < ($3 AT TIME ZONE 'UTC')
		AND p.role = ANY($1)
		GROUP BY p.role, w.author, i.projectcharge, yearmonth
		ORDER BY w.author, i.projectcharge, yearmonth;`

	err := s.DB.Select(&result, query, roles, startDate, endDate)
	return result, err
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./test/ -run TestTimesheetHours -v`
Expected: PASS (requires `bin/config.yaml` with a reachable DB containing dev worklogs in Q1 2026). If the DB has no dev data for that window, adjust the window in the test to a populated range, but keep the bounds assertions.

- [ ] **Step 7: Commit**

```bash
git add types/parentIssue.go repository/repo.go repository/posgres.go test/repo_test.go
git commit -m "feat(timesheets): add TimesheetHours type and repo method"
```

---

### Task 2: Handler date-range helpers (pure functions)

**Files:**
- Create: `cmd/jiraworklog/timesheets.go`
- Test: `cmd/jiraworklog/timesheets_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces (all in package `main`):
  - `previousMonth(now time.Time) string` → `YYYY-MM` of the month before `now`'s month.
  - `defaultAndClampRange(from, to string, now time.Time) (string, string)` → fills blanks/invalids with `previousMonth(now)`, clamps any month later than `previousMonth(now)` down to it, and ensures `from <= to`.
  - `enumerateMonths(from, to string) []string` → ordered inclusive list of `YYYY-MM` from `from` to `to`.
  - `monthRangeToTimes(from, to string) (time.Time, time.Time)` → start = first day of `from`, end = first day of the month after `to` (exclusive). Both instants are built in `America/New_York` so the UTC-timestamp filter in the SQL aligns exactly with the query's NY month bucketing (no boundary leakage).

- [ ] **Step 1: Write the failing tests**

Create `cmd/jiraworklog/timesheets_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPreviousMonth(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	require.Equal(t, "2026-05", previousMonth(now))

	// January rolls back to previous December.
	jan := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	require.Equal(t, "2025-12", previousMonth(jan))
}

func TestDefaultAndClampRange(t *testing.T) {
	now := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC) // max allowed = 2026-05

	// Blank inputs default to the previous completed month.
	from, to := defaultAndClampRange("", "", now)
	require.Equal(t, "2026-05", from)
	require.Equal(t, "2026-05", to)

	// Valid range passes through untouched.
	from, to = defaultAndClampRange("2026-01", "2026-05", now)
	require.Equal(t, "2026-01", from)
	require.Equal(t, "2026-05", to)

	// Current/future month clamps down to the max allowed month.
	from, to = defaultAndClampRange("2026-06", "2026-09", now)
	require.Equal(t, "2026-05", from)
	require.Equal(t, "2026-05", to)

	// Invalid strings fall back to the max allowed month.
	from, to = defaultAndClampRange("garbage", "", now)
	require.Equal(t, "2026-05", from)
	require.Equal(t, "2026-05", to)

	// from later than to collapses from down to to.
	from, to = defaultAndClampRange("2026-05", "2026-02", now)
	require.Equal(t, "2026-02", from)
	require.Equal(t, "2026-02", to)
}

func TestEnumerateMonths(t *testing.T) {
	require.Equal(t,
		[]string{"2026-01", "2026-02", "2026-03", "2026-04", "2026-05"},
		enumerateMonths("2026-01", "2026-05"))

	// Single month range.
	require.Equal(t, []string{"2026-05"}, enumerateMonths("2026-05", "2026-05"))

	// Range spanning a year boundary.
	require.Equal(t,
		[]string{"2025-11", "2025-12", "2026-01"},
		enumerateMonths("2025-11", "2026-01"))
}

func TestMonthRangeToTimes(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	start, end := monthRangeToTimes("2026-01", "2026-05")
	require.True(t, start.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, loc)))
	// Exclusive upper bound is the first day of the month AFTER May => June 1 (NY).
	require.True(t, end.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, loc)))

	// To in December rolls the exclusive bound into the next January.
	start, end = monthRangeToTimes("2026-12", "2026-12")
	require.True(t, start.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, loc)))
	require.True(t, end.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, loc)))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/jiraworklog/ -run 'TestPreviousMonth|TestDefaultAndClampRange|TestEnumerateMonths|TestMonthRangeToTimes' -v`
Expected: compile failure — the four functions are undefined.

- [ ] **Step 3: Implement the helpers**

Create `cmd/jiraworklog/timesheets.go`:

```go
package main

import "time"

// previousMonth returns the YYYY-MM of the month immediately before now's month.
// This is the latest month the Timesheets report allows, since the current month
// is incomplete.
func previousMonth(now time.Time) string {
	firstOfThis := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prev := firstOfThis.AddDate(0, 0, -1)
	return prev.Format("2006-01")
}

// validMonth reports whether s parses as a YYYY-MM month string.
func validMonth(s string) bool {
	_, err := time.Parse("2006-01", s)
	return err == nil
}

// defaultAndClampRange normalizes the From/To month inputs. Blank or invalid
// values default to the previous completed month. Any month later than the
// previous completed month is clamped down to it (the current/future months are
// not selectable). Finally, if from is after to, from collapses to to.
// YYYY-MM strings compare correctly with string ordering.
func defaultAndClampRange(from, to string, now time.Time) (string, string) {
	max := previousMonth(now)
	if !validMonth(from) {
		from = max
	}
	if !validMonth(to) {
		to = max
	}
	if from > max {
		from = max
	}
	if to > max {
		to = max
	}
	if from > to {
		from = to
	}
	return from, to
}

// enumerateMonths returns the ordered, inclusive list of YYYY-MM months from
// from to to. Assumes from <= to and both are valid (callers pass the output of
// defaultAndClampRange).
func enumerateMonths(from, to string) []string {
	start, err := time.Parse("2006-01", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01", to)
	if err != nil {
		return nil
	}
	var months []string
	for m := start; !m.After(end); m = m.AddDate(0, 1, 0) {
		months = append(months, m.Format("2006-01"))
	}
	return months
}

// monthRangeToTimes converts a YYYY-MM range into time bounds for querying:
// start is the first day of the from month; end is the first day of the month
// AFTER the to month (exclusive upper bound covering all of the to month).
// Both instants are built in America/New_York so they align exactly with the
// query's NY month bucketing — otherwise worklogs in the midnight-to-dawn UTC
// window at a month edge would leak into the adjacent NY month. If the zone
// fails to load, fall back to UTC.
func monthRangeToTimes(from, to string) (time.Time, time.Time) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	f, _ := time.Parse("2006-01", from)
	t, _ := time.Parse("2006-01", to)
	start := time.Date(f.Year(), f.Month(), 1, 0, 0, 0, 0, loc)
	end := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
	return start, end
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./cmd/jiraworklog/ -run 'TestPreviousMonth|TestDefaultAndClampRange|TestEnumerateMonths|TestMonthRangeToTimes' -v`
Expected: PASS (all four).

- [ ] **Step 5: Commit**

```bash
git add cmd/jiraworklog/timesheets.go cmd/jiraworklog/timesheets_test.go
git commit -m "feat(timesheets): add month range helpers with tests"
```

---

### Task 3: Pivot and trend helpers in the pages package (pure functions)

**Files:**
- Create: `templates/pages/timesheets_logic.go`
- Test: `templates/pages/timesheets_logic_test.go`

**Interfaces:**
- Consumes: `types.TimesheetHours` (Task 1).
- Produces (package `pages`):
  - `type TimesheetChargeRow struct { ProjectCharge string; MonthHours map[string]float64; Total float64 }`
  - `type TimesheetAuthor struct { Author string; Role string; Charges []TimesheetChargeRow; MonthTotals map[string]float64; GrandTotal float64 }`
  - `pivotTimesheet(data []types.TimesheetHours, months []string) []TimesheetAuthor` — authors sorted alphabetically; charges sorted alphabetically within each author; `MonthTotals` keyed by every month in `months`.
  - `trendClass(cur, prev float64, isFirst bool) string` — `""` when `isFirst` or `prev == 0`; `"bg-green-50"` when `(cur-prev)/prev > 0.10`; `"bg-red-50"` when `< -0.10`; else `""`.

- [ ] **Step 1: Write the failing tests**

Create `templates/pages/timesheets_logic_test.go`:

```go
package pages

import (
	"testing"

	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
)

func TestPivotTimesheet(t *testing.T) {
	months := []string{"2026-01", "2026-02"}
	data := []types.TimesheetHours{
		{Author: "bob.smith", Role: "dev", ProjectCharge: "TD-100", YearMonth: "2026-01", Hours: 10},
		{Author: "bob.smith", Role: "dev", ProjectCharge: "TD-100", YearMonth: "2026-02", Hours: 12},
		{Author: "bob.smith", Role: "dev", ProjectCharge: "AM-200", YearMonth: "2026-01", Hours: 5},
		{Author: "amy.jones", Role: "qa", ProjectCharge: "TD-100", YearMonth: "2026-02", Hours: 8},
	}

	authors := pivotTimesheet(data, months)
	require.Len(t, authors, 2)

	// Authors sorted alphabetically: amy before bob.
	require.Equal(t, "amy.jones", authors[0].Author)
	require.Equal(t, "bob.smith", authors[1].Author)

	bob := authors[1]
	require.Equal(t, "dev", bob.Role)
	// Charges sorted alphabetically: AM-200 before TD-100.
	require.Len(t, bob.Charges, 2)
	require.Equal(t, "AM-200", bob.Charges[0].ProjectCharge)
	require.Equal(t, "TD-100", bob.Charges[1].ProjectCharge)

	td := bob.Charges[1]
	require.Equal(t, 10.0, td.MonthHours["2026-01"])
	require.Equal(t, 12.0, td.MonthHours["2026-02"])
	require.Equal(t, 22.0, td.Total)

	// Per-month totals across charges: Jan = 10 + 5 = 15, Feb = 12.
	require.Equal(t, 15.0, bob.MonthTotals["2026-01"])
	require.Equal(t, 12.0, bob.MonthTotals["2026-02"])
	require.Equal(t, 27.0, bob.GrandTotal)
}

func TestTrendClass(t *testing.T) {
	// First column is always neutral.
	require.Equal(t, "", trendClass(100, 0, true))
	// No baseline (prev == 0) is neutral even when not first.
	require.Equal(t, "", trendClass(100, 0, false))
	// Within ±10% is neutral.
	require.Equal(t, "", trendClass(105, 100, false))
	require.Equal(t, "", trendClass(95, 100, false))
	// More than +10% is green.
	require.Equal(t, "bg-green-50", trendClass(120, 100, false))
	// More than -10% is red.
	require.Equal(t, "bg-red-50", trendClass(80, 100, false))
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./templates/pages/ -run 'TestPivotTimesheet|TestTrendClass' -v`
Expected: compile failure — `pivotTimesheet`, `trendClass`, and the structs are undefined.

- [ ] **Step 3: Implement the logic**

Create `templates/pages/timesheets_logic.go`:

```go
package pages

import (
	"sort"

	"github.com/mkobaly/jiraworklog/types"
)

// TimesheetChargeRow is one author's hours for a single raw project charge,
// broken out by month plus an across-months total.
type TimesheetChargeRow struct {
	ProjectCharge string
	MonthHours    map[string]float64 // "YYYY-MM" -> hours
	Total         float64
}

// TimesheetAuthor is the pivoted view of one author: their per-charge rows, the
// per-month sums across all their charges, and the grand total.
type TimesheetAuthor struct {
	Author      string
	Role        string
	Charges     []TimesheetChargeRow
	MonthTotals map[string]float64 // "YYYY-MM" -> sum across charges
	GrandTotal  float64
}

// pivotTimesheet transforms flat TimesheetHours rows into author-centric data.
// months is the ordered list of columns to populate (so months with no hours
// still get a zero entry). Authors are sorted alphabetically and each author's
// charges are sorted alphabetically.
func pivotTimesheet(data []types.TimesheetHours, months []string) []TimesheetAuthor {
	type acc struct {
		role    string
		charges map[string]*TimesheetChargeRow
	}
	authorMap := make(map[string]*acc)

	for _, item := range data {
		a, ok := authorMap[item.Author]
		if !ok {
			a = &acc{role: item.Role, charges: make(map[string]*TimesheetChargeRow)}
			authorMap[item.Author] = a
		}
		row, ok := a.charges[item.ProjectCharge]
		if !ok {
			row = &TimesheetChargeRow{
				ProjectCharge: item.ProjectCharge,
				MonthHours:    make(map[string]float64),
			}
			a.charges[item.ProjectCharge] = row
		}
		row.MonthHours[item.YearMonth] += item.Hours
		row.Total += item.Hours
	}

	authorNames := make([]string, 0, len(authorMap))
	for name := range authorMap {
		authorNames = append(authorNames, name)
	}
	sort.Strings(authorNames)

	result := make([]TimesheetAuthor, 0, len(authorNames))
	for _, name := range authorNames {
		a := authorMap[name]

		chargeKeys := make([]string, 0, len(a.charges))
		for k := range a.charges {
			chargeKeys = append(chargeKeys, k)
		}
		sort.Strings(chargeKeys)

		author := TimesheetAuthor{
			Author:      name,
			Role:        a.role,
			MonthTotals: make(map[string]float64),
		}
		// Ensure every month key exists in MonthTotals (zero default).
		for _, m := range months {
			author.MonthTotals[m] = 0
		}
		for _, k := range chargeKeys {
			row := a.charges[k]
			author.Charges = append(author.Charges, *row)
			for _, m := range months {
				author.MonthTotals[m] += row.MonthHours[m]
			}
			author.GrandTotal += row.Total
		}
		result = append(result, author)
	}
	return result
}

// trendClass returns the Tailwind background class for a month cell based on its
// percent change from the previous month. Neutral (empty string) on the first
// column or when there is no prior-month baseline (prev == 0). Green when up
// more than 10%, red when down more than 10%.
func trendClass(cur, prev float64, isFirst bool) string {
	if isFirst || prev == 0 {
		return ""
	}
	pct := (cur - prev) / prev
	if pct > 0.10 {
		return "bg-green-50"
	}
	if pct < -0.10 {
		return "bg-red-50"
	}
	return ""
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./templates/pages/ -run 'TestPivotTimesheet|TestTrendClass' -v`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
git add templates/pages/timesheets_logic.go templates/pages/timesheets_logic_test.go
git commit -m "feat(timesheets): add pivot and trend helpers with tests"
```

---

### Task 4: Timesheets templ page

**Files:**
- Create: `templates/pages/timesheets.templ`
- (Generated) `templates/pages/timesheets_templ.go` via `templ generate`

**Interfaces:**
- Consumes: `pages.TimesheetAuthor`, `pages.TimesheetChargeRow`, `pivotTimesheet`, `trendClass` (Task 3); `types.TimesheetHours` (Task 1); `layouts.Base`.
- Produces:
  - `pages.Timesheets(data []types.TimesheetHours, months []string, allRoles, selectedRoles []string, fromMonth, toMonth, maxMonth string)` templ component.

- [ ] **Step 1: Write the template**

Create `templates/pages/timesheets.templ`:

```templ
package pages

import (
	"fmt"

	"github.com/mkobaly/jiraworklog/templates/layouts"
	"github.com/mkobaly/jiraworklog/types"
)

// formatMonthLabel converts "2026-05" into "May 2026" for column headers.
func formatMonthLabel(ym string) string {
	t, err := timeParseMonth(ym)
	if err != nil {
		return ym
	}
	return t.Format("Jan 2006")
}

func isRoleSelectedTimesheet(role string, selectedRoles []string) bool {
	for _, r := range selectedRoles {
		if r == role {
			return true
		}
	}
	return false
}

// fmtHours renders an hours value, blank for zero so empty cells read cleanly.
func fmtHours(h float64) string {
	if h == 0 {
		return ""
	}
	return fmt.Sprintf("%.1f", h)
}

templ Timesheets(data []types.TimesheetHours, months []string, allRoles []string, selectedRoles []string, fromMonth string, toMonth string, maxMonth string) {
	@layouts.Base("Timesheets") {
		<div class="mb-6">
			<h1 class="text-3xl font-bold text-gray-900">Timesheets</h1>
			<p class="text-gray-600 mt-2">Monthly hours by author, grouped by project charge. Cells turn subtly green or red when a month is more than 10% above or below the prior month.</p>
		</div>
		<!-- Filters -->
		<form method="GET" action="/reports/timesheets" class="mb-6 bg-white rounded-lg shadow-md p-4">
			<div class="flex flex-wrap items-end gap-6">
				<div class="flex items-center gap-2">
					<label class="text-sm font-medium text-gray-700 whitespace-nowrap">From:</label>
					<input
						type="month"
						name="from"
						value={ fromMonth }
						max={ maxMonth }
						class="border border-gray-300 rounded-md px-2 py-1 text-sm focus:outline-hidden focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<div class="flex items-center gap-2">
					<label class="text-sm font-medium text-gray-700 whitespace-nowrap">To:</label>
					<input
						type="month"
						name="to"
						value={ toMonth }
						max={ maxMonth }
						class="border border-gray-300 rounded-md px-2 py-1 text-sm focus:outline-hidden focus:ring-2 focus:ring-blue-500"
					/>
				</div>
				<!-- Role filter dropdown -->
				<div>
					<label class="block text-sm font-medium text-gray-700 mb-1">Roles</label>
					<div class="relative inline-block">
						<button
							type="button"
							id="role-dropdown-btn"
							class="inline-flex items-center px-4 py-2 bg-white border border-gray-300 rounded-lg shadow-xs hover:bg-gray-50 focus:outline-hidden focus:ring-2 focus:ring-blue-500"
							onclick="toggleRoleDropdown()"
						>
							<span class="mr-2">Filter by Roles</span>
							<span id="selected-count" class="bg-blue-100 text-blue-800 text-xs font-medium px-2 py-0.5 rounded-full">{ fmt.Sprintf("%d", len(selectedRoles)) }</span>
							<svg class="ml-2 h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
							</svg>
						</button>
						<div id="role-dropdown" class="hidden absolute z-10 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200">
							<div class="p-3 border-b border-gray-200">
								<label class="flex items-center cursor-pointer">
									<input
										type="checkbox"
										id="select-all-roles"
										class="h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
										onclick="toggleAllRoles(this)"
										if len(selectedRoles) == len(allRoles) {
											checked
										}
									/>
									<span class="ml-2 text-sm font-medium text-gray-700">Select All</span>
								</label>
							</div>
							<div class="max-h-60 overflow-y-auto p-2">
								for _, role := range allRoles {
									<label class="flex items-center px-2 py-1.5 rounded hover:bg-gray-100 cursor-pointer">
										<input
											type="checkbox"
											name="roles"
											value={ role }
											class="role-checkbox h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
											if isRoleSelectedTimesheet(role, selectedRoles) {
												checked
											}
										/>
										<span class="ml-2 text-sm text-gray-700">{ role }</span>
									</label>
								}
							</div>
						</div>
					</div>
				</div>
				<div>
					<button type="submit" class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-lg hover:bg-blue-700 focus:outline-hidden focus:ring-2 focus:ring-blue-500">Apply</button>
				</div>
			</div>
		</form>
		<!-- Legend -->
		<div class="mb-4 flex flex-wrap gap-4 text-sm">
			<div class="flex items-center">
				<span class="w-3 h-3 bg-green-50 border border-green-200 rounded mr-2"></span>
				<span>More than 10% above prior month</span>
			</div>
			<div class="flex items-center">
				<span class="w-3 h-3 bg-red-50 border border-red-200 rounded mr-2"></span>
				<span>More than 10% below prior month</span>
			</div>
		</div>
		if len(data) == 0 {
			<div class="bg-yellow-50 border-l-4 border-yellow-400 p-4">
				<p class="text-yellow-700">No hours data found for the selected roles in the selected range.</p>
			</div>
		} else {
			<div class="bg-white rounded-lg shadow-md overflow-hidden">
				<div class="overflow-x-auto">
					<table class="min-w-full divide-y divide-gray-200">
						<thead class="bg-gray-50">
							<tr>
								<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider sticky left-0 bg-gray-50">Author</th>
								<th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Project Charge</th>
								for _, m := range months {
									<th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider min-w-[90px]">{ formatMonthLabel(m) }</th>
								}
								<th class="px-4 py-3 text-center text-xs font-medium text-gray-500 uppercase tracking-wider bg-blue-50 min-w-[90px]">Total</th>
							</tr>
						</thead>
						<tbody class="bg-white divide-y divide-gray-200">
							for _, author := range pivotTimesheet(data, months) {
								for ci, charge := range author.Charges {
									<tr class="hover:bg-gray-50">
										if ci == 0 {
											<td class="px-4 py-2 whitespace-nowrap sticky left-0 bg-white align-top text-sm font-medium text-gray-900" rowspan={ fmt.Sprintf("%d", len(author.Charges)) }>
												{ author.Author }
												<div class="text-xs text-gray-400 font-normal">{ author.Role }</div>
											</td>
										}
										<td class="px-4 py-2 whitespace-nowrap text-sm text-gray-700">{ charge.ProjectCharge }</td>
										for mi, m := range months {
											<td class={ "px-4 py-2 text-center text-sm text-gray-700", trendClass(charge.MonthHours[m], prevMonthHours(charge, months, mi), mi == 0) }>
												{ fmtHours(charge.MonthHours[m]) }
											</td>
										}
										<td class="px-4 py-2 text-center text-sm font-medium text-gray-900 bg-blue-50">{ fmt.Sprintf("%.1f", charge.Total) }</td>
									</tr>
								}
								<!-- Author subtotal row -->
								<tr class="bg-gray-100 font-semibold border-t-2 border-gray-300">
									<td class="px-4 py-2 sticky left-0 bg-gray-100 text-sm text-gray-900">Total</td>
									for _, m := range months {
										<td class="px-4 py-2 text-center text-sm text-gray-900">{ fmt.Sprintf("%.1f", author.MonthTotals[m]) }</td>
									}
									<td class="px-4 py-2 text-center text-sm text-gray-900 bg-blue-100">{ fmt.Sprintf("%.1f", author.GrandTotal) }</td>
								</tr>
							}
						</tbody>
					</table>
				</div>
			</div>
		}
		@timesheetsScript()
	}
}

templ timesheetsScript() {
	<script>
		function toggleRoleDropdown() {
			document.getElementById('role-dropdown').classList.toggle('hidden');
		}
		document.addEventListener('click', function(event) {
			const dropdown = document.getElementById('role-dropdown');
			const btn = document.getElementById('role-dropdown-btn');
			if (dropdown && btn && !dropdown.contains(event.target) && !btn.contains(event.target)) {
				dropdown.classList.add('hidden');
			}
		});
		function toggleAllRoles(selectAllCheckbox) {
			document.querySelectorAll('.role-checkbox').forEach(cb => { cb.checked = selectAllCheckbox.checked; });
			updateSelectedCount();
		}
		function updateSelectedCount() {
			const checked = document.querySelectorAll('.role-checkbox:checked');
			const countEl = document.getElementById('selected-count');
			if (countEl) countEl.textContent = checked.length;
		}
		document.querySelectorAll('.role-checkbox').forEach(cb => cb.addEventListener('change', updateSelectedCount));
	</script>
}
```

- [ ] **Step 2: Add the two small Go helpers the template references**

These are referenced by the template (`timeParseMonth`, `prevMonthHours`). Add them to `templates/pages/timesheets_logic.go` so they compile. Append:

```go
// timeParseMonth parses a YYYY-MM string. Wrapper kept here so the .templ file
// does not need to import the time package directly.
func timeParseMonth(ym string) (time.Time, error) {
	return time.Parse("2006-01", ym)
}

// prevMonthHours returns the hours for the month immediately before index i in
// months for the given charge row (0 when i == 0). Used to feed trendClass from
// the template.
func prevMonthHours(charge TimesheetChargeRow, months []string, i int) float64 {
	if i == 0 {
		return 0
	}
	return charge.MonthHours[months[i-1]]
}
```

And add `"time"` to the imports of `templates/pages/timesheets_logic.go`:

```go
import (
	"sort"
	"time"

	"github.com/mkobaly/jiraworklog/types"
)
```

- [ ] **Step 3: Generate templ code and compile**

Run: `templ generate && go build ./...`
Expected: no errors; `templates/pages/timesheets_templ.go` is created.

(If `templ` is not on PATH, run `./build.sh`, which invokes it.)

- [ ] **Step 4: Re-run the pages unit tests to confirm nothing broke**

Run: `go test ./templates/pages/ -run 'TestPivotTimesheet|TestTrendClass' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add templates/pages/timesheets.templ templates/pages/timesheets_templ.go templates/pages/timesheets_logic.go
git commit -m "feat(timesheets): add timesheets page template"
```

---

### Task 5: Wire up the handler, route, and nav

**Files:**
- Modify: `cmd/jiraworklog/handlers.go` (add `GetTimesheets`, e.g. after `GetWeeklyHours` at line ~966)
- Modify: `cmd/jiraworklog/main.go` (register route after line 130)
- Modify: `templates/layouts/base.templ` (add nav links after lines 133 and 153)
- (Generated) `templates/layouts/base_templ.go` via `templ generate`

**Interfaces:**
- Consumes: `repo.AllRoles`, `repo.TimesheetHours` (Task 1); `previousMonth`, `defaultAndClampRange`, `enumerateMonths`, `monthRangeToTimes` (Task 2); `pages.Timesheets` (Task 4); existing `wantsHTML`, `h.repo`, `h.logger`.
- Produces: `Handler.GetTimesheets(c echo.Context) error`; route `GET /reports/timesheets`; nav links.

- [ ] **Step 1: Add the handler**

In `cmd/jiraworklog/handlers.go`, after `GetWeeklyHours` (after line 966), add:

```go
func (h *Handler) GetTimesheets(c echo.Context) error {
	// Available roles for the filter.
	allRoles, err := h.repo.AllRoles()
	if err != nil {
		h.logger.Error("error fetching roles", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch roles")
	}

	// Selected roles default to all.
	selectedRoles := c.QueryParams()["roles"]
	if len(selectedRoles) == 0 {
		selectedRoles = allRoles
	}

	// Normalize the month range: default blanks to the previous completed month
	// and clamp out the current/future months.
	now := time.Now()
	fromMonth, toMonth := defaultAndClampRange(c.QueryParam("from"), c.QueryParam("to"), now)
	maxMonth := previousMonth(now)
	months := enumerateMonths(fromMonth, toMonth)

	start, end := monthRangeToTimes(fromMonth, toMonth)
	data, err := h.repo.TimesheetHours(selectedRoles, start, end)
	if err != nil {
		h.logger.Error("error fetching timesheet hours", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch timesheet hours")
	}

	if wantsHTML(c) {
		return pages.Timesheets(data, months, allRoles, selectedRoles, fromMonth, toMonth, maxMonth).Render(c.Request().Context(), c.Response().Writer)
	}

	return c.JSON(http.StatusOK, data)
}
```

- [ ] **Step 2: Compile-check the handler**

Run: `go build ./cmd/jiraworklog/`
Expected: builds (the route is not registered yet, but the function must compile). If `time`, `http`, or `pages` are reported unused/missing, they are already imported in `handlers.go` — confirm no new import is needed.

- [ ] **Step 3: Register the route**

In `cmd/jiraworklog/main.go`, after line 130 (`reports.GET("/weekly-hours", handler.GetWeeklyHours)`), add:

```go
	reports.GET("/timesheets", handler.GetTimesheets)
```

- [ ] **Step 4: Add the nav links**

In `templates/layouts/base.templ`, in the **desktop** menu after line 133 (the Weekly Hours link), add:

```templ
						<a href="/reports/timesheets" class="px-3 py-2 rounded-md hover:bg-blue-700 transition">Timesheets</a>
```

And in the **mobile** menu after line 153 (the Weekly Hours link), add the same:

```templ
						<a href="/reports/timesheets" class="px-3 py-2 rounded-md hover:bg-blue-700 transition">Timesheets</a>
```

- [ ] **Step 5: Generate templ and build everything**

Run: `./build.sh`
Expected: `templ generate` regenerates `base_templ.go` (and `timesheets_templ.go`); `go build` succeeds for both platforms.

- [ ] **Step 6: Full test sweep (pure unit tests)**

Run: `go test ./cmd/jiraworklog/ ./templates/pages/`
Expected: PASS. (Integration tests in `./test/...` need a DB and are run separately when available.)

- [ ] **Step 7: Manual smoke test**

Run the app: `go run ./cmd/jiraworklog -v` (needs a valid `config.yaml`). Then:
- Visit `http://localhost:8380/reports/timesheets` — the page loads defaulting to the previous month only.
- Confirm the From/To month pickers will not let you pick the current month (the `max` attribute caps them at the previous month).
- Set From = an earlier month, To = the previous month, click Apply — confirm one column per month appears and cells show subtle green/red tints where a charge's month is >10% above/below the prior month.
- Confirm the Roles dropdown filters authors.

- [ ] **Step 8: Commit**

```bash
git add cmd/jiraworklog/handlers.go cmd/jiraworklog/main.go templates/layouts/base.templ templates/layouts/base_templ.go
git commit -m "feat(timesheets): wire up handler, route, and nav"
```

---

## Self-Review Notes

- **Spec coverage:** Tab/nav (Task 5) ✓; From/To month pickers (Task 4 + Task 2) ✓; current month not selectable (`max` attr Task 4 + `defaultAndClampRange` clamp Task 2) ✓; default = previous month only (Task 2/Task 5) ✓; Roles filter (Task 4/Task 5) ✓; Author → grouped by raw project charge → per-month columns (Task 3/Task 4) ✓; per-charge Total column (Task 3/Task 4) ✓; author subtotal row (Task 3/Task 4) ✓; ±10% green/red vs previous month, subtle (Task 3 `trendClass` `bg-green-50`/`bg-red-50` + Task 4) ✓; testing (Task 1 integration, Tasks 2–3 unit) ✓.
- **Type consistency:** `TimesheetHours` (db tags `role/author/projectcharge/yearmonth/hours`), `TimesheetChargeRow` (`ProjectCharge/MonthHours/Total`), `TimesheetAuthor` (`Author/Role/Charges/MonthTotals/GrandTotal`), and `pages.Timesheets(...)` signature are used identically across Tasks 1, 3, 4, 5.
- **Boundary consistency:** repo SQL uses `w.date < $3` (exclusive); `monthRangeToTimes` returns first-day-of-next-month as the exclusive end — they match. Both bounds are built in `America/New_York`, matching the SQL's NY month bucketing, so no hours leak across a month edge.
```
