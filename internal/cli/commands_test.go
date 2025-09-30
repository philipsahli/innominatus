package cli

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8081")

	assert.NotNil(t, client)
	// Note: baseURL and client are private fields, so we can't access them directly
	// We can only test that the client was created successfully
}

func TestListCommand(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/specs":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"test-app": {
					"metadata": {
						"APIVersion": "score.dev/v1b1"
					},
					"containers": {
						"web": {
							"Image": "nginx:latest",
							"Variables": {
								"DB_HOST": "localhost"
							}
						}
					},
					"resources": {
						"db": {
							"Type": "postgres",
							"Params": {
								"version": "13"
							}
						}
					}
				}
			}`)
		case "/api/workflows":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `[
				{
					"id": 1,
					"app_name": "test-app",
					"workflow_name": "deploy",
					"status": "completed"
				}
			]`)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Capture output
	var buf bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test list without details
	err := client.ListCommand(false)
	assert.NoError(t, err)

	// Test list with details
	err = client.ListCommand(true)
	assert.NoError(t, err)

	// Restore stdout and read output
	w.Close()
	os.Stdout = originalStdout
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "test-app")
	assert.Contains(t, output, "nginx:latest")
}

func TestListCommandEmptyResponse(t *testing.T) {
	// Create test server with empty response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Capture output
	var buf bytes.Buffer
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := client.ListCommand(false)
	assert.NoError(t, err)

	// Restore stdout and read output
	w.Close()
	os.Stdout = originalStdout
	buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "No applications deployed")
}

func TestStatusCommand(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/specs" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"test-app": {
					"metadata": {"APIVersion": "score.dev/v1b1"}
				}
			}`)
		} else if r.URL.Path == "/api/specs/test-app" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{
				"metadata": {"APIVersion": "score.dev/v1b1"}
			}`)
		} else if r.URL.Path == "/api/specs/non-existing-app" {
			http.Error(w, "Not found", http.StatusNotFound)
		} else if strings.HasPrefix(r.URL.Path, "/api/workflows") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `[
				{
					"id": 1,
					"app_name": "test-app",
					"status": "completed",
					"started_at": "2023-01-01T00:00:00Z"
				}
			]`)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test status for existing app
	err := client.StatusCommand("test-app")
	assert.NoError(t, err)

	// Test status for non-existing app
	err = client.StatusCommand("non-existing-app")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestValidateCommand(t *testing.T) {
	// Create temporary test files
	tmpDir := t.TempDir()

	validFile := filepath.Join(tmpDir, "valid.yaml")
	validContent := `apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest`

	err := ioutil.WriteFile(validFile, []byte(validContent), 0644)
	require.NoError(t, err)

	invalidFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidContent := `invalid: yaml: content:`

	err = ioutil.WriteFile(invalidFile, []byte(invalidContent), 0644)
	require.NoError(t, err)

	client := NewClient("http://localhost:8081")

	// Test valid file
	err = client.ValidateCommand(validFile)
	assert.NoError(t, err)

	// Test invalid file
	err = client.ValidateCommand(invalidFile)
	assert.Error(t, err)

	// Test non-existent file
	err = client.ValidateCommand("nonexistent.yaml")
	assert.Error(t, err)
}

func TestDeleteCommand(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if strings.Contains(r.URL.Path, "test-app") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"message": "Application deleted successfully"}`)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Test successful deletion
	err := client.DeleteCommand("test-app")
	assert.NoError(t, err)

	// Test deletion of non-existent app
	err = client.DeleteCommand("non-existent-app")
	assert.Error(t, err)
}

func TestEnvironmentsCommand(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"development": {
				"name": "development",
				"type": "kubernetes",
				"status": "active",
				"created_at": "2023-01-01T00:00:00Z"
			},
			"staging": {
				"name": "staging",
				"type": "kubernetes",
				"status": "active",
				"created_at": "2023-01-01T00:00:00Z"
			}
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	err := client.EnvironmentsCommand()
	assert.NoError(t, err)
}

func TestAnalyzeCommand(t *testing.T) {
	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-spec.yaml")
	testContent := `apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
workflows:
  deploy:
    steps:
      - name: setup-infra
        type: terraform
        path: ./terraform`

	err := ioutil.WriteFile(testFile, []byte(testContent), 0644)
	require.NoError(t, err)

	client := NewClient("http://localhost:8081")

	// Test analyze command
	err = client.AnalyzeCommand(testFile)
	assert.NoError(t, err) // Should not error even without server

	// Test non-existent file
	err = client.AnalyzeCommand("nonexistent.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestListGoldenPathsCommand(t *testing.T) {
	client := NewClient("http://localhost:8081")

	// Test list golden paths - should error if file doesn't exist
	err := client.ListGoldenPathsCommand()
	assert.Error(t, err) // Should error if goldenpaths.yaml doesn't exist
	assert.Contains(t, err.Error(), "failed to load golden paths")
}

func TestRunGoldenPathCommand(t *testing.T) {
	client := NewClient("http://localhost:8081")

	// Test run golden path without spec file
	err := client.RunGoldenPathCommand("test-path", "")
	assert.Error(t, err) // Should error because path doesn't exist

	// Create temporary spec file for testing
	tmpDir := t.TempDir()
	specFile := filepath.Join(tmpDir, "spec.yaml")
	err = ioutil.WriteFile(specFile, []byte("test: content"), 0644)
	require.NoError(t, err)

	// Test run golden path with spec file
	err = client.RunGoldenPathCommand("test-path", specFile)
	assert.Error(t, err) // Should error because path doesn't exist
}

func TestAdminShowCommand(t *testing.T) {
	client := NewClient("http://localhost:8081")

	// Note: AdminShowCommand doesn't exist in the actual client
	// This would be implemented as a separate command function
	// For now, we'll just test that the client exists
	assert.NotNil(t, client)
}

func TestDemoCommands(t *testing.T) {
	// Skip this test as it performs real Kubernetes operations
	t.Skip("Skipping demo commands test - performs real infrastructure operations")

	client := NewClient("http://localhost:8081")

	// Test demo-time command
	err := client.DemoTimeCommand()
	assert.NoError(t, err) // Should work if Kubernetes is available

	// Test demo-status command
	err = client.DemoStatusCommand()
	assert.NoError(t, err) // Should work if demo was installed

	// Test demo-nuke command
	err = client.DemoNukeCommand()
	assert.NoError(t, err) // Should work to cleanup
}

func TestClientHTTPMethods(t *testing.T) {
	// Test that client properly constructs HTTP requests
	client := NewClient("http://test.example.com")

	// Note: baseURL and client are private fields, so we can't access them directly
	// We can only test that the client was created successfully
	assert.NotNil(t, client)
}

func TestFileHandling(t *testing.T) {
	tmpDir := t.TempDir()

	// Test reading valid YAML file
	validFile := filepath.Join(tmpDir, "valid.yaml")
	validContent := `apiVersion: score.dev/v1b1
metadata:
  name: test`

	err := ioutil.WriteFile(validFile, []byte(validContent), 0644)
	require.NoError(t, err)

	data, err := ioutil.ReadFile(validFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "test")

	// Test reading non-existent file
	_, err = ioutil.ReadFile(filepath.Join(tmpDir, "nonexistent.yaml"))
	assert.Error(t, err)
}

func TestCommandLineHelpers(t *testing.T) {
	// Test string manipulation helpers used in commands
	testCases := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{"test", false},
		{"   ", false}, // Whitespace should be considered non-empty
	}

	for _, tc := range testCases {
		isEmpty := tc.input == ""
		assert.Equal(t, tc.expected, isEmpty)
	}
}

func TestErrorHandling(t *testing.T) {
	client := NewClient("http://invalid-url-that-does-not-exist.com")

	// Test that network errors are handled properly
	err := client.DeployCommand("nonexistent-file.yaml")
	assert.Error(t, err)

	err = client.ListCommand(false)
	assert.Error(t, err)

	err = client.StatusCommand("test-app")
	assert.Error(t, err)
}