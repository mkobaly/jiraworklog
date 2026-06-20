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
	// A blank/no-hours cell (cur == 0) is neutral, never tinted.
	require.Equal(t, "", trendClass(0, 100, false))
}
