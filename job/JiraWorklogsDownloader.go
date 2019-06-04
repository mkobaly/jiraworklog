package job

import (
	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/mkobaly/jiraworklog/writers"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

//JiraWorklogsDownloader is a background job that will download all worklogs from Jira
type JiraWorklogsDownloader struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	writer writers.Writer
	logger *log.Entry
}

func NewJiraDownloadWorklogs(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, writer writers.Writer, logger *log.Entry) *JiraWorklogsDownloader {
	return &JiraWorklogsDownloader{
		cfg:    cfg,
		jira:   jira,
		writer: writer,
		logger: logger,
	}
}

func (j *JiraWorklogsDownloader) GetName() string {
	return "JiraDownloadWorklogs"
}

func (j *JiraWorklogsDownloader) GetInterval() time.Duration {
	return time.Second * 90
}

func (j *JiraWorklogsDownloader) Run() error {
	//Keep looping until we are at the last page and nothing else to read

	//jira := jiraworklog.NewJira(j.cfg)

	lastTimestamp := j.cfg.LastTimestamp
	if lastTimestamp == 0 {
		// default going back 60 days
		lastTimestamp = time.Now().AddDate(0, -2, 0).UnixNano() / 1e6
	}
	maxWorklogID := j.cfg.MaxWorklogID

	//Fetch worklogs updated since last timestamp check
	wl, err := j.jira.WorklogsUpdated(lastTimestamp)
	if err != nil {
		return errors.Wrap(err, "failed to fetch updated worklogs from jira")
	}

	//For given worklog Ids we now need to get the worklog details
	var ids []int
	//query := jiraworklog.NewWorklogQuery()
	for _, w := range wl.Values {
		ids = append(ids, w.WorklogID)
		//query.Add(w.WorklogID)
	}
	details, err := j.jira.WorklogDetails(ids)

	for _, wd := range details {
		if !j.okToProcess(wd, j.cfg.UserList) {
			continue
		}

		issue, err := j.jira.Issue(wd.IssueID)
		if err != nil {
			switch err {
			case jiraworklog.ErrIssueNotFound:
				continue
			default:
				return errors.Wrap(err, "unknown error getting issue details from jira. id="+wd.ID)
			}
		}

		issueParent := jiraworklog.Issue{}
		if issue.HasParent() {
			issueParent, err = j.jira.Issue(issue.ParentID())
		}

		workItem := j.convert(wd, issue, issueParent)
		err = j.writer.Write(workItem)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+workItem.IssueKey)
		}
		maxWorklogID = workItem.ID
		j.logger.WithField("IssueKey", workItem.IssueKey).Info("inserted jira issue")
		time.Sleep(200 * time.Millisecond)
	}
	lastTimestamp = wl.Until
	j.cfg.MaxWorklogID = maxWorklogID
	j.cfg.LastTimestamp = lastTimestamp
	j.cfg.Save()
	j.logger.WithField("lasttimestamp", lastTimestamp).WithField("maxworklogID", maxWorklogID).Info("finished processing batch")
	return nil
}

func (j *JiraWorklogsDownloader) okToProcess(w jiraworklog.Worklog, userNames []string) bool {
	if len(userNames) == 0 {
		return true
	}

	for _, u := range userNames {
		if strings.ToLower(w.Author.Key) == strings.ToLower(u) {
			return true
		}
	}
	return false
}

func (j *JiraWorklogsDownloader) convert(w jiraworklog.Worklog, i jiraworklog.Issue, parentIssue jiraworklog.Issue) types.WorklogItem {

	id, _ := strconv.Atoi(w.ID)
	issueID, _ := strconv.Atoi(i.ID)
	started, _ := time.Parse("2006-01-02T15:04:05.000-0700", w.Started)
	created, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Created)
	resolved := time.Time{}
	if i.Fields.ResolutionDate != nil {
		resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
	}

	wi := types.WorklogItem{
		ID:                id,
		Author:            w.Author.Key,
		IssueID:           issueID,
		IssueKey:          i.Key,
		IssuePriority:     i.Fields.Priority.Name,
		IssueType:         i.Fields.Issuetype.Name,
		IssueSummary:      i.Fields.Summary,
		IssueStatus:       i.Fields.Status.Name,
		IssueCreateDate:   created,
		IssueResolvedDate: resolved,

		TimeSpentSeconds:        w.TimeSpentSeconds,
		OriginalEstimateSeconds: i.Fields.Timeoriginalestimate,
		Started:                 started,
		Project:                 strings.Split(i.Key, "-")[0],
	}

	if i.HasParent() {

		created, _ := time.Parse("2006-01-02T15:04:05.000-0700", parentIssue.Fields.Created)
		resolved := time.Time{}
		if parentIssue.Fields.ResolutionDate != nil {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *parentIssue.Fields.ResolutionDate)
		}

		parentID, _ := strconv.Atoi(i.Fields.Parent.ID)
		wi.ParentIssueID = &parentID
		wi.ParentIssueKey = &i.Fields.Parent.Key
		wi.ParentIssueType = &parentIssue.Fields.Issuetype.Name
		wi.ParentIssuePriority = &parentIssue.Fields.Priority.Name
		wi.ParentIssueSummary = &parentIssue.Fields.Summary
		wi.ParentIssueStatus = &parentIssue.Fields.Status.Name
		wi.ParentIssueCreateDate = created
		wi.ParentIssueResolvedDate = resolved
	}
	return wi
}
