# Monthly Leadership Dashboard — Design

**Date:** 2026-05-29
**Status:** Approved
**Audience:** Engineering leadership (monthly review) + engineering managers (drill-down)

## Goal

Add two new dashboards to surface monthly engineering health metrics:

1. **Leadership view** (`/dashboard/leadership`) — sparse, all teams side-by-side, optimized for a monthly leadership review meeting. Answers "who's improving vs regressing?"
2. **Manager view** (`/dashboard/manager?team=SYM`) — dense, single-team, optimized for managers diagnosing where their team is hurting.

Both pages share the same data layer and pull from existing tables (`issue`, `issue_transition`, `status_stints`, `worklog`, `people`). No new schema.

## v1 scope decisions

- **Team = Jira project** (IDM, SYM, ESG). Each project includes both project-team and stability-team work blended together; v2 will split via `projectCharge`.
- **Time window**: 13 months trailing (last complete month + 12 prior). Matches existing patterns in `CustomerBugTrends` and `ProjectChargeHours`.
- **Granularity**: monthly. The mockup's weekly throughput was a mockup artifact; leadership cadence is monthly.
- **Dropped from v1** (revisit later if leadership asks): Commitment Reliability, Planned vs. Unplanned Ratio.
- **Caching**: none. Queries operate over a few thousand rows of `status_stints` per team-month; expected response well under 1s. Add caching only if measurement shows it's needed.

## Architecture

```
cmd/jiraworklog/handlers.go
  ├── GET /dashboard/leadership   → renders templates/pages/leadership.templ
  └── GET /dashboard/manager      → renders templates/pages/manager.templ

repository/posgres.go (extended, no new file)
  ├── MonthlyTeamMetrics(teams, fromMonth, toMonth) []MonthlyTeamMetrics
  └── ManagerMetrics(team, fromMonth, toMonth)      ManagerMetricsData

types/monthly_metrics.go (new)
  Types: MonthlyTeamMetrics, ManagerMetricsData, AgingWIPItem,
         HoursRatioPoint, ReworkCyclesPoint, BouncePoint, TimeInStatusPoint
```

### Key architectural choices

1. **Two query entry points, not one giant query.** `MonthlyTeamMetrics` is cheap (≈39 rows for 3 teams × 13 months); `ManagerMetrics` is the heavier single-team detail and only runs when a team is selected. Keeps the leadership page fast.
2. **Reuse existing bucket logic.** The Dev/QA/Waiting status mapping at `posgres.go:644-700` is lifted into Go-level SQL constants so both per-epic and monthly queries share the same definition. Eliminates drift risk.
3. **No new schema.** Everything derives from existing tables.

### Shared SQL constants

```go
// Population for the 5 close-month metrics
const closeableTypes = `('Story','Bug','Hardware Bug','Customer Bug','HW / FW Customer Bug')`

// Bucket mapping — matches existing per-epic logic at posgres.go:644-700
const bucketCase = `
    CASE
      WHEN status IN ('In Development','In Progress','Code Complete','Code Merged','In Review') THEN 'Dev'
      WHEN status IN ('In QA','Failed QA') THEN 'QA'
      WHEN status IN ('On Hold','QA Backlog') THEN 'Waiting'
      ELSE 'Other'
    END`
```

The existing per-epic query bodies in `ProjectKPIs()` will be updated to reference these constants. A regression check confirms identical output before/after the lift.

## Status bucketing (settled in existing code)

| Bucket | Statuses |
|---|---|
| **Dev** (counted in cycle time + flow active) | In Development, In Progress, Code Complete, Code Merged, In Review |
| **QA** (counted in cycle time + flow active) | In QA, Failed QA |
| **Waiting** (counted in cycle time denominator only, not active) | On Hold, QA Backlog |
| **Excluded** (not counted at all) | Development Backlog, To Do, Bug Draft, Done, Closed, Cancelled, Awaiting Release to Customer |

**Cycle Time** = sum of time in Dev + QA buckets.
**Flow Efficiency** = (Dev + QA) / (Dev + QA + Waiting + Done).

## Monthly bucketing rules

| Metric | Bucket month is determined by | Notes |
|---|---|---|
| Median Cycle Time | Month issue first entered `Done`/`Closed`/`Cancelled`/`Awaiting Release to Customer` | "Close month" |
| Throughput | Close month | Same definition |
| Flow Efficiency | Close month | Same definition |
| Failed QA Ratio | Close month | Stories only |
| Defect Escape Rate | Bug's `createdate` month | Measures when bugs were *found*, not closed |
| Bug vs Forward-Work Hours | `worklog.date` month | Measures effort spent that month |
| QA vs Eng Hours | `worklog.date` month | Same |
| Time in Status | Month status was *exited* | Each stint contributes to its exit month |
| Rework Cycles | Close month | QA→Dev transitions per closed issue |
| Status Bounce Rate | Close month | Re-entries per closed issue |
| WIP series | Daily snapshot, charted as monthly time series | Issues currently in Dev/QA on the snapshot date |
| Aging WIP | Current snapshot only — no monthly trend | Lists issues currently exceeding team's 85th-pctile cycle time |

## Type shapes

```go
// One row per (team, year_month) — drives the leadership grid
type MonthlyTeamMetrics struct {
    Team                 string  `db:"team"`
    YearMonth            string  `db:"year_month"`     // "2026-05"
    MedianCycleTimeSecs  float64 `db:"median_cycle_time_secs"`
    ThroughputCount      int     `db:"throughput_count"`
    FlowEfficiencyPct    float64 `db:"flow_efficiency_pct"`
    FailedQARatioPct     float64 `db:"failed_qa_ratio_pct"`
    DefectEscapeRatePct  float64 `db:"defect_escape_rate_pct"`
    // Stability — embedded for one-page render; sourced from existing CustomerBugTrend
    StabilityNewCount    int     `db:"stability_new"`
    StabilityClosedCount int     `db:"stability_closed"`
    StabilityOpenCount   int     `db:"stability_open"`
}

// Single-team detail page payload
type ManagerMetricsData struct {
    Team               string
    Leadership         []MonthlyTeamMetrics    // 13 months for this team
    BugVsForwardHours  []HoursRatioPoint       // 13 monthly points
    QAVsEngHours       []HoursRatioPoint       // 13 monthly points
    ReworkCycles       []ReworkCyclesPoint     // 13 monthly points
    StatusBounceRate   []BouncePoint           // 13 monthly points
    TimeInStatus       []TimeInStatusPoint     // 13 months × N buckets (stacked)
    WIPSeries          []WIPDataPoint          // existing type, team-filtered
    AgingWIPItems      []AgingWIPItem          // current snapshot, list
}

type HoursRatioPoint struct {
    YearMonth   string  `db:"year_month"`
    Numerator   float64 `db:"numerator_hours"`
    Denominator float64 `db:"denominator_hours"`
    Ratio       float64 `db:"ratio"`
}

type ReworkCyclesPoint struct {
    YearMonth string  `db:"year_month"`
    AvgCycles float64 `db:"avg_cycles"`
    IssueCount int    `db:"issue_count"`
}

type BouncePoint struct {
    YearMonth   string  `db:"year_month"`
    BouncePct   float64 `db:"bounce_pct"`
    TotalIssues int     `db:"total_issues"`
}

type TimeInStatusPoint struct {
    YearMonth string  `db:"year_month"`
    Bucket    string  `db:"bucket"`     // 'Dev' | 'QA' | 'Waiting'
    AvgSecs   float64 `db:"avg_seconds"`
}

type AgingWIPItem struct {
    Key          string `db:"key"`
    Status       string `db:"status"`
    DaysInStatus int    `db:"days_in_status"`
}
```

## Metric definitions (SQL-precise)

### Close-month CTE (reused by 5 leadership metrics + manager rework/bounce)

```sql
issue_close_month AS (
  SELECT issueid, to_char(MIN(datestarted), 'YYYY-MM') AS year_month
  FROM status_stints
  WHERE status IN ('Done','Closed','Cancelled','Awaiting Release to Customer')
  GROUP BY issueid
)
```

### Leadership metrics

**Median Cycle Time** — `percentile_cont(0.5)` of per-issue total Dev+QA seconds, grouped by (project, close month). Population: `closeableTypes`.

**Throughput** — `COUNT(DISTINCT issueid)` grouped by (project, close month). Population: `closeableTypes`.

**Flow Efficiency** — `SUM(active) / SUM(active + waiting + done)` across closed issues in the month. Active = Dev + QA stint seconds. Same population.

**Failed QA Ratio** — `COUNT(stories that hit 'Failed QA' status at least once) / COUNT(closed stories)`, by (project, close month). Population: `Story` only.

**Defect Escape Rate** — `COUNT(customer bugs created in month) / COUNT(all bugs created in month)`, by (project, `createdate` month). Customer bugs = `Customer Bug` + `HW / FW Customer Bug`. Internal bugs = `Bug` + `Hardware Bug`.

### Stability (existing — no new query)

Uses `CustomerBugTrends()` from `posgres.go:144` unchanged. Called once per team. Result feeds both the leadership stability strip and the manager Stability section.

### Manager-only metrics

**Rework Cycles** — `AVG(QA→Dev transition count per issue)` for issues closed in the month. Reuses the pattern from `posgres.go:776` (QA Rework Cycles).

**Status Bounce Rate** — `SUM(re-entries) / SUM(total status entries)` for issues closed in the month. Adaptation of `posgres.go:731`.

**QA vs Eng Hours** — `SUM(worklog.timespenthours WHERE people.role='qa') / SUM(WHERE people.role='dev')`, by (project, `worklog.date` month).

**Bug vs Forward-Work Hours** — `SUM(hours on Bug + Hardware Bug) / SUM(hours on Story + Task)`, by (project, `worklog.date` month).

**Time in Status** — `AVG(durationseconds)` from `status_stints`, grouped by (project, exit month, bucket). Excludes `Excluded` bucket statuses. Chart stacks Dev/QA/Waiting per month.

**WIP series** — Daily count of issues currently in a Dev/QA status, team-filtered. Reuses the `WIPHistory` query pattern from `posgres.go:794`, removes the per-epic filter.

**Aging WIP** — Current snapshot only. For each open issue in Dev/QA, compute `time_in_current_bucket`. Include if it exceeds the team's own 85th-percentile cycle time over the last 13 months. Returns `Key, Status, DaysInStatus`, sorted descending, capped at 20.

## Page layouts

### Leadership page (`/dashboard/leadership`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ Leadership Dashboard                            [ May 2026 ▼ ]      │
├──────────────────────────────────────────────────────────────────────┤
│                                                                       │
│                       IDM            SYM            ESG               │
│              ┌──────────────┬──────────────┬──────────────┐          │
│ Cycle Time   │ 5.8d   ▼     │ 7.2d   ▲     │ 4.1d   ▼     │          │
│              │ ▁▂▃▂▁▂▁      │ ▁▂▃▅▆▇█      │ ▆▅▄▃▂▁▁      │          │
│              ├──────────────┼──────────────┼──────────────┤          │
│ Throughput   │  42    ▲     │  38    ▬     │  18    ▼     │          │
│              │ ▂▃▄▅▆▇█      │ ▅▅▆▅▆▅▆      │ ▇▆▅▄▃▂▁      │          │
│              ├──────────────┼──────────────┼──────────────┤          │
│ Flow Eff     │  52%   ▼     │  48%   ▲     │  61%   ▲     │          │
│ Failed QA    │  12%   ▼     │  18%   ▲     │   9%   ▼     │          │
│ Defect Esc.  │   8%   ▼     │  15%   ▲     │   5%   ▼     │          │
│              └──────────────┴──────────────┴──────────────┘          │
│                                                                       │
│ Stability (Customer Bugs)                                            │
│              ┌──────────────┬──────────────┬──────────────┐          │
│              │ IDM trend    │ SYM trend    │ ESG trend    │          │
│              └──────────────┴──────────────┴──────────────┘          │
│                                                                       │
│ Footer: "Each project includes both project-team and stability-team   │
│ work; v2 will split these."                                          │
└──────────────────────────────────────────────────────────────────────┘
```

Components:

1. **Header** — title + month-selector dropdown. Defaults to last complete month. Changing the month re-renders the page; sparklines stay 12-month-trailing relative to selected month.
2. **KPI grid** — 5 rows × 3 columns. Each cell: current value, direction arrow vs. prior month (▲/▼/▬), 12-month sparkline. Arrow colors are per-metric (lower cycle time = green ▼, higher throughput = green ▲).
3. **Stability strip** — one chart per team using existing `CustomerBugTrend` data, rendered three times.
4. **Team headers link** to `/dashboard/manager?team=IDM`.
5. **Empty states** — cell shows "—" not "0" when a team has no closed issues that month. When fewer than 5 issues closed that month, the median cycle time cell is annotated with the actual count (e.g., "* n=3") as a statistical caveat.

### Manager page (`/dashboard/manager?team=SYM`)

```
┌──────────────────────────────────────────────────────────────────────┐
│ Manager Dashboard       Team: [ SYM ▼ ]    Window: [ Last 13 months ]│
├──────────────────────────────────────────────────────────────────────┤
│ ── Delivery Predictability ─────────────────────────────────────────  │
│   ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐        │
│   │ Cycle Time │ │ Throughput │ │ Flow Eff   │ │ Failed QA  │        │
│   │  7.2 days  │ │  38/mo     │ │   48%      │ │   18%      │        │
│   │ [13-mo bar]│ │ [13-mo bar]│ │ [13-mo bar]│ │ [13-mo bar]│        │
│   └────────────┘ └────────────┘ └────────────┘ └────────────┘        │
│                                                                       │
│   ┌──────────────────────────────────┐ ┌─────────────────────────┐   │
│   │ Time in Status (monthly stacked) │ │ Aging WIP (current)     │   │
│   │ [Dev / QA / Waiting per month]   │ │ Top 20 stale issues     │   │
│   └──────────────────────────────────┘ └─────────────────────────┘   │
│                                                                       │
│   ┌──────────────────────────────────┐                              │
│   │ WIP Trend (line, last 13 months) │                              │
│   └──────────────────────────────────┘                              │
│                                                                       │
│ ── Quality ─────────────────────────────────────────────────────────  │
│   ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐        │
│   │ Rework     │ │ Status     │ │ QA hrs /   │ │ Bug hrs /  │        │
│   │ Cycles     │ │ Bounce     │ │ Eng hrs    │ │ Fwd-work   │        │
│   └────────────┘ └────────────┘ └────────────┘ └────────────┘        │
│                                                                       │
│ ── Stability ───────────────────────────────────────────────────────  │
│   [Embedded SYM customer-bug new/closed/open trend chart]            │
│                                                                       │
│   ← Back to Leadership Dashboard                                     │
└──────────────────────────────────────────────────────────────────────┘
```

Components:

1. **Team selector** — dropdown of IDM, SYM, ESG. Updates `?team=` URL param so view is shareable. If no `?team=` is provided, defaults to IDM (first alphabetically).
2. **Three sections** — Delivery Predictability, Quality, Stability — matching the original category structure.
3. **KPI cards** with 13-month bar/line charts (bigger than leadership sparklines, axis-labeled).
4. **Time in Status** — stacked bar by month, stack = Dev/QA/Waiting buckets.
5. **Aging WIP list** — current snapshot, top 20 issues, each links to Jira.
6. **WIP trend** — line chart of daily WIP count, last 13 months.
7. **Stability strip** — embedded customer bug trend for the selected team.
8. **Back link** to leadership page.

## Sparklines & direction arrows (UI-side)

- Each metric returns 12 prior values in the same payload — no separate sparkline endpoint.
- Direction arrow compares current vs. prior month with a 5% flat threshold (▬).
- "Improving" direction per metric:
  - Lower is better: Cycle Time, Rework Cycles, Defect Escape Rate, Failed QA Ratio, Status Bounce Rate, Bug-vs-Forward-Work Hours
  - Higher is better: Throughput, Flow Efficiency, QA-vs-Eng Hours

## Navigation

- Add "Leadership" link to base layout nav.
- Team-header links on leadership grid → manager page (`?team=`).
- Manager page has a "← Back to Leadership Dashboard" link.

## Testing approach

- **Repository queries**: extend the existing `test/...` integration tests, hitting a real database. Verify monthly totals against hand-checked Jira counts for one team-month.
- **Templates**: render with sample data, smoke-test that the page compiles and renders.
- **Regression check**: confirm `ProjectKPIs()` output is byte-identical before vs. after lifting the shared SQL constants.

## Implementation phases

### Phase 1 — Leadership page (shippable on its own)

1. Lift `bucketCase` + `closeableTypes` into Go SQL constants; update existing `ProjectKPIs()` to use them; regression-check identical output.
2. Add `MonthlyTeamMetrics(teams, fromMonth, toMonth)` to `repository/posgres.go`.
3. Add `types/monthly_metrics.go` with the new type definitions.
4. Add `templates/pages/leadership.templ` — grid layout, KPI cells, Chart.js sparklines, direction arrows.
5. Per-team Stability strip — reuse existing `CustomerBugTrends()` called once per team.
6. Add `GET /dashboard/leadership` handler in `cmd/jiraworklog/handlers.go` with HTML/JSON content negotiation.
7. Add "Leadership" link to base layout nav.
8. Footer caveat about blended team data.

**Ship Phase 1 → use it for one monthly leadership review → collect feedback → start Phase 2.**

### Phase 2 — Manager page

1. Add `ManagerMetrics(team, fromMonth, toMonth)` query group — 7 new queries.
2. Add `templates/pages/manager.templ` — three-section layout with bigger charts.
3. Add `GET /dashboard/manager?team=SYM` handler.
4. Wire cross-page navigation: leadership grid team headers → manager page; manager → back to leadership.

### Explicitly NOT in v1

- Project-team vs Stability-team split (deferred to v2; will use `projectCharge` code `3` to identify stability work)
- Commitment Reliability
- Planned vs Unplanned Ratio
- Caching layer
- Manual snapshot job for "committed at month start"
- Per-issue drill-down page (link to Jira instead)
