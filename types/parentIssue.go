package types

import "time"

//ParentIssue represents a top level issue that a work log
//was tracked against
type ParentIssue struct {
	ID           int        `storm:"id" db:"id"`
	Key          string     `storm:"index" storm:"unique" db:"key"`
	Type         string     `storm:"index" db:"type"`
	Summary      string     `db:"summary"`
	Priority     string     `storm:"index" db:"priority"`
	Status       string     `storm:"index" db:"status"`
	CreateDate   time.Time  `storm:"index" db:"createDate"`
	ResolvedDate *time.Time `db:"resolvedDate"`
	IsResolved   bool       `storm:"index" db:"isResolved"`
	Project      string     `storm:"index" db:"project"`
}
