package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/users"
	"net/http"
	"time"
)

// DBSessionManager manages user sessions using PostgreSQL database
type DBSessionManager struct {
	db *database.Database
}

// NewDBSessionManager creates a new database-backed session manager
func NewDBSessionManager(db *database.Database) *DBSessionManager {
	if db == nil {
		fmt.Printf("WARNING: NewDBSessionManager called with nil database!\n")
	} else {
		// Test the database connection
		var currentDB string
		err := db.DB().QueryRow("SELECT current_database()").Scan(&currentDB)
		if err != nil {
			fmt.Printf("WARNING: NewDBSessionManager - failed to query database: %v\n", err)
		} else {
			fmt.Printf("DEBUG: NewDBSessionManager - database pointer %p connected to: %s\n", db, currentDB)
		}
	}
	sm := &DBSessionManager{
		db: db,
	}

	// Start cleanup goroutine for expired sessions
	go sm.cleanupExpiredSessions()

	return sm
}

// CreateSession creates a new session for a user
func (sm *DBSessionManager) CreateSession(user *users.User) (*Session, error) {
	if sm.db == nil {
		return nil, fmt.Errorf("database is nil in DBSessionManager")
	}
	sessionID, err := generateDBSessionID()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(3 * time.Hour) // 3 hour expiry

	err = sm.db.CreateSession(sessionID, user, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session in database: %w", err)
	}

	session := &Session{
		ID:        sessionID,
		User:      user,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	return session, nil
}

// GetSession retrieves a session by ID
func (sm *DBSessionManager) GetSession(sessionID string) (*Session, bool) {
	sessionData, err := sm.db.GetSession(sessionID)
	if err != nil {
		return nil, false
	}

	session := &Session{
		ID:               sessionData.SessionID,
		User:             sessionData.User,
		CreatedAt:        sessionData.CreatedAt,
		ExpiresAt:        sessionData.ExpiresAt,
		OriginalUser:     sessionData.OriginalUser,
		ImpersonatedUser: sessionData.ImpersonatedUser,
		IsImpersonating:  sessionData.IsImpersonating,
	}

	return session, true
}

// DeleteSession removes a session
func (sm *DBSessionManager) DeleteSession(sessionID string) {
	_ = sm.db.DeleteSession(sessionID)
	// Ignore error - session might already be deleted
}

// ExtendSession extends a session's expiry time
func (sm *DBSessionManager) ExtendSession(sessionID string) {
	// Get current session data
	sessionData, err := sm.db.GetSession(sessionID)
	if err != nil {
		// Log warning if session retrieval fails (might be expired or DB issue)
		fmt.Printf("Warning: Failed to extend session %s: %v\n", sessionID[:8]+"...", err)
		return // Session doesn't exist or expired
	}

	// Prepare updated user data
	userData := map[string]interface{}{
		"user":              sessionData.User,
		"is_impersonating":  sessionData.IsImpersonating,
		"original_user":     sessionData.OriginalUser,
		"impersonated_user": sessionData.ImpersonatedUser,
	}

	// Extend expiry
	newExpiresAt := time.Now().Add(3 * time.Hour)

	err = sm.db.UpdateSession(sessionID, userData, newExpiresAt)
	if err != nil {
		// Log warning if session update fails (DB issue)
		fmt.Printf("Warning: Failed to update session %s expiry: %v\n", sessionID[:8]+"...", err)
	}
}

// SetSessionCookie sets the session cookie in the response
func (sm *DBSessionManager) SetSessionCookie(w http.ResponseWriter, session *Session) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    session.ID,
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Path:     "/",
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

// ClearSessionCookie clears the session cookie
func (sm *DBSessionManager) ClearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

// GetSessionFromRequest extracts session from request cookie
func (sm *DBSessionManager) GetSessionFromRequest(r *http.Request) (*Session, bool) {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil, false
	}

	return sm.GetSession(cookie.Value)
}

// StartImpersonation starts impersonating another user (admin only)
func (sm *DBSessionManager) StartImpersonation(sessionID string, targetUser *users.User) error {
	sessionData, err := sm.db.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	// Only admins can impersonate
	if !sessionData.User.IsAdmin() {
		return fmt.Errorf("only administrators can impersonate users")
	}

	// Can't impersonate yourself
	if sessionData.User.Username == targetUser.Username {
		return fmt.Errorf("cannot impersonate yourself")
	}

	// Store original user if not already impersonating
	originalUser := sessionData.User
	if sessionData.IsImpersonating && sessionData.OriginalUser != nil {
		originalUser = sessionData.OriginalUser
	}

	// Update session with impersonation
	userData := map[string]interface{}{
		"user":              targetUser, // The user being impersonated becomes the current user
		"is_impersonating":  true,
		"original_user":     originalUser,
		"impersonated_user": targetUser,
	}

	// Extend session to give more time for impersonation testing
	newExpiresAt := time.Now().Add(3 * time.Hour)

	return sm.db.UpdateSession(sessionID, userData, newExpiresAt)
}

// StopImpersonation stops impersonating and returns to original user
func (sm *DBSessionManager) StopImpersonation(sessionID string) error {
	sessionData, err := sm.db.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("session not found")
	}

	if !sessionData.IsImpersonating {
		return fmt.Errorf("not currently impersonating")
	}

	// Restore original user
	userData := map[string]interface{}{
		"user":              sessionData.OriginalUser,
		"is_impersonating":  false,
		"original_user":     nil,
		"impersonated_user": nil,
	}

	return sm.db.UpdateSession(sessionID, userData, sessionData.ExpiresAt)
}

// GetImpersonationInfo returns impersonation details for a session
func (sm *DBSessionManager) GetImpersonationInfo(sessionID string) (isImpersonating bool, originalUser *users.User, impersonatedUser *users.User) {
	sessionData, err := sm.db.GetSession(sessionID)
	if err != nil {
		return false, nil, nil
	}

	return sessionData.IsImpersonating, sessionData.OriginalUser, sessionData.ImpersonatedUser
}

// cleanupExpiredSessions periodically removes expired sessions from database
func (sm *DBSessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		deleted, err := sm.db.CleanupExpiredSessions()
		if err != nil {
			fmt.Printf("Warning: Failed to cleanup expired sessions: %v\n", err)
		} else if deleted > 0 {
			fmt.Printf("✅ Cleaned up %d expired sessions\n", deleted)
		}
	}
}

// generateDBSessionID creates a cryptographically secure session ID
func generateDBSessionID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
