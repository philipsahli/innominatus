package vault
// #nosec G204 - Demo/vault components execute commands with controlled parameters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client represents a Vault API client
type Client struct {
	address string
	token   string
	client  *http.Client
}

// NewClient creates a new Vault client
func NewClient(address, token string) *Client {
	return &Client{
		address: strings.TrimSuffix(address, "/"),
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateNamespace creates a new namespace in Vault
func (c *Client) CreateNamespace(namespace string) error {
	fmt.Printf("🔐 Creating Vault namespace: %s\n", namespace)

	// Vault namespace creation (Enterprise feature - simulate for dev mode)
	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating namespace creation: %s\n", namespace)
		return nil
	}

	// In production, would use: sys/namespaces/{namespace}
	path := fmt.Sprintf("/v1/sys/namespaces/%s", namespace)
	data := map[string]interface{}{
		"path": namespace,
	}

	return c.makeRequest("POST", path, data, nil)
}

// CreateKubernetesAuthMethod creates a Kubernetes auth method for the namespace
func (c *Client) CreateKubernetesAuthMethod(namespace, mountPath string) error {
	fmt.Printf("🔐 Creating Kubernetes auth method for namespace: %s\n", namespace)

	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating auth method creation for: %s\n", namespace)
		return nil
	}

	// Enable auth method
	enablePath := fmt.Sprintf("/v1/sys/auth/%s", mountPath)
	enableData := map[string]interface{}{
		"type": "kubernetes",
	}

	if err := c.makeRequest("POST", enablePath, enableData, nil); err != nil {
		return fmt.Errorf("failed to enable kubernetes auth: %w", err)
	}

	// Configure auth method
	configPath := fmt.Sprintf("/v1/auth/%s/config", mountPath)
	configData := map[string]interface{}{
		"kubernetes_host":     "https://kubernetes.default.svc",
		"kubernetes_ca_cert":  "", // Will be auto-detected in cluster
		"token_reviewer_jwt":  "", // Will use service account token
	}

	return c.makeRequest("POST", configPath, configData, nil)
}

// CreatePolicy creates a policy for application access
func (c *Client) CreatePolicy(policyName, namespace string, capabilities []string) error {
	fmt.Printf("🔐 Creating Vault policy: %s for namespace: %s\n", policyName, namespace)

	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating policy creation: %s\n", policyName)
		return nil
	}

	// Create policy allowing access to application secrets
	policy := fmt.Sprintf(`
path "secret/data/applications/%s/*" {
  capabilities = [%s]
}

path "secret/metadata/applications/%s/*" {
  capabilities = ["list", "read"]
}
`, namespace, strings.Join(capabilities, ", "), namespace)

	path := fmt.Sprintf("/v1/sys/policies/acl/%s", policyName)
	data := map[string]interface{}{
		"policy": policy,
	}

	return c.makeRequest("PUT", path, data, nil)
}

// CreateKubernetesRole creates a Kubernetes role binding
func (c *Client) CreateKubernetesRole(mountPath, roleName, namespace, serviceAccount string, policies []string) error {
	fmt.Printf("🔐 Creating Kubernetes role: %s for SA: %s in namespace: %s\n", roleName, serviceAccount, namespace)

	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating role creation: %s\n", roleName)
		return nil
	}

	path := fmt.Sprintf("/v1/auth/%s/role/%s", mountPath, roleName)
	data := map[string]interface{}{
		"bound_service_account_names":      []string{serviceAccount},
		"bound_service_account_namespaces": []string{namespace},
		"policies":                         policies,
		"ttl":                             "1h",
		"max_ttl":                         "24h",
	}

	return c.makeRequest("POST", path, data, nil)
}

// CreateSecret creates a secret in the application's Vault space
func (c *Client) CreateSecret(appNamespace, secretName string, secretData map[string]interface{}) error {
	fmt.Printf("🔐 Creating secret: %s in app namespace: %s\n", secretName, appNamespace)

	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating secret creation: %s\n", secretName)
		return nil
	}

	path := fmt.Sprintf("/v1/secret/data/applications/%s/%s", appNamespace, secretName)
	data := map[string]interface{}{
		"data": secretData,
	}

	return c.makeRequest("POST", path, data, nil)
}

// GetSecret retrieves a secret from the application's Vault space
func (c *Client) GetSecret(appNamespace, secretName string) (map[string]interface{}, error) {
	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating secret retrieval: %s\n", secretName)
		return map[string]interface{}{
			"demo-key": "demo-value",
		}, nil
	}

	path := fmt.Sprintf("/v1/secret/data/applications/%s/%s", appNamespace, secretName)
	var result map[string]interface{}

	if err := c.makeRequest("GET", path, nil, &result); err != nil {
		return nil, err
	}

	// Extract data from Vault response structure
	if data, ok := result["data"].(map[string]interface{}); ok {
		if secretData, ok := data["data"].(map[string]interface{}); ok {
			return secretData, nil
		}
	}

	return nil, fmt.Errorf("invalid secret response format")
}

// DeleteSecret deletes a secret from the application's Vault space
func (c *Client) DeleteSecret(appNamespace, secretName string) error {
	fmt.Printf("🔐 Deleting secret: %s from app namespace: %s\n", secretName, appNamespace)

	if c.isDevMode() {
		fmt.Printf("   [DEV MODE] Simulating secret deletion: %s\n", secretName)
		return nil
	}

	path := fmt.Sprintf("/v1/secret/metadata/applications/%s/%s", appNamespace, secretName)
	return c.makeRequest("DELETE", path, nil, nil)
}

// HealthCheck performs a health check against Vault
func (c *Client) HealthCheck() error {
	var result map[string]interface{}
	return c.makeRequest("GET", "/v1/sys/health", nil, &result)
}

// makeRequest performs an HTTP request to Vault
func (c *Client) makeRequest(method, path string, data interface{}, result interface{}) error {
	url := c.address + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("vault request failed with status %d: %s", resp.StatusCode, string(body))
	}

	if result != nil && resp.ContentLength != 0 {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// SetupKubernetesAuthMethod configures Vault's Kubernetes auth method for VSO
func (c *Client) SetupKubernetesAuthMethod() error {
	fmt.Printf("🔐 Setting up Kubernetes auth method in Vault\n")

	// Enable Kubernetes auth method if not already enabled
	enableData := map[string]interface{}{"type": "kubernetes"}
	err := c.makeRequest("POST", "/v1/sys/auth/kubernetes", enableData, nil)
	if err != nil && !strings.Contains(err.Error(), "path is already in use") {
		return fmt.Errorf("failed to enable kubernetes auth: %w", err)
	}

	// Configure the auth method
	configData := map[string]interface{}{
		"kubernetes_host": "https://kubernetes.default.svc",
	}
	err = c.makeRequest("POST", "/v1/auth/kubernetes/config", configData, nil)
	if err != nil {
		return fmt.Errorf("failed to configure kubernetes auth: %w", err)
	}

	fmt.Printf("✅ Kubernetes auth method configured\n")
	return nil
}

// SetupVaultPolicy creates a policy for accessing application secrets
func (c *Client) SetupVaultPolicy(appName string) error {
	fmt.Printf("🔐 Creating Vault policy for app: %s\n", appName)

	policyName := fmt.Sprintf("%s-policy", appName)
	policy := fmt.Sprintf(`
path "secret/data/applications/%s/*" {
  capabilities = ["read"]
}
path "secret/metadata/applications/%s/*" {
  capabilities = ["read"]
}`, appName, appName)

	policyData := map[string]interface{}{"policy": policy}
	err := c.makeRequest("PUT", fmt.Sprintf("/v1/sys/policies/acl/%s", policyName), policyData, nil)
	if err != nil {
		return fmt.Errorf("failed to create policy %s: %w", policyName, err)
	}

	fmt.Printf("✅ Policy %s created\n", policyName)
	return nil
}

// SetupKubernetesRole creates a Kubernetes role for the service account
func (c *Client) SetupKubernetesRole(appName, namespace, serviceAccount string) error {
	fmt.Printf("🔐 Creating Kubernetes role for app: %s\n", appName)

	roleData := map[string]interface{}{
		"bound_service_account_names":      []string{serviceAccount},
		"bound_service_account_namespaces": []string{namespace},
		"policies":                         []string{fmt.Sprintf("%s-policy", appName)},
		"ttl":                             "1h",
	}

	err := c.makeRequest("POST", "/v1/auth/kubernetes/role/vault-secrets-operator", roleData, nil)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes role: %w", err)
	}

	fmt.Printf("✅ Kubernetes role created for %s\n", appName)
	return nil
}

// isDevMode checks if Vault is running in development mode
func (c *Client) isDevMode() bool {
	// For demo environment, we want to actually interact with Vault even with root token
	// In production this would check for more sophisticated dev mode indicators
	return false
}