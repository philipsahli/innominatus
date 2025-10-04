package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		bucket := NewTokenBucket(60, 10) // 60 req/min, 10 burst

		// Should allow burst requests
		for i := 0; i < 10; i++ {
			if !bucket.TryConsume(1) {
				t.Errorf("Request %d should be allowed within burst", i+1)
			}
		}
	})

	t.Run("blocks requests exceeding burst", func(t *testing.T) {
		bucket := NewTokenBucket(60, 5) // 60 req/min, 5 burst

		// Consume all burst tokens
		for i := 0; i < 5; i++ {
			bucket.TryConsume(1)
		}

		// Next request should be blocked
		if bucket.TryConsume(1) {
			t.Error("Request should be blocked after burst exhausted")
		}
	})

	t.Run("refills tokens over time", func(t *testing.T) {
		bucket := NewTokenBucket(60, 5) // 60 req/min = 1 req/sec, 5 burst

		// Consume all tokens
		for i := 0; i < 5; i++ {
			bucket.TryConsume(1)
		}

		// Wait for 1 token to refill (1 second)
		time.Sleep(1100 * time.Millisecond)

		// Should allow 1 more request
		if !bucket.TryConsume(1) {
			t.Error("Token should have refilled after 1 second")
		}
	})
}

func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within per-user limit", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:    true,
			PerUserRPM: 100,
			PerIPRPM:   200,
			BurstSize:  10,
		}
		rl := NewRateLimiter(config)

		// User should be allowed within burst
		for i := 0; i < 10; i++ {
			allowed, _ := rl.Allow("user1", "192.168.1.1", "/api/test")
			if !allowed {
				t.Errorf("Request %d should be allowed for user within burst", i+1)
			}
		}
	})

	t.Run("blocks requests exceeding per-user limit", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:    true,
			PerUserRPM: 60,
			PerIPRPM:   200,
			BurstSize:  5,
		}
		rl := NewRateLimiter(config)

		// Consume all burst tokens
		for i := 0; i < 5; i++ {
			rl.Allow("user1", "192.168.1.1", "/api/test")
		}

		// Next request should be blocked
		allowed, reason := rl.Allow("user1", "192.168.1.1", "/api/test")
		if allowed {
			t.Error("Request should be blocked for user exceeding limit")
		}
		if reason == "" {
			t.Error("Reason should be provided for blocked request")
		}
	})

	t.Run("enforces per-IP limit independently", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:    true,
			PerUserRPM: 100,
			PerIPRPM:   5,
			BurstSize:  5,
		}
		rl := NewRateLimiter(config)

		// Different users from same IP
		for i := 0; i < 5; i++ {
			rl.Allow("", "192.168.1.1", "/api/test")
		}

		// Next request from same IP should be blocked regardless of user
		allowed, _ := rl.Allow("user2", "192.168.1.1", "/api/test")
		if allowed {
			t.Error("Request should be blocked due to IP limit")
		}
	})

	t.Run("applies endpoint-specific limits", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:    true,
			PerUserRPM: 100,
			PerIPRPM:   200,
			BurstSize:  10,
			EndpointLimits: map[string]int{
				"/api/login": 3,
			},
		}
		rl := NewRateLimiter(config)

		// Consume login endpoint limit (3 + burst = different logic)
		for i := 0; i < 10; i++ {
			rl.Allow("user1", "192.168.1.1", "/api/login")
		}

		// Should be blocked on login endpoint
		allowed, reason := rl.Allow("user1", "192.168.1.1", "/api/login")
		if allowed {
			t.Error("Request should be blocked for login endpoint")
		}
		if reason == "" {
			t.Error("Reason should mention endpoint limit")
		}
	})

	t.Run("different IPs are independent", func(t *testing.T) {
		config := RateLimitConfig{
			Enabled:    true,
			PerUserRPM: 100,
			PerIPRPM:   5,
			BurstSize:  5,
		}
		rl := NewRateLimiter(config)

		// Exhaust limit for IP1
		for i := 0; i < 5; i++ {
			rl.Allow("", "192.168.1.1", "/api/test")
		}

		// IP2 should still be allowed
		allowed, _ := rl.Allow("", "192.168.1.2", "/api/test")
		if !allowed {
			t.Error("Different IP should not be affected by other IP's limit")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		server := &Server{
			rateLimiter: NewRateLimiter(RateLimitConfig{
				Enabled:    true,
				PerUserRPM: 100,
				PerIPRPM:   200,
				BurstSize:  10,
			}),
		}

		handler := server.RateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()

		handler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		server := &Server{
			rateLimiter: NewRateLimiter(RateLimitConfig{
				Enabled:    true,
				PerUserRPM: 60,
				PerIPRPM:   5,
				BurstSize:  5,
			}),
		}

		handler := server.RateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make 5 requests to exhaust limit
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			handler(rec, req)
		}

		// 6th request should be blocked
		req := httptest.NewRequest("GET", "/api/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rec := httptest.NewRecorder()
		handler(rec, req)

		if rec.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status 429, got %d", rec.Code)
		}

		// Check rate limit headers
		if rec.Header().Get("Retry-After") == "" {
			t.Error("Retry-After header should be set")
		}
	})

	t.Run("bypasses rate limiting when disabled", func(t *testing.T) {
		server := &Server{
			rateLimiter: nil, // Disabled
		}

		handler := server.RateLimitMiddleware(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		// Make many requests - all should succeed
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			rec := httptest.NewRecorder()
			handler(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i+1, rec.Code)
			}
		}
	})
}

func TestGetRateLimitStats(t *testing.T) {
	config := RateLimitConfig{
		Enabled:    true,
		PerUserRPM: 100,
		PerIPRPM:   200,
		BurstSize:  10,
	}
	rl := NewRateLimiter(config)

	// Make some requests to create buckets
	rl.Allow("user1", "192.168.1.1", "/api/test")
	rl.Allow("user2", "192.168.1.2", "/api/test")

	stats := rl.GetRateLimitStats()

	if stats["total_user_buckets"].(int) != 2 {
		t.Errorf("Expected 2 user buckets, got %d", stats["total_user_buckets"])
	}

	if stats["total_ip_buckets"].(int) != 2 {
		t.Errorf("Expected 2 IP buckets, got %d", stats["total_ip_buckets"])
	}

	if stats["per_user_rpm"].(int) != 100 {
		t.Errorf("Expected per_user_rpm 100, got %d", stats["per_user_rpm"])
	}
}

func BenchmarkRateLimiter(b *testing.B) {
	config := RateLimitConfig{
		Enabled:    true,
		PerUserRPM: 10000,
		PerIPRPM:   20000,
		BurstSize:  100,
	}
	rl := NewRateLimiter(config)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow("user1", "192.168.1.1", "/api/test")
	}
}

func BenchmarkRateLimiterConcurrent(b *testing.B) {
	config := RateLimitConfig{
		Enabled:    true,
		PerUserRPM: 10000,
		PerIPRPM:   20000,
		BurstSize:  100,
	}
	rl := NewRateLimiter(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			userID := fmt.Sprintf("user%d", i%10)
			ip := fmt.Sprintf("192.168.1.%d", i%255)
			rl.Allow(userID, ip, "/api/test")
			i++
		}
	})
}
