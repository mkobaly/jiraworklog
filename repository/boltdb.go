package repository

import (
	"github.com/mkobaly/jiraworklog/types"
	"github.com/timshannon/bolthold"
	"time"
)

//BoltDB is a repository with BoltDB as the backend
type BoltDB struct {
	//db *storm.DB
	db *bolthold.Store
}

//NewBoltDBRepo will create a new BoltDB repository
func NewBoltDBRepo(dbfile string) (*BoltDB, error) {
	//db, err := storm.Open(dbfile)
	db, err := bolthold.Open(dbfile, 0666, nil)
	if err != nil {
		return nil, err
		//panic("Unable to open boltdb")
	}

	// err = db.Init(&types.ParentIssue{})
	// if err != nil {
	// 	return nil, err
	// 	//panic("Unable to initialize bucket")
	// }

	// err = db.Init(&types.WorklogItem{})
	// if err != nil {
	// 	return nil, err
	// 	//panic("Unable to initialize bucket")
	// }
	return &BoltDB{
		db: db,
	}, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (r *BoltDB) NonResolvedIssues() ([]types.ParentIssue, error) {
	var issues []types.ParentIssue
	//var results []string

	err := r.db.Find(&issues, bolthold.Where("IsResolved").Eq(false))
	return issues, err

	// if err != nil && err.Error() != "not found" {
	// 	return nil, err
	// }
	// // query := w.db.Select(q.Eq("IsResolved", false))
	// // err := query.Find(&issues)
	// // if err != nil && err.Error() != "not found" {
	// // 	return nil, err
	// // }
	// for _, i := range issues {
	// 	results = append(results, i.Key)
	// }
	// return results, nil
}

//Write will add the worklogItem to BoltDB
func (r *BoltDB) Write(wi *types.WorklogItem, pi *types.ParentIssue) error {

	err := r.db.Upsert(wi.ID, &wi)
	if err != nil {
		return err
	}
	//parent := wi.GetParent()
	err = r.db.Upsert(pi.ID, &pi)
	return err

	// tx, err := w.db.Begin(true)
	// err = tx.Save(&wi)
	// if err != nil {
	// 	return err
	// }
	// err = tx.Save(wi.GetParent())
	// if err != nil {
	// 	return err
	// }
	// return tx.Commit()
}

func (w *BoltDB) UpdateIssue(issue *types.ParentIssue) error {
	return w.db.Upsert(issue.ID, &issue)

	// err := w.db.Update(&types.ParentIssue{
	// 	ID:                            issue.ID,
	// 	ResolvedDate:                  issue.ResolvedDate,
	// 	IsResolved:                    true,
	// 	AggregateTimeOriginalEstimate: issue.AggregateTimeOriginalEstimate,
	// 	AggregateTimeSpent:            issue.AggregateTimeSpent,
	// })
	// return err
}

//Close will close the boltDB connection
func (r *BoltDB) Close() {
	r.db.Close()
}

func (r *BoltDB) AllWorkLogs() ([]types.WorklogItem, error) {
	var worklogs []types.WorklogItem
	// err := w.db.All(&worklogs)
	err := r.db.Find(&worklogs, nil)
	return worklogs, err
}

func (r *BoltDB) AllIssues() ([]types.ParentIssue, error) {
	var issues []types.ParentIssue
	err := r.db.Find(&issues, nil)
	//err := w.db.All(&issues)
	return issues, err
}

// func (r *BoltDB) IssueCountByDeveloper(daysBack int) (map[string]int, error) {
// 	results := make(map[string]int)
// 	y, m, d := time.Now().AddDate(0, 0, -1*daysBack).Date()
// 	date := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
// 	query := bolthold.Where("CreateDate").Ge(date)
// 	agg, err := r.db.FindAggregate(&types.ParentIssue{}, query, "Developer")
// 	if err != nil {
// 		return nil, err
// 	}
// 	for i := range agg {
// 		var developer string
// 		agg[i].Group(&developer)
// 		results[developer] = agg[i].Count()
// 	}
// 	return results, nil
// }

func (r *BoltDB) IssuesGroupedBy(groupBy string, daysBack int) ([]types.IssueChartData, error) {
	y, m, d := time.Now().AddDate(0, 0, -1*daysBack).Date()
	date := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	query := bolthold.Where("CreateDate").Ge(date)
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
			results[group].DaysToComplete = int(agg[i].Avg("DaysToResolve"))
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
