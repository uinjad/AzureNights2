.PHONY: build test lint run tidy

build:
	go build ./...

test:
	go test ./... -race -count=1

lint:
	golangci-lint run

run:
	go run ./cmd/rpg

tidy:
	go mod tidy

balance:
	go run ./cmd/balance