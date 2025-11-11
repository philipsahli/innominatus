package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/rs/zerolog/log"
)

// BaseTool provides common functionality for all tools
type BaseTool struct {
	client *APIClient
}

// NewBaseTool creates a new base tool
func NewBaseTool(client *APIClient) *BaseTool {
	return &BaseTool{client: client}
}

// ===================================================================
// 1. ListGoldenPathsTool
// ===================================================================

type ListGoldenPathsTool struct {
	*BaseTool
}

func NewListGoldenPathsTool(client *APIClient) *ListGoldenPathsTool {
	return &ListGoldenPathsTool{BaseTool: NewBaseTool(client)}
}

func (t *ListGoldenPathsTool) Name() string {
	return "list_golden_paths"
}

func (t *ListGoldenPathsTool) Description() string {
	return "List all available golden path workflows (pre-configured multi-resource deployment patterns)"
}

func (t *ListGoldenPathsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListGoldenPathsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	resp, err := t.client.Get(ctx, "/api/providers")
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch providers")
		return "", fmt.Errorf("failed to fetch providers: %w", err)
	}

	// Parse response
	var providers []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &providers); err != nil {
		return "", fmt.Errorf("failed to parse providers response: %w", err)
	}

	// Extract golden path workflows
	goldenPaths := []map[string]interface{}{}
	for _, provider := range providers {
		if workflows, ok := provider["workflows"].([]interface{}); ok {
			for _, wf := range workflows {
				if workflow, ok := wf.(map[string]interface{}); ok {
					if category, _ := workflow["category"].(string); category == "goldenpath" {
						goldenPaths = append(goldenPaths, workflow)
					}
				}
			}
		}
	}

	result := map[string]interface{}{
		"golden_paths": goldenPaths,
		"count":        len(goldenPaths),
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

// ===================================================================
// 2. ListProvidersTool
// ===================================================================

type ListProvidersTool struct {
	*BaseTool
}

func NewListProvidersTool(client *APIClient) *ListProvidersTool {
	return &ListProvidersTool{BaseTool: NewBaseTool(client)}
}

func (t *ListProvidersTool) Name() string {
	return "list_providers"
}

func (t *ListProvidersTool) Description() string {
	return "List all platform providers and their capabilities"
}

func (t *ListProvidersTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListProvidersTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	resp, err := t.client.Get(ctx, "/api/providers")
	if err != nil {
		return "", fmt.Errorf("failed to fetch providers: %w", err)
	}

	// Parse and enhance response
	var providers []map[string]interface{}
	if err := json.Unmarshal([]byte(resp), &providers); err != nil {
		return "", fmt.Errorf("failed to parse providers response: %w", err)
	}

	// Add resource type counts
	for i := range providers {
		if capabilities, ok := providers[i]["capabilities"].(map[string]interface{}); ok {
			if resourceTypes, ok := capabilities["resourceTypes"].([]interface{}); ok {
				providers[i]["resource_type_count"] = len(resourceTypes)
			}
		}
	}

	jsonResult, _ := json.Marshal(map[string]interface{}{
		"providers": providers,
		"count":     len(providers),
	})
	return string(jsonResult), nil
}

// ===================================================================
// 3. GetProviderDetailsTool
// ===================================================================

type GetProviderDetailsTool struct {
	*BaseTool
}

func NewGetProviderDetailsTool(client *APIClient) *GetProviderDetailsTool {
	return &GetProviderDetailsTool{BaseTool: NewBaseTool(client)}
}

func (t *GetProviderDetailsTool) Name() string {
	return "get_provider_details"
}

func (t *GetProviderDetailsTool) Description() string {
	return "Get detailed information about a specific provider"
}

func (t *GetProviderDetailsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Provider name (e.g., 'database-team', 'container-team')",
			},
		},
		"required": []string{"name"},
	}
}

func (t *GetProviderDetailsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	name, ok := input["name"].(string)
	if !ok {
		return "", fmt.Errorf("name parameter is required and must be a string")
	}

	endpoint := fmt.Sprintf("/api/providers/%s", name)
	resp, err := t.client.Get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch provider details: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 4. ExecuteWorkflowTool
// ===================================================================

type ExecuteWorkflowTool struct {
	*BaseTool
}

func NewExecuteWorkflowTool(client *APIClient) *ExecuteWorkflowTool {
	return &ExecuteWorkflowTool{BaseTool: NewBaseTool(client)}
}

func (t *ExecuteWorkflowTool) Name() string {
	return "execute_workflow"
}

func (t *ExecuteWorkflowTool) Description() string {
	return "Execute a workflow with provided inputs"
}

func (t *ExecuteWorkflowTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"workflow_name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the workflow to execute",
			},
			"inputs": map[string]interface{}{
				"type":        "object",
				"description": "Input parameters for the workflow",
			},
		},
		"required": []string{"workflow_name", "inputs"},
	}
}

func (t *ExecuteWorkflowTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	workflowName, ok := input["workflow_name"].(string)
	if !ok {
		return "", fmt.Errorf("workflow_name parameter is required and must be a string")
	}

	inputs, ok := input["inputs"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("inputs parameter is required and must be an object")
	}

	requestBody := map[string]interface{}{
		"workflow_name": workflowName,
		"inputs":        inputs,
	}

	resp, err := t.client.Post(ctx, "/api/workflows/execute", requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to execute workflow: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 5. GetWorkflowStatusTool
// ===================================================================

type GetWorkflowStatusTool struct {
	*BaseTool
}

func NewGetWorkflowStatusTool(client *APIClient) *GetWorkflowStatusTool {
	return &GetWorkflowStatusTool{BaseTool: NewBaseTool(client)}
}

func (t *GetWorkflowStatusTool) Name() string {
	return "get_workflow_status"
}

func (t *GetWorkflowStatusTool) Description() string {
	return "Get the status and details of a workflow execution"
}

func (t *GetWorkflowStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"execution_id": map[string]interface{}{
				"type":        "string",
				"description": "Workflow execution ID",
			},
		},
		"required": []string{"execution_id"},
	}
}

func (t *GetWorkflowStatusTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	executionID, ok := input["execution_id"].(string)
	if !ok {
		return "", fmt.Errorf("execution_id parameter is required and must be a string")
	}

	endpoint := fmt.Sprintf("/api/workflows/%s", executionID)
	resp, err := t.client.Get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workflow status: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 6. ListWorkflowExecutionsTool
// ===================================================================

type ListWorkflowExecutionsTool struct {
	*BaseTool
}

func NewListWorkflowExecutionsTool(client *APIClient) *ListWorkflowExecutionsTool {
	return &ListWorkflowExecutionsTool{BaseTool: NewBaseTool(client)}
}

func (t *ListWorkflowExecutionsTool) Name() string {
	return "list_workflow_executions"
}

func (t *ListWorkflowExecutionsTool) Description() string {
	return "List recent workflow executions"
}

func (t *ListWorkflowExecutionsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of executions to return (default: 10)",
			},
		},
	}
}

func (t *ListWorkflowExecutionsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	limit := 10
	if limitVal, ok := input["limit"]; ok {
		switch v := limitVal.(type) {
		case float64:
			limit = int(v)
		case int:
			limit = v
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				limit = parsed
			}
		}
	}

	endpoint := fmt.Sprintf("/api/workflows?limit=%d", limit)
	resp, err := t.client.Get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch workflow executions: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 7. ListResourcesTool
// ===================================================================

type ListResourcesTool struct {
	*BaseTool
}

func NewListResourcesTool(client *APIClient) *ListResourcesTool {
	return &ListResourcesTool{BaseTool: NewBaseTool(client)}
}

func (t *ListResourcesTool) Name() string {
	return "list_resources"
}

func (t *ListResourcesTool) Description() string {
	return "List provisioned resources, optionally filtered by type"
}

func (t *ListResourcesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"type": map[string]interface{}{
				"type":        "string",
				"description": "Filter by resource type (e.g., 'postgres', 's3', 'namespace')",
			},
		},
	}
}

func (t *ListResourcesTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	endpoint := "/api/resources"

	if resourceType, ok := input["type"].(string); ok && resourceType != "" {
		endpoint = fmt.Sprintf("%s?type=%s", endpoint, resourceType)
	}

	resp, err := t.client.Get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch resources: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 8. GetResourceDetailsTool
// ===================================================================

type GetResourceDetailsTool struct {
	*BaseTool
}

func NewGetResourceDetailsTool(client *APIClient) *GetResourceDetailsTool {
	return &GetResourceDetailsTool{BaseTool: NewBaseTool(client)}
}

func (t *GetResourceDetailsTool) Name() string {
	return "get_resource_details"
}

func (t *GetResourceDetailsTool) Description() string {
	return "Get detailed information about a specific resource"
}

func (t *GetResourceDetailsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"resource_id": map[string]interface{}{
				"type":        "string",
				"description": "Resource ID",
			},
		},
		"required": []string{"resource_id"},
	}
}

func (t *GetResourceDetailsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	resourceID, ok := input["resource_id"].(string)
	if !ok {
		return "", fmt.Errorf("resource_id parameter is required and must be a string")
	}

	endpoint := fmt.Sprintf("/api/resources/%s", resourceID)
	resp, err := t.client.Get(ctx, endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch resource details: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 9. ListSpecsTool
// ===================================================================

type ListSpecsTool struct {
	*BaseTool
}

func NewListSpecsTool(client *APIClient) *ListSpecsTool {
	return &ListSpecsTool{BaseTool: NewBaseTool(client)}
}

func (t *ListSpecsTool) Name() string {
	return "list_specs"
}

func (t *ListSpecsTool) Description() string {
	return "List deployed Score specifications (applications)"
}

func (t *ListSpecsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListSpecsTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	resp, err := t.client.Get(ctx, "/api/specs")
	if err != nil {
		return "", fmt.Errorf("failed to fetch specs: %w", err)
	}

	return resp, nil
}

// ===================================================================
// 10. SubmitSpecTool
// ===================================================================

type SubmitSpecTool struct {
	*BaseTool
}

func NewSubmitSpecTool(client *APIClient) *SubmitSpecTool {
	return &SubmitSpecTool{BaseTool: NewBaseTool(client)}
}

func (t *SubmitSpecTool) Name() string {
	return "submit_spec"
}

func (t *SubmitSpecTool) Description() string {
	return "Deploy a new Score specification"
}

func (t *SubmitSpecTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"spec": map[string]interface{}{
				"type":        "string",
				"description": "Score specification in YAML format",
			},
		},
		"required": []string{"spec"},
	}
}

func (t *SubmitSpecTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	spec, ok := input["spec"].(string)
	if !ok {
		return "", fmt.Errorf("spec parameter is required and must be a string")
	}

	resp, err := t.client.PostYAML(ctx, "/api/specs", spec)
	if err != nil {
		return "", fmt.Errorf("failed to submit spec: %w", err)
	}

	return resp, nil
}

// ===================================================================
// Registry Builder
// ===================================================================

// BuildRegistry creates a registry with all standard tools
func BuildRegistry(apiBaseURL, authToken string) *ToolRegistry {
	client := NewAPIClient(apiBaseURL, authToken)
	registry := NewToolRegistry()

	// Register all 10 tools
	registry.Register(NewListGoldenPathsTool(client))
	registry.Register(NewListProvidersTool(client))
	registry.Register(NewGetProviderDetailsTool(client))
	registry.Register(NewExecuteWorkflowTool(client))
	registry.Register(NewGetWorkflowStatusTool(client))
	registry.Register(NewListWorkflowExecutionsTool(client))
	registry.Register(NewListResourcesTool(client))
	registry.Register(NewGetResourceDetailsTool(client))
	registry.Register(NewListSpecsTool(client))
	registry.Register(NewSubmitSpecTool(client))

	log.Info().Int("tool_count", len(registry.tools)).Msg("Tool registry initialized")
	return registry
}
