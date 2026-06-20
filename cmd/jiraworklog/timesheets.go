package main

import "time"

// previousMonth returns the YYYY-MM of the month immediately before now's month.
// This is the latest month the Timesheets report allows, since the current month
// is incomplete.
func previousMonth(now time.Time) string {
	firstOfThis := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	prev := firstOfThis.AddDate(0, 0, -1)
	return prev.Format("2006-01")
}

// validMonth reports whether s parses as a YYYY-MM month string.
func validMonth(s string) bool {
	_, err := time.Parse("2006-01", s)
	return err == nil
}

// defaultAndClampRange normalizes the From/To month inputs. Blank or invalid
// values default to the previous completed month. Any month later than the
// previous completed month is clamped down to it (the current/future months are
// not selectable). Finally, if from is after to, from collapses to to.
// YYYY-MM strings compare correctly with string ordering.
func defaultAndClampRange(from, to string, now time.Time) (string, string) {
	max := previousMonth(now)
	if !validMonth(from) {
		from = max
	}
	if !validMonth(to) {
		to = max
	}
	if from > max {
		from = max
	}
	if to > max {
		to = max
	}
	if from > to {
		from = to
	}
	return from, to
}

// enumerateMonths returns the ordered, inclusive list of YYYY-MM months from
// from to to. Assumes from <= to and both are valid (callers pass the output of
// defaultAndClampRange).
func enumerateMonths(from, to string) []string {
	start, err := time.Parse("2006-01", from)
	if err != nil {
		return nil
	}
	end, err := time.Parse("2006-01", to)
	if err != nil {
		return nil
	}
	var months []string
	for m := start; !m.After(end); m = m.AddDate(0, 1, 0) {
		months = append(months, m.Format("2006-01"))
	}
	return months
}

// monthRangeToTimes converts a YYYY-MM range into time bounds for querying:
// start is the first day of the from month; end is the first day of the month
// AFTER the to month (exclusive upper bound covering all of the to month).
// Both instants are built in America/New_York so they align exactly with the
// query's NY month bucketing — otherwise worklogs in the midnight-to-dawn UTC
// window at a month edge would leak into the adjacent NY month. If the zone
// fails to load, fall back to UTC.
func monthRangeToTimes(from, to string) (time.Time, time.Time) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		loc = time.UTC
	}
	f, _ := time.Parse("2006-01", from)
	t, _ := time.Parse("2006-01", to)
	start := time.Date(f.Year(), f.Month(), 1, 0, 0, 0, 0, loc)
	end := time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
	return start, end
}
