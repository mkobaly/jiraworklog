package repository

import (
	"database/sql"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
	"log"
	"time"
)

//SQL is the SQL Server writer
type SQL struct {
	DB *sqlx.DB
}

//NewSQLRepo will create a new writer with a SQL database
func NewSQLRepo(cfg *jiraworklog.Config) (*SQL, error) {
	db, err := sqlx.Connect("sqlserver", cfg.SQLConnection)
	if err != nil {
		return nil, err
	}
	writer := &SQL{DB: db}
	return writer, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (s *SQL) NonResolvedIssues() ([]string, error) {
	result := []string{}
	err := s.DB.Select(&result, `
		SELECT [key]
		FROM issue
		WHERE isResolved = 0
		AND dateInserted <= dateadd(hour,-2, getutcdate())
	 `)
	return result, err
}

//Write will add the worklogItem to SQL server
func (s *SQL) Write(w types.WorklogItem) error {
	p := w.GetParent()
	stmt, err := s.DB.Prepare(`
	IF NOT EXISTS (SELECT * FROM worklog WHERE id = @p1)
	INSERT INTO worklog 
	(
		id, author, date, timeSpentSeconds, originalEstimateSeconds, project, 
		issueId, issueKey, issueType, issueSummary, issuePriority, issueStatus,
		parentIssueId, parentIssueKey, parentIssueType, parentIssueSummary, parentIssuePriority, parentIssueStatus
	) 
	VALUES(@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, @p13, @p14, @p15, @p16, @p17, @p18)
	
	IF NOT EXISTS (SELECT * FROM issue WHERE [id] = @p13)
	INSERT INTO issue
	(
		[id],[key],[type],summary,priority,status, project, createDate,resolvedDate,isResolved
	)
	VALUES(@p13, @p14, @p15, @p16, @p17, @p18, @p19, @p20, @p21, @p22)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(w.ID, w.Author, mssql.DateTime1(w.Started), w.TimeSpentSeconds, w.OriginalEstimateSeconds, w.Project,
		w.IssueID, w.IssueKey, w.IssueType, w.IssueSummary, w.IssuePriority, w.IssueStatus,
		w.ParentIssueID, w.ParentIssueKey, w.ParentIssueType, w.ParentIssueSummary, w.ParentIssuePriority,
		w.ParentIssueStatus, p.Project, sqlDate(&p.CreateDate), sqlDate(p.ResolvedDate), p.IsResolved)

	if err != nil {
		switch err := err.(type) {
		case mssql.Error:
			if err.Number != 2627 { //unique constraint
				return err
			}
			break
		default:
			return err
		}
	}
	return nil
}

//UpdateResolutionDate will update the resolution date for a jira issue
func (s *SQL) UpdateResolutionDate(issueKey string, resolvedDate time.Time) error {
	stmt, err := s.DB.Prepare(`
		UPDATE issue 
			SET resolvedDate = @p2, isResolved = 1
		WHERE [key] = @p1`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(issueKey, sqlDate(&resolvedDate))
	if err != nil {
		return err
	}
	return nil
}

func (s *SQL) Close() {
	s.DB.Close()
}

func (s *SQL) AllWorkLogs() ([]types.WorklogItem, error) {
	result := []types.WorklogItem{}
	err := s.DB.Select(&result, `
		SELECT  [id]
		,[author]
		,[date] [started]
		,[timespentseconds]
		,[originalestimateseconds]
		,[project]
		,[issueid]
		,[issuekey]
		,[issuetype]
		,[issuesummary]
		,[issuepriority]
		,[issuestatus]
		,[parentissueid]
		,[parentissuekey]
		,[parentissuetype]
		,[parentissuesummary]
		,[parentissuepriority]
		,[parentissuestatus]
		FROM worklog`)
	return result, err
}

func (s *SQL) AllIssues() ([]types.ParentIssue, error) {
	result := []types.ParentIssue{}
	err := s.DB.Select(&result, `
	SELECT [id]
	,[key]
	,[type]
	,[summary]
	,[priority]
	,[status]
	,[project]
	,[createDate]
	,[resolvedDate]
	,[isResolved] FROM [dbo].[issue]`)
	return result, err
}

func sqlDate(t *time.Time) interface{} {
	var r interface{}
	r = &sql.NullBool{}
	if t == nil {
		return r
	}
	if !t.IsZero() {
		r = mssql.DateTime1(*t)
	}
	return r
}
