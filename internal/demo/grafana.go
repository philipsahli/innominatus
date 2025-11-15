package demo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GrafanaManager handles Grafana configuration and dashboard management
type GrafanaManager struct {
	url      string
	username string
	password string
	client   *http.Client
}

// NewGrafanaManager creates a new Grafana manager
func NewGrafanaManager(url, username, password string) *GrafanaManager {
	return &GrafanaManager{
		url:      url,
		username: username,
		password: password,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WaitForGrafana waits for Grafana to be ready
func (g *GrafanaManager) WaitForGrafana(maxRetries int) error {
	fmt.Printf("⏳ Waiting for Grafana to be ready...\n")

	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(10 * time.Second)
		}

		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/health", g.url), nil)
		if err != nil {
			continue
		}

		req.SetBasicAuth(g.username, g.password)
		resp, err := g.client.Do(req)
		if err != nil {
			fmt.Printf("   Retry %d/%d...\\n", i+1, maxRetries)
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode == 200 {
			fmt.Printf("✅ Grafana is ready\\n")
			return nil
		}

		fmt.Printf("   Retry %d/%d...\\n", i+1, maxRetries)
	}

	return fmt.Errorf("grafana did not become ready within timeout")
}

// InstallClusterHealthDashboard installs a cluster health dashboard
func (g *GrafanaManager) InstallClusterHealthDashboard() error {
	fmt.Printf("📊 Installing Cluster Health Dashboard in Grafana...\\n")

	// Wait for Grafana to be ready first
	if err := g.WaitForGrafana(20); err != nil {
		return err
	}

	// Cluster Health Dashboard JSON
	dashboard := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"id":       nil,
			"title":    "Kubernetes Cluster Health",
			"tags":     []string{"kubernetes", "cluster", "health"},
			"timezone": "browser",
			"panels": []map[string]interface{}{
				{
					"id":    1,
					"title": "Cluster CPU Usage",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "100 - (avg(irate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
							"legendFormat": "CPU Usage %",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 6,
						"x": 0,
						"y": 0,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				{
					"id":    2,
					"title": "Cluster Memory Usage",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100",
							"legendFormat": "Memory Usage %",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 6,
						"x": 6,
						"y": 0,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				{
					"id":    3,
					"title": "Total Pods",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "sum(kube_pod_info)",
							"legendFormat": "Total Pods",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 6,
						"x": 12,
						"y": 0,
					},
				},
				{
					"id":    4,
					"title": "Node Count",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "count(kube_node_info)",
							"legendFormat": "Nodes",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 6,
						"x": 18,
						"y": 0,
					},
				},
				{
					"id":    5,
					"title": "Pods by Namespace",
					"type":  "piechart",
					"targets": []map[string]interface{}{
						{
							"expr":         "count by (namespace) (kube_pod_info)",
							"legendFormat": "{{namespace}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 0,
						"y": 8,
					},
				},
				{
					"id":    6,
					"title": "Pod Status",
					"type":  "piechart",
					"targets": []map[string]interface{}{
						{
							"expr":         "count by (phase) (kube_pod_status_phase)",
							"legendFormat": "{{phase}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 12,
						"y": 8,
					},
				},
			},
			"time": map[string]interface{}{
				"from": "now-1h",
				"to":   "now",
			},
			"refresh": "30s",
		},
		"overwrite": true,
	}

	// Convert to JSON
	dashboardJSON, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard JSON: %v", err)
	}

	// Create request to Grafana API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/dashboards/db", g.url), bytes.NewBuffer(dashboardJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(g.username, g.password)

	// Send request
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send dashboard request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to install dashboard, status: %d", resp.StatusCode)
	}

	fmt.Printf("✅ Cluster Health Dashboard installed in Grafana\\n")
	return nil
}

// InstallInnominatusDashboard installs an Innominatus platform metrics dashboard
func (g *GrafanaManager) InstallInnominatusDashboard() error {
	fmt.Printf("📊 Installing Innominatus Platform Metrics Dashboard in Grafana...\\n")

	// Wait for Grafana to be ready first
	if err := g.WaitForGrafana(20); err != nil {
		return err
	}

	// Innominatus Platform Metrics Dashboard JSON
	dashboard := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"id":       nil,
			"title":    "Innominatus Platform Metrics",
			"tags":     []string{"innominatus", "platform", "workflows"},
			"timezone": "browser",
			"panels": []map[string]interface{}{
				// Row 1: Overview
				{
					"id":    1,
					"title": "Server Uptime",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_uptime_seconds",
							"legendFormat": "Uptime",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 0,
						"y": 0,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "s",
						},
					},
				},
				{
					"id":    2,
					"title": "Total Workflows",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflows_executed_total",
							"legendFormat": "Total",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 6,
						"y": 0,
					},
				},
				{
					"id":    3,
					"title": "Workflow Success Rate",
					"type":  "gauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "(innominatus_workflows_succeeded_total / innominatus_workflows_executed_total) * 100",
							"legendFormat": "Success Rate",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 12,
						"y": 0,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				{
					"id":    4,
					"title": "HTTP Requests",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_http_requests_total",
							"legendFormat": "Total Requests",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 18,
						"y": 0,
					},
				},
				// Row 2: Workflows
				{
					"id":    5,
					"title": "Workflow Executions",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflows_executed_total",
							"legendFormat": "Total",
						},
						{
							"expr":         "innominatus_workflows_succeeded_total",
							"legendFormat": "Succeeded",
						},
						{
							"expr":         "innominatus_workflows_failed_total",
							"legendFormat": "Failed",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 0,
						"y": 6,
					},
				},
				{
					"id":    6,
					"title": "Average Workflow Duration",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflow_duration_seconds_avg",
							"legendFormat": "Avg Duration",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 12,
						"y": 6,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "s",
						},
					},
				},
				// Row 3: Database & HTTP
				{
					"id":    7,
					"title": "Database Queries",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_db_queries_total",
							"legendFormat": "Total Queries",
						},
						{
							"expr":         "innominatus_db_query_errors_total",
							"legendFormat": "Errors",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 0,
						"y": 14,
					},
				},
				{
					"id":    8,
					"title": "HTTP Requests & Errors",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_http_requests_total",
							"legendFormat": "Requests",
						},
						{
							"expr":         "innominatus_http_errors_total",
							"legendFormat": "Errors (5xx)",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 12,
						"y": 14,
					},
				},
				// Row 4: Runtime & Performance (Go metrics)
				{
					"id":    9,
					"title": "Go Goroutines",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "go_goroutines",
							"legendFormat": "Goroutines",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 0,
						"y": 22,
					},
				},
				{
					"id":    10,
					"title": "Memory Allocated",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "go_memstats_alloc_bytes",
							"legendFormat": "Allocated",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 6,
						"y": 22,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "bytes",
						},
					},
				},
				{
					"id":    11,
					"title": "Process Memory (Resident)",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "process_resident_memory_bytes",
							"legendFormat": "Resident Memory",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 12,
						"y": 22,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "bytes",
						},
					},
				},
				{
					"id":    12,
					"title": "Build Info",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_build_info",
							"legendFormat": "{{version}} ({{commit}})",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 18,
						"y": 22,
					},
				},
				{
					"id":    13,
					"title": "GC Duration Rate",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "rate(go_gc_duration_seconds_sum[5m])",
							"legendFormat": "GC Rate",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 0,
						"y": 28,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "s",
						},
					},
				},
				{
					"id":    14,
					"title": "CPU Usage Rate",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "rate(process_cpu_seconds_total[1m])",
							"legendFormat": "CPU Rate",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 12,
						"x": 12,
						"y": 28,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percentunit",
						},
					},
				},
				// Row 5: Enhanced Workflow Metrics
				{
					"id":    15,
					"title": "Currently Running Workflows",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflows_running",
							"legendFormat": "Running",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 0,
						"y": 36,
					},
				},
				{
					"id":    16,
					"title": "Workflow Executions by Name",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflows_by_name_total",
							"legendFormat": "{{workflow_name}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 9,
						"x": 6,
						"y": 36,
					},
				},
				{
					"id":    17,
					"title": "Workflow Failure Rate by Name",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "(innominatus_workflows_failures_by_name_total / innominatus_workflows_by_name_total) * 100",
							"legendFormat": "{{workflow_name}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 9,
						"x": 15,
						"y": 36,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				// Row 6: Workflow Step Metrics
				{
					"id":    18,
					"title": "Workflow Steps by Type",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflow_steps_total",
							"legendFormat": "{{step_type}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 0,
						"y": 44,
					},
				},
				{
					"id":    19,
					"title": "Step Failure Rate by Type",
					"type":  "bargauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "(innominatus_workflow_steps_failed_total / innominatus_workflow_steps_total) * 100",
							"legendFormat": "{{step_type}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 8,
						"y": 44,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				{
					"id":    20,
					"title": "Average Step Duration by Type",
					"type":  "bargauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_workflow_step_duration_seconds_avg",
							"legendFormat": "{{step_type}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 16,
						"y": 44,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "s",
						},
					},
				},
				// Row 7: Resource Metrics
				{
					"id":    21,
					"title": "Resources by Lifecycle State",
					"type":  "piechart",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_resources_by_state",
							"legendFormat": "{{state}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 0,
						"y": 52,
					},
				},
				{
					"id":    22,
					"title": "Resources by Type",
					"type":  "bargauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_resources_by_type",
							"legendFormat": "{{resource_type}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 8,
						"y": 52,
					},
				},
				{
					"id":    23,
					"title": "Resource State Transitions",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_resource_state_transitions_total",
							"legendFormat": "{{transition}}",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 8,
						"w": 8,
						"x": 16,
						"y": 52,
					},
				},
				// Row 8: Resource Health Metrics
				{
					"id":    24,
					"title": "Resource Health Checks",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_resource_health_checks_total",
							"legendFormat": "Total Checks",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 0,
						"y": 60,
					},
				},
				{
					"id":    25,
					"title": "Health Check Success Rate",
					"type":  "gauge",
					"targets": []map[string]interface{}{
						{
							"expr":         "((innominatus_resource_health_checks_total - innominatus_resource_health_checks_failed_total) / innominatus_resource_health_checks_total) * 100",
							"legendFormat": "Success Rate",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 6,
						"y": 60,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "percent",
							"min":  0,
							"max":  100,
						},
					},
				},
				{
					"id":    26,
					"title": "Average Health Check Response Time",
					"type":  "stat",
					"targets": []map[string]interface{}{
						{
							"expr":         "innominatus_resource_health_check_duration_ms_avg",
							"legendFormat": "Avg Response Time",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 12,
						"y": 60,
					},
					"fieldConfig": map[string]interface{}{
						"defaults": map[string]interface{}{
							"unit": "ms",
						},
					},
				},
				{
					"id":    27,
					"title": "Health Check Trends",
					"type":  "timeseries",
					"targets": []map[string]interface{}{
						{
							"expr":         "rate(innominatus_resource_health_checks_total[5m])",
							"legendFormat": "Check Rate",
						},
						{
							"expr":         "rate(innominatus_resource_health_checks_failed_total[5m])",
							"legendFormat": "Failure Rate",
						},
					},
					"gridPos": map[string]interface{}{
						"h": 6,
						"w": 6,
						"x": 18,
						"y": 60,
					},
				},
			},
			"time": map[string]interface{}{
				"from": "now-1h",
				"to":   "now",
			},
			"refresh": "15s",
		},
		"overwrite": true,
	}

	// Convert to JSON
	dashboardJSON, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard JSON: %v", err)
	}

	// Create request to Grafana API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/dashboards/db", g.url), bytes.NewBuffer(dashboardJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(g.username, g.password)

	// Send request
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send dashboard request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to install dashboard, status: %d", resp.StatusCode)
	}

	fmt.Printf("✅ Innominatus Platform Metrics Dashboard installed in Grafana\\n")
	return nil
}
