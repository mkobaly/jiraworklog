package test

import (
	"testing"
	"time"

	"github.com/mkobaly/jiraworklog/internal"
	"github.com/stretchr/testify/require"
)

var utcOffset = -5
var tz, _ = time.LoadLocation("America/New_York")

func TestUnixTimeStamp(t *testing.T) {
	te := time.Unix(1773169200, 0)
	str := te.Format("2006-01-02 15:04")
	require.Equal(t, "2026-03-10 15:00", str)
}

// 1774735800
func TestGetDateOnly(t *testing.T) {
	lastTS := time.Unix(1774735800, 0) //Saturday, March 28, 2026 at 10:10:00 PM UTC
	lastTSDateOnly := internal.DateOnly(time.Unix(lastTS.Unix(), 0)).Unix()
	require.Equal(t, int64(1774656000), lastTSDateOnly)
}

func TestGetDateRangeManyDaysAgo(t *testing.T) {
	now := time.Date(2026, 3, 5, 16, 25, 0, 0, tz)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-01 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-03 00:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeOneDayAgo(t *testing.T) {
	now := time.Date(2026, 3, 3, 16, 25, 0, 0, tz)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-01 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-03 00:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForSameDaysHoursApart(t *testing.T) {
	now := time.Date(2026, 3, 2, 8, 0, 0, 0, tz)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-01 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 01:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForSameDaysHoursMinsApartMidnight(t *testing.T) {
	now := time.Date(2026, 3, 2, 8, 23, 0, 0, tz)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-01 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 01:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForSameDaysHoursMinsApartTwoAM(t *testing.T) {
	now := time.Date(2026, 3, 2, 8, 23, 0, 0, tz)
	lastTS := time.Date(2026, 3, 2, 2, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-02 01:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 03:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForFutureDatePastNow(t *testing.T) {
	now := time.Date(2026, 3, 3, 3, 23, 0, 0, tz)
	lastTS := time.Date(2026, 3, 5, 15, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	//Expect ts to be now start of day - 5m & te to be now
	require.Equal(t, "2026-03-02 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-03 03:23", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForTodayWithHour2HoursFromNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 12, 16, 46, 0, tz)
	lastTS := time.Date(2026, 3, 2, 10, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-02 09:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 11:00", internal.JiraDateString(te, tz))
}

func TestGetDateRangeForTodayWithHour1HourFromNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 12, 16, 46, 0, tz)
	lastTS := time.Date(2026, 3, 2, 11, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	require.Equal(t, "2026-03-02 10:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 12:00", internal.JiraDateString(te, tz))

}

func TestGetDateRangeForTodayWithHourEqualNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 12, 16, 46, 0, tz)
	lastTS := time.Date(2026, 3, 2, 12, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	//Expect ts to be lastTS - 5m & te to be now hour + now mins
	require.Equal(t, "2026-03-02 11:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 12:16", internal.JiraDateString(te, tz))
}

func TestGetDateRangePastNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 17, 16, 46, 0, tz)
	lastTS := time.Date(2026, 3, 2, 18, 0, 0, 0, tz)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.UTC().Unix(), tz)
	//Expect ts to be day of now - 5m & te to be day of now + hour + min
	require.Equal(t, "2026-03-01 23:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-02 17:16", internal.JiraDateString(te, tz))
}

func TestFoo(t *testing.T) {
	now := time.Unix(1774820683, 0).UTC()
	lastTS := time.Unix(1774818000, 0).UTC()
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix(), tz)
	require.Equal(t, "2026-03-29 16:55", internal.JiraDateString(ts, tz))
	require.Equal(t, "2026-03-29 17:44", internal.JiraDateString(te, tz))
}
