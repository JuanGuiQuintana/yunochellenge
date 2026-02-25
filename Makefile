.PHONY: run migrate seed test test-unit lint docker-up docker-down build

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

migrate:
	go run ./cmd/api --migrate-only

seed:
	psql $$DATABASE_URL -f scripts/seed.sql

test:
	TEST_DATABASE_URL=$$TEST_DATABASE_URL go test ./... -v -race

test-unit:
	go test ./internal/scoring/... ./internal/processor/... ./internal/domain/... -v

lint:
	golangci-lint run ./...

docker-up:
	docker-compose up -d postgres

docker-down:
	docker-compose down
