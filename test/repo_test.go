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
