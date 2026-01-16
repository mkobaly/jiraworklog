package types

import (
	"database/sql"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
)

type StoredIssue struct {
	ID                            int           `db:"id"`
	Key                           string        `db:"key"`
	ParentId                      sql.NullInt32 `db:"parentId"`
	Type                          string        `db:"type"`
	Summary                       string        `db:"summary"`
	Priority                      string        `db:"priority"`
	Status                        string        `db:"status"`
	Project                       string        `db:"project"`
	ProjectCharge                 string        `db:"projectcharge"`
	FixedVersions                 []string      `db:"fixedversions"`
	CreateDate                    time.Time     `db:"createdate"`
	UpdateDate                    time.Time     `db:"updatedate"`
	ResolvedDate                  sql.NullTime  `db:"resolveddate"`
	DaysToResolve                 int           `db:"daystoresolve"`
	AggregateTimeSpent            int           `db:"aggregatetimespent"`
	AggregateTimeOriginalEstimate int           `db:"aggregatetimeoriginalestimate"`
	RemainingEstimate             int           `db:"remainingestimate"`
	DevAggregateTimeSpent         int           `db:"devaggregatetimespent"`
}

func ToDomain(i jiraworklog.Issue) StoredIssue {
	id, _ := strconv.Atoi(i.ID)
	parentId := sql.NullInt32{}
	if i.Fields.Parent != nil {
		parentId = MustNullInt32(i.Fields.Parent.ID)
	}
	created, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Created)
	updated, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
	daysToResolve := 0
	resolvedDate := sql.NullTime{}
	if i.Fields.ResolutionDate != nil {
		rd, _ := time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
		resolvedDate.Scan(rd)
		daysToResolve = int(math.Ceil(rd.Sub(created).Hours() / 24))
	}
	fixedVersions := []string{}
	if i.Fields.FixVersions != nil {
		for _, v := range i.Fields.FixVersions {
			fixedVersions = append(fixedVersions, v.Name)
		}
	}

	return StoredIssue{
		ID:                            id,
		Key:                           i.Key,
		ParentId:                      parentId,
		Type:                          i.Fields.Issuetype.Name,
		Summary:                       i.Fields.Summary,
		Priority:                      i.Fields.Priority.Name,
		Status:                        i.Fields.Status.Name,
		Project:                       strings.Split(i.Key, "-")[0],
		ProjectCharge:                 i.Fields.ProjectCharge.Value,
		FixedVersions:                 fixedVersions,
		CreateDate:                    created,
		ResolvedDate:                  resolvedDate,
		UpdateDate:                    updated,
		DaysToResolve:                 daysToResolve,
		AggregateTimeSpent:            i.Fields.Timetracking.TimeSpentSeconds,
		AggregateTimeOriginalEstimate: i.Fields.Timetracking.OriginalEstimateSeconds,
		RemainingEstimate:             i.Fields.Timetracking.RemainingEstimateSeconds,
	}
}

func (si *StoredIssue) Merge(i jiraworklog.Issue) {
	daysToResolve := 0
	updated, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Updated)
	resolvedDate := sql.NullTime{}
	if i.Fields.ResolutionDate != nil {
		rd, _ := time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
		resolvedDate.Scan(rd)
		daysToResolve = int(math.Ceil(rd.Sub(si.CreateDate).Hours() / 24))
	}

	if i.Fields.Parent != nil {
		si.ParentId = MustNullInt32(i.Fields.Parent.ID)
	}
	fixedVersions := []string{}
	if i.Fields.FixVersions != nil {
		for _, v := range i.Fields.FixVersions {
			fixedVersions = append(fixedVersions, v.Name)
		}
	}

	si.Type = i.Fields.Issuetype.Name
	si.Summary = i.Fields.Summary
	si.Priority = i.Fields.Priority.Name
	si.Status = i.Fields.Status.Name
	si.Project = strings.Split(i.Key, "-")[0]
	si.ProjectCharge = i.Fields.ProjectCharge.Value
	si.FixedVersions = fixedVersions
	si.ResolvedDate = resolvedDate
	si.UpdateDate = updated
	si.DaysToResolve = daysToResolve
	si.AggregateTimeSpent = i.Fields.Timetracking.TimeSpentSeconds
	si.AggregateTimeOriginalEstimate = i.Fields.Timetracking.OriginalEstimateSeconds
	si.RemainingEstimate = i.Fields.Timetracking.RemainingEstimateSeconds

}

// ParentIssue represents a top level issue that a work log
// was tracked against
type ParentIssue struct {
	ID                            int        `db:"id"`
	Key                           string     `db:"key"`
	Type                          string     `db:"type"`
	Summary                       string     `db:"summary"`
	Priority                      string     `db:"priority"`
	Status                        string     `db:"status"`
	CreateDate                    time.Time  `db:"createDate"`
	UpdateDate                    time.Time  `db:"updateDate"`
	ResolvedDate                  *time.Time `db:"resolvedDate"`
	IsResolved                    bool       `db:"isResolved"`
	DaysToResolve                 int        `db:"daysToResolve"`
	AggregateTimeSpent            int        `db:"aggregateTimeSpent"`
	AggregateTimeOriginalEstimate int        `db:"aggregateTimeOriginalEstimate"`
	Project                       string     `db:"project"`
	Developer                     string     `db:"developer"`
}

// MergeIssue will take an existing parentIssue and merge it with the changes from Jira. This will only happen
// for issues that are not resolved yet.
// func MergeIssue(parentIssue *ParentIssue, i jiraworklog.Issue) {
// 	resolvedDate := time.Time{}
// 	if i.Fields.ResolutionDate != nil {
// 		resolvedDate, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
// 	} else {
// 		resolvedDate, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.StatusCategoryChangeDate)
// 	}
// 	daysToResolve := int(math.Ceil(resolvedDate.Sub(parentIssue.CreateDate).Hours() / 24))

// 	parentIssue.IsResolved = true
// 	parentIssue.ResolvedDate = &resolvedDate
// 	parentIssue.UpdateDate = resolvedDate
// 	parentIssue.DaysToResolve = daysToResolve
// 	parentIssue.Type = i.Fields.Issuetype.Name
// 	parentIssue.Priority = i.Fields.Priority.Name
// 	parentIssue.Status = i.Fields.Status.Name
// 	parentIssue.Summary = i.Fields.Summary
// 	parentIssue.Project = strings.Split(i.Key, "-")[0]
// 	parentIssue.AggregateTimeOriginalEstimate = i.Fields.Aggregatetimeoriginalestimate
// 	parentIssue.AggregateTimeSpent = i.Fields.Aggregatetimespent
// }

type MaitenanceRatio struct {
	YearMonth      string  `db:"year-month"`
	NonRecoverable float32 `db:"NR"`
	AfterMarket    float32 `db:"AM"`
	Project        float32 `db:"PR"`
}

func (mr MaitenanceRatio) Ratio() float32 {
	return mr.Project / (mr.AfterMarket + mr.Project)
}

type People struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
	Role string `db:"role"`
}

func MustNullInt32(s string) sql.NullInt32 {
	if s == "" {
		return sql.NullInt32{}
	}
	v, _ := strconv.Atoi(s)
	return sql.NullInt32{Int32: int32(v), Valid: true}
}
