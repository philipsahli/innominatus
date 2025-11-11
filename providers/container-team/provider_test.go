package container_team

import (
	"os"
	"testing"

	"innominatus/internal/providers"

	"gopkg.in/yaml.v3"
)

func TestContainerTeamProviderValid(t *testing.T) {
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
	if metadata["name"] != "container-team" {
		t.Errorf("Expected name=container-team, got %v", metadata["name"])
	}

	// Check capabilities
	capabilities := provider["capabilities"].(map[string]interface{})
	resourceTypes := capabilities["resourceTypes"].([]interface{})

	expectedTypes := map[string]bool{
		"container":            false,
		"application":          false,
		"namespace":            false,
		"kubernetes-namespace": false,
		"gitea-repo":           false,
		"git-repository":       false,
		"argocd-app":           false,
		"argocd-application":   false,
	}

	for _, rt := range resourceTypes {
		rtStr := rt.(string)
		if _, expected := expectedTypes[rtStr]; expected {
			expectedTypes[rtStr] = true
		}
	}

	for expectedType, found := range expectedTypes {
		if !found {
			t.Errorf("Expected capability %s not found", expectedType)
		}
	}

	// Check workflows
	workflows := provider["workflows"].([]interface{})
	if len(workflows) == 0 {
		t.Fatal("No workflows defined")
	}

	// Check provision-container workflow exists
	foundContainerWorkflow := false
	for _, wf := range workflows {
		workflow := wf.(map[string]interface{})
		if workflow["name"] == "provision-container" {
			foundContainerWorkflow = true
			if workflow["category"] != "provisioner" {
				t.Errorf("Expected provision-container category=provisioner, got %v", workflow["category"])
			}
		}
	}

	if !foundContainerWorkflow {
		t.Error("provision-container workflow not found")
	}
}

func TestContainerTeamProviderLoadsWithSDK(t *testing.T) {
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
	if !provider.CanProvisionResourceType("container") {
		t.Error("Provider should be able to provision container")
	}
	if !provider.CanProvisionResourceType("application") {
		t.Error("Provider should be able to provision application")
	}
	if !provider.CanProvisionResourceType("namespace") {
		t.Error("Provider should be able to provision namespace")
	}
	if !provider.CanProvisionResourceType("gitea-repo") {
		t.Error("Provider should be able to provision gitea-repo")
	}
	if !provider.CanProvisionResourceType("argocd-app") {
		t.Error("Provider should be able to provision argocd-app")
	}

	// Check workflow
	workflow := provider.GetProvisionerWorkflow()
	if workflow == nil {
		t.Fatal("No provisioner workflow found")
	}
	if workflow.Name != "provision-container" {
		t.Errorf("Expected provision-container workflow, got %s", workflow.Name)
	}
}

func TestProvisionContainerWorkflowExists(t *testing.T) {
	// Check workflow file exists
	if _, err := os.Stat("workflows/provision-container.yaml"); os.IsNotExist(err) {
		t.Fatal("provision-container.yaml workflow file not found")
	}

	// Read and validate it's valid YAML
	data, err := os.ReadFile("workflows/provision-container.yaml")
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

	// Check metadata
	metadata := workflow["metadata"].(map[string]interface{})
	if metadata["name"] != "provision-container" {
		t.Errorf("Expected name=provision-container, got %v", metadata["name"])
	}

	// Check steps exist
	if workflow["steps"] == nil {
		t.Error("Workflow missing steps")
	}

	steps := workflow["steps"].([]interface{})
	if len(steps) < 4 {
		t.Errorf("Expected at least 4 steps, got %d", len(steps))
	}

	// Verify step names
	expectedSteps := map[string]bool{
		"create-namespace":   false,
		"create-git-repo":    false,
		"generate-manifests": false,
		"create-argocd-app":  false,
	}

	for _, s := range steps {
		step := s.(map[string]interface{})
		stepName := step["name"].(string)
		if _, expected := expectedSteps[stepName]; expected {
			expectedSteps[stepName] = true
		}
	}

	for expectedStep, found := range expectedSteps {
		if !found {
			t.Errorf("Expected step %s not found", expectedStep)
		}
	}

	// Check outputs
	if workflow["outputs"] == nil {
		t.Error("Workflow missing outputs")
	}
}

func TestOtherWorkflowsExist(t *testing.T) {
	workflows := []string{
		"workflows/provision-namespace.yaml",
		"workflows/provision-gitea-repo.yaml",
		"workflows/provision-argocd-app.yaml",
	}

	for _, wf := range workflows {
		if _, err := os.Stat(wf); os.IsNotExist(err) {
			t.Errorf("Workflow file not found: %s", wf)
		}
	}
}
