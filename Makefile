.PHONY: build run test lint

build:
	go build -o bin/app ./cmd

run:
	go run ./cmd

test:
	go test ./...

lint:
	golangci-lint run
