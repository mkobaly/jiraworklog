FROM golang:1.16.0-alpine3.13
WORKDIR /go/src/github.com/mkobaly/jiraworklog/
COPY . .
RUN apk add --no-cache git \
    && CGO_ENABLED=0 GOOS=linux go build \
        -a -installsuffix cgo -ldflags "-X main.Version=$(cat VERSION)" \
        -o ./bin/jiraworklog ./cmd/jiraworklog


FROM alpine:3.13
RUN apk --no-cache add ca-certificates
COPY --from=0 /go/src/github.com/mkobaly/jiraworklog/bin/. /app/.
COPY --from=0 /go/src/github.com/mkobaly/jiraworklog/web/. /app/web/.
WORKDIR /app/
CMD ["./jiraworklog"]