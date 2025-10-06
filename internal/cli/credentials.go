package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Credentials stores the user's authentication information
type Credentials struct {
	ServerURL string    `json:"server_url"`
	Username  string    `json:"username"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	KeyName   string    `json:"key_name"`
}

// GetCredentialsPath returns the path to the credentials file
func GetCredentialsPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	credDir := filepath.Join(homeDir, ".idp-o")
	return filepath.Join(credDir, "credentials"), nil
}

// SaveCredentials saves the credentials to the credentials file
func SaveCredentials(creds *Credentials) error {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	credDir := filepath.Dir(credPath)
	if err := os.MkdirAll(credDir, 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	// Marshal credentials to JSON
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Write to file with secure permissions (owner read/write only)
	if err := os.WriteFile(credPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// LoadCredentials loads the credentials from the credentials file
func LoadCredentials() (*Credentials, error) {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return nil, nil // No credentials file, not an error
	}

	// Read the file
	// #nosec G304 - credPath is constructed from os.UserHomeDir() + fixed path, no user input
	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Unmarshal JSON
	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	// Check if credentials have expired
	if time.Now().After(creds.ExpiresAt) {
		// Expired credentials, remove the file
		_ = ClearCredentials()
		return nil, fmt.Errorf("API key has expired on %s", creds.ExpiresAt.Format("2006-01-02"))
	}

	return &creds, nil
}

// ClearCredentials removes the credentials file
func ClearCredentials() error {
	credPath, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to do
	}

	// Remove the file
	if err := os.Remove(credPath); err != nil {
		return fmt.Errorf("failed to remove credentials file: %w", err)
	}

	return nil
}

// HasValidCredentials checks if there are valid credentials stored
func HasValidCredentials() bool {
	creds, err := LoadCredentials()
	return err == nil && creds != nil
}
