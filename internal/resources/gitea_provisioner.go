package resources

import (
	"encoding/json"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"io"
	"net/http"
	"strings"
	"time"
)

// GiteaProvisioner handles Gitea repository creation
type GiteaProvisioner struct {
	repo *database.ResourceRepository
}

// NewGiteaProvisioner creates a new Gitea provisioner
func NewGiteaProvisioner(repo *database.ResourceRepository) *GiteaProvisioner {
	return &GiteaProvisioner{
		repo: repo,
	}
}

// Provision creates a Gitea repository
func (gp *GiteaProvisioner) Provision(resource *database.ResourceInstance, config map[string]interface{}, provisionedBy string) error {
	repoName := resource.ResourceName

	fmt.Printf("📦 Creating Gitea repository '%s'\n", repoName)

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found in admin-config.yaml")
	}

	// Extract parameters from config
	description := ""
	private := false
	owner := adminConfig.Gitea.Username

	if descParam, ok := config["description"]; ok {
		if descStr, ok := descParam.(string); ok {
			description = descStr
		}
	}
	if privateParam, ok := config["private"]; ok {
		if privateBool, ok := privateParam.(bool); ok {
			private = privateBool
		}
	}
	if ownerParam, ok := config["owner"]; ok {
		if ownerStr, ok := ownerParam.(string); ok {
			owner = ownerStr
		}
	}

	// Create repository via Gitea API
	repoData := map[string]interface{}{
		"name":        repoName,
		"description": description,
		"private":     private,
		"auto_init":   true, // Initialize with README
	}

	repoJSON, err := json.Marshal(repoData)
	if err != nil {
		return fmt.Errorf("failed to marshal repository data: %w", err)
	}

	// Determine API endpoint based on owner
	createURL := fmt.Sprintf("%s/api/v1/user/repos", adminConfig.Gitea.URL)
	if owner != adminConfig.Gitea.Username {
		// Use organization endpoint if owner is not the admin user
		createURL = fmt.Sprintf("%s/api/v1/orgs/%s/repos", adminConfig.Gitea.URL, owner)
	}

	req, err := http.NewRequest("POST", createURL, strings.NewReader(string(repoJSON)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 409 {
		fmt.Printf("   ℹ️  Repository %s/%s already exists\n", owner, repoName)
		// Not an error - repository exists
	} else if resp.StatusCode != 201 {
		return fmt.Errorf("failed to create repository, status %d: %s", resp.StatusCode, string(body))
	} else {
		fmt.Printf("   ✅ Repository created successfully\n")
	}

	// Store repository URL - outputs updated by Manager
	repoURL := fmt.Sprintf("%s/%s/%s", adminConfig.Gitea.URL, owner, repoName)
	fmt.Printf("   🔗 Repository available at: %s\n", repoURL)
	return nil
}

// Deprovision deletes a Gitea repository
func (gp *GiteaProvisioner) Deprovision(resource *database.ResourceInstance) error {
	repoName := resource.ResourceName

	fmt.Printf("🗑️  Deleting Gitea repository '%s'\n", repoName)

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	owner := adminConfig.Gitea.Username

	// Delete repository via Gitea API
	deleteURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", adminConfig.Gitea.URL, owner, repoName)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete repository, status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("✅ Repository '%s' deleted\n", repoName)
	return nil
}

// GetStatus returns the status of the Gitea repository
func (gp *GiteaProvisioner) GetStatus(resource *database.ResourceInstance) (map[string]interface{}, error) {
	repoName := resource.ResourceName
	status := make(map[string]interface{})

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to load config: %v", err)
		return status, nil
	}

	owner := adminConfig.Gitea.Username

	// Check repository existence via API
	checkURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", adminConfig.Gitea.URL, owner, repoName)

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to create request: %v", err)
		return status, nil
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to check repository: %v", err)
		return status, nil
	}
	defer func() { _ = resp.Body.Close() }()

	//nolint:staticcheck // Simple if-else is clearer for HTTP status check - QF1003
	if resp.StatusCode == 200 {
		status["state"] = "active"
		status["repository_url"] = fmt.Sprintf("%s/%s/%s", adminConfig.Gitea.URL, owner, repoName)
	} else if resp.StatusCode == 404 {
		status["state"] = "not_found"
	} else {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
	}

	return status, nil
}
