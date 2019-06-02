package jiraworklog

type WorklogQuery struct {
	Ids []int `json:"ids"`
}

func NewWorklogQuery() *WorklogQuery {
	return &WorklogQuery{
		Ids: []int{},
	}
}

func (w *WorklogQuery) Add(worklogID int) {
	w.Ids = append(w.Ids, worklogID)
}
