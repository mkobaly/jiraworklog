package test

import (
	"encoding/json"
	"github.com/mkobaly/jiraworklog"
)

type FakeJira struct {
}

func (j *FakeJira) WorklogsUpdated(timestamp int64) (jiraworklog.UpdatedWorklogs, error) {
	var result jiraworklog.UpdatedWorklogs
	jsonStr := `
	{
		"values": [
			{
				"worklogId": 1111,
				"updatedTime": 1556697267173,
				"properties": []
			},        
			{
			"worklogId": 1112,
			"updatedTime": 1559600429921,
			"properties": []
			}
		],
		"since": 1556683200000,
		"until": 1559600637162,
		"self": "https://atlassian.net/rest/api/3/worklog/updated?since=1556683200000",
		"lastPage": true
	}`
	err := json.Unmarshal([]byte(jsonStr), result)
	return result, err

}

func (j *FakeJira) WorklogDetails(ids []int) ([]jiraworklog.Worklog, error) {
	var result []jiraworklog.Worklog
	jsonStr := `
	[
		{
			"self": "https://atlassian.net/rest/api/3/issue/60025/worklog/1111",
			"author": {
				"self": "https://atlassian.net/rest/api/3/user?accountId=117058%3Aad714362-0c30-4ad3-ba5b-70a84f2ab3f3",
				"name": "big.bird",
				"key": "big.bird",
				"accountId": "111111:ad714362-0c30-4ad3-ba5b-70a84f2ab3f3",
				"emailAddress": "big.bird@example.com",
				"avatarUrls": {
					"48x48": "https://avatar-cdn.atlassian.com/xxxxxx",
					"24x24": "https://avatar-cdn.atlassian.com/yyyyyy",
					"16x16": "https://avatar-cdn.atlassian.com/zzzzzz",
					"32x32": "https://avatar-cdn.atlassian.com/zxzxzx"
				},
				"displayName": "Big Bird",
				"active": true,
				"timeZone": "America/Chicago",
				"accountType": "atlassian"
			},
			"created": "2019-05-09T17:05:02.339-0500",
			"updated": "2019-05-09T17:05:02.339-0500",
			"started": "2019-05-09T17:04:00.000-0500",
			"timeSpent": "45m",
			"timeSpentSeconds": 2700,
			"id": "46666",
			"issueId": "6000"
		},
		{
			"self": "https://atlassian.net/rest/api/3/issue/60025/worklog/1112",
			"author": {
				"self": "https://atlassian.net/rest/api/3/user?accountId=117058%3Aad714362-0c30-4ad3-ba5b-70a84f2ab3f3",
				"name": "big.bird",
				"key": "big.bird",
				"accountId": "111111:ad714362-0c30-4ad3-ba5b-70a84f2ab3f3",
				"emailAddress": "big.bird@example.com",
				"avatarUrls": {
					"48x48": "https://avatar-cdn.atlassian.com/xxxxxx",
					"24x24": "https://avatar-cdn.atlassian.com/yyyyyy",
					"16x16": "https://avatar-cdn.atlassian.com/zzzzzz",
					"32x32": "https://avatar-cdn.atlassian.com/zxzxzx"
				},
				"displayName": "Big Bird",
				"active": true,
				"timeZone": "America/Chicago",
				"accountType": "atlassian"
			},
			"created": "2019-05-10T17:05:02.339-0500",
			"updated": "2019-05-10T17:05:02.339-0500",
			"started": "2019-05-10T17:04:00.000-0500",
			"timeSpent": "45m",
			"timeSpentSeconds": 2700,
			"id": "46667",
			"issueId": "6001"
		}
	]`
	err := json.Unmarshal([]byte(jsonStr), result)
	return result, err
}

func (j *FakeJira) Issue(idOrKey string) (jiraworklog.Issue, error) {
	var result jiraworklog.Issue
	jsonStr := `
	{
		"expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
		"id": "6001",
		"self": "https://atlassian.net/rest/api/3/issue/6001",
		"key": "ABC-1234",
		"fields": {
			"statuscategorychangedate": "2019-05-13T09:09:08.237-0500",
			"summary": "Summary",
			"issuetype": {
				"self": "https://atlassian.net/rest/api/3/issuetype/10001",
				"id": "10001",
				"description": "A user story. Created by JIRA Software - do not edit or delete.",
				"iconUrl": "https://atlassian.net/secure/viewavatar?size=xsmall&avatarId=10315&avatarType=issuetype",
				"name": "Story",
				"subtask": false,
				"avatarId": 10315
			},
			"timespent": null,
			"created": "2018-08-06T09:14:40.627-0500",
			"timeoriginalestimate": null,
			"aggregateprogress": {
				"progress": 504000,
				"total": 504000,
				"percent": 100
			},
			"aggregatetimespent": 504000,
			"priority": {
				"self": "https://atlassian.net/rest/api/3/priority/1",
				"iconUrl": "https://atlassian.net/images/icons/priorities/critical.svg",
				"name": "Highest",
				"id": "1"
			},
			"timetracking": {},
			"resolutiondate": null,
			"progress": {
				"progress": 0,
				"total": 0
			},
			"status": {
				"self": "https://atlassian.net/rest/api/3/status/6",
				"description": "The issue is considered finished, the resolution is correct. Issues which are closed can be reopened.",
				"iconUrl": "https://atlassian.net/images/icons/statuses/closed.png",
				"name": "Closed",
				"id": "6",
				"statusCategory": {
					"self": "https://atlassian.net/rest/api/3/statuscategory/3",
					"id": 3,
					"key": "done",
					"colorName": "green",
					"name": "Done"
				}
			}
		}
	}`
	err := json.Unmarshal([]byte(jsonStr), result)
	return result, err
}
