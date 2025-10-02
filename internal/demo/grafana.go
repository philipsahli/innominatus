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
