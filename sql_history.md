```sql
--Ran these scripts to clean up the worklogs having wrong project charges (I think)
SELECT *
  FROM [jira_new].[dbo].[worklog]
  where issueProjectCharge != parentIssueProjectCharge
  order by parentIssueProjectCharge

  update [jira_new].[dbo].[worklog]
  set issueProjectCharge = parentIssueProjectCharge
  where issueProjectCharge != parentIssueProjectCharge
  and parentIssueProjectCharge = 'TD107AC/00 - Consolidated Identity Management'


  update [jira_new].[dbo].[worklog]
  set issueProjectCharge = parentIssueProjectCharge
  where issueProjectCharge != parentIssueProjectCharge
  and parentIssueProjectCharge = 'TD103DQ/00 - Symmetry Mobile Application - Identity Management'

  

  update [jira_new].[dbo].[worklog]
  set issueProjectCharge = parentIssueProjectCharge
  where issueProjectCharge != parentIssueProjectCharge
  and parentIssueProjectCharge = 'TD103DY/00 - M2150 OSDP'

  
  update [jira_new].[dbo].[worklog]
  set issueProjectCharge = parentIssueProjectCharge
  where issueProjectCharge != parentIssueProjectCharge
  and parentIssueProjectCharge = 'TD103DZ/00 - SR OSDP Project' and issueProjectCharge like '%after market%'

  update [jira_new].[dbo].[worklog]
  set issueProjectCharge = parentIssueProjectCharge
  where issueProjectCharge != parentIssueProjectCharge
  and parentIssueProjectCharge = 'TD103EB/00 - Symmetry Wallet' and issueProjectCharge like '%after market%'

update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD103ED/00 - Download Queue Performance'


update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD107AA/00 - Quick Connect'

update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD107V/00 - SAMA Enhancement'

update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD107AD/00 - SwiftConnect Integration'

update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD107Z/00 - NFC Wallet'

update [jira_new].[dbo].[worklog]
set issueProjectCharge = parentIssueProjectCharge
where issueProjectCharge != parentIssueProjectCharge
and parentIssueProjectCharge = 'TD107Y/00 - Equinix Enhancements'
```