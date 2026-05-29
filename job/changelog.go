package job

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
)

// fetchIssueChangelogPaged walks the paginated Jira changelog API and returns
// every status-transition row for an issue. Shared by JiraSyncIssuesJob
// (which fetches on first sight of an updated issue) and
// StatusStintsBackfillJob (which fetches on drift detection).
//
// Advances startAt by the actual page size returned, so a changelog with
// total >100 entries is fully walked. The previous in-place implementation
// used startAt = cl.Total + startAt - 1, which on a 106-entry changelog
// jumped from 0 to 105 after the first page and silently skipped entries
// 100-104.
func fetchIssueChangelogPaged(jira jiraworklog.JiraReader, issueId int) ([]types.ChangelogStatus, error) {
	changelog := []types.ChangelogStatus{}
	startAt := 0
	for {
		cl, err := jira.Changelog(issueId, startAt)
		if err != nil {
			return nil, err
		}
		changelog = append(changelog, types.ToChangelogStatus(cl, issueId)...)
		if cl.IsLast {
			break
		}
		advance := len(cl.Values)
		if advance == 0 {
			// Defensive: avoid an infinite loop if Jira ever returns zero
			// items while also reporting isLast=false.
			break
		}
		startAt += advance
	}
	return changelog, nil
}
