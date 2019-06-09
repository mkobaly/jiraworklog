package test

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test(t *testing.T) {
	cfg := &jiraworklog.Config{
		Jira: jiraworklog.JiraSettings{
			URL:      "https://amagsymmetry.atlassian.net/rest/api/3",
			Username: "michael.kobaly@usa.g4s.com",
			Password: "VFVYMGaOfmMLPfoh2ft7A71F",
		},
	}
	jira := jiraworklog.NewJira(cfg)
	result, err := jira.WorklogDetails([]int{47136, 47137})
	require.NoError(t, err)
	require.Equal(t, 2, len(result))
}
