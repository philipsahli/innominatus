package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"innominatus/internal/users"
	"net/http"
	"os"
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
	redirectURL := fmt.Sprintf("/auth/oidc/callback?token=%s", session.ID)
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
