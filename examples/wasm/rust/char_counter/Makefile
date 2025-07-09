# Variables
PLUGIN_NAME := plugin
VERSION := 0.1.0

.PHONY: all
all: help

## setup: Check dependencies and install required WASM targets
.PHONY: setup
setup:
	@command -v cargo >/dev/null 2>&1 || { echo "Error: cargo not found. Install Rust from https://rustup.rs/"; exit 1; }
	@command -v rustup >/dev/null 2>&1 || { echo "Error: rustup not found. Check Rust installation."; exit 1; }
	@rustup target list --installed | grep -q '^wasm32-wasip1$$' || rustup target add wasm32-wasip1
	@rustup target list --installed | grep -q '^wasm32-unknown-unknown$$' || rustup target add wasm32-unknown-unknown

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
build: setup
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
	cd test && cargo build --target wasm32-unknown-unknown --release
	xtp plugin test target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm \
		--with test/target/wasm32-unknown-unknown/release/test.wasm \
		--verbose

## build-base64: Build WASM and output base64 encoding
.PHONY: build-base64
build-base64: build
	@base64 -i target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm 2>/dev/null

## clean: Clean build artifacts
.PHONY: clean
clean:
	cargo clean
	cd test && cargo clean
	rm -f target/wasm32-wasip1/release/$(PLUGIN_NAME).wasm
	rm -f mock-input.json