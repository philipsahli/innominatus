package database_team

import (
	"os"
	"testing"

	"innominatus/internal/providers"

	"gopkg.in/yaml.v3"
)

func TestDatabaseTeamProviderValid(t *testing.T) {
	// Load provider manifest
	data, err := os.ReadFile("provider.yaml")
	if err != nil {
		t.Fatalf("Failed to read provider.yaml: %v", err)
	}

	// Parse YAML
	var provider map[string]interface{}
	if err := yaml.Unmarshal(data, &provider); err != nil {
		t.Fatalf("Failed to parse provider.yaml: %v", err)
	}

	// Check required fields
	if provider["apiVersion"] == nil {
		t.Error("Missing apiVersion")
	}
	if provider["kind"] == nil {
		t.Error("Missing kind")
	}

	metadata := provider["metadata"].(map[string]interface{})
	if metadata["name"] != "database-team" {
		t.Errorf("Expected name=database-team, got %v", metadata["name"])
	}

	// Check capabilities
	capabilities := provider["capabilities"].(map[string]interface{})
	resourceTypes := capabilities["resourceTypes"].([]interface{})

	found := false
	for _, rt := range resourceTypes {
		if rt == "postgres" || rt == "postgresql" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected postgres capability not found")
	}

	// Check workflows
	workflows := provider["workflows"].([]interface{})
	if len(workflows) == 0 {
		t.Fatal("No workflows defined")
	}

	workflow := workflows[0].(map[string]interface{})
	if workflow["name"] != "provision-postgres" {
		t.Errorf("Expected workflow name=provision-postgres, got %v", workflow["name"])
	}
	if workflow["category"] != "provisioner" {
		t.Errorf("Expected category=provisioner, got %v", workflow["category"])
	}
}

func TestDatabaseTeamProviderLoadsWithSDK(t *testing.T) {
	loader := providers.NewLoader("1.0.0")

	provider, err := loader.LoadFromFile("provider.yaml")
	if err != nil {
		t.Fatalf("Failed to load provider with SDK: %v", err)
	}

	// Validate provider
	if err := provider.Validate(); err != nil {
		t.Fatalf("Provider validation failed: %v", err)
	}

	// Check capabilities
	if !provider.CanProvisionResourceType("postgres") {
		t.Error("Provider should be able to provision postgres")
	}
	if !provider.CanProvisionResourceType("postgresql") {
		t.Error("Provider should be able to provision postgresql")
	}

	// Check workflow
	workflow := provider.GetProvisionerWorkflow()
	if workflow == nil {
		t.Fatal("No provisioner workflow found")
	}
	if workflow.Name != "provision-postgres" {
		t.Errorf("Expected provision-postgres workflow, got %s", workflow.Name)
	}
}

func TestPostgresWorkflowExists(t *testing.T) {
	// Check workflow file exists
	if _, err := os.Stat("workflows/provision-postgres.yaml"); os.IsNotExist(err) {
		t.Fatal("provision-postgres.yaml workflow file not found")
	}

	// Read and validate it's valid YAML
	data, err := os.ReadFile("workflows/provision-postgres.yaml")
	if err != nil {
		t.Fatalf("Failed to read workflow: %v", err)
	}

	var workflow map[string]interface{}
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		t.Fatalf("Failed to parse workflow YAML: %v", err)
	}

	// Check required workflow fields
	if workflow["apiVersion"] == nil {
		t.Error("Workflow missing apiVersion")
	}
	if workflow["kind"] != "Workflow" {
		t.Errorf("Expected kind=Workflow, got %v", workflow["kind"])
	}
	if workflow["steps"] == nil {
		t.Error("Workflow missing steps")
	}
}
