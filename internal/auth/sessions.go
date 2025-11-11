package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"innominatus/internal/users"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Session represents a user session
type Session struct {
	ID        string
	User      *users.User
	CreatedAt time.Time
	ExpiresAt time.Time
	// Impersonation fields
	OriginalUser     *users.User // The admin who started impersonation
	ImpersonatedUser *users.User // The user being impersonated (if any)
	IsImpersonating  bool        // Whether this session is currently impersonating
}

// SessionManager manages user sessions
type SessionManager struct {
	sessions    map[string]*Session
	mutex       sync.RWMutex
	sessionFile string
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	// Create data directory if it doesn't exist
	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		fmt.Printf("Warning: Could not create data directory: %v\n", err)
	}

	sm := &SessionManager{
		sessions:    make(map[string]*Session),
		sessionFile: filepath.Join(dataDir, "sessions.json"),
	}

	// Load existing sessions from disk
	sm.loadSessions()

	// Start cleanup goroutine
	go sm.cleanupExpiredSessions()

	return sm
}

// CreateSession creates a new session for a user
func (sm *SessionManager) CreateSession(user *users.User) (*Session, error) {
	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:        sessionID,
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(3 * time.Hour), // 3 hour expiry
	}

	sm.mutex.Lock()
	sm.sessions[sessionID] = session
	sm.mutex.Unlock()

	// Save sessions to disk
	sm.saveSessions()

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}

	return session, true
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(sessionID string) {
	sm.mutex.Lock()
	delete(sm.sessions, sessionID)
	sm.mutex.Unlock()

	// Save sessions to disk
	sm.saveSessions()
}

// ExtendSession extends a session's expiry time
func (sm *SessionManager) ExtendSession(sessionID string) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.ExpiresAt = time.Now().Add(3 * time.Hour)
		// Save sessions to disk (do this outside the defer to avoid deadlock)
		go sm.saveSessions()
	}
}

// SetSessionCookie sets the session cookie in the response
func (sm *SessionManager) SetSessionCookie(w http.ResponseWriter, session *Session) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Path:     "/",
		Secure:   true, // SECURITY: Always use Secure flag to prevent transmission over HTTP
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the session cookie
func (sm *SessionManager) ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		Secure:   true, // SECURITY: Match Secure flag from SetSessionCookie
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

// GetSessionFromRequest extracts session from request cookie
func (sm *SessionManager) GetSessionFromRequest(r *http.Request) (*Session, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, false
	}

	return sm.GetSession(cookie.Value)
}

// cleanupExpiredSessions periodically removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.mutex.Lock()
		now := time.Now()
		changed := false
		for id, session := range sm.sessions {
			if now.After(session.ExpiresAt) {
				delete(sm.sessions, id)
				changed = true
			}
		}
		sm.mutex.Unlock()

		// Save sessions if any were deleted
		if changed {
			sm.saveSessions()
		}
	}
}

// StartImpersonation starts impersonating another user (admin only)
func (sm *SessionManager) StartImpersonation(sessionID string, targetUser *users.User) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return fmt.Errorf("session expired")
	}

	// Only admins can impersonate
	if !session.User.IsAdmin() {
		return fmt.Errorf("only administrators can impersonate users")
	}

	// Can't impersonate yourself
	if session.User.Username == targetUser.Username {
		return fmt.Errorf("cannot impersonate yourself")
	}

	// Store original user if not already impersonating
	if !session.IsImpersonating {
		session.OriginalUser = session.User
	}

	// Set impersonation
	session.ImpersonatedUser = targetUser
	session.User = targetUser // This is what getUserFromContext will return
	session.IsImpersonating = true

	// Extend session to give more time for impersonation testing
	session.ExpiresAt = time.Now().Add(3 * time.Hour)

	return nil
}

// StopImpersonation stops impersonating and returns to original user
func (sm *SessionManager) StopImpersonation(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	if !session.IsImpersonating {
		return fmt.Errorf("not currently impersonating")
	}

	// Restore original user
	session.User = session.OriginalUser
	session.ImpersonatedUser = nil
	session.IsImpersonating = false

	return nil
}

// GetImpersonationInfo returns impersonation details for a session
func (sm *SessionManager) GetImpersonationInfo(sessionID string) (isImpersonating bool, originalUser *users.User, impersonatedUser *users.User) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return false, nil, nil
	}

	return session.IsImpersonating, session.OriginalUser, session.ImpersonatedUser
}

// generateSessionID creates a cryptographically secure session ID
func generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// loadSessions loads sessions from disk
func (sm *SessionManager) loadSessions() {
	data, err := os.ReadFile(sm.sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, that's okay for first run
			return
		}
		fmt.Printf("Warning: Could not read sessions file: %v\n", err)
		return
	}

	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		fmt.Printf("Warning: Could not parse sessions file: %v\n", err)
		return
	}

	// Load sessions and remove expired ones
	now := time.Now()
	loadedCount := 0
	for id, session := range sessions {
		if now.Before(session.ExpiresAt) {
			sm.sessions[id] = session
			loadedCount++
		}
	}

	if loadedCount > 0 {
		fmt.Printf("✅ Loaded %d active sessions from disk\n", loadedCount)
	}
}

// saveSessions saves sessions to disk
func (sm *SessionManager) saveSessions() {
	sm.mutex.RLock()
	sessions := make(map[string]*Session)
	for k, v := range sm.sessions {
		sessions[k] = v
	}
	sm.mutex.RUnlock()

	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Could not marshal sessions: %v\n", err)
		return
	}

	if err := os.WriteFile(sm.sessionFile, data, 0600); err != nil {
		fmt.Printf("Warning: Could not save sessions to disk: %v\n", err)
	}
}
