package auth

import (
	"innominatus/internal/users"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionManager_CreateSession(t *testing.T) {
	tmpDir := t.TempDir()
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: filepath.Join(tmpDir, "sessions.json"),
	}

	user := &users.User{
		Username: "testuser",
		Team:     "team1",
		Role:     "user",
	}

	session, err := sm.CreateSession(user)
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session == nil {
		t.Fatal("CreateSession() returned nil session")
	}

	if session.ID == "" {
		t.Error("Session ID is empty")
	}

	if session.User.Username != "testuser" {
		t.Errorf("Session user = %v, want testuser", session.User.Username)
	}

	if session.ExpiresAt.Before(time.Now()) {
		t.Error("Session already expired")
	}

	// Verify session was stored
	storedSession, exists := sm.sessions[session.ID]
	if !exists {
		t.Error("Session was not stored in manager")
	}

	if storedSession != session {
		t.Error("Stored session doesn't match created session")
	}
}

func TestSessionManager_GetSession(t *testing.T) {
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: "test-sessions.json",
	}

	user := &users.User{Username: "testuser"}

	// Create a session
	session, _ := sm.CreateSession(user)

	// Retrieve the session
	retrieved, exists := sm.GetSession(session.ID)
	if !exists {
		t.Error("GetSession() returned exists = false")
	}

	if retrieved.ID != session.ID {
		t.Errorf("GetSession() ID = %v, want %v", retrieved.ID, session.ID)
	}

	// Try to get non-existent session
	_, exists = sm.GetSession("non-existent-id")
	if exists {
		t.Error("GetSession() returned exists = true for non-existent session")
	}
}

func TestSessionManager_GetExpiredSession(t *testing.T) {
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: "test-sessions.json",
	}

	// Create an expired session manually
	expiredSession := &Session{
		ID:        "expired-123",
		User:      &users.User{Username: "testuser"},
		CreatedAt: time.Now().Add(-5 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
	}

	sm.sessions[expiredSession.ID] = expiredSession

	// Try to get expired session
	_, exists := sm.GetSession(expiredSession.ID)
	if exists {
		t.Error("GetSession() returned exists = true for expired session")
	}
}

func TestSessionManager_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: filepath.Join(tmpDir, "sessions.json"),
	}

	user := &users.User{Username: "testuser"}
	session, _ := sm.CreateSession(user)

	// Verify session exists
	_, exists := sm.GetSession(session.ID)
	if !exists {
		t.Fatal("Session was not created")
	}

	// Delete session
	sm.DeleteSession(session.ID)

	// Verify session was deleted
	_, exists = sm.GetSession(session.ID)
	if exists {
		t.Error("Session still exists after deletion")
	}
}

func TestSessionManager_ExtendSession(t *testing.T) {
	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: "test-sessions.json",
	}

	user := &users.User{Username: "testuser"}
	session, _ := sm.CreateSession(user)

	originalExpiry := session.ExpiresAt

	// Wait a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Extend session
	sm.ExtendSession(session.ID)

	// Give goroutine time to save (it's async)
	time.Sleep(50 * time.Millisecond)

	// Check that expiry was extended
	extended, _ := sm.GetSession(session.ID)
	if !extended.ExpiresAt.After(originalExpiry) {
		t.Error("Session expiry was not extended")
	}
}

func TestSessionManager_SetSessionCookie(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	session := &Session{
		ID:        "test-session-id",
		User:      &users.User{Username: "testuser"},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	// Create a response recorder
	w := httptest.NewRecorder()

	// Set cookie
	sm.SetSessionCookie(w, session)

	// Check response headers
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "session_id" {
		t.Errorf("Cookie name = %v, want session_id", cookie.Name)
	}

	if cookie.Value != "test-session-id" {
		t.Errorf("Cookie value = %v, want test-session-id", cookie.Value)
	}

	if !cookie.HttpOnly {
		t.Error("Cookie should be HttpOnly")
	}

	if cookie.Path != "/" {
		t.Errorf("Cookie path = %v, want /", cookie.Path)
	}
}

func TestSessionManager_ClearSessionCookie(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	w := httptest.NewRecorder()
	sm.ClearSessionCookie(w)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != "session_id" {
		t.Errorf("Cookie name = %v, want session_id", cookie.Name)
	}

	if cookie.Value != "" {
		t.Errorf("Cookie value should be empty, got %v", cookie.Value)
	}

	if cookie.MaxAge != -1 {
		t.Errorf("Cookie MaxAge = %v, want -1", cookie.MaxAge)
	}
}

func TestSessionManager_GetSessionFromRequest(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	user := &users.User{Username: "testuser"}
	session, _ := sm.CreateSession(user)

	// Create request with session cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: session.ID,
	})

	// Get session from request
	retrieved, exists := sm.GetSessionFromRequest(req)
	if !exists {
		t.Error("GetSessionFromRequest() returned exists = false")
	}

	if retrieved.ID != session.ID {
		t.Errorf("Retrieved session ID = %v, want %v", retrieved.ID, session.ID)
	}

	// Test request without cookie
	reqNoCookie := httptest.NewRequest("GET", "/", nil)
	_, exists = sm.GetSessionFromRequest(reqNoCookie)
	if exists {
		t.Error("GetSessionFromRequest() returned exists = true for request without cookie")
	}
}

func TestSessionManager_StartImpersonation(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create admin user session
	adminUser := &users.User{
		Username: "admin",
		Role:     "admin",
	}
	session, _ := sm.CreateSession(adminUser)

	// Create target user to impersonate
	targetUser := &users.User{
		Username: "targetuser",
		Role:     "user",
	}

	// Start impersonation
	err := sm.StartImpersonation(session.ID, targetUser)
	if err != nil {
		t.Fatalf("StartImpersonation() error = %v", err)
	}

	// Verify impersonation
	retrieved, _ := sm.GetSession(session.ID)
	if !retrieved.IsImpersonating {
		t.Error("Session should be impersonating")
	}

	if retrieved.OriginalUser.Username != "admin" {
		t.Errorf("OriginalUser = %v, want admin", retrieved.OriginalUser.Username)
	}

	if retrieved.ImpersonatedUser.Username != "targetuser" {
		t.Errorf("ImpersonatedUser = %v, want targetuser", retrieved.ImpersonatedUser.Username)
	}

	if retrieved.User.Username != "targetuser" {
		t.Errorf("Current user = %v, want targetuser", retrieved.User.Username)
	}
}

func TestSessionManager_StartImpersonationNonAdmin(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	// Create regular user session
	regularUser := &users.User{
		Username: "regular",
		Role:     "user",
	}
	session, _ := sm.CreateSession(regularUser)

	targetUser := &users.User{
		Username: "target",
		Role:     "user",
	}

	// Try to impersonate (should fail)
	err := sm.StartImpersonation(session.ID, targetUser)
	if err == nil {
		t.Error("StartImpersonation() should fail for non-admin user")
	}
}

func TestSessionManager_StartImpersonationSelf(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	adminUser := &users.User{
		Username: "admin",
		Role:     "admin",
	}
	session, _ := sm.CreateSession(adminUser)

	// Try to impersonate self (should fail)
	err := sm.StartImpersonation(session.ID, adminUser)
	if err == nil {
		t.Error("StartImpersonation() should fail when impersonating self")
	}
}

func TestSessionManager_StopImpersonation(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	adminUser := &users.User{
		Username: "admin",
		Role:     "admin",
	}
	session, _ := sm.CreateSession(adminUser)

	targetUser := &users.User{
		Username: "target",
		Role:     "user",
	}

	// Start impersonation
	_ = sm.StartImpersonation(session.ID, targetUser)

	// Stop impersonation
	err := sm.StopImpersonation(session.ID)
	if err != nil {
		t.Fatalf("StopImpersonation() error = %v", err)
	}

	// Verify impersonation stopped
	retrieved, _ := sm.GetSession(session.ID)
	if retrieved.IsImpersonating {
		t.Error("Session should not be impersonating")
	}

	if retrieved.User.Username != "admin" {
		t.Errorf("User = %v, want admin", retrieved.User.Username)
	}

	if retrieved.ImpersonatedUser != nil {
		t.Error("ImpersonatedUser should be nil")
	}
}

func TestSessionManager_GetImpersonationInfo(t *testing.T) {
	sm := &SessionManager{
		sessions: make(map[string]*Session),
	}

	adminUser := &users.User{
		Username: "admin",
		Role:     "admin",
	}
	session, _ := sm.CreateSession(adminUser)

	// Before impersonation
	isImpersonating, _, _ := sm.GetImpersonationInfo(session.ID)
	if isImpersonating {
		t.Error("Should not be impersonating initially")
	}

	// Start impersonation
	targetUser := &users.User{Username: "target"}
	_ = sm.StartImpersonation(session.ID, targetUser)

	// After impersonation
	isImpersonating, originalUser, impersonatedUser := sm.GetImpersonationInfo(session.ID)
	if !isImpersonating {
		t.Error("Should be impersonating")
	}

	if originalUser.Username != "admin" {
		t.Errorf("OriginalUser = %v, want admin", originalUser.Username)
	}

	if impersonatedUser.Username != "target" {
		t.Errorf("ImpersonatedUser = %v, want target", impersonatedUser.Username)
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err1 := generateSessionID()
	if err1 != nil {
		t.Fatalf("generateSessionID() error = %v", err1)
	}

	if id1 == "" {
		t.Error("Generated session ID is empty")
	}

	if len(id1) != 64 { // 32 bytes hex encoded = 64 characters
		t.Errorf("Session ID length = %d, want 64", len(id1))
	}

	// Generate another ID and verify uniqueness
	id2, err2 := generateSessionID()
	if err2 != nil {
		t.Fatalf("generateSessionID() error = %v", err2)
	}

	if id1 == id2 {
		t.Error("Generated session IDs should be unique")
	}
}

func TestSessionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "sessions.json")

	// Create session manager and add sessions
	sm1 := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: sessionFile,
	}

	user := &users.User{Username: "testuser"}
	session1, _ := sm1.CreateSession(user)

	// Save sessions
	sm1.saveSessions()

	// Verify file exists
	if _, err := os.Stat(sessionFile); os.IsNotExist(err) {
		t.Fatal("Session file was not created")
	}

	// Create new session manager and load sessions
	sm2 := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: sessionFile,
	}
	sm2.loadSessions()

	// Verify session was loaded
	loaded, exists := sm2.GetSession(session1.ID)
	if !exists {
		t.Error("Session was not loaded from file")
	}

	if loaded.User.Username != "testuser" {
		t.Errorf("Loaded session user = %v, want testuser", loaded.User.Username)
	}
}

func TestSessionPersistence_ExpiredSessionsNotLoaded(t *testing.T) {
	tmpDir := t.TempDir()
	sessionFile := filepath.Join(tmpDir, "sessions.json")

	// Create session manager with expired session
	sm1 := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: sessionFile,
	}

	expiredSession := &Session{
		ID:        "expired-123",
		User:      &users.User{Username: "testuser"},
		CreatedAt: time.Now().Add(-5 * time.Hour),
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
	}
	sm1.sessions[expiredSession.ID] = expiredSession
	sm1.saveSessions()

	// Load sessions
	sm2 := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: sessionFile,
	}
	sm2.loadSessions()

	// Verify expired session was not loaded
	_, exists := sm2.GetSession(expiredSession.ID)
	if exists {
		t.Error("Expired session should not be loaded")
	}
}

func TestSession_Struct(t *testing.T) {
	user := &users.User{Username: "testuser"}
	session := &Session{
		ID:        "test-123",
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if session.ID != "test-123" {
		t.Errorf("Session ID = %v, want test-123", session.ID)
	}

	if session.User != user {
		t.Error("Session user doesn't match")
	}

	if session.IsImpersonating {
		t.Error("New session should not be impersonating")
	}

	if session.OriginalUser != nil {
		t.Error("OriginalUser should be nil for new session")
	}
}
