package writers

import (
	"database/sql"
	"log"
	"time"

	"github.com/mkobaly/jiraworklog/types"

	_ "github.com/denisenkom/go-mssqldb"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
)

//Writer is the interface that handles writing Jira Worklog items
type Writer interface {
	Write(w types.WorklogItem) error
	NonResolvedIssues() ([]string, error)
	UpdateResolutionDate(issueKey string, resolvedDate time.Time) error
}

//SQLWriter is the SQL Server writer
type SQLWriter struct {
	DB *sqlx.DB
}

// func (r *Repository) Fetch() ([]types.IssueResponse, error) {
// 	result := []types.IssueResponse{}
// 	err := r.DB.Select(&result, `SELECT worklogId, developer, date,
// 	timeSpendSeconds, issueId, issueKey, issueType, issueSummary,
// 	issuePriority FROM worklog`)
// 	return result, err
// }

func (r *SQLWriter) NonResolvedIssues() ([]string, error) {
	result := []string{}
	err := r.DB.Select(&result, `
		SELECT distinct IssueKey
		FROM worklog
		where IssueResolvedDate IS NULL
		UNION
		SELECT distinct parentIssueKey
		FROM worklog
		where parentIssueId IS NOT NULL
		and parentIssueResolvedDate IS NULL
	 `)
	return result, err
}

//Write will add the worklogItem to SQL server
func (s *SQLWriter) Write(w types.WorklogItem) error {
	stmt, err := s.DB.Prepare(`
	INSERT INTO worklog 
	(
		worklogId, developer, date, timeSpendSeconds, originalEstimateSeconds, project, 
		issueId, issueKey, issueType, issueSummary, issuePriority, issueStatus, issueCreateDate, issueResolvedDate,
		parentIssueId, parentIssueKey, parentIssueType, parentIssueSummary, parentIssuePriority, parentIssueStatus, parentIssueCreateDate, parentIssueResolvedDate
	) 
	VALUES(@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9, @p10, @p11, @p12, @p13, @p14, @p15, @p16, @p17, @p18, @p19, @p20, @p21, @p22)`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(w.ID, w.Author, mssql.DateTime1(w.Started), w.TimeSpentSeconds, w.OriginalEstimateSeconds, w.Project,
		w.IssueID, w.IssueKey, w.IssueType, w.IssueSummary, w.IssuePriority, w.IssueStatus, sqlDate(w.IssueCreateDate), sqlDate(w.IssueResolvedDate),
		w.ParentIssueID, w.ParentIssueKey, w.ParentIssueType, w.ParentIssueSummary, w.ParentIssuePriority,
		w.ParentIssueStatus, sqlDate(w.ParentIssueCreateDate), sqlDate(w.ParentIssueResolvedDate))

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

func (s *SQLWriter) UpdateResolutionDate(issueKey string, resolvedDate time.Time) error {
	stmt, err := s.DB.Prepare(`
		UPDATE worklog 
			SET issueResolvedDate = @p2
		WHERE issueKey = @p1

		UPDATE worklog
			SET parentIssueResolvedDate = @p2
		WHERE parentIssueKey = @p1`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(issueKey, sqlDate(resolvedDate))
	if err != nil {
		return err
	}
	return nil
}

func sqlDate(t time.Time) interface{} {
	var r interface{}
	r = &sql.NullBool{}
	if !t.IsZero() {
		r = mssql.DateTime1(t)
	}
	return r
}
