# Variables
PACKAGES := $(shell go list ./...)
BINARY_NAME := firelynx
VERSION := 0.1.0

.PHONY: all
all: help

## help: Display this help message
.PHONY: help
help: Makefile
	@echo
	@echo " Choose a make command to run"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' | sed -e 's/^/ /'
	@echo

## build: Build the binary
.PHONY: build
build: protogen
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/firelynx

## install: Install the binary
.PHONY: install
install: protogen
	go install -ldflags "-X main.Version=$(VERSION)" ./cmd/firelynx

## protogen: Generate code from protobuf definitions
.PHONY: protogen
protogen:
	buf generate

## test: Run tests with race detection and coverage
.PHONY: test
test: protogen
	go test -race -cover $(PACKAGES)

## lint: Run golangci-lint code quality checks
.PHONY: lint
lint: protogen
	golangci-lint run ./...

## lint-fix: Run golangci-lint with auto-fix for common issues
.PHONY: lint-fix
lint-fix: protogen
	golangci-lint fmt
	golangci-lint run --fix ./...

## clean: Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/
	rm -rf gen/
