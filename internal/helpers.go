package internal

import (
	"time"
)

func DateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func JiraDateString(timestamp int64, tz *time.Location) string {
	//return time.Unix(timestamp, 0).UTC().Format("2006-01-02 15:04")
	return time.Unix(timestamp, 0).In(tz).Format("2006-01-02 15:04")
}

//America/New_York

func GetDateRange(lastTimestamp int64, utcNow int64, timezone *time.Location) (ts, te int64) {
	utcNowTime := time.Unix(utcNow, 0).UTC()
	localNowTime := utcNowTime.In(timezone)

	lastrun := time.Unix(lastTimestamp, 0).In(timezone)
	lry, lrm, lrd := lastrun.Date()
	ny, nm, nd := localNowTime.Date()
	isToday := lry == ny && lrm == nm && lrd == nd

	//this should never happen but account for it and use local now time
	if lastrun.After(localNowTime) {
		startOfDay := time.Date(ny, nm, nd, 0, 0, 0, 0, timezone)
		end := time.Date(ny, nm, nd, localNowTime.Hour(), localNowTime.Minute(), 0, 0, timezone)
		ts = startOfDay.Add(time.Minute * -5).Unix()
		te = end.Unix()
		return
	}

	if isToday {
		lastRunHour := lastrun.Hour()
		nowHour := localNowTime.Hour()
		startOfHour := time.Date(lry, lrm, lrd, lastRunHour, 0, 0, 0, timezone)
		if lastRunHour < nowHour {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			te = startOfHour.Add(time.Hour).Unix()
		} else if lastRunHour > nowHour {
			//should never happen but if it does go back to hour zero for given last run day
			startOfDay := time.Date(lry, lrm, lrd, 0, 0, 0, 0, timezone)
			end := time.Date(lry, lrm, lrd, localNowTime.Hour(), localNowTime.Minute(), 0, 0, timezone)
			ts = startOfDay.Add(time.Minute * -5).Unix()
			te = end.Unix()
		} else {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			end := time.Date(lry, lrm, lrd, localNowTime.Hour(), localNowTime.Minute(), 0, 0, timezone)
			te = end.Unix()
		}
	} else {
		startOfDay := time.Date(lry, lrm, lrd, 0, 0, 0, 0, timezone)
		ts = startOfDay.Add(time.Minute * -5).Unix()
		te = startOfDay.Add(time.Hour * 24).Unix()
	}
	return
}
