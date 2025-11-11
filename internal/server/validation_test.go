package server

import (
	"testing"

	"innominatus/internal/orchestration"
	"innominatus/internal/providers"
	"innominatus/internal/types"
	providersdk "innominatus/pkg/sdk"

	"github.com/stretchr/testify/assert"
)

// TestResourceTypeValidation tests the resource type validation logic
func TestResourceTypeValidation(t *testing.T) {
	// Create a test provider registry
	registry := providers.NewRegistry()

	// Create a test provider with postgres capability
	testProvider := &providersdk.Provider{
		Metadata: providersdk.ProviderMetadata{
			Name:     "test-provider",
			Version:  "1.0.0",
			Category: "data",
		},
		Capabilities: providersdk.ProviderCapabilities{
			ResourceTypes: []string{"postgres", "postgresql"},
		},
		Workflows: []providersdk.WorkflowMetadata{
			{
				Name:     "provision-postgres",
				Category: "provisioner",
			},
		},
	}

	// Register the provider
	err := registry.RegisterProvider(testProvider)
	assert.NoError(t, err)

	// Create resolver
	resolver := orchestration.NewResolver(registry)

	// Create server with resolver
	server := &Server{
		providerResolver: resolver,
	}

	t.Run("Valid resource type passes validation", func(t *testing.T) {
		spec := &types.ScoreSpec{
			Resources: map[string]types.Resource{
				"db": {
					Type: "postgres",
					Params: map[string]interface{}{
						"version": "15",
					},
				},
			},
		}

		err := server.validateResourceTypes(spec)
		assert.NoError(t, err)
	})

	t.Run("Unknown resource type fails validation", func(t *testing.T) {
		spec := &types.ScoreSpec{
			Resources: map[string]types.Resource{
				"unknown_db": {
					Type: "unknown-database",
					Params: map[string]interface{}{
						"version": "15",
					},
				},
			},
		}

		err := server.validateResourceTypes(spec)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown resource types")
		assert.Contains(t, err.Error(), "unknown_db")
		assert.Contains(t, err.Error(), "unknown-database")
	})

	t.Run("Multiple unknown resource types reported", func(t *testing.T) {
		spec := &types.ScoreSpec{
			Resources: map[string]types.Resource{
				"unknown_db": {
					Type: "unknown-database",
				},
				"mystery_cache": {
					Type: "mystery-cache",
				},
			},
		}

		err := server.validateResourceTypes(spec)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_db")
		assert.Contains(t, err.Error(), "mystery_cache")
	})

	t.Run("Resource type alias works", func(t *testing.T) {
		spec := &types.ScoreSpec{
			Resources: map[string]types.Resource{
				"db": {
					Type: "postgresql", // Alias for postgres
				},
			},
		}

		err := server.validateResourceTypes(spec)
		assert.NoError(t, err)
	})

	t.Run("No resolver skips validation", func(t *testing.T) {
		serverNoResolver := &Server{
			providerResolver: nil,
		}

		spec := &types.ScoreSpec{
			Resources: map[string]types.Resource{
				"unknown_db": {
					Type: "unknown-database",
				},
			},
		}

		err := serverNoResolver.validateResourceTypes(spec)
		assert.NoError(t, err) // Should pass when no resolver
	})
}
