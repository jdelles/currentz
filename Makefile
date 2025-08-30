.PHONY: build run migrate-up migrate-down migrate-status clean sqlc-generate deps setup-db dev-setup install-tools install-hooks verify-hooks

# Default database URL for development
DB_URL ?= postgres://jamesdelles@localhost:5432/personal_finance?sslmode=disable

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

# Generate sqlc code (uses paths from sqlc.yaml)
sqlc-generate:
	sqlc generate

# Database migrations (point to your actual migrations folder)
migrate-up:
	goose -dir sql/migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir sql/migrations postgres "$(DB_URL)" down

migrate-status:
	goose -dir sql/migrations postgres "$(DB_URL)" status

# Clean build artifacts
clean:
	rm -rf bin/

# Setup database
setup-db:
	createdb personal_finance || true

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

# One-shot dev setup
dev-setup: setup-db deps sqlc-generate migrate-up install-tools install-hooks verify-hooks
