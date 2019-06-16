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

//SQL is the SQL Server repository
type SQL struct {
	DB *sqlx.DB
}

//NewSQLRepo will create a new repository using a SQL database as storage
func NewSQLRepo(cfg *jiraworklog.Config) (*SQL, error) {
	db, err := sqlx.Connect("sqlserver", cfg.SQLConnection)
	if err != nil {
		return nil, err
	}
	repo := &SQL{DB: db}
	return repo, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (s *SQL) NonResolvedIssues() ([]types.ParentIssue, error) {
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
		,[isResolved]
		,aggregateTimeSpent
		,aggregateTimeOriginalEstimate
	FROM [issue]
	WHERE isResolved = 0
	AND dateInserted <= dateadd(minute,-10, getutcdate())`)
	return result, err
}

//Write will add the worklogItem to SQL server
func (s *SQL) Write(w *types.WorklogItem, pi *types.ParentIssue) error {
	//p := w.GetParent()
	stmt, err := s.DB.Prepare(`	
		IF NOT EXISTS (SELECT * FROM worklog WHERE id = @p1)
		INSERT INTO worklog 
		(
			id, author, date, weekNumber, weekDay, timeSpentSeconds, timeSpentHours, project, 
			issueId, issueKey, issueType, issueSummary, issuePriority, issueStatus,
			parentIssueId, parentIssueKey, parentIssueType, parentIssueSummary, parentIssuePriority, parentIssueStatus
		) 
		VALUES(@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, @p13, @p14, @p15, @p16, @p17, @p18, @p19, @p20)
		
		IF NOT EXISTS (SELECT * FROM issue WHERE [id] = @p15)
		INSERT INTO issue
		(
			[id], [key], [type], summary, priority, status, project, developer, createDate, 
			resolvedDate, isResolved, daysToResolve, aggregateTimeSpent, aggregateTimeOriginalEstimate
		)
		VALUES(@p15, @p16, @p17, @p18, @p19, @p20, @p21, @p22, @p23, @p24, @p25, @p26, @p27, @p28)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(w.ID, w.Author, mssql.DateTime1(w.Date), w.WeekNumber, w.WeekDay, w.TimeSpentSeconds, w.TimeSpentHours,
		w.Project, w.IssueID, w.IssueKey, w.IssueType, w.IssueSummary, w.IssuePriority, w.IssueStatus,
		w.ParentIssueID, w.ParentIssueKey, w.ParentIssueType, w.ParentIssueSummary, w.ParentIssuePriority,
		w.ParentIssueStatus, pi.Project, pi.Developer, sqlDate(&pi.CreateDate), sqlDate(pi.ResolvedDate), pi.IsResolved,
		pi.DaysToResolve, pi.AggregateTimeSpent, pi.AggregateTimeOriginalEstimate)

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

func (s *SQL) UpdateIssue(issue *types.ParentIssue) error {
	stmt, err := s.DB.Prepare(`
		UPDATE issue 
			SET resolvedDate = @p2, 
			isResolved = 1, 
			aggregateTimeSpent = @p3,
			aggregateTimeOriginalEstimate = @p4,
			daysToResolve = @p5
		WHERE id = @p1`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(issue.ID, sqlDate(issue.ResolvedDate),
		issue.AggregateTimeSpent, issue.AggregateTimeOriginalEstimate, issue.DaysToResolve)
	if err != nil {
		return err
	}
	return nil
}

//UpdateResolutionDate will update the resolution date for a jira issue
// func (s *SQL) UpdateResolutionDate(issueKey string, resolvedDate time.Time) error {
// 	stmt, err := s.DB.Prepare(`
// 		UPDATE issue
// 			SET resolvedDate = @p2, isResolved = 1
// 		WHERE [key] = @p1`)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	_, err = stmt.Exec(issueKey, sqlDate(&resolvedDate))
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (s *SQL) Close() {
	s.DB.Close()
}

func (s *SQL) AllWorkLogs() ([]types.WorklogItem, error) {
	result := []types.WorklogItem{}
	err := s.DB.Select(&result, `		
		SELECT  [id]
		,[author]
		,[date]
		,weekNumber
		,weekDay
		,[timeSpentSeconds]
		,timeSpentHours
		,[project]
		,[issueId]
		,[issueKey]
		,[issueType]
		,[issueSummary]
		,[issuePriority]
		,[issueStatus]
		,[parentIssueId]
		,[parentIssueKey]
		,[parentIssueType]
		,[parentIssueSummary]
		,[parentIssuePriority]
		,[parentIssueStatus]
		FROM worklog`)
	return result, err
}

func (s *SQL) AllIssues() ([]types.ParentIssue, error) {
	result := []types.ParentIssue{}
	err := s.DB.Select(&result, `	
	SELECT 
		[id]
		,[key]
		,[type]
		,[summary]
		,[priority]
		,[status]
		,[project]
		,[createDate]
		,[resolvedDate]
		,[isResolved]
		,daysToResolve
		,aggregateTimeSpent
		,aggregateTimeOriginalEstimate
		,developer
	FROM [issue]`)
	return result, err
}

func (s *SQL) IssuesGroupedBy(groupBy string, daysBack int) ([]types.IssueChartData, error) {

	result := []types.IssueChartData{}
	err := s.DB.Select(&result, `
	select [type],
		sum(case when isResolved = 1 then 1 else 0 end) [resolved],
		sum(case when isResolved = 0 then 1 else 0 end) [nonResolved],
		sum(daystoResolve) / sum(case when isResolved = 1 then 1 else 0 end)  [daysToResolve],
		sum(aggregateTimeSpent / 3600.00)  [timeSpent],
		sum(aggregateTimeOriginalEstimate / 3600.00)  [timeSpent]
	FROM issue
	WHERE createDate >= dateadd(day,-1 * @p1, getdate())
	group by [type]`, daysBack)
	if err != nil {
		return nil, err
	}
	return result, nil

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
