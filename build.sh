#build tagit (runs on build server..windows)
go get -u github.com/kardianos/govendor
govendor sync
env GOOS=windows GOARCH=386 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog.exe ./cmd/jiraworklog
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog ./cmd/jiraworklog