package types

import (
	"errors"
	"strings"
)

//IssueChartData represents ussue data that will be charted
type IssueChartData struct {
	GroupBy       string  `db:"groupBy"`
	NonResolved   int     `db:"nonResolved"`
	Resolved      int     `db:"resolved"`
	DaysToResolve int     `db:"daysToResolve"`
	TimeSpent     float64 `db:"timeSpent"`
	TimeEstimate  float64 `db:"timeEstimate"`
}

//IssueAccuracy represents how accurate a developers estimate was vs actual time logged
type IssueAccuracy struct {
	Developer string  `db:"developer"`
	Count     int     `db:"count"`
	Accuracy  float64 `db:"accuracy"`
}

//ErrUnknownGroupBy is error when there is an unknown group by value
var ErrUnknownGroupBy = errors.New("unknown group by value. Valid values are deveoper, type, priority, project, status")

var ErrUnknownWorklogsGroupBy = errors.New("unknown group by value. Valid values are type, priority, project")

//ValidateGroupBy will check the group value is an acceptable value
func ValidateGroupBy(group string) (string, error) {
	groupLower := strings.ToLower(group)
	switch groupLower {
	case "developer":
		return "Developer", nil
	case "type":
		return "Type", nil
	case "priority":
		return "Priority", nil
	case "project":
		return "Project", nil
	case "status":
		return "Status", nil
	default:
		return "", ErrUnknownGroupBy
	}
}

//ValidateWorklogsGroupBy will check the group value is an acceptable value for worklogs
func ValidateWorklogsGroupBy(group string) (string, error) {
	groupLower := strings.ToLower(group)
	switch groupLower {
	case "type":
		return "ParentIssueType", nil
	case "priority":
		return "ParentIssuePriority", nil
	case "project":
		return "Project", nil
	default:
		return "", ErrUnknownWorklogsGroupBy
	}
}
