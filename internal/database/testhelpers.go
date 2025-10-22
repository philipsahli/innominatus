// testhelpers.go provides test utilities for database testing with testcontainers

//go:build !production

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDatabase wraps a Database with testcontainer lifecycle management
type TestDatabase struct {
	*Database
	container *postgres.PostgresContainer
	cleanup   func()
}

// SetupTestDatabase creates a PostgreSQL testcontainer and returns a connected Database
// The container is automatically cleaned up when the test completes via t.Cleanup()
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("idp_orchestrator_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Skipf("Failed to start PostgreSQL container (Docker not available?): %v", err)
		return nil
	}

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Parse connection string to get host, port, etc.
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get mapped port: %v", err)
	}

	// Create database config
	config := Config{
		Host:     host,
		Port:     port.Port(),
		User:     "postgres",
		Password: "postgres",
		DBName:   "idp_orchestrator_test",
		SSLMode:  "disable",
	}

	// Connect to database
	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		db.Close()
		postgresContainer.Terminate(ctx)
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Create test database wrapper
	testDB := &TestDatabase{
		Database:  db,
		container: postgresContainer,
		cleanup: func() {
			db.Close()
			if err := postgresContainer.Terminate(ctx); err != nil {
				t.Logf("Failed to terminate container: %v", err)
			}
		},
	}

	// Register cleanup
	t.Cleanup(testDB.cleanup)

	t.Logf("Test database ready at %s:%s (connection: %s)", host, port.Port(), connStr)
	return testDB
}

// SetupTestDatabaseShared creates a single PostgreSQL testcontainer for a test suite
// Use this for TestMain to share a database across all tests in a package
// Returns cleanup function that must be called in TestMain after m.Run()
func SetupTestDatabaseShared() (*Database, func(), error) {
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("idp_orchestrator_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start PostgreSQL container: %w", err)
	}

	// Get connection details
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	// Create database config
	config := Config{
		Host:     host,
		Port:     port.Port(),
		User:     "postgres",
		Password: "postgres",
		DBName:   "idp_orchestrator_test",
		SSLMode:  "disable",
	}

	// Connect to database
	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		db.Close()
		postgresContainer.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	cleanup := func() {
		db.Close()
		if err := postgresContainer.Terminate(ctx); err != nil {
			fmt.Printf("Failed to terminate container: %v\n", err)
		}
	}

	return db, cleanup, nil
}

// CleanupTestData truncates all tables for test isolation
// Use this between tests if sharing a database container
func (td *TestDatabase) CleanupTestData(t *testing.T) {
	t.Helper()

	if err := td.TruncateAllTables(); err != nil {
		t.Fatalf("Failed to cleanup test data: %v", err)
	}
}
