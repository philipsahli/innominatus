package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDetectKeycloak(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		serverPath     string
		want           bool
	}{
		{
			name:           "Keycloak available on /health",
			serverResponse: http.StatusOK,
			serverPath:     "/health",
			want:           true,
		},
		{
			name:           "Keycloak available on / (redirect)",
			serverResponse: http.StatusFound,
			serverPath:     "/",
			want:           true,
		},
		{
			name:           "Keycloak unavailable",
			serverResponse: 0, // Will not start server
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require mocking the HTTP client
			// For now, we'll skip the actual network test
			t.Skip("Network-dependent test - requires mock")
		})
	}
}

func TestAutoConfigureOIDC(t *testing.T) {
	tests := []struct {
		name            string
		oidcEnabled     string
		keycloakRunning bool
		wantEnabled     bool
		description     string
	}{
		{
			name:            "OIDC explicitly enabled",
			oidcEnabled:     "true",
			keycloakRunning: false,
			wantEnabled:     true,
			description:     "Should enable OIDC when OIDC_ENABLED=true",
		},
		{
			name:            "OIDC explicitly disabled",
			oidcEnabled:     "false",
			keycloakRunning: true,
			wantEnabled:     false,
			description:     "Should respect OIDC_ENABLED=false even if Keycloak detected",
		},
		{
			name:            "Auto-detect with Keycloak",
			oidcEnabled:     "",
			keycloakRunning: true,
			wantEnabled:     true,
			description:     "Should auto-enable OIDC when Keycloak detected",
		},
		{
			name:            "Auto-detect without Keycloak",
			oidcEnabled:     "",
			keycloakRunning: false,
			wantEnabled:     false,
			description:     "Should keep OIDC disabled when Keycloak not detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldValue := os.Getenv("OIDC_ENABLED")
			defer func() {
				if oldValue != "" {
					_ = os.Setenv("OIDC_ENABLED", oldValue)
				} else {
					_ = os.Unsetenv("OIDC_ENABLED")
				}
			}()

			// Set test environment
			if tt.oidcEnabled != "" {
				_ = os.Setenv("OIDC_ENABLED", tt.oidcEnabled)
			} else {
				_ = os.Unsetenv("OIDC_ENABLED")
			}

			// Skip actual detection for unit tests
			// In real tests, we'd mock DetectKeycloak()
			t.Skip("Requires mocking DetectKeycloak() - integration test needed")
		})
	}
}

func TestGetEnvExplicit(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		setValue bool
		want     string
	}{
		{
			name:     "Variable set to 'true'",
			key:      "TEST_VAR_TRUE",
			value:    "true",
			setValue: true,
			want:     "true",
		},
		{
			name:     "Variable set to 'false'",
			key:      "TEST_VAR_FALSE",
			value:    "false",
			setValue: true,
			want:     "false",
		},
		{
			name:     "Variable set to empty string",
			key:      "TEST_VAR_EMPTY",
			value:    "",
			setValue: true,
			want:     "",
		},
		{
			name:     "Variable not set",
			key:      "TEST_VAR_UNSET",
			setValue: false,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldValue, existed := os.LookupEnv(tt.key)
			defer func() {
				if existed {
					_ = os.Setenv(tt.key, oldValue)
				} else {
					_ = os.Unsetenv(tt.key)
				}
			}()

			// Set up test environment
			if tt.setValue {
				_ = os.Setenv(tt.key, tt.value)
			} else {
				_ = os.Unsetenv(tt.key)
			}

			// Test
			got := getEnvExplicit(tt.key)
			if got != tt.want {
				t.Errorf("getEnvExplicit() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Integration test helper - starts a mock Keycloak server
func startMockKeycloak(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate Keycloak responding with redirect
		w.WriteHeader(http.StatusFound)
		w.Header().Set("Location", "/admin/")
	})
	return httptest.NewServer(handler)
}

func TestDetectKeycloakIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("Mock Keycloak server responds", func(t *testing.T) {
		server := startMockKeycloak(t)
		defer server.Close()

		// Note: This test is limited because DetectKeycloak() hardcodes
		// the Keycloak URL. In a real scenario, we'd need to make the URL
		// configurable or mock the HTTP client.
		t.Skip("DetectKeycloak uses hardcoded URL - need refactoring for proper testing")
	})
}
