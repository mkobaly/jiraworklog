package types

import (
	"math"
	"strconv"
	"time"

	"github.com/mkobaly/jiraworklog"
)

// WorklogItem represents the finalized worklog. This includes the issue the worklog
// was logged against along with the issue's parent if one exists
type Worklog struct {
	ID               int       `db:"id" `
	Author           string    `db:"author"`
	Date             time.Time `db:"date"`
	WeekNumber       int       `db:"weekNumber"`
	WeekDay          string    `db:"weekDay"`
	TimeSpentSeconds int       `db:"timeSpentSeconds"`
	TimeSpentHours   float64   `db:"timeSpentHours"`
	IssueId          int       `db:"issueid"`
}

// WorklogItem represents the finalized worklog. This includes the issue the worklog
// was logged against along with the issue's parent if one exists
type WorklogItem struct {
	ID               int       `db:"id" `
	Author           string    `db:"author"`
	Date             time.Time `db:"date"`
	WeekNumber       int       `db:"weekNumber"`
	WeekDay          string    `db:"weekDay"`
	TimeSpentSeconds int       `db:"timeSpentSeconds"`
	TimeSpentHours   float64   `db:"timeSpentHours"`

	Project string `db:"project"`

	IssueID            int    `db:"issueId"`
	IssueKey           string `db:"issueKey"`
	IssueType          string `db:"issueType"`
	IssueSummary       string `db:"issueSummary"`
	IssuePriority      string `db:"issuePriority"`
	IssueStatus        string `db:"issueStatus"`
	IssueProjectCharge string `db:"issueProjectCharge"`

	ParentIssueID            int    `db:"parentIssueId"`
	ParentIssueKey           string `db:"parentIssueKey"`
	ParentIssueType          string `db:"parentIssueType"`
	ParentIssueSummary       string `db:"parentIssueSummary"`
	ParentIssuePriority      string `db:"parentIssuePriority"`
	ParentIssueStatus        string `db:"parentIssueStatus"`
	ParentIssueProjectCharge string `db:"parentIssueProjectCharge"`
}

// ConvertToModels will take rest api results from JIRA and transform them into a model we can work with
func ToModel(w jiraworklog.Worklog) *Worklog {
	id, _ := strconv.Atoi(w.ID)
	started, _ := time.Parse("2006-01-02T15:04:05.000-0700", w.Started)
	_, week := started.ISOWeek()
	timespent := float64(w.TimeSpentSeconds) / 3600

	issueId, _ := strconv.Atoi(w.IssueID)

	worklogResult := &Worklog{
		ID:               id,
		Author:           w.Author.DisplayName,
		IssueId:          issueId,
		Date:             started,
		TimeSpentSeconds: w.TimeSpentSeconds,
		TimeSpentHours:   math.Round(timespent*100) / 100,
		WeekNumber:       week,
		WeekDay:          started.Weekday().String(),
	}
	return worklogResult
}

// ConvertToModels will take rest api results from JIRA and transform them into a model we can work with
// func ConvertToModels(w jiraworklog.Worklog, i jiraworklog.Issue, parentIssue jiraworklog.Issue) (*WorklogItem, *ParentIssue) {
// 	id, _ := strconv.Atoi(w.ID)
// 	issueID, _ := strconv.Atoi(i.ID)
// 	started, _ := time.Parse("2006-01-02T15:04:05.000-0700", w.Started)
// 	_, week := started.ISOWeek()
// 	timespent := float64(w.TimeSpentSeconds) / 3600

// 	worklogResult := &WorklogItem{
// 		ID:                 id,
// 		Author:             w.Author.DisplayName,
// 		Project:            strings.Split(i.Key, "-")[0],
// 		IssueID:            issueID,
// 		IssueKey:           i.Key,
// 		IssuePriority:      i.Fields.Priority.Name,
// 		IssueType:          i.Fields.Issuetype.Name,
// 		IssueSummary:       i.Fields.Summary,
// 		IssueStatus:        i.Fields.Status.Name,
// 		IssueProjectCharge: i.Fields.ProjectCharge.Value,
// 		//Setting parent to issue and if issue has parent override below. This way its always populated
// 		ParentIssueID:            issueID,
// 		ParentIssueKey:           i.Key,
// 		ParentIssueType:          i.Fields.Issuetype.Name,
// 		ParentIssuePriority:      i.Fields.Priority.Name,
// 		ParentIssueSummary:       i.Fields.Summary,
// 		ParentIssueStatus:        i.Fields.Status.Name,
// 		ParentIssueProjectCharge: i.Fields.ProjectCharge.Value,
// 		Date:                     started,
// 		TimeSpentSeconds:         w.TimeSpentSeconds,
// 		TimeSpentHours:           math.Round(timespent*100) / 100,
// 		WeekNumber:               week,
// 		WeekDay:                  started.Weekday().String(),
// 	}

// 	if i.HasParent() {
// 		parentID, _ := strconv.Atoi(parentIssue.ID)
// 		worklogResult.ParentIssueID = parentID
// 		worklogResult.ParentIssueKey = parentIssue.Key
// 		worklogResult.ParentIssueType = parentIssue.Fields.Issuetype.Name
// 		worklogResult.ParentIssuePriority = parentIssue.Fields.Priority.Name
// 		worklogResult.ParentIssueSummary = parentIssue.Fields.Summary
// 		worklogResult.ParentIssueStatus = parentIssue.Fields.Status.Name
// 		worklogResult.ParentIssueProjectCharge = parentIssue.Fields.ProjectCharge.Value
// 	}

// 	pi := i
// 	if i.HasParent() {
// 		pi = parentIssue
// 	}

// 	created, _ := time.Parse("2006-01-02T15:04:05.000-0700", pi.Fields.Created)
// 	resolvedDate := time.Time{}
// 	isResolved := false
// 	daysToResolve := 0
// 	if pi.Fields.ResolutionDate != nil {
// 		resolvedDate, _ = time.Parse("2006-01-02T15:04:05.000-0700", *pi.Fields.ResolutionDate)
// 		isResolved = true
// 		daysToResolve = int(math.Ceil(resolvedDate.Sub(created).Hours() / 24))
// 	}

// 	issueResult := &ParentIssue{
// 		ID:                            worklogResult.ParentIssueID,
// 		Key:                           worklogResult.ParentIssueKey,
// 		Type:                          pi.Fields.Issuetype.Name,
// 		Summary:                       pi.Fields.Summary,
// 		Priority:                      pi.Fields.Priority.Name,
// 		Status:                        pi.Fields.Status.Name,
// 		CreateDate:                    created,
// 		UpdateDate:                    started, //always update issue's updateDate when a worklog item is entered
// 		ResolvedDate:                  &resolvedDate,
// 		IsResolved:                    isResolved,
// 		DaysToResolve:                 daysToResolve,
// 		AggregateTimeOriginalEstimate: pi.Fields.Aggregatetimeoriginalestimate,
// 		AggregateTimeSpent:            pi.Fields.Aggregatetimespent,
// 		Project:                       strings.Split(pi.Key, "-")[0],
// 		Developer:                     worklogResult.Author,
// 	}

// 	return worklogResult, issueResult
// }
