package main

import (
	"errors"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	cmdline "github.com/galdor/go-cmdline"
	"github.com/jmoiron/sqlx"

	"github.com/mkobaly/jiraworklog/types"
	"github.com/mkobaly/jiraworklog/writers"
)

var db *sqlx.DB
var ErrUnknownWriter = errors.New("unkown Writer")

func main() {
	//Define command line params and parse input
	cmdline := cmdline.New()
	cmdline.AddOption("c", "config", "config.yaml", "Path to configuration file")
	cmdline.AddOption("w", "writer", "MSSQL", "The writer to use")
	cmdline.Parse(os.Args)

	//Load up configuration. This holds Jira and SQL connection information
	cfgPath := "config.yaml"
	if cmdline.IsOptionSet("c") {
		cfgPath = cmdline.OptionValue("c")
	}

	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		switch err {
		case errNoConfigFile:
			color.Yellow("============================================================================================================")
			color.Yellow("Config file not present. Config.yaml was just created for you but you must edit the credential information")
			color.Yellow("============================================================================================================")
			os.Exit(0)
		default:
			checkErr(err)
		}
	}

	//Writer Settings
	writerType := "MSSQL"
	if cmdline.IsOptionSet("w") {
		writerType = cmdline.OptionValue("w")
	}

	//load writer
	writer, err := loadWriter(writerType, cfg)
	if err != nil {
		log.Fatal(err)
	}

	lastTimestamp := cfg.LastTimestamp
	if lastTimestamp == 0 {
		// default going back 60 days
		lastTimestamp = time.Now().Add(time.Hour*-1440).UnixNano() / 1e6
	}

	maxWorklogID := cfg.MaxWorklogID

	//Keep looping until we are at the last page and nothing else to read
	for {
		jira := NewJira(cfg)
		//Fetch worklogs updated since last timestamp check
		wl, err := jira.WorklogsUpdated(lastTimestamp)
		if err != nil {
			color.Red(err.Error())
			os.Exit(1)
		}

		//For given worklog Ids we now need to get the worklog details
		query := NewWorklogQuery()
		for _, w := range wl.Values {
			query.Add(w.WorklogID)
		}
		details, err := jira.WorklogDetails(query)

		for _, wd := range details {
			if !okToProcess(wd, cfg.UserList) {
				continue
			}

			issue, err := jira.Issue(wd)
			if err != nil {
				switch err {
				case ErrIssueNotFound:
					color.Yellow("Issue not found for worklog: " + wd.ID)
					continue
				default:
					color.Red(err.Error())
					os.Exit(1)
				}
			}

			issueParent := Issue{}
			if issue.HasParent() {
				issueParent, err = jira.IssueById(issue.ParentID())
			}

			workItem := convert(wd, issue, issueParent)
			err = writer.Write(workItem)
			if err != nil {
				color.Yellow("worklogId: %d, JiraIssue: %s, ParentIssue: %s Details: %v", workItem.ID, workItem.IssueKey, workItem.ParentIssueKey, workItem)
				color.Red(err.Error())
				os.Exit(1)
			}
			maxWorklogID = workItem.ID
			color.Green("Inserted record: " + workItem.IssueKey)
			time.Sleep(200 * time.Millisecond)
		}
		//Only setting last timestamp once done with batch
		lastTimestamp = wl.Until

		//Saving after every batch of records
		cfg.MaxWorklogID = maxWorklogID
		cfg.LastTimestamp = lastTimestamp
		cfg.Save(cfgPath)

		if wl.LastPage {
			break
		}
	}

}

func loadWriter(writerType string, cfg *Config) (writers.Writer, error) {
	switch writerType {
	case "MSSQL":
		db, err := sqlx.Connect("sqlserver", cfg.SQLConnection)
		if err != nil {
			return nil, err
		}
		writer := &writers.SQLWriter{DB: db}
		return writer, nil
	case "GOOGLESHEET":
		return nil, ErrUnknownWriter
	default:
		return nil, ErrUnknownWriter
	}
	return nil, ErrUnknownWriter
}

func okToProcess(w Worklog, userNames []string) bool {
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

func convert(w Worklog, i Issue, parentIssue Issue) types.WorklogItem {

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
		if i.Fields.ResolutionDate != nil {
			resolved, _ = time.Parse("2006-01-02T15:04:05.000-0700", *parentIssue.Fields.ResolutionDate)
		}

		parentID, _ := strconv.Atoi(i.Fields.Parent.ID)
		wi.ParentIssueID = &parentID
		wi.ParentIssueKey = &i.Fields.Parent.Key
		wi.ParentIssueType = &i.Fields.Parent.Fields.Issuetype.Name
		wi.ParentIssuePriority = &i.Fields.Parent.Fields.Priority.Name
		wi.ParentIssueSummary = &i.Fields.Parent.Fields.Summary
		wi.ParentIssueStatus = &i.Fields.Parent.Fields.Status.Name
		wi.IssueCreateDate = created
		wi.ParentIssueResolvedDate = resolved
	}
	return wi
}

//simple helper around errors
func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}
