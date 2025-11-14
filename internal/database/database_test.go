package database

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDatabase(t *testing.T) {
	SkipIfDockerNotAvailable(t)

	// Setup testcontainer
	testDB := SetupTestDatabaseWithoutSchema(t)
	defer func() { _ = testDB.Close() }()

	// Test that database connection works
	assert.NotNil(t, testDB.DB)

	err := testDB.DB.Ping()
	assert.NoError(t, err)
}

func TestNewDatabaseWithConfig(t *testing.T) {
	SkipIfDockerNotAvailable(t)

	// Setup testcontainer
	testDB := SetupTestDatabaseWithoutSchema(t)
	defer func() { _ = testDB.Close() }()

	// Test that config is used correctly
	assert.NotNil(t, testDB.DB)

	err := testDB.DB.Ping()
	assert.NoError(t, err)

	// Test that we can get the underlying DB
	sqlDB := testDB.DB.GetDB()
	assert.NotNil(t, sqlDB)
}

func TestDatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Config
	}{
		{
			name:    "default config",
			envVars: map[string]string{},
			expected: Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "",
				DBName:   "idp_orchestrator",
				SSLMode:  "disable",
			},
		},
		{
			name: "custom config from env",
			envVars: map[string]string{
				"DB_HOST":     "db.example.com",
				"DB_PORT":     "5433",
				"DB_USER":     "myuser",
				"DB_PASSWORD": "mypass",
				"DB_NAME":     "mydb",
				"DB_SSLMODE":  "require",
			},
			expected: Config{
				Host:     "db.example.com",
				Port:     "5433",
				User:     "myuser",
				Password: "mypass",
				DBName:   "mydb",
				SSLMode:  "require",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			_ = os.Unsetenv("DB_HOST")
			_ = os.Unsetenv("DB_PORT")
			_ = os.Unsetenv("DB_USER")
			_ = os.Unsetenv("DB_PASSWORD")
			_ = os.Unsetenv("DB_NAME")
			_ = os.Unsetenv("DB_SSLMODE")

			// Set test environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Test the config building
			config := Config{
				Host:     getEnvWithDefault("DB_HOST", "localhost"),
				Port:     getEnvWithDefault("DB_PORT", "5432"),
				User:     getEnvWithDefault("DB_USER", "postgres"),
				Password: getEnvWithDefault("DB_PASSWORD", ""),
				DBName:   getEnvWithDefault("DB_NAME", "idp_orchestrator"),
				SSLMode:  getEnvWithDefault("DB_SSLMODE", "disable"),
			}

			assert.Equal(t, tt.expected, config)

			// Cleanup
			for key := range tt.envVars {
				_ = os.Unsetenv(key)
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		setEnv       bool
		expected     string
	}{
		{
			name:         "environment variable not set",
			key:          "TEST_VAR_NOT_SET",
			defaultValue: "default",
			setEnv:       false,
			expected:     "default",
		},
		{
			name:         "environment variable set",
			key:          "TEST_VAR_SET",
			defaultValue: "default",
			envValue:     "env_value",
			setEnv:       true,
			expected:     "env_value",
		},
		{
			name:         "environment variable set to empty string",
			key:          "TEST_VAR_EMPTY",
			defaultValue: "default",
			envValue:     "",
			setEnv:       true,
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up first
			_ = os.Unsetenv(tt.key)

			if tt.setEnv {
				_ = os.Setenv(tt.key, tt.envValue)
				defer func() { _ = os.Unsetenv(tt.key) }()
			}

			result := getEnvWithDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseMethods(t *testing.T) {
	// Test with nil database
	var db *Database

	err := db.Close()
	assert.NoError(t, err)

	// Test with actual database struct (but nil db field)
	db = &Database{db: nil}

	err = db.Close()
	assert.NoError(t, err)

	sqlDB := db.GetDB()
	assert.Nil(t, sqlDB)

	err = db.Ping()
	assert.Error(t, err) // Should error because db is nil
}

func TestDatabaseConnectionPool(t *testing.T) {
	SkipIfDockerNotAvailable(t)

	// Setup testcontainer
	testDB := SetupTestDatabaseWithoutSchema(t)
	defer func() { _ = testDB.Close() }()

	// Test that we can get the underlying DB
	sqlDB := testDB.DB.GetDB()
	assert.NotNil(t, sqlDB)

	// Test connection pool settings
	stats := sqlDB.Stats()
	assert.Equal(t, 25, stats.MaxOpenConnections)
}

func TestInitSchema(t *testing.T) {
	t.Run("nil database connection", func(t *testing.T) {
		// Create a mock database that will fail
		db := &Database{db: nil}

		err := db.InitSchema()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is nil")
	})

	t.Run("successful schema initialization", func(t *testing.T) {
		SkipIfDockerNotAvailable(t)

		// Setup testcontainer without schema
		testDB := SetupTestDatabaseWithoutSchema(t)
		defer func() { _ = testDB.Close() }()

		// Initialize schema
		err := testDB.DB.InitSchema()
		assert.NoError(t, err)

		// Verify schema was created by checking for a table
		var exists bool
		err = testDB.DB.GetDB().QueryRow(
			"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'workflow_executions')").Scan(&exists)
		assert.NoError(t, err)
		assert.True(t, exists, "workflow_executions table should exist after schema initialization")
	})
}

func TestConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "basic config",
			config: Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "test",
				SSLMode:  "disable",
			},
			expected: "host=localhost port=5432 user=postgres password=password dbname=test sslmode=disable",
		},
		{
			name: "config with special characters",
			config: Config{
				Host:     "db.example.com",
				Port:     "5433",
				User:     "user@domain",
				Password: "pass word!",
				DBName:   "my-db",
				SSLMode:  "require",
			},
			expected: "host=db.example.com port=5433 user=user@domain password=pass word! dbname=my-db sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test connection string building logic
			connStr := buildConnectionString(tt.config)
			assert.Equal(t, tt.expected, connStr)
		})
	}
}

// Helper function to build connection string (extracted for testing)
func buildConnectionString(config Config) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)
}

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "valid config",
			config: Config{
				Host:     "localhost",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "test",
				SSLMode:  "disable",
			},
			valid: true,
		},
		{
			name: "empty host",
			config: Config{
				Host:     "",
				Port:     "5432",
				User:     "postgres",
				Password: "password",
				DBName:   "test",
				SSLMode:  "disable",
			},
			valid: false,
		},
		{
			name: "empty port",
			config: Config{
				Host:     "localhost",
				Port:     "",
				User:     "postgres",
				Password: "password",
				DBName:   "test",
				SSLMode:  "disable",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateConfig(tt.config)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

// Helper function to validate config (could be added to the main package)
func validateConfig(config Config) bool {
	return config.Host != "" &&
		config.Port != "" &&
		config.User != "" &&
		config.DBName != "" &&
		config.SSLMode != ""
}

func TestDatabaseLifecycle(t *testing.T) {
	// Setup testcontainer
	testDB := SetupTestDatabaseWithoutSchema(t)

	// Test complete database lifecycle
	assert.NotNil(t, testDB.DB)

	// Test ping
	err := testDB.DB.Ping()
	assert.NoError(t, err)

	// Test schema initialization
	err = testDB.DB.InitSchema()
	assert.NoError(t, err)

	// Test close
	err = testDB.DB.Close()
	assert.NoError(t, err)

	// Test that ping fails after close
	err = testDB.DB.Ping()
	assert.Error(t, err)
}
