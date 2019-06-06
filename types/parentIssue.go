package types

import "time"

//ParentIssue represents a top level issue that a work log
//was tracked against
type ParentIssue struct {
	ID           int
	Key          string `storm:"index"`
	Type         string
	Summary      string
	Priority     string
	Status       string
	CreateDate   time.Time
	ResolvedDate time.Time
	IsResolved   bool `storm:"index"`
	Project      string
}
