.PHONY: build run scheduler tidy test

build:
	go build -ldflags="-s -w" -o bin/mad-news-bot-server ./cmd/server
	go build -ldflags="-s -w" -o bin/mad-news-bot-scheduler ./cmd/scheduler

run:
	go run ./cmd/server

scheduler:
	go run ./cmd/scheduler

tidy:
	go mod tidy

test:
	go test ./...
