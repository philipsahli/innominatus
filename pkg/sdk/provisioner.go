package sdk

import "context"

// Provisioner defines the interface that all resource provisioners must implement.
// Platform teams implement this interface to provide custom resource provisioning logic.
//
// Example:
//
//	type MyProvisioner struct {}
//
//	func (p *MyProvisioner) Provision(ctx context.Context, resource *Resource, config Config) error {
//	    // Provision resource using platform-specific logic
//	    return nil
//	}
type Provisioner interface {
	// Name returns the unique name of this provisioner
	// Example: "aws-rds", "gitea-repo", "azure-cosmosdb"
	Name() string

	// Type returns the resource type this provisioner handles
	// Example: "postgres", "gitea-repo", "redis"
	Type() string

	// Version returns the semantic version of this provisioner
	// Example: "1.2.3", "2.0.0-beta.1"
	Version() string

	// Provision creates a new resource instance
	// The provisioner should:
	// 1. Create the resource in the target platform
	// 2. Update resource.State to reflect progress
	// 3. Return error if provisioning fails
	Provision(ctx context.Context, resource *Resource, config Config) error

	// Deprovision removes an existing resource instance
	// The provisioner should:
	// 1. Delete the resource from the target platform
	// 2. Clean up any associated resources
	// 3. Return error if deprovisioning fails
	Deprovision(ctx context.Context, resource *Resource) error

	// GetStatus retrieves the current status of a resource
	// Returns ResourceStatus with state, health, and metadata
	GetStatus(ctx context.Context, resource *Resource) (*ResourceStatus, error)

	// GetHints returns contextual hints for the resource
	// Hints provide quick access links, commands, and connection strings
	// Returns empty slice if no hints available
	GetHints(ctx context.Context, resource *Resource) ([]Hint, error)
}

// ProvisionerMetadata contains metadata about a provisioner
// Used for platform manifest and discovery
type ProvisionerMetadata struct {
	// Name is the unique identifier for this provisioner
	Name string `yaml:"name" json:"name"`

	// Type is the resource type this provisioner handles
	Type string `yaml:"type" json:"type"`

	// Version is the semantic version of this provisioner
	Version string `yaml:"version" json:"version"`

	// Description provides a human-readable description
	Description string `yaml:"description,omitempty" json:"description,omitempty"`

	// Author identifies the maintainer of this provisioner
	Author string `yaml:"author,omitempty" json:"author,omitempty"`

	// Tags are searchable keywords for discovery
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}
