package internal

import "time"

func GetDateRange(lastTimestamp int64, utcNow int64, offset int) (ts, te int64) {
	utcNowTime := time.Unix(utcNow, 0)
	loc := time.FixedZone("UTC-x", offset*3600)
	localNowTime := utcNowTime.In(loc)

	lastrun := time.Unix(lastTimestamp, 0)
	ty, tm, td := lastrun.Date()
	ny, nm, nd := localNowTime.Date()
	isToday := ty == ny && tm == nm && td == nd

	if isToday {
		lastRunHour := lastrun.Hour()
		nowHour := localNowTime.Hour()
		startOfHour := time.Date(ty, tm, td, lastRunHour, 0, 0, 0, lastrun.Location())
		if lastRunHour < nowHour {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			te = startOfHour.Add(time.Hour).Unix()
		} else if lastRunHour > nowHour {
			startOfDay := time.Date(ty, tm, td, 0, 0, 0, 0, lastrun.Location())
			end := time.Date(ty, tm, td, localNowTime.Hour(), localNowTime.Minute(), 0, 0, lastrun.Location())
			ts = startOfDay.Add(time.Minute * -5).Unix()
			te = end.Unix()
		} else {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			end := time.Date(ty, tm, td, localNowTime.Hour(), localNowTime.Minute(), 0, 0, lastrun.Location())
			te = end.Unix()
		}
	} else {
		startOfDay := time.Date(ty, tm, td, 0, 0, 0, 0, lastrun.Location())
		ts = startOfDay.Add(time.Minute * -5).Unix()
		te = startOfDay.Add(time.Hour * 24).Unix()
	}
	return
}
