#!/bin/bash
set -e

echo "Setting up Personal Finance App with sqlc..."

# Install required tools
echo "Installing tools..."
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Create database if it doesn't exist
echo "Setting up database..."
createdb personal_finance 2>/dev/null || echo "Database already exists"

# Install Go dependencies
echo "Installing dependencies..."
go mod tidy

# Generate sqlc code
echo "Generating sqlc code..."
sqlc generate

# Run migrations
echo "Running migrations..."
make migrate-up

echo "Setup complete! Run 'make run' to start the application."