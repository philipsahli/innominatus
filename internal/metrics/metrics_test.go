package metrics

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGetGlobal(t *testing.T) {
	m := GetGlobal()
	if m == nil {
		t.Fatal("GetGlobal() returned nil")
	}

	// Calling GetGlobal() twice should return the same instance
	m2 := GetGlobal()
	if m != m2 {
		t.Error("GetGlobal() returned different instances")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
	}

	// Record a successful request (status 200)
	m.RecordHTTPRequest("GET", "/api/specs", 200)

	m.mu.RLock()
	count := m.httpRequestsTotal["GET"]["/api/specs"]
	errorCount := m.httpRequestErrors["/api/specs"]
	m.mu.RUnlock()

	if count != 1 {
		t.Errorf("Expected request count = 1, got %d", count)
	}

	if errorCount != 0 {
		t.Errorf("Expected error count = 0 for 200 status, got %d", errorCount)
	}

	// Record multiple requests
	m.RecordHTTPRequest("GET", "/api/specs", 200)
	m.RecordHTTPRequest("POST", "/api/specs", 201)

	m.mu.RLock()
	getCount := m.httpRequestsTotal["GET"]["/api/specs"]
	postCount := m.httpRequestsTotal["POST"]["/api/specs"]
	m.mu.RUnlock()

	if getCount != 2 {
		t.Errorf("Expected GET count = 2, got %d", getCount)
	}

	if postCount != 1 {
		t.Errorf("Expected POST count = 1, got %d", postCount)
	}
}

func TestRecordHTTPRequest_Errors(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
	}

	// Record error requests (5xx status codes)
	m.RecordHTTPRequest("GET", "/api/workflows", 500)
	m.RecordHTTPRequest("GET", "/api/workflows", 502)
	m.RecordHTTPRequest("GET", "/api/workflows", 503)

	m.mu.RLock()
	errorCount := m.httpRequestErrors["/api/workflows"]
	totalCount := m.httpRequestsTotal["GET"]["/api/workflows"]
	m.mu.RUnlock()

	if errorCount != 3 {
		t.Errorf("Expected error count = 3, got %d", errorCount)
	}

	if totalCount != 3 {
		t.Errorf("Expected total count = 3, got %d", totalCount)
	}

	// Non-error status codes should not increment error count
	m.RecordHTTPRequest("GET", "/api/specs", 200)
	m.RecordHTTPRequest("GET", "/api/specs", 404)
	m.RecordHTTPRequest("GET", "/api/specs", 400)

	m.mu.RLock()
	specsErrorCount := m.httpRequestErrors["/api/specs"]
	m.mu.RUnlock()

	if specsErrorCount != 0 {
		t.Errorf("Expected error count = 0 for 4xx and 2xx status codes, got %d", specsErrorCount)
	}
}

func TestRecordWorkflowExecution(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
		workflowDurations: make([]time.Duration, 0, 100),
	}

	// Record successful workflow
	m.RecordWorkflowExecution(true, 5*time.Second)

	m.mu.RLock()
	executed := m.workflowsExecuted
	succeeded := m.workflowsSucceeded
	failed := m.workflowsFailed
	m.mu.RUnlock()

	if executed != 1 {
		t.Errorf("Expected executed = 1, got %d", executed)
	}

	if succeeded != 1 {
		t.Errorf("Expected succeeded = 1, got %d", succeeded)
	}

	if failed != 0 {
		t.Errorf("Expected failed = 0, got %d", failed)
	}

	// Record failed workflow
	m.RecordWorkflowExecution(false, 3*time.Second)

	m.mu.RLock()
	executed = m.workflowsExecuted
	succeeded = m.workflowsSucceeded
	failed = m.workflowsFailed
	durationsCount := len(m.workflowDurations)
	m.mu.RUnlock()

	if executed != 2 {
		t.Errorf("Expected executed = 2, got %d", executed)
	}

	if succeeded != 1 {
		t.Errorf("Expected succeeded = 1, got %d", succeeded)
	}

	if failed != 1 {
		t.Errorf("Expected failed = 1, got %d", failed)
	}

	if durationsCount != 2 {
		t.Errorf("Expected 2 durations recorded, got %d", durationsCount)
	}
}

func TestRecordWorkflowExecution_DurationLimit(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
		workflowDurations: make([]time.Duration, 0, 100),
	}

	// Record 101 workflows to test the 100-duration limit
	for i := 0; i < 101; i++ {
		m.RecordWorkflowExecution(true, time.Duration(i)*time.Second)
	}

	m.mu.RLock()
	durationsCount := len(m.workflowDurations)
	m.mu.RUnlock()

	if durationsCount != 100 {
		t.Errorf("Expected max 100 durations, got %d", durationsCount)
	}

	// Verify oldest duration was removed (should be 1 second, not 0)
	m.mu.RLock()
	firstDuration := m.workflowDurations[0]
	m.mu.RUnlock()

	if firstDuration != 1*time.Second {
		t.Errorf("Expected first duration = 1s (oldest should be removed), got %v", firstDuration)
	}
}

func TestRecordDBQuery(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
	}

	// Record successful query
	m.RecordDBQuery(nil)

	m.mu.RLock()
	total := m.dbQueriesTotal
	errorCount := m.dbQueryErrors
	m.mu.RUnlock()

	if total != 1 {
		t.Errorf("Expected total = 1, got %d", total)
	}

	if errorCount != 0 {
		t.Errorf("Expected errors = 0, got %d", errorCount)
	}

	// Record failed query
	m.RecordDBQuery(errors.New("connection timeout"))

	m.mu.RLock()
	total = m.dbQueriesTotal
	errorCount = m.dbQueryErrors
	m.mu.RUnlock()

	if total != 2 {
		t.Errorf("Expected total = 2, got %d", total)
	}

	if errorCount != 1 {
		t.Errorf("Expected errors = 1, got %d", errorCount)
	}
}

func TestRecordResourceCount(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
	}

	tests := []struct {
		resourceType string
		count        int64
		checkField   *int64
	}{
		{"native", 10, &m.resourcesNative},
		{"delegated", 5, &m.resourcesDelegated},
		{"external", 3, &m.resourcesExternal},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			m.RecordResourceCount(tt.resourceType, tt.count)

			m.mu.RLock()
			value := *tt.checkField
			m.mu.RUnlock()

			if value != tt.count {
				t.Errorf("Expected %s count = %d, got %d", tt.resourceType, tt.count, value)
			}
		})
	}

	// Test unknown resource type (should not panic)
	m.RecordResourceCount("unknown", 99)
}

func TestRecordExternalResourceHealth(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal: make(map[string]map[string]int64),
		httpRequestErrors: make(map[string]int64),
		startTime:         time.Now(),
	}

	m.RecordExternalResourceHealth(8, 2)

	m.mu.RLock()
	healthy := m.resourcesExternalHealthy
	failed := m.resourcesExternalFailed
	m.mu.RUnlock()

	if healthy != 8 {
		t.Errorf("Expected healthy = 8, got %d", healthy)
	}

	if failed != 2 {
		t.Errorf("Expected failed = 2, got %d", failed)
	}
}

func TestRecordGitOpsWaitDuration(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal:   make(map[string]map[string]int64),
		httpRequestErrors:   make(map[string]int64),
		startTime:           time.Now(),
		gitopsWaitDurations: make([]time.Duration, 0, 100),
	}

	// Record a GitOps wait duration
	m.RecordGitOpsWaitDuration(30 * time.Second)

	m.mu.RLock()
	count := len(m.gitopsWaitDurations)
	duration := m.gitopsWaitDurations[0]
	m.mu.RUnlock()

	if count != 1 {
		t.Errorf("Expected 1 duration recorded, got %d", count)
	}

	if duration != 30*time.Second {
		t.Errorf("Expected duration = 30s, got %v", duration)
	}

	// Test 100-duration limit
	for i := 0; i < 101; i++ {
		m.RecordGitOpsWaitDuration(time.Duration(i) * time.Second)
	}

	m.mu.RLock()
	count = len(m.gitopsWaitDurations)
	m.mu.RUnlock()

	if count != 100 {
		t.Errorf("Expected max 100 durations, got %d", count)
	}
}

func TestExport(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal:   make(map[string]map[string]int64),
		httpRequestErrors:   make(map[string]int64),
		startTime:           time.Now(),
		workflowDurations:   make([]time.Duration, 0, 100),
		gitopsWaitDurations: make([]time.Duration, 0, 100),
	}

	// Record some test data
	m.RecordHTTPRequest("GET", "/api/specs", 200)
	m.RecordHTTPRequest("POST", "/api/specs", 201)
	m.RecordHTTPRequest("GET", "/api/workflows", 500)

	m.RecordWorkflowExecution(true, 5*time.Second)
	m.RecordWorkflowExecution(false, 3*time.Second)

	m.RecordDBQuery(nil)
	m.RecordDBQuery(errors.New("test error"))

	m.RecordResourceCount("native", 10)
	m.RecordResourceCount("delegated", 5)
	m.RecordResourceCount("external", 3)

	m.RecordExternalResourceHealth(8, 2)
	m.RecordGitOpsWaitDuration(30 * time.Second)

	// Export metrics
	output := m.Export()

	// Verify Prometheus format
	if output == "" {
		t.Fatal("Export() returned empty string")
	}

	// Check for required metric types
	requiredMetrics := []string{
		"innominatus_build_info",
		"innominatus_uptime_seconds",
		"innominatus_http_requests_total",
		"innominatus_http_errors_total",
		"innominatus_workflows_executed_total",
		"innominatus_workflows_succeeded_total",
		"innominatus_workflows_failed_total",
		"innominatus_workflow_duration_seconds_avg",
		"innominatus_db_queries_total",
		"innominatus_db_query_errors_total",
		"innominatus_resources_total",
		"innominatus_resources_external_healthy_total",
		"innominatus_resources_external_failed_total",
		"innominatus_gitops_wait_duration_seconds",
		"innominatus_go_goroutines",
		"innominatus_go_memory_alloc_bytes",
		"innominatus_go_memory_total_alloc_bytes",
		"innominatus_go_memory_sys_bytes",
		"innominatus_go_gc_runs_total",
	}

	for _, metric := range requiredMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Export() missing required metric: %s", metric)
		}
	}

	// Verify HELP and TYPE annotations
	if !strings.Contains(output, "# HELP") {
		t.Error("Export() missing HELP annotations")
	}

	if !strings.Contains(output, "# TYPE") {
		t.Error("Export() missing TYPE annotations")
	}

	// Verify specific values
	if !strings.Contains(output, "innominatus_workflows_executed_total 2") {
		t.Error("Export() incorrect workflow executed count")
	}

	if !strings.Contains(output, "innominatus_workflows_succeeded_total 1") {
		t.Error("Export() incorrect workflow succeeded count")
	}

	if !strings.Contains(output, "innominatus_workflows_failed_total 1") {
		t.Error("Export() incorrect workflow failed count")
	}

	if !strings.Contains(output, "innominatus_db_queries_total 2") {
		t.Error("Export() incorrect DB queries count")
	}

	if !strings.Contains(output, "innominatus_db_query_errors_total 1") {
		t.Error("Export() incorrect DB query errors count")
	}
}

func TestExport_EmptyMetrics(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal:   make(map[string]map[string]int64),
		httpRequestErrors:   make(map[string]int64),
		startTime:           time.Now(),
		workflowDurations:   make([]time.Duration, 0, 100),
		gitopsWaitDurations: make([]time.Duration, 0, 100),
	}

	// Export with no recorded metrics
	output := m.Export()

	// Should still have basic metrics
	if !strings.Contains(output, "innominatus_build_info") {
		t.Error("Export() missing build_info for empty metrics")
	}

	if !strings.Contains(output, "innominatus_uptime_seconds") {
		t.Error("Export() missing uptime for empty metrics")
	}

	// Should have zero values
	if !strings.Contains(output, "innominatus_workflows_executed_total 0") {
		t.Error("Export() should show 0 for workflows when none recorded")
	}
}

func TestConcurrentAccess(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal:   make(map[string]map[string]int64),
		httpRequestErrors:   make(map[string]int64),
		startTime:           time.Now(),
		workflowDurations:   make([]time.Duration, 0, 100),
		gitopsWaitDurations: make([]time.Duration, 0, 100),
	}

	// Test concurrent writes
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				m.RecordHTTPRequest("GET", "/test", 200)
				m.RecordWorkflowExecution(true, time.Second)
				m.RecordDBQuery(nil)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify counts
	m.mu.RLock()
	httpCount := m.httpRequestsTotal["GET"]["/test"]
	workflowCount := m.workflowsExecuted
	dbCount := m.dbQueriesTotal
	m.mu.RUnlock()

	expectedCount := int64(1000) // 10 goroutines * 100 iterations

	if httpCount != expectedCount {
		t.Errorf("Expected HTTP count = %d, got %d", expectedCount, httpCount)
	}

	if workflowCount != expectedCount {
		t.Errorf("Expected workflow count = %d, got %d", expectedCount, workflowCount)
	}

	if dbCount != expectedCount {
		t.Errorf("Expected DB count = %d, got %d", expectedCount, dbCount)
	}
}

func TestConcurrentExport(t *testing.T) {
	m := &Metrics{
		httpRequestsTotal:   make(map[string]map[string]int64),
		httpRequestErrors:   make(map[string]int64),
		startTime:           time.Now(),
		workflowDurations:   make([]time.Duration, 0, 100),
		gitopsWaitDurations: make([]time.Duration, 0, 100),
	}

	// Record some data
	m.RecordHTTPRequest("GET", "/api/specs", 200)
	m.RecordWorkflowExecution(true, 5*time.Second)

	// Test concurrent exports (should use RLock, allowing multiple readers)
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			output := m.Export()
			if output == "" {
				t.Error("Export() returned empty string during concurrent access")
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
