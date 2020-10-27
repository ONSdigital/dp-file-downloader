SHELL=bash
MAIN=dp-file-downloader

BUILD_DIR=build
BUILD_ARCH=$(GOOS)-$(GOARCH)

BIN_DIR ?= $(BUILD_DIR)/$(BUILD_ARCH)

BUILD_TIME=$(shell date +%s)
GIT_COMMIT=$(shell git rev-parse HEAD)
VERSION ?= $(shell git tag --points-at HEAD | grep ^v | head -n 1)

export GOOS=$(shell go env GOOS)
export GOARCH=$(shell go env GOARCH)

.PHONY: all
all: audit test build

.PHONY: audit
audit:
	nancy go.sum

.PHONY: build
build:
	go build -tags 'production' -o $(BIN_DIR)/dp-file-downloader -ldflags "-X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" cmd/dp-file-downloader/main.go
	@mkdir -p $(BIN_DIR) 

.PHONY: debug
debug: 
	go build -tags 'debug' -race -o $(BUILD_DIR)/dp-file-downloader -ldflags "-X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT) -X main.Version=$(VERSION)" cmd/dp-file-downloader/main.go
	HUMAN_LOG=1 DEBUG=1 $(BUILD_DIR)/dp-file-downloader

.PHONY: test
test:
	go test -cover $(shell go list ./... | grep -v /vendor/)
