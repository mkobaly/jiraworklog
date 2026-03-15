package internal

import "time"

func GetDateRange(lastTimestamp int64, now int64) (ts, te int64) {
	nowTime := time.Unix(now, 0)
	lastrun := time.Unix(lastTimestamp, 0)
	ty, tm, td := lastrun.Date()
	ny, nm, nd := nowTime.Date()
	isToday := ty == ny && tm == nm && td == nd

	if isToday {
		lastRunHour := lastrun.Hour()
		nowHour := nowTime.Hour()
		startOfHour := time.Date(ty, tm, td, lastRunHour, 0, 0, 0, lastrun.Location())
		if lastRunHour < nowHour {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			te = startOfHour.Add(time.Hour).Unix()
		} else if lastRunHour > nowHour {
			startOfDay := time.Date(ty, tm, td, 0, 0, 0, 0, lastrun.Location())
			end := time.Date(ty, tm, td, nowTime.Hour(), nowTime.Minute(), 0, 0, lastrun.Location())
			ts = startOfDay.Add(time.Minute * -5).Unix()
			te = end.Unix()
		} else {
			ts = startOfHour.Add(time.Minute * -5).Unix()
			end := time.Date(ty, tm, td, nowTime.Hour(), nowTime.Minute(), 0, 0, lastrun.Location())
			te = end.Unix()
		}
	} else {
		startOfDay := time.Date(ty, tm, td, 0, 0, 0, 0, lastrun.Location())
		ts = startOfDay.Add(time.Minute * -5).Unix()
		te = startOfDay.Add(time.Hour * 24).Unix()
	}
	return
}
