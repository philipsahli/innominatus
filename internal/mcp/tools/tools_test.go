package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test ListGoldenPathsTool
func TestListGoldenPathsTool_Execute(t *testing.T) {
	mockResponse := `[
		{
			"name": "database-team",
			"workflows": [
				{"name": "provision-postgres", "category": "goldenpath"},
				{"name": "backup-db", "category": "provisioner"}
			]
		},
		{
			"name": "container-team",
			"workflows": [
				{"name": "onboard-team", "category": "goldenpath"}
			]
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/providers" {
			t.Errorf("Expected path /api/providers, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "test-token")
	tool := NewListGoldenPathsTool(client)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "provision-postgres") {
		t.Errorf("Expected result to contain 'provision-postgres', got %s", result)
	}

	if !strings.Contains(result, "onboard-team") {
		t.Errorf("Expected result to contain 'onboard-team', got %s", result)
	}

	// Should NOT contain non-goldenpath workflows
	if strings.Contains(result, "backup-db") {
		t.Errorf("Should not contain non-goldenpath workflow 'backup-db'")
	}
}

// Test ListProvidersTool
func TestListProvidersTool_Execute(t *testing.T) {
	mockResponse := `[
		{
			"name": "database-team",
			"capabilities": {
				"resourceTypes": ["postgres", "mysql"]
			}
		}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "test-token")
	tool := NewListProvidersTool(client)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "database-team") {
		t.Errorf("Expected result to contain 'database-team'")
	}
}

// Test GetProviderDetailsTool
func TestGetProviderDetailsTool_Execute(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{
			name:      "valid provider name",
			input:     map[string]interface{}{"name": "database-team"},
			wantError: false,
		},
		{
			name:      "missing name parameter",
			input:     map[string]interface{}{},
			wantError: true,
		},
		{
			name:      "invalid name type",
			input:     map[string]interface{}{"name": 123},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"name":"database-team","version":"1.0.0"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewGetProviderDetailsTool(client)

			_, err := tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// Test ExecuteWorkflowTool
func TestExecuteWorkflowTool_Execute(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{
			name: "valid workflow execution",
			input: map[string]interface{}{
				"workflow_name": "provision-postgres",
				"inputs":        map[string]interface{}{"namespace": "test"},
			},
			wantError: false,
		},
		{
			name:      "missing workflow_name",
			input:     map[string]interface{}{"inputs": map[string]interface{}{}},
			wantError: true,
		},
		{
			name:      "missing inputs",
			input:     map[string]interface{}{"workflow_name": "test"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/workflows/execute" {
					t.Errorf("Expected path /api/workflows/execute, got %s", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"execution_id":"123","status":"running"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewExecuteWorkflowTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && !strings.Contains(result, "123") {
				t.Errorf("Expected result to contain execution_id '123'")
			}
		})
	}
}

// Test GetWorkflowStatusTool
func TestGetWorkflowStatusTool_Execute(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
		wantPath  string
	}{
		{
			name:      "valid execution id",
			input:     map[string]interface{}{"execution_id": "exec-123"},
			wantError: false,
			wantPath:  "/api/workflows/exec-123",
		},
		{
			name:      "missing execution_id",
			input:     map[string]interface{}{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !tt.wantError && r.URL.Path != tt.wantPath {
					t.Errorf("Expected path %s, got %s", tt.wantPath, r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"exec-123","status":"completed"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewGetWorkflowStatusTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && !strings.Contains(result, "completed") {
				t.Errorf("Expected result to contain 'completed'")
			}
		})
	}
}

// Test ListWorkflowExecutionsTool
func TestListWorkflowExecutionsTool_Execute(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		expectedPath string
	}{
		{
			name:         "default limit",
			input:        map[string]interface{}{},
			expectedPath: "/api/workflows?limit=10",
		},
		{
			name:         "custom limit",
			input:        map[string]interface{}{"limit": 25},
			expectedPath: "/api/workflows?limit=25",
		},
		{
			name:         "limit as string",
			input:        map[string]interface{}{"limit": "15"},
			expectedPath: "/api/workflows?limit=15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path+"?"+r.URL.RawQuery != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, r.URL.Path+"?"+r.URL.RawQuery)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[{"id":"1"},{"id":"2"}]`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewListWorkflowExecutionsTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			if !strings.Contains(result, "\"id\":\"1\"") {
				t.Errorf("Expected result to contain workflow executions")
			}
		})
	}
}

// Test ListResourcesTool
func TestListResourcesTool_Execute(t *testing.T) {
	tests := []struct {
		name         string
		input        map[string]interface{}
		expectedPath string
	}{
		{
			name:         "no filter",
			input:        map[string]interface{}{},
			expectedPath: "/api/resources",
		},
		{
			name:         "with type filter",
			input:        map[string]interface{}{"type": "postgres"},
			expectedPath: "/api/resources?type=postgres",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedURL := tt.expectedPath
				actualURL := r.URL.Path
				if r.URL.RawQuery != "" {
					actualURL += "?" + r.URL.RawQuery
				}

				if actualURL != expectedURL {
					t.Errorf("Expected URL %s, got %s", expectedURL, actualURL)
				}

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[{"id":"res-1","type":"postgres"}]`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewListResourcesTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if err != nil {
				t.Errorf("Execute() error = %v", err)
			}

			if !strings.Contains(result, "res-1") {
				t.Errorf("Expected result to contain resource")
			}
		})
	}
}

// Test GetResourceDetailsTool
func TestGetResourceDetailsTool_Execute(t *testing.T) {
	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{
			name:      "valid resource id",
			input:     map[string]interface{}{"resource_id": "res-123"},
			wantError: false,
		},
		{
			name:      "missing resource_id",
			input:     map[string]interface{}{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"res-123","type":"postgres","state":"active"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewGetResourceDetailsTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && !strings.Contains(result, "active") {
				t.Errorf("Expected result to contain 'active'")
			}
		})
	}
}

// Test ListSpecsTool
func TestListSpecsTool_Execute(t *testing.T) {
	mockResponse := `[
		{"name":"app1","status":"running"},
		{"name":"app2","status":"stopped"}
	]`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/specs" {
			t.Errorf("Expected path /api/specs, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewAPIClient(server.URL, "test-token")
	tool := NewListSpecsTool(client)

	result, err := tool.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(result, "app1") || !strings.Contains(result, "app2") {
		t.Errorf("Expected result to contain both apps")
	}
}

// Test SubmitSpecTool
func TestSubmitSpecTool_Execute(t *testing.T) {
	yamlSpec := `
apiVersion: score.dev/v1b1
metadata:
  name: test-app
`

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantError bool
	}{
		{
			name:      "valid spec",
			input:     map[string]interface{}{"spec": yamlSpec},
			wantError: false,
		},
		{
			name:      "missing spec",
			input:     map[string]interface{}{},
			wantError: true,
		},
		{
			name:      "invalid spec type",
			input:     map[string]interface{}{"spec": 123},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify content type is YAML
				if ct := r.Header.Get("Content-Type"); ct != "application/yaml" {
					t.Errorf("Expected Content-Type application/yaml, got %s", ct)
				}

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"status":"spec submitted","app_name":"test-app"}`))
			}))
			defer server.Close()

			client := NewAPIClient(server.URL, "test-token")
			tool := NewSubmitSpecTool(client)

			result, err := tool.Execute(context.Background(), tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Execute() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && !strings.Contains(result, "spec submitted") {
				t.Errorf("Expected result to contain 'spec submitted'")
			}
		})
	}
}

// Test BuildRegistry
func TestBuildRegistry(t *testing.T) {
	registry := BuildRegistry("http://localhost:8081", "test-token")

	expectedTools := []string{
		"list_golden_paths",
		"list_providers",
		"get_provider_details",
		"execute_workflow",
		"get_workflow_status",
		"list_workflow_executions",
		"list_resources",
		"get_resource_details",
		"list_specs",
		"submit_spec",
	}

	for _, toolName := range expectedTools {
		if _, ok := registry.Get(toolName); !ok {
			t.Errorf("Expected tool '%s' to be registered", toolName)
		}
	}

	allTools := registry.List()
	if len(allTools) != 10 {
		t.Errorf("Expected 10 tools, got %d", len(allTools))
	}
}
