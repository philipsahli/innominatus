package orchestration

import (
	"testing"

	"innominatus/internal/providers"
	"innominatus/pkg/sdk"
)

func TestResolverResolveProviderForResource(t *testing.T) {
	// Create registry with test providers
	registry := providers.NewRegistry()

	// Create database-team provider with postgres capability
	dbProvider := &sdk.Provider{
		APIVersion: "v1",
		Kind:       "Provider",
		Metadata: sdk.ProviderMetadata{
			Name:    "database-team",
			Version: "1.0.0",
		},
		Capabilities: sdk.ProviderCapabilities{
			ResourceTypes: []string{"postgres", "postgresql"},
		},
		Workflows: []sdk.WorkflowMetadata{
			{
				Name:     "provision-postgres",
				File:     "./workflows/provision-postgres.yaml",
				Category: "provisioner",
			},
		},
	}

	err := registry.RegisterProvider(dbProvider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Create resolver
	resolver := NewResolver(registry)

	tests := []struct {
		name         string
		resourceType string
		wantProvider string
		wantWorkflow string
		wantError    bool
	}{
		{
			name:         "resolve postgres to database-team",
			resourceType: "postgres",
			wantProvider: "database-team",
			wantWorkflow: "provision-postgres",
			wantError:    false,
		},
		{
			name:         "resolve postgresql alias",
			resourceType: "postgresql",
			wantProvider: "database-team",
			wantWorkflow: "provision-postgres",
			wantError:    false,
		},
		{
			name:         "unknown resource type",
			resourceType: "unknown-type",
			wantProvider: "",
			wantWorkflow: "",
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, workflow, err := resolver.ResolveProviderForResource(tt.resourceType)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if provider.Metadata.Name != tt.wantProvider {
				t.Errorf("Got provider %s, want %s", provider.Metadata.Name, tt.wantProvider)
			}

			if workflow.Name != tt.wantWorkflow {
				t.Errorf("Got workflow %s, want %s", workflow.Name, tt.wantWorkflow)
			}
		})
	}
}

func TestResolverValidateProviders(t *testing.T) {
	tests := []struct {
		name      string
		providers []*sdk.Provider
		wantError bool
	}{
		{
			name: "no conflicts",
			providers: []*sdk.Provider{
				{
					Metadata: sdk.ProviderMetadata{Name: "database-team"},
					Capabilities: sdk.ProviderCapabilities{
						ResourceTypes: []string{"postgres"},
					},
					Workflows: []sdk.WorkflowMetadata{{Name: "provision-postgres", Category: "provisioner"}},
				},
				{
					Metadata: sdk.ProviderMetadata{Name: "storage-team"},
					Capabilities: sdk.ProviderCapabilities{
						ResourceTypes: []string{"s3"},
					},
					Workflows: []sdk.WorkflowMetadata{{Name: "provision-s3", Category: "provisioner"}},
				},
			},
			wantError: false,
		},
		{
			name: "capability conflict",
			providers: []*sdk.Provider{
				{
					Metadata: sdk.ProviderMetadata{Name: "database-team"},
					Capabilities: sdk.ProviderCapabilities{
						ResourceTypes: []string{"postgres"},
					},
					Workflows: []sdk.WorkflowMetadata{{Name: "provision-postgres", Category: "provisioner"}},
				},
				{
					Metadata: sdk.ProviderMetadata{Name: "backup-team"},
					Capabilities: sdk.ProviderCapabilities{
						ResourceTypes: []string{"postgres"}, // Conflict!
					},
					Workflows: []sdk.WorkflowMetadata{{Name: "backup-postgres", Category: "provisioner"}},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := providers.NewRegistry()

			for _, p := range tt.providers {
				p.APIVersion = "v1"
				p.Kind = "Provider"
				if err := registry.RegisterProvider(p); err != nil {
					t.Fatalf("Failed to register provider: %v", err)
				}
			}

			resolver := NewResolver(registry)
			err := resolver.ValidateProviders()

			if tt.wantError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.wantError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestProviderCanProvisionResourceType(t *testing.T) {
	provider := &sdk.Provider{
		Capabilities: sdk.ProviderCapabilities{
			ResourceTypes: []string{"postgres", "mysql", "mongodb"},
		},
	}

	tests := []struct {
		resourceType string
		want         bool
	}{
		{"postgres", true},
		{"mysql", true},
		{"mongodb", true},
		{"redis", false},
		{"s3", false},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			got := provider.CanProvisionResourceType(tt.resourceType)
			if got != tt.want {
				t.Errorf("CanProvisionResourceType(%s) = %v, want %v", tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestProviderGetProvisionerWorkflow(t *testing.T) {
	tests := []struct {
		name      string
		workflows []sdk.WorkflowMetadata
		want      string
		wantNil   bool
	}{
		{
			name: "returns first provisioner workflow",
			workflows: []sdk.WorkflowMetadata{
				{Name: "provision-db", Category: "provisioner"},
				{Name: "backup-db", Category: "goldenpath"},
			},
			want:    "provision-db",
			wantNil: false,
		},
		{
			name: "returns workflow with empty category",
			workflows: []sdk.WorkflowMetadata{
				{Name: "provision-db", Category: ""},
			},
			want:    "provision-db",
			wantNil: false,
		},
		{
			name: "no provisioner workflows",
			workflows: []sdk.WorkflowMetadata{
				{Name: "backup-db", Category: "goldenpath"},
			},
			want:    "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &sdk.Provider{
				Workflows: tt.workflows,
			}

			workflow := provider.GetProvisionerWorkflow()

			if tt.wantNil {
				if workflow != nil {
					t.Errorf("Expected nil workflow, got %s", workflow.Name)
				}
				return
			}

			if workflow == nil {
				t.Errorf("Expected workflow, got nil")
				return
			}

			if workflow.Name != tt.want {
				t.Errorf("Got workflow %s, want %s", workflow.Name, tt.want)
			}
		})
	}
}
