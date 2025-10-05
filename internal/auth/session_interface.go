package auth

import (
	"innominatus/internal/users"
	"net/http"
)

// ISessionManager defines the interface for session management
type ISessionManager interface {
	CreateSession(user *users.User) (*Session, error)
	GetSession(sessionID string) (*Session, bool)
	DeleteSession(sessionID string)
	ExtendSession(sessionID string)
	SetSessionCookie(w http.ResponseWriter, session *Session)
	ClearSessionCookie(w http.ResponseWriter)
	GetSessionFromRequest(r *http.Request) (*Session, bool)
	StartImpersonation(sessionID string, targetUser *users.User) error
	StopImpersonation(sessionID string) error
	GetImpersonationInfo(sessionID string) (isImpersonating bool, originalUser *users.User, impersonatedUser *users.User)
}
