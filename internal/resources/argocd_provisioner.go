package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"innominatus/internal/security"
	"io"
	"net/http"
	"time"
)

// ArgoCDProvisioner handles ArgoCD Application creation
type ArgoCDProvisioner struct {
	repo *database.ResourceRepository
}

// NewArgoCDProvisioner creates a new ArgoCD provisioner
func NewArgoCDProvisioner(repo *database.ResourceRepository) *ArgoCDProvisioner {
	return &ArgoCDProvisioner{
		repo: repo,
	}
}

// Provision creates an ArgoCD Application
func (ap *ArgoCDProvisioner) Provision(resource *database.ResourceInstance, config map[string]interface{}, provisionedBy string) error {
	appName := resource.ResourceName

	fmt.Printf("🚀 Creating ArgoCD Application '%s'\n", appName)

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.ArgoCD.URL == "" {
		return fmt.Errorf("argocd configuration not found in admin-config.yaml")
	}

	// Extract parameters from config
	repoURL := ""
	path := "."
	namespace := appName
	syncPolicy := "manual" // manual or auto

	if repoParam, ok := config["repo_url"]; ok {
		if repoStr, ok := repoParam.(string); ok {
			repoURL = repoStr
		}
	}
	if pathParam, ok := config["path"]; ok {
		if pathStr, ok := pathParam.(string); ok {
			path = pathStr
		}
	}
	if nsParam, ok := config["namespace"]; ok {
		if nsStr, ok := nsParam.(string); ok {
			namespace = nsStr
		}
	}
	if syncParam, ok := config["sync_policy"]; ok {
		if syncStr, ok := syncParam.(string); ok {
			syncPolicy = syncStr
		}
	}

	// If repo_url not provided, construct from repoName
	if repoURL == "" {
		repoName := appName
		if repoNameParam, ok := config["repo_name"]; ok {
			if repoNameStr, ok := repoNameParam.(string); ok {
				repoName = repoNameStr
			}
		}
		// Use internal Gitea service URL
		owner := adminConfig.Gitea.Username
		repoURL = fmt.Sprintf("http://gitea-http.gitea.svc.cluster.local:3000/%s/%s.git", owner, repoName)
	}

	fmt.Printf("   Repository: %s\n", repoURL)
	fmt.Printf("   Namespace: %s\n", namespace)

	// Authenticate with ArgoCD
	token, err := ap.authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with ArgoCD: %w", err)
	}

	// Create ArgoCD Application
	appSpec := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      appName,
			"namespace": "argocd",
		},
		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"repoURL":        repoURL,
				"targetRevision": "HEAD",
				"path":           path,
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": namespace,
			},
		},
	}

	// Add sync policy if auto
	if syncPolicy == "auto" {
		appSpec["spec"].(map[string]interface{})["syncPolicy"] = map[string]interface{}{
			"automated": map[string]interface{}{
				"prune":    true,
				"selfHeal": true,
			},
		}
	}

	appJSON, err := json.Marshal(appSpec)
	if err != nil {
		return fmt.Errorf("failed to marshal application spec: %w", err)
	}

	// Create application via ArgoCD API
	createURL := fmt.Sprintf("%s/api/v1/applications", adminConfig.ArgoCD.URL)
	req, err := http.NewRequest("POST", createURL, bytes.NewReader(appJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 409 {
		fmt.Printf("   ℹ️  Application '%s' already exists\n", appName)
		// Not an error - application exists
	} else if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("failed to create application, status %d: %s", resp.StatusCode, string(body))
	} else {
		fmt.Printf("   ✅ Application created successfully\n")
	}

	// ArgoCD app info - outputs updated by Manager
	fmt.Printf("   🔗 ArgoCD Application: %s/applications/%s\n", adminConfig.ArgoCD.URL, appName)
	return nil
}

// Deprovision deletes an ArgoCD Application
func (ap *ArgoCDProvisioner) Deprovision(resource *database.ResourceInstance) error {
	appName := resource.ResourceName

	fmt.Printf("🗑️  Deleting ArgoCD Application '%s'\n", appName)

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	// Authenticate with ArgoCD
	token, err := ap.authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with ArgoCD: %w", err)
	}

	// Delete application via ArgoCD API
	deleteURL := fmt.Sprintf("%s/api/v1/applications/%s", adminConfig.ArgoCD.URL, appName)
	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete application, status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Application '%s' deleted\n", appName)
	return nil
}

// GetStatus returns the status of the ArgoCD Application
func (ap *ArgoCDProvisioner) GetStatus(resource *database.ResourceInstance) (map[string]interface{}, error) {
	appName := resource.ResourceName
	status := make(map[string]interface{})

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to load config: %v", err)
		return status, nil
	}

	// Authenticate with ArgoCD
	token, err := ap.authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to authenticate: %v", err)
		return status, nil
	}

	// Get application status via ArgoCD API
	getURL := fmt.Sprintf("%s/api/v1/applications/%s", adminConfig.ArgoCD.URL, appName)
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to create request: %v", err)
		return status, nil
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to get application: %v", err)
		return status, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 404 {
		status["state"] = "not_found"
		return status, nil
	}

	if resp.StatusCode != 200 {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
		return status, nil
	}

	// Parse response
	var appInfo map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &appInfo); err == nil {
		if statusInfo, ok := appInfo["status"].(map[string]interface{}); ok {
			if syncStatus, ok := statusInfo["sync"].(map[string]interface{}); ok {
				status["sync_status"] = syncStatus["status"]
			}
			if healthStatus, ok := statusInfo["health"].(map[string]interface{}); ok {
				status["health_status"] = healthStatus["status"]
			}
		}
		status["state"] = "active"
	} else {
		status["state"] = "active"
	}

	status["argocd_url"] = fmt.Sprintf("%s/applications/%s", adminConfig.ArgoCD.URL, appName)
	return status, nil
}

// authenticateArgoCD authenticates with ArgoCD and returns a JWT token
func (ap *ArgoCDProvisioner) authenticateArgoCD(argoCDURL, username, password string) (string, error) {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	loginJSON, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login data: %w", err)
	}

	// Validate URL to prevent SSRF attacks
	if err := security.ValidateArgoCDURL(argoCDURL); err != nil {
		return "", fmt.Errorf("invalid ArgoCD URL: %w", err)
	}

	loginURL := fmt.Sprintf("%s/api/v1/session", argoCDURL)
	resp, err := http.Post(loginURL, "application/json", bytes.NewReader(loginJSON)) // #nosec G107 - URL validated above
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed, status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	token, ok := result["token"].(string)
	if !ok {
		return "", fmt.Errorf("token not found in response")
	}

	return token, nil
}