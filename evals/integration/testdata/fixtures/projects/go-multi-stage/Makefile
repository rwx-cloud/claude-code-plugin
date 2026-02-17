.PHONY: lint test build

lint:
	go vet ./...
	golangci-lint run

test:
	go test -race ./...

build:
	go build -o bin/app ./cmd/app
