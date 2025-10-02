package validation

import (
	"database/sql"
	"fmt"
	"innominatus/internal/database"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// DatabaseValidator validates database configuration and connectivity
type DatabaseValidator struct {
	config database.Config
}

// NewDatabaseValidator creates a new database validator
func NewDatabaseValidator() *DatabaseValidator {
	// Use the same environment variable logic as the database package
	config := database.Config{
		Host:     getEnvWithDefault("DB_HOST", "localhost"),
		Port:     getEnvWithDefault("DB_PORT", "5432"),
		User:     getEnvWithDefault("DB_USER", "postgres"),
		Password: getEnvWithDefault("DB_PASSWORD", ""),
		DBName:   getEnvWithDefault("DB_NAME", "idp_orchestrator"),
		SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
	}

	return &DatabaseValidator{config: config}
}

// NewDatabaseValidatorWithConfig creates a validator with custom config
func NewDatabaseValidatorWithConfig(config database.Config) *DatabaseValidator {
	return &DatabaseValidator{config: config}
}

// Validate validates the database configuration and connectivity
func (v *DatabaseValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Component: "Database Configuration",
	}

	// Validate configuration values
	v.validateConfiguration(result)

	// If configuration is valid, test connectivity
	if len(result.Errors) == 0 {
		v.testConnectivity(result)
		v.validateSchema(result)
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// GetComponent returns the component name
func (v *DatabaseValidator) GetComponent() string {
	return "Database Configuration"
}

func (v *DatabaseValidator) validateConfiguration(result *ValidationResult) {
	// Validate host
	if err := ValidateRequired("DB_HOST", v.config.Host); err != nil {
		result.Errors = append(result.Errors, err.Error())
	}

	// Validate port
	if err := ValidateRequired("DB_PORT", v.config.Port); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		port, err := strconv.Atoi(v.config.Port)
		if err != nil {
			result.Errors = append(result.Errors, "DB_PORT must be a valid integer")
		} else if port < 1 || port > 65535 {
			result.Errors = append(result.Errors, "DB_PORT must be between 1 and 65535")
		} else if port != 5432 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Non-standard PostgreSQL port %d - ensure this is correct", port))
		}
	}

	// Validate user
	if err := ValidateRequired("DB_USER", v.config.User); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else if v.config.User == "postgres" && v.config.Host != "localhost" {
		result.Warnings = append(result.Warnings, "Using 'postgres' superuser for remote database - consider using a dedicated application user")
	}

	// Validate password (only warn if empty for non-local connections)
	if v.config.Password == "" {
		if v.config.Host != "localhost" && v.config.Host != "127.0.0.1" {
			result.Warnings = append(result.Warnings, "Empty database password for remote connection - this may cause authentication failures")
		}
	}

	// Validate database name
	if err := ValidateRequired("DB_NAME", v.config.DBName); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		// Database name should follow PostgreSQL naming conventions
		if err := ValidateRegex("DB_NAME", v.config.DBName,
			`^[a-z_][a-z0-9_]*$`, "lowercase letters, numbers, and underscores only"); err != nil {
			result.Warnings = append(result.Warnings, err.Error())
		}
	}

	// Validate SSL mode
	allowedSSLModes := []string{"disable", "require", "verify-ca", "verify-full"}
	if err := ValidateEnum("DB_SSLMODE", v.config.SSLMode, allowedSSLModes); err != nil {
		result.Errors = append(result.Errors, err.Error())
	} else {
		// Warn about insecure SSL modes in production-like environments
		if v.config.SSLMode == "disable" {
			if !isLocalDatabase(v.config.Host) {
				result.Warnings = append(result.Warnings, "SSL is disabled for remote database connection - consider enabling SSL for security")
			}
		}
	}
}

func (v *DatabaseValidator) testConnectivity(result *ValidationResult) {
	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		v.config.Host, v.config.Port, v.config.User, v.config.Password, v.config.DBName, v.config.SSLMode)

	// Test connection with timeout
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create database connection: %s - server will run without database features", err.Error()))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("Warning: Failed to close database connection: %v\n", err)
		}
	}()

	// Set connection timeout
	db.SetConnMaxLifetime(5 * time.Second)

	// Test ping
	if err := db.Ping(); err != nil {
		if strings.Contains(err.Error(), "database") && strings.Contains(err.Error(), "does not exist") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Database '%s' does not exist - it will need to be created", v.config.DBName))
		} else if strings.Contains(err.Error(), "authentication failed") || strings.Contains(err.Error(), "role") && strings.Contains(err.Error(), "does not exist") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Database authentication failed: %s - server will run without database features", err.Error()))
		} else if strings.Contains(err.Error(), "connection refused") {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot connect to database server at %s:%s - server will run without database features", v.config.Host, v.config.Port))
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Database connectivity test failed: %s - server will run without database features", err.Error()))
		}
		return
	}

	// Test basic query execution
	var version string
	err = db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot query database version: %s", err.Error()))
	} else {
		// Check PostgreSQL version
		v.checkPostgreSQLVersion(version, result)
	}

	// Check database permissions
	v.checkDatabasePermissions(db, result)
}

func (v *DatabaseValidator) validateSchema(result *ValidationResult) {
	// Try to create a database connection to check schema
	db, err := database.NewDatabaseWithConfig(v.config)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot validate database schema: %s", err.Error()))
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("Warning: Failed to close database connection: %v\n", err)
		}
	}()

	// Check if required tables exist
	requiredTables := []string{"workflow_executions", "workflow_step_executions"}
	for _, table := range requiredTables {
		exists, err := v.tableExists(db.GetDB(), table)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot check table '%s': %s", table, err.Error()))
		} else if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Required table '%s' does not exist - run schema initialization", table))
		}
	}

	// If tables exist, check basic structure
	if exists, _ := v.tableExists(db.GetDB(), "workflow_executions"); exists {
		v.validateWorkflowExecutionsTable(db.GetDB(), result)
	}
}

func (v *DatabaseValidator) checkPostgreSQLVersion(version string, result *ValidationResult) {
	// Extract version number (format: "PostgreSQL 13.4 on ...")
	if strings.Contains(version, "PostgreSQL") {
		parts := strings.Fields(version)
		if len(parts) >= 2 {
			versionStr := parts[1]
			// Basic version check - warn if very old
			if strings.HasPrefix(versionStr, "9.") || strings.HasPrefix(versionStr, "10.") {
				result.Warnings = append(result.Warnings, fmt.Sprintf("PostgreSQL version %s is quite old - consider upgrading for better performance and security", versionStr))
			}
		}
	}
}

func (v *DatabaseValidator) checkDatabasePermissions(db *sql.DB, result *ValidationResult) {
	// Test CREATE TABLE permission
	testTableName := "temp_validation_test_table"
	_, err := db.Exec(fmt.Sprintf("CREATE TEMP TABLE %s (id INTEGER)", testTableName))
	if err != nil {
		if strings.Contains(err.Error(), "permission denied") {
			result.Warnings = append(result.Warnings, "Database user lacks CREATE TABLE permissions - this may limit functionality")
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Could not test CREATE permissions: %s", err.Error()))
		}
		return // Skip other permission tests if CREATE fails
	}

	// Test INSERT permission
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (id) VALUES (1)", testTableName))
	if err != nil && strings.Contains(err.Error(), "permission denied") {
		result.Warnings = append(result.Warnings, "Database user lacks INSERT permissions - this may limit functionality")
	}

	// Test UPDATE permission
	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET id = 2 WHERE id = 1", testTableName))
	if err != nil && strings.Contains(err.Error(), "permission denied") {
		result.Warnings = append(result.Warnings, "Database user lacks UPDATE permissions - this may limit functionality")
	}

	// Test DELETE permission
	_, err = db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = 2", testTableName))
	if err != nil && strings.Contains(err.Error(), "permission denied") {
		result.Warnings = append(result.Warnings, "Database user lacks DELETE permissions - this may limit functionality")
	}
}

func (v *DatabaseValidator) tableExists(db *sql.DB, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	return exists, err
}

func (v *DatabaseValidator) validateWorkflowExecutionsTable(db *sql.DB, result *ValidationResult) {
	// Check for required columns
	requiredColumns := []string{
		"id", "application_name", "workflow_name", "status",
		"started_at", "total_steps", "created_at", "updated_at",
	}

	for _, column := range requiredColumns {
		exists, err := v.columnExists(db, "workflow_executions", column)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Cannot check column 'workflow_executions.%s': %s", column, err.Error()))
		} else if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Required column 'workflow_executions.%s' is missing", column))
		}
	}
}

func (v *DatabaseValidator) columnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = $1
			AND column_name = $2
		)
	`
	var exists bool
	err := db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists, err
}

// Helper functions
func isLocalDatabase(host string) bool {
	localHosts := []string{"localhost", "127.0.0.1", "::1"}
	for _, localHost := range localHosts {
		if host == localHost {
			return true
		}
	}
	return false
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := getEnv(key); value != "" {
		return value
	}
	return defaultValue
}

// Simple environment variable getter for validation package
func getEnv(key string) string {
	return os.Getenv(key)
}
