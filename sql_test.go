package jiraworklog

import (
	"log"
	"testing"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/mkobaly/jiraworklog/writers"
)

var cnnString = "Server=localhost;Database=Jira;User Id=sa;Password=Passw0rd1"

// func Init() *Repository {

// }

func TestFetch(t *testing.T) {
	db, err := sqlx.Connect("sqlserver", cnnString)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()
	repo := &writers.Repository{DB: db}
	_, err = repo.Fetch()
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
	repo := &writers.Repository{DB: db}

	worklog := types.WorklogItem{
		ID:               1000,
		Author:           "bob.smith",
		TimeSpentSeconds: 456,
		Started:          time.Now(),
		IssueID:          444,
		IssueKey:         "ABC-123",
		IssuePriority:    "high",
		IssueType:        "story",
		IssueSummary:     "Test ticket",
	}

	repo.Insert(worklog)
	if err != nil {
		t.Error("Error executing repository.Fetch()", err.Error())
	}
}
