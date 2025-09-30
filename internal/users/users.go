package users

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

type APIKey struct {
	Key         string    `yaml:"key"`
	Name        string    `yaml:"name"`
	CreatedAt   time.Time `yaml:"created_at"`
	LastUsedAt  time.Time `yaml:"last_used_at,omitempty"`
	ExpiresAt   time.Time `yaml:"expires_at"`
}

type User struct {
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
	Team     string   `yaml:"team"`
	Role     string   `yaml:"role"`
	APIKeys  []APIKey `yaml:"api_keys,omitempty"`
}

type UserStore struct {
	Users []User `yaml:"users"`
}

const UsersFile = "users.yaml"

// LoadUsers loads users from the YAML file
func LoadUsers() (*UserStore, error) {
	data, err := os.ReadFile(UsersFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read users file: %w", err)
	}

	var store UserStore
	err = yaml.Unmarshal(data, &store)
	if err != nil {
		return nil, fmt.Errorf("failed to parse users file: %w", err)
	}

	return &store, nil
}

// SaveUsers saves users to the YAML file
func (store *UserStore) SaveUsers() error {
	data, err := yaml.Marshal(store)
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	err = os.WriteFile(UsersFile, data, 0600)
	if err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

// Authenticate checks username and password against stored users
func (store *UserStore) Authenticate(username, password string) (*User, error) {
	for _, user := range store.Users {
		if user.Username == username && user.Password == password {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("invalid credentials")
}

// AddUser adds a new user to the store
func (store *UserStore) AddUser(username, password, team, role string) error {
	// Check if user already exists
	for _, user := range store.Users {
		if user.Username == username {
			return fmt.Errorf("user '%s' already exists", username)
		}
	}

	newUser := User{
		Username: username,
		Password: password,
		Team:     team,
		Role:     role,
	}

	store.Users = append(store.Users, newUser)
	return store.SaveUsers()
}

// DeleteUser removes a user from the store
func (store *UserStore) DeleteUser(username string) error {
	for i, user := range store.Users {
		if user.Username == username {
			// Remove user from slice
			store.Users = append(store.Users[:i], store.Users[i+1:]...)
			return store.SaveUsers()
		}
	}
	return fmt.Errorf("user '%s' not found", username)
}

// GetUser returns a user by username
func (store *UserStore) GetUser(username string) (*User, error) {
	for _, user := range store.Users {
		if user.Username == username {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user '%s' not found", username)
}

// IsAdmin checks if a user has admin role
func (user *User) IsAdmin() bool {
	return user.Role == "admin"
}

// PromptLogin prompts user for credentials and authenticates
func PromptLogin() (*User, error) {
	store, err := LoadUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)

	// Prompt for username
	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	// Prompt for password (hidden input)
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return nil, fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after password input
	password := string(passwordBytes)

	// Authenticate
	user, err := store.Authenticate(username, password)
	if err != nil {
		return nil, err
	}

	fmt.Printf("✓ Authenticated as %s (%s, %s)\n\n", user.Username, user.Team, user.Role)
	return user, nil
}

// GenerateAPIKey creates a new API key for a user
func (store *UserStore) GenerateAPIKey(username, keyName string, expiryDays int) (*APIKey, error) {
	// Validate expiry days
	if expiryDays <= 0 {
		return nil, fmt.Errorf("expiry days must be greater than 0, got %d", expiryDays)
	}

	// Find the user
	userIndex := -1
	for i, user := range store.Users {
		if user.Username == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	// Check if API key name already exists for this user
	for _, apiKey := range store.Users[userIndex].APIKeys {
		if apiKey.Name == keyName {
			return nil, fmt.Errorf("API key with name '%s' already exists for user '%s'", keyName, username)
		}
	}

	// Generate a cryptographically secure API key
	key, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create the API key with mandatory expiry
	apiKey := APIKey{
		Key:       key,
		Name:      keyName,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().AddDate(0, 0, expiryDays),
	}

	// Add to user's API keys
	store.Users[userIndex].APIKeys = append(store.Users[userIndex].APIKeys, apiKey)

	// Save changes
	err = store.SaveUsers()
	if err != nil {
		return nil, fmt.Errorf("failed to save API key: %w", err)
	}

	return &apiKey, nil
}

// AuthenticateWithAPIKey checks if an API key is valid and returns the associated user
func (store *UserStore) AuthenticateWithAPIKey(apiKey string) (*User, error) {
	for i, user := range store.Users {
		for j, key := range user.APIKeys {
			if key.Key == apiKey {
				// Check if key is expired (all keys now have expiry dates)
				if time.Now().After(key.ExpiresAt) {
					return nil, fmt.Errorf("API key expired")
				}

				// Update last used time
				store.Users[i].APIKeys[j].LastUsedAt = time.Now()
				store.SaveUsers() // Save last used time (ignore error to not block authentication)

				return &user, nil
			}
		}
	}
	return nil, fmt.Errorf("invalid API key")
}

// ListAPIKeys lists all API keys for a user
func (store *UserStore) ListAPIKeys(username string) ([]APIKey, error) {
	for _, user := range store.Users {
		if user.Username == username {
			return user.APIKeys, nil
		}
	}
	return nil, fmt.Errorf("user '%s' not found", username)
}

// RevokeAPIKey removes an API key from a user
func (store *UserStore) RevokeAPIKey(username, keyName string) error {
	userIndex := -1
	for i, user := range store.Users {
		if user.Username == username {
			userIndex = i
			break
		}
	}

	if userIndex == -1 {
		return fmt.Errorf("user '%s' not found", username)
	}

	// Find and remove the API key
	keyIndex := -1
	for i, key := range store.Users[userIndex].APIKeys {
		if key.Name == keyName {
			keyIndex = i
			break
		}
	}

	if keyIndex == -1 {
		return fmt.Errorf("API key '%s' not found for user '%s'", keyName, username)
	}

	// Remove the key from slice
	store.Users[userIndex].APIKeys = append(
		store.Users[userIndex].APIKeys[:keyIndex],
		store.Users[userIndex].APIKeys[keyIndex+1:]...,
	)

	return store.SaveUsers()
}

// generateAPIKey creates a cryptographically secure API key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}