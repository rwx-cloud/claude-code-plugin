.PHONY: test-api test-worker test build

test: test-api test-worker

test-api:
	cd services/api && go test ./...

test-worker:
	cd services/worker && go test ./...

build:
	cd services/api && go build -o ../../bin/api .
	cd services/worker && go build -o ../../bin/worker .
