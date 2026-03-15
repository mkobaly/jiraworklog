package test

import (
	"testing"
	"time"

	"github.com/mkobaly/jiraworklog/internal"
	"github.com/stretchr/testify/require"
)

func TestUnixTimeStamp(t *testing.T) {
	te := time.Unix(1773169200, 0)
	str := te.Format("2006-01-02 15:04")
	require.Equal(t, "2026-03-10 15:00", str)
}

func TestGetDateRangeManyDaysAgo(t *testing.T) {
	now, _ := time.ParseInLocation("2006-01-02 15:04", "2026-03-05 16:25", time.UTC)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 24 hrs
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, lastTS.Add(time.Hour*24), time.Unix(te, 0))
}

func TestGetDateRangeForToday(t *testing.T) {
	now := time.Date(2026, 3, 2, 4, 23, 0, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 0, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, lastTS.Add(time.Hour), time.Unix(te, 0))
}

func TestGetDateRangeForTodayWithHoursPopulated(t *testing.T) {
	now := time.Date(2026, 3, 2, 4, 23, 0, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 2, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, lastTS.Add(time.Hour), time.Unix(te, 0))
}

func TestGetDateRangeForTodayWithHour2HoursFromNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 17, 16, 46, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 15, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, lastTS.Add(time.Hour), time.Unix(te, 0))
}

func TestGetDateRangeForTodayWithHour1HourFromNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 17, 16, 46, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 16, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, lastTS.Add(time.Hour), time.Unix(te, 0))
}

func TestGetDateRangeForTodayWithHourEqualNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 17, 16, 46, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 17, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, now.Add(time.Second*-46), time.Unix(te, 0))
}

func TestGetDateRangePastNow(t *testing.T) {
	now := time.Date(2026, 3, 2, 17, 16, 46, 0, time.Local)
	lastTS := time.Date(2026, 3, 2, 18, 0, 0, 0, time.Local)
	ts, te := internal.GetDateRange(lastTS.Unix(), now.Unix())
	//Expect ts to be lastTS - 5m & te to be lastTS + 1 hr
	require.Equal(t, lastTS.Add(time.Hour*-18).Add(time.Minute*-5), time.Unix(ts, 0))
	require.Equal(t, now.Add(time.Second*-46), time.Unix(te, 0))
}
