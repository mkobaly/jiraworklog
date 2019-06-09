package repository

import (
	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/mkobaly/jiraworklog/types"
	"time"
)

//BoltDB is a writer with BoltDB as the database
type BoltDB struct {
	db *storm.DB
}

//NewBoltDBRepo will create a new BoltDB writer
func NewBoltDBRepo(dbfile string) (*BoltDB, error) {
	db, err := storm.Open(dbfile)
	if err != nil {
		return nil, err
		//panic("Unable to open boltdb")
	}

	err = db.Init(&types.ParentIssue{})
	if err != nil {
		return nil, err
		//panic("Unable to initialize bucket")
	}

	err = db.Init(&types.WorklogItem{})
	if err != nil {
		return nil, err
		//panic("Unable to initialize bucket")
	}
	return &BoltDB{
		db: db,
	}, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (w *BoltDB) NonResolvedIssues() ([]string, error) {
	var issues []types.ParentIssue
	var results []string
	query := w.db.Select(q.Eq("IsResolved", false))
	err := query.Find(&issues)
	if err != nil && err.Error() != "not found" {
		return nil, err
	}
	for _, i := range issues {
		results = append(results, i.Key)
	}
	return results, nil
}

//Write will add the worklogItem to BoltDB
func (w *BoltDB) Write(wi types.WorklogItem) error {
	tx, err := w.db.Begin(true)
	err = tx.Save(&wi)
	if err != nil {
		return err
	}
	err = tx.Save(wi.GetParent())
	if err != nil {
		return err
	}
	return tx.Commit()
}

//UpdateResolutionDate will update the resolution date for a jira issue
func (w *BoltDB) UpdateResolutionDate(issueKey string, resolvedDate time.Time) error {
	var issue types.ParentIssue
	err := w.db.One("Key", issueKey, &issue)
	if err != nil {
		return err
	}
	err = w.db.Update(&types.ParentIssue{ID: issue.ID, ResolvedDate: &resolvedDate, IsResolved: true})
	if err != nil {
		return err
	}

	// query := w.db.Select(q.Eq("IssueKey", issueKey))
	// query.Each(new(types.ParentIssue), func(record interface{}) error) {

	// }
	return nil
}

//Close will close the boltDB connection
func (w *BoltDB) Close() {
	w.db.Close()
}

func (w *BoltDB) AllWorkLogs() ([]types.WorklogItem, error) {
	var worklogs []types.WorklogItem
	err := w.db.All(&worklogs)
	return worklogs, err
}

func (w *BoltDB) AllIssues() ([]types.ParentIssue, error) {
	var issues []types.ParentIssue
	err := w.db.All(&issues)
	return issues, err
}
