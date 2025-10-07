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
	if err := c.http.doYAMLRequest("POST", "/api/specs", yamlContent, &result); err != nil {
		return nil, fmt.Errorf("failed to deploy spec: %w", err)
	}
	return &result, nil
}

func (c *Client) ListSpecs() (map[string]*SpecResponse, error) {
	var result map[string]*SpecResponse
	if err := c.http.GET("/api/specs", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetSpec(name string) (*SpecResponse, error) {
	var result SpecResponse
	if err := c.http.GET("/api/specs/"+name, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSpec(name string) error {
	return c.http.DELETE("/api/specs/" + name)
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
