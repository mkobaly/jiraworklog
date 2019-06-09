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
		ID:     id,
		Author: w.Author.Key,

		IssueID:       issueID,
		IssueKey:      i.Key,
		IssuePriority: i.Fields.Priority.Name,
		IssueType:     i.Fields.Issuetype.Name,
		IssueSummary:  i.Fields.Summary,
		IssueStatus:   i.Fields.Status.Name,
		//IssueCreateDate:   created,
		//IssueResolvedDate: resolved,

		//Setting parent to issue and if issue has parent override below. This way its always populated
		ParentIssueID:           issueID,
		ParentIssueKey:          i.Key,
		ParentIssueType:         i.Fields.Issuetype.Name,
		ParentIssuePriority:     i.Fields.Priority.Name,
		ParentIssueSummary:      i.Fields.Summary,
		ParentIssueStatus:       i.Fields.Status.Name,
		parentIssueCreateDate:   created,
		parentIssueResolvedDate: resolved,

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
		wi.ParentIssueID = parentID
		wi.ParentIssueKey = i.Fields.Parent.Key
		wi.ParentIssueType = parentIssue.Fields.Issuetype.Name
		wi.ParentIssuePriority = parentIssue.Fields.Priority.Name
		wi.ParentIssueSummary = parentIssue.Fields.Summary
		wi.ParentIssueStatus = parentIssue.Fields.Status.Name
		wi.parentIssueCreateDate = created
		wi.parentIssueResolvedDate = resolved
	}
	return wi
}

//WorklogItem represents the finalized worklog. This includes the issue the worklog
//was logged against along with the issue's parent if one exists
type WorklogItem struct {
	ID                      int    `storm:"id"`
	Author                  string `storm:"index"`
	Started                 time.Time
	TimeSpentSeconds        int
	OriginalEstimateSeconds int

	IssueID       int
	IssueKey      string
	IssueType     string
	IssueSummary  string
	IssuePriority string
	IssueStatus   string
	//IssueCreateDate   time.Time
	//IssueResolvedDate time.Time

	ParentIssueID       int
	ParentIssueKey      string
	ParentIssueType     string
	ParentIssueSummary  string
	ParentIssuePriority string
	ParentIssueStatus   string

	//only want these to hydrate parent issue later
	parentIssueCreateDate   time.Time
	parentIssueResolvedDate time.Time

	Project string
}

func (w WorklogItem) GetParent() *ParentIssue {
	return &ParentIssue{
		ID:           w.ParentIssueID,
		Key:          w.ParentIssueKey,
		Type:         w.ParentIssueType,
		Priority:     w.ParentIssuePriority,
		Status:       w.ParentIssueStatus,
		Summary:      w.ParentIssueSummary,
		Project:      w.Project,
		CreateDate:   w.parentIssueCreateDate,
		ResolvedDate: &w.parentIssueResolvedDate,
		IsResolved:   !w.parentIssueResolvedDate.IsZero(),
	}
}
