#build tagit (runs on build server..windows)
mkdir -p ./bin/web
cp ./web/* ./bin/web
env GOOS=windows GOARCH=386 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog.exe ./cmd/jiraworklog
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(cat VERSION)" -o ./bin/jiraworklog ./cmd/jiraworklog