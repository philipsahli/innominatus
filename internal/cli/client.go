package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	token   string
	http    *HTTPHelper // HTTP helper for common operations
}

func NewClient(baseURL string) *Client {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	token := ""
	// Priority order for API key:
	// 1. Environment variable (highest priority - for CI/CD)
	// 2. Credentials file ($HOME/.idp-o/credentials)
	// 3. No API key (will prompt for login when needed)

	if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		// Environment variable takes precedence
		token = apiKey
	} else {
		// Try to load from credentials file
		creds, err := LoadCredentials()
		if err == nil && creds != nil {
			token = creds.APIKey
		}
		// If no credentials or error loading, token remains empty
	}

	client := &Client{
		baseURL: baseURL,
		client:  httpClient,
		token:   token,
		http:    newHTTPHelper(baseURL, httpClient, token),
	}

	return client
}

// HasToken returns true if the client has an API token loaded
func (c *Client) HasToken() bool {
	return c.token != ""
}

type DeployResponse struct {
	Message     string `json:"message"`
	Name        string `json:"name"`
	Environment string `json:"environment,omitempty"`
}

type SpecResponse struct {
	Metadata    map[string]interface{} `json:"metadata"`
	Containers  map[string]interface{} `json:"containers"`
	Resources   map[string]interface{} `json:"resources"`
	Environment map[string]interface{} `json:"environment,omitempty"`
	Graph       map[string][]string    `json:"graph"`
}

type Environment struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	TTL       string            `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
	Status    string            `json:"status"`
	Resources map[string]string `json:"resources"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Team     string `json:"team"`
	Role     string `json:"role"`
}

type ResourceInstance struct {
	ID               int64                  `json:"id"`
	ApplicationName  string                 `json:"application_name"`
	ResourceName     string                 `json:"resource_name"`
	ResourceType     string                 `json:"resource_type"`
	State            string                 `json:"state"`
	HealthStatus     string                 `json:"health_status"`
	Configuration    map[string]interface{} `json:"configuration"`
	ProviderID       *string                `json:"provider_id,omitempty"`
	ProviderMetadata map[string]interface{} `json:"provider_metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastHealthCheck  *time.Time             `json:"last_health_check,omitempty"`
	ErrorMessage     *string                `json:"error_message,omitempty"`
}

type ProviderSummary struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	Provisioners int    `json:"provisioners"`
	GoldenPaths  int    `json:"golden_paths"`
}

type ProviderStats struct {
	Providers    int `json:"providers"`
	Provisioners int `json:"provisioners"`
}

// Login authenticates with the server and stores the token
func (c *Client) Login(username, password string) error {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	var loginResp LoginResponse
	if err := c.http.POST("/api/login", loginData, &loginResp); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Update token in both client and http helper
	c.token = loginResp.Token
	c.http.token = loginResp.Token
	return nil
}

func (c *Client) Deploy(yamlContent []byte) (*DeployResponse, error) {
	var result DeployResponse
	// Updated to use /api/applications endpoint
	if err := c.http.doYAMLRequest("POST", "/api/applications", yamlContent, &result); err != nil {
		return nil, fmt.Errorf("failed to deploy spec: %w", err)
	}
	return &result, nil
}

func (c *Client) ListSpecs() (map[string]*SpecResponse, error) {
	var result map[string]*SpecResponse
	// Updated to use /api/applications endpoint
	if err := c.http.GET("/api/applications", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetSpec(name string) (*SpecResponse, error) {
	var result SpecResponse
	// Updated to use /api/applications endpoint
	if err := c.http.GET("/api/applications/"+name, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSpec(name string) error {
	// Updated to use /api/applications endpoint
	return c.http.DELETE("/api/applications/" + name)
}

func (c *Client) ListEnvironments() (map[string]*Environment, error) {
	var result map[string]*Environment
	if err := c.http.GET("/api/environments", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListWorkflows retrieves workflow executions from the server
func (c *Client) ListWorkflows(appName string) ([]interface{}, error) {
	path := "/api/workflows"
	if appName != "" {
		path += "?app=" + appName
	}

	var result []interface{}
	if err := c.http.GET(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListResources retrieves resource instances from the server
func (c *Client) ListResources(appName string) (map[string][]*ResourceInstance, error) {
	path := "/api/resources"
	if appName != "" {
		path += "?app=" + appName
	}

	var result map[string][]*ResourceInstance
	if err := c.http.GET(path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteApplication performs complete application deletion (infrastructure + database records)
func (c *Client) DeleteApplication(name string) error {
	return c.http.DELETE("/api/applications/" + name)
}

// DeprovisionApplication performs infrastructure teardown with audit trail preserved
func (c *Client) DeprovisionApplication(name string) error {
	return c.http.POST("/api/applications/"+name+"/deprovision", nil, nil)
}

// GetResource retrieves details of a specific resource
func (c *Client) GetResource(id string) (*ResourceInstance, error) {
	var result ResourceInstance
	if err := c.http.GET("/api/resources/"+id, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteResource deletes a specific resource
func (c *Client) DeleteResource(id string) error {
	return c.http.DELETE("/api/resources/" + id)
}

// UpdateResource updates resource configuration
func (c *Client) UpdateResource(id string, config map[string]interface{}) error {
	return c.http.PUT("/api/resources/"+id, config, nil)
}

// TransitionResource transitions resource to a new state
func (c *Client) TransitionResource(id string, state string) error {
	data := map[string]string{"state": state}
	return c.http.POST("/api/resources/"+id+"/transition", data, nil)
}

// GetResourceHealth gets cached resource health status
func (c *Client) GetResourceHealth(id string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.http.GET("/api/resources/"+id+"/health", &result); err != nil {
		return nil, err
	}
	return result, nil
}

// CheckResourceHealth triggers a new resource health check
func (c *Client) CheckResourceHealth(id string) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.http.POST("/api/resources/"+id+"/health", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// WorkflowStepDetail represents a detailed workflow step with logs
type WorkflowStepDetail struct {
	ID           int64      `json:"id"`
	StepNumber   int        `json:"step_number"`
	StepName     string     `json:"step_name"`
	StepType     string     `json:"step_type"`
	Status       string     `json:"status"`
	StartedAt    time.Time  `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	DurationMs   *int64     `json:"duration_ms,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	OutputLogs   *string    `json:"output_logs,omitempty"`
}

// WorkflowExecutionDetail represents detailed workflow execution information
type WorkflowExecutionDetail struct {
	ID              int64                `json:"id"`
	ApplicationName string               `json:"application_name"`
	WorkflowName    string               `json:"workflow_name"`
	Status          string               `json:"status"`
	StartedAt       time.Time            `json:"started_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	TotalSteps      int                  `json:"total_steps"`
	ErrorMessage    *string              `json:"error_message,omitempty"`
	Steps           []WorkflowStepDetail `json:"steps"`
}

// GetWorkflowDetail retrieves detailed workflow execution information including step logs
func (c *Client) GetWorkflowDetail(workflowID string) (*WorkflowExecutionDetail, error) {
	var result WorkflowExecutionDetail
	if err := c.http.GET("/api/workflows/"+workflowID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GraphExportCommand exports the workflow graph for an application
func (c *Client) GraphExportCommand(appName, format, outputFile string) error {
	// Make request to graph export endpoint
	url := fmt.Sprintf("%s/api/graph/%s/export?format=%s", c.baseURL, appName, format)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to export graph: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Write to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0600); err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
		fmt.Printf("Graph exported to %s (format: %s)\n", outputFile, format)
	} else {
		// Write to stdout
		if _, err := os.Stdout.Write(data); err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	}

	return nil
}

// GraphStatusCommand shows graph status and statistics for an application
func (c *Client) GraphStatusCommand(appName string) error {
	// Make request to graph status endpoint
	url := fmt.Sprintf("%s/api/graph/%s", c.baseURL, appName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get graph: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var graphData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&graphData); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display graph statistics
	fmt.Printf("Graph Status for Application: %s\n\n", appName)

	if nodes, ok := graphData["nodes"].(map[string]interface{}); ok {
		fmt.Printf("Total Nodes: %d\n", len(nodes))

		// Count by type
		typeCounts := make(map[string]int)
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if nodeType, ok := nodeMap["type"].(string); ok {
					typeCounts[nodeType]++
				}
			}
		}

		fmt.Println("\nNode Counts by Type:")
		for nodeType, count := range typeCounts {
			fmt.Printf("  %s: %d\n", nodeType, count)
		}

		// Count by state
		stateCounts := make(map[string]int)
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				if state, ok := nodeMap["state"].(string); ok {
					stateCounts[state]++
				}
			}
		}

		fmt.Println("\nNode Counts by State:")
		for state, count := range stateCounts {
			fmt.Printf("  %s: %d\n", state, count)
		}
	}

	if edges, ok := graphData["edges"].(map[string]interface{}); ok {
		fmt.Printf("\nTotal Edges: %d\n", len(edges))
	}

	return nil
}

// User represents a user in the system
type User struct {
	Username string `json:"username"`
	Team     string `json:"team"`
	Role     string `json:"role"`
}

// CreateUser creates a new user via the API
func (c *Client) CreateUser(username, password, team, role string) error {
	data := map[string]string{
		"username": username,
		"password": password,
		"team":     team,
		"role":     role,
	}
	return c.http.POST("/admin/users", data, nil)
}

// GetUser retrieves user information
func (c *Client) GetUser(username string) (*User, error) {
	var user User
	if err := c.http.GET(fmt.Sprintf("/admin/users/%s", username), &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers retrieves all users
func (c *Client) ListUsers() ([]User, error) {
	var result struct {
		Users []User `json:"users"`
	}
	if err := c.http.GET("/users", &result); err != nil {
		return nil, err
	}
	return result.Users, nil
}

// UpdateUser updates user information
func (c *Client) UpdateUser(username string, updates map[string]string) error {
	return c.http.PUT(fmt.Sprintf("/admin/users/%s", username), updates, nil)
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(username string) error {
	return c.http.DELETE(fmt.Sprintf("/admin/users/%s", username))
}

// AdminGetAPIKeys retrieves API keys for a specific user (admin only)
func (c *Client) AdminGetAPIKeys(username string) ([]map[string]interface{}, error) {
	var result struct {
		Username string                   `json:"username"`
		APIKeys  []map[string]interface{} `json:"api_keys"`
	}
	if err := c.http.GET(fmt.Sprintf("/admin/users/%s/api-keys", username), &result); err != nil {
		return nil, err
	}
	return result.APIKeys, nil
}

// AdminGenerateAPIKey generates an API key for a user (admin only)
func (c *Client) AdminGenerateAPIKey(username, name string, expiryDays int) (map[string]interface{}, error) {
	data := map[string]interface{}{
		"name":        name,
		"expiry_days": expiryDays,
	}
	var result map[string]interface{}
	if err := c.http.POST(fmt.Sprintf("/admin/users/%s/api-keys", username), data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// AdminRevokeAPIKey revokes an API key for a user (admin only)
func (c *Client) AdminRevokeAPIKey(username, keyName string) error {
	return c.http.DELETE(fmt.Sprintf("/admin/users/%s/api-keys/%s", username, keyName))
}

// Team represents a team in the system
type Team struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Members     []string `json:"members,omitempty"`
}

// ListTeams retrieves all teams
func (c *Client) ListTeams() ([]Team, error) {
	var teams []Team
	if err := c.http.GET("/teams", &teams); err != nil {
		return nil, err
	}
	return teams, nil
}

// GetTeam retrieves a specific team
func (c *Client) GetTeam(teamID string) (*Team, error) {
	var team Team
	if err := c.http.GET(fmt.Sprintf("/teams/%s", teamID), &team); err != nil {
		return nil, err
	}
	return &team, nil
}

// CreateTeam creates a new team
func (c *Client) CreateTeam(name, description string) error {
	data := map[string]string{
		"name":        name,
		"description": description,
	}
	return c.http.POST("/teams", data, nil)
}

// DeleteTeam deletes a team
func (c *Client) DeleteTeam(teamID string) error {
	return c.http.DELETE(fmt.Sprintf("/teams/%s", teamID))
}

// ListProviders retrieves all loaded providers from the server
func (c *Client) ListProviders() ([]ProviderSummary, error) {
	var providers []ProviderSummary
	if err := c.http.GET("/api/providers", &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

// GetProviderStats retrieves provider statistics from the server
func (c *Client) GetProviderStats() (*ProviderStats, error) {
	var stats ProviderStats
	if err := c.http.GET("/api/providers/stats", &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}
