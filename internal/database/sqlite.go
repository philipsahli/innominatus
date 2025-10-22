package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// NewSQLiteDatabase creates a new database connection using SQLite
// This is useful for:
// - Quick local development without PostgreSQL
// - Fast test execution (in-memory)
// - CI environments without Docker
//
// Usage:
//   db, err := NewSQLiteDatabase(":memory:")  // In-memory (fastest)
//   db, err := NewSQLiteDatabase("./test.db") // File-based (persists)
func NewSQLiteDatabase(dbPath string) (*Database, error) {
	// Ensure directory exists for file-based databases
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open SQLite database with optimizations
	connStr := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_journal_mode=WAL&_busy_timeout=5000", dbPath)
	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure connection pool (SQLite works best with limited connections)
	db.SetMaxOpenConns(1) // SQLite serializes writes anyway
	db.SetMaxIdleConns(1)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	// Enable foreign keys (not enabled by default in SQLite)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return &Database{db: db}, nil
}

// NewDatabaseAuto creates a database connection based on DB_DRIVER environment variable
// Supports:
//   - DB_DRIVER=postgres (default) - Uses PostgreSQL
//   - DB_DRIVER=sqlite - Uses SQLite
//
// SQLite configuration (when DB_DRIVER=sqlite):
//   - DB_PATH=./data/innominatus.db (default) - Database file path
//   - DB_PATH=:memory: - In-memory database (fastest, no persistence)
func NewDatabaseAuto() (*Database, error) {
	driver := getEnvWithDefault("DB_DRIVER", "postgres")

	switch driver {
	case "sqlite":
		dbPath := getEnvWithDefault("DB_PATH", "./data/innominatus.db")
		return NewSQLiteDatabase(dbPath)

	case "postgres":
		return NewDatabase()

	default:
		return nil, fmt.Errorf("unsupported database driver: %s (supported: postgres, sqlite)", driver)
	}
}
