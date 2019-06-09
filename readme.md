# JiraWorklog   

This is a WIP. Its not complete yet

Simple utility to extract all Jira Worklogs and record them so they can be easily reported on

The current reporting options in Jira Cloud are somewhat limiting and unless you purchase an add-on the
time spent by developers is difficult to report on. This utility is looking to change that

## Workflow

This utility will query the Jira REST API and extract all of the worklogs updated and log them. For every worklog it will then pull in the issue details and if that issue has a parent (say: Sub Task => Story) then the parent issue will also be pulled.



It will default going back 60 days but you can configure that to go back to the start of your Jira usage by using a timestamp of 0

## Jira Writers

Jira writers handle the persistance of the Jira data pulled. The goal is to have different "writters" that are available for output. Currently only SQL Server is implemented. Other writers could be Google Sheets, Excel, mysql, mongodb, etc

```sql
create table worklog
(
	worklogId int NOT NULL PRIMARY KEY,
	author varchar(50) NOT NULL,
	[date] datetime NOT NULL,
	timeSpentSeconds int NOT NULL,
	originalEstimateSeconds int NOT NULL,
	project varchar(10) NOT NULL,
	issueId int NOT NULL,
	issueKey varchar(20) NOT NULL,
	issueType varchar(20) NOT NULL,
	issueSummary varchar(255) NOT NULL,
	issuePriority varchar(20) NOT NULL,
	issueStatus varchar(50) NOT NULL,
	parentIssueId int NULL,
	parentIssueKey varchar(20) NULL,
	parentIssueType varchar(20) NULL,
	parentIssueSummary varchar(255) NULL,
	parentIssuePriority varchar(20) NULL,
	parentIssueStatus varchar(50) NULL,
	dateInserted datetime NOT NULL default(getutcdate())
)

create table issue
(
	id int NOT NULL PRIMARY KEY,
	[key] varchar(20) NOT NULL UNIQUE,
	[type] varchar(20) NOT NULL,
	summary varchar(255) NOT NULL,
	priority varchar(20) NOT NULL,
	status varchar(50) NOT NULL,
	project varchar(10) NOT NULL,
	createDate datetime NOT NULL,
	resolvedDate datetime NULL,
	isResolved bit NOT NULL DEFAULT(0),
	dateInserted datetime NOT NULL default(getutcdate())
)
```