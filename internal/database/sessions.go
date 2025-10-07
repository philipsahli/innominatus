package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"innominatus/internal/users"
	"time"
)

// SessionData represents session information stored in the database
type SessionData struct {
	ID               int64       `json:"id"`
	SessionID        string      `json:"session_id"`
	User             *users.User `json:"user"`
	OriginalUser     *users.User `json:"original_user,omitempty"`
	ImpersonatedUser *users.User `json:"impersonated_user,omitempty"`
	IsImpersonating  bool        `json:"is_impersonating"`
	CreatedAt        time.Time   `json:"created_at"`
	ExpiresAt        time.Time   `json:"expires_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// CreateSession stores a new session in the database
func (d *Database) CreateSession(sessionID string, user *users.User, expiresAt time.Time) error {
	if d.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Diagnostic: Check which database we're connected to
	var currentDB string
	err := d.db.QueryRow("SELECT current_database()").Scan(&currentDB)
	if err != nil {
		return fmt.Errorf("failed to get current database: %w", err)
	}
	fmt.Printf("DEBUG: CreateSession - Database pointer %p connected to: %s\n", d, currentDB)

	userData := map[string]interface{}{
		"user":              user,
		"is_impersonating":  false,
		"original_user":     nil,
		"impersonated_user": nil,
	}

	userJSON, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	query := `
		INSERT INTO sessions (session_id, user_data, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
	`

	_, err = d.db.Exec(query, sessionID, userJSON, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to create session in database: %w", err)
	}

	return nil
}

// GetSession retrieves a session by session ID
func (d *Database) GetSession(sessionID string) (*SessionData, error) {
	query := `
		SELECT id, session_id, user_data, created_at, expires_at, updated_at
		FROM sessions
		WHERE session_id = $1 AND expires_at > NOW()
	`

	var session SessionData
	var userJSON []byte

	err := d.db.QueryRow(query, sessionID).Scan(
		&session.ID,
		&session.SessionID,
		&userJSON,
		&session.CreatedAt,
		&session.ExpiresAt,
		&session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found or expired")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Unmarshal user data
	var userData map[string]interface{}
	if err := json.Unmarshal(userJSON, &userData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user data: %w", err)
	}

	// Extract user
	if userMap, ok := userData["user"].(map[string]interface{}); ok {
		userBytes, _ := json.Marshal(userMap)
		var user users.User
		if err := json.Unmarshal(userBytes, &user); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user: %w", err)
		}
		session.User = &user
	}

	// Extract impersonation data
	if isImp, ok := userData["is_impersonating"].(bool); ok {
		session.IsImpersonating = isImp
	}

	if session.IsImpersonating {
		if origUserMap, ok := userData["original_user"].(map[string]interface{}); ok && origUserMap != nil {
			userBytes, _ := json.Marshal(origUserMap)
			var origUser users.User
			if err := json.Unmarshal(userBytes, &origUser); err == nil {
				session.OriginalUser = &origUser
			}
		}

		if impUserMap, ok := userData["impersonated_user"].(map[string]interface{}); ok && impUserMap != nil {
			userBytes, _ := json.Marshal(impUserMap)
			var impUser users.User
			if err := json.Unmarshal(userBytes, &impUser); err == nil {
				session.ImpersonatedUser = &impUser
			}
		}
	}

	return &session, nil
}

// UpdateSession updates an existing session (for extending expiry or updating impersonation)
func (d *Database) UpdateSession(sessionID string, userData map[string]interface{}, expiresAt time.Time) error {
	userJSON, err := json.Marshal(userData)
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	query := `
		UPDATE sessions
		SET user_data = $1, expires_at = $2, updated_at = NOW()
		WHERE session_id = $3
	`

	result, err := d.db.Exec(query, userJSON, expiresAt, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// DeleteSession removes a session from the database
func (d *Database) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = $1`

	result, err := d.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// CleanupExpiredSessions removes all expired sessions
func (d *Database) CleanupExpiredSessions() (int64, error) {
	query := `DELETE FROM sessions WHERE expires_at <= NOW()`

	result, err := d.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// CountActiveSessions returns the number of active (non-expired) sessions
func (d *Database) CountActiveSessions() (int64, error) {
	query := `SELECT COUNT(*) FROM sessions WHERE expires_at > NOW()`

	var count int64
	err := d.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count active sessions: %w", err)
	}

	return count, nil
}
