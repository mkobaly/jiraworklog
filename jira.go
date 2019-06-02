package jiraworklog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

//var errUnknownProject = errors.New("Unknown Project")
var ErrIssueNotFound = errors.New("Jira Issue not found")

type Jira struct {
	Config *Config
	client *http.Client
}

func NewJira(c *Config) *Jira {
	return &Jira{
		Config: c,
		client: &http.Client{},
	}
}

func (j *Jira) WorklogsUpdated(timestamp int64) (UpdatedWorklogs, error) {
	worklog := UpdatedWorklogs{}
	since := ""
	if timestamp > 0 {
		since = "?since=" + strconv.FormatInt(timestamp, 10)
	}

	req, err := http.NewRequest("GET", j.Config.Jira.URL+"/worklog/updated"+since, nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return worklog, err
	}

	if resp.StatusCode != 200 {
		return worklog, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&worklog)
	if err != nil {
		return worklog, err
	}
	return worklog, nil
}

func (j *Jira) WorklogDetails(q *WorklogQuery) ([]Worklog, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return nil, err
	}

	var worklogs []Worklog
	req, err := http.NewRequest("POST", j.Config.Jira.URL+"/worklog/list", bytes.NewReader(b))
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	req.Header.Add("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return worklogs, err
	}

	if resp.StatusCode != 200 {
		return worklogs, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&worklogs)
	if err != nil {
		return worklogs, err
	}
	return worklogs, nil
}

func (j *Jira) Issue(idOrKey string) (Issue, error) {
	issue := Issue{}
	req, err := http.NewRequest("GET", j.Config.Jira.URL+"/issue/"+idOrKey+"?fields=priority,summary,parent,status,aggregateprogress,progress,issuetype,timespent,aggregatetimespent,timeoriginalestimate,timetracking,resolutiondate,created,statuscategorychangedate", nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return issue, err
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return issue, ErrIssueNotFound
		}

		return issue, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&issue)
	if err != nil {
		return issue, err
	}
	return issue, nil
}

type UpdatedWorklogs struct {
	Values []struct {
		WorklogID   int           `json:"worklogId"`
		UpdatedTime int64         `json:"updatedTime"`
		Properties  []interface{} `json:"properties"`
	} `json:"values"`
	Since    int64  `json:"since"`
	Until    int64  `json:"until"`
	Self     string `json:"self"`
	LastPage bool   `json:"lastPage"`
}

type Worklog struct {
	Self   string `json:"self"`
	Author struct {
		Self         string `json:"self"`
		Name         string `json:"name"`
		Key          string `json:"key"`
		AccountID    string `json:"accountId"`
		EmailAddress string `json:"emailAddress"`
		AvatarUrls   struct {
			Four8X48  string `json:"48x48"`
			Two4X24   string `json:"24x24"`
			One6X16   string `json:"16x16"`
			Three2X32 string `json:"32x32"`
		} `json:"avatarUrls"`
		DisplayName string `json:"displayName"`
		Active      bool   `json:"active"`
		TimeZone    string `json:"timeZone"`
		AccountType string `json:"accountType"`
	} `json:"author"`
	UpdateAuthor struct {
		Self         string `json:"self"`
		Name         string `json:"name"`
		Key          string `json:"key"`
		AccountID    string `json:"accountId"`
		EmailAddress string `json:"emailAddress"`
		AvatarUrls   struct {
			Four8X48  string `json:"48x48"`
			Two4X24   string `json:"24x24"`
			One6X16   string `json:"16x16"`
			Three2X32 string `json:"32x32"`
		} `json:"avatarUrls"`
		DisplayName string `json:"displayName"`
		Active      bool   `json:"active"`
		TimeZone    string `json:"timeZone"`
		AccountType string `json:"accountType"`
	} `json:"updateAuthor"`
	Created          string `json:"created"`
	Updated          string `json:"updated"`
	Started          string `json:"started"`
	TimeSpent        string `json:"timeSpent"`
	TimeSpentSeconds int    `json:"timeSpentSeconds"`
	ID               string `json:"id"`
	IssueID          string `json:"issueId"`
}

type Issue struct {
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
	Fields struct {
		Summary                  string  `json:"summary"`
		Created                  string  `json:created`
		ResolutionDate           *string `json:resolutiondate`
		StatusCategoryChangeDate *string `json:"statuscategorychangedate"`
		Issuetype                struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			IconURL     string `json:"iconUrl"`
			Name        string `json:"name"`
			Subtask     bool   `json:"subtask"`
			AvatarID    int    `json:"avatarId"`
		} `json:"issuetype"`
		Parent struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Self           string `json:"self"`
					Description    string `json:"description"`
					IconURL        string `json:"iconUrl"`
					Name           string `json:"name"`
					ID             string `json:"id"`
					StatusCategory struct {
						Self      string `json:"self"`
						ID        int    `json:"id"`
						Key       string `json:"key"`
						ColorName string `json:"colorName"`
						Name      string `json:"name"`
					} `json:"statusCategory"`
				} `json:"status"`
				Priority struct {
					Self    string `json:"self"`
					IconURL string `json:"iconUrl"`
					Name    string `json:"name"`
					ID      string `json:"id"`
				} `json:"priority"`
				Issuetype struct {
					Self        string `json:"self"`
					ID          string `json:"id"`
					Description string `json:"description"`
					IconURL     string `json:"iconUrl"`
					Name        string `json:"name"`
					Subtask     bool   `json:"subtask"`
					AvatarID    int    `json:"avatarId"`
				} `json:"issuetype"`
			} `json:"fields"`
		} `json:"parent"`
		Timespent            int `json:"timespent"`
		Timeoriginalestimate int `json:"timeoriginalestimate"`
		Description          struct {
			Version int    `json:"version"`
			Type    string `json:"type"`
			Content []struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"content"`
		} `json:"description"`
		Progress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
			Percent  int `json:"percent"`
		} `json:"progress"`
		Aggregateprogress struct {
			Progress int `json:"progress"`
			Total    int `json:"total"`
			Percent  int `json:"percent"`
		} `json:"aggregateprogress"`
		Aggregatetimespent int `json:"aggregatetimespent"`
		Priority           struct {
			Self    string `json:"self"`
			IconURL string `json:"iconUrl"`
			Name    string `json:"name"`
			ID      string `json:"id"`
		} `json:"priority"`
		Status struct {
			Self           string `json:"self"`
			Description    string `json:"description"`
			IconURL        string `json:"iconUrl"`
			Name           string `json:"name"`
			ID             string `json:"id"`
			StatusCategory struct {
				Self      string `json:"self"`
				ID        int    `json:"id"`
				Key       string `json:"key"`
				ColorName string `json:"colorName"`
				Name      string `json:"name"`
			} `json:"statusCategory"`
		} `json:"status"`
		Timetracking struct {
			OriginalEstimate         string `json:"originalEstimate"`
			RemainingEstimate        string `json:"remainingEstimate"`
			TimeSpent                string `json:"timeSpent"`
			OriginalEstimateSeconds  int    `json:"originalEstimateSeconds"`
			RemainingEstimateSeconds int    `json:"remainingEstimateSeconds"`
			TimeSpentSeconds         int    `json:"timeSpentSeconds"`
		} `json:"timetracking"`
	} `json:"fields"`
}

func (i Issue) HasParent() bool {
	return i.Fields.Parent.ID != ""
}

func (i Issue) ParentID() string {
	if i.HasParent() {
		return i.Fields.Parent.ID
	}
	return ""
}
