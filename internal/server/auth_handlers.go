package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"innominatus/internal/auth"
	"innominatus/internal/users"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HandleLogin handles the login page and authentication
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.showLoginPage(w, r)
	case "POST":
		s.processLogin(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleLogout handles user logout
func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Get session from request
	session, exists := s.sessionManager.GetSessionFromRequest(r)
	if exists {
		s.sessionManager.DeleteSession(session.ID)
	}

	// Clear session cookie
	s.sessionManager.ClearSessionCookie(w)

	// Redirect to login page
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

// showLoginPage redirects to React app since we use SPA
func (s *Server) showLoginPage(w http.ResponseWriter, r *http.Request) {
	// Since we're using React SPA for all UI, redirect any direct access
	// to /auth/login back to the root where React will handle routing
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// processLogin handles login form submission
func (s *Server) processLogin(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)

	// Check rate limiting
	if s.isRateLimited(clientIP) {
		http.Redirect(w, r, "/auth/login?error=Too+many+login+attempts.+Please+wait+15+minutes.", http.StatusSeeOther)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		s.recordLoginAttempt(clientIP)
		http.Redirect(w, r, "/auth/login?error=Username+and+password+are+required", http.StatusSeeOther)
		return
	}

	// Load users and authenticate
	store, err := users.LoadUsers()
	if err != nil {
		http.Redirect(w, r, "/auth/login?error=System+error%3A+unable+to+load+user+data", http.StatusSeeOther)
		return
	}

	user, err := store.Authenticate(username, password)
	if err != nil {
		s.recordLoginAttempt(clientIP)
		http.Redirect(w, r, "/auth/login?error=Invalid+username+or+password", http.StatusSeeOther)
		return
	}

	// Clear login attempts on successful authentication
	s.clearLoginAttempts(clientIP)

	// Create session
	session, err := s.sessionManager.CreateSession(user)
	if err != nil {
		http.Redirect(w, r, "/auth/login?error=Unable+to+create+session", http.StatusSeeOther)
		return
	}

	// Set session cookie
	s.sessionManager.SetSessionCookie(w, session)

	// Redirect to dashboard
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// HandleUserInfo returns current user information (API endpoint)
func (s *Server) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get user from session (middleware should have set this)
	user := s.getUserFromContext(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get session to check impersonation status
	session, exists := s.sessionManager.GetSessionFromRequest(r)
	if !exists {
		http.Error(w, "Session not found", http.StatusUnauthorized)
		return
	}

	// Load fresh user data from users file to get API keys
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load user data", http.StatusInternalServerError)
		return
	}

	// Get complete user data including API keys
	fullUser, err := store.GetUser(user.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	userInfo := map[string]interface{}{
		"username": fullUser.Username,
		"team":     fullUser.Team,
		"role":     fullUser.Role,
		"is_admin": fullUser.IsAdmin(),
		"api_keys": fullUser.APIKeys,
	}

	// Add impersonation information
	isImpersonating, originalUser, impersonatedUser := s.sessionManager.GetImpersonationInfo(session.ID)
	userInfo["is_impersonating"] = isImpersonating

	if isImpersonating {
		userInfo["original_user"] = map[string]interface{}{
			"username": originalUser.Username,
			"team":     originalUser.Team,
			"role":     originalUser.Role,
			"is_admin": originalUser.IsAdmin(),
		}
		userInfo["impersonated_user"] = map[string]interface{}{
			"username": impersonatedUser.Username,
			"team":     impersonatedUser.Team,
			"role":     impersonatedUser.Role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleAPILogin handles API authentication for CLI clients
func (s *Server) HandleAPILogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientIP := getClientIP(r)

	// Check rate limiting
	if s.isRateLimited(clientIP) {
		http.Error(w, "Too many login attempts. Please wait 15 minutes.", http.StatusTooManyRequests)
		return
	}

	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if loginReq.Username == "" || loginReq.Password == "" {
		s.recordLoginAttempt(clientIP)
		http.Error(w, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Load users and authenticate
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "System error: unable to load user data", http.StatusInternalServerError)
		return
	}

	user, err := store.Authenticate(loginReq.Username, loginReq.Password)
	if err != nil {
		s.recordLoginAttempt(clientIP)
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Clear login attempts on successful authentication
	s.clearLoginAttempts(clientIP)

	// Create session
	session, err := s.sessionManager.CreateSession(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create session: %v\n", err)
		http.Error(w, fmt.Sprintf("Unable to create session: %v", err), http.StatusInternalServerError)
		return
	}

	// Return session token
	response := map[string]interface{}{
		"token":    session.ID,
		"username": user.Username,
		"team":     user.Team,
		"role":     user.Role,
		"expires":  session.ExpiresAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleImpersonate handles user impersonation requests (admin only)
func (s *Server) HandleImpersonate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.startImpersonation(w, r)
	case "DELETE":
		s.stopImpersonation(w, r)
	case "GET":
		s.getImpersonationStatus(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleListUsers returns a list of users for impersonation (admin only)
func (s *Server) HandleListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load users
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	// Return user list (without sensitive data)
	userList := make([]map[string]interface{}, 0)
	for _, user := range store.Users {
		userInfo := map[string]interface{}{
			"username": user.Username,
			"team":     user.Team,
			"role":     user.Role,
		}
		userList = append(userList, userInfo)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userList); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleUserManagement handles CRUD operations for users
func (s *Server) HandleUserManagement(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.handleCreateUser(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleUserDetail handles operations on a specific user
func (s *Server) HandleUserDetail(w http.ResponseWriter, r *http.Request) {
	// Extract username from path /api/admin/users/{username}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	username := pathParts[4]

	switch r.Method {
	case "GET":
		s.handleGetUser(w, r, username)
	case "PUT":
		s.handleUpdateUser(w, r, username)
	case "DELETE":
		s.handleDeleteUser(w, r, username)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var request struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Team     string `json:"team"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if request.Username == "" || request.Password == "" || request.Team == "" {
		http.Error(w, "Username, password, and team are required", http.StatusBadRequest)
		return
	}

	// Default role to "user" if not specified
	if request.Role == "" {
		request.Role = "user"
	}

	// Validate role
	if request.Role != "user" && request.Role != "admin" {
		http.Error(w, "Role must be 'user' or 'admin'", http.StatusBadRequest)
		return
	}

	// Load existing users
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	// AddUser will check if user already exists and hash password
	if err := store.AddUser(request.Username, request.Password, request.Team, request.Role); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, "User already exists", http.StatusConflict)
		} else {
			http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]interface{}{
		"message":  "User created successfully",
		"username": request.Username,
		"team":     request.Team,
		"role":     request.Role,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, username string) {
	// Load users
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	// Find user
	user, err := store.GetUser(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return user info (without password)
	userInfo := map[string]interface{}{
		"username": user.Username,
		"team":     user.Team,
		"role":     user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userInfo); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, username string) {
	// Parse request body
	var request struct {
		Password *string `json:"password,omitempty"`
		Team     *string `json:"team,omitempty"`
		Role     *string `json:"role,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate role if provided
	if request.Role != nil && *request.Role != "user" && *request.Role != "admin" {
		http.Error(w, "Role must be 'user' or 'admin'", http.StatusBadRequest)
		return
	}

	// Load users
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	// Find and update user
	found := false
	for i, user := range store.Users {
		if user.Username == username {
			found = true
			// Update user fields
			if request.Password != nil {
				// SECURITY: Hash password with bcrypt before storage
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*request.Password), bcrypt.DefaultCost)
				if err != nil {
					http.Error(w, "Failed to hash password", http.StatusInternalServerError)
					return
				}
				store.Users[i].Password = string(hashedPassword)
			}
			if request.Team != nil {
				store.Users[i].Team = *request.Team
			}
			if request.Role != nil {
				store.Users[i].Role = *request.Role
			}
			break
		}
	}

	if !found {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Save updated store
	if err := store.SaveUsers(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update user: %v", err), http.StatusInternalServerError)
		return
	}

	// Get updated user for response
	updatedUser, _ := store.GetUser(username)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":  "User updated successfully",
		"username": username,
		"team":     updatedUser.Team,
		"role":     updatedUser.Role,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, username string) {
	// Load users
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	// Delete user (DeleteUser will check if exists and return error if not found)
	if err := store.DeleteUser(username); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("Failed to delete user: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":  "User deleted successfully",
		"username": username,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// HandleAdminUserAPIKeys handles admin operations on user API keys
func (s *Server) HandleAdminUserAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Extract username from path: /api/admin/users/{username}/api-keys
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 6 {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	username := pathParts[4]

	switch r.Method {
	case "GET":
		s.handleAdminGetAPIKeys(w, r, username)
	case "POST":
		s.handleAdminGenerateAPIKey(w, r, username)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleAdminUserAPIKeyDetail handles admin operations on specific API keys
func (s *Server) HandleAdminUserAPIKeyDetail(w http.ResponseWriter, r *http.Request) {
	// Extract username and key name from path: /api/admin/users/{username}/api-keys/{keyname}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 7 {
		http.Error(w, "Username and key name required", http.StatusBadRequest)
		return
	}
	username := pathParts[4]
	keyName := pathParts[6]

	switch r.Method {
	case "DELETE":
		s.handleAdminRevokeAPIKey(w, r, username, keyName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminGetAPIKeys(w http.ResponseWriter, r *http.Request, username string) {
	// Verify target user exists
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	targetUser, err := store.GetUser(username)
	isOIDCUser := err != nil

	var keys []users.APIKey

	if isOIDCUser && s.db != nil {
		// Get API keys from database for OIDC user
		dbKeys, err := s.db.GetAPIKeys(username)
		if err != nil {
			http.Error(w, "Failed to retrieve API keys", http.StatusInternalServerError)
			return
		}

		// Convert database records to users.APIKey format
		for _, dbKey := range dbKeys {
			lastUsed := time.Time{}
			if dbKey.LastUsedAt != nil {
				lastUsed = *dbKey.LastUsedAt
			}
			keys = append(keys, users.APIKey{
				Key:        dbKey.KeyHash, // Will be masked anyway
				Name:       dbKey.KeyName,
				CreatedAt:  dbKey.CreatedAt,
				LastUsedAt: lastUsed,
				ExpiresAt:  dbKey.ExpiresAt,
			})
		}
	} else if targetUser != nil {
		// Get API keys for local user
		keys = targetUser.APIKeys
	} else {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Mask keys before sending
	maskedKeys := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		maskedKey := map[string]interface{}{
			"name":       key.Name,
			"key":        maskAPIKey(key.Key),
			"created_at": key.CreatedAt.Format(time.RFC3339),
			"expires_at": key.ExpiresAt.Format(time.RFC3339),
		}
		if !key.LastUsedAt.IsZero() {
			maskedKey["last_used_at"] = key.LastUsedAt.Format(time.RFC3339)
		}
		maskedKeys = append(maskedKeys, maskedKey)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"username": username,
		"api_keys": maskedKeys,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) handleAdminGenerateAPIKey(w http.ResponseWriter, r *http.Request, username string) {
	var req struct {
		Name       string `json:"name"`
		ExpiryDays int    `json:"expiry_days"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "API key name is required", http.StatusBadRequest)
		return
	}

	if req.ExpiryDays <= 0 {
		req.ExpiryDays = 90 // Default to 90 days
	}

	// Check if user exists
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	_, err = store.GetUser(username)
	isOIDCUser := err != nil

	if isOIDCUser && s.db != nil {
		// Generate API key for OIDC user (store in database)
		apiKey, err := s.generateDatabaseAPIKey(username, req.Name, req.ExpiryDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return the full key only on creation
		response := map[string]interface{}{
			"username":   username,
			"key":        apiKey.Key,
			"name":       apiKey.Name,
			"created_at": apiKey.CreatedAt.Format(time.RFC3339),
			"expires_at": apiKey.ExpiresAt.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode API key", http.StatusInternalServerError)
		}
	} else if err == nil {
		// Generate API key for local user (store in users.yaml)
		apiKey, err := store.GenerateAPIKey(username, req.Name, req.ExpiryDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return the full key only on creation
		response := map[string]interface{}{
			"username":   username,
			"key":        apiKey.Key,
			"name":       apiKey.Name,
			"created_at": apiKey.CreatedAt.Format(time.RFC3339),
			"expires_at": apiKey.ExpiresAt.Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode API key", http.StatusInternalServerError)
		}
	} else {
		http.Error(w, "User not found", http.StatusNotFound)
	}
}

func (s *Server) handleAdminRevokeAPIKey(w http.ResponseWriter, r *http.Request, username, keyName string) {
	// Check if user exists
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Failed to load users", http.StatusInternalServerError)
		return
	}

	_, err = store.GetUser(username)
	isOIDCUser := err != nil

	if isOIDCUser && s.db != nil {
		// Revoke database API key for OIDC user
		if err := s.db.DeleteAPIKey(username, keyName); err != nil {
			http.Error(w, fmt.Sprintf("Failed to revoke API key: %v", err), http.StatusInternalServerError)
			return
		}
	} else if err == nil {
		// Revoke file-based API key for local user
		if err := store.RevokeAPIKey(username, keyName); err != nil {
			http.Error(w, fmt.Sprintf("Failed to revoke API key: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"message":  "API key revoked successfully",
		"username": username,
		"key_name": keyName,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// maskAPIKey masks an API key showing only first 8 and last 4 characters
func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "****"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func (s *Server) startImpersonation(w http.ResponseWriter, r *http.Request) {
	// Get session from request
	session, exists := s.sessionManager.GetSessionFromRequest(r)
	if !exists {
		http.Error(w, "Session not found", http.StatusUnauthorized)
		return
	}

	// Parse request body
	var request struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Load users to find target user
	store, err := users.LoadUsers()
	if err != nil {
		http.Error(w, "Unable to load users", http.StatusInternalServerError)
		return
	}

	targetUser, err := store.GetUser(request.Username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Start impersonation
	err = s.sessionManager.StartImpersonation(session.ID, targetUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success":            true,
		"message":            "Impersonation started",
		"impersonating":      targetUser.Username,
		"impersonating_team": targetUser.Team,
		"impersonating_role": targetUser.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) stopImpersonation(w http.ResponseWriter, r *http.Request) {
	// Get session from request
	session, exists := s.sessionManager.GetSessionFromRequest(r)
	if !exists {
		http.Error(w, "Session not found", http.StatusUnauthorized)
		return
	}

	// Stop impersonation
	err := s.sessionManager.StopImpersonation(session.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": "Impersonation stopped",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

func (s *Server) getImpersonationStatus(w http.ResponseWriter, r *http.Request) {
	// Get session from request
	session, exists := s.sessionManager.GetSessionFromRequest(r)
	if !exists {
		http.Error(w, "Session not found", http.StatusUnauthorized)
		return
	}

	// Get impersonation info
	isImpersonating, originalUser, impersonatedUser := s.sessionManager.GetImpersonationInfo(session.ID)

	response := map[string]interface{}{
		"is_impersonating": isImpersonating,
	}

	if isImpersonating {
		response["original_user"] = map[string]interface{}{
			"username": originalUser.Username,
			"team":     originalUser.Team,
			"role":     originalUser.Role,
		}
		response["impersonated_user"] = map[string]interface{}{
			"username": impersonatedUser.Username,
			"team":     impersonatedUser.Team,
			"role":     impersonatedUser.Role,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode response: %v\n", err)
	}
}

// getUserFromContext retrieves user from request context
// This will be set by the authentication middleware
func (s *Server) getUserFromContext(r *http.Request) *users.User {
	if user, ok := r.Context().Value(contextKeyUser).(*users.User); ok {
		return user
	}
	return nil
}

// HandleOIDCLogin redirects to Keycloak for OIDC authentication
func (s *Server) HandleOIDCLogin(w http.ResponseWriter, r *http.Request) {
	if s.oidcAuthenticator == nil || !s.oidcAuthenticator.IsEnabled() {
		http.Error(w, "OIDC authentication not enabled", http.StatusNotFound)
		return
	}

	// Generate random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	// Store state in cookie for verification
	http.SetCookie(w, &http.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	// Redirect to Keycloak authorization URL
	authURL := s.oidcAuthenticator.AuthCodeURL(state)
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleOIDCCallback handles the OAuth2 callback from Keycloak
func (s *Server) HandleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	if s.oidcAuthenticator == nil || !s.oidcAuthenticator.IsEnabled() {
		http.Error(w, "OIDC authentication not enabled", http.StatusNotFound)
		return
	}

	// Verify state (CSRF protection)
	stateCookie, err := r.Cookie("oidc_state")
	if err != nil {
		http.Redirect(w, r, "/?error=missing_state", http.StatusSeeOther)
		return
	}

	queryState := r.URL.Query().Get("state")
	if stateCookie.Value != queryState {
		http.Redirect(w, r, "/?error=invalid_state", http.StatusSeeOther)
		return
	}

	// Clear state cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "oidc_state",
		MaxAge: -1,
		Path:   "/",
	})

	// Check for error from provider
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		errDesc := r.URL.Query().Get("error_description")
		fmt.Fprintf(os.Stderr, "OIDC error: %s - %s\n", errParam, errDesc)
		http.Redirect(w, r, "/?error=oidc_auth_failed", http.StatusSeeOther)
		return
	}

	// Exchange authorization code for token
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/?error=missing_code", http.StatusSeeOther)
		return
	}

	oauth2Token, err := s.oidcAuthenticator.Exchange(r.Context(), code)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to exchange token: %v\n", err)
		http.Redirect(w, r, "/?error=token_exchange_failed", http.StatusSeeOther)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		fmt.Fprintf(os.Stderr, "No id_token in oauth2 token\n")
		http.Redirect(w, r, "/?error=missing_id_token", http.StatusSeeOther)
		return
	}

	userInfo, err := s.oidcAuthenticator.VerifyIDToken(r.Context(), rawIDToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to verify ID token: %v\n", err)
		http.Redirect(w, r, "/?error=token_verification_failed", http.StatusSeeOther)
		return
	}

	// Create user object for session
	// Use preferred_username or email as username
	username := userInfo.PreferredUsername
	if username == "" {
		username = userInfo.Email
	}

	user := &users.User{
		Username: username,
		Team:     "oidc-users",
		Role:     determineRole(userInfo.Roles),
	}

	// Create session
	session, err := s.sessionManager.CreateSession(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		http.Redirect(w, r, "/?error=session_creation_failed", http.StatusSeeOther)
		return
	}

	// Set session cookie
	s.sessionManager.SetSessionCookie(w, session)

	fmt.Printf("OIDC login successful for user: %s (role: %s)\n", username, user.Role)

	// Redirect to callback page with session ID so frontend can store it
	// IMPORTANT: Trailing slash prevents 301 redirect that strips query params
	redirectURL := fmt.Sprintf("/auth/oidc/callback/?token=%s", session.ID)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// generateRandomState generates a random state for CSRF protection
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// determineRole determines user role from Keycloak roles
func determineRole(roles []string) string {
	for _, role := range roles {
		if role == "admin" {
			return "admin"
		}
	}
	return "user"
}

// HandleOIDCConfig returns OIDC configuration for CLI authentication
func (s *Server) HandleOIDCConfig(w http.ResponseWriter, r *http.Request) {
	if s.oidcAuthenticator == nil || !s.oidcAuthenticator.IsEnabled() {
		http.Error(w, "OIDC authentication not enabled", http.StatusNotFound)
		return
	}

	// Get authorization URL without state (CLI will add its own state/PKCE params)
	authURL := s.oidcAuthenticator.AuthCodeURL("")
	// Remove the state parameter from URL since CLI will add its own
	if idx := strings.Index(authURL, "&state="); idx != -1 {
		authURL = authURL[:idx]
	}

	config := map[string]interface{}{
		"enabled":   true,
		"auth_url":  authURL,
		"client_id": getClientID(s.oidcAuthenticator),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// HandleOIDCTokenExchange handles PKCE token exchange from CLI
func (s *Server) HandleOIDCTokenExchange(w http.ResponseWriter, r *http.Request) {
	if s.oidcAuthenticator == nil || !s.oidcAuthenticator.IsEnabled() {
		http.Error(w, "OIDC authentication not enabled", http.StatusNotFound)
		return
	}

	var req struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
		RedirectURI  string `json:"redirect_uri"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" || req.CodeVerifier == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// Exchange authorization code for token with PKCE verifier
	ctx := r.Context()
	oauth2Token, err := s.oidcAuthenticator.ExchangeWithPKCE(ctx, req.Code, req.CodeVerifier, req.RedirectURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to exchange token: %v\n", err)
		http.Error(w, "Token exchange failed", http.StatusUnauthorized)
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "No ID token in response", http.StatusInternalServerError)
		return
	}

	userInfo, err := s.oidcAuthenticator.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to verify ID token: %v\n", err)
		http.Error(w, "Token verification failed", http.StatusUnauthorized)
		return
	}

	// Use preferred_username or email as username
	username := userInfo.PreferredUsername
	if username == "" {
		username = userInfo.Email
	}

	// Create temporary session for API key generation
	user := &users.User{
		Username: username,
		Team:     "oidc-users",
		Role:     determineRole(userInfo.Roles),
	}

	session, err := s.sessionManager.CreateSession(user)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	// Return access token (session ID) and username
	response := map[string]interface{}{
		"access_token": session.ID,
		"token_type":   "Bearer",
		"username":     username,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getClientID extracts client ID from OIDC authenticator (helper function)
func getClientID(authenticator *auth.OIDCAuthenticator) string {
	// This is a bit of a hack since we don't expose the oauth2Config
	// but we can extract it from the AuthCodeURL
	authURL := authenticator.AuthCodeURL("dummy")
	if idx := strings.Index(authURL, "client_id="); idx != -1 {
		start := idx + len("client_id=")
		end := strings.Index(authURL[start:], "&")
		if end == -1 {
			return authURL[start:]
		}
		return authURL[start : start+end]
	}
	return ""
}
