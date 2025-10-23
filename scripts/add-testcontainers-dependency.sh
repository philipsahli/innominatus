#!/bin/bash
# Script to add testcontainers-go dependencies
# Run this script to add the required dependencies for database testing

set -e

echo "Adding testcontainers-go dependencies..."

# Add main testcontainers package
go get github.com/testcontainers/testcontainers-go@latest

# Add PostgreSQL module
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest

# Tidy up dependencies
go mod tidy

echo "✅ Testcontainers dependencies added successfully!"
echo ""
echo "Now you can run database tests with:"
echo "  go test ./internal/database/... -v"
