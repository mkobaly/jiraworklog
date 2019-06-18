#build tagit (runs on build server..windows)
env GOOS=windows GOARCH=386 go build -o ./bin/jiraworklog.exe ./cmd/jiraworklog
env GOOS=linux GOARCH=amd64 go build -o ./bin/jiraworklog ./cmd/jiraworklog