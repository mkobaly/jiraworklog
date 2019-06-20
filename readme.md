# JiraWorklog   

This is a WIP. Its not complete yet

Simple utility to extract all Jira Worklogs and record them so they can be easily reported on. 
This is a self contained solution that will collect all of the worklogs and provide a simple
dashboard to present the data

Currently two databases are supported. BoltDB and MS SQL Server. This is configured via the command line with boltDB being the default option.

For SQL Server its expected that the database and tables already exist. See below for table schema

## Reasons why this is needed

Jira does not provide a good way to see developers productivity across all projects

Time tracking is typically done at the lowest issue level. For a bug that makes sense. For a 
story many times the time tracking is logged against the sub-task. From a management level we care more
about the "parent issues" when it comes to seeing what was completed

The assignee gets changed over time so you really don't know what developer worked on what "parent issue"
unless you dig into the history of an issue. 

The existing reports in Jira don't provide good insight into productivity of developers unless you use
a jira plug-in that costs money and could change the standard jira time tracking functionality. I wanted
this to work with out of the box jira time tracking (work logs)

The current reporting options in Jira Cloud are somewhat limiting and unless you purchase an add-on the
time spent by developers is difficult to report on. This utility is looking to change that

## How to run

 Execute the jiraworklog executable. The first time you run this it will warn you that a 
 configuration file was not present but one has been created for you. Edit the config.yaml file
 accordingly

 - jira section (required): this must point to your cloud Jira URL. Ensure username and password are set correctly. Its recommended to use an api_token vs an individual password

 - sqlconnection (optional): If you want to use SQL server as the backend repository this is required

 - userlist: (optional) - If you want to filter the worklogs to a subset of users

 - donestatus: (required) - Due to the way jira works with status transitions you can sometimes transition to a done status BUT the resolution date / reason could be missing. If we know the done statuses then we can ensure we mark the issue resolved when it reaches this status. Only used when the resolution date is not being populated

## Workflow

Jira should be easy to use but its not. Here we are assuming 1 simple rule. Developers log their time on tickets they work on. Given that we can pull all work log entries and from those entries get the issue and parent issue (ex: sub-task => story) developers work on we now know all of the "parent issues" being worked on and we can track when those parent issues are resolved


## Rest API

- /worklogs - get all worklogs (todo: add paging)
- /issues - get all issues (todo: add paging)
- /issues/groupby - issue data going back x days group by given value
- /dashboard/

## dashboard

- Historical page?
  - The days to complete by type line chart below could fit here
  - Pivot result of per developer past 6 weeks, hours worked per week

----------------------------------------------------------------------------
Issues
----------------------------------------------------------------------------

TODO: Think need lastUpdated property on the Issues. Can't just use create date or resolvedDate
 - Any time developer adds time update issue / parent issue last updated
 - Any time issue resolved update last updated
 - Now can query on last updated

 Ex:
 1) created 5/1/19, logged time 5/5, 5/7 and closed on 5/10 
 2) created 5/3/19, logged time 5/6, never closed
     - This should stay on list I would imagine

---------------  ------------  ----------------
| By Priority |  | By Issue |  | By Developer |
---------------  ------------  ----------------

issues by type: avg days to resolve (ones that are closed)

per developer: %accuracy on estimate

all developers: %accuracy (guage)

List - Issues not closed and older than x days

Days to complete by type
 - line chart going back 6 weeks. Each week point

------------------------------------------------------
Worklogs
------------------------------------------------------

Who hasn't logged any hours for today

By Weekday
--------------
  | | |
| | | | | | |
S M T W T F S
---------------

% of workweek 60% (circle with % inside. < 40% red, 40-60% yellow, 60%+ green)

By Developer
- Bar chart per weekday

By Developer - change week over week?


## Repository

A repository handle the persistance and querying of the Jira data from our database. The goal is to have different "repositories" available. Currently only BoltDB and SQL Server are implemented. 

```sql
-- Script that will create Jira database and two tables needed
-- This needs to be manually executed in order to use SQL server
create database Jira
GO
Use Jira
GO
create table worklog
(
	id int NOT NULL PRIMARY KEY,
	author varchar(50) NOT NULL,
	[date] datetime NOT NULL,

	weekNumber       int NOT NULL,
	weekDay          varchar(10) NOT NULL,
	timeSpentSeconds int NOT NULL,
	timeSpentHours   numeric(5,2) NOT NULL,
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
	developer varchar(50) NOT NULL,
	createDate datetime NOT NULL,
	updateDate datetime NOT NULL,
	resolvedDate datetime NULL,
	daysToResolve int NOT NULL,
	isResolved bit NOT NULL DEFAULT(0),
	aggregateTimeSpent int NOT NULL DEFAULT(0),
	aggregateTimeOriginalEstimate int NOT NULL DEFAULT(0),
	dateInserted datetime NOT NULL default(getutcdate())
)
```