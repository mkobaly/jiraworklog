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
