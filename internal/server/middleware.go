package server

import (
	"context"
	"fmt"
	"innominatus/internal/auth"
	"innominatus/internal/logging"
	"innominatus/internal/users"
	"log"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	contextKeyUser       contextKey = "user"
	contextKeyTeamFilter contextKey = "team_filter"
)

// CorsMiddleware adds CORS headers to allow cross-origin requests from the frontend
func (s *Server) CorsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// SECURITY: Explicit origin whitelist - never use wildcard with credentials
		origin := r.Header.Get("Origin")
		allowedOrigins := map[string]bool{
			"http://localhost:3000":           true, // Next.js dev server
			"http://localhost:3001":           true, // Alternative dev port
			"http://localhost:8081":           true, // Same-origin
			"http://innominatus.localtest.me": true, // Demo environment
		}

		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		// SECURITY: No CORS headers for requests without Origin (same-origin requests don't need CORS)
		// Browsers automatically allow same-origin requests

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Trace-Id")
		w.Header().Set("Access-Control-Expose-Headers", "X-Trace-Id")

		// Handle preflight OPTIONS request
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// TraceIDMiddleware adds trace ID to request context and response headers
// This enables request tracing across the entire request lifecycle
func (s *Server) TraceIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if trace ID is provided in request header (for distributed tracing)
		traceID := r.Header.Get("X-Trace-Id")

		// If no custom trace ID, try to get from OpenTelemetry span context
		if traceID == "" {
			span := trace.SpanFromContext(r.Context())
			if span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}
		}

		if traceID == "" {
			// Generate new trace ID if not provided and no span context
			traceID = logging.GenerateTraceID()
		}

		// Add trace ID to request context
		ctx := logging.WithTraceID(r.Context(), traceID)

		// Also add as request ID for backward compatibility
		ctx = logging.WithRequestID(ctx, traceID)

		// Update request with new context
		r = r.WithContext(ctx)

		// Add trace ID to response headers for client visibility
		w.Header().Set("X-Trace-Id", traceID)

		// Call next handler
		next(w, r)
	}
}

// TracingMiddleware creates OpenTelemetry spans for HTTP requests
// This provides distributed tracing for all HTTP requests
func (s *Server) TracingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	tracer := otel.Tracer("innominatus/http")

	return func(w http.ResponseWriter, r *http.Request) {
		// Start a new span for this HTTP request
		ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.scheme", r.URL.Scheme),
				attribute.String("http.host", r.Host),
				attribute.String("http.target", r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
				attribute.String("http.client_ip", getClientIP(r)),
			),
		)
		defer span.End()

		// Update request with span context
		r = r.WithContext(ctx)

		// Wrap response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		// Call next handler
		next(rw, r)

		// Add response attributes to span
		span.SetAttributes(
			attribute.Int("http.status_code", rw.statusCode),
			attribute.Int("http.response_size", rw.size),
		)

		// Record error if status code indicates an error
		if rw.statusCode >= 400 {
			span.SetAttributes(attribute.Bool("http.error", true))
		}
	}
}

// AuthMiddleware provides authentication for web requests
func (s *Server) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for OPTIONS requests (CORS preflight)
		if r.Method == "OPTIONS" {
			next(w, r)
			return
		}

		// Skip authentication for login, logout, and static assets
		if s.isPublicPath(r.URL.Path) {
			next(w, r)
			return
		}

		// Check for valid session (cookie or Authorization header)
		session, exists := s.getSessionFromRequestWithToken(r)
		if !exists {
			// Redirect to login for web pages
			if s.isWebRequest(r) {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			// Return 401 for API requests
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extend session on activity
		s.sessionManager.ExtendSession(session.ID)

		// Add user to request context
		ctx := context.WithValue(r.Context(), contextKeyUser, session.User)
		r = r.WithContext(ctx)

		next(w, r)
	}
}

// AdminOnlyMiddleware restricts access to admin users only
func (s *Server) AdminOnlyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return s.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user := s.getUserFromContext(r)
		if user == nil || !user.IsAdmin() {
			if s.isWebRequest(r) {
				http.Error(w, "Access Denied: Admin privileges required", http.StatusForbidden)
			} else {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}
			return
		}
		next(w, r)
	})
}

// isPublicPath checks if a path should be accessible without authentication
func (s *Server) isPublicPath(path string) bool {
	publicPaths := []string{
		"/login",
		"/logout",
		"/api/login",
		"/favicon.ico",
	}

	for _, publicPath := range publicPaths {
		if path == publicPath {
			return true
		}
	}

	// Allow static assets (if any)
	// SECURITY: Removed /v2/ wildcard - no internal routes use this prefix
	return strings.HasPrefix(path, "/static/")
}

// isWebRequest determines if this is a web browser request vs API request
func (s *Server) isWebRequest(r *http.Request) bool {
	// Check Accept header for HTML content
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html") || accept == ""
}

// TeamFilterMiddleware ensures users only see their team's data
func (s *Server) TeamFilterMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return s.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		user := s.getUserFromContext(r)
		if user == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add team filter to request context for API handlers to use
		ctx := context.WithValue(r.Context(), contextKeyTeamFilter, user.Team)
		r = r.WithContext(ctx)

		next(w, r)
	})
}

// getTeamFromContext retrieves team filter from request context
//
//nolint:unused // Reserved for future multi-tenancy filtering
func (s *Server) getTeamFromContext(r *http.Request) string {
	if team, ok := r.Context().Value(contextKeyTeamFilter).(string); ok {
		return team
	}
	return ""
}

// getSessionFromRequestWithToken checks for session from cookie or Authorization header
func (s *Server) getSessionFromRequestWithToken(r *http.Request) (*auth.Session, bool) {
	// First try to get session from cookie (for web UI)
	if session, exists := s.sessionManager.GetSessionFromRequest(r); exists {
		return session, true
	}

	// Then try to get token from Authorization header (for CLI/API)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		// Support "Bearer <token>" format
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token := authHeader[7:]

			// First try session token
			if session, exists := s.sessionManager.GetSession(token); exists {
				return session, true
			}

			// Then try API key authentication
			if user, err := s.authenticateWithAPIKey(token); err == nil {
				// Create a temporary session for the API key user
				session := &auth.Session{
					ID:        token, // Use API key as session ID
					User:      user,
					CreatedAt: time.Now(),
					ExpiresAt: time.Now().Add(24 * time.Hour), // Temporary session
				}
				return session, true
			}
		}
	}

	return nil, false
}

// authenticateWithAPIKey validates an API key and returns the associated user
// Checks both file-based users (users.yaml) and database-stored API keys (OIDC users)
func (s *Server) authenticateWithAPIKey(apiKey string) (*users.User, error) {
	// First try file-based users (users.yaml)
	store, err := users.LoadUsers()
	if err == nil {
		if user, err := store.AuthenticateWithAPIKey(apiKey); err == nil {
			return user, nil
		}
	}

	// Then try database API keys (for OIDC users)
	if s.db != nil {
		keyHash := hashAPIKey(apiKey)
		username, team, role, err := s.db.GetUserByAPIKeyHash(keyHash)
		if err == nil {
			// Update last used timestamp
			_ = s.db.UpdateAPIKeyLastUsed(keyHash)

			// Return user object (OIDC user from database)
			return &users.User{
				Username: username,
				Team:     team,
				Role:     role,
			}, nil
		}
	}

	return nil, fmt.Errorf("invalid API key")
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// LoggingMiddleware logs HTTP requests in access log format with trace IDs
func (s *Server) LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code and response size
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     200, // default status code
		}

		// Get client IP
		clientIP := getClientIP(r)

		// Store original request for logging
		method := r.Method
		path := r.URL.Path
		if r.URL.RawQuery != "" {
			path += "?" + r.URL.RawQuery
		}
		proto := r.Proto
		userAgent := r.UserAgent()

		// Get trace ID from context if available
		traceID := logging.GetTraceID(r.Context())
		if traceID == "" {
			traceID = "-"
		}

		// Call the next handler
		next(rw, r)

		// Get user info if available (after authentication middleware has run)
		username := "-"
		if user := s.getUserFromContext(r); user != nil {
			username = user.Username
		}

		// Calculate duration
		duration := time.Since(start)

		// Log in Common Log Format (CLF) with trace ID and additional info
		log.Printf("%s - %s [%s] \"%s %s %s\" %d %d %v \"%s\" trace_id=%s",
			clientIP,
			username,
			start.Format("02/Jan/2006:15:04:05 -0700"),
			method,
			path,
			proto,
			rw.statusCode,
			rw.size,
			duration,
			userAgent,
			traceID,
		)
	}
}
