package tools

import (
	"context"
	"testing"
)

// MockTool implements the Tool interface for testing
type MockTool struct {
	name        string
	description string
	schema      map[string]interface{}
	execFunc    func(context.Context, map[string]interface{}) (string, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) InputSchema() map[string]interface{} {
	return m.schema
}

func (m *MockTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	return m.execFunc(ctx, input)
}

func TestToolRegistry_Register(t *testing.T) {
	registry := NewToolRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "A test tool",
		schema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}

	registry.Register(tool)

	// Verify tool was registered
	retrieved, ok := registry.Get("test_tool")
	if !ok {
		t.Fatal("Expected tool to be registered")
	}

	if retrieved.Name() != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", retrieved.Name())
	}
}

func TestToolRegistry_Get(t *testing.T) {
	registry := NewToolRegistry()

	tool := &MockTool{
		name:        "existing_tool",
		description: "Exists",
		schema:      map[string]interface{}{},
	}

	registry.Register(tool)

	tests := []struct {
		name      string
		toolName  string
		wantFound bool
	}{
		{
			name:      "existing tool",
			toolName:  "existing_tool",
			wantFound: true,
		},
		{
			name:      "non-existing tool",
			toolName:  "nonexistent",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, found := registry.Get(tt.toolName)
			if found != tt.wantFound {
				t.Errorf("Get() found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestToolRegistry_List(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := &MockTool{name: "tool1", description: "Tool 1", schema: map[string]interface{}{}}
	tool2 := &MockTool{name: "tool2", description: "Tool 2", schema: map[string]interface{}{}}
	tool3 := &MockTool{name: "tool3", description: "Tool 3", schema: map[string]interface{}{}}

	registry.Register(tool1)
	registry.Register(tool2)
	registry.Register(tool3)

	tools := registry.List()

	if len(tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(tools))
	}

	// Verify all tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name()] = true
	}

	expectedNames := []string{"tool1", "tool2", "tool3"}
	for _, name := range expectedNames {
		if !toolNames[name] {
			t.Errorf("Expected tool '%s' to be in list", name)
		}
	}
}

func TestToolRegistry_GetDefinitions(t *testing.T) {
	registry := NewToolRegistry()

	tool := &MockTool{
		name:        "test_tool",
		description: "A test tool",
		schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	registry.Register(tool)

	defs := registry.GetDefinitions()

	if len(defs) != 1 {
		t.Fatalf("Expected 1 definition, got %d", len(defs))
	}

	def := defs[0]
	if def.Name != "test_tool" {
		t.Errorf("Expected name 'test_tool', got '%s'", def.Name)
	}

	if def.Description != "A test tool" {
		t.Errorf("Expected description 'A test tool', got '%s'", def.Description)
	}

	if def.InputSchema == nil {
		t.Error("Expected InputSchema to be non-nil")
	}
}

func TestExecuteResult_ToJSON(t *testing.T) {
	tests := []struct {
		name   string
		result *ExecuteResult
		want   string
	}{
		{
			name:   "success result",
			result: NewSuccessResult(map[string]string{"status": "ok"}),
			want:   `{"success":true,"data":{"status":"ok"}}`,
		},
		{
			name:   "error result",
			result: NewErrorResult(context.DeadlineExceeded),
			want:   `{"success":false,"error":"context deadline exceeded"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.ToJSON()
			if got != tt.want {
				t.Errorf("ToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToolRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewToolRegistry()

	tool1 := &MockTool{name: "duplicate", description: "First", schema: map[string]interface{}{}}
	tool2 := &MockTool{name: "duplicate", description: "Second", schema: map[string]interface{}{}}

	registry.Register(tool1)
	registry.Register(tool2) // Should overwrite

	retrieved, _ := registry.Get("duplicate")
	if retrieved.Description() != "Second" {
		t.Errorf("Expected second tool to overwrite first, got description '%s'", retrieved.Description())
	}
}
