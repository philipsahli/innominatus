#!/bin/bash
# Script to add SQLite dependencies
# Run this script to add the required dependencies for SQLite support

set -e

echo "Adding SQLite dependencies..."

# Add SQLite driver
go get github.com/mattn/go-sqlite3@latest

# Tidy up dependencies
go mod tidy

echo "✅ SQLite dependencies added successfully!"
echo ""
echo "Now you can run tests with SQLite:"
echo "  TEST_DB_DRIVER=sqlite go test ./..."
echo "  make test-sqlite"
