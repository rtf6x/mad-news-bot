.PHONY: build run worker scheduler tidy test

build:
	go build -ldflags="-s -w" -o bin/mad-news-bot-server ./cmd/server
	go build -ldflags="-s -w" -o bin/mad-news-bot-worker ./cmd/worker
	go build -ldflags="-s -w" -o bin/mad-news-bot-scheduler ./cmd/scheduler

run:
	go run ./cmd/server

worker:
	go run ./cmd/worker

scheduler:
	go run ./cmd/scheduler

tidy:
	go mod tidy

test:
	go test ./...
