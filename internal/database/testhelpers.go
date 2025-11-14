package database

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SkipIfDockerNotAvailable skips the test if Docker is not available
// This is needed for CI environments like GitHub Actions macOS runners
func SkipIfDockerNotAvailable(t *testing.T) {
	t.Helper()

	// Check if Docker CLI is available
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("Docker not found in PATH - skipping test requiring Docker")
		return
	}

	// Try to ping Docker daemon
	cmd := exec.Command("docker", "info")
	if err := cmd.Run(); err != nil {
		t.Skipf("Docker daemon not available (OS: %s) - skipping test: %v", runtime.GOOS, err)
		return
	}
}

// TestDatabase holds the test container and database connection
type TestDatabase struct {
	Container *postgres.PostgresContainer
	DB        *Database
	Config    Config
}

// SetupTestDatabase creates a PostgreSQL testcontainer and returns a connected Database instance
// The container will be automatically cleaned up when the test ends
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Cleanup container when test ends
	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate PostgreSQL container: %v", err)
		}
	})

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Get individual connection parameters
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	config := Config{
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	// Create database connection
	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v (connection string: %s)", err, connStr)
	}

	// Initialize schema
	if err := db.InitSchema(); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	return &TestDatabase{
		Container: postgresContainer,
		DB:        db,
		Config:    config,
	}
}

// SetupTestDatabaseWithoutSchema creates a PostgreSQL testcontainer without initializing schema
// Useful for testing schema initialization itself
func SetupTestDatabaseWithoutSchema(t *testing.T) *TestDatabase {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Cleanup container when test ends
	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate PostgreSQL container: %v", err)
		}
	})

	// Get connection parameters
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatalf("Failed to get container host: %v", err)
	}

	port, err := postgresContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("Failed to get container port: %v", err)
	}

	config := Config{
		Host:     host,
		Port:     port.Port(),
		User:     "testuser",
		Password: "testpass",
		DBName:   "testdb",
		SSLMode:  "disable",
	}

	// Create database connection
	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	return &TestDatabase{
		Container: postgresContainer,
		DB:        db,
		Config:    config,
	}
}

// Close closes the database connection
func (td *TestDatabase) Close() error {
	if td.DB != nil {
		return td.DB.Close()
	}
	return nil
}

// ConnectionString returns the full connection string for the test database
func (td *TestDatabase) ConnectionString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		td.Config.Host, td.Config.Port, td.Config.User, td.Config.Password, td.Config.DBName, td.Config.SSLMode)
}
