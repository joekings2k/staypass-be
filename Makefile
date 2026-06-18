include .env
export

MIGRATIONS_DIR := ./migrations

.PHONY: migrate-up migrate-down  migrate-status migrate-new sqlc test-db-up test run install-hooks

migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

# make migrate-new name=add_dues
migrate-new:
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

sqlc:
	sqlc generate

test-db-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(TEST_DATABASE_URL)" up

test: test-db-up
	go test ./...

run:
	go run ./cmd/api

# Point git at the tracked hooks so CHANGELOG.md updates on every commit.
install-hooks:
	chmod +x scripts/git-hooks/*
	git config core.hooksPath scripts/git-hooks
	@echo "git hooks installed (core.hooksPath -> scripts/git-hooks)"