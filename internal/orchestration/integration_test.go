package orchestration

import (
	"os"
	"testing"

	"innominatus/internal/providers"
)

func TestDatabaseTeamProviderIntegration(t *testing.T) {
	// Skip if providers directory doesn't exist
	if _, err := os.Stat("../../providers/database-team/provider.yaml"); os.IsNotExist(err) {
		t.Skip("Providers directory not available")
	}

	// Load database-team provider
	loader := providers.NewLoader("1.0.0")
	provider, err := loader.LoadFromFile("../../providers/database-team/provider.yaml")
	if err != nil {
		t.Fatalf("Failed to load database-team provider: %v", err)
	}

	// Validate provider
	if err := provider.Validate(); err != nil {
		t.Fatalf("Provider validation failed: %v", err)
	}

	// Test capabilities
	t.Run("has postgres capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("postgres") {
			t.Error("Provider should be able to provision postgres")
		}
	})

	t.Run("has postgresql capability", func(t *testing.T) {
		if !provider.CanProvisionResourceType("postgresql") {
			t.Error("Provider should be able to provision postgresql")
		}
	})

	t.Run("has provisioner workflow", func(t *testing.T) {
		workflow := provider.GetProvisionerWorkflow()
		if workflow == nil {
			t.Fatal("No provisioner workflow found")
		}
		if workflow.Name != "provision-postgres" {
			t.Errorf("Expected provision-postgres, got %s", workflow.Name)
		}
	})

	// Test with resolver
	t.Run("resolver can find provider for postgres", func(t *testing.T) {
		registry := providers.NewRegistry()
		err := registry.RegisterProvider(provider)
		if err != nil {
			t.Fatalf("Failed to register provider: %v", err)
		}

		resolver := NewResolver(registry)
		foundProvider, foundWorkflow, err := resolver.ResolveProviderForResource("postgres")
		if err != nil {
			t.Fatalf("Failed to resolve: %v", err)
		}

		if foundProvider.Metadata.Name != "database-team" {
			t.Errorf("Expected database-team, got %s", foundProvider.Metadata.Name)
		}

		if foundWorkflow.Name != "provision-postgres" {
			t.Errorf("Expected provision-postgres, got %s", foundWorkflow.Name)
		}
	})
}

func TestAllProviderCapabilitiesValid(t *testing.T) {
	// Skip if providers directory doesn't exist
	if _, err := os.Stat("../../providers"); os.IsNotExist(err) {
		t.Skip("Providers directory not available")
	}

	loader := providers.NewLoader("1.0.0")
	allProviders, err := loader.LoadFromDirectory("../../providers")
	if err != nil {
		t.Fatalf("Failed to load providers: %v", err)
	}

	if len(allProviders) == 0 {
		t.Skip("No providers found")
	}

	// Create registry and check for conflicts
	registry := providers.NewRegistry()
	for _, p := range allProviders {
		if err := registry.RegisterProvider(p); err != nil {
			t.Logf("Warning: Failed to register provider %s: %v", p.Metadata.Name, err)
		}
	}

	// Validate no capability conflicts
	resolver := NewResolver(registry)
	if err := resolver.ValidateProviders(); err != nil {
		t.Fatalf("Provider capability conflicts detected: %v", err)
	}

	t.Logf("Successfully validated %d providers with no conflicts", len(allProviders))
}
