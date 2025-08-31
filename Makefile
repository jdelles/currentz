ifneq (,$(wildcard .env))
include .env
export $(shell sed -n 's/^\([A-Za-z_][A-Za-z0-9_]*\)=.*/\1/p' .env)
endif

.PHONY: build run migrate-up migrate-down migrate-status clean sqlc-generate deps setup-db dev-setup install-tools install-hooks verify-hooks copy-env print-env

DB_USER ?= $(shell id -un 2>/dev/null || whoami)
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_NAME ?= personal_finance
DB_URL ?= postgres://$(DB_USER)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
export DB_URL

# Build the application
build:
	go build -o bin/currentz cmd/currentz/main.go

# Run the application
run:
	go run cmd/currentz/main.go

# Install dependencies
deps:
	go mod tidy
	go mod download

# Generate sqlc code 
sqlc-generate:
	sqlc generate

MIGRATIONS_DIR ?= sql/migrations

# Database migrations 
migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" down

migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DB_URL)" status

# Clean build artifacts
clean:
	rm -rf bin/

# Setup database
setup-db:
	createdb $(DB_NAME) || true

# Install CLI tools needed for development
install-tools:
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/zricethezav/gitleaks/v8@latest

# Install repo-scoped git hooks (shared via githooks/)
install-hooks:
	@echo "🔗 Setting core.hooksPath to ./githooks"
	git config core.hooksPath githooks
	@echo "🔒 Marking hooks executable"
	chmod +x githooks/* || true
	@echo "✅ Hooks installed. They'll run on commit/push."

# Quick check that hooks are wired up
verify-hooks:
	@echo "core.hooksPath = $$(git config --get core.hooksPath)"
	@echo "Listing hooks in ./githooks:"
	@ls -l githooks || true

# Copies .env.example to .env if .env doesn't already exist
copy-env:
	@test -f .env || cp .env.example .env
	@echo "✅ .env ready"

# Prints environment variables to confirm they were set 
print-env:
	@echo "DB_USER=$(DB_USER)"
	@echo "DB_HOST=$(DB_HOST)"
	@echo "DB_PORT=$(DB_PORT)"
	@echo "DB_NAME=$(DB_NAME)"
	@echo "DB_URL=$(DB_URL)"
	@echo "MIGRATIONS_DIR=$(MIGRATIONS_DIR)"

# One-shot dev setup
dev-setup: copy-env setup-db deps sqlc-generate migrate-up install-tools install-hooks verify-hooks
