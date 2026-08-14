# GophKeeper — Makefile
#
# Common developer tasks. Run `make help` for the list of targets.

# ---- Build metadata injected via ldflags -----------------------------------
VERSION   ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DATE ?= $(shell date -u +%Y-%m-%d)
COMMIT    ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)

PKG        := github.com/warenik/gophkeeper/internal/buildmeta
LDFLAGS    := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).BuildDate=$(BUILD_DATE) \
	-X $(PKG).Commit=$(COMMIT)

BIN_DIR    := bin
SERVER_BIN := $(BIN_DIR)/gophkeeper-server
CLIENT_BIN := $(BIN_DIR)/gophkeeper-client

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F': ' '{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## build: build server and client binaries into ./bin
.PHONY: build
build: build-server build-client

## build-server: build the server binary
.PHONY: build-server
build-server:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(SERVER_BIN) ./cmd/server

## build-client: build the client binary
.PHONY: build-client
build-client:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(CLIENT_BIN) ./cmd/client

## run-server: run the server via `go run`
.PHONY: run-server
run-server:
	go run -ldflags "$(LDFLAGS)" ./cmd/server

## test: run all unit tests with the race detector
.PHONY: test
test:
	go test -race ./...

# Пакеты, исключаемые из измерения покрытия: сгенерированный код (pb),
# точки входа (cmd) и БД-слой (покрывается интеграционными тестами).
COVER_EXCLUDE := /internal/pb|/cmd/|/internal/server/storage/postgres

## cover: run tests and report business-logic coverage (excludes generated/cmd/db)
.PHONY: cover
cover:
	go test -race -covermode=atomic \
		-coverpkg=$$(go list ./... | grep -vE '$(COVER_EXCLUDE)' | paste -sd, -) \
		-coverprofile=coverage.out \
		$$(go list ./... | grep -vE '$(COVER_EXCLUDE)')
	go tool cover -func=coverage.out | tail -n 1

## cover-html: open the HTML coverage report
.PHONY: cover-html
cover-html: cover
	go tool cover -html=coverage.out

## test-integration: run all tests incl. DB (requires GOPHKEEPER_TEST_DSN)
.PHONY: test-integration
test-integration:
	go test -race ./...

## cover-integration: coverage incl. DB layer (requires GOPHKEEPER_TEST_DSN)
.PHONY: cover-integration
cover-integration:
	go test -race -covermode=atomic \
		-coverpkg=$$(go list ./... | grep -vE '/internal/pb|/cmd/' | paste -sd, -) \
		-coverprofile=coverage.out \
		$$(go list ./... | grep -vE '/internal/pb|/cmd/')
	go tool cover -func=coverage.out | tail -n 1

## lint: run golangci-lint
.PHONY: lint
lint:
	golangci-lint run ./...

## proto: сгенерировать Go-код из .proto (нужны protoc, protoc-gen-go, protoc-gen-go-grpc)
.PHONY: proto
proto:
	protoc \
		--proto_path=proto \
		--go_out=. --go_opt=module=github.com/warenik/gophkeeper \
		--go-grpc_out=. --go-grpc_opt=module=github.com/warenik/gophkeeper \
		proto/gophkeeper/v1/*.proto

## tidy: tidy go.mod / go.sum
.PHONY: tidy
tidy:
	go mod tidy

## fmt: format the code
.PHONY: fmt
fmt:
	gofmt -s -w .

## up: start the local stack (Postgres + server) via docker compose
.PHONY: up
up:
	docker compose -f deployments/docker-compose.yml up -d --build

## down: stop the local stack
.PHONY: down
down:
	docker compose -f deployments/docker-compose.yml down

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR) coverage.out
