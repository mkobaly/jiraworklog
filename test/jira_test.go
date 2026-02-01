package test

import (
	"testing"
	"time"

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

func TestWorklogMapping(t *testing.T) {
	local, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}

	wl := jiraworklog.Worklog{
		ID:               "444",
		IssueID:          "555",
		TimeSpentSeconds: 6000,
		Started:          "2026-01-23T21:11:00.000-0600",
		Author: struct {
			Self       string "json:\"self\""
			AccountID  string "json:\"accountId\""
			AvatarUrls struct {
				Four8X48  string "json:\"48x48\""
				Two4X24   string "json:\"24x24\""
				One6X16   string "json:\"16x16\""
				Three2X32 string "json:\"32x32\""
			} "json:\"avatarUrls\""
			DisplayName string "json:\"displayName\""
			Active      bool   "json:\"active\""
			TimeZone    string "json:\"timeZone\""
			AccountType string "json:\"accountType\""
		}{
			DisplayName: "bobsmith",
		},
	}
	model := types.ToModel(wl, local)
	require.Equal(t, 23, model.Date.Day())
	require.Equal(t, "Friday", model.WeekDay)
}

func TestWorklogStartDateValid(t *testing.T) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		t.Fail()
	}

	local, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}

	jira := jiraworklog.NewJira(cfg)
	ids := []int{304819}
	worklogs, err := jira.WorklogDetails(ids)
	require.NoError(t, err)
	require.Equal(t, 1, len(worklogs))

	// repo, err := repository.NewPostgresRepo(cfg)
	// if err != nil {
	// 	t.Fatal("unable to get postgres repo")
	// }
	for _, w := range worklogs {
		wl := types.ToModel(w, local)
		require.Equal(t, 23, wl.Date.Day())
		require.Equal(t, "Friday", wl.Date.Weekday().String())
	}
}

func TestWorklogFetchAndSave(t *testing.T) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		t.Fail()
	}

	local, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}

	jira := jiraworklog.NewJira(cfg)
	ids := []int{304819}
	worklogs, err := jira.WorklogDetails(ids)
	require.NoError(t, err)
	require.Equal(t, 1, len(worklogs))

	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal("unable to get postgres repo")
	}
	for _, w := range worklogs {
		wl := types.ToModel(w, local)
		err = repo.SaveWorklog(wl)
		if err != nil {
			t.Fatal("unable to save worklog to postgres")
		}
	}
}
