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
