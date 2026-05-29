package test

import (
	"testing"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
)

var cnnString = "Server=192.168.0.2;Database=jira_new;User Id=sa;Password=Kobaly!123"

// func Init() *Repository {

// }

func GetTestConfig() (*jiraworklog.Config, error) {
	cfg, err := jiraworklog.LoadConfig("../bin/config.yaml")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func TestMaintenanceRatio(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}
	foo, err := repo.MaitenanceRatio([]string{"dev"})
	require.Greater(t, len(foo), 1)
	if err != nil {
		t.Fatal("Error executing repository.Fetch()", err.Error())
	}
}

func TestNullDates(t *testing.T) {
	//var createDate *mssql.DateTime1

}

func TestInsert(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	worklog := &types.Worklog{
		ID:               -1,
		Author:           "bob.smith",
		TimeSpentSeconds: 456,
		Date:             time.Now(),
		IssueId:          1,
		WeekNumber:       23,
		WeekDay:          "Friday",
		TimeSpentHours:   3,
	}

	repo.SaveWorklog(worklog)
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}

func TestBulkInsertChangelog(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	jira := jiraworklog.NewJira(cfg)
	id := 97039
	changelog, err := jira.Changelog(id, 0)
	require.NoError(t, err)
	cs := types.ToChangelogStatus(changelog, id)

	err = repo.BulkInsertChangelogs(cs)
	if err != nil {
		t.Error("Error bulk inserting changlogs", err.Error())
	}
}

func TestRefreshOfStatus(t *testing.T) {
	cfg, err := GetTestConfig()
	if err != nil {
		t.Fatal()
	}
	repo, err := repository.NewPostgresRepo(cfg)
	if err != nil {
		t.Fatal()
	}

	id := 97039

	err = repo.RefreshStatusStints(id)
	if err != nil {
		t.Error("Error bulk inserting changlogs", err.Error())
	}
}

func TestProjectKPIsSmoke(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	// Pull any real epic key from the DB so the test doesn't depend on a
	// hard-coded key that may not exist.
	var key string
	err = repo.DB.Get(&key, `SELECT key FROM issue WHERE type = 'Epic' LIMIT 1`)
	if err != nil {
		t.Skip("no epic in DB; skipping ProjectKPIs smoke test")
	}

	data, err := repo.ProjectKPIs(key)
	require.NoError(t, err)
	require.GreaterOrEqual(t, data.IssueCount, 0)
}

func TestMonthlyTeamMetricsSkeleton(t *testing.T) {
	cfg, err := GetTestConfig()
	require.NoError(t, err)
	repo, err := repository.NewPostgresRepo(cfg)
	require.NoError(t, err)

	rows, err := repo.MonthlyTeamMetrics([]string{"IDM", "SYM", "ESG"}, "", "")
	require.NoError(t, err)
	// 3 teams * 13 months = 39 rows
	require.Equal(t, 39, len(rows))

	// Each team should appear with 13 distinct year_months
	teamCounts := map[string]int{}
	for _, r := range rows {
		teamCounts[r.Team]++
		require.Regexp(t, `^\d{4}-\d{2}$`, r.YearMonth)
	}
	require.Equal(t, 13, teamCounts["IDM"])
	require.Equal(t, 13, teamCounts["SYM"])
	require.Equal(t, 13, teamCounts["ESG"])
}
