package types

import "time"

//WorklogItem represents the finalized worklog. This includes the issue the worklog
//was logged against along with the issue's parent if one exists
type WorklogItem struct {
	ID                      int
	Author                  string
	Started                 time.Time
	TimeSpentSeconds        int
	OriginalEstimateSeconds int

	IssueID           int
	IssueKey          string
	IssueType         string
	IssueSummary      string
	IssuePriority     string
	IssueStatus       string
	IssueCreateDate   time.Time
	IssueResolvedDate time.Time

	ParentIssueID           *int
	ParentIssueKey          *string
	ParentIssueType         *string
	ParentIssueSummary      *string
	ParentIssuePriority     *string
	ParentIssueStatus       *string
	ParentIssueCreateDate   time.Time
	ParentIssueResolvedDate time.Time

	Project string
}
