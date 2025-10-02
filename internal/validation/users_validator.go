package validation

import (
	"fmt"
	"innominatus/internal/users"
	"strings"
	"time"
)

// UsersValidator validates users configuration
type UsersValidator struct {
	store *users.UserStore
}

// NewUsersValidator creates a new users validator
func NewUsersValidator(usersFile string) (*UsersValidator, error) {
	// Use default path if not provided
	_ = usersFile // Parameter kept for interface compatibility but not used

	store, err := users.LoadUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to load users config: %w", err)
	}

	return &UsersValidator{store: store}, nil
}

// Validate validates the users configuration
func (v *UsersValidator) Validate() *ValidationResult {
	result := &ValidationResult{
		Valid:     true,
		Errors:    make([]string, 0),
		Warnings:  make([]string, 0),
		Component: "Users Configuration",
	}

	// Check if any users are defined
	if len(v.store.Users) == 0 {
		result.Errors = append(result.Errors, "No users defined - at least one user is required")
		result.Valid = false
		return result
	}

	// Validate each user
	v.validateUsers(result)

	// Check for security requirements
	v.checkSecurityRequirements(result)

	// Validate API keys
	v.validateAPIKeys(result)

	result.Valid = len(result.Errors) == 0
	return result
}

// GetComponent returns the component name
func (v *UsersValidator) GetComponent() string {
	return "Users Configuration"
}

func (v *UsersValidator) validateUsers(result *ValidationResult) {
	usernames := make(map[string]bool)
	adminCount := 0

	for i, user := range v.store.Users {
		userContext := fmt.Sprintf("User %d", i+1)

		// Validate username
		if err := ValidateRequired("username", user.Username); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", userContext, err.Error()))
			continue
		}

		// Check for duplicate usernames
		if usernames[user.Username] {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): duplicate username", userContext, user.Username))
		}
		usernames[user.Username] = true

		// Validate username format
		if err := ValidateRegex("username", user.Username,
			`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
		}

		// Validate password
		if err := ValidateRequired("password", user.Password); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
		} else {
			// Check for weak passwords
			if err := v.validateUserPassword(user.Password); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
			}
		}

		// Validate team
		if err := ValidateRequired("team", user.Team); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
		} else {
			// Validate team format
			if err := ValidateRegex("team", user.Team,
				`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
			}
		}

		// Validate role
		allowedRoles := []string{"admin", "user"}
		if err := ValidateEnum("role", user.Role, allowedRoles); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): %s", userContext, user.Username, err.Error()))
		}

		// Count admin users
		if user.Role == "admin" {
			adminCount++
		}
	}

	// Check admin requirements
	if adminCount == 0 {
		result.Warnings = append(result.Warnings, "No admin users found - consider having at least one admin user")
	}

	if adminCount > 3 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("High number of admin users (%d) - consider using regular users with appropriate team assignments", adminCount))
	}
}

func (v *UsersValidator) validateUserPassword(password string) error {
	// Check for common weak passwords
	weakPasswords := []string{
		"admin", "password", "123456", "admin123", "password123",
		"user", "test", "demo", "changeme", "secret",
	}

	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if lowerPassword == weak {
			return fmt.Errorf("password '%s' is commonly used and weak - consider using a stronger password", password)
		}
	}

	// Check password length
	if len(password) < 8 {
		return fmt.Errorf("password is too short (%d characters) - consider using at least 8 characters", len(password))
	}

	// Check if password is the same as username (this requires context we don't have here)
	// We'll skip this check since we don't have the username in this context

	// Check for simple patterns
	if strings.Contains(lowerPassword, "123") || strings.Contains(lowerPassword, "abc") {
		return fmt.Errorf("password contains simple patterns - consider using a more complex password")
	}

	return nil
}

func (v *UsersValidator) checkSecurityRequirements(result *ValidationResult) {
	// Check for users with default credentials
	defaultUsers := []struct {
		username, password string
	}{
		{"admin", "admin"},
		{"admin", "admin123"},
		{"user", "user"},
		{"test", "test"},
	}

	for _, user := range v.store.Users {
		for _, defaultCred := range defaultUsers {
			if user.Username == defaultCred.username && user.Password == defaultCred.password {
				result.Warnings = append(result.Warnings, fmt.Sprintf("User '%s' has default credentials - change password for security", user.Username))
			}
		}
	}

	// Check team distribution
	teams := make(map[string]int)
	for _, user := range v.store.Users {
		teams[user.Team]++
	}

	if len(teams) == 1 {
		result.Warnings = append(result.Warnings, "All users belong to the same team - consider organizing users into different teams")
	}

	// Check for recommended team names
	recommendedTeams := []string{"platform", "development", "operations", "security"}
	hasRecommendedTeam := false
	for teamName := range teams {
		for _, recommended := range recommendedTeams {
			if strings.EqualFold(teamName, recommended) {
				hasRecommendedTeam = true
				break
			}
		}
	}

	if !hasRecommendedTeam && len(teams) > 1 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Consider using standard team names: %v", recommendedTeams))
	}
}

func (v *UsersValidator) validateAPIKeys(result *ValidationResult) {
	totalAPIKeys := 0
	expiredKeys := 0
	expiringSoonKeys := 0
	now := time.Now()
	soon := now.Add(7 * 24 * time.Hour) // 7 days from now

	for _, user := range v.store.Users {
		if len(user.APIKeys) == 0 {
			continue
		}

		totalAPIKeys += len(user.APIKeys)
		keyNames := make(map[string]bool)

		for j, apiKey := range user.APIKeys {
			keyContext := fmt.Sprintf("User '%s', API key %d", user.Username, j+1)

			// Validate key name
			if err := ValidateRequired("name", apiKey.Name); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", keyContext, err.Error()))
				continue
			}

			// Check for duplicate key names within user
			if keyNames[apiKey.Name] {
				result.Errors = append(result.Errors, fmt.Sprintf("%s (%s): duplicate API key name for user", keyContext, apiKey.Name))
			}
			keyNames[apiKey.Name] = true

			// Validate API key format
			if err := ValidateAPIKeyFormat(apiKey.Key); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): %s", keyContext, apiKey.Name, err.Error()))
			}

			// Check expiration
			if now.After(apiKey.ExpiresAt) {
				expiredKeys++
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): API key has expired on %s",
					keyContext, apiKey.Name, apiKey.ExpiresAt.Format("2006-01-02")))
			} else if soon.After(apiKey.ExpiresAt) {
				expiringSoonKeys++
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): API key expires soon on %s",
					keyContext, apiKey.Name, apiKey.ExpiresAt.Format("2006-01-02")))
			}

			// Check if key has never been used
			if apiKey.LastUsedAt.IsZero() {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): API key has never been used - consider removing if not needed",
					keyContext, apiKey.Name))
			}

			// Check key name format
			if err := ValidateRegex("keyName", apiKey.Name,
				`^[a-z][a-z0-9\-]*[a-z0-9]$`, "lowercase alphanumeric with hyphens"); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("%s (%s): %s", keyContext, apiKey.Name, err.Error()))
			}
		}
	}

	// Summary warnings
	if totalAPIKeys == 0 {
		result.Warnings = append(result.Warnings, "No API keys defined - consider creating API keys for programmatic access")
	}

	if expiredKeys > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d API keys have expired - consider removing or renewing them", expiredKeys))
	}

	if expiringSoonKeys > 0 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("%d API keys expire soon - consider renewing them", expiringSoonKeys))
	}

	if totalAPIKeys > 10 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("High number of API keys (%d) - consider regular cleanup of unused keys", totalAPIKeys))
	}
}
