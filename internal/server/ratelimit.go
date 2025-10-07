package server

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiting algorithm
type RateLimiter struct {
	perUserLimit   int           // requests per minute per user
	perIPLimit     int           // requests per minute per IP
	burstSize      int           // burst allowance
	cleanupPeriod  time.Duration // how often to cleanup old entries
	userBuckets    map[string]*TokenBucket
	ipBuckets      map[string]*TokenBucket
	endpointLimits map[string]int // custom limits per endpoint
	mu             sync.RWMutex
}

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	Enabled        bool
	PerUserRPM     int            // Requests Per Minute per user
	PerIPRPM       int            // Requests Per Minute per IP
	BurstSize      int            // Burst allowance
	CleanupPeriod  time.Duration  // Cleanup interval
	EndpointLimits map[string]int // Custom limits per endpoint (path -> RPM)
}

// DefaultRateLimitConfig returns sensible defaults
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:       true,
		PerUserRPM:    100, // 100 requests per minute per user
		PerIPRPM:      200, // 200 requests per minute per IP
		BurstSize:     10,  // Allow bursts of 10 requests
		CleanupPeriod: 5 * time.Minute,
		EndpointLimits: map[string]int{
			"/api/login":     10, // Stricter for login
			"/api/specs":     50, // Moderate for spec submission
			"/api/workflows": 30, // Moderate for workflow execution
			"/api/admin":     20, // Stricter for admin operations
		},
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	// Set default cleanup period if not specified
	cleanupPeriod := config.CleanupPeriod
	if cleanupPeriod == 0 {
		cleanupPeriod = 5 * time.Minute
	}

	rl := &RateLimiter{
		perUserLimit:   config.PerUserRPM,
		perIPLimit:     config.PerIPRPM,
		burstSize:      config.BurstSize,
		cleanupPeriod:  cleanupPeriod,
		userBuckets:    make(map[string]*TokenBucket),
		ipBuckets:      make(map[string]*TokenBucket),
		endpointLimits: config.EndpointLimits,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if the request should be allowed
func (rl *RateLimiter) Allow(userID, ip, endpoint string) (bool, string) {
	// Check endpoint-specific limit first
	if limit, exists := rl.endpointLimits[endpoint]; exists {
		if userID != "" {
			bucket := rl.getOrCreateUserBucket(userID, limit)
			if !bucket.TryConsume(1) {
				return false, fmt.Sprintf("Rate limit exceeded for endpoint %s: %d req/min", endpoint, limit)
			}
		}
		ipBucket := rl.getOrCreateIPBucket(ip, limit)
		if !ipBucket.TryConsume(1) {
			return false, fmt.Sprintf("Rate limit exceeded for IP on endpoint %s: %d req/min", endpoint, limit)
		}
		return true, ""
	}

	// Check per-user limit
	if userID != "" {
		bucket := rl.getOrCreateUserBucket(userID, rl.perUserLimit)
		if !bucket.TryConsume(1) {
			return false, fmt.Sprintf("Rate limit exceeded for user: %d req/min", rl.perUserLimit)
		}
	}

	// Check per-IP limit
	ipBucket := rl.getOrCreateIPBucket(ip, rl.perIPLimit)
	if !ipBucket.TryConsume(1) {
		return false, fmt.Sprintf("Rate limit exceeded for IP: %d req/min", rl.perIPLimit)
	}

	return true, ""
}

// getOrCreateUserBucket gets or creates a token bucket for a user
func (rl *RateLimiter) getOrCreateUserBucket(userID string, limit int) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if bucket, exists := rl.userBuckets[userID]; exists {
		return bucket
	}

	bucket := NewTokenBucket(limit, rl.burstSize)
	rl.userBuckets[userID] = bucket
	return bucket
}

// getOrCreateIPBucket gets or creates a token bucket for an IP
func (rl *RateLimiter) getOrCreateIPBucket(ip string, limit int) *TokenBucket {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if bucket, exists := rl.ipBuckets[ip]; exists {
		return bucket
	}

	bucket := NewTokenBucket(limit, rl.burstSize)
	rl.ipBuckets[ip] = bucket
	return bucket
}

// cleanup periodically removes old unused buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupPeriod)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()

		// Cleanup user buckets that haven't been used in 10 minutes
		for userID, bucket := range rl.userBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.userBuckets, userID)
			}
			bucket.mu.Unlock()
		}

		// Cleanup IP buckets that haven't been used in 10 minutes
		for ip, bucket := range rl.ipBuckets {
			bucket.mu.Lock()
			if now.Sub(bucket.lastRefill) > 10*time.Minute {
				delete(rl.ipBuckets, ip)
			}
			bucket.mu.Unlock()
		}

		rl.mu.Unlock()
	}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(ratePerMinute, burst int) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: float64(ratePerMinute) / 60.0, // Convert to tokens per second
		lastRefill: time.Now(),
	}
}

// TryConsume attempts to consume tokens from the bucket
func (tb *TokenBucket) TryConsume(tokens float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= tokens {
		tb.tokens -= tokens
		return true
	}

	return false
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Add tokens based on elapsed time
	tb.tokens += elapsed * tb.refillRate

	// Cap at max tokens
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	tb.lastRefill = now
}

// RateLimitMiddleware creates a rate limiting middleware
func (s *Server) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting if disabled
		if s.rateLimiter == nil {
			next(w, r)
			return
		}

		// Get user ID from context (if authenticated)
		userID := ""
		if user := s.getUserFromContext(r); user != nil {
			userID = user.Username
		}

		// Get client IP
		clientIP := getClientIP(r)

		// Get endpoint for custom limits
		endpoint := r.URL.Path

		// Check rate limit
		allowed, reason := s.rateLimiter.Allow(userID, clientIP, endpoint)
		if !allowed {
			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", s.rateLimiter.perUserLimit))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("Retry-After", "60")

			http.Error(w, fmt.Sprintf("Rate limit exceeded: %s", reason), http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// GetRateLimitStats returns current rate limiting statistics
func (rl *RateLimiter) GetRateLimitStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	return map[string]interface{}{
		"total_user_buckets": len(rl.userBuckets),
		"total_ip_buckets":   len(rl.ipBuckets),
		"per_user_rpm":       rl.perUserLimit,
		"per_ip_rpm":         rl.perIPLimit,
		"burst_size":         rl.burstSize,
	}
}
