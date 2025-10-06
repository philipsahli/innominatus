package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/lib/pq"
)

// Database wraps the SQL database connection
type Database struct {
	db *sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewDatabase creates a new database connection
func NewDatabase() (*Database, error) {
	config := Config{
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "5432"),
		User:     getEnvWithDefault("DB_USER", "postgres"),
		Password: getEnvWithDefault("DB_PASSWORD", ""),
		DBName:   getEnvWithDefault("DB_NAME", "idp_orchestrator"),
		SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
	}

	// Build connection string - omit password if empty to avoid lib/pq default behavior
	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.DBName, config.SSLMode)
	if config.Password != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
	}

	fmt.Printf("DEBUG: NewDatabase connecting to: %s (DBName=%s)\n", connStr, config.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Verify which database we actually connected to
	var actualDB string
	if err := db.QueryRow("SELECT current_database()").Scan(&actualDB); err != nil {
		fmt.Printf("WARNING: Failed to verify database connection: %v\n", err)
	} else {
		fmt.Printf("DEBUG: NewDatabase - verified connection to database: %s\n", actualDB)
	}

	result := &Database{db: db}
	fmt.Printf("DEBUG: NewDatabase - returning Database pointer: %p\n", result)
	return result, nil
}

// NewDatabaseWithConfig creates a new database connection with custom config
func NewDatabaseWithConfig(config Config) (*Database, error) {
	// Build connection string - omit password if empty to avoid lib/pq default behavior
	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.DBName, config.SSLMode)
	if config.Password != "" {
		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

// GetDB returns the underlying sql.DB instance
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// DB returns the underlying sql.DB instance (alias for GetDB)
func (d *Database) DB() *sql.DB {
	return d.db
}

// Ping tests the database connection
func (d *Database) Ping() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database connection is nil")
	}
	return d.db.Ping()
}

// InitSchema initializes the database schema
func (d *Database) InitSchema() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	schema := `
-- Workflow executions table
CREATE TABLE IF NOT EXISTS workflow_executions (
    id SERIAL PRIMARY KEY,
    application_name VARCHAR(255) NOT NULL,
    workflow_name VARCHAR(255) NOT NULL DEFAULT 'deploy',
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE NULL,
    error_message TEXT NULL,
    total_steps INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Workflow step executions table
CREATE TABLE IF NOT EXISTS workflow_step_executions (
    id SERIAL PRIMARY KEY,
    workflow_execution_id INTEGER NOT NULL REFERENCES workflow_executions(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    step_name VARCHAR(255) NOT NULL,
    step_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP WITH TIME ZONE NULL,
    completed_at TIMESTAMP WITH TIME ZONE NULL,
    duration_ms INTEGER NULL,
    error_message TEXT NULL,
    step_config JSONB NULL,
    output_logs TEXT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_workflow_executions_app_name ON workflow_executions(application_name);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_executions_started_at ON workflow_executions(started_at);

CREATE INDEX IF NOT EXISTS idx_workflow_step_executions_workflow_id ON workflow_step_executions(workflow_execution_id);
CREATE INDEX IF NOT EXISTS idx_workflow_step_executions_status ON workflow_step_executions(status);
CREATE INDEX IF NOT EXISTS idx_workflow_step_executions_step_number ON workflow_step_executions(step_number);

-- Update trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Update triggers
DROP TRIGGER IF EXISTS update_workflow_executions_updated_at ON workflow_executions;
CREATE TRIGGER update_workflow_executions_updated_at
    BEFORE UPDATE ON workflow_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_step_executions_updated_at ON workflow_step_executions;
CREATE TRIGGER update_workflow_step_executions_updated_at
    BEFORE UPDATE ON workflow_step_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Resource instances table for lifecycle tracking
CREATE TABLE IF NOT EXISTS resource_instances (
    id SERIAL PRIMARY KEY,
    application_name VARCHAR(255) NOT NULL,
    resource_name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL DEFAULT 'requested',
    health_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    configuration JSONB NOT NULL DEFAULT '{}',
    provider_id VARCHAR(255) NULL,
    provider_metadata JSONB NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_health_check TIMESTAMP WITH TIME ZONE NULL,
    error_message TEXT NULL,
    UNIQUE(application_name, resource_name)
);

-- Resource state transitions for audit trail
CREATE TABLE IF NOT EXISTS resource_state_transitions (
    id SERIAL PRIMARY KEY,
    resource_instance_id INTEGER NOT NULL REFERENCES resource_instances(id) ON DELETE CASCADE,
    from_state VARCHAR(50) NOT NULL,
    to_state VARCHAR(50) NOT NULL,
    reason TEXT NOT NULL,
    transitioned_by VARCHAR(255) NOT NULL,
    transitioned_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB NULL
);

-- Resource health checks
CREATE TABLE IF NOT EXISTS resource_health_checks (
    id SERIAL PRIMARY KEY,
    resource_instance_id INTEGER NOT NULL REFERENCES resource_instances(id) ON DELETE CASCADE,
    check_type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL,
    checked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    response_time INTEGER NULL,
    error_message TEXT NULL,
    metrics JSONB NULL
);

-- Resource dependencies
CREATE TABLE IF NOT EXISTS resource_dependencies (
    id SERIAL PRIMARY KEY,
    resource_instance_id INTEGER NOT NULL REFERENCES resource_instances(id) ON DELETE CASCADE,
    depends_on_id INTEGER NOT NULL REFERENCES resource_instances(id) ON DELETE CASCADE,
    dependency_type VARCHAR(50) NOT NULL DEFAULT 'hard',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(resource_instance_id, depends_on_id)
);

-- Indexes for resource lifecycle tables
CREATE INDEX IF NOT EXISTS idx_resource_instances_app_name ON resource_instances(application_name);
CREATE INDEX IF NOT EXISTS idx_resource_instances_state ON resource_instances(state);
CREATE INDEX IF NOT EXISTS idx_resource_instances_type ON resource_instances(resource_type);
CREATE INDEX IF NOT EXISTS idx_resource_instances_health ON resource_instances(health_status);
CREATE INDEX IF NOT EXISTS idx_resource_instances_updated ON resource_instances(updated_at);

CREATE INDEX IF NOT EXISTS idx_resource_state_transitions_resource_id ON resource_state_transitions(resource_instance_id);
CREATE INDEX IF NOT EXISTS idx_resource_state_transitions_transitioned_at ON resource_state_transitions(transitioned_at);

CREATE INDEX IF NOT EXISTS idx_resource_health_checks_resource_id ON resource_health_checks(resource_instance_id);
CREATE INDEX IF NOT EXISTS idx_resource_health_checks_checked_at ON resource_health_checks(checked_at);

CREATE INDEX IF NOT EXISTS idx_resource_dependencies_resource_id ON resource_dependencies(resource_instance_id);
CREATE INDEX IF NOT EXISTS idx_resource_dependencies_depends_on ON resource_dependencies(depends_on_id);

-- Update triggers for resource instances
DROP TRIGGER IF EXISTS update_resource_instances_updated_at ON resource_instances;
CREATE TRIGGER update_resource_instances_updated_at
    BEFORE UPDATE ON resource_instances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Check constraint for valid resource states
ALTER TABLE resource_instances DROP CONSTRAINT IF EXISTS chk_resource_state;
ALTER TABLE resource_instances ADD CONSTRAINT chk_resource_state
    CHECK (state IN ('requested', 'provisioning', 'active', 'scaling', 'updating', 'degraded', 'terminating', 'terminated', 'failed'));

-- Check constraint for valid health status
ALTER TABLE resource_instances DROP CONSTRAINT IF EXISTS chk_health_status;
ALTER TABLE resource_instances ADD CONSTRAINT chk_health_status
    CHECK (health_status IN ('healthy', 'degraded', 'unhealthy', 'unknown'));

-- Check constraint for valid dependency types
ALTER TABLE resource_dependencies DROP CONSTRAINT IF EXISTS chk_dependency_type;
ALTER TABLE resource_dependencies ADD CONSTRAINT chk_dependency_type
    CHECK (dependency_type IN ('hard', 'soft', 'optional'));
`

	_, err := d.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Run migrations from migrations/ directory
	if err := d.RunMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// RunMigrations executes SQL migration files from the migrations/ directory
func (d *Database) RunMigrations() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Get migrations directory path
	migrationsDir := "migrations"

	// Read migration files
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort files to ensure consistent execution order
	sort.Strings(files)

	// Execute each migration file
	for _, file := range files {
		log.Printf("Running migration: %s", filepath.Base(file))

		// Execute migration using psql directly for proper multi-statement support
		// This avoids issues with comment parsing and complex SQL statements
		psqlCmd := fmt.Sprintf("psql -d %s -f %s",
			getEnvWithDefault("DB_NAME", "idp_orchestrator"),
			file,
		)

		// Set environment variables for psql connection
		cmd := fmt.Sprintf("PGHOST=%s PGPORT=%s PGUSER=%s PGPASSWORD=%s %s",
			getEnvWithDefault("DB_HOST", "localhost"),
			getEnvWithDefault("DB_PORT", "5432"),
			getEnvWithDefault("DB_USER", "postgres"),
			getEnvWithDefault("DB_PASSWORD", ""),
			psqlCmd,
		)

		// Execute using shell
		output, err := exec.Command("sh", "-c", cmd).CombinedOutput() // #nosec G204 - Database migration with controlled SQL files
		if err != nil {
			log.Printf("Migration output: %s", string(output))
			log.Printf("Full error: %v", err)
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}

		log.Printf("Successfully executed migration: %s", filepath.Base(file))
	}

	if len(files) == 0 {
		log.Printf("No migration files found in %s", migrationsDir)
	} else {
		log.Printf("Successfully executed %d migration(s)", len(files))
	}

	return nil
}

// CleanDatabase truncates all tables, removing all data while preserving schema
// This is intended for demo/testing environments only
func (d *Database) CleanDatabase() error {
	if d == nil || d.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Truncate tables in order (respecting foreign key constraints)
	// CASCADE will automatically clean child tables
	truncateSQL := `
-- Truncate workflow tables (CASCADE will clean workflow_step_executions)
TRUNCATE TABLE workflow_executions CASCADE;

-- Truncate resource tables (CASCADE will clean transitions, health checks, dependencies)
TRUNCATE TABLE resource_instances CASCADE;

-- Truncate graph tables (CASCADE will clean nodes, edges, graph_runs)
TRUNCATE TABLE apps CASCADE;
`

	_, err := d.db.Exec(truncateSQL)
	if err != nil {
		return fmt.Errorf("failed to clean database: %w", err)
	}

	return nil
}

// APIKeyRecord represents an API key stored in the database
type APIKeyRecord struct {
	ID         int64
	Username   string
	KeyHash    string
	KeyName    string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  time.Time
}

// CreateAPIKey stores an API key in the database (for OIDC users)
func (d *Database) CreateAPIKey(username, keyHash, keyName string, expiresAt time.Time) error {
	query := `
		INSERT INTO user_api_keys (username, key_hash, key_name, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := d.db.Exec(query, username, keyHash, keyName, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

// GetAPIKeys retrieves all API keys for a user from the database
func (d *Database) GetAPIKeys(username string) ([]APIKeyRecord, error) {
	query := `
		SELECT id, username, key_hash, key_name, created_at, last_used_at, expires_at
		FROM user_api_keys
		WHERE username = $1
		ORDER BY created_at DESC
	`
	rows, err := d.db.Query(query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to query API keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var keys []APIKeyRecord
	for rows.Next() {
		var key APIKeyRecord
		err := rows.Scan(&key.ID, &key.Username, &key.KeyHash, &key.KeyName,
			&key.CreatedAt, &key.LastUsedAt, &key.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return keys, nil
}

// UpdateAPIKeyLastUsed updates the last_used_at timestamp for an API key
func (d *Database) UpdateAPIKeyLastUsed(keyHash string) error {
	query := `
		UPDATE user_api_keys
		SET last_used_at = NOW()
		WHERE key_hash = $1
	`
	_, err := d.db.Exec(query, keyHash)
	if err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	return nil
}

// DeleteAPIKey removes an API key from the database
func (d *Database) DeleteAPIKey(username, keyName string) error {
	query := `
		DELETE FROM user_api_keys
		WHERE username = $1 AND key_name = $2
	`
	result, err := d.db.Exec(query, username, keyName)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// GetUserByAPIKeyHash retrieves user information by API key hash
func (d *Database) GetUserByAPIKeyHash(keyHash string) (username string, team string, role string, err error) {
	// First check if key exists and is not expired
	query := `
		SELECT username
		FROM user_api_keys
		WHERE key_hash = $1 AND expires_at > NOW()
	`
	err = d.db.QueryRow(query, keyHash).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", fmt.Errorf("API key not found or expired")
		}
		return "", "", "", fmt.Errorf("failed to query API key: %w", err)
	}

	// OIDC users don't have persistent records, so we need to get user info from session
	// For now, return the username and default team/role
	// The actual user object will be reconstructed from session data
	return username, "oidc-users", "user", nil
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
