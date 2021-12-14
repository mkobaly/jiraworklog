package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
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

func (s *SQL) WorklogGetLastTimestamp() (int64, error) {
	var lastTimestamp int64
	err := s.DB.Get(&lastTimestamp, "SELECT lastTimestamp FROM config")
	return lastTimestamp, err
}

func (s *SQL) WorklogUpdateLastTimestamp(lastTimestamp int64) error {
	_, err := s.DB.Exec("UPDATE config SET lastTimestamp=?", lastTimestamp)
	return err
}

func (s *SQL) WorklogGetMaxWorklogID() (int, error) {
	var maxWorklogID int
	err := s.DB.Get(&maxWorklogID, "SELECT maxWorklogID FROM config")
	return maxWorklogID, err
}

func (s *SQL) WorklogUpdateMaxWorklogID(maxWorklogID int) error {
	_, err := s.DB.Exec("UPDATE config SET maxWorklogID=?", maxWorklogID)
	return err
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
func (s *SQL) IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error) {
	result := []types.IssueChartData{}
	err := s.DB.Select(&result, fmt.Sprintf(`
	select [%s] [groupBy],
		sum(case when isResolved = 1 then 1 else 0 end) [resolved],
		sum(case when isResolved = 0 then 1 else 0 end) [nonResolved],
		ISNULL(sum(daystoResolve) / NULLIF(sum(case when isResolved = 1 then 1 else 0 end),0),0)  [daysToResolve],
		CONVERT(DECIMAL(10,2),sum(aggregateTimeSpent / 3600.00))  [timeSpent],
		sum(aggregateTimeOriginalEstimate / 3600.00)  [timeEstimate]
	FROM issue
	WHERE updateDate >= @p1
	AND updateDate <= @p2
	group by [%s]`, groupBy, groupBy), start, stop)
	if err != nil {
		return nil, err
	}
	return result, nil
}

//IssueAccuracy will return how accurate a developers estimate is vs actual time logged
func (s *SQL) IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error) {
	result := []types.IssueAccuracy{}
	err := s.DB.Select(&result, `
	SELECT developer, count(*) [count],
		CAST(100 - abs(((sum(aggregateTimeOriginalEstimate) - sum(aggregateTimeSpent)) /
		cast(sum(aggregateTimeOriginalEstimate) as decimal(18,2))) * 100.00) as decimal(5,2)) [accuracy]
	FROM issue
	WHERE updateDate >= @p1 and updateDate <= @p2
	and isResolved = 1
	and aggregateTimeOriginalEstimate > 0
	GROUP BY developer`, start, stop)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SQL) WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error) {
	result := []types.WorklogGroupByChart{}
	//date := time.Now().AddDate(0, 0, -7)
	err := s.DB.Select(&result, fmt.Sprintf(`
	SELECT %s [groupBy], sum(timeSpentHours) [timeSpentHrs]
	FROM worklog
	WHERE date >= @p1
	AND date <= @p2
	GROUP BY %s`, groupBy, groupBy), start, stop)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *SQL) WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error) {
	final := []map[string]string{}
	authors := make(map[string]bool)
	activity := make(map[types.DeveloperDateKey]float64)

	qr := []types.WorklogsPerDay{}
	err := s.DB.Select(&qr, `
	SELECT author [developer], convert(varchar(10),date,102) [day], sum(timeSpentHours) [timeSpentHrs]
	FROM worklog
	WHERE date >= @p1
	AND date <= @p2
	GROUP BY author, convert(varchar(10),date,102)
	ORDER BY author, day`, start, stop)
	if err != nil {
		return nil, err
	}

	for _, v := range qr {
		if _, ok := authors[v.Developer]; !ok {
			authors[v.Developer] = true
		}
		date, _ := time.Parse("2006.01.02", v.Day)
		key := types.DeveloperDateKey{Date: date.YearDay(), Developer: v.Developer}
		activity[key] = v.TimeSpentHrs
	}

	for a, _ := range authors {
		worklog := types.WorklogsPerDev{Developer: a}
		hoursTotal := 0.00
		for i := 0; i < int(stop.Sub(start).Hours()/24); i++ {
			date := start.AddDate(0, 0, i)
			key := types.DeveloperDateKey{Date: date.YearDay(), Developer: a}

			if _, ok := activity[key]; !ok {
				worklog.TimeSpent = append(worklog.TimeSpent, types.HoursPerDay{Date: date, TimeSpentHrs: 0})
			} else {
				worklog.TimeSpent = append(worklog.TimeSpent, types.HoursPerDay{Date: date, TimeSpentHrs: activity[key]})
				hoursTotal += activity[key]
			}
		}
		tmp := make(map[string]string)
		tmp[" Developer"] = a
		tmp[" Total Hrs"] = strconv.FormatFloat(hoursTotal, 'f', 2, 64)
		for _, v := range worklog.TimeSpent {
			tmp[v.Date.Format("Jan 02")] = strconv.FormatFloat(v.TimeSpentHrs, 'f', 2, 64)
		}
		final = append(final, tmp)
	}
	return final, nil
}

func (s *SQL) WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error) {
	results := make(map[string]*types.WorklogsPerDevWeek)
	final := []types.WorklogsPerDevWeek{}

	_, week := time.Now().ISOWeek()

	//TODO: need to account for new year so weeks could be (3,2,1,52)
	qr := []types.WorklogsAggQueryResult{}
	err := s.DB.Select(&qr, `
	SELECT author [developer], weekNumber [group], sum(timeSpentHours) [timeSpentHrs]
	FROM worklog
	WHERE weekNumber >= datepart(WEEK, getdate()) - 4
	AND year(date) = year(getdate())
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

// func (s *SQL) WorklogsPerDay() ([]types.WorklogsPerDay, error) {
// 	finalResults := []types.WorklogsPerDay{
// 		types.WorklogsPerDay{Day: "Sunday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Monday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Tuesday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Wednesday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Thursday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Friday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Saturday", TimeSpentHrs: 0},
// 	}

// 	date := time.Now().AddDate(0, 0, -7)
// 	qr := []types.WorklogsPerDay{}
// 	err := s.DB.Select(&qr, `
// 		SELECT weekDay [day], sum(timeSpentHours) [timeSpentHrs]
// 		FROM worklog
// 		WHERE weekNumber = datepart(WEEK, @p1)
// 		AND year(date) = year(@p1)
// 		GROUP BY weekDay
// 		ORDER BY weekDay`, date)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for i := range finalResults {
// 		for _, w := range qr {
// 			if strings.ToLower(finalResults[i].Day) == strings.ToLower(w.Day) {
// 				finalResults[i].TimeSpentHrs = w.TimeSpentHrs
// 				continue
// 			}
// 		}
// 	}
// 	return finalResults, nil
// }

// func (s *SQL) WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error) {
// 	results := make(map[string]*types.WorklogsPerDevDay)
// 	final := []types.WorklogsPerDevDay{}

// 	date := time.Now().AddDate(0, 0, -7)
// 	qr := []types.WorklogsPerDay{}
// 	err := s.DB.Select(&qr, `
// 	SELECT author [developer], weekDay [day], sum(timeSpentHours) [timeSpentHrs]
// 	FROM worklog
// 	WHERE weekNumber = datepart(WEEK, @p1)
// 	AND year(date) = year(@p1)
// 	GROUP BY author, weekDay
// 	ORDER BY author, weekDay`, date)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for _, item := range qr {
// 		if _, ok := results[item.Developer]; !ok {
// 			results[item.Developer] = &types.WorklogsPerDevDay{Developer: item.Developer}
// 		}

// 		switch item.Day {
// 		case "Monday":
// 			results[item.Developer].Monday = item.TimeSpentHrs
// 		case "Tuesday":
// 			results[item.Developer].Tuesday = item.TimeSpentHrs
// 		case "Wednesday":
// 			results[item.Developer].Wednesday = item.TimeSpentHrs
// 		case "Thursday":
// 			results[item.Developer].Thursday = item.TimeSpentHrs
// 		case "Friday":
// 			results[item.Developer].Friday = item.TimeSpentHrs
// 		}
// 	}

// 	for _, v := range results {
// 		final = append(final, *v)
// 	}
// 	return final, nil
// }

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
