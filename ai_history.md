I would like to take this existing golang project that has an http endpoint but currently responds with json. I would like to start using golang templ and hotwire since I would like this also be mobile friendly to be used by hotwire native

can you convert the logging in the project from logrus to slog applying all the best practices

for building you should run build.sh not manually running templ and go build

can you make it so the echo v4 library is used instead of the standard go http lib

It seems like the web page does not load anymore. Can you review what is going on? 
to test you need to run the jiraworklog with the config.yaml in the bin folder using -c option

using the new MaintenanceRatio method on the Repo inteface can you create a new html page using templ and hotwire to render a graph that looks like this
<-- screenshot of graph inserted -->

there looks to be an error rendering the graph. Can you check what is going on?

can you create me a new configuration page to manage people. It should just list out the person name and role with the ability to update the role. You can see I added new methods to the repo.go interface to support those actions. You can use People() to get all the users, UpatePersonRole() to update the role for a person. Ideally it should be a type ahead drop down where it scans existing roles with the ability to add a new one if needed. You can use the AllRoles() method to get all of the existing roles

can you add a quick filter search box at the top of the people page to filter down the user list

can you make it so the footer is pinned to the bottom of the page? When you load the dashboard its not fixed at the bottom
<-- screenshot of graph inserted -->

I updated the MaintenanceRatio() method to accept an array of roles. Can you now update the UI to allow choosing from a checked drop down list the roles they want to be part of the request