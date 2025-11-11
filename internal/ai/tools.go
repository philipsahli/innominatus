package ai

import (
	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai/llm"
)

// GetAvailableTools returns the list of tools available to the AI assistant
func GetAvailableTools() []llm.Tool {
	return []llm.Tool{
		{
			Name:        "list_applications",
			Description: "List all deployed applications in the platform. Returns application names, environments, status, and resource counts.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "get_application",
			Description: "Get detailed information about a specific application including its Score specification, resources, and deployment status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"app_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the application to retrieve",
					},
				},
				"required": []string{"app_name"},
			},
		},
		{
			Name:        "deploy_application",
			Description: "Deploy a new application or update an existing one using a Score specification. The spec should be valid YAML format.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"spec_content": map[string]interface{}{
						"type":        "string",
						"description": "The complete Score specification in YAML format",
					},
				},
				"required": []string{"spec_content"},
			},
		},
		{
			Name:        "delete_application",
			Description: "Delete a deployed application and all its associated resources. This action cannot be undone.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"app_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the application to delete",
					},
				},
				"required": []string{"app_name"},
			},
		},
		{
			Name:        "list_workflows",
			Description: "List workflow executions. Can optionally filter by application name.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"app_name": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter workflows by application name",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "get_workflow",
			Description: "Get detailed information about a specific workflow execution including all steps, status, and logs.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"workflow_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the workflow execution to retrieve",
					},
				},
				"required": []string{"workflow_id"},
			},
		},
		{
			Name:        "list_resources",
			Description: "List all resources (databases, caches, volumes, etc.) deployed in the platform. Can optionally filter by application.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"app_name": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Filter resources by application name",
					},
				},
				"required": []string{},
			},
		},
		{
			Name:        "get_dashboard_stats",
			Description: "Get platform statistics including total applications, workflows, resources, and users.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "list_golden_paths",
			Description: "List all available golden path workflows. Golden paths are standardized, opinionated workflows for common platform tasks like team onboarding, database provisioning, or application deployment.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "list_providers",
			Description: "List all platform providers and their capabilities. Shows which teams handle which resource types (e.g., database-team handles postgres, container-team handles kubernetes).",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
		{
			Name:        "get_current_user",
			Description: "Get information about the current authenticated user including username, team/group membership, and role. Use this to understand who you're assisting and their permissions.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
				"required":   []string{},
			},
		},
	}
}
