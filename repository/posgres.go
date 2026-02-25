package repository

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
)

// SQL is the SQL Server repository
type Postgres struct {
	DB *sqlx.DB
}

// NewSQLRepo will create a new repository using a Postgres database as storage
func NewPostgresRepo(cfg *jiraworklog.Config) (*Postgres, error) {
	db, err := sqlx.Connect("pgx", cfg.SQLConnection)
	if err != nil {
		return nil, err
	}
	repo := &Postgres{DB: db}
	return repo, nil
}

func (s *Postgres) MaitenanceRatio(roles []string) ([]types.MaitenanceRatio, error) {
	result := []types.MaitenanceRatio{}

	query := `
        SELECT
			year_month,
			SUM(CASE WHEN category = 'NR' THEN hours ELSE 0 END) AS nr,
			SUM(CASE WHEN category = 'AM' THEN hours ELSE 0 END) AS am,
			SUM(CASE WHEN category = 'PR' THEN hours ELSE 0 END) AS pr,
			SUM(CASE WHEN category = '--' THEN hours ELSE 0 END) AS other
		FROM (
				SELECT
					to_char(date, 'YYYY-MM') AS year_month,
					timespenthours AS hours,
					CASE
						WHEN i.projectcharge = 'Non-Recoverable' THEN 'NR'
						WHEN i.projectcharge ILIKE '%after market%' THEN 'AM'
						WHEN i.projectcharge ILIKE 'TD%' THEN 'PR'
						ELSE '--'
						END AS category
				FROM worklog w
				JOIN issue i on w.issueid = i.id
				WHERE i.projectcharge <> ''
				AND i.projectcharge NOT ILIKE 'SS%'
				AND date >= date_trunc('month', CURRENT_DATE) - INTERVAL '2 years'
				AND date < date_trunc('month', now() AT TIME ZONE 'UTC')
				AND author IN (
					SELECT name FROM people WHERE role = ANY($1)
				)
			) AS src
		GROUP BY year_month
		ORDER BY year_month;`

	// PostgreSQL's ANY($1) works directly with a string array, no need for sqlx.In
	err := s.DB.Select(&result, query, roles)
	return result, err
}

func (s *Postgres) People() ([]types.People, error) {
	result := []types.People{}
	err := s.DB.Select(&result, `	
		SELECT id, name, isEmployee, role, location FROM people order by name;`)
	return result, err
}

func (s *Postgres) UpdatePerson(personId int, role string, isEmployee bool, location string) error {
	stmt, err := s.DB.Prepare(`
		UPDATE people SET role = $2, isEmployee = $3, location = $4 WHERE id = $1`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(personId, role, isEmployee, location)
	return err
}

func (s *Postgres) AllRoles() ([]string, error) {
	result := []string{}
	err := s.DB.Select(&result, `
		SELECT distinct role FROM people;`)
	return result, err
}

func (s *Postgres) IssuesMissingProjectCharge() ([]types.IssueMissingCharge, error) {
	result := []types.IssueMissingCharge{}
	err := s.DB.Select(&result, `
		SELECT project, key, type, summary, priority, status, updatedate
		FROM issue
		WHERE projectcharge = ''
		AND id IN (SELECT worklog.issueid FROM worklog)
		AND updatedate >= NOW() - INTERVAL '60 days'
		AND project not in ('HT', 'AHT')
		ORDER BY updatedate DESC;`)
	return result, err
}

func (s *Postgres) IssuesMismatchedProjectCharge() ([]types.IssueMismatchedCharge, error) {
	result := []types.IssueMismatchedCharge{}
	err := s.DB.Select(&result, `
		SELECT
			i2.key as parentkey,
			i2.type as parenttype,
			i2.projectcharge as parentprojectcharge,
			i2.summary as parentsummary,
			i.key,
			i.type,
			i.projectcharge,
			i.summary
		FROM issue i
		JOIN issue i2 ON i.parentid = i2.id
		WHERE i2.projectcharge != i.projectcharge
		AND i.createdate >= now() - INTERVAL '2 months'
		AND i.project IN ('IDM', 'SYM', 'ESG')
		ORDER BY i2.projectcharge;`)
	return result, err
}

func (s *Postgres) CustomerBugCounts(project string) ([]types.CustomerBugCount, error) {
	result := []types.CustomerBugCount{}
	query := `
		SELECT
			project,
			COALESCE(NULLIF(priority, ''), 'Medium') AS priority,
			to_char(createdate, 'YYYY-MM') AS year_month,
			count(*) AS count
		FROM issue
		WHERE type IN ('Customer Bug', 'HW / FW Customer Bug')
		AND createdate >= now() - INTERVAL '2 years'
		AND createdate < date_trunc('month', now() AT TIME ZONE 'UTC')
		AND ($1 = '' OR project = $1)
		GROUP BY project, priority, to_char(createdate, 'YYYY-MM')
		ORDER BY year_month, priority;`
	err := s.DB.Select(&result, query, project)
	return result, err
}

func (s *Postgres) CustomerBugProjects() ([]string, error) {
	result := []string{}
	err := s.DB.Select(&result, `
		SELECT DISTINCT project
		FROM issue
		WHERE type IN ('Customer Bug', 'HW / FW Customer Bug')
		AND createdate >= now() - INTERVAL '2 years'
		ORDER BY project;`)
	return result, err
}

// AllProjects returns all projects from the database
func (s *Postgres) AllProjects() ([]types.Project, error) {
	result := []types.Project{}
	err := s.DB.Select(&result, `
		SELECT id, name, visible, projectcharge
		FROM project
		ORDER BY name;`)
	return result, err
}

// CreateProject creates a new project in the database
func (s *Postgres) CreateProject(name string, projectCharges []string) error {
	_, err := s.DB.Exec(`
		INSERT INTO project (name, projectcharge)
		VALUES ($1, $2)`,
		name, types.StringArray(projectCharges))
	return err
}

// UpdateProject updates an existing project
func (s *Postgres) UpdateProject(id int, name string, visible bool, projectCharges []string) error {
	_, err := s.DB.Exec(`
		UPDATE project
		SET name = $2, visible = $3, projectcharge = $4
		WHERE id = $1`,
		id, name, visible, types.StringArray(projectCharges))
	return err
}

// DeleteProject deletes a project by ID
func (s *Postgres) DeleteProject(id int) error {
	_, err := s.DB.Exec(`DELETE FROM project WHERE id = $1`, id)
	return err
}

// AllProjectCharges returns all distinct project charges from the issue table
func (s *Postgres) AllProjectCharges() ([]string, error) {
	result := []string{}
	err := s.DB.Select(&result, `
		SELECT DISTINCT projectcharge
		FROM issue
		WHERE projectcharge <> ''
		ORDER BY projectcharge;`)
	return result, err
}

// ProjectChargeHours returns hours worked per project charge and role
func (s *Postgres) ProjectChargeHours() ([]types.ProjectChargeHours, error) {
	result := []types.ProjectChargeHours{}
	err := s.DB.Select(&result, `
		SELECT
			COALESCE(p.name, 'UNDEFINED') as project,
			COALESCE(NULLIF(j.projectcharge, ''), 'PROJECT CHARGE MISSING') as projectcharge,
			to_char(w.date, 'YYYY-MM') AS yearmonth,
			NULLIF(per.role, 'UNKNOWN') as role,
			per.isemployee,
			per.location,
			SUM(w.timespenthours) as hours
		FROM worklog w
				JOIN issue j ON w.issueid = j.id
				LEFT JOIN project p ON j.projectcharge = ANY(p.projectcharge)
				LEFT JOIN people per ON w.author = per.name
		WHERE w.date >= now() - INTERVAL '13 months'
		AND COALESCE(p.visible, true)  = true
		GROUP BY p.name, j.projectcharge, to_char(w.date, 'YYYY-MM'), per.role, per.isemployee, per.location
		ORDER BY p.name, j.projectcharge;`)
	return result, err
}

// DailyHoursByRole returns daily hours breakdown by author for the given date range
func (s *Postgres) DailyHoursByRole(roles []string, startDate, endDate time.Time) ([]types.DailyHours, error) {
	result := []types.DailyHours{}

	query := `
		SELECT
			role,
			author,
			date,
			SUM(CASE WHEN category = 'NR' THEN hours ELSE 0 END) AS nonrecoverable,
			SUM(CASE WHEN category = 'AM' THEN hours ELSE 0 END) AS aftermarket,
			SUM(CASE WHEN category = 'PR' THEN hours ELSE 0 END) AS project,
			SUM(CASE WHEN category = '--' THEN hours ELSE 0 END) AS missing
		FROM (
			SELECT
				author,
				p.role,
				to_char((date AT TIME ZONE 'America/New_York'), 'YYYY-MM-DD') AS date,
				timespenthours AS hours,
				CASE
					WHEN i.projectcharge = 'Non-Recoverable' THEN 'NR'
					WHEN i.projectcharge ILIKE '%after market%' THEN 'AM'
					WHEN i.projectcharge ILIKE 'TD%' THEN 'PR'
					ELSE '--'
				END AS category
			FROM worklog w
			JOIN issue i ON w.issueid = i.id
			JOIN people p ON w.author = p.name
			WHERE w.date >= ($2 AT TIME ZONE 'UTC')
			AND w.date <= ($3 AT TIME ZONE 'UTC')
			AND p.role = ANY($1)
		) AS src
		GROUP BY role, author, date
		ORDER BY author, date;`

	err := s.DB.Select(&result, query, roles, startDate, endDate)
	return result, err
}

// ProjectTimeTracking returns time tracking data for issues in a given fixed version
func (s *Postgres) ProjectTimeTracking(fixedVersion string) ([]types.ProjectTimeTracking, error) {
	result := []types.ProjectTimeTracking{}
	query := `
		SELECT i.id, i.parentid, i.key, i.type, i.summary, i.status, i.projectcharge,
			i.originalestimate as estimateseconds,
			format_seconds(i.originalestimate, 7.5) as estimate,
			w.timespentseconds as devseconds,
			format_seconds(w.timespentseconds, 7.5) as devtimespent
		FROM issue i
		LEFT JOIN (
			SELECT issueId, cast(sum(timespentseconds) as integer) timespentseconds
			FROM worklog as w
			JOIN people p on w.author = p.name
			WHERE p.role = 'dev'
			GROUP BY issueId
		) w on i.id = w.issueid
		WHERE i.type not in ('Epic', 'Release Candidate')
		AND ($1 = any(fixedversions) OR i.parentid in (SELECT id from issue where key = $1))
		UNION
		SELECT i.id, i.parentid, i.key, i.type, i.summary, i.status, i.projectcharge,
			i.originalestimate as estimateseconds,
			format_seconds(i.originalestimate, 7.5) as estimate,
			w.timespentseconds as devseconds,
			format_seconds(w.timespentseconds, 7.5) as devtimespent
		FROM issue i
		JOIN issue i2 on i.parentid = i2.id
		LEFT JOIN (
			SELECT issueId, cast(sum(timespentseconds) as integer) as timespentseconds
			FROM worklog as w
			JOIN people p on w.author = p.name
			WHERE p.role = 'dev'
			GROUP BY issueId
		) w on i.id = w.issueid
		WHERE i2.type not in ('Release Candidate')
		AND ($1 = any(i2.fixedversions) OR i2.parentid in (SELECT id from issue where key = $1));`
	err := s.DB.Select(&result, query, fixedVersion)
	return result, err
}

func (s *Postgres) MissingIssues() ([]int, error) {
	result := []int{}
	err := s.DB.Select(&result, `	
		SELECT DISTINCT issueid FROM worklog
		WHERE issueid NOT IN 
		(
			SELECT id FROM issue
		)
		UNION
		SELECT parentid from issue where parentid not in (select id from issue);`)
	return result, err
}

// SaveWorklog will write the worklog entry to the database
func (s *Postgres) SaveWorklog(w *types.Worklog) error {
	stmt, err := s.DB.Prepare(`
        INSERT INTO worklog (
            id, author, "date", weeknumber, weekday, timespentseconds, timespenthours,  issueid
        )
        VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8
        )
        ON CONFLICT (id) DO UPDATE SET
			id = EXCLUDED.id,
			author = EXCLUDED.author,
			"date" = EXCLUDED."date",
			weeknumber = EXCLUDED.weeknumber,
			weekday = EXCLUDED.weekday,
			timespentseconds = EXCLUDED.timespentseconds,
			timespenthours = EXCLUDED.timespenthours,
			issueid = EXCLUDED.issueid;`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(w.ID, w.Author, w.Date, w.WeekNumber, w.WeekDay, w.TimeSpentSeconds, w.TimeSpentHours, w.IssueId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Postgres) DeleteWorklog(id int) error {
	stmt, err := s.DB.Prepare(`
        DELETE FROM worklog WHERE id = $1`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	return nil
}

// UpdateIssue will insert or update the issue in the database
func (s *Postgres) UpdateIssue(issue *types.StoredIssue) error {
	_, err := s.DB.Exec(`
		INSERT INTO issue (
			id, "key", parentid, type, summary, priority, status, project,
			projectcharge, fixedversions, createdate, updatedate, resolveddate, daystoresolve,
			timespent, originalestimate, remainingestimate
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (id) DO UPDATE SET
			"key" = EXCLUDED."key",
			parentid = EXCLUDED.parentid,
			type = EXCLUDED.type,
			summary = EXCLUDED.summary,
			priority = EXCLUDED.priority,
			status = EXCLUDED.status,
			project = EXCLUDED.project,
			projectcharge = EXCLUDED.projectcharge,
			fixedversions = EXCLUDED.fixedversions,
			updatedate = EXCLUDED.updatedate,
			resolveddate = EXCLUDED.resolveddate,
			daystoresolve = EXCLUDED.daystoresolve,
			timespent = EXCLUDED.timespent,
			originalestimate = EXCLUDED.originalestimate,
			remainingestimate = EXCLUDED.remainingestimate,
			dateupdated = now()`,
		issue.ID, issue.Key, issue.ParentId, issue.Type, issue.Summary, issue.Priority, issue.Status, issue.Project,
		issue.ProjectCharge, issue.FixedVersions, issue.CreateDate, issue.UpdateDate, issue.ResolvedDate, issue.DaysToResolve,
		issue.TimeSpent, issue.OriginalEstimate, issue.RemainingEstimate)
	return err
}

// Close will close the database connection
func (s *Postgres) Close() {
	s.DB.Close()
}

// IssuesGroupedBy will return issues group by the given groupBy value going
// back daysBack. This data will be used for charting
func (s *Postgres) IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error) {
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

// IssueAccuracy will return how accurate a developers estimate is vs actual time logged
func (s *Postgres) IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error) {
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

func (s *Postgres) WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error) {
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

func (s *Postgres) WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error) {
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

func (s *Postgres) WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error) {
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

// func (s *Postgres) WorklogsPerDay() ([]types.WorklogsPerDay, error) {
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
