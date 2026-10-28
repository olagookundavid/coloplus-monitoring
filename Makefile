SHELL := /bin/bash
DB_URL ?= postgres://coloplus:coloplus@127.0.0.1:5432/coloplus?sslmode=disable

.PHONY: help swag run build migrate vet test test/integration test/all lint tidy db/up db/down db/psql

## help: list the available commands
help:
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

## swag: generate the Swagger docs (needed to compile; not committed)
swag:
	cd backend && swag init -g cmd/api/main.go -o docs

## run: start the API using backend/.env
run: swag
	cd backend && go run ./cmd/api

## build: build the API binary into backend/bin
build: swag
	cd backend && CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/api ./cmd/api

## migrate: apply pending database migrations
migrate:
	cd backend && go run ./cmd/api migrate

## vet: go vet both modules
vet: swag
	cd backend && go vet ./...
	cd pkg-common && go vet ./...

## test: unit tests, no database needed
test: swag
	cd backend && go test ./tests/unit/...
	cd pkg-common && go test ./...

## test/integration: tests against a real Postgres (run `make db/up` first)
test/integration: swag
	cd backend && TEST_DATABASE_URL="$(DB_URL)" go test -count=1 ./tests/integration/...

## test/all: unit and integration tests
test/all: test test/integration

## lint: golangci-lint (includes the layer import rules)
lint: swag
	cd backend && golangci-lint run ./...
	cd pkg-common && golangci-lint run ./...

## tidy: tidy both modules
tidy:
	cd backend && go mod tidy
	cd pkg-common && go mod tidy

## db/up: start the local Postgres
db/up:
	docker compose up -d --wait postgres

## db/down: stop the local Postgres (data is kept)
db/down:
	docker compose down

## db/psql: open psql on the local database
db/psql:
	docker compose exec postgres psql -U coloplus -d coloplus
