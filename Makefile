# Variables
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
build:
	go build -ldflags "-X main.Version=$(VERSION)" -o bin/$(BINARY_NAME) ./cmd/firelynx

## install: Install the binary
.PHONY: install
install: protogen
	go install -ldflags "-X main.Version=$(VERSION)" ./cmd/firelynx

## protogen: Generate code from protobuf definitions
.PHONY: protogen
protogen: clean
	@echo "Validating protobuf files..."
	@protoc --proto_path=proto --descriptor_set_out=/dev/null $$(find proto -name "*.proto")
	@echo "Generating protobuf code..."
	@mkdir -p gen
	protoc \
		--proto_path=proto \
		--plugin=protoc-gen-go=$$(go tool -n google.golang.org/protobuf/cmd/protoc-gen-go) \
		--plugin=protoc-gen-go-grpc=$$(go tool -n google.golang.org/grpc/cmd/protoc-gen-go-grpc) \
		--go_out=gen \
		--go_opt=paths=source_relative \
		--go-grpc_out=gen \
		--go-grpc_opt=paths=source_relative \
		$$(find proto -name "*.proto")

## test: Run tests with race detection and coverage
.PHONY: test
test:
	go test -race -cover -timeout 2m ./...

## test-short: Run unit tests in short mode (fast)
.PHONY: test-short
test-short:
	go test -short -timeout 1m ./...

## test-e2e: Run end-to-end tests
.PHONY: test-e2e
test-e2e:
	go test -race -timeout 3m -tags e2e ./...

## test-integration: Run integration tests
.PHONY: test-integration
test-integration:
	go test -count 1 -race -timeout 1m -tags integration ./internal/server/integration_tests/...

## test-all: Run all tests (unit, integration, and e2e)
.PHONY: test-all
test-all:
	go test -race -cover -timeout 5m -tags "integration e2e" ./...

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
