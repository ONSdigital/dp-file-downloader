SHELL=bash
MAIN=dp-file-downloader

BUILD_DIR=build
BUILD_ARCH=$(GOOS)-$(GOARCH)

BIN_DIR ?= $(BUILD_DIR)/$(BUILD_ARCH)

export GOOS=$(shell go env GOOS)
export GOARCH=$(shell go env GOARCH)

.PHONY: all
all: audit test build

.PHONY: audit
audit:
	nancy go.sum

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/dp-file-downloader cmd/$(MAIN)/main.go

.PHONY: debug
debug: build
	HUMAN_LOG=1 go run -race cmd/$(MAIN)/main.go

.PHONY: test
test:
	go test -cover $(shell go list ./... | grep -v /vendor/)
