package repository

import (
	"time"

	"github.com/mkobaly/jiraworklog/types"
	"github.com/timshannon/bolthold"
)

//BoltDB is a repository with BoltDB as the backend
type BoltDB struct {
	//db *storm.DB
	db *bolthold.Store
}

//NewBoltDBRepo will create a new BoltDB repository
func NewBoltDBRepo(dbfile string) (*BoltDB, error) {
	db, err := bolthold.Open(dbfile, 0666, nil)
	if err != nil {
		return nil, err
	}
	return &BoltDB{
		db: db,
	}, nil
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
	return nil, nil
}

func (r *BoltDB) WorklogsGroupBy(groupBy string) ([]types.WorklogGroupByChart, error) {
	return nil, nil
}

func (r *BoltDB) WorklogsPerDay() ([]types.WorklogsPerDay, error) {
	return nil, nil
}

func (r *BoltDB) WorklogsPerDevDay() ([]types.WorklogsPerDevDay, error) {
	return nil, nil
}

func (r *BoltDB) WorklogsPerDevWeek() ([]types.WorklogsPerDevWeek, error) {
	return nil, nil
}
