package test

import (
	"log"
	"testing"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/stretchr/testify/require"
)

var cnnString = "Server=192.168.0.2;Database=jira_new;User Id=sa;Password=Kobaly!123"

// func Init() *Repository {

// }

func TestFetch(t *testing.T) {
	db, err := sqlx.Connect("sqlserver", cnnString)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	repo := &repository.SQL{DB: db}
	_, err = repo.NonResolvedIssues()
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}

func TestMaintenanceRatio(t *testing.T) {
	db, err := sqlx.Connect("sqlserver", cnnString)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	repo := &repository.SQL{DB: db}
	foo, err := repo.MaitenanceRatio()
	require.Greater(t, len(foo), 1)
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}

func TestNullDates(t *testing.T) {
	//var createDate *mssql.DateTime1

}

func TestInsert(t *testing.T) {
	db, err := sqlx.Connect("sqlserver", cnnString)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	repo := &repository.SQL{DB: db}

	worklog := &types.WorklogItem{
		ID:               1000,
		Author:           "bob.smith",
		TimeSpentSeconds: 456,
		Date:             time.Now(),
		IssueID:          444,
		IssueKey:         "ABC-123",
		IssuePriority:    "high",
		IssueType:        "story",
		IssueSummary:     "Test ticket",
	}

	repo.Write(worklog, nil)
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}
