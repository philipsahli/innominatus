package tools

import (
	"context"
	"encoding/json"
)

// Tool defines the interface for MCP tools
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// InputSchema returns the JSON schema for the tool's input
	InputSchema() map[string]interface{}

	// Execute runs the tool with the given input
	Execute(ctx context.Context, input map[string]interface{}) (string, error)
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools
func (r *ToolRegistry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ToolDefinition represents a tool definition for MCP protocol
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// GetDefinitions returns MCP-compatible tool definitions
func (r *ToolRegistry) GetDefinitions() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			InputSchema: tool.InputSchema(),
		})
	}
	return defs
}

// ExecuteResult represents the result of tool execution
type ExecuteResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ToJSON converts ExecuteResult to JSON string
func (r *ExecuteResult) ToJSON() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// NewSuccessResult creates a success result
func NewSuccessResult(data interface{}) *ExecuteResult {
	return &ExecuteResult{
		Success: true,
		Data:    data,
	}
}

// NewErrorResult creates an error result
func NewErrorResult(err error) *ExecuteResult {
	return &ExecuteResult{
		Success: false,
		Error:   err.Error(),
	}
}
