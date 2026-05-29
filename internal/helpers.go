package internal

import (
	"time"
)

// JiraDateString formats a Unix timestamp for use in a JQL query string.
// Jira evaluates `updated` filters in the user's configured timezone (it
// ignores any explicit offset in the string), so we format in that tz and
// emit at minute precision — Jira's smallest understood granularity.
func JiraDateString(timestamp int64, tz *time.Location) string {
	return time.Unix(timestamp, 0).In(tz).Format("2006-01-02 15:04")
}
