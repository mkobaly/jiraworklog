package types

import (
	"github.com/mkobaly/jiraworklog"
	"strconv"
	"strings"
	"time"
)

//NewWorklogItem will create a new worklogItem from jira objects
func NewWorklogItem(w jiraworklog.Worklog, i jiraworklog.Issue, parentIssue jiraworklog.Issue) WorklogItem {
	id, _ := strconv.Atoi(w.ID)
	issueID, _ := strconv.Atoi(i.ID)
	started, _ := time.Parse("2006-01-02T15:04:05.000-0700", w.Started)
	created, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Created)
	resolved := time.Time{}
	if i.Fields.ResolutionDate != nil {
		resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
	}

	wi := WorklogItem{
		ID:                id,
		Author:            w.Author.Key,
		IssueID:           issueID,
		IssueKey:          i.Key,
		IssuePriority:     i.Fields.Priority.Name,
		IssueType:         i.Fields.Issuetype.Name,
		IssueSummary:      i.Fields.Summary,
		IssueStatus:       i.Fields.Status.Name,
		IssueCreateDate:   created,
		IssueResolvedDate: resolved,

		TimeSpentSeconds:        w.TimeSpentSeconds,
		OriginalEstimateSeconds: i.Fields.Timeoriginalestimate,
		Started:                 started,
		Project:                 strings.Split(i.Key, "-")[0],
	}

	if i.HasParent() {

		created, _ := time.Parse("2006-01-02T15:04:05.000-0700", parentIssue.Fields.Created)
		resolved := time.Time{}
		if parentIssue.Fields.ResolutionDate != nil {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *parentIssue.Fields.ResolutionDate)
		}

		parentID, _ := strconv.Atoi(i.Fields.Parent.ID)
		wi.ParentIssueID = &parentID
		wi.ParentIssueKey = &i.Fields.Parent.Key
		wi.ParentIssueType = &parentIssue.Fields.Issuetype.Name
		wi.ParentIssuePriority = &parentIssue.Fields.Priority.Name
		wi.ParentIssueSummary = &parentIssue.Fields.Summary
		wi.ParentIssueStatus = &parentIssue.Fields.Status.Name
		wi.ParentIssueCreateDate = created
		wi.ParentIssueResolvedDate = resolved
	}
	return wi
}

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

func (w WorklogItem) GetParent() *ParentIssue {
	if w.ParentIssueID != nil {
		return &ParentIssue{
			ID:           *w.ParentIssueID,
			Key:          *w.ParentIssueKey,
			Type:         *w.ParentIssueType,
			Priority:     *w.ParentIssuePriority,
			Status:       *w.ParentIssueStatus,
			Summary:      *w.ParentIssueSummary,
			CreateDate:   w.ParentIssueCreateDate,
			ResolvedDate: w.ParentIssueResolvedDate,
			IsResolved:   !w.ParentIssueResolvedDate.IsZero(),
		}
	}
	return &ParentIssue{
		ID:           w.IssueID,
		Key:          w.IssueKey,
		Type:         w.IssueType,
		Priority:     w.IssuePriority,
		Status:       w.IssueStatus,
		Summary:      w.IssueSummary,
		CreateDate:   w.IssueCreateDate,
		ResolvedDate: w.IssueResolvedDate,
		IsResolved:   !w.IssueResolvedDate.IsZero(),
	}
}
