package demo

// #nosec G204 - Demo/vault components execute commands with controlled parameters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name    string
	Host    string
	Healthy bool
	Status  string
	Error   string
	Latency time.Duration
}

// HealthChecker performs health checks on demo components
type HealthChecker struct {
	client  *http.Client
	timeout time.Duration
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// CheckComponent performs a health check on a single component
func (h *HealthChecker) CheckComponent(component DemoComponent) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:    component.Name,
		Host:    component.IngressHost,
		Healthy: false,
		Status:  "Unknown",
		Latency: 0,
	}

	// Special handling for components that don't use ingress but need custom health checks
	if component.IngressHost == "" {
		if component.Name == "vault-secrets-operator" {
			return h.checkVaultSecretsOperator(component, status)
		}
		status.Status = "No ingress configured"
		status.Healthy = true // Consider it healthy if no ingress is expected
		return status
	}

	// Build health check URL
	url := fmt.Sprintf("http://%s%s", component.IngressHost, component.HealthPath)

	// Create request with context
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create request: %v", err)
		return status
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "OpenAlps-Demo-Health-Checker/1.0")

	// Perform request
	resp, err := h.client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("Request failed: %v", err)
		return status
	}
	defer func() { _ = resp.Body.Close() }()

	status.Latency = time.Since(start)

	// Component-specific health check logic
	switch component.Name {
	case "gitea":
		status = h.checkGitea(resp, status)
	case "argocd":
		status = h.checkArgoCD(resp, status)
	case "vault":
		status = h.checkVault(resp, status)
	case "grafana":
		status = h.checkGrafana(resp, status)
	case "prometheus":
		status = h.checkPrometheus(resp, status)
	case "pushgateway":
		status = h.checkPushgateway(resp, status)
	case "demo-app":
		status = h.checkDemoApp(resp, status)
	case "kubernetes-dashboard":
		status = h.checkKubernetesDashboard(resp, status)
	case "vault-secrets-operator":
		status = h.checkVaultSecretsOperator(component, status)
	case "minio":
		status = h.checkMinio(resp, status)
	case "backstage":
		status = h.checkBackstage(resp, status)
	case "keycloak":
		status = h.checkKeycloak(resp, status)
	default:
		status = h.checkGeneric(resp, status)
	}

	return status
}

// CheckAll performs health checks on all components
func (h *HealthChecker) CheckAll(env *DemoEnvironment) []HealthStatus {
	var results []HealthStatus

	// Check all system components (ingress and non-ingress)
	for _, component := range env.GetSystemComponents() {
		result := h.CheckComponent(component)
		results = append(results, result)
	}

	return results
}

// CheckComponents performs health checks on specific components
func (h *HealthChecker) CheckComponents(components []DemoComponent) []HealthStatus {
	var results []HealthStatus

	// Check only the specified components that have ingress or are system components
	for _, component := range components {
		if component.IngressHost != "" || component.Name == "vault-secrets-operator" {
			result := h.CheckComponent(component)
			results = append(results, result)
		}
	}

	return results
}

// checkGitea performs Gitea-specific health check
func (h *HealthChecker) checkGitea(resp *http.Response, status HealthStatus) HealthStatus {
	switch resp.StatusCode {
	case 200:
		status.Healthy = true
		status.Status = "API Available"
	case 404:
		// Gitea API might not be available, but service could be running
		status.Healthy = true
		status.Status = "Service Available"
	default:
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkArgoCD performs ArgoCD-specific health check
func (h *HealthChecker) checkArgoCD(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Healthy"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkVault performs Vault-specific health check
func (h *HealthChecker) checkVault(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		// Try to parse Vault health response
		var vaultHealth map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&vaultHealth); err == nil {
			if sealed, ok := vaultHealth["sealed"].(bool); ok && !sealed {
				status.Healthy = true
				status.Status = "Unsealed"
			} else {
				status.Status = "Sealed"
			}
		} else {
			status.Healthy = true
			status.Status = "Available"
		}
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkGrafana performs Grafana-specific health check
func (h *HealthChecker) checkGrafana(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		// Try to parse Grafana health response
		var grafanaHealth map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&grafanaHealth); err == nil {
			if database, ok := grafanaHealth["database"].(string); ok && database == "ok" {
				status.Healthy = true
				status.Status = "Database OK"
			} else {
				status.Status = "Database Issues"
			}
		} else {
			status.Healthy = true
			status.Status = "Available"
		}
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkPrometheus performs Prometheus-specific health check
func (h *HealthChecker) checkPrometheus(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Ready"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkPushgateway performs Pushgateway health check
func (h *HealthChecker) checkPushgateway(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Ready"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkDemoApp performs demo app health check
func (h *HealthChecker) checkDemoApp(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Running"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkKubernetesDashboard performs Kubernetes Dashboard health check
func (h *HealthChecker) checkKubernetesDashboard(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Available"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkGeneric performs generic HTTP health check
func (h *HealthChecker) checkGeneric(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		status.Healthy = true
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// GetHealthSummary returns a summary of all health checks
func (h *HealthChecker) GetHealthSummary(results []HealthStatus) (int, int, string) {
	healthy := 0
	total := len(results)

	for _, result := range results {
		if result.Healthy {
			healthy++
		}
	}

	var status string
	//nolint:staticcheck // Simple if-else is more readable for this binary check - QF1003
	if healthy == total {
		status = "All services healthy"
	} else if healthy == 0 {
		status = "All services unhealthy"
	} else {
		status = fmt.Sprintf("%d/%d services healthy", healthy, total)
	}

	return healthy, total, status
}

// checkVaultSecretsOperator performs VSO health check using kubectl
func (h *HealthChecker) checkVaultSecretsOperator(component DemoComponent, status HealthStatus) HealthStatus {
	start := time.Now()

	// Check if VSO pods are running
	cmd := exec.Command("kubectl", "get", "pods", "-n", component.Namespace, "-l", "app.kubernetes.io/name=vault-secrets-operator", "-o", "jsonpath={.items[*].status.phase}") // #nosec G204 - curl command for health check
	output, err := cmd.Output()

	status.Latency = time.Since(start)

	if err != nil {
		status.Error = fmt.Sprintf("Failed to check VSO pods: %v", err)
		status.Status = "Pod check failed"
		return status
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		status.Status = "No pods found"
		return status
	}

	// Check if all pods are running
	phases := strings.Fields(outputStr)
	runningCount := 0
	totalCount := len(phases)

	for _, phase := range phases {
		if phase == "Running" {
			runningCount++
		}
	}

	if runningCount == totalCount && totalCount > 0 {
		status.Healthy = true
		status.Status = fmt.Sprintf("Running (%d/%d pods)", runningCount, totalCount)
	} else {
		status.Status = fmt.Sprintf("Unhealthy (%d/%d pods running)", runningCount, totalCount)
	}

	return status
}

// checkMinio performs Minio-specific health check
func (h *HealthChecker) checkMinio(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Live"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkBackstage performs Backstage-specific health check
func (h *HealthChecker) checkBackstage(resp *http.Response, status HealthStatus) HealthStatus {
	if resp.StatusCode == 200 {
		status.Healthy = true
		status.Status = "Available"
	} else {
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// checkKeycloak performs Keycloak-specific health check
func (h *HealthChecker) checkKeycloak(resp *http.Response, status HealthStatus) HealthStatus {
	// CloudPirates Keycloak chart returns HTTP 302 (redirect to /admin/) when healthy
	switch resp.StatusCode {
	case 302, 200:
		status.Healthy = true
		status.Status = "Running"
	case 503:
		status.Status = "Starting"
	default:
		status.Status = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return status
}

// WaitForHealthy waits for all components to become healthy
func (h *HealthChecker) WaitForHealthy(env *DemoEnvironment, maxRetries int, delay time.Duration) error {
	fmt.Printf("⏳ Waiting for all services to become healthy...\n")

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("   Retry %d/%d...\n", retry+1, maxRetries)
			time.Sleep(delay)
		}

		results := h.CheckAll(env)
		healthy, total, _ := h.GetHealthSummary(results)

		if healthy == total {
			fmt.Printf("✅ All services are healthy!\n")
			return nil
		}

		// Show which services are still unhealthy
		unhealthy := []string{}
		for _, result := range results {
			if !result.Healthy {
				unhealthy = append(unhealthy, result.Name)
			}
		}

		fmt.Printf("   Waiting for: %s\n", strings.Join(unhealthy, ", "))
	}

	return fmt.Errorf("services did not become healthy after %d retries", maxRetries)
}

// WaitForComponentsHealthy waits for specific components to become healthy
func (h *HealthChecker) WaitForComponentsHealthy(components []DemoComponent, maxRetries int, delay time.Duration) error {
	fmt.Printf("⏳ Waiting for %d services to become healthy...\n", len(components))

	for retry := 0; retry < maxRetries; retry++ {
		if retry > 0 {
			fmt.Printf("   Retry %d/%d...\n", retry+1, maxRetries)
			time.Sleep(delay)
		}

		// Check only the specified components
		var results []HealthStatus
		for _, component := range components {
			// Only check components with ingress or specific system components
			if component.IngressHost != "" || component.Name == "vault-secrets-operator" {
				result := h.CheckComponent(component)
				results = append(results, result)
			}
		}

		healthy, total, _ := h.GetHealthSummary(results)

		if healthy == total {
			fmt.Printf("✅ All services are healthy!\n")
			return nil
		}

		// Show which services are still unhealthy
		unhealthy := []string{}
		for _, result := range results {
			if !result.Healthy {
				unhealthy = append(unhealthy, result.Name)
			}
		}

		fmt.Printf("   Waiting for: %s\n", strings.Join(unhealthy, ", "))
	}

	return fmt.Errorf("services did not become healthy after %d retries", maxRetries)
}
