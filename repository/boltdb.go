package repository

import (
	"math"
	"strconv"
	"time"

	"github.com/mkobaly/jiraworklog/types"
	"github.com/timshannon/bolthold"
)

//BoltDB is a repository with BoltDB as the backend
type BoltDB struct {
	//db *storm.DB
	db *bolthold.Store
}

type Config struct {
	LastTimestamp int64
	MaxWorklogID  int
}

//NewBoltDBRepo will create a new BoltDB repository
func NewBoltDBRepo(dbfile string) (*BoltDB, error) {
	db, err := bolthold.Open(dbfile, 0666, nil)
	if err != nil {
		return nil, err
	}

	bolt := &BoltDB{
		db: db,
	}

	if mwi, err := bolt.WorklogGetMaxWorklogID(); mwi == 0 {
		err = bolt.WorklogUpdateMaxWorklogID(0)
		if err != nil {
			return nil, err
		}
	}
	if ts, err := bolt.WorklogGetLastTimestamp(); ts == 0 {
		err = bolt.WorklogUpdateLastTimestamp(0)
		if err != nil {
			return nil, err
		}
	}

	return bolt, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (r *BoltDB) NonResolvedIssues() ([]types.ParentIssue, error) {
	var issues []types.ParentIssue
	err := r.db.Find(&issues, bolthold.Where("IsResolved").Eq(false))
	return issues, err
}

//Write will add the worklogItem to BoltDB
func (r *BoltDB) Write(wi *types.WorklogItem, pi *types.ParentIssue) error {

	err := r.db.Upsert(wi.ID, &wi)
	if err != nil {
		return err
	}
	pi.UpdateDate = wi.Date
	//parent := wi.GetParent()
	err = r.db.Upsert(pi.ID, &pi)
	return err
}

func (r *BoltDB) WorklogGetLastTimestamp() (int64, error) {
	var c Config
	err := r.db.Get("lastTimestamp", &c)
	return c.LastTimestamp, err
}

func (r *BoltDB) WorklogUpdateLastTimestamp(lastTimestamp int64) error {
	var c Config
	c.LastTimestamp = lastTimestamp
	err := r.db.Upsert("lastTimestamp", c)
	return err
}

func (r *BoltDB) WorklogGetMaxWorklogID() (int, error) {
	var c Config
	err := r.db.Get("maxWorklogID", &c)
	return c.MaxWorklogID, err
}

func (r *BoltDB) WorklogUpdateMaxWorklogID(maxWorklogID int) error {
	var c Config
	c.MaxWorklogID = maxWorklogID
	err := r.db.Upsert("maxWorklogID", c)
	return err
}

//UpdateIssue will update the resolved information for the given issue
func (r *BoltDB) UpdateIssue(issue *types.ParentIssue) error {
	return r.db.Upsert(issue.ID, &issue)
}

//Close will close the boltDB connection
func (r *BoltDB) Close() {
	r.db.Close()
}

//AllWorkLogs will return all of the work logs from boltDB
func (r *BoltDB) AllWorkLogs() ([]types.WorklogItem, error) {
	var worklogs []types.WorklogItem
	err := r.db.Find(&worklogs, nil)
	return worklogs, err
}

//AllIssues will return all of the issues from boltDB
func (r *BoltDB) AllIssues() ([]types.ParentIssue, error) {
	var issues []types.ParentIssue
	err := r.db.Find(&issues, nil)
	return issues, err
}

//IssuesGroupedBy will return issues group by the given groupBy value going
//back weeksBack. This data will be used for charting
func (r *BoltDB) IssuesGroupedBy(groupBy string, start time.Time, stop time.Time) ([]types.IssueChartData, error) {
	//y, m, d := time.Now().AddDate(0, 0, -1*weeksBack).Date()
	//date := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	query := bolthold.Where("UpdateDate").Ge(start).And("UpdateDate").Le(stop)
	agg, err := r.db.FindAggregate(&types.ParentIssue{}, query, groupBy, "IsResolved")
	if err != nil {
		return nil, err
	}

	results := make(map[string]*types.IssueChartData)
	for i := range agg {
		var group string
		var isResolved bool
		agg[i].Group(&group, &isResolved)

		_, ok := results[group]
		if !ok {
			results[group] = &types.IssueChartData{GroupBy: group}
		}

		if isResolved {
			results[group].Resolved = agg[i].Count()
			results[group].DaysToResolve = int(agg[i].Avg("DaysToResolve"))
			results[group].TimeSpent = agg[i].Sum("AggregateTimeSpent") / 3600
			results[group].TimeEstimate = agg[i].Sum("AggregateTimeOriginalEstimate") / 3600
		} else {
			results[group].NonResolved = agg[i].Count()
		}
	}

	values := make([]types.IssueChartData, 0, len(results))
	for _, v := range results {
		values = append(values, *v)
	}
	return values, nil
}

func (r *BoltDB) IssueAccuracy(start time.Time, stop time.Time) ([]types.IssueAccuracy, error) {
	results := []types.IssueAccuracy{}

	query := bolthold.Where("UpdateDate").Ge(start).And("UpdateDate").Le(stop).And("IsResolved").Eq(true)
	agg, err := r.db.FindAggregate(&types.ParentIssue{}, query, "Developer")
	if err != nil {
		return nil, err
	}

	for i := range agg {
		var developer string
		agg[i].Group(&developer)
		timeSpent := agg[i].Sum("AggregateTimeSpent")
		origEstimate := agg[i].Sum("AggregateTimeOriginalEstimate")
		accuracy := 100 - math.Abs(((origEstimate-timeSpent)/origEstimate)*100.00)
		count := agg[i].Count()
		results = append(results, types.IssueAccuracy{Developer: developer, Count: count, Accuracy: math.Round(accuracy*100) / 100})
	}

	return results, nil
}

func (r *BoltDB) WorklogsGroupBy(groupBy string, start time.Time, stop time.Time) ([]types.WorklogGroupByChart, error) {

	//start := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -7)
	//_, week := start.ISOWeek()

	query := bolthold.Where("Date").Ge(start).And("Date").Le(stop)
	agg, err := r.db.FindAggregate(&types.WorklogItem{}, query, groupBy)
	if err != nil {
		return nil, err
	}

	results := []types.WorklogGroupByChart{}
	for i := range agg {
		var group string
		agg[i].Group(&group)
		hours := math.Round(agg[i].Sum("TimeSpentHours")*100) / 100
		results = append(results, types.WorklogGroupByChart{GroupBy: group, TimeSpentHrs: hours})
	}
	return results, nil
}

func (r *BoltDB) WorklogsPerDev(start time.Time, stop time.Time) ([]map[string]string, error) {
	final := []map[string]string{}
	authors := make(map[string]bool)
	activity := make(map[types.DeveloperDateKey]float64)

	query := bolthold.Where("Date").Ge(start).And("Date").Le(stop)
	agg, err := r.db.FindAggregate(&types.WorklogItem{}, query, "Author", "Date")
	if err != nil {
		return nil, err
	}

	for i := range agg {
		var author string
		var date time.Time
		agg[i].Group(&author, &date)
		hours := math.Round(agg[i].Sum("TimeSpentHours")*100) / 100

		date.YearDay()
		if _, ok := authors[author]; !ok {
			authors[author] = true
		}
		key := types.DeveloperDateKey{Date: date.YearDay(), Developer: author}
		activity[key] += hours
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

func (r *BoltDB) WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error) {
	results := make(map[string]*types.WorklogsPerDevWeek)
	final := []types.WorklogsPerDevWeek{}

	start := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -35)
	_, week := time.Now().ISOWeek()

	query := bolthold.Where("Date").Ge(start)
	agg, err := r.db.FindAggregate(&types.WorklogItem{}, query, "Author", "WeekNumber")
	if err != nil {
		return nil, err
	}

	for i := range agg {
		var author string
		var weekNumber int
		agg[i].Group(&author, &weekNumber)
		hours := math.Round(agg[i].Sum("TimeSpentHours")*100) / 100

		if _, ok := results[author]; !ok {
			results[author] = &types.WorklogsPerDevWeek{Developer: author}
		}

		switch week - weekNumber {
		case 0:
			results[author].ThisWeek = hours
		case 1:
			results[author].LastWeek = hours
		case 2:
			results[author].TwoWeeks = hours
		case 3:
			results[author].ThreeWeeks = hours
		case 4:
			results[author].FourWeeks = hours
		}
	}

	for _, v := range results {
		final = append(final, *v)
	}
	return final, nil
}

// func (r *BoltDB) WorklogsPerDay() ([]types.WorklogsPerDay, error) {
// 	finalResults := []types.WorklogsPerDay{
// 		types.WorklogsPerDay{Day: "Sunday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Monday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Tuesday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Wednesday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Thursday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Friday", TimeSpentHrs: 0},
// 		types.WorklogsPerDay{Day: "Saturday", TimeSpentHrs: 0},
// 	}

// 	weekAgo := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -7)
// 	_, week := weekAgo.ISOWeek()

// 	query := bolthold.Where("Date").Ge(weekAgo.AddDate(0, 0, -5)).And("WeekNumber").Eq(week)
// 	agg, err := r.db.FindAggregate(&types.WorklogItem{}, query, "WeekDay")
// 	if err != nil {
// 		return nil, err
// 	}

// 	for i := range agg {
// 		var weekDay string
// 		agg[i].Group(&weekDay)
// 		hours := math.Round(agg[i].Sum("TimeSpentHours")*100) / 100

// 		for j := range finalResults {
// 			if strings.ToLower(finalResults[j].Day) == strings.ToLower(weekDay) {
// 				finalResults[i].TimeSpentHrs = hours
// 				continue
// 			}
// 		}
// 	}
// 	return finalResults, nil
// }

// func (r *BoltDB) WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error) {

// 	results := make(map[string]*types.WorklogsPerDevDay)
// 	final := []types.WorklogsPerDevDay{}

// 	start := time.Now().Truncate(24*time.Hour).AddDate(0, 0, -7)
// 	_, week := start.ISOWeek()

// 	query := bolthold.Where("Date").Ge(start.AddDate(0, 0, -5)).And("WeekNumber").Eq(week)
// 	agg, err := r.db.FindAggregate(&types.WorklogItem{}, query, "Author", "WeekDay")
// 	if err != nil {
// 		return nil, err
// 	}

// 	for i := range agg {
// 		var author string
// 		var weekDay string
// 		agg[i].Group(&author, &weekDay)
// 		hours := math.Round(agg[i].Sum("TimeSpentHours")*100) / 100

// 		if _, ok := results[author]; !ok {
// 			results[author] = &types.WorklogsPerDevDay{Developer: author}
// 		}

// 		switch weekDay {
// 		case "Monday":
// 			results[author].Monday = hours
// 		case "Tuesday":
// 			results[author].Tuesday = hours
// 		case "Wednesday":
// 			results[author].Wednesday = hours
// 		case "Thursday":
// 			results[author].Thursday = hours
// 		case "Friday":
// 			results[author].Friday = hours
// 		}

// 	}
// 	for _, v := range results {
// 		final = append(final, *v)
// 	}
// 	return final, nil
// }
