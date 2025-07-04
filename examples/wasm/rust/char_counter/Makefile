# Variables
PLUGIN_NAME := plugin
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

## build: Build the WASM plugin
.PHONY: build
build:
	xtp plugin build

## format: Format the Rust code
.PHONY: format
format:
	cargo fmt

## check: Check the Rust code for errors
.PHONY: check
check:
	cargo check

## test: Run plugin tests (WASM plugins require special test harness)
.PHONY: test
test: build
	@if ! command -v xtp &> /dev/null; then \
		echo "Error: xtp CLI not found in PATH. Please install xtp first."; \
		echo "Visit: https://docs.xtp.dylibso.com/docs/install"; \
		exit 1; \
	fi
	rustup target add wasm32-unknown-unknown
	cd test && cargo build --target wasm32-unknown-unknown --release
	xtp plugin test target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm \
		--with test/target/wasm32-unknown-unknown/release/test.wasm

## test-verbose: Run plugin tests with verbose output
.PHONY: test-verbose
test-verbose: build
	@if ! command -v xtp &> /dev/null; then \
		echo "Error: xtp CLI not found in PATH. Please install xtp first."; \
		echo "Visit: https://docs.xtp.dylibso.com/docs/install"; \
		exit 1; \
	fi
	rustup target add wasm32-unknown-unknown
	cd test && cargo build --target wasm32-unknown-unknown --release
	xtp plugin test target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm \
		--with test/target/wasm32-unknown-unknown/release/test.wasm \
		--verbose

## clean: Clean build artifacts
.PHONY: clean
clean:
	cargo clean
	cd test && cargo clean
	rm -f target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm
	rm -f mock-input.json