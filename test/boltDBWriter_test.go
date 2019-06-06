package test

import (
	"github.com/mkobaly/jiraworklog/types"
	"github.com/mkobaly/jiraworklog/writers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestNewBoldDBWriter(t *testing.T) {
	w, err := writers.NewBoltDBWriter("my.db")
	require.NoError(t, err)
	w.Close()
	os.Remove("my.db")
}

func TestWrite(t *testing.T) {
	w, err := writers.NewBoltDBWriter("my.db")
	require.NoError(t, err)
	err = w.Write(types.WorklogItem{
		ID:                      12,
		IssueID:                 1234,
		IssueKey:                "ABC-1234",
		IssuePriority:           "High",
		OriginalEstimateSeconds: 1000,
		IssueStatus:             "Closed",
	})
	require.NoError(t, err)
	w.Close()
	os.Remove("my.db")
}

func TestNonResolvedIssues(t *testing.T) {
	w, err := writers.NewBoltDBWriter("my.db")
	require.NoError(t, err)
	w.Write(types.WorklogItem{
		ID:                      12,
		IssueID:                 1234,
		IssueKey:                "ABC-1234",
		IssuePriority:           "High",
		OriginalEstimateSeconds: 1000,
		IssueStatus:             "Closed",
	})
	keys, err := w.NonResolvedIssues()
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	require.Equal(t, "ABC-1234", keys[0])
	w.Close()
	os.Remove("my.db")
}
