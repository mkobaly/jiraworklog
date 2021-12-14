package job

import (
	"strings"
	"time"

	"github.com/mkobaly/jiraworklog"
	"github.com/mkobaly/jiraworklog/repository"
	"github.com/mkobaly/jiraworklog/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

//JiraWorklogsDownloader is a background job that will download all worklogs from Jira
type JiraWorklogsDownloader struct {
	cfg    *jiraworklog.Config
	jira   jiraworklog.JiraReader
	repo   repository.Repo
	logger *log.Entry
}

func NewJiraDownloadWorklogs(cfg *jiraworklog.Config, jira jiraworklog.JiraReader, repo repository.Repo, logger *log.Entry) *JiraWorklogsDownloader {
	return &JiraWorklogsDownloader{
		cfg:    cfg,
		jira:   jira,
		repo:   repo,
		logger: logger,
	}
}

func (j *JiraWorklogsDownloader) GetName() string {
	return "JiraWorklogsDownloader"
}

func (j *JiraWorklogsDownloader) GetInterval() time.Duration {
	return time.Second * 300
}

func (j *JiraWorklogsDownloader) Run() error {
	//Keep looping until we are at the last page and nothing else to read

	//jira := jiraworklog.NewJira(j.cfg)

	lastTimestamp, err := j.repo.WorklogGetLastTimestamp()
	if err != nil {
		return errors.Wrap(err, "failed to get lastTimestamp from DB")
	}
	if lastTimestamp == 0 {
		// default going back 4 months
		lastTimestamp = time.Now().AddDate(0, -4, 0).UnixNano() / 1e6
	}
	maxWorklogID, err := j.repo.WorklogGetMaxWorklogID()
	if err != nil {
		return errors.Wrap(err, "failed to get maxWorklogID from DB")
	}
	j.logger.WithField("timestamp", lastTimestamp).Info("last timestamp")

	//Fetch worklogs updated since last timestamp check
	wl, err := j.jira.WorklogsUpdated(lastTimestamp)
	if err != nil {
		return errors.Wrap(err, "failed to fetch updated worklogs from jira")
	}
	j.logger.WithFields(log.Fields{"since": wl.Since, "until": wl.Until, "lastPage": wl.LastPage, "count": len(wl.Values)}).Info("worklogs updated stats")

	//For given worklog Ids we now need to get the worklog details
	var ids []int
	for _, w := range wl.Values {
		if w.WorklogID > maxWorklogID {
			ids = append(ids, w.WorklogID)
		}
	}

	if len(ids) == 0 {
		return nil //nothing to do
	}

	details, err := j.jira.WorklogDetails(ids)
	if err != nil {
		return errors.Wrap(err, "failed to fetch worklog details")
	}

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

		workItem, parentIssue := types.ConvertToModels(wd, issue, issueParent)
		err = j.repo.Write(workItem, parentIssue)
		if err != nil {
			return errors.Wrap(err, "error writting issue "+workItem.IssueKey)
		}
		maxWorklogID = workItem.ID
		j.logger.WithField("IssueKey", workItem.IssueKey).Info("inserted jira issue")
		time.Sleep(200 * time.Millisecond)
	}
	lastTimestamp = wl.Until
	j.repo.WorklogUpdateMaxWorklogID(maxWorklogID)
	j.repo.WorklogUpdateLastTimestamp(lastTimestamp)
	j.logger.WithField("lasttimestamp", lastTimestamp).WithField("maxworklogID", maxWorklogID).Info("finished processing batch")
	return nil
}

func (j *JiraWorklogsDownloader) okToProcess(w jiraworklog.Worklog, userNames []string) bool {
	if len(userNames) == 0 {
		return true
	}

	for _, u := range userNames {
		if strings.ToLower(w.Author.DisplayName) == strings.ToLower(u) {
			return true
		}
	}
	return false
}

// func (j *JiraWorklogsDownloader) convert(w jiraworklog.Worklog, i jiraworklog.Issue, parentIssue jiraworklog.Issue) types.WorklogItem {

// 	id, _ := strconv.Atoi(w.ID)
// 	issueID, _ := strconv.Atoi(i.ID)
// 	started, _ := time.Parse("2006-01-02T15:04:05.000-0700", w.Started)
// 	created, _ := time.Parse("2006-01-02T15:04:05.000-0700", i.Fields.Created)
// 	resolved := time.Time{}
// 	if i.Fields.ResolutionDate != nil {
// 		resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *i.Fields.ResolutionDate)
// 	}

// 	wi := types.WorklogItem{
// 		ID:                id,
// 		Author:            w.Author.Key,
// 		IssueID:           issueID,
// 		IssueKey:          i.Key,
// 		IssuePriority:     i.Fields.Priority.Name,
// 		IssueType:         i.Fields.Issuetype.Name,
// 		IssueSummary:      i.Fields.Summary,
// 		IssueStatus:       i.Fields.Status.Name,
// 		IssueCreateDate:   created,
// 		IssueResolvedDate: resolved,

// 		TimeSpentSeconds:        w.TimeSpentSeconds,
// 		OriginalEstimateSeconds: i.Fields.Timeoriginalestimate,
// 		Started:                 started,
// 		Project:                 strings.Split(i.Key, "-")[0],
// 	}

// 	if i.HasParent() {

// 		created, _ := time.Parse("2006-01-02T15:04:05.000-0700", parentIssue.Fields.Created)
// 		resolved := time.Time{}
// 		if parentIssue.Fields.ResolutionDate != nil {
// 			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *parentIssue.Fields.ResolutionDate)
// 		}

// 		parentID, _ := strconv.Atoi(i.Fields.Parent.ID)
// 		wi.ParentIssueID = &parentID
// 		wi.ParentIssueKey = &i.Fields.Parent.Key
// 		wi.ParentIssueType = &parentIssue.Fields.Issuetype.Name
// 		wi.ParentIssuePriority = &parentIssue.Fields.Priority.Name
// 		wi.ParentIssueSummary = &parentIssue.Fields.Summary
// 		wi.ParentIssueStatus = &parentIssue.Fields.Status.Name
// 		wi.ParentIssueCreateDate = created
// 		wi.ParentIssueResolvedDate = resolved
// 	}
// 	return wi
// }
