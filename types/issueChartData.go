package types

import (
	"errors"
	"strings"
)

//IssueChartData represents ussue data that will be charted
type IssueChartData struct {
	GroupBy        string
	NonResolved    int
	Resolved       int
	DaysToComplete int
	TimeSpent      float64
	TimeEstimate   float64
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
