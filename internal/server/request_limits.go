package server

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// RequestLimitsConfig holds configuration for request limits
type RequestLimitsConfig struct {
	MaxBodySizeMB      int64            // Maximum request body size in MB
	RequestTimeout     time.Duration    // Maximum time for request processing
	MaxHeaderSize      int64            // Maximum header size in bytes
	EndpointSizeLimits map[string]int64 // Custom size limits per endpoint (path -> MB)
}

// DefaultRequestLimitsConfig returns sensible defaults
func DefaultRequestLimitsConfig() RequestLimitsConfig {
	return RequestLimitsConfig{
		MaxBodySizeMB:  1,                // 1MB default max body size
		RequestTimeout: 30 * time.Second, // 30 seconds default timeout
		MaxHeaderSize:  1 << 20,          // 1MB max headers
		EndpointSizeLimits: map[string]int64{
			"/api/specs":     5,  // 5MB for Score specs (may include large configs)
			"/api/workflows": 10, // 10MB for workflow definitions
			"/api/resources": 2,  // 2MB for resource configurations
		},
	}
}

// RequestSizeLimitMiddleware limits the size of request bodies
func (s *Server) RequestSizeLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	config := DefaultRequestLimitsConfig()

	return func(w http.ResponseWriter, r *http.Request) {
		// Skip for GET, HEAD, OPTIONS
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		// Check for custom endpoint limits
		maxSize := config.MaxBodySizeMB * 1024 * 1024 // Convert MB to bytes
		if customLimit, exists := config.EndpointSizeLimits[r.URL.Path]; exists {
			maxSize = customLimit * 1024 * 1024
		}

		// Limit request body size
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)

		// Set headers to inform client of limits
		w.Header().Set("X-Content-Length-Limit", fmt.Sprintf("%d", maxSize))

		next(w, r)
	}
}

// RequestTimeoutMiddleware enforces request timeouts
func (s *Server) RequestTimeoutMiddleware(next http.HandlerFunc) http.HandlerFunc {
	config := DefaultRequestLimitsConfig()

	return func(w http.ResponseWriter, r *http.Request) {
		// Create a context with timeout
		ctx := r.Context()

		// Set timeout
		ctx, cancel := context.WithTimeout(ctx, config.RequestTimeout)
		defer cancel()

		// Add timeout context to request
		r = r.WithContext(ctx)

		// Channel to capture if handler completed
		done := make(chan bool, 1)

		go func() {
			next(w, r)
			done <- true
		}()

		select {
		case <-done:
			// Handler completed successfully
			return
		case <-ctx.Done():
			// Timeout occurred
			http.Error(w, "Request timeout exceeded", http.StatusRequestTimeout)
			return
		}
	}
}

// SecureHeadersMiddleware adds security-related HTTP headers
func (s *Server) SecureHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (when enabled)
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy (restrictive)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline';")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (formerly Feature-Policy)
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		next(w, r)
	}
}

// ContentTypeValidationMiddleware validates Content-Type for requests
func (s *Server) ContentTypeValidationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip for GET, HEAD, OPTIONS
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		// Check Content-Type for POST, PUT, PATCH
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			http.Error(w, "Content-Type header is required", http.StatusBadRequest)
			return
		}

		// Allow specific content types
		allowedTypes := []string{
			"application/json",
			"application/yaml",
			"application/x-yaml",
			"text/yaml",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
		}

		valid := false
		for _, allowed := range allowedTypes {
			if len(contentType) >= len(allowed) && contentType[:len(allowed)] == allowed {
				valid = true
				break
			}
		}

		if !valid {
			http.Error(w, fmt.Sprintf("Invalid Content-Type: %s", contentType), http.StatusUnsupportedMediaType)
			return
		}

		next(w, r)
	}
}

// Note: getClientIP is defined in handlers.go
