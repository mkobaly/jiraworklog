package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
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
				[id], [key], [type], summary, priority, status, project, developer, createDate, updateDate,
				resolvedDate, isResolved, daysToResolve, aggregateTimeSpent, aggregateTimeOriginalEstimate
			)
			VALUES(@p15, @p16, @p17, @p18, @p19, @p20, @p21, @p22, @p23, @p24, @p25, @p26, @p27, @p28, @p29)
		ELSE
			UPDATE issue SET updateDate = @p24 WHERE [id] = @p15`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = stmt.Exec(w.ID, w.Author, mssql.DateTime1(w.Date), w.WeekNumber, w.WeekDay, w.TimeSpentSeconds, w.TimeSpentHours,
		w.Project, w.IssueID, w.IssueKey, w.IssueType, w.IssueSummary, w.IssuePriority, w.IssueStatus,
		w.ParentIssueID, w.ParentIssueKey, w.ParentIssueType, w.ParentIssueSummary, w.ParentIssuePriority,
		w.ParentIssueStatus, pi.Project, pi.Developer, mssql.DateTime1(pi.CreateDate), mssql.DateTime1(pi.UpdateDate),
		sqlDate(pi.ResolvedDate), pi.IsResolved, pi.DaysToResolve, pi.AggregateTimeSpent, pi.AggregateTimeOriginalEstimate)

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

//UpdateIssue will update the resolved information for the given issue
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

//Close will close the database connection
func (s *SQL) Close() {
	s.DB.Close()
}

//AllWorkLogs will return all of the work logs from SQL server
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

//AllIssues will return all issues from SQL server
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

//IssuesGroupedBy will return issues group by the given groupBy value going
//back daysBack. This data will be used for charting
func (s *SQL) IssuesGroupedBy(groupBy string, weeksBack int) ([]types.IssueChartData, error) {

	result := []types.IssueChartData{}
	err := s.DB.Select(&result, fmt.Sprintf(`
	select [%s] [groupBy],
		sum(case when isResolved = 1 then 1 else 0 end) [resolved],
		sum(case when isResolved = 0 then 1 else 0 end) [nonResolved],
		ISNULL(sum(daystoResolve) / NULLIF(sum(case when isResolved = 1 then 1 else 0 end),0),0)  [daysToResolve],
		CONVERT(DECIMAL(10,2),sum(aggregateTimeSpent / 3600.00))  [timeSpent],
		sum(aggregateTimeOriginalEstimate / 3600.00)  [timeEstimate]
	FROM issue
	WHERE datepart(week,updateDate) >= datepart(week,getutcdate()) - @p1
	group by [%s]`, groupBy, groupBy), weeksBack)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// func (s *SQL) WorklogsPerDevPerWeek(weeksBack int) ([]types.IssueChartData, error) {

// 	var _, weekNumber = time.Now().ISOWeek()

// 	result := []types.IssueChartData{}
// 	err := s.DB.Select(&result, fmt.Sprintf(`
// 	SELECT author, weekNumber, sum(timeSpentHours) [timeSpentHours]
// 	FROM weekNumber >= datepart(WEEK, getdate()) - 4
// 	GROUP BY author, weekNumber
// 	ORDER BY author, weekNumber`, groupBy, groupBy), daysBack)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return result, nil
// }

func (s *SQL) WorklogsPerDay() ([]types.WorklogsPerDay, error) {
	finalResults := []types.WorklogsPerDay{
		types.WorklogsPerDay{Day: "Sunday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Monday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Tuesday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Wednesday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Thursday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Friday", TimeSpentHrs: 0},
		types.WorklogsPerDay{Day: "Saturday", TimeSpentHrs: 0},
	}

	qr := []types.WorklogsPerDay{}
	err := s.DB.Select(&qr, `
		SELECT weekDay [day], sum(timeSpentHours) [timeSpentHrs]
		FROM worklog
		WHERE weekNumber = datepart(WEEK, dateadd(day, -7, getdate()))
		GROUP BY weekDay
		ORDER BY weekDay`)
	if err != nil {
		return nil, err
	}

	for i := range finalResults {
		for _, w := range qr {
			if strings.ToLower(finalResults[i].Day) == strings.ToLower(w.Day) {
				finalResults[i].TimeSpentHrs = w.TimeSpentHrs
				continue
			}
		}
	}

	// for _, wd := range finalResults {
	// 	for _, w := range qr {
	// 		if strings.ToLower(wd.Day) == strings.ToLower(w.Day) {
	// 			wd.TimeSpentHrs = w.TimeSpentHrs
	// 			continue
	// 		}
	// 	}
	// }
	return finalResults, nil
}

func (s *SQL) WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error) {
	results := make(map[string]*types.WorklogsPerDevDay)
	final := []types.WorklogsPerDevDay{}

	qr := []types.WorklogsPerDay{}
	err := s.DB.Select(&qr, `
	SELECT author [developer], weekDay [day], sum(timeSpentHours) [timeSpentHrs]
	FROM worklog
	WHERE weekNumber = datepart(WEEK, dateadd(day, -7, getdate()))
	GROUP BY author, weekDay
	ORDER BY author, weekDay`)
	if err != nil {
		return nil, err
	}

	for _, item := range qr {
		if _, ok := results[item.Developer]; !ok {
			results[item.Developer] = &types.WorklogsPerDevDay{Developer: item.Developer}
		}

		switch item.Day {
		case "Monday":
			results[item.Developer].Monday = item.TimeSpentHrs
		case "Tuesday":
			results[item.Developer].Tuesday = item.TimeSpentHrs
		case "Wednesday":
			results[item.Developer].Wednesday = item.TimeSpentHrs
		case "Thursday":
			results[item.Developer].Thursday = item.TimeSpentHrs
		case "Friday":
			results[item.Developer].Friday = item.TimeSpentHrs
		}
	}

	for _, v := range results {
		final = append(final, *v)
	}
	return final, nil
}

func (s *SQL) WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error) {
	results := make(map[string]*types.WorklogsPerDevWeek)
	final := []types.WorklogsPerDevWeek{}

	_, week := time.Now().ISOWeek()

	qr := []types.WorklogsAggQueryResult{}
	err := s.DB.Select(&qr, `
	SELECT author [developer], weekNumber [group], sum(timeSpentHours) [timeSpentHrs]
	FROM worklog
	WHERE weekNumber >= datepart(WEEK, getdate()) - 4
	GROUP BY author, weekNumber
	ORDER BY author`)
	if err != nil {
		return nil, err
	}

	for _, item := range qr {
		if _, ok := results[item.Developer]; !ok {
			results[item.Developer] = &types.WorklogsPerDevWeek{Developer: item.Developer}
		}
		w, _ := strconv.Atoi(item.Group)

		switch week - w {
		case 0:
			results[item.Developer].ThisWeek = item.TimeSpentHrs
		case 1:
			results[item.Developer].LastWeek = item.TimeSpentHrs
		case 2:
			results[item.Developer].TwoWeeks = item.TimeSpentHrs
		case 3:
			results[item.Developer].ThreeWeeks = item.TimeSpentHrs
		case 4:
			results[item.Developer].FourWeeks = item.TimeSpentHrs
		}
	}

	for _, v := range results {
		final = append(final, *v)
	}
	return final, nil
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
