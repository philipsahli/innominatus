package database

import (
	"database/sql"
	"fmt"
	"os"
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

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

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

// NewDatabaseWithConfig creates a new database connection with custom config
func NewDatabaseWithConfig(config Config) (*Database, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

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

	return nil
}

// getEnvWithDefault returns environment variable value or default if not set
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}