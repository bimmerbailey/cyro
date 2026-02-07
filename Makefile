BINARY_NAME := cyro
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X github.com/bimmerbailey/cyro/cmd.version=$(VERSION) \
	-X github.com/bimmerbailey/cyro/cmd.commit=$(COMMIT) \
	-X github.com/bimmerbailey/cyro/cmd.date=$(DATE)"

.PHONY: help build run test lint fmt vet clean install tidy

## help: show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: compile the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) .

## run: build and run the binary
run: build
	./bin/$(BINARY_NAME)

## install: install the binary to GOPATH/bin
install:
	go install $(LDFLAGS) .

## test: run all tests
test:
	go test -v -race -count=1 ./...

## test-cover: run tests with coverage
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: run golangci-lint (install: https://golangci-lint.run)
lint:
	golangci-lint run ./...

## fmt: format all Go files
fmt:
	gofmt -s -w .

## vet: run go vet
vet:
	go vet ./...

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

## check: run fmt, vet, and test
check: fmt vet test
