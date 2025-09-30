package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
	token   string
}

func NewClient(baseURL string) *Client {
	client := &Client{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Check for API key in environment variable
	if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		client.token = apiKey
	}

	return client
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
	ID                  int64                  `json:"id"`
	ApplicationName     string                 `json:"application_name"`
	ResourceName        string                 `json:"resource_name"`
	ResourceType        string                 `json:"resource_type"`
	State               string                 `json:"state"`
	HealthStatus        string                 `json:"health_status"`
	Configuration       map[string]interface{} `json:"configuration"`
	ProviderID          *string                `json:"provider_id,omitempty"`
	ProviderMetadata    map[string]interface{} `json:"provider_metadata,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	LastHealthCheck     *time.Time             `json:"last_health_check,omitempty"`
	ErrorMessage        *string                `json:"error_message,omitempty"`
}

// Login authenticates with the server and stores the token
func (c *Client) Login(username, password string) error {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/login", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed (%d): %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	c.token = loginResp.Token
	return nil
}

// setAuthHeader adds the Authorization header if token is available
func (c *Client) setAuthHeader(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *Client) Deploy(yamlContent []byte) (*DeployResponse, error) {
	req, err := http.NewRequest("POST", c.baseURL+"/api/specs", bytes.NewReader(yamlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-yaml")
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy spec: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result DeployResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) ListSpecs() (map[string]*SpecResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/specs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list specs: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]*SpecResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

func (c *Client) GetSpec(name string) (*SpecResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/specs/"+name, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get spec: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("spec '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result SpecResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *Client) DeleteSpec(name string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/api/specs/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete spec: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("spec '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) ListEnvironments() (map[string]*Environment, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/environments", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string]*Environment
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ListWorkflows retrieves workflow executions from the server
func (c *Client) ListWorkflows(appName string) ([]interface{}, error) {
	url := c.baseURL + "/api/workflows"
	if appName != "" {
		url += "?app=" + appName
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result []interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ListResources retrieves resource instances from the server
func (c *Client) ListResources(appName string) (map[string][]*ResourceInstance, error) {
	url := c.baseURL + "/api/resources"
	if appName != "" {
		url += "?app=" + appName
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result map[string][]*ResourceInstance
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteApplication performs complete application deletion (infrastructure + database records)
func (c *Client) DeleteApplication(name string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/api/applications/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("application '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeprovisionApplication performs infrastructure teardown with audit trail preserved
func (c *Client) DeprovisionApplication(name string) error {
	req, err := http.NewRequest("POST", c.baseURL+"/api/applications/"+name+"/deprovision", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deprovision application: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("application '%s' not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// WorkflowStepDetail represents a detailed workflow step with logs
type WorkflowStepDetail struct {
	ID              int64     `json:"id"`
	StepNumber      int       `json:"step_number"`
	StepName        string    `json:"step_name"`
	StepType        string    `json:"step_type"`
	Status          string    `json:"status"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	DurationMs      *int64    `json:"duration_ms,omitempty"`
	ErrorMessage    *string   `json:"error_message,omitempty"`
	OutputLogs      *string   `json:"output_logs,omitempty"`
}

// WorkflowExecutionDetail represents detailed workflow execution information
type WorkflowExecutionDetail struct {
	ID              int64                   `json:"id"`
	ApplicationName string                  `json:"application_name"`
	WorkflowName    string                  `json:"workflow_name"`
	Status          string                  `json:"status"`
	StartedAt       time.Time               `json:"started_at"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	TotalSteps      int                     `json:"total_steps"`
	ErrorMessage    *string                 `json:"error_message,omitempty"`
	Steps           []WorkflowStepDetail    `json:"steps"`
}

// GetWorkflowDetail retrieves detailed workflow execution information including step logs
func (c *Client) GetWorkflowDetail(workflowID string) (*WorkflowExecutionDetail, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/workflows/"+workflowID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.setAuthHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow detail: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("workflow '%s' not found", workflowID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))
	}

	var result WorkflowExecutionDetail
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}