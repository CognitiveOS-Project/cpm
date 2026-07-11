SHELL := /bin/sh
.SHELLFLAGS := -eu -c
.ONESHELL:
.DELETE_ON_ERROR:

BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin
GO := go

.PHONY: build test lint clean build-dependencies
 
build-dependencies:
	@echo "No external dependencies for cpm"
 
build: $(BIN_DIR)/cpm


$(BIN_DIR)/cpm:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags="-s -w" -o $@ ./cmd/cpm
	@echo "  -> $@"

test:
	$(GO) test ./... -v -count=1

lint:
	shellcheck scripts/build.sh
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)
