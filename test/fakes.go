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
            },
            {
                "worklogId": 1113,
                "updatedTime": 1559600429929,
                "properties": []
            }
            
		],
		"since": 1556697267172,
		"until": 1559600429929,
		"self": "https://atlassian.net/rest/api/3/worklog/updated?since=1556697267172",
		"lastPage": true
	}`
	err := json.Unmarshal([]byte(jsonStr), &result)
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
			"timeSpentSeconds": 2000,
			"id": "1111",
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
			"id": "1112",
			"issueId": "6001"
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
			"id": "1113",
			"issueId": "6003"
		}
	]`
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

func (j *FakeJira) Issue(idOrKey string) (jiraworklog.Issue, error) {
	var result jiraworklog.Issue
	jsonStr := ""
	switch idOrKey {
	case "6000", "ABC-1234":
		jsonStr = issue6000
	case "6001", "ABC-1235":
		jsonStr = issue6001
	case "6002", "ABC-1236":
		jsonStr = issue6002
	case "6003", "ABC-1237":
		jsonStr = issue6003
	}
	err := json.Unmarshal([]byte(jsonStr), &result)
	return result, err
}

var issue6000 = `
{
	"expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
	"id": "6000",
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
		"created": "2019-05-12T09:08:40.627-0500",
		"timeoriginalestimate": null,
		"aggregateprogress": {
			"progress": 504000,
			"total": 504000,
			"percent": 100
		},
        "aggregatetimespent": 504000,
        "aggregatetimeoriginalestimate": 500000,
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

var issue6001 = `
{
    "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
    "id": "6001",
    "self": "https://atlassian.net/rest/api/3/issue/54464",
    "key": "ABC-1235",
    "fields": {
        "summary": "Summary test",
        "statuscategorychangedate": "2019-03-18T15:20:34.877-0500",
        "issuetype": {
            "self": "https://atlassian.net/rest/api/3/issuetype/5",
            "id": "5",
            "description": "The sub-task of the issue",
            "iconUrl": "https://atlassian.net/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype",
            "name": "Sub-task",
            "subtask": true,
            "avatarId": 10316
        },
        "parent": {
            "id": "6002",
            "key": "ABC-1236",
            "self": "https://atlassian.net/rest/api/3/issue/54462",
            "fields": {
                "summary": "Parent Summary",
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
                },
                "priority": {
                    "self": "https://atlassian.net/rest/api/3/priority/1",
                    "iconUrl": "https://atlassian.net/images/icons/priorities/critical.svg",
                    "name": "Highest",
                    "id": "1"
                },
                "issuetype": {
                    "self": "https://atlassian.net/rest/api/3/issuetype/10001",
                    "id": "10001",
                    "description": "A user story. Created by JIRA Software - do not edit or delete.",
                    "iconUrl": "https://amagsymmetry.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10315&avatarType=issuetype",
                    "name": "Story",
                    "subtask": false,
                    "avatarId": 10315
                }
            }
        },
        "timespent": 252000,
        "created": "2018-08-06T09:21:20.745-0500",
        "timeoriginalestimate": 75600,
        "aggregateprogress": {
            "progress": 252000,
            "total": 252000,
            "percent": 100
        },
        "aggregatetimespent": 252000,
        "aggregatetimeoriginalestimate": 400000,
        "priority": {
            "self": "https://atlassian.net/rest/api/3/priority/4",
            "iconUrl": "https://atlassian.net/images/icons/priorities/minor.svg",
            "name": "Low",
            "id": "4"
        },
        "timetracking": {
            "originalEstimate": "3d",
            "remainingEstimate": "0m",
            "timeSpent": "2w",
            "originalEstimateSeconds": 75600,
            "remainingEstimateSeconds": 0,
            "timeSpentSeconds": 252000
        },
        "resolutiondate": null,
        "progress": {
            "progress": 252000,
            "total": 252000,
            "percent": 100
        },
        "status": {
            "self": "https://atlassian.net/rest/api/3/status/11402",
            "description": "These are items that the development work is complete on but are waiting to be deployed for testing.\nUnit testing has been completed.",
            "iconUrl": "https://atlassian.net/images/icons/statuses/generic.png",
            "name": "Code Complete",
            "id": "11402",
            "statusCategory": {
                "self": "https://atlassian.net/rest/api/3/statuscategory/4",
                "id": 4,
                "key": "indeterminate",
                "colorName": "yellow",
                "name": "In Progress"
            }
        }
    }
}`

var issue6002 = `
{
    "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
    "id": "6002",
    "self": "https://atlassian.net/rest/api/3/issue/54462",
    "key": "ABC-1236",
    "fields": {
        "summary": "Parent Summary",
        "statuscategorychangedate": "2019-05-13T09:09:08.237-0500",
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
        "priority": {
            "self": "https://atlassian.net/rest/api/3/priority/1",
            "iconUrl": "https://atlassian.net/images/icons/priorities/critical.svg",
            "name": "Highest",
            "id": "1"
        },
        "aggregatetimespent": 504000,
        "aggregatetimeoriginalestimate": 500000,
        "timetracking": {},
        "resolutiondate": "2018-08-07T09:20:00.000-0500",
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
}
`

var issue6003 = `
{
    "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
    "id": "6003",
    "self": "https://atlassian.net/rest/api/3/issue/50916",
    "key": "ABC-1237",
    "fields": {
        "summary": "Story without time estimate",
        "statuscategorychangedate": "2018-03-05T13:10:29.812-0600",
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
        "created": "2018-03-05T13:10:29.812-0600",
        "timeoriginalestimate": null,
        "aggregateprogress": {
            "progress": 0,
            "total": 0
        },
        "priority": {
            "self": "https://atlassian.net/rest/api/3/priority/3",
            "iconUrl": "https://atlassian.net/images/icons/priorities/major.svg",
            "name": "Medium",
            "id": "3"
        },
        "aggregatetimespent": null,
        "timetracking": {},
        "aggregatetimeoriginalestimate": null,
        "resolutiondate": null,
        "progress": {
            "progress": 0,
            "total": 0
        },
        "status": {
            "self": "https://atlassian.net/rest/api/3/status/11000",
            "description": "Items in this status are awaiting further refinement to get them to a development ready state.",
            "iconUrl": "https://atlassian.net/images/icons/statuses/generic.png",
            "name": "Awaiting Refinement",
            "id": "11000",
            "statusCategory": {
                "self": "https://atlassian.net/rest/api/3/statuscategory/2",
                "id": 2,
                "key": "new",
                "colorName": "blue-gray",
                "name": "To Do"
            }
        }
    }
}
`
