package types

import (
	"math"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
)

//ParentIssue represents a top level issue that a work log
//was tracked against
type ParentIssue struct {
	ID                            int        `db:"id"`
	Key                           string     `db:"key"`
	Type                          string     `db:"type" boltholdIndex:"Type"`
	Summary                       string     `db:"summary"`
	Priority                      string     `db:"priority" boltholdIndex:"Priority"`
	Status                        string     `db:"status" boltholdIndex:"Status"`
	CreateDate                    time.Time  `db:"createDate" boltholdIndex:"CreateDate"`
	UpdateDate                    time.Time  `db:"updateDate" boltholdIndex:"UpdateDate"`
	ResolvedDate                  *time.Time `db:"resolvedDate" boltholdIndex:"ResolvedDate"`
	IsResolved                    bool       `db:"isResolved" boltholdIndex:"IsResolved"`
	DaysToResolve                 int        `db:"daysToResolve" boltholdIndex:"DaysToResolve"`
	AggregateTimeSpent            int        `db:"aggregateTimeSpent" boltholdIndex:"AggregateTimeSpent"`
	AggregateTimeOriginalEstimate int        `db:"aggregateTimeOriginalEstimate" boltholdIndex:"AggregateTimeOriginalEstimate"`
	Project                       string     `db:"project" boltholdIndex:"Project"`
	Developer                     string     `db:"developer" boltholdIndex:"Developer"`
}

//MergeIssue will take an existing parentIssue and merge it with the changes from Jira. This will only happen
//for issues that are not resolved yet.
func MergeIssue(parentIssue *ParentIssue, i jiraworklog.Issue) {
	resolvedDate := time.Time{}
	if i.Fields.ResolutionDate != nil {
		resolvedDate, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
	} else {
		resolvedDate, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.StatusCategoryChangeDate)
	}
	daysToResolve := int(math.Ceil(resolvedDate.Sub(parentIssue.CreateDate).Hours() / 24))

	parentIssue.IsResolved = true
	parentIssue.ResolvedDate = &resolvedDate
	parentIssue.UpdateDate = resolvedDate
	parentIssue.DaysToResolve = daysToResolve
	parentIssue.Type = i.Fields.Issuetype.Name
	parentIssue.Priority = i.Fields.Priority.Name
	parentIssue.Status = i.Fields.Status.Name
	parentIssue.Summary = i.Fields.Summary
	parentIssue.Project = strings.Split(i.Key, "-")[0]
	parentIssue.AggregateTimeOriginalEstimate = i.Fields.Aggregatetimeoriginalestimate
	parentIssue.AggregateTimeSpent = i.Fields.Aggregatetimespent
}
