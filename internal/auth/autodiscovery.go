package auth

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// DetectKeycloak checks if Keycloak is available at the demo-time URL
// This is used to auto-enable OIDC when running in a demo-time environment
func DetectKeycloak() bool {
	// Check Keycloak health endpoint with short timeout
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects - we just want to know if the service responds
			return http.ErrUseLastResponse
		},
	}

	// Try to reach Keycloak at demo-time URL
	resp, err := client.Get("http://keycloak.localtest.me/health")
	if err != nil {
		// Health endpoint might not exist, try root path
		resp, err = client.Get("http://keycloak.localtest.me/")
		if err != nil {
			return false
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Keycloak is reachable if we get any HTTP response (200, 302, 404, etc.)
	// The important thing is that the service is running
	return resp.StatusCode > 0
}

// AutoConfigureOIDC returns an OIDCConfig with auto-detection for demo-time
// If OIDC_ENABLED is not explicitly set and Keycloak is detected, it will be auto-enabled
func AutoConfigureOIDC() OIDCConfig {
	// Load base configuration from environment (without auto-detection to avoid recursion)
	config := LoadOIDCConfigManual()

	// Check if user explicitly disabled OIDC
	if explicitlyDisabled := getEnvExplicit("OIDC_ENABLED"); explicitlyDisabled == "false" {
		fmt.Println("ℹ️  OIDC explicitly disabled by user (OIDC_ENABLED=false)")
		return config
	}

	// If OIDC is already enabled explicitly, use that configuration
	if config.Enabled {
		fmt.Println("✅ OIDC authentication enabled (configured via environment variables)")
		return config
	}

	// Auto-detect demo-time environment
	fmt.Println("🔍 Checking for demo-time environment...")
	if DetectKeycloak() {
		fmt.Println("✅ Keycloak detected at keycloak.localtest.me")
		fmt.Println("✅ Auto-enabling OIDC for demo-time environment")

		// Enable OIDC with demo-time defaults
		config.Enabled = true

		// The defaults in LoadOIDCConfig() already match demo-time:
		// - IssuerURL: http://keycloak.localtest.me/realms/demo-realm
		// - ClientID: innominatus
		// - ClientSecret: innominatus-client-secret

		return config
	}

	fmt.Println("ℹ️  Demo-time environment not detected - OIDC disabled")
	fmt.Println("   To enable OIDC manually, set OIDC_ENABLED=true")
	return config
}

// getEnvExplicit checks if an environment variable is explicitly set
// Returns the value if set, empty string if not set
func getEnvExplicit(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return ""
	}
	return value
}
