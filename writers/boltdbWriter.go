package writers

import (
	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
	"github.com/mkobaly/jiraworklog/types"
	"time"
)

type BoltDBWriter struct {
	db *storm.DB
}

//NewBoltDBWriter will create a new BoltDB writer
func NewBoltDBWriter(dbfile string) (*BoltDBWriter, error) {
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
	return &BoltDBWriter{
		db: db,
	}, nil
}

//NonResolvedIssues gets all issue keys that are not resolved yet
func (w *BoltDBWriter) NonResolvedIssues() ([]string, error) {
	var issues []types.ParentIssue
	var results []string
	query := w.db.Select(q.Eq("IsResolved", false))
	err := query.Find(&issues)
	if err != nil {
		return nil, err
	}
	for _, i := range issues {
		results = append(results, i.Key)
	}
	return results, nil
}

//Write will add the worklogItem to BoltDB
func (w *BoltDBWriter) Write(wi types.WorklogItem) error {
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

func (w *BoltDBWriter) UpdateResolutionDate(issueKey string, resolvedDate time.Time) error {
	var issue types.ParentIssue
	err := w.db.One("Key", issueKey, &issue)
	if err != nil {
		return err
	}
	err = w.db.Update(&types.ParentIssue{ID: issue.ID, ResolvedDate: resolvedDate, IsResolved: true})
	if err != nil {
		return err
	}

	// query := w.db.Select(q.Eq("IssueKey", issueKey))
	// query.Each(new(types.ParentIssue), func(record interface{}) error) {

	// }
	return nil
}

func (w *BoltDBWriter) Close() {
	w.db.Close()
}
