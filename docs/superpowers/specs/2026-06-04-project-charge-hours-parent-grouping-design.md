# Project Charge Hours — Customizable Parent Grouping

**Date:** 2026-06-04
**Status:** Approved (design)

## Problem

The Project Charge Hours report (`/reports/project-hours`) renders results in
top-level "parent" boxes (the blue headers) that are **hard-coded** to the
project charge's `Label` (e.g. "After Market", "CONNECT 2.0"). Stakeholders now
want to view the same data grouped at the top level by **Category** or **Type**
instead of by Label.

Today the parent key is set in three places in `groupProjectChargeHours`
(`templates/pages/projectchargehours.templ`, ~L70-74) as `item.Label`. Every
other grouping axis — the project-charge sub-rows and the
Month/Employee/Role/Location detail breakdowns — is already parameterized via
`GroupingOptions`. The parent axis is the only rigid dimension.

`ProjectChargeHours` already exposes `Category()` and `Type()` methods (derived
from the charge code via `projectCategory()` / `projectType()`), so **no schema
or SQL change is required**. This is purely a grouping + UI change.

## Decision

Single mutually-exclusive parent selection: the user picks exactly **one** of
`Project (Label)` | `Category` | `Type` as the top-level group. Default remains
`Label`. (Multi-level nesting was considered and rejected as YAGNI.)

## Design

### 1. Grouping options

In `templates/pages/projectchargehours.templ` add `ParentBy` to
`GroupingOptions`:

```go
type GroupingOptions struct {
    ParentBy   string // "label" (default), "category", or "type"
    ByDate     bool
    ByEmployee bool
    ByRole     bool
    ByCharge   bool
    ByLocation bool
}
```

Add a parent-key extractor:

```go
func parentKey(item types.ProjectChargeHours, by string) string {
    switch by {
    case "category":
        return item.Category()
    case "type":
        return item.Type()
    default:
        return item.Label
    }
}
```

In `groupProjectChargeHours`, replace the three hard-coded `item.Label`
references with `parentKey(item, opts.ParentBy)`. Nothing else in the grouping
logic changes — charge sub-grouping, detail-row aggregation, and all existing
sorting (charges by name, projects by name) compose unchanged.

### 2. Template signature cleanup

Collapse the current 6 positional bool parameters of `ProjectChargeHoursPage`
into the single `GroupingOptions` struct (now carrying the 7th option,
`ParentBy`). New signature:

```go
templ ProjectChargeHoursPage(data []types.ProjectChargeHours, filters ChargeFilterState, opts GroupingOptions)
```

All in-template references to `groupByDate`, `groupByCharge`, `groupByEmployee`,
`groupByRole`, `groupByLocation` become `opts.ByDate`, `opts.ByCharge`, etc.
(both desktop and mobile sections, plus the form checkbox `checked` conditions).
The two `groupProjectChargeHours(data, GroupingOptions{...})` call sites simply
pass the already-constructed `opts`.

### 3. Handler

In `GetProjectChargeHours` (`cmd/jiraworklog/handlers.go`):

- Read `groupBy := c.QueryParam("groupBy")`; validate against
  `{"label","category","type"}`, defaulting to `"label"` for any other/empty
  value.
- Build a `pages.GroupingOptions{ParentBy: groupBy, ByDate: ..., ...}` and pass
  it to `ProjectChargeHoursPage`.
- The existing default-on-initial-load behavior for `ByCharge`/`ByRole` (driven
  by the `_f` sentinel) is preserved.

### 4. UI

In the grouping row of the filter form:

- Add a `<select name="groupBy">` with options Project / Category / Type,
  `onchange="this.form.submit()"`, with the current value marked `selected` via
  `opts.ParentBy`.
- Relabel the existing checkbox group label from **"Group by:"** to
  **"Break down by:"**, giving a clean mental model: one top-level group (the
  dropdown) plus detail breakdowns (the checkboxes).

### 5. Redundant-column suppression

When `ParentBy == "category"`, the per-row "Category" column duplicates the
parent header; same for `"type"`. Hide the row-level column that matches the
active parent axis (desktop table headers + cells, and the mobile equivalent).
The "Project Charge" column always remains.

## Out of scope

- **CSV export** stays flat (raw, ungrouped rows). This is intentional: it lets
  users pivot the export however they like in a spreadsheet. `groupBy` does not
  affect the CSV endpoint.
- Multi-level / nested parent grouping.
- Any change to the SQL query, `project_charge` table, or `ProjectChargeHours`
  type fields.

## Testing / verification

- Regenerate templ (`templ generate`) and `go build ./...`.
- Manual: load `/reports/project-hours`, switch the Group by dropdown across
  Project / Category / Type; confirm parent boxes regroup, totals still sum
  correctly, and the redundant column is hidden for Category/Type.
- Confirm refreshing keeps a stable order (parent + charge sorting already
  deterministic) and that the dropdown selection persists across submits.
- Confirm CSV export is unchanged.
