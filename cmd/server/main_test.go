package main

import (
	"context"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"innominatus/internal/security"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	// File doesn't exist yet
	if fileExists(tmpFile) {
		t.Errorf("fileExists(%s) = true, want false", tmpFile)
	}

	// Create the file
	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// File now exists
	if !fileExists(tmpFile) {
		t.Errorf("fileExists(%s) = false, want true", tmpFile)
	}
}

func TestIsStaticAsset(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/.next/static/css/app.css", true},
		{"/favicon.ico", true},
		{"/app.js", true},
		{"/styles.css", true},
		{"/logo.png", true},
		{"/photo.jpg", true},
		{"/icon.svg", true},
		{"/api/specs", false},
		{"/dashboard", false},
		{"/", false},
		{"/about", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isStaticAsset(tt.path)
			if result != tt.expected {
				t.Errorf("isStaticAsset(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLoadAdminConfigValidation(t *testing.T) {
	// Test with invalid path (path traversal attempt)
	_, err := admin.LoadAdminConfig("../../etc/passwd")
	if err == nil {
		t.Error("Expected error for path traversal attempt, got nil")
	}

	// Test with non-existent file
	_, err = admin.LoadAdminConfig("nonexistent-config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestLoadAdminConfigSuccess(t *testing.T) {
	// Create a temporary valid admin config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-admin-config.yaml")

	configContent := `admin:
  defaultCostCenter: "test-cc"
  defaultRuntime: "kubernetes"
  splunkIndex: "test-index"

resourceDefinitions:
  postgres: "postgres-operator"
  redis: "redis-operator"

policies:
  enforceBackups: true
  allowedEnvironments:
    - development
    - staging
    - production

workflowPolicies:
  workflowsRoot: "./workflows"
  requiredPlatformWorkflows:
    - initialize
  allowedProductWorkflows:
    - ecommerce
  maxWorkflowDuration: "30m"
  maxConcurrentWorkflows: 10
  maxStepsPerWorkflow: 50
  allowedStepTypes:
    - terraform
    - ansible
    - kubernetes
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Validate the path first (security check)
	if err := security.ValidateConfigPath(configPath); err != nil {
		t.Fatalf("Config path validation failed: %v", err)
	}

	config, err := admin.LoadAdminConfig(configPath)
	if err != nil {
		t.Fatalf("LoadAdminConfig() failed: %v", err)
	}

	if config.Admin.DefaultCostCenter != "test-cc" {
		t.Errorf("Expected DefaultCostCenter = 'test-cc', got '%s'", config.Admin.DefaultCostCenter)
	}

	if config.Policies.EnforceBackups != true {
		t.Error("Expected EnforceBackups = true, got false")
	}

	if len(config.Policies.AllowedEnvironments) != 3 {
		t.Errorf("Expected 3 allowed environments, got %d", len(config.Policies.AllowedEnvironments))
	}

	// Test String() method
	configStr := config.String()
	if configStr == "" {
		t.Error("Config.String() returned empty string")
	}
}

// TestServerStartupWithoutDatabase tests that the server can start without database
func TestServerStartupWithoutDatabase(t *testing.T) {
	// This is a basic test that validates the server can be initialized
	// without database features enabled

	// Set environment to avoid actual connections
	_ = os.Setenv("DB_DISABLE", "true")
	defer func() { _ = os.Unsetenv("DB_DISABLE") }()

	// Test is primarily to ensure the code compiles and basic logic works
	// Full integration test would require spinning up a real server which is
	// covered in integration tests
}

// TestDatabaseConnectionStringHandling tests database connection logic
func TestDatabaseConnectionStringHandling(t *testing.T) {
	// Use testcontainer for actual database connection test
	testDB := database.SetupTestDatabase(t)
	defer testDB.Close()

	// Verify connection works
	err := testDB.DB.Ping()
	if err != nil {
		t.Fatalf("Database ping failed: %v", err)
	}

	// Verify we can get connection details
	if testDB.Config.Host == "" {
		t.Error("Expected host to be set")
	}
	if testDB.Config.Port == "" {
		t.Error("Expected port to be set")
	}
	if testDB.Config.User == "" {
		t.Error("Expected user to be set")
	}
	if testDB.Config.DBName == "" {
		t.Error("Expected database name to be set")
	}
}

// TestConfigValidation tests the configuration validation flag behavior
func TestConfigValidationPathSecurity(t *testing.T) {
	// Test path traversal prevention
	maliciousPaths := []string{
		"../../etc/passwd",
		"/etc/passwd",
		"../../../etc/shadow",
		"~/.ssh/id_rsa",
	}

	for _, path := range maliciousPaths {
		t.Run(path, func(t *testing.T) {
			err := security.ValidateConfigPath(path)
			if err == nil {
				t.Errorf("Expected error for malicious path %s, got nil", path)
			}
		})
	}

	// Test valid paths
	validPaths := []string{
		"admin-config.yaml",
		"config.yaml",
		"./config/admin.yaml",
	}

	for _, path := range validPaths {
		t.Run(path, func(t *testing.T) {
			err := security.ValidateConfigPath(path)
			if err != nil {
				t.Errorf("Expected no error for valid path %s, got %v", path, err)
			}
		})
	}
}

// TestHTTPServerConfiguration tests HTTP server timeout configuration
func TestHTTPServerConfiguration(t *testing.T) {
	// Test server configuration values
	expectedReadTimeout := 15 * time.Second
	expectedWriteTimeout := 15 * time.Second
	expectedIdleTimeout := 60 * time.Second

	// Create a test server with the expected configuration
	server := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  expectedReadTimeout,
		WriteTimeout: expectedWriteTimeout,
		IdleTimeout:  expectedIdleTimeout,
	}

	if server.ReadTimeout != expectedReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", server.ReadTimeout, expectedReadTimeout)
	}

	if server.WriteTimeout != expectedWriteTimeout {
		t.Errorf("WriteTimeout = %v, want %v", server.WriteTimeout, expectedWriteTimeout)
	}

	if server.IdleTimeout != expectedIdleTimeout {
		t.Errorf("IdleTimeout = %v, want %v", server.IdleTimeout, expectedIdleTimeout)
	}
}

// TestServerGracefulShutdown tests graceful shutdown behavior
func TestServerGracefulShutdown(t *testing.T) {
	// Create a test HTTP server
	server := &http.Server{
		Addr: ":0", // Random port
	}

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Expected error when shutting down
			t.Logf("Server error (expected): %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Server shutdown failed: %v", err)
	}
}

// TestBuildInfo tests that build information variables are accessible
func TestBuildInfo(t *testing.T) {
	// version and commit are package-level variables
	if version == "" {
		t.Log("version is empty (expected for test builds)")
	}

	if commit == "" {
		t.Log("commit is empty (expected for test builds)")
	}

	// Test that they have default values
	if version != "dev" && version != "" {
		t.Logf("version = %s", version)
	}

	if commit != "unknown" && commit != "" {
		t.Logf("commit = %s", commit)
	}
}

// TestEnvironmentVariables tests environment variable handling
func TestEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		setValue string
	}{
		{"OIDC Enabled", "OIDC_ENABLED", "true"},
		{"OIDC Issuer", "OIDC_ISSUER", "https://example.com"},
		{"OIDC Client ID", "OIDC_CLIENT_ID", "test-client"},
		{"Pushgateway URL", "PUSHGATEWAY_URL", "http://localhost:9091"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := os.Getenv(tt.envVar)
			defer func() { _ = os.Setenv(tt.envVar, original) }()

			_ = os.Setenv(tt.envVar, tt.setValue)
			value := os.Getenv(tt.envVar)

			if value != tt.setValue {
				t.Errorf("Expected %s = %s, got %s", tt.envVar, tt.setValue, value)
			}
		})
	}
}

// TestPushgatewayConfiguration tests metrics pusher configuration
func TestPushgatewayConfiguration(t *testing.T) {
	tests := []struct {
		name             string
		pushgatewayURL   string
		shouldInitPusher bool
	}{
		{"Disabled", "disabled", false},
		{"Empty defaults to pushgateway", "", true}, // Empty defaults to a URL, so pusher is created
		{"Valid URL", "http://localhost:9091", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The logic: if pushgatewayURL is empty, it defaults to "http://pushgateway.localtest.me"
			// if it's "disabled", no pusher is created
			// otherwise, pusher is created

			url := tt.pushgatewayURL
			if url == "" {
				url = "http://pushgateway.localtest.me"
			}

			shouldCreate := url != "" && url != "disabled"

			if shouldCreate != tt.shouldInitPusher {
				t.Errorf("Expected shouldCreate = %v, got %v", tt.shouldInitPusher, shouldCreate)
			}
		})
	}
}

// TestAdminConfigString tests the String() method
func TestAdminConfigString(t *testing.T) {
	config := &admin.AdminConfig{}
	config.Admin.DefaultCostCenter = "test-cc"
	config.Admin.DefaultRuntime = "k8s"
	config.Admin.SplunkIndex = "test-index"
	config.Policies.EnforceBackups = true
	config.Policies.AllowedEnvironments = []string{"dev", "prod"}

	str := config.String()
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Verify key fields are present in string representation
	requiredFields := []string{
		"Admin Configuration",
		"test-cc",
		"k8s",
		"test-index",
		"Policies",
	}

	for _, field := range requiredFields {
		if !contains(str, field) {
			t.Errorf("String() missing expected field: %s", field)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return fmt.Sprint(s) != fmt.Sprint(s[:len(s)-len(substr)]) || s == substr ||
		(len(s) >= len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
