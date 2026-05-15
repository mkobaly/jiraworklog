package jiraworklog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// var errUnknownProject = errors.New("Unknown Project")
var ErrIssueNotFound = errors.New("Jira Issue not found")

type JiraReader interface {
	WorklogsUpdated(timestamp int64) (UpdatedWorklogs, error)
	WorklogsDeleted(timestamp int64) (DeletedWorklogs, error)
	WorklogDetails(ids []int) ([]Worklog, error)
	Issue(idOrKey string) (Issue, error)
	BulkFetchIssues(idOrKeys []string) ([]Issue, error)
	IssuesUpdated(timeStart string, timeEnd string, nextPageToken string) (IssuesUpdated, error)

	Changelog(id int, startAt int) (Changelog, error)
	GetJiraUser() (JiraUser, error)
	GetTimezone() (*time.Location, error)
}

type Jira struct {
	Config   *Config
	client   *http.Client
	timezone *time.Location
	mu       sync.Mutex
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
	defer resp.Body.Close()

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

func (j *Jira) WorklogsDeleted(timestamp int64) (DeletedWorklogs, error) {
	worklog := DeletedWorklogs{}
	since := ""
	if timestamp > 0 {
		since = "?since=" + strconv.FormatInt(timestamp, 10)
	}

	req, err := http.NewRequest("GET", j.Config.Jira.URL+"/worklog/deleted"+since, nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return worklog, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return worklog, fmt.Errorf("WorklogsDeleted bad response %d, since: %s", resp.StatusCode, since)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&worklog)
	if err != nil {
		return worklog, err
	}
	return worklog, nil
}

func (j *Jira) WorklogDetails(ids []int) ([]Worklog, error) {
	worklogList := WorklogList{
		IDs: ids,
	}
	b, err := json.Marshal(worklogList)
	if err != nil {
		return nil, err
	}

	var worklogs []Worklog
	req, err := http.NewRequest("POST", j.Config.Jira.URL+"/worklog/list", bytes.NewBuffer(b))
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	req.Header.Add("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return worklogs, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return worklogs, fmt.Errorf("WorklogDetails bad response: %d. Trying to fetch ids: %v", resp.StatusCode, ids)
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
	req, err := http.NewRequest("GET", j.Config.Jira.URL+"/issue/"+idOrKey+"?fields=priority,summary,parent,status,aggregateprogress,progress,issuetype,timespent,aggregatetimespent,timeoriginalestimate,aggregatetimeoriginalestimate,timetracking,resolutiondate,created,statuscategorychangedate,customfield_13521", nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return issue, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return issue, ErrIssueNotFound
		}
		if resp.StatusCode == 410 {
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

// IssuesUpdated will fetch all issues that were updated within the calculated date range.
// For past days it queries the full day; for today it queries the current hour window.
func (j *Jira) IssuesUpdated(timeStart string, timeEnd string, nextPageToken string) (IssuesUpdated, error) {
	issuesUpdated := IssuesUpdated{}

	query := fmt.Sprintf("jql=updated>=\"%s\" AND updated < \"%s\" order by updated ASC", timeStart, timeEnd)
	//slog.Info("Issue Updated", slog.String("start", ts), slog.String("end", te))
	if nextPageToken != "" {
		query += fmt.Sprintf("&nextPageToken=%s", nextPageToken)
	}
	req, err := http.NewRequest("GET", j.Config.Jira.URL+fmt.Sprintf("/search/jql?%s", url.PathEscape(query)), nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return issuesUpdated, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return issuesUpdated, ErrIssueNotFound
		}

		return issuesUpdated, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&issuesUpdated)
	if err != nil {
		return issuesUpdated, err
	}
	return issuesUpdated, nil
}

// BulkFetchIssues will fetch multiple jira issues at one time
// WARNING: This fails silently when you don't have permissions. It returns a 200 result with zero records.
func (j *Jira) BulkFetchIssues(idOrKeys []string) ([]Issue, error) {
	issues := []Issue{}

	// Step 2: Use the JQL to search for issues
	searchPayload := map[string]interface{}{
		"fields": []string{"priority", "summary", "parent", "status", "aggregateprogress", "progress", "assignee", "labels",
			"issuetype", "timespent", "aggregatetimespent", "timeoriginalestimate", "aggregatetimeoriginalestimate", "timetracking",
			"resolutiondate", "created", "updated", "statuscategorychangedate", "fixVersions", "versions", "customfield_13521"},
		"issueIdsOrKeys": idOrKeys,
		//"maxResults":     200, // adjust as needed
	}
	payloadBytes, err := json.Marshal(searchPayload)
	if err != nil {
		return issues, err
	}

	req, err := http.NewRequest("POST", j.Config.Jira.URL+"/issue/bulkfetch", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return issues, err
	}
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return issues, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return issues, ErrIssueNotFound
		}
		return issues, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	response := BulkJiraResponse{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return issues, err
	}
	return response.Issues, nil
}

func (j *Jira) Changelog(id int, startAt int) (Changelog, error) {
	changelog := Changelog{}
	req, err := http.NewRequest("GET", j.Config.Jira.URL+fmt.Sprintf("/issue/%d/changelog?maxResults=100&startAt=%d", id, startAt), nil)
	if err != nil {
		return changelog, err
	}
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	req.Header.Set("Content-Type", "application/json")
	resp, err := j.client.Do(req)
	if err != nil {
		return changelog, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if resp.StatusCode == 404 {
			return changelog, ErrIssueNotFound
		}
		return changelog, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&changelog)
	return changelog, err
}

func (j *Jira) GetJiraUser() (JiraUser, error) {
	user := JiraUser{}

	req, err := http.NewRequest("GET", j.Config.Jira.URL+"/myself", nil)
	req.SetBasicAuth(j.Config.Jira.Username, j.Config.Jira.Password)
	resp, err := j.client.Do(req)
	if err != nil {
		return user, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return user, fmt.Errorf("Not 200 response %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&user)
	if err != nil {
		return user, err
	}
	return user, nil
}

func (j *Jira) GetTimezone() (*time.Location, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.timezone == nil {
		user, err := j.GetJiraUser()
		if err != nil {
			//slog.Error("error getting jira user. Needed to fetch timezone", slog.Any("error", err))
			return time.Local, err
		}
		tz, err := time.LoadLocation(user.TimeZone)
		if err != nil {
			//slog.Error("error loading location from jira user timezone. Needed to fetch timezone", slog.String("timezone", user.TimeZone))
			return time.Local, err
		}
		j.timezone = tz
	}
	return j.timezone, nil
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

type DeletedWorklogs struct {
	LastPage bool   `json:"lastPage"`
	NextPage string `json:"nextPage"`
	Self     string `json:"self"`
	Since    int64  `json:"since"`
	Until    int64  `json:"until"`
	Values   []struct {
		UpdatedTime int64 `json:"updatedTime"`
		WorklogID   int   `json:"worklogId"`
	} `json:"values"`
}

type Worklog struct {
	Self   string `json:"self"`
	Author struct {
		Self string `json:"self"`
		//Name         string `json:"name"`
		//Key          string `json:"key"`
		AccountID string `json:"accountId"`
		//EmailAddress string `json:"emailAddress"`
		AvatarUrls struct {
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

type WorklogList struct {
	IDs []int `json:"ids"`
}

type Issue struct {
	Expand string `json:"expand"`
	ID     string `json:"id"`
	Self   string `json:"self"`
	Key    string `json:"key"`
	Fields struct {
		Summary                  string  `json:"summary"`
		Created                  string  `json:"created"`
		Updated                  string  `json:"updated"`
		ResolutionDate           *string `json:"resolutiondate"`
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
		Parent *struct {
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
		Labels   []string `json:"labels"`
		Assignee struct {
			Self       string `json:"self"`
			AccountID  string `json:"accountId"`
			AvatarUrls struct {
				Four8X48  string `json:"48x48"`
				Two4X24   string `json:"24x24"`
				One6X16   string `json:"16x16"`
				Three2X32 string `json:"32x32"`
			} `json:"avatarUrls"`
			DisplayName string `json:"displayName"`
			Active      bool   `json:"active"`
			TimeZone    string `json:"timeZone"`
			AccountType string `json:"accountType"`
		} `json:"assignee"`
		//Timespent            int `json:"timespent"`
		//Timeoriginalestimate int `json:"timeoriginalestimate"`
		Description struct {
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
		Priority struct {
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
		ProjectCharge struct {
			Self  string `json:"self"`
			Value string `json:"value"`
			ID    string `json:"id"`
		} `json:"customfield_13521"`
		FixVersions []struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
			Archived    bool   `json:"archived"`
			Released    bool   `json:"released"`
			ReleaseDate string `json:"releaseDate"`
		} `json:"fixVersions"`
		AffectsVersions []struct {
			Self        string `json:"self"`
			ID          string `json:"id"`
			Description string `json:"description"`
			Name        string `json:"name"`
			Archived    bool   `json:"archived"`
			Released    bool   `json:"released"`
			ReleaseDate string `json:"releaseDate"`
		} `json:"versions"`
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

type IssuesUpdated struct {
	IsLast        bool   `json:"isLast"`
	NextPageToken string `json:"nextPageToken"`
	Issues        []struct {
		ID string `json:"id"`
	} `json:"issues"`
}

type BulkJiraResponse struct {
	Expand string  `json:"expand"`
	Issues []Issue `json:"issues"`
}

type Changelog struct {
	Self       string `json:"self"`
	MaxResults int    `json:"maxResults"`
	StartAt    int    `json:"startAt"`
	Total      int    `json:"total"`
	IsLast     bool   `json:"isLast"`
	Values     []struct {
		ID     string `json:"id"`
		Author struct {
			Self       string `json:"self"`
			AccountID  string `json:"accountId"`
			AvatarUrls struct {
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
		Created string `json:"created"`
		Items   []struct {
			Field      string `json:"field"`
			Fieldtype  string `json:"fieldtype"`
			From       string `json:"from"`
			FromString string `json:"fromString"`
			To         string `json:"to"`
			ToString   string `json:"toString"`
		} `json:"items"`
		HistoryMetadata struct {
		} `json:"historyMetadata,omitempty"`
	} `json:"values"`
}

type JiraUser struct {
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
	Active       bool   `json:"active"`
	TimeZone     string `json:"timeZone"`
	Locale       string `json:"locale"`
}
