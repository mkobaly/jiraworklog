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

//ErrUnknownGroupBy is error when there is an unknown group by value
var ErrUnknownGroupBy = errors.New("unknown group by value. Valid values are deveoper, type, priority, project, status")

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
