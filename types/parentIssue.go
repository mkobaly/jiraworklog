package types

import (
	"database/sql"
	"database/sql/driver"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
)

// StringArray is a custom type for PostgreSQL text[] arrays
type StringArray []string

// Scan implements the sql.Scanner interface for reading PostgreSQL arrays
func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		*a = nil
		return nil
	}

	// Handle empty array
	if str == "{}" || str == "" {
		*a = []string{}
		return nil
	}

	// Remove the curly braces
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")

	// Parse the array elements
	var result []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for _, r := range str {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		switch r {
		case '\\':
			escaped = true
		case '"':
			inQuotes = !inQuotes
		case ',':
			if inQuotes {
				current.WriteRune(r)
			} else {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	// Don't forget the last element
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	*a = result
	return nil
}

// Value implements the driver.Valuer interface for writing PostgreSQL arrays
func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}

	var escaped []string
	for _, s := range a {
		// Escape quotes and backslashes
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		// Quote strings that contain special characters
		if strings.ContainsAny(s, `,{}"\\`) || s == "" {
			s = `"` + s + `"`
		}
		escaped = append(escaped, s)
	}

	return "{" + strings.Join(escaped, ",") + "}", nil
}

type StoredIssue struct {
	ID                int           `db:"id"`
	Key               string        `db:"key"`
	ParentId          sql.NullInt32 `db:"parentId"`
	Type              string        `db:"type"`
	Summary           string        `db:"summary"`
	Priority          string        `db:"priority"`
	Status            string        `db:"status"`
	Project           string        `db:"project"`
	ProjectCharge     string        `db:"projectcharge"`
	FixedVersions     []string      `db:"fixedversions"`
	CreateDate        time.Time     `db:"createdate"`
	UpdateDate        time.Time     `db:"updatedate"`
	ResolvedDate      sql.NullTime  `db:"resolveddate"`
	DaysToResolve     int           `db:"daystoresolve"`
	TimeSpent         int           `db:"timespent"`
	OriginalEstimate  int           `db:"originalestimate"`
	RemainingEstimate int           `db:"remainingestimate"`
	DevTimeSpent      int           `db:"devtimespent"`
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
		ID:                id,
		Key:               i.Key,
		ParentId:          parentId,
		Type:              i.Fields.Issuetype.Name,
		Summary:           i.Fields.Summary,
		Priority:          i.Fields.Priority.Name,
		Status:            i.Fields.Status.Name,
		Project:           strings.Split(i.Key, "-")[0],
		ProjectCharge:     i.Fields.ProjectCharge.Value,
		FixedVersions:     fixedVersions,
		CreateDate:        created,
		ResolvedDate:      resolvedDate,
		UpdateDate:        updated,
		DaysToResolve:     daysToResolve,
		TimeSpent:         i.Fields.Timetracking.TimeSpentSeconds,
		OriginalEstimate:  i.Fields.Timetracking.OriginalEstimateSeconds,
		RemainingEstimate: i.Fields.Timetracking.RemainingEstimateSeconds,
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
	si.TimeSpent = i.Fields.Timetracking.TimeSpentSeconds
	si.OriginalEstimate = i.Fields.Timetracking.OriginalEstimateSeconds
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
	YearMonth      string  `db:"year_month"`
	NonRecoverable float32 `db:"nr"`
	AfterMarket    float32 `db:"am"`
	Project        float32 `db:"pr"`
	Other          float32 `db:"other"`
}

func (mr MaitenanceRatio) Ratio() float32 {
	return mr.Project / (mr.AfterMarket + mr.Project)
}

type People struct {
	Id         int            `db:"id"`
	Name       string         `db:"name"`
	IsEmployee bool           `db:"isemployee"`
	Role       string         `db:"role"`
	Location   sql.NullString `db:"location"`
}

func (p People) GetLocation() string {
	if p.Location.Valid {
		return p.Location.String
	}
	return ""
}

// IssueMissingCharge represents an issue that has no project charge assigned
type IssueMissingCharge struct {
	Project    string    `db:"project"`
	Key        string    `db:"key"`
	Type       string    `db:"type"`
	Summary    string    `db:"summary"`
	Priority   string    `db:"priority"`
	Status     string    `db:"status"`
	UpdateDate time.Time `db:"updatedate"`
}

// CustomerBugCount represents customer bug counts grouped by project, priority, and month
type CustomerBugCount struct {
	Project   string `db:"project"`
	Priority  string `db:"priority"`
	YearMonth string `db:"year_month"`
	Count     int    `db:"count"`
}

// Project represents a project with associated project charges
type Project struct {
	Id             int         `db:"id"`
	Name           string      `db:"name"`
	Visible        bool        `db:"visible"`
	ProjectCharges StringArray `db:"projectcharge"`
}

// ProjectChargeHours represents hours worked per project charge and role
type ProjectChargeHours struct {
	Project       string         `db:"project"`
	ProjectCharge string         `db:"projectcharge"`
	YearMonth     string         `db:"yearmonth"`
	IsEmployee    bool           `db:"isemployee"`
	Role          sql.NullString `db:"role"`
	Location      sql.NullString `db:"location"`
	Hours         float64        `db:"hours"`
}

// DailyHours represents hours worked by an author on a specific day
type DailyHours struct {
	Role           string  `db:"role"`
	Author         string  `db:"author"`
	Date           string  `db:"date"`
	NonRecoverable float64 `db:"nonrecoverable"`
	AfterMarket    float64 `db:"aftermarket"`
	Project        float64 `db:"project"`
	Missing        float64 `db:"missing"`
}

// Total returns the total hours for the day
func (d DailyHours) Total() float64 {
	return d.NonRecoverable + d.AfterMarket + d.Project + d.Missing
}

// ProductiveRatio returns the ratio of project work vs (project + aftermarket)
func (d DailyHours) ProductiveRatio() float64 {
	if d.AfterMarket+d.Project == 0 {
		return 0
	}
	return d.Project / (d.AfterMarket + d.Project)
}

// WeekOption represents a week choice for dropdown selection
type WeekOption struct {
	Offset int
	Label  string
	Start  time.Time
	End    time.Time
}

// IssueMismatchedCharge represents an issue where the project charge differs from its parent
type IssueMismatchedCharge struct {
	ParentKey           string `db:"parentkey"`
	ParentType          string `db:"parenttype"`
	ParentProjectCharge string `db:"parentprojectcharge"`
	ParentSummary       string `db:"parentsummary"`
	Key                 string `db:"key"`
	Type                string `db:"type"`
	ProjectCharge       string `db:"projectcharge"`
	Summary             string `db:"summary"`
}

// ProjectTimeTracking represents time tracking data for issues in a project/version
type ProjectTimeTracking struct {
	ID              int            `db:"id"`
	ParentID        sql.NullInt32  `db:"parentid"`
	Key             string         `db:"key"`
	Type            string         `db:"type"`
	Summary         string         `db:"summary"`
	Status          string         `db:"status"`
	ProjectCharge   string         `db:"projectcharge"`
	EstimateSeconds sql.NullInt32  `db:"estimateseconds"`
	Estimate        sql.NullString `db:"estimate"`
	DevSeconds      sql.NullInt32  `db:"devseconds"`
	DevTimeSpent    sql.NullString `db:"devtimespent"`
}

// Progress returns the percentage of time spent vs estimate (0-100+)
func (p ProjectTimeTracking) Progress() float64 {
	if !p.EstimateSeconds.Valid || p.EstimateSeconds.Int32 == 0 {
		return 0
	}
	devSecs := int32(0)
	if p.DevSeconds.Valid {
		devSecs = p.DevSeconds.Int32
	}
	return float64(devSecs) / float64(p.EstimateSeconds.Int32) * 100
}

func (p ProjectTimeTracking) StillWithDev() bool {
	switch p.Status {
	case "In Progress":
		return true
	case "To Do":
		return true
	case "Development Backlog":
		return true
	case "In Development":
		return true
	case "Code Complete":
		return true
	case "On Hold":
		return true
	case "Bug Draft":
		return true
	case "Backlog":
		return true
	default:
		return false
	}
}

// IsOverBudget returns true if time spent exceeds estimate
func (p ProjectTimeTracking) IsOverBudget() bool {
	return p.Progress() > 100
}

func MustNullInt32(s string) sql.NullInt32 {
	if s == "" {
		return sql.NullInt32{}
	}
	v, _ := strconv.Atoi(s)
	return sql.NullInt32{Int32: int32(v), Valid: true}
}
