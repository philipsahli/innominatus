package health

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check result for a component
type Check struct {
	Name      string        `json:"name"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Latency   time.Duration `json:"latency_ms"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthResponse represents the overall health status
type HealthResponse struct {
	Status    Status           `json:"status"`
	Timestamp time.Time        `json:"timestamp"`
	Uptime    time.Duration    `json:"uptime_seconds"`
	Checks    map[string]Check `json:"checks"`
}

// ReadinessResponse represents the readiness status
type ReadinessResponse struct {
	Ready     bool      `json:"ready"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message,omitempty"`
}

// Checker defines the interface for health checks
type Checker interface {
	Check(ctx context.Context) Check
	Name() string
}

// HealthChecker manages multiple health checks
type HealthChecker struct {
	checkers  []Checker
	startTime time.Time
	mu        sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checkers:  make([]Checker, 0),
		startTime: time.Now(),
	}
}

// Register adds a new health checker
func (h *HealthChecker) Register(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// CheckAll runs all registered health checks
func (h *HealthChecker) CheckAll(ctx context.Context) HealthResponse {
	h.mu.RLock()
	checkers := make([]Checker, len(h.checkers))
	copy(checkers, h.checkers)
	h.mu.RUnlock()

	checks := make(map[string]Check)
	overallStatus := StatusHealthy

	// Run all checks in parallel
	var wg sync.WaitGroup
	resultChan := make(chan Check, len(checkers))

	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			result := c.Check(ctx)
			resultChan <- result
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for check := range resultChan {
		checks[check.Name] = check

		// Determine overall status
		if check.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if check.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	return HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime),
		Checks:    checks,
	}
}

// IsReady checks if the service is ready to serve traffic
func (h *HealthChecker) IsReady(ctx context.Context) ReadinessResponse {
	response := h.CheckAll(ctx)

	// Service is ready if all critical checks are healthy
	// For now, we consider degraded as ready (can still serve traffic)
	ready := response.Status != StatusUnhealthy

	message := "Service is ready"
	if !ready {
		message = "Service is not ready - critical dependencies are unhealthy"
	}

	return ReadinessResponse{
		Ready:     ready,
		Timestamp: time.Now(),
		Message:   message,
	}
}

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	db      *sql.DB
	timeout time.Duration
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(db *sql.DB, timeout time.Duration) *DatabaseChecker {
	return &DatabaseChecker{
		db:      db,
		timeout: timeout,
	}
}

// Name returns the checker name
func (c *DatabaseChecker) Name() string {
	return "database"
}

// Check performs the database health check
func (c *DatabaseChecker) Check(ctx context.Context) Check {
	start := time.Now()
	check := Check{
		Name:      c.Name(),
		Timestamp: start,
	}

	if c.db == nil {
		check.Status = StatusDegraded
		check.Message = "Database connection not configured (running in memory mode)"
		check.Latency = time.Since(start)
		return check
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Ping database
	if err := c.db.PingContext(timeoutCtx); err != nil {
		check.Status = StatusUnhealthy
		check.Error = fmt.Sprintf("Database ping failed: %v", err)
		check.Latency = time.Since(start)
		return check
	}

	// Check connection stats
	stats := c.db.Stats()
	if stats.OpenConnections == 0 {
		check.Status = StatusDegraded
		check.Message = "No active database connections"
	} else {
		check.Status = StatusHealthy
		check.Message = fmt.Sprintf("%d active connections", stats.OpenConnections)
	}

	check.Latency = time.Since(start)
	return check
}

// AlwaysHealthyChecker is a simple checker that always returns healthy
type AlwaysHealthyChecker struct {
	name string
}

// NewAlwaysHealthyChecker creates a checker that always returns healthy
func NewAlwaysHealthyChecker(name string) *AlwaysHealthyChecker {
	return &AlwaysHealthyChecker{name: name}
}

// Name returns the checker name
func (c *AlwaysHealthyChecker) Name() string {
	return c.name
}

// Check always returns healthy status
func (c *AlwaysHealthyChecker) Check(ctx context.Context) Check {
	return Check{
		Name:      c.name,
		Status:    StatusHealthy,
		Message:   "OK",
		Timestamp: time.Now(),
		Latency:   0,
	}
}
