package test

import (
	"fmt"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
	"github.com/timshannon/bolthold"
	"os"
	"testing"
	"time"
)

var worklogItem10 = &types.WorklogItem{
	ID:                  10,
	IssueID:             100,
	IssueKey:            "ABC-2000",
	IssuePriority:       "High",
	IssueStatus:         "Closed",
	Author:              "bob.smith",
	IssueType:           "story",
	Project:             "ABC",
	Date:                time.Date(2019, 1, 5, 0, 0, 0, 0, time.UTC),
	IssueSummary:        "Summary",
	ParentIssueID:       10,
	ParentIssueKey:      "ABC-2000",
	ParentIssuePriority: "High",
	ParentIssueStatus:   "Closed",
	TimeSpentSeconds:    3600,
	TimeSpentHours:      1,
}

var parentIssue1 = &types.ParentIssue{
	ID:           1,
	Priority:     "High",
	Key:          "ABC-2000",
	Developer:    "bob.smith",
	Status:       "Closed",
	Project:      "ABC",
	ResolvedDate: &resolvedDate,
}
var resolvedDate = time.Date(2019, 1, 5, 0, 0, 0, 0, time.UTC)

func TestNewBoldDBRepo(t *testing.T) {
	w, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	w.Close()
	os.Remove("my.db")
}

func TestWriteHasNoError(t *testing.T) {
	r, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	err = r.Write(worklogItem10, parentIssue1)
	require.NoError(t, err)
	r.Close()
	os.Remove("my.db")
}

func TestNonResolvedIssuesNotFound(t *testing.T) {
	w, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	keys, err := w.NonResolvedIssues()
	require.NoError(t, err)
	require.Equal(t, 0, len(keys))
	w.Close()
	os.Remove("my.db")
}

func TestNonResolvedIssues(t *testing.T) {
	w, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	w.Write(worklogItem10, parentIssue1)
	keys, err := w.NonResolvedIssues()
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	require.Equal(t, "ABC-2000", keys[0].Key)
	w.Close()
	os.Remove("my.db")
}

func TestIssueGroupByDeveloper(t *testing.T) {
	r, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	defer r.Close()
	defer os.Remove("my.db")

	//Insert 9 parent Issues
	issues := getParentIssues()
	for _, i := range issues {
		r.Write(worklogItem10, &i)
	}

	results, err := r.IssuesGroupedBy("Developer", 100)
	require.Equal(t, 2, len(results))
}

func TestIssueGroupByType(t *testing.T) {
	r, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	defer r.Close()
	defer os.Remove("my.db")
	//Insert 9 parent Issues
	issues := getParentIssues()
	for _, i := range issues {
		r.Write(worklogItem10, &i)
	}

	results, err := r.IssuesGroupedBy("Type", 100)
	require.Equal(t, 3, len(results))
}

func TestIssueGroupByPriority(t *testing.T) {
	r, err := repository.NewBoltDBRepo("my.db")
	require.NoError(t, err)
	defer r.Close()
	defer os.Remove("my.db")

	//Insert 9 parent Issues
	issues := getParentIssues()
	for _, i := range issues {
		r.Write(worklogItem10, &i)
	}

	results, err := r.IssuesGroupedBy("Priority", 100)
	require.Equal(t, 4, len(results))
}

func getParentIssues() []types.ParentIssue {
	resolvedDate := time.Now().Add(-2 * time.Hour)
	createDate := time.Now().Add(-36 * time.Hour)

	result := []types.ParentIssue{}
	result = append(result, types.ParentIssue{
		ID:                            1,
		Priority:                      "High",
		Type:                          "Story",
		Key:                           "ABC-2000",
		Developer:                     "bob.smith",
		Status:                        "Closed",
		Project:                       "ABC",
		AggregateTimeSpent:            3600,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 2,
		IsResolved:                    true,
		CreateDate:                    createDate,
		ResolvedDate:                  &resolvedDate,
	})
	result = append(result, types.ParentIssue{
		ID:                            2,
		Priority:                      "Low",
		Type:                          "Task",
		Key:                           "ABC-2001",
		Developer:                     "mary.jane",
		Status:                        "Closed",
		Project:                       "ABC",
		AggregateTimeSpent:            1800,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 1,
		IsResolved:                    true,
		CreateDate:                    createDate,
		ResolvedDate:                  &resolvedDate,
	})
	result = append(result, types.ParentIssue{
		ID:                            3,
		Priority:                      "Urgent",
		Type:                          "Bug",
		Key:                           "ABC-2003",
		Developer:                     "mary.jane",
		Status:                        "In-Development",
		Project:                       "ABC",
		AggregateTimeSpent:            1800,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 0,
		IsResolved:                    false,
		CreateDate:                    createDate,
		ResolvedDate:                  &time.Time{},
	})
	result = append(result, types.ParentIssue{
		ID:                            4,
		Priority:                      "Low",
		Type:                          "Story",
		Key:                           "ABC-2004",
		Developer:                     "mary.jane",
		Status:                        "Open",
		Project:                       "ABC",
		AggregateTimeSpent:            7200,
		AggregateTimeOriginalEstimate: 7200,
		DaysToResolve:                 0,
		IsResolved:                    false,
		CreateDate:                    createDate,
		ResolvedDate:                  &time.Time{},
	})
	result = append(result, types.ParentIssue{
		ID:                            5,
		Priority:                      "High",
		Type:                          "Bug",
		Key:                           "ABC-2005",
		Developer:                     "mary.jane",
		Status:                        "Closed",
		Project:                       "ABC",
		AggregateTimeSpent:            1800,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 1,
		IsResolved:                    true,
		CreateDate:                    createDate,
		ResolvedDate:                  &resolvedDate,
	})
	result = append(result, types.ParentIssue{
		ID:                            6,
		Priority:                      "Medium",
		Type:                          "Task",
		Key:                           "ABC-2006",
		Developer:                     "bob.smith",
		Status:                        "Open",
		Project:                       "ABC",
		AggregateTimeSpent:            1800,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 0,
		IsResolved:                    false,
		CreateDate:                    createDate,
		ResolvedDate:                  &time.Time{},
	})
	result = append(result, types.ParentIssue{
		ID:                            7,
		Priority:                      "High",
		Type:                          "Story",
		Key:                           "ABC-2007",
		Developer:                     "bob.smith",
		Status:                        "Closed",
		Project:                       "ABC",
		AggregateTimeSpent:            7200,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 3,
		IsResolved:                    true,
		CreateDate:                    createDate,
		ResolvedDate:                  &resolvedDate,
	})
	result = append(result, types.ParentIssue{
		ID:                            8,
		Priority:                      "High",
		Type:                          "Story",
		Key:                           "ABC-2008",
		Developer:                     "mary.jane",
		Status:                        "In-Deveopment",
		Project:                       "ABC",
		AggregateTimeSpent:            1800,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 0,
		IsResolved:                    false,
		CreateDate:                    createDate,
		ResolvedDate:                  &time.Time{},
	})
	result = append(result, types.ParentIssue{
		ID:                            9,
		Priority:                      "Low",
		Type:                          "Task",
		Key:                           "ABC-2009",
		Developer:                     "mary.jane",
		Status:                        "Closed",
		Project:                       "ABC",
		AggregateTimeSpent:            3600,
		AggregateTimeOriginalEstimate: 3600,
		DaysToResolve:                 1,
		IsResolved:                    true,
		CreateDate:                    createDate,
		ResolvedDate:                  &resolvedDate,
	})
	return result
}

func getWorklogs() []types.WorklogItem {

	result := []types.WorklogItem{}
	result = append(result, types.WorklogItem{
		ID:            10,
		IssueID:       100,
		IssueKey:      "ABC-2000",
		IssuePriority: "High",
		IssueStatus:   "Closed",
		Author:        "bob.smith",
		IssueType:     "story",
		Project:       "ABC",
	})
	result = append(result, types.WorklogItem{
		ID:            11,
		IssueID:       101,
		IssueKey:      "ABC-2001",
		IssuePriority: "Low",
		IssueStatus:   "Open",
		Author:        "bob.smith",
		IssueType:     "bug",
		Project:       "ABC",
	})
	result = append(result, types.WorklogItem{
		ID:            12,
		IssueID:       102,
		IssueKey:      "DEF-2002",
		IssuePriority: "High",
		IssueStatus:   "Closed",
		Author:        "mary.jane",
		IssueType:     "story",
		Project:       "DEF",
	})
	result = append(result, types.WorklogItem{
		ID:            13,
		IssueID:       103,
		IssueKey:      "DEF-2003",
		IssuePriority: "High",
		IssueStatus:   "Closed",
		Author:        "bob.smith",
		IssueType:     "story",
		Project:       "DEF",
	})
	return result
}
