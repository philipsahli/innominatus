package orchestration

import (
	"os"
	"testing"

	"innominatus/internal/providers"
)

func TestContainerTeamProviderIntegration(t *testing.T) {
	// Skip if providers directory doesn't exist
	if _, err := os.Stat("../../providers/container-team/provider.yaml"); os.IsNotExist(err) {
		t.Skip("Providers directory not available")
	}

	// Load container-team provider
	loader := providers.NewLoader("1.0.0")
	provider, err := loader.LoadFromFile("../../providers/container-team/provider.yaml")
	if err != nil {
		t.Fatalf("Failed to load container-team provider: %v", err)
	}

	// Validate provider
	if err := provider.Validate(); err != nil {
		t.Fatalf("Provider validation failed: %v", err)
	}

	// Test capabilities
	t.Run("has container capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("container") {
			t.Error("Provider should be able to provision container")
		}
	})

	t.Run("has application capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("application") {
			t.Error("Provider should be able to provision application")
		}
	})

	t.Run("has namespace capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("namespace") {
			t.Error("Provider should be able to provision namespace")
		}
	})

	t.Run("has gitea-repo capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("gitea-repo") {
			t.Error("Provider should be able to provision gitea-repo")
		}
	})

	t.Run("has argocd-app capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("argocd-app") {
			t.Error("Provider should be able to provision argocd-app")
		}
	})

	t.Run("has provisioner workflow", func(t *testing.T) {
		workflow := provider.GetProvisionerWorkflow()
		if workflow == nil {
			t.Fatal("No provisioner workflow found")
		}
		if workflow.Name != "provision-container" {
			t.Errorf("Expected provision-container, got %s", workflow.Name)
		}
	})

	// Test with resolver
	t.Run("resolver can find provider for container", func(t *testing.T) {
		registry := providers.NewRegistry()
		err := registry.RegisterProvider(provider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		resolver := NewResolver(registry)
		foundProvider, foundWorkflow, err := resolver.ResolveProviderForResource("container")
		if err != nil {
			t.Fatalf("Failed to resolve: %v", err)
		}

		if foundProvider.Metadata.Name != "container-team" {
			t.Errorf("Expected container-team, got %s", foundProvider.Metadata.Name)
		}

		if foundWorkflow.Name != "provision-container" {
			t.Errorf("Expected provision-container, got %s", foundWorkflow.Name)
		}
	})

	t.Run("resolver can find provider for application alias", func(t *testing.T) {
		registry := providers.NewRegistry()
		err := registry.RegisterProvider(provider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		resolver := NewResolver(registry)
		foundProvider, foundWorkflow, err := resolver.ResolveProviderForResource("application")
		if err != nil {
			t.Fatalf("Failed to resolve: %v", err)
		}

		if foundProvider.Metadata.Name != "container-team" {
			t.Errorf("Expected container-team, got %s", foundProvider.Metadata.Name)
		}

		if foundWorkflow.Name != "provision-container" {
			t.Errorf("Expected provision-container, got %s", foundWorkflow.Name)
		}
	})
}
