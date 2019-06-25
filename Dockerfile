FROM golang:1.12.6-alpine3.10
WORKDIR /go/src/github.com/mkobaly/jiraworklog/
COPY . .
RUN apk add --no-cache git \
    && go get -u github.com/kardianos/govendor \
    && govendor sync \
    && CGO_ENABLED=0 GOOS=linux go build \
        -a -installsuffix cgo -ldflags "-X main.Version=$(cat VERSION)" \
        -o ./bin/jiraworklog ./cmd/jiraworklog


FROM alpine:3.10
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/mkobaly/jiraworklog/bin/. /app/.
COPY --from=0 /go/src/github.com/mkobaly/jiraworklog/web/. /app/.
WORKDIR /app/
CMD ["./jiraworklog"] 