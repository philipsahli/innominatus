package orchestration

import (
	"testing"

	"innominatus/internal/database"
	"innominatus/internal/types"
	"innominatus/pkg/sdk"
)

func TestBuildWorkflowInputs(t *testing.T) {
	engine := &Engine{}

	resource := &database.ResourceInstance{
		ApplicationName: "my-app",
		ResourceName:    "my-db",
		ResourceType:    "postgres",
		Configuration: map[string]interface{}{
			"size":     "small",
			"version":  "15",
			"replicas": 3,
		},
	}

	workflow := &types.Workflow{
		Variables: map[string]string{
			"default_version": "14",
			"environment":     "production",
		},
	}

	inputs := engine.buildWorkflowInputs(resource, workflow)

	// Check resource metadata is included
	if inputs["app_name"] != "my-app" {
		t.Errorf("Expected app_name=my-app, got %s", inputs["app_name"])
	}
	if inputs["resource_name"] != "my-db" {
		t.Errorf("Expected resource_name=my-db, got %s", inputs["resource_name"])
	}
	if inputs["resource_type"] != "postgres" {
		t.Errorf("Expected resource_type=postgres, got %s", inputs["resource_type"])
	}

	// Check resource configuration is included
	if inputs["size"] != "small" {
		t.Errorf("Expected size=small, got %s", inputs["size"])
	}
	if inputs["version"] != "15" {
		t.Errorf("Expected version=15, got %s", inputs["version"])
	}
	if inputs["replicas"] != "3" {
		t.Errorf("Expected replicas=3, got %s", inputs["replicas"])
	}

	// Check workflow defaults are included (but don't override resource config)
	if inputs["environment"] != "production" {
		t.Errorf("Expected environment=production, got %s", inputs["environment"])
	}

	// default_version should not override resource version
	if inputs["default_version"] != "14" {
		t.Errorf("Expected default_version=14, got %s", inputs["default_version"])
	}
}

func TestLoadWorkflowFromProvider(t *testing.T) {
	// This is an integration test that would require actual filesystem access
	// For now, we'll test the error cases

	engine := &Engine{
		providersDir: "/nonexistent",
	}

	provider := &sdk.Provider{
		Metadata: sdk.ProviderMetadata{
			Name: "test-provider",
		},
	}

	workflowMeta := &sdk.WorkflowMetadata{
		Name: "test-workflow",
		File: "./workflows/test.yaml",
	}

	// This should fail because the file doesn't exist
	_, err := engine.loadWorkflowFromProvider(provider, workflowMeta)
	if err == nil {
		t.Error("Expected error for nonexistent workflow file")
	}
}

func TestEngineCreation(t *testing.T) {
	engine := NewEngine(
		nil, // db
		nil, // registry
		nil, // workflowRepo
		nil, // resourceRepo
		nil, // workflowExec
		nil, // graphAdapter
		"/tmp/providers",
	)

	if engine == nil {
		t.Fatal("Expected engine to be created")
	}

	if engine.providersDir != "/tmp/providers" {
		t.Errorf("Expected providersDir=/tmp/providers, got %s", engine.providersDir)
	}

	if engine.pollInterval.Seconds() != 5 {
		t.Errorf("Expected poll interval 5s, got %v", engine.pollInterval)
	}

	if engine.resolver == nil {
		t.Error("Expected resolver to be initialized")
	}

	if engine.logger == nil {
		t.Error("Expected logger to be initialized")
	}
}
