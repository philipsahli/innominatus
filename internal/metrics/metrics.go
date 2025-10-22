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
	workflowsRunning   int64           // Currently running workflows

	// Enhanced workflow step metrics
	workflowStepsTotal      map[string]int64        // step_type -> count
	workflowStepsFailed     map[string]int64        // step_type -> failed count
	workflowStepDurations   map[string][]int64      // step_type -> durations in ms (last 100)
	workflowsByName         map[string]int64        // workflow_name -> execution count
	workflowFailuresByName  map[string]int64        // workflow_name -> failure count

	// Database metrics
	dbQueriesTotal int64
	dbQueryErrors  int64

	// Enhanced resource metrics
	resourcesNative          int64
	resourcesDelegated       int64
	resourcesExternal        int64
	resourcesExternalHealthy int64
	resourcesExternalFailed  int64
	gitopsWaitDurations      []time.Duration // For calculating average GitOps wait time

	// Resource state distribution
	resourcesByState       map[string]int64 // state -> count
	resourcesByType        map[string]int64 // resource_type -> count (postgres, redis, etc.)
	resourceStateTransitions map[string]int64 // "from_state->to_state" -> count
	resourceHealthChecks   int64           // Total health checks performed
	resourceHealthChecksFailed int64       // Failed health checks
	resourceHealthCheckDurations []int64  // Response times in ms (last 100)
}

// Global metrics instance
var global = &Metrics{
	httpRequestsTotal: make(map[string]map[string]int64),
	httpRequestErrors: make(map[string]int64),
	startTime:         time.Now(),
	workflowDurations: make([]time.Duration, 0, 100), // Keep last 100

	// Initialize workflow step metrics
	workflowStepsTotal:     make(map[string]int64),
	workflowStepsFailed:    make(map[string]int64),
	workflowStepDurations:  make(map[string][]int64),
	workflowsByName:        make(map[string]int64),
	workflowFailuresByName: make(map[string]int64),

	// Initialize resource metrics
	resourcesByState:             make(map[string]int64),
	resourcesByType:              make(map[string]int64),
	resourceStateTransitions:     make(map[string]int64),
	resourceHealthCheckDurations: make([]int64, 0, 100),
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

// RecordWorkflowExecutionByName records a workflow execution with workflow name tracking
func (m *Metrics) RecordWorkflowExecutionByName(workflowName string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workflowsExecuted++
	m.workflowsByName[workflowName]++

	if success {
		m.workflowsSucceeded++
	} else {
		m.workflowsFailed++
		m.workflowFailuresByName[workflowName]++
	}

	// Keep last 100 durations for average calculation
	if len(m.workflowDurations) >= 100 {
		m.workflowDurations = m.workflowDurations[1:]
	}
	m.workflowDurations = append(m.workflowDurations, duration)
}

// RecordWorkflowRunning tracks currently running workflows
func (m *Metrics) RecordWorkflowRunning(delta int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workflowsRunning += delta
	if m.workflowsRunning < 0 {
		m.workflowsRunning = 0
	}
}

// RecordWorkflowStep records a workflow step execution
func (m *Metrics) RecordWorkflowStep(stepType string, success bool, durationMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.workflowStepsTotal[stepType]++
	if !success {
		m.workflowStepsFailed[stepType]++
	}

	// Keep last 100 durations per step type
	if m.workflowStepDurations[stepType] == nil {
		m.workflowStepDurations[stepType] = make([]int64, 0, 100)
	}
	durations := m.workflowStepDurations[stepType]
	if len(durations) >= 100 {
		durations = durations[1:]
	}
	m.workflowStepDurations[stepType] = append(durations, durationMs)
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

// RecordResourceCount records resource counts by type
func (m *Metrics) RecordResourceCount(resourceType string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch resourceType {
	case "native":
		m.resourcesNative = count
	case "delegated":
		m.resourcesDelegated = count
	case "external":
		m.resourcesExternal = count
	}
}

// RecordExternalResourceHealth records health status of external resources
func (m *Metrics) RecordExternalResourceHealth(healthy, failed int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.resourcesExternalHealthy = healthy
	m.resourcesExternalFailed = failed
}

// RecordGitOpsWaitDuration records the duration waited for GitOps operations
func (m *Metrics) RecordGitOpsWaitDuration(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Keep last 100 durations for average calculation
	if len(m.gitopsWaitDurations) >= 100 {
		m.gitopsWaitDurations = m.gitopsWaitDurations[1:]
	}
	m.gitopsWaitDurations = append(m.gitopsWaitDurations, duration)
}

// RecordResourceByState records resource count by lifecycle state
func (m *Metrics) RecordResourceByState(state string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.resourcesByState[state] = count
}

// RecordResourceByType records resource count by resource type
func (m *Metrics) RecordResourceByType(resourceType string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.resourcesByType[resourceType] = count
}

// RecordResourceStateTransition records a state transition
func (m *Metrics) RecordResourceStateTransition(fromState, toState string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s->%s", fromState, toState)
	m.resourceStateTransitions[key]++
}

// RecordResourceHealthCheck records a resource health check
func (m *Metrics) RecordResourceHealthCheck(success bool, responseTimeMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.resourceHealthChecks++
	if !success {
		m.resourceHealthChecksFailed++
	}

	// Keep last 100 response times
	if len(m.resourceHealthCheckDurations) >= 100 {
		m.resourceHealthCheckDurations = m.resourceHealthCheckDurations[1:]
	}
	m.resourceHealthCheckDurations = append(m.resourceHealthCheckDurations, responseTimeMs)
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

	// Currently running workflows
	output += "# HELP innominatus_workflows_running Currently running workflows\n"
	output += "# TYPE innominatus_workflows_running gauge\n"
	output += fmt.Sprintf("innominatus_workflows_running %d\n", m.workflowsRunning)
	output += "\n"

	// Workflow executions by name
	if len(m.workflowsByName) > 0 {
		output += "# HELP innominatus_workflows_by_name_total Total workflow executions by workflow name\n"
		output += "# TYPE innominatus_workflows_by_name_total counter\n"
		for name, count := range m.workflowsByName {
			output += fmt.Sprintf("innominatus_workflows_by_name_total{workflow_name=\"%s\"} %d\n", name, count)
		}
		output += "\n"
	}

	// Workflow failures by name
	if len(m.workflowFailuresByName) > 0 {
		output += "# HELP innominatus_workflows_failures_by_name_total Total workflow failures by workflow name\n"
		output += "# TYPE innominatus_workflows_failures_by_name_total counter\n"
		for name, count := range m.workflowFailuresByName {
			output += fmt.Sprintf("innominatus_workflows_failures_by_name_total{workflow_name=\"%s\"} %d\n", name, count)
		}
		output += "\n"
	}

	// Workflow steps by type
	if len(m.workflowStepsTotal) > 0 {
		output += "# HELP innominatus_workflow_steps_total Total workflow steps executed by type\n"
		output += "# TYPE innominatus_workflow_steps_total counter\n"
		for stepType, count := range m.workflowStepsTotal {
			output += fmt.Sprintf("innominatus_workflow_steps_total{step_type=\"%s\"} %d\n", stepType, count)
		}
		output += "\n"
	}

	// Workflow step failures by type
	if len(m.workflowStepsFailed) > 0 {
		output += "# HELP innominatus_workflow_steps_failed_total Total failed workflow steps by type\n"
		output += "# TYPE innominatus_workflow_steps_failed_total counter\n"
		for stepType, count := range m.workflowStepsFailed {
			output += fmt.Sprintf("innominatus_workflow_steps_failed_total{step_type=\"%s\"} %d\n", stepType, count)
		}
		output += "\n"
	}

	// Average step duration by type
	if len(m.workflowStepDurations) > 0 {
		output += "# HELP innominatus_workflow_step_duration_seconds_avg Average step duration by type (last 100 executions)\n"
		output += "# TYPE innominatus_workflow_step_duration_seconds_avg gauge\n"
		for stepType, durations := range m.workflowStepDurations {
			if len(durations) > 0 {
				var total int64
				for _, d := range durations {
					total += d
				}
				avgSeconds := float64(total) / float64(len(durations)) / 1000.0
				output += fmt.Sprintf("innominatus_workflow_step_duration_seconds_avg{step_type=\"%s\"} %.2f\n", stepType, avgSeconds)
			}
		}
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

	// Resource metrics
	output += "# HELP innominatus_resources_total Total resources by type\n"
	output += "# TYPE innominatus_resources_total gauge\n"
	output += fmt.Sprintf("innominatus_resources_total{type=\"native\"} %d\n", m.resourcesNative)
	output += fmt.Sprintf("innominatus_resources_total{type=\"delegated\"} %d\n", m.resourcesDelegated)
	output += fmt.Sprintf("innominatus_resources_total{type=\"external\"} %d\n", m.resourcesExternal)
	output += "\n"

	output += "# HELP innominatus_resources_external_healthy_total Total healthy external resources\n"
	output += "# TYPE innominatus_resources_external_healthy_total gauge\n"
	output += fmt.Sprintf("innominatus_resources_external_healthy_total %d\n", m.resourcesExternalHealthy)
	output += "\n"

	output += "# HELP innominatus_resources_external_failed_total Total failed external resources\n"
	output += "# TYPE innominatus_resources_external_failed_total gauge\n"
	output += fmt.Sprintf("innominatus_resources_external_failed_total %d\n", m.resourcesExternalFailed)
	output += "\n"

	// GitOps wait duration
	if len(m.gitopsWaitDurations) > 0 {
		var total time.Duration
		for _, d := range m.gitopsWaitDurations {
			total += d
		}
		avgSeconds := (total / time.Duration(len(m.gitopsWaitDurations))).Seconds()
		output += "# HELP innominatus_gitops_wait_duration_seconds Average GitOps wait duration (last 100 operations)\n"
		output += "# TYPE innominatus_gitops_wait_duration_seconds gauge\n"
		output += fmt.Sprintf("innominatus_gitops_wait_duration_seconds %.2f\n", avgSeconds)
		output += "\n"
	}

	// Resources by lifecycle state
	if len(m.resourcesByState) > 0 {
		output += "# HELP innominatus_resources_by_state Resources count by lifecycle state\n"
		output += "# TYPE innominatus_resources_by_state gauge\n"
		for state, count := range m.resourcesByState {
			output += fmt.Sprintf("innominatus_resources_by_state{state=\"%s\"} %d\n", state, count)
		}
		output += "\n"
	}

	// Resources by resource type
	if len(m.resourcesByType) > 0 {
		output += "# HELP innominatus_resources_by_type Resources count by resource type\n"
		output += "# TYPE innominatus_resources_by_type gauge\n"
		for resourceType, count := range m.resourcesByType {
			output += fmt.Sprintf("innominatus_resources_by_type{resource_type=\"%s\"} %d\n", resourceType, count)
		}
		output += "\n"
	}

	// Resource state transitions
	if len(m.resourceStateTransitions) > 0 {
		output += "# HELP innominatus_resource_state_transitions_total Total resource state transitions\n"
		output += "# TYPE innominatus_resource_state_transitions_total counter\n"
		for transition, count := range m.resourceStateTransitions {
			// Parse transition like "requested->provisioning"
			output += fmt.Sprintf("innominatus_resource_state_transitions_total{transition=\"%s\"} %d\n", transition, count)
		}
		output += "\n"
	}

	// Resource health checks
	output += "# HELP innominatus_resource_health_checks_total Total resource health checks performed\n"
	output += "# TYPE innominatus_resource_health_checks_total counter\n"
	output += fmt.Sprintf("innominatus_resource_health_checks_total %d\n", m.resourceHealthChecks)
	output += "\n"

	output += "# HELP innominatus_resource_health_checks_failed_total Total failed resource health checks\n"
	output += "# TYPE innominatus_resource_health_checks_failed_total counter\n"
	output += fmt.Sprintf("innominatus_resource_health_checks_failed_total %d\n", m.resourceHealthChecksFailed)
	output += "\n"

	// Average health check response time
	if len(m.resourceHealthCheckDurations) > 0 {
		var total int64
		for _, d := range m.resourceHealthCheckDurations {
			total += d
		}
		avgMs := float64(total) / float64(len(m.resourceHealthCheckDurations))
		output += "# HELP innominatus_resource_health_check_duration_ms_avg Average health check response time in milliseconds (last 100 checks)\n"
		output += "# TYPE innominatus_resource_health_check_duration_ms_avg gauge\n"
		output += fmt.Sprintf("innominatus_resource_health_check_duration_ms_avg %.2f\n", avgMs)
		output += "\n"
	}

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
