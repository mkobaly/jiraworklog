package test

import (
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
)

func TestWorklogsUpdated(t *testing.T) {
	fj := &FakeJira{}
	wl, err := fj.WorklogsUpdated(12121212)
	require.NoError(t, err)
	require.Equal(t, 3, len(wl.Values))
}

func TestGetIssueWihoutAggTimesDefaultsToZero(t *testing.T) {
	fj := &FakeJira{}
	issue, err := fj.Issue("6003")
	require.NoError(t, err)
	require.Equal(t, 0, issue.Fields.Timetracking.TimeSpentSeconds)
	require.Equal(t, 0, issue.Fields.Timetracking.OriginalEstimateSeconds)
}

func TestBulkIssueFetch(t *testing.T) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		t.Fail()
	}
	jira := jiraworklog.NewJira(cfg)
	keys := []string{"IDM-2501", "IDM-2694"}
	issues, err := jira.BulkFetchIssues(keys)
	require.NoError(t, err)
	require.Equal(t, 2, len(issues))
}

func TestBulkIssueFetchSaveToDB(t *testing.T) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		t.Fail()
	}
	jira := jiraworklog.NewJira(cfg)
	keys := []string{"SYM-6763"}
	issues, err := jira.BulkFetchIssues(keys)
	require.NoError(t, err)
	require.Equal(t, 1, len(issues))

	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal("unable to get postgres repo")
	}
	for _, i := range issues {
		issue := types.ToDomain(i)
		err := repo.UpdateIssue(&issue)
		if err != nil {
			t.Fatal("error updating issue")
		}
	}
}
