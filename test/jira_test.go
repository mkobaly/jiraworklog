package test

import (
	"testing"

	"github.com/mkobaly/jiraworklog"
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
	require.Equal(t, 0, issue.Fields.Aggregatetimespent)
	require.Equal(t, 0, issue.Fields.Aggregatetimeoriginalestimate)
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
