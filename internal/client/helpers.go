package client

import "os"

// GetAPIKey returns the API key from environment or credentials file
func GetAPIKey() string {
	// Priority order for API key:
	// 1. Environment variable (highest priority - for CI/CD)
	// 2. Credentials file ($HOME/.innominatus/credentials)

	if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		return apiKey
	}

	// Try to load from credentials file (from cli package)
	// For now, just return empty - the deploy command will handle auth
	return ""
}
