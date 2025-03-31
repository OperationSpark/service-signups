## test: Run tests
.PHONY: test
test:
	GITHUB_WORKFLOW=CI go test ./...

## cover: Run tests and show coverage result
.PHONY: cover
cover:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## tidy: Cleanup and download missing dependencies
.PHONY: tidy
tidy:
	go mod tidy
	go mod verify

## lint: Run linting
.PHONY: lint
lint:
	golangci-lint run
