package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAPIClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantError      bool
		wantContains   string
	}{
		{
			name:           "successful GET",
			serverResponse: `{"status":"ok"}`,
			serverStatus:   http.StatusOK,
			wantError:      false,
			wantContains:   "ok",
		},
		{
			name:           "404 error",
			serverResponse: `{"error":"not found"}`,
			serverStatus:   http.StatusNotFound,
			wantError:      true,
		},
		{
			name:           "500 error",
			serverResponse: `{"error":"internal server error"}`,
			serverStatus:   http.StatusInternalServerError,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify auth header
				if auth := r.Header.Get("Authorization"); !strings.HasPrefix(auth, "Bearer ") {
					t.Error("Expected Authorization header with Bearer token")
				}

				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			result, err := client.Get(context.Background(), "/api/test")

			if (err != nil) != tt.wantError {
				t.Errorf("Get() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && !strings.Contains(result, tt.wantContains) {
				t.Errorf("Get() = %v, want to contain %v", result, tt.wantContains)
			}
		})
	}
}

func TestAPIClient_Post(t *testing.T) {
	tests := []struct {
		name         string
		requestBody  interface{}
		serverStatus int
		wantError    bool
	}{
		{
			name:         "successful POST with map",
			requestBody:  map[string]string{"key": "value"},
			serverStatus: http.StatusOK,
			wantError:    false,
		},
		{
			name:         "successful POST with struct",
			requestBody:  struct{ Name string }{Name: "test"},
			serverStatus: http.StatusCreated,
			wantError:    false,
		},
		{
			name:         "server error",
			requestBody:  map[string]string{"key": "value"},
			serverStatus: http.StatusBadRequest,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != "POST" {
					t.Errorf("Expected POST, got %s", r.Method)
				}

				// Verify content type
				if ct := r.Header.Get("Content-Type"); ct != "application/json" {
					t.Errorf("Expected Content-Type application/json, got %s", ct)
				}

				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(`{"result":"success"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			_, err := client.Post(context.Background(), "/api/test", tt.requestBody)

			if (err != nil) != tt.wantError {
				t.Errorf("Post() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestAPIClient_PostYAML(t *testing.T) {
	yamlContent := `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/yaml" {
			t.Errorf("Expected Content-Type application/yaml, got %s", ct)
		}

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"spec submitted"}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "test-token")
	result, err := client.PostYAML(context.Background(), "/api/specs", yamlContent)

	if err != nil {
		t.Errorf("PostYAML() error = %v", err)
	}

	if !strings.Contains(result, "spec submitted") {
		t.Errorf("PostYAML() = %v, want to contain 'spec submitted'", result)
	}
}

func TestAPIClient_AuthenticationHeader(t *testing.T) {
	expectedToken := "secret-test-token"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expectedAuth := "Bearer " + expectedToken

		if auth != expectedAuth {
			t.Errorf("Expected Authorization '%s', got '%s'", expectedAuth, auth)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticated":true}`))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, expectedToken)
	_, err := client.Get(context.Background(), "/api/test")

	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
}

func TestAPIClient_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second) // Longer than 30s client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/api/test")

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") &&
		!strings.Contains(err.Error(), "request failed") {
		t.Errorf("Expected context deadline error, got %v", err)
	}
}

func TestAPIClient_InvalidJSON(t *testing.T) {
	// Test marshaling error for Post
	client := NewAPIClient("http://localhost", "test-token")

	// channel cannot be marshaled to JSON
	invalidBody := make(chan int)

	_, err := client.Post(context.Background(), "/api/test", invalidBody)

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}

	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("Expected marshal error, got %v", err)
	}
}

func TestAPIClient_NetworkError(t *testing.T) {
	// Use invalid URL to trigger network error
	client := NewAPIClient("http://invalid-host-that-does-not-exist-12345", "test-token")

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := client.Get(ctx, "/api/test")

	if err == nil {
		t.Error("Expected network error, got nil")
	}
}
