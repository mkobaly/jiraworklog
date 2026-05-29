package repository

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
	"golang.org/x/sync/errgroup"
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
		SELECT distinct role FROM people WHERE role IS NOT NULL;`)
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

func (s *Postgres) CustomerBugTrends(project string) ([]types.CustomerBugTrend, error) {
	result := []types.CustomerBugTrend{}
	query := `
		WITH window_start AS (
			-- Start of the 12-month window: same month one year before the last complete month.
			-- e.g. today = Apr 21 2026 → last complete month = Mar 2026 → start = Mar 2025
			SELECT date_trunc('month', now()) - INTERVAL '13 months' AS ts
		),
		months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				(SELECT ts FROM window_start),
				date_trunc('month', now()) - INTERVAL '1 month',
				'1 month'::interval
			) gs
		),
		new_bugs AS (
			SELECT to_char(createdate, 'YYYY-MM') AS year_month, COUNT(*) AS cnt
			FROM issue
			WHERE type IN ('Customer Bug', 'HW / FW Customer Bug')
			AND createdate >= (SELECT ts FROM window_start)
			AND createdate < date_trunc('month', now())
			AND ($1 = '' OR project = $1)
			GROUP BY to_char(createdate, 'YYYY-MM')
		),
		closed_bugs AS (
			SELECT to_char(coalesce(resolveddate, updatedate), 'YYYY-MM') AS year_month, COUNT(*) AS cnt
			FROM issue
			WHERE type IN ('Customer Bug', 'HW / FW Customer Bug')
			AND ($1 = '' OR project = $1)
			AND (resolveddate >= (SELECT ts FROM window_start) AND resolveddate < date_trunc('month', now())
					OR (status LIKE 'Awaiting Release%' AND updatedate >= (SELECT ts FROM window_start) AND updatedate < date_trunc('month', now()))
				)
			GROUP BY to_char(coalesce(resolveddate, updatedate), 'YYYY-MM')
		),
		baseline AS (
			SELECT COUNT(*) AS open_count
			FROM issue
			WHERE type IN ('Customer Bug', 'HW / FW Customer Bug')
			AND createdate < (SELECT ts FROM window_start)
			AND (resolveddate IS NULL OR resolveddate >= (SELECT ts FROM window_start))
			AND ($1 = '' OR project = $1)
		)
		SELECT
			m.year_month,
			COALESCE(n.cnt, 0) AS new_count,
			COALESCE(c.cnt, 0) AS closed_count,
			SUM(COALESCE(n.cnt, 0) - COALESCE(c.cnt, 0)) OVER (
				ORDER BY m.year_month
				ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
			) + b.open_count AS open_count
		FROM months m
		LEFT JOIN new_bugs    n ON n.year_month = m.year_month
		LEFT JOIN closed_bugs c ON c.year_month = m.year_month
		CROSS JOIN baseline b
		ORDER BY m.year_month`
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

func (s *Postgres) ProjectCharges() ([]types.ProjectCharge, error) {
	result := []types.ProjectCharge{}
	err := s.DB.Select(&result, `
		SELECT name, visible, label
		FROM project_charge
		ORDER BY name;`)
	return result, err
}

func (s *Postgres) UpdateProjectCharge(name string, visible bool, label string) error {
	stmt, err := s.DB.Prepare(`
		UPDATE project_charge SET visible = $2, label = $3 WHERE name = $1`)
	if err != nil {
		log.Fatal(err)
	}
	_, err = stmt.Exec(name, visible, label)
	return err
}

// ProjectChargeHours returns hours worked per project charge and role.
// fromMonth and toMonth are YYYY-MM strings; empty strings fall back to the last 13 months.
func (s *Postgres) ProjectChargeHours(fromYearMonth, toYearMonth string) ([]types.ProjectChargeHours, error) {
	result := []types.ProjectChargeHours{}
	from, to := s.calculateDateRange(fromYearMonth, toYearMonth)

	query := `
		SELECT
			COALESCE(p.label, 'UNDEFINED') as label,
			COALESCE(NULLIF(j.projectcharge, ''), 'PROJECT CHARGE MISSING') as projectcharge,
			to_char(w.date, 'YYYY-MM') AS yearmonth,
			NULLIF(per.role, 'UNKNOWN') as role,
			per.isemployee,
			per.location,
			SUM(w.timespenthours) as hours
		FROM worklog w
			JOIN issue j ON w.issueid = j.id
			LEFT JOIN project_charge p ON j.projectcharge = p.name
			LEFT JOIN people per ON w.author = per.name
		WHERE COALESCE(p.visible, true) = true
			AND w.date >= $1 AND w.date < $2
		GROUP BY p.name, j.projectcharge, to_char(w.date, 'YYYY-MM'), per.role, per.isemployee, per.location
		ORDER BY p.name, j.projectcharge`

	err := s.DB.Select(&result, query, from, to)
	return result, err
}

// calculateDateRange converts optional YYYY-MM strings into concrete time.Time boundaries.
// When blank, from defaults to 13 months ago (start of that month) and to defaults to now.
// When provided, from is the first day of fromYearMonth and to is the first day of the month
// after toYearMonth (exclusive upper bound), giving a complete inclusive month range.
func (s *Postgres) calculateDateRange(fromYearMonth, toYearMonth string) (from time.Time, to time.Time) {
	now := time.Now().UTC()
	if fromYearMonth == "" {
		t := now.AddDate(0, -13, 0)
		from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		t, _ := time.Parse("2006-01", fromYearMonth)
		from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if toYearMonth == "" {
		to = now
	} else {
		t, _ := time.Parse("2006-01", toYearMonth)
		// exclusive upper bound: first day of the next month captures the entire toYearMonth
		to = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	}
	return
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

// ProjectName returns the summary of the epic or fixed version for display purposes.
func (s *Postgres) ProjectName(epicOrVersion string) (string, error) {
	var name string
	err := s.DB.QueryRow(`
		SELECT summary FROM issue
		WHERE key = $1
		   OR id IN (SELECT id FROM issue WHERE $1 = ANY(fixedversions) AND type = 'Epic')
		LIMIT 1`, epicOrVersion).Scan(&name)
	return name, err
}

// ProjectTimeTracking returns time tracking data for issues in a given fixed version
func (s *Postgres) ProjectTimeTracking(fixedVersion string) ([]types.ProjectTimeTracking, error) {
	result := []types.ProjectTimeTracking{}
	query := `
		SELECT i.id, i.parentid, i.key, i.type, i.summary, i.status, i.projectcharge,
			i.originalestimate as estimateseconds,
			w.timespentseconds as devseconds,
			i.remainingestimate as remainingseconds
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
			w.timespentseconds as devseconds,
			i.remainingestimate as remainingseconds
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

// IssuesWithStaleStints — see repository/repo.go for contract.
func (s *Postgres) IssuesWithStaleStints(notUpdatedSince time.Duration, limit int) ([]int, error) {
	cutoff := time.Now().Add(-notUpdatedSince)
	result := []int{}
	// Compare each issue.status against the status of its most recent
	// status_stints row. DISTINCT ON (issueid) ORDER BY datestarted DESC
	// gives us the latest stint per issue. Issues with no stints don't
	// appear (inner join) — those are an absent-stints problem, separate
	// from drift, and outside the backfill job's scope.
	err := s.DB.Select(&result, `
		WITH latest_stint AS (
			SELECT DISTINCT ON (issueid)
				issueid,
				status
			FROM status_stints
			ORDER BY issueid, datestarted DESC
		)
		SELECT i.id
		FROM issue i
		JOIN latest_stint ls ON ls.issueid = i.id
		WHERE i.status <> ls.status
		AND i.updatedate < $1
		ORDER BY i.updatedate ASC
		LIMIT $2`, cutoff, limit)
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
		AND date >= now() - INTERVAL '3 months'
		UNION
		SELECT parentid from issue where parentid not in (select id from issue);`)
	return result, err
}

func (s *Postgres) LastSeenIssues(threshold time.Duration) ([]int, error) {
	cutoff := time.Now().Add(-threshold)
	result := []int{}
	err := s.DB.Select(&result, `	
		SELECT id 
		FROM issue
		WHERE dateUpdated < $1
		and updatedate >= now() - INTERVAL '3 months'
		LIMIT 200;`, cutoff)
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

func (s *Postgres) SyncPeople() error {
	stmt, err := s.DB.Prepare(`
        INSERT INTO people(name)
		SELECT DISTINCT author from worklog
		WHERE date >= now() - INTERVAL '5 days'
		ON CONFLICT (name) DO NOTHING;`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}

func (s *Postgres) SyncProjectCharges() error {
	stmt, err := s.DB.Prepare(`
        INSERT INTO project_charge(name)
		SELECT DISTINCT projectcharge from issue
		WHERE createdate >= now() - INTERVAL '5 days'
		and projectcharge != ''
		ON CONFLICT (name) DO NOTHING;`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec()
	if err != nil {
		return err
	}
	return nil
}

func (s *Postgres) DeleteWorklog(id int) error {
	stmt, err := s.DB.Prepare(`DELETE FROM worklog WHERE id = $1`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Postgres) DeleteIssue(id int) error {
	stmt, err := s.DB.Prepare(`DELETE FROM issue WHERE id = $1`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id)
	if err != nil {
		return err
	}
	return nil
}

func (s *Postgres) UpdateIssueLastSeen(id int) error {
	stmt, err := s.DB.Prepare(`UPDATE issue set dateUpdated = now() WHERE id = $1`)
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
			timespent, originalestimate, remainingestimate, labels, assignee
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
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
			labels = EXCLUDED.labels,
			assignee = EXCLUDED.assignee,
			dateupdated = now()`,
		issue.ID, issue.Key, issue.ParentId, issue.Type, issue.Summary, issue.Priority, issue.Status, issue.Project,
		issue.ProjectCharge, issue.FixedVersions, issue.CreateDate, issue.UpdateDate, issue.ResolvedDate, issue.DaysToResolve,
		issue.TimeSpent, issue.OriginalEstimate, issue.RemainingEstimate, issue.Labels, issue.Assignee)
	return err
}

// Bulk insert all status changelogs. This is typically for a single jira issue
func (s *Postgres) BulkInsertChangelogs(transitions []types.ChangelogStatus) error {
	if len(transitions) == 0 {
		return nil
	}

	issueIDs := make([]int, len(transitions))
	fromStatuses := make([]string, len(transitions))
	toStatuses := make([]string, len(transitions))
	dates := make([]time.Time, len(transitions))
	ids := make([]int, len(transitions))

	for i, t := range transitions {
		issueIDs[i] = t.IssueID
		fromStatuses[i] = t.FromStatus
		toStatuses[i] = t.ToStatus
		dates[i] = t.Date
		ids[i] = t.ID
	}

	_, err := s.DB.Exec(`
		INSERT INTO issue_transition (id, issueid, fromstatus, tostatus, datetransitioned)
		SELECT * FROM unnest($1::integer[], $2::integer[], $3::text[], $4::text[], $5::TIMESTAMPTZ[])
		ON CONFLICT DO NOTHING`,
		ids, issueIDs, fromStatuses, toStatuses, dates)

	if err != nil {
		return err
	}
	return nil
}

// Refresh status stints will repopulate the current status state for a given issue (deletes and then inserts results)
func (s *Postgres) RefreshStatusStints(issueID int) error {
	_, err := s.DB.Exec(`
		WITH deleted AS (
			DELETE FROM status_stints WHERE issueid = $1
		),
		transitions AS (
			SELECT
				issueid,
				tostatus                                                                AS status,
				datetransitioned                                                        AS datestarted,
				LEAD(datetransitioned) OVER (ORDER BY datetransitioned)                AS dateended,
				EXTRACT(EPOCH FROM (
					LEAD(datetransitioned) OVER (ORDER BY datetransitioned) - datetransitioned
				))::bigint                                                              AS durationseconds
			FROM issue_transition
			WHERE issueid = $1
		)
		INSERT INTO status_stints (issueid, status, datestarted, dateended, durationseconds)
		SELECT issueid, status, datestarted, dateended, durationseconds
		FROM transitions`, issueID)
	return err
}

// projectIssuesCTE is the base CTE used by all ProjectKPIs queries to resolve the
// set of leaf issues belonging to a given epic key or fixed version.
const projectIssuesCTE = `
	WITH project_issues AS (
		SELECT i.id FROM issue i
		WHERE i.type NOT IN ('Epic', 'Release Candidate')
		AND ($1 = ANY(i.fixedversions) OR i.parentid IN (SELECT id FROM issue WHERE key = $1))
		AND i.type NOT IN ('Sub-task', 'Dev Sub-Task', 'QA Needed Sub-Task', 'Authoring Task')
		UNION
		SELECT i.id FROM issue i
		JOIN issue i2 ON i.parentid = i2.id
		WHERE i2.type NOT IN ('Release Candidate')
		AND ($1 = ANY(i2.fixedversions) OR i2.parentid IN (SELECT id FROM issue WHERE key = $1))
		AND i.type NOT IN ('Sub-task', 'Dev Sub-Task', 'QA Needed Sub-Task', 'Authoring Task')
	)`

const (
	// closeableTypes is the set of issue types that flow through Dev → QA and so
	// participate in cycle-time/throughput/flow-efficiency/defect-escape metrics.
	// Excludes Task (skips QA), Epic/Release Candidate (containers), and sub-tasks.
	closeableTypes = `('Story','Bug','Hardware Bug','Customer Bug','HW / FW Customer Bug')`

	// bucketCase maps a status-stints `status` column to its flow bucket
	// ('Dev'/'QA'/'Waiting'/'Other'). Mirrors the inline CASE expressions in
	// ProjectKPIs (lines ~672-677, ~686-691); future tasks may consolidate
	// those onto this constant.
	bucketCase = `
    CASE
      WHEN status IN ('In Development','In Progress','Code Complete','Code Merged','In Review') THEN 'Dev'
      WHEN status IN ('In QA','Failed QA') THEN 'QA'
      WHEN status IN ('On Hold','QA Backlog') THEN 'Waiting'
      ELSE 'Other'
    END`

	// devQAStatuses lists the statuses that count toward "active" cycle time —
	// Dev bucket + QA bucket. Used by cycle-time and flow-efficiency queries.
	devQAStatuses = `('In Development','In Progress','Code Complete','Code Merged','In Review','In QA','Failed QA')`

	// doneStatuses lists the statuses that mark an issue as closed.
	doneStatuses = `('Done','Closed','Cancelled','Awaiting Release to Customer')`

	// waitingStatuses lists the statuses that represent waiting time
	// (issue is open but not actively progressing in Dev or QA).
	waitingStatuses = `('On Hold','QA Backlog')`
)

// ProjectKPIs runs all KPI queries for the given epic key or fixed version and
// returns the aggregated results in a single ProjectKPIData struct.
func (s *Postgres) ProjectKPIs(epicOrVersion string) (types.ProjectKPIData, error) {
	data := types.ProjectKPIData{}

	// ── 1. Issue count ──────────────────────────────────────────────────────
	err := s.DB.Get(&data.IssueCount, projectIssuesCTE+`
		SELECT COUNT(*) FROM project_issues`, epicOrVersion)
	if err != nil || data.IssueCount == 0 {
		return data, err
	}

	// ── 2. Cycle time percentiles (active-work statuses only) ───────────────
	var ct struct {
		Avg    float64 `db:"avg_secs"`
		Median float64 `db:"median_secs"`
		P85    float64 `db:"p85_secs"`
		P95    float64 `db:"p95_secs"`
	}
	err = s.DB.Get(&ct, projectIssuesCTE+`
		SELECT
			COALESCE(AVG(total), 0)                                              AS avg_secs,
			COALESCE(PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY total), 0)    AS median_secs,
			COALESCE(PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY total), 0)    AS p85_secs,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY total), 0)    AS p95_secs
		FROM (
			SELECT issueid, SUM(EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted))) AS total
			FROM status_stints
			WHERE issueid IN (SELECT id FROM project_issues)
			AND status IN `+devQAStatuses+`
			GROUP BY issueid
		) ct`, epicOrVersion)
	if err != nil {
		return data, err
	}
	data.AvgCycleTimeSecs = ct.Avg
	data.MedianCycleTimeSecs = ct.Median
	data.P85CycleTimeSecs = ct.P85
	data.P95CycleTimeSecs = ct.P95

	// ── 3. Cycle time breakdown by status (with bucket label) ───────────────
	err = s.DB.Select(&data.CycleTimeByStatus, projectIssuesCTE+`
		SELECT
			status,
			`+bucketCase+` AS bucket,
			AVG(EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted))) AS avg_seconds,
			COUNT(DISTINCT issueid) AS issue_count
		FROM status_stints
		WHERE issueid IN (SELECT id FROM project_issues)
		AND status NOT IN ('Development Backlog', 'To Do', 'Bug Draft', 'Done','Closed','Cancelled', 'Awaiting Release to Customer')
		GROUP BY status
		ORDER BY
			CASE
				WHEN status IN ('In Development','In Progress','Code Complete','Code Merged', 'In Review') THEN 1
				WHEN status IN ('In QA','Failed QA') THEN 2
				WHEN status IN ('On Hold','QA Backlog') THEN 3
				--WHEN status IN ('Done','Closed','Cancelled', 'Awaiting Release to Customer') THEN 4
				ELSE 4
			END,
			avg_seconds DESC`, epicOrVersion)
	if err != nil {
		return data, err
	}

	// ── 3b. Flow bucket aggregation ──────────────────────────────────────────
	// Wrap in a subquery so GROUP BY can reference the computed `bucket` column.
	err = s.DB.Select(&data.FlowBuckets, projectIssuesCTE+`
		SELECT
			bucket,
			AVG(elapsed) AS avg_seconds,
			SUM(elapsed) AS total_seconds,
			COUNT(DISTINCT issueid) AS issue_count
		FROM (
			SELECT
				issueid,
				EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted)) AS elapsed,
				`+bucketCase+` AS bucket
			FROM status_stints
			WHERE issueid IN (SELECT id FROM project_issues)
			AND status NOT IN ('Development Backlog', 'To Do', 'Bug Draft', 'Done','Closed','Cancelled', 'Awaiting Release to Customer')
		) s
		GROUP BY bucket
		ORDER BY CASE bucket WHEN 'Dev' THEN 1 WHEN 'QA' THEN 2 WHEN 'Waiting' THEN 3 ELSE 4 END`,
		epicOrVersion)
	if err != nil {
		return data, err
	}

	// ── 4. Blocked time ─────────────────────────────────────────────────────
	var blocked struct {
		Avg   float64 `db:"avg_blocked"`
		Count int     `db:"blocked_count"`
	}
	err = s.DB.Get(&blocked, projectIssuesCTE+`
		SELECT
			COALESCE(AVG(total), 0)  AS avg_blocked,
			COUNT(*)                 AS blocked_count
		FROM (
			SELECT issueid, SUM(EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted))) AS total
			FROM status_stints
			WHERE issueid IN (SELECT id FROM project_issues)
			AND status in ('On Hold')
			GROUP BY issueid
		) b`, epicOrVersion)
	if err != nil {
		return data, err
	}
	data.AvgBlockedTimeSecs = blocked.Avg
	data.BlockedIssueCount = blocked.Count

	// ── 5. Status bounce / thrash rate ──────────────────────────────────────
	var bounce struct {
		Total      int `db:"total_bounces"`
		IssueCount int `db:"bounce_issue_count"`
	}
	err = s.DB.Get(&bounce, projectIssuesCTE+`
		SELECT
			COALESCE(SUM(re_entries), 0) AS total_bounces,
			COUNT(DISTINCT issueid)      AS bounce_issue_count
		FROM (
			SELECT issueid, tostatus, COUNT(*) - 1 AS re_entries
			FROM issue_transition
			WHERE issueid IN (SELECT id FROM project_issues)
			GROUP BY issueid, tostatus
			HAVING COUNT(*) > 1
		) b`, epicOrVersion)
	if err != nil {
		return data, err
	}
	data.TotalBounces = bounce.Total
	data.BounceIssueCount = bounce.IssueCount

	// ── 6. Flow efficiency ──────────────────────────────────────────────────
	var flow struct {
		ActiveSecs float64 `db:"active_secs"`
		TotalSecs  float64 `db:"total_secs"`
	}
	err = s.DB.Get(&flow, projectIssuesCTE+`
		SELECT
			SUM(EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted))) FILTER (
				WHERE status IN `+devQAStatuses+`
			) AS active_secs,
			-- Wider than devQAStatuses: active + waiting + done. Uses the same
			-- constants as MonthlyTeamMetrics.flow_efficiency to prevent drift.
			SUM(EXTRACT(EPOCH FROM (COALESCE(dateended, now()) - datestarted))) FILTER (
				WHERE status IN `+devQAStatuses+` OR status IN `+waitingStatuses+` OR status IN `+doneStatuses+`
			) AS total_secs
		FROM status_stints
		WHERE issueid IN (SELECT id FROM project_issues)`, epicOrVersion)
	if err != nil {
		return data, err
	}
	if flow.TotalSecs > 0 {
		data.FlowEfficiencyPct = (flow.ActiveSecs / flow.TotalSecs) * 100
	}

	//might just need to check for failed QA?? TODO
	// ── 7. QA rework cycles ─────────────────────────────────────────────────
	var qa struct {
		AvgCycles  float64 `db:"avg_cycles"`
		IssueCount int     `db:"issue_count"`
	}
	err = s.DB.Get(&qa, projectIssuesCTE+`
		SELECT
			COALESCE(AVG(qa_cycles), 0) AS avg_cycles,
			COUNT(*)                    AS issue_count
		FROM (
			SELECT issueid, COUNT(*) AS qa_cycles
			FROM issue_transition
			WHERE issueid IN (SELECT id FROM project_issues)
			AND (fromstatus ILIKE '%qa%' OR fromstatus ILIKE '%test%')
			AND (tostatus ILIKE '%dev%' OR tostatus = 'In Progress' OR tostatus = 'In Development')
			GROUP BY issueid
		) qac`, epicOrVersion)
	if err != nil {
		return data, err
	}
	data.AvgQACycles = qa.AvgCycles
	data.IssuesReturnedFromQA = qa.IssueCount

	// ── 8. WIP history (one row per day) ────────────────────────────────────
	err = s.DB.Select(&data.WIPHistory, projectIssuesCTE+`
		SELECT
			gs.day::date                AS day,
			COUNT(DISTINCT ss.issueid)  AS wip_count
		FROM generate_series(
			(SELECT MIN(datestarted)::date FROM status_stints WHERE issueid IN (SELECT id FROM project_issues)),
			NOW()::date,
			'1 day'::interval
		) gs(day)
		LEFT JOIN status_stints ss
			ON  ss.status IN `+devQAStatuses+`
			AND ss.datestarted::date <= gs.day::date
			AND (ss.dateended IS NULL OR ss.dateended::date > gs.day::date)
			AND ss.issueid IN (SELECT id FROM project_issues)
		GROUP BY gs.day
		ORDER BY gs.day`, epicOrVersion)
	if err != nil {
		return data, err
	}

	// ── 9. Burndown — two falling lines: remaining-in-dev and remaining-overall
	// dev_complete: first time an issue entered QA or beyond (dev is done with it)
	// fully_done:   first time an issue entered a done/closed status
	err = s.DB.Select(&data.BurndownHistory, projectIssuesCTE+`,
		dev_complete AS (
			SELECT issueid, MIN(datestarted) AS crossed_at
			FROM status_stints
			WHERE issueid IN (SELECT id FROM project_issues)
			AND status IN ('In QA','QA Backlog','Done','Closed','Cancelled')
			GROUP BY issueid
		),
		fully_done AS (
			SELECT issueid, MIN(datestarted) AS crossed_at
			FROM status_stints
			WHERE issueid IN (SELECT id FROM project_issues)
			AND status IN ('Done','Closed','Cancelled')
			GROUP BY issueid
		)
		SELECT
			gs.day::date AS day,
			COUNT(DISTINCT CASE WHEN dc.crossed_at::date <= gs.day THEN dc.issueid END) AS dev_complete_count,
			COUNT(DISTINCT CASE WHEN fd.crossed_at::date <= gs.day THEN fd.issueid END) AS fully_done_count
		FROM generate_series(
			(SELECT MIN(datestarted)::date FROM status_stints WHERE issueid IN (SELECT id FROM project_issues)),
			NOW()::date,
			'1 day'::interval
		) gs(day)
		LEFT JOIN dev_complete  dc ON true
		LEFT JOIN fully_done    fd ON true
		GROUP BY gs.day
		ORDER BY gs.day`, epicOrVersion)
	if err != nil {
		return data, err
	}

	// ── 10. Burn-up — completed hours vs total scope hours ──────────────────
	// completed_seconds:    cumulative timespent logged on project issues up to each day
	// total_scope_seconds:  cumulative originalestimate for all issues (incl. Bug / Hardware Bug = scope creep)
	// planned_scope_seconds: same but excluding Bug / Hardware Bug (original plan only)
	err = s.DB.Select(&data.BurnupHistory, projectIssuesCTE+`,
		daily_logged AS (
			SELECT w.date::date AS day, SUM(w.timespentseconds)::float AS seconds_logged
			FROM worklog w
			WHERE w.issueid IN (SELECT id FROM project_issues)
			GROUP BY w.date::date
		),
		daily_scope AS (
			SELECT
				i.createdate::date AS day,
				SUM(i.originalestimate)::float AS scope_added,
				SUM(CASE WHEN i.type NOT IN ('Bug','Hardware Bug') THEN i.originalestimate ELSE 0 END)::float AS planned_scope_added
			FROM issue i
			WHERE i.id IN (SELECT id FROM project_issues)
			GROUP BY i.createdate::date
		)
		SELECT
			gs.day::date AS day,
			SUM(COALESCE(dl.seconds_logged,       0)) OVER (ORDER BY gs.day) AS completed_seconds,
			SUM(COALESCE(ds.scope_added,          0)) OVER (ORDER BY gs.day) AS total_scope_seconds,
			SUM(COALESCE(ds.planned_scope_added,  0)) OVER (ORDER BY gs.day) AS planned_scope_seconds
		FROM generate_series(
			LEAST(
				(SELECT MIN(w.date)::date       FROM worklog w WHERE w.issueid IN (SELECT id FROM project_issues)),
				(SELECT MIN(i.createdate)::date FROM issue i   WHERE i.id      IN (SELECT id FROM project_issues))
			),
			NOW()::date,
			'1 day'::interval
		) gs(day)
		LEFT JOIN daily_logged dl ON dl.day = gs.day::date
		LEFT JOIN daily_scope   ds ON ds.day = gs.day::date
		ORDER BY gs.day`, epicOrVersion)
	if err != nil {
		return data, err
	}

	return data, nil
}

// MonthlyTeamMetrics — see repository/repo.go for contract.
func (s *Postgres) MonthlyTeamMetrics(ctx context.Context, teams []string, fromMonth, toMonth string) ([]types.MonthlyTeamMetrics, error) {
	result := []types.MonthlyTeamMetrics{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		team_list AS (
			SELECT unnest($1::text[]) AS team
		),
		issue_close_month AS (
			-- First time within the reporting window each closeable-type issue
			-- entered a done status. Date filter excludes historical closes so a
			-- reopened-and-reclosed issue is attributed to its in-window close
			-- month, not a years-old MIN that would fall outside the months CTE.
			SELECT
				ss.issueid,
				i.project,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN ` + doneStatuses + `
			AND i.type IN ` + closeableTypes + `
			AND i.project = ANY($1::text[])
			AND ss.datestarted >= $2::timestamptz
			AND ss.datestarted <  date_trunc('month', $3::timestamptz)
			GROUP BY ss.issueid, i.project
		),
		throughput AS (
			SELECT project AS team, year_month, COUNT(DISTINCT issueid) AS closed_issue_count
			FROM issue_close_month
			GROUP BY project, year_month
		),
		issue_cycle_secs AS (
			-- Per-issue total active (Dev+QA) seconds, joined to its close month.
			SELECT
				icm.project   AS team,
				icm.year_month,
				ss.issueid,
				SUM(ss.durationseconds)::float8 AS total_secs
			FROM status_stints ss
			JOIN issue_close_month icm ON icm.issueid = ss.issueid
			WHERE ss.status IN ` + devQAStatuses + `
			GROUP BY icm.project, icm.year_month, ss.issueid
		),
		cycle_time AS (
			SELECT
				team,
				year_month,
				PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY total_secs) AS median_cycle_time_secs
			FROM issue_cycle_secs
			GROUP BY team, year_month
		),
		flow_efficiency AS (
			SELECT
				icm.project   AS team,
				icm.year_month,
				CASE
					WHEN SUM(CASE WHEN ss.status IN ` + devQAStatuses + ` OR ss.status IN ` + waitingStatuses + ` OR ss.status IN ` + doneStatuses + ` THEN ss.durationseconds ELSE 0 END) > 0
					THEN
						100.0 * SUM(CASE WHEN ss.status IN ` + devQAStatuses + ` THEN ss.durationseconds ELSE 0 END)::float8
						/ SUM(CASE WHEN ss.status IN ` + devQAStatuses + ` OR ss.status IN ` + waitingStatuses + ` OR ss.status IN ` + doneStatuses + ` THEN ss.durationseconds ELSE 0 END)::float8
					ELSE 0
				END AS flow_efficiency_pct
			FROM status_stints ss
			JOIN issue_close_month icm ON icm.issueid = ss.issueid
			GROUP BY icm.project, icm.year_month
		),
		closed_stories AS (
			SELECT
				icm.project   AS team,
				icm.year_month,
				icm.issueid
			FROM issue_close_month icm
			JOIN issue i ON i.id = icm.issueid
			WHERE i.type = 'Story'
		),
		failed_qa_stories AS (
			SELECT DISTINCT cs.team, cs.year_month, cs.issueid
			FROM closed_stories cs
			JOIN issue_transition it ON it.issueid = cs.issueid
			WHERE it.tostatus = 'Failed QA'
		),
		failed_qa_ratio AS (
			SELECT
				cs.team,
				cs.year_month,
				CASE
					WHEN COUNT(DISTINCT cs.issueid) > 0
					THEN 100.0 * COUNT(DISTINCT fqs.issueid)::float8 / COUNT(DISTINCT cs.issueid)::float8
					ELSE 0
				END AS failed_qa_ratio_pct
			FROM closed_stories cs
			LEFT JOIN failed_qa_stories fqs
			  ON fqs.team = cs.team AND fqs.year_month = cs.year_month AND fqs.issueid = cs.issueid
			GROUP BY cs.team, cs.year_month
		),
		defect_escape AS (
			SELECT
				project AS team,
				to_char(createdate, 'YYYY-MM') AS year_month,
				COUNT(*)::int AS total_bugs_count,
				CASE
					WHEN COUNT(*) > 0
					THEN 100.0 *
						COUNT(*) FILTER (WHERE type IN ('Customer Bug','HW / FW Customer Bug'))::float8
						/ COUNT(*)::float8
					ELSE 0
				END AS defect_escape_rate_pct
			FROM issue
			WHERE project = ANY($1::text[])
			AND createdate >= $2::timestamptz
			AND createdate <  $3::timestamptz
			AND type IN ('Customer Bug','HW / FW Customer Bug','Bug','Hardware Bug')
			GROUP BY project, to_char(createdate, 'YYYY-MM')
		),
		stability_new AS (
			SELECT
				project AS team,
				to_char(createdate, 'YYYY-MM') AS year_month,
				COUNT(*) AS new_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND createdate >= $2::timestamptz
			AND createdate <  $3::timestamptz
			GROUP BY project, to_char(createdate, 'YYYY-MM')
		),
		stability_closed AS (
			SELECT
				project AS team,
				to_char(COALESCE(resolveddate, updatedate), 'YYYY-MM') AS year_month,
				COUNT(*) AS closed_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND (
				(resolveddate >= $2::timestamptz AND resolveddate < $3::timestamptz)
				OR (status LIKE 'Awaiting Release%' AND updatedate >= $2::timestamptz AND updatedate < $3::timestamptz)
			)
			GROUP BY project, to_char(COALESCE(resolveddate, updatedate), 'YYYY-MM')
		),
		stability_baseline AS (
			-- Open customer bugs at the start of the window, per team.
			SELECT
				project AS team,
				COUNT(*) AS open_count
			FROM issue
			WHERE type IN ('Customer Bug','HW / FW Customer Bug')
			AND project = ANY($1::text[])
			AND createdate < $2::timestamptz
			AND (resolveddate IS NULL OR resolveddate >= $2::timestamptz)
			GROUP BY project
		),
		stability AS (
			SELECT
				tl.team,
				m.year_month,
				COALESCE(sn.new_count, 0)    AS stability_new_count,
				COALESCE(sc.closed_count, 0) AS stability_closed_count,
				SUM(COALESCE(sn.new_count, 0) - COALESCE(sc.closed_count, 0)) OVER (
					PARTITION BY tl.team
					ORDER BY m.year_month
					ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
				) + COALESCE(sb.open_count, 0) AS stability_open_count
			FROM team_list tl
			CROSS JOIN months m
			LEFT JOIN stability_new    sn ON sn.team = tl.team AND sn.year_month = m.year_month
			LEFT JOIN stability_closed sc ON sc.team = tl.team AND sc.year_month = m.year_month
			LEFT JOIN stability_baseline sb ON sb.team = tl.team
		)
		SELECT
			tl.team,
			m.year_month,
			COALESCE(ct.median_cycle_time_secs, 0)::float8 AS median_cycle_time_secs,
			COALESCE(th.closed_issue_count, 0)   AS closed_issue_count,
			COALESCE(fe.flow_efficiency_pct, 0)::float8 AS flow_efficiency_pct,
			COALESCE(fqr.failed_qa_ratio_pct, 0)::float8 AS failed_qa_ratio_pct,
			COALESCE(de.defect_escape_rate_pct, 0)::float8 AS defect_escape_rate_pct,
			COALESCE(de.total_bugs_count, 0)               AS total_bugs_count,
			COALESCE(st.stability_new_count, 0)    AS stability_new_count,
			COALESCE(st.stability_closed_count, 0) AS stability_closed_count,
			COALESCE(st.stability_open_count, 0)   AS stability_open_count
		FROM team_list tl
		CROSS JOIN months m
		LEFT JOIN throughput th
		  ON th.team = tl.team AND th.year_month = m.year_month
		LEFT JOIN cycle_time ct
		  ON ct.team = tl.team AND ct.year_month = m.year_month
		LEFT JOIN flow_efficiency fe
		  ON fe.team = tl.team AND fe.year_month = m.year_month
		LEFT JOIN failed_qa_ratio fqr
		  ON fqr.team = tl.team AND fqr.year_month = m.year_month
		LEFT JOIN defect_escape de
		  ON de.team = tl.team AND de.year_month = m.year_month
		LEFT JOIN stability st
		  ON st.team = tl.team AND st.year_month = m.year_month
		ORDER BY tl.team, m.year_month`

	err := s.DB.SelectContext(ctx, &result, query, teams, from, to)
	return result, err
}

// ManagerMetrics — see repository/repo.go for contract.
//
// The 8 sub-queries (MonthlyTeamMetrics + 7 manager-only) are independent —
// none reads what another writes — so we run them concurrently via
// errgroup. Each writes to a different field of `data`, which is safe in
// Go (no shared memory location). errgroup returns the first error if any
// sub-query fails, and waits for all in-flight ones to finish.
func (s *Postgres) ManagerMetrics(ctx context.Context, team string, fromMonth, toMonth string) (types.ManagerMetricsData, error) {
	data := types.ManagerMetricsData{Team: team}

	// errgroup.WithContext gives each sub-query a derived context that is
	// cancelled when either the parent (HTTP request) is cancelled or when
	// any sibling sub-query fails. Postgres receives the cancellation via
	// SelectContext and aborts the in-flight query mid-execution — so a
	// user toggling the team selector quickly does not leave orphan
	// queries piling up in the DB.
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		rows, err := s.MonthlyTeamMetrics(gctx, []string{team}, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.LeadershipMonthly = rows
		return nil
	})
	g.Go(func() error {
		v, err := s.timeInStatus(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.TimeInStatus = v
		return nil
	})
	g.Go(func() error {
		v, err := s.wipSeries(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.WIPSeries = v
		return nil
	})
	g.Go(func() error {
		v, err := s.agingWIP(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.AgingWIPItems = v
		return nil
	})
	g.Go(func() error {
		v, err := s.reworkCycles(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.ReworkCycles = v
		return nil
	})
	g.Go(func() error {
		v, err := s.statusBounce(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.StatusBounce = v
		return nil
	})
	g.Go(func() error {
		v, err := s.qaVsEngHours(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.QAvsEngHours = v
		return nil
	})
	g.Go(func() error {
		v, err := s.bugVsForwardHours(gctx, team, fromMonth, toMonth)
		if err != nil {
			return err
		}
		data.BugvsForwardHours = v
		return nil
	})

	if err := g.Wait(); err != nil {
		return data, err
	}
	return data, nil
}

// timeInStatus returns one row per (month status was exited, bucket) for the
// team's closeable-type issues. Used by the Time-in-Status stacked-bar chart.
func (s *Postgres) timeInStatus(ctx context.Context, team string, fromMonth, toMonth string) ([]types.TimeInStatusPoint, error) {
	result := []types.TimeInStatusPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	// Inline CASE mirrors bucketCase but uses the ss.status qualifier — the
	// issue table also has a `status` column, so bucketCase's unqualified
	// reference would be ambiguous here. The WHERE clause restricts to the
	// Dev/QA/Waiting statuses so an 'Other' bucket is impossible. Upper
	// bound is clamped to the start of the current month so the partial
	// in-progress month is excluded (matches MonthlyTeamMetrics' months CTE).
	query := `
		SELECT
			to_char(COALESCE(ss.dateended, NOW()), 'YYYY-MM') AS year_month,
			CASE
				WHEN ss.status IN ('In Development','In Progress','Code Complete','Code Merged','In Review') THEN 'Dev'
				WHEN ss.status IN ('In QA','Failed QA') THEN 'QA'
				WHEN ss.status IN ` + waitingStatuses + ` THEN 'Waiting'
				ELSE 'Other'
			END AS bucket,
			AVG(ss.durationseconds)::float8 AS avg_seconds
		FROM status_stints ss
		JOIN issue i ON i.id = ss.issueid
		WHERE i.project = $1
		AND i.type IN ` + closeableTypes + `
		AND COALESCE(ss.dateended, NOW()) >= $2::timestamptz
		AND COALESCE(ss.dateended, NOW()) <  date_trunc('month', $3::timestamptz)
		AND (
			ss.status IN ` + devQAStatuses + `
			OR ss.status IN ` + waitingStatuses + `
		)
		GROUP BY year_month, bucket
		ORDER BY year_month, bucket`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// wipSeries returns one row per day across the 13-month window — the count
// of the team's closeable-type issues currently in a Dev or QA status on
// that day. Used by the WIP-trend line chart.
func (s *Postgres) wipSeries(ctx context.Context, team string, fromMonth, toMonth string) ([]types.WIPPoint, error) {
	result := []types.WIPPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH days AS (
			SELECT generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 day'::interval
			)::date AS day
		),
		team_issues AS (
			SELECT id FROM issue
			WHERE project = $1
			AND type IN ` + closeableTypes + `
		)
		SELECT
			to_char(d.day, 'YYYY-MM-DD') AS day,
			COUNT(DISTINCT ss.issueid) AS wip_count
		FROM days d
		LEFT JOIN status_stints ss
			ON ss.issueid IN (SELECT id FROM team_issues)
			AND ss.status IN ` + devQAStatuses + `
			AND ss.datestarted::date <= d.day
			AND (ss.dateended IS NULL OR ss.dateended::date > d.day)
		GROUP BY d.day
		ORDER BY d.day`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// agingWIP returns the team's currently open issues in Dev or QA whose
// time-in-current-status exceeds the team's 85th-percentile single-status
// duration (i.e., longer than a typical residence in one bucket). Compared
// to a total-cycle p85, this is the right unit-of-comparison: time spent in
// the current status vs. typical time spent in any one status. Returns at
// most 20 rows, sorted by current_secs DESC.
func (s *Postgres) agingWIP(ctx context.Context, team string, fromMonth, toMonth string) ([]types.AgingWIPItem, error) {
	result := []types.AgingWIPItem{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH p85_status AS (
			-- 85th-percentile single-status duration across the team's
			-- closed-issue stints in the window. Closed stints (dateended
			-- IS NOT NULL) have a meaningful durationseconds.
			SELECT COALESCE(
				PERCENTILE_CONT(0.85) WITHIN GROUP (ORDER BY ss.durationseconds::float8),
				7 * 86400  -- fallback if no historical stints: 7 days
			) AS p85_secs
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			JOIN (
				SELECT ss2.issueid, MIN(ss2.datestarted) AS close_at
				FROM status_stints ss2
				JOIN issue i2 ON i2.id = ss2.issueid
				WHERE ss2.status IN ` + doneStatuses + `
				AND i2.project = $1
				AND i2.type IN ` + closeableTypes + `
				AND ss2.datestarted >= $2::timestamptz
				AND ss2.datestarted <  date_trunc('month', $3::timestamptz)
				GROUP BY ss2.issueid
			) closed ON closed.issueid = ss.issueid
			WHERE ss.status IN ` + devQAStatuses + `
			AND i.project = $1
			AND i.type IN ` + closeableTypes + `
			AND ss.durationseconds IS NOT NULL
		),
		current_stints AS (
			-- Issues that are currently in a Dev/QA status (dateended IS NULL)
			SELECT
				i.key,
				ss.status,
				EXTRACT(EPOCH FROM (NOW() - ss.datestarted))::bigint AS current_secs
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.dateended IS NULL
			AND ss.status IN ` + devQAStatuses + `
			AND i.project = $1
			AND i.type IN ` + closeableTypes + `
		)
		SELECT
			cs.key,
			cs.status,
			(cs.current_secs / 86400)::int AS days_in_status
		FROM current_stints cs
		CROSS JOIN p85_status p
		WHERE cs.current_secs > p.p85_secs
		ORDER BY cs.current_secs DESC
		LIMIT 20`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// reworkCycles returns the average number of QA→Dev transitions per closed
// issue, bucketed by close month. Reuses the pattern from ProjectKPIs.
func (s *Postgres) reworkCycles(ctx context.Context, team string, fromMonth, toMonth string) ([]types.ReworkCyclesPoint, error) {
	result := []types.ReworkCyclesPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	// avg_cycles is per closed issue (denominator = COUNT(*) of all closed
	// issues in the month, NOT only the ones that had rework). Using SQL AVG
	// on the LEFT JOIN result would skip NULLs and inflate the metric.
	// reworked_issue_count is the count of issues with at least one rework
	// cycle — separate from total closed.
	// Both the close month and the rework transitions themselves are clamped
	// to the reporting window; otherwise pre-window thrash would be credited
	// to the close month, and the current partial month would leak into the
	// trailing slice element (which currentMgrRework reads as "current").
	query := `
		WITH issue_close_month AS (
			SELECT
				ss.issueid,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN ` + doneStatuses + `
			AND i.project = $1
			AND i.type IN ` + closeableTypes + `
			AND ss.datestarted >= $2::timestamptz
			AND ss.datestarted <  date_trunc('month', $3::timestamptz)
			GROUP BY ss.issueid
		),
		qa_to_dev AS (
			SELECT
				icm.year_month,
				icm.issueid,
				COUNT(*) AS cycles
			FROM issue_close_month icm
			JOIN issue_transition it ON it.issueid = icm.issueid
			WHERE (it.fromstatus ILIKE '%qa%' OR it.fromstatus ILIKE '%test%')
			AND (it.tostatus ILIKE '%dev%' OR it.tostatus = 'In Progress' OR it.tostatus = 'In Development')
			AND it.datetransitioned >= $2::timestamptz
			AND it.datetransitioned <  date_trunc('month', $3::timestamptz)
			GROUP BY icm.year_month, icm.issueid
		)
		SELECT
			icm.year_month,
			COALESCE(SUM(COALESCE(qtd.cycles, 0))::float8 / NULLIF(COUNT(*)::float8, 0), 0) AS avg_cycles,
			COUNT(DISTINCT qtd.issueid) AS reworked_issue_count
		FROM issue_close_month icm
		LEFT JOIN qa_to_dev qtd ON qtd.issueid = icm.issueid
		GROUP BY icm.year_month
		ORDER BY icm.year_month`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// statusBounce returns monthly bounce rate (status re-entries / total
// transitions) for the team's closed issues.
func (s *Postgres) statusBounce(ctx context.Context, team string, fromMonth, toMonth string) ([]types.BouncePoint, error) {
	result := []types.BouncePoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	// total_entries is the sum of status_count across (issue, tostatus)
	// pairs — i.e., the total number of transitions for the closed issues
	// in the month. Earlier this used COUNT(*) of the grouped rows, which
	// counts (issue, status) pairs and overstates bounce rate ~2x.
	// Close month and transitions are both clamped to the reporting window
	// so a long-lived issue doesn't carry pre-window thrash into its close
	// month, and the current partial month doesn't leak.
	query := `
		WITH issue_close_month AS (
			SELECT
				ss.issueid,
				to_char(MIN(ss.datestarted), 'YYYY-MM') AS year_month
			FROM status_stints ss
			JOIN issue i ON i.id = ss.issueid
			WHERE ss.status IN ` + doneStatuses + `
			AND i.project = $1
			AND i.type IN ` + closeableTypes + `
			AND ss.datestarted >= $2::timestamptz
			AND ss.datestarted <  date_trunc('month', $3::timestamptz)
			GROUP BY ss.issueid
		),
		per_month AS (
			SELECT
				icm.year_month,
				SUM(tx.status_count) AS total_entries,
				SUM(CASE WHEN tx.status_count > 1 THEN tx.status_count - 1 ELSE 0 END) AS re_entries,
				COUNT(DISTINCT icm.issueid) AS total_issues
			FROM issue_close_month icm
			JOIN (
				SELECT issueid, tostatus, COUNT(*) AS status_count
				FROM issue_transition
				WHERE datetransitioned >= $2::timestamptz
				AND datetransitioned <  date_trunc('month', $3::timestamptz)
				GROUP BY issueid, tostatus
			) tx ON tx.issueid = icm.issueid
			GROUP BY icm.year_month
		)
		SELECT
			year_month,
			CASE
				WHEN total_entries > 0
				THEN 100.0 * re_entries::float8 / total_entries::float8
				ELSE 0
			END AS bounce_pct,
			total_issues
		FROM per_month
		ORDER BY year_month`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// qaVsEngHours returns the monthly ratio of QA hours to Eng hours for the
// team's issues (worklog.date month bucketing). Numerator = QA-role hours,
// Denominator = Dev-role hours.
func (s *Postgres) qaVsEngHours(ctx context.Context, team string, fromMonth, toMonth string) ([]types.HoursRatioPoint, error) {
	result := []types.HoursRatioPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		hours AS (
			SELECT
				to_char(w.date, 'YYYY-MM') AS year_month,
				SUM(CASE WHEN p.role = 'qa'  THEN w.timespenthours ELSE 0 END)::float8 AS qa_hours,
				SUM(CASE WHEN p.role = 'dev' THEN w.timespenthours ELSE 0 END)::float8 AS dev_hours
			FROM worklog w
			JOIN issue i  ON i.id = w.issueid
			LEFT JOIN people p ON p.name = w.author
			WHERE i.project = $1
			AND w.date >= $2::timestamptz
			AND w.date <  $3::timestamptz
			GROUP BY to_char(w.date, 'YYYY-MM')
		)
		SELECT
			m.year_month,
			COALESCE(h.qa_hours, 0)  AS numerator_hours,
			COALESCE(h.dev_hours, 0) AS denominator_hours,
			CASE
				WHEN COALESCE(h.dev_hours, 0) > 0
				THEN h.qa_hours / h.dev_hours
				ELSE 0
			END AS ratio
		FROM months m
		LEFT JOIN hours h ON h.year_month = m.year_month
		ORDER BY m.year_month`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
}

// bugVsForwardHours returns the monthly ratio of bug-fix hours to forward-
// work (Story + Task) hours for the team. Bucketed by worklog.date month.
// Numerator = Bug + Hardware Bug hours. Denominator = Story + Task hours.
func (s *Postgres) bugVsForwardHours(ctx context.Context, team string, fromMonth, toMonth string) ([]types.HoursRatioPoint, error) {
	result := []types.HoursRatioPoint{}
	from, to := s.calculateDateRange(fromMonth, toMonth)

	query := `
		WITH months AS (
			SELECT to_char(gs, 'YYYY-MM') AS year_month
			FROM generate_series(
				date_trunc('month', $2::timestamptz),
				date_trunc('month', $3::timestamptz) - INTERVAL '1 day',
				'1 month'::interval
			) gs
		),
		hours AS (
			SELECT
				to_char(w.date, 'YYYY-MM') AS year_month,
				SUM(CASE WHEN i.type IN ('Bug','Hardware Bug')  THEN w.timespenthours ELSE 0 END)::float8 AS bug_hours,
				SUM(CASE WHEN i.type IN ('Story','Task')        THEN w.timespenthours ELSE 0 END)::float8 AS fwd_hours
			FROM worklog w
			JOIN issue i ON i.id = w.issueid
			WHERE i.project = $1
			AND w.date >= $2::timestamptz
			AND w.date <  $3::timestamptz
			GROUP BY to_char(w.date, 'YYYY-MM')
		)
		SELECT
			m.year_month,
			COALESCE(h.bug_hours, 0) AS numerator_hours,
			COALESCE(h.fwd_hours, 0) AS denominator_hours,
			CASE
				WHEN COALESCE(h.fwd_hours, 0) > 0
				THEN h.bug_hours / h.fwd_hours
				ELSE 0
			END AS ratio
		FROM months m
		LEFT JOIN hours h ON h.year_month = m.year_month
		ORDER BY m.year_month`

	err := s.DB.SelectContext(ctx, &result, query, team, from, to)
	return result, err
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
