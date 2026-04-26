FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY . .

RUN apk add --no-cache git && \
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    if [ -f go.mod ]; then go mod download; fi && \
    if [ -f go.mod ]; then CGO_ENABLED=0 GOOS=linux go build -o /tmp/server ./cmd/server; else printf 'go.mod missing\n' > /tmp/server; fi

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates postgresql-client

COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /tmp/server /app/server
COPY . .

RUN chmod +x /app/server /app/scripts/*.sh

EXPOSE 8000

CMD ["/bin/sh", "./scripts/start.sh"]
