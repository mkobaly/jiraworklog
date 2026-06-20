# Jira last updated
The Jira API does not support a timestamp for pulling all jira issues updated since last timestamp. It needs to be a date and in the correct timezone. It looks like the timezone
is fixed per Jira account. For our account its -5 hours and you can see that in all dates  returned from the Jira rest api
        
    2026-03-29T12:22:43.492-0500

