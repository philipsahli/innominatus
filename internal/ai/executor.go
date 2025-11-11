package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"innominatus/internal/mcp/tools"

	"github.com/rs/zerolog/log"
)

// ToolExecutor executes tool calls by making internal API requests
// Now uses shared tool registry for common tools
type ToolExecutor struct {
	apiBaseURL string
	authToken  string
	registry   *tools.ToolRegistry
}

// NewToolExecutor creates a new tool executor
// authToken should be the user's authentication token for API access
func NewToolExecutor(apiBaseURL, authToken string) *ToolExecutor {
	return &ToolExecutor{
		apiBaseURL: apiBaseURL,
		authToken:  authToken,
		registry:   tools.BuildRegistry(apiBaseURL, authToken),
	}
}

// ExecuteTool executes a tool by name with the given input parameters
// Uses shared tool registry when available, falls back to AI-specific tools
func (e *ToolExecutor) ExecuteTool(ctx context.Context, toolName string, input map[string]interface{}) (string, error) {
	// Try shared tool registry first (for tools shared with MCP server)
	if tool, ok := e.registry.Get(toolName); ok {
		log.Debug().Str("tool", toolName).Msg("Using shared tool from registry")
		return tool.Execute(ctx, input)
	}

	// Fall back to AI-specific tools (not in shared registry)
	log.Debug().Str("tool", toolName).Msg("Using AI-specific tool implementation")
	switch toolName {
	case "list_applications":
		// Alias for list_specs (AI assistant uses different naming)
		if specsTool, ok := e.registry.Get("list_specs"); ok {
			return specsTool.Execute(ctx, input)
		}
		return e.listApplications(ctx)
	case "get_application":
		appName, ok := input["app_name"].(string)
		if !ok {
			return "", fmt.Errorf("app_name parameter is required and must be a string")
		}
		return e.getApplication(ctx, appName)
	case "deploy_application":
		// Alias for submit_spec (AI uses spec_content, MCP uses spec)
		specContent, ok := input["spec_content"].(string)
		if !ok {
			return "", fmt.Errorf("spec_content parameter is required and must be a string")
		}
		if submitTool, ok := e.registry.Get("submit_spec"); ok {
			return submitTool.Execute(ctx, map[string]interface{}{"spec": specContent})
		}
		return e.deployApplication(ctx, specContent)
	case "delete_application":
		appName, ok := input["app_name"].(string)
		if !ok {
			return "", fmt.Errorf("app_name parameter is required and must be a string")
		}
		return e.deleteApplication(ctx, appName)
	case "list_workflows":
		// MCP tool is list_workflow_executions
		if workflowsTool, ok := e.registry.Get("list_workflow_executions"); ok {
			return workflowsTool.Execute(ctx, input)
		}
		appName, _ := input["app_name"].(string) // Optional parameter
		return e.listWorkflows(ctx, appName)
	case "get_workflow":
		// MCP tool is get_workflow_status
		workflowID, ok := input["workflow_id"].(string)
		if !ok {
			return "", fmt.Errorf("workflow_id parameter is required and must be a string")
		}
		if workflowTool, ok := e.registry.Get("get_workflow_status"); ok {
			return workflowTool.Execute(ctx, map[string]interface{}{"execution_id": workflowID})
		}
		return e.getWorkflow(ctx, workflowID)
	case "get_dashboard_stats":
		return e.getDashboardStats(ctx)
	case "get_current_user":
		return e.getCurrentUser(ctx)
	default:
		return "", fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (e *ToolExecutor) listApplications(ctx context.Context) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/specs", nil)
	if err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}

	// If no specs, provide friendly message
	var specs map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &specs); err == nil && len(specs) == 0 {
		return "No applications are currently deployed.", nil
	}

	return resp, nil
}

func (e *ToolExecutor) getApplication(ctx context.Context, appName string) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/specs/"+appName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get application: %w", err)
	}
	return resp, nil
}

func (e *ToolExecutor) deployApplication(ctx context.Context, specContent string) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "POST", "/api/specs", []byte(specContent))
	if err != nil {
		return "", fmt.Errorf("failed to deploy application: %w", err)
	}
	return fmt.Sprintf("Application deployed successfully. Response: %s", resp), nil
}

func (e *ToolExecutor) deleteApplication(ctx context.Context, appName string) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "DELETE", "/api/applications/"+appName, nil)
	if err != nil {
		return "", fmt.Errorf("failed to delete application: %w", err)
	}
	return fmt.Sprintf("Application '%s' deleted successfully. %s", appName, resp), nil
}

func (e *ToolExecutor) listWorkflows(ctx context.Context, appName string) (string, error) {
	endpoint := "/api/workflows"
	if appName != "" {
		endpoint += "?app=" + appName
	}

	resp, err := e.makeAPIRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list workflows: %w", err)
	}
	return resp, nil
}

func (e *ToolExecutor) getWorkflow(ctx context.Context, workflowID string) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/workflows/"+workflowID, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow: %w", err)
	}
	return resp, nil
}

func (e *ToolExecutor) listResources(ctx context.Context, appName string) (string, error) {
	endpoint := "/api/resources"
	if appName != "" {
		endpoint += "?app=" + appName
	}

	resp, err := e.makeAPIRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("failed to list resources: %w", err)
	}
	return resp, nil
}

func (e *ToolExecutor) getDashboardStats(ctx context.Context) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/stats", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get dashboard stats: %w", err)
	}
	return resp, nil
}

func (e *ToolExecutor) getCurrentUser(ctx context.Context) (string, error) {
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/auth/whoami", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return resp, nil
}

// makeAPIRequest makes an internal HTTP request to the innominatus API
func (e *ToolExecutor) makeAPIRequest(ctx context.Context, method, endpoint string, body []byte) (string, error) {
	url := e.apiBaseURL + endpoint

	log.Debug().
		Str("method", method).
		Str("endpoint", endpoint).
		Str("url", url).
		Bool("has_body", body != nil).
		Int("body_size", len(body)).
		Msg("Making API request")

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("method", method).
			Str("endpoint", endpoint).
			Msg("Failed to create HTTP request")
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication and headers
	req.Header.Set("Authorization", "Bearer "+e.authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json") // Ensure JSON response instead of HTML redirect

	// Make request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("method", method).
			Str("endpoint", endpoint).
			Msg("Failed to execute request")
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warn().
				Err(err).
				Msg("Failed to close response body")
		}
	}()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().
			Err(err).
			Str("endpoint", endpoint).
			Msg("Failed to read response body")
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("method", method).
			Str("endpoint", endpoint).
			Str("response_body", string(respBody)).
			Msg("Request failed")
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	log.Debug().
		Int("status_code", resp.StatusCode).
		Str("method", method).
		Str("endpoint", endpoint).
		Int("response_size", len(respBody)).
		Msg("Completed API request")

	return string(respBody), nil
}

func (e *ToolExecutor) listGoldenPaths(ctx context.Context) (string, error) {
	// Get all providers
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/providers", nil)
	if err != nil {
		return "", fmt.Errorf("failed to list providers: %w", err)
	}

	// Parse providers response
	var providers []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &providers); err != nil {
		return "", fmt.Errorf("failed to parse providers response: %w", err)
	}

	// Extract golden paths from all providers
	var goldenPaths []map[string]interface{}
	for _, provider := range providers {
		providerName, _ := provider["name"].(string)
		workflows, ok := provider["workflows"].([]interface{})
		if !ok {
			continue
		}

		for _, workflow := range workflows {
			w, ok := workflow.(map[string]interface{})
			if !ok {
				continue
			}

			// Filter for golden paths only
			if category, ok := w["category"].(string); ok && category == "goldenpath" {
				goldenPath := map[string]interface{}{
					"name":        w["name"],
					"provider":    providerName,
					"description": w["description"],
					"tags":        w["tags"],
				}
				goldenPaths = append(goldenPaths, goldenPath)
			}
		}
	}

	// Format response
	result := map[string]interface{}{
		"count":        len(goldenPaths),
		"golden_paths": goldenPaths,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal golden paths: %w", err)
	}

	return string(resultJSON), nil
}

func (e *ToolExecutor) listProviders(ctx context.Context) (string, error) {
	// Get all providers
	resp, err := e.makeAPIRequest(ctx, "GET", "/api/providers", nil)
	if err != nil {
		return "", fmt.Errorf("failed to list providers: %w", err)
	}

	// Parse providers response
	var providers []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &providers); err != nil {
		return "", fmt.Errorf("failed to parse providers response: %w", err)
	}

	// Format providers summary
	var providersSummary []map[string]interface{}
	for _, provider := range providers {
		workflows, _ := provider["workflows"].([]interface{})
		capabilities, _ := provider["capabilities"].(map[string]interface{})
		resourceTypes, _ := capabilities["resourceTypes"].([]interface{})

		summary := map[string]interface{}{
			"name":           provider["name"],
			"category":       provider["category"],
			"description":    provider["description"],
			"resource_types": resourceTypes,
			"workflow_count": len(workflows),
		}
		providersSummary = append(providersSummary, summary)
	}

	// Format response
	result := map[string]interface{}{
		"count":     len(providersSummary),
		"providers": providersSummary,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal providers: %w", err)
	}

	return string(resultJSON), nil
}
