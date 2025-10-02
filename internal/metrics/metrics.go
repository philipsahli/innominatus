package metrics

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// Metrics holds application metrics
type Metrics struct {
	mu                sync.RWMutex
	httpRequestsTotal map[string]map[string]int64 // method -> path -> count
	httpRequestErrors map[string]int64            // path -> error count
	startTime         time.Time

	// Workflow metrics
	workflowsExecuted  int64
	workflowsSucceeded int64
	workflowsFailed    int64
	workflowDurations  []time.Duration // For calculating average

	// Database metrics
	dbQueriesTotal int64
	dbQueryErrors  int64
}

// Global metrics instance
var global = &Metrics{
	httpRequestsTotal: make(map[string]map[string]int64),
	httpRequestErrors: make(map[string]int64),
	startTime:         time.Now(),
	workflowDurations: make([]time.Duration, 0, 100), // Keep last 100
}

// GetGlobal returns the global metrics instance
func GetGlobal() *Metrics {
	return global
}

// RecordHTTPRequest records an HTTP request
func (m *Metrics) RecordHTTPRequest(method, path string, statusCode int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.httpRequestsTotal[method] == nil {
		m.httpRequestsTotal[method] = make(map[string]int64)
	}
	m.httpRequestsTotal[method][path]++

	// Record errors (5xx status codes)
	if statusCode >= 500 {
		m.httpRequestErrors[path]++
	}
}

// RecordWorkflowExecution records a workflow execution
func (m *Metrics) RecordWorkflowExecution(success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workflowsExecuted++
	if success {
		m.workflowsSucceeded++
	} else {
		m.workflowsFailed++
	}

	// Keep last 100 durations for average calculation
	if len(m.workflowDurations) >= 100 {
		m.workflowDurations = m.workflowDurations[1:]
	}
	m.workflowDurations = append(m.workflowDurations, duration)
}

// RecordDBQuery records a database query
func (m *Metrics) RecordDBQuery(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dbQueriesTotal++
	if err != nil {
		m.dbQueryErrors++
	}
}

// Export exports metrics in Prometheus format
func (m *Metrics) Export() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var output string

	// Process info
	output += "# HELP innominatus_build_info Build information\n"
	output += "# TYPE innominatus_build_info gauge\n"
	output += "innominatus_build_info{version=\"1.0.0\",go_version=\"" + runtime.Version() + "\"} 1\n"
	output += "\n"

	// Uptime
	uptime := time.Since(m.startTime).Seconds()
	output += "# HELP innominatus_uptime_seconds Server uptime in seconds\n"
	output += "# TYPE innominatus_uptime_seconds gauge\n"
	output += fmt.Sprintf("innominatus_uptime_seconds %.2f\n", uptime)
	output += "\n"

	// HTTP requests
	output += "# HELP innominatus_http_requests_total Total HTTP requests\n"
	output += "# TYPE innominatus_http_requests_total counter\n"
	for method, paths := range m.httpRequestsTotal {
		for path, count := range paths {
			output += fmt.Sprintf("innominatus_http_requests_total{method=\"%s\",path=\"%s\"} %d\n", method, path, count)
		}
	}
	output += "\n"

	// HTTP errors
	output += "# HELP innominatus_http_errors_total Total HTTP 5xx errors\n"
	output += "# TYPE innominatus_http_errors_total counter\n"
	for path, count := range m.httpRequestErrors {
		if count > 0 {
			output += fmt.Sprintf("innominatus_http_errors_total{path=\"%s\"} %d\n", path, count)
		}
	}
	output += "\n"

	// Workflow metrics
	output += "# HELP innominatus_workflows_executed_total Total workflow executions\n"
	output += "# TYPE innominatus_workflows_executed_total counter\n"
	output += fmt.Sprintf("innominatus_workflows_executed_total %d\n", m.workflowsExecuted)
	output += "\n"

	output += "# HELP innominatus_workflows_succeeded_total Total successful workflow executions\n"
	output += "# TYPE innominatus_workflows_succeeded_total counter\n"
	output += fmt.Sprintf("innominatus_workflows_succeeded_total %d\n", m.workflowsSucceeded)
	output += "\n"

	output += "# HELP innominatus_workflows_failed_total Total failed workflow executions\n"
	output += "# TYPE innominatus_workflows_failed_total counter\n"
	output += fmt.Sprintf("innominatus_workflows_failed_total %d\n", m.workflowsFailed)
	output += "\n"

	// Average workflow duration
	if len(m.workflowDurations) > 0 {
		var total time.Duration
		for _, d := range m.workflowDurations {
			total += d
		}
		avgSeconds := (total / time.Duration(len(m.workflowDurations))).Seconds()
		output += "# HELP innominatus_workflow_duration_seconds_avg Average workflow duration (last 100 executions)\n"
		output += "# TYPE innominatus_workflow_duration_seconds_avg gauge\n"
		output += fmt.Sprintf("innominatus_workflow_duration_seconds_avg %.2f\n", avgSeconds)
		output += "\n"
	}

	// Database metrics
	output += "# HELP innominatus_db_queries_total Total database queries\n"
	output += "# TYPE innominatus_db_queries_total counter\n"
	output += fmt.Sprintf("innominatus_db_queries_total %d\n", m.dbQueriesTotal)
	output += "\n"

	output += "# HELP innominatus_db_query_errors_total Total database query errors\n"
	output += "# TYPE innominatus_db_query_errors_total counter\n"
	output += fmt.Sprintf("innominatus_db_query_errors_total %d\n", m.dbQueryErrors)
	output += "\n"

	// Go runtime metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	output += "# HELP innominatus_go_goroutines Number of goroutines\n"
	output += "# TYPE innominatus_go_goroutines gauge\n"
	output += fmt.Sprintf("innominatus_go_goroutines %d\n", runtime.NumGoroutine())
	output += "\n"

	output += "# HELP innominatus_go_memory_alloc_bytes Bytes allocated and in use\n"
	output += "# TYPE innominatus_go_memory_alloc_bytes gauge\n"
	output += fmt.Sprintf("innominatus_go_memory_alloc_bytes %d\n", memStats.Alloc)
	output += "\n"

	output += "# HELP innominatus_go_memory_total_alloc_bytes Total bytes allocated (cumulative)\n"
	output += "# TYPE innominatus_go_memory_total_alloc_bytes counter\n"
	output += fmt.Sprintf("innominatus_go_memory_total_alloc_bytes %d\n", memStats.TotalAlloc)
	output += "\n"

	output += "# HELP innominatus_go_memory_sys_bytes Total memory obtained from OS\n"
	output += "# TYPE innominatus_go_memory_sys_bytes gauge\n"
	output += fmt.Sprintf("innominatus_go_memory_sys_bytes %d\n", memStats.Sys)
	output += "\n"

	output += "# HELP innominatus_go_gc_runs_total Total number of GC runs\n"
	output += "# TYPE innominatus_go_gc_runs_total counter\n"
	output += fmt.Sprintf("innominatus_go_gc_runs_total %d\n", memStats.NumGC)
	output += "\n"

	return output
}
