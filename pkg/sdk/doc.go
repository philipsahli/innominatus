// Package sdk provides the public API for building innominatus platform extensions.
//
// # Overview
//
// The SDK enables platform teams to create custom resource provisioners, workflows,
// and golden paths that integrate with the innominatus orchestration engine.
//
// Platform implementations are versioned independently using Semantic Versioning
// and distributed as separate packages, allowing enterprises to:
//   - Create proprietary platforms for internal systems
//   - Version platform logic independently of core engine
//   - Test platform implementations in isolation
//   - Share platforms across teams and organizations
//
// # Core Interfaces
//
// The SDK defines several key interfaces that platforms must implement:
//
//   - Provisioner: Resource provisioning and lifecycle management
//   - Config: Type-safe configuration access
//   - Resource: Resource instance representation
//   - Hint: Contextual quick-access links and commands
//
// # Creating a Platform
//
// To create a custom platform:
//
//  1. Create a platform.yaml manifest describing your platform
//  2. Implement the Provisioner interface for each resource type
//  3. Package your platform as a Go module or OCI artifact
//  4. Register your platform with innominatus
//
// # Example Platform
//
// Here's a minimal platform implementation:
//
//	// platform.yaml
//	apiVersion: innominatus.io/v1
//	kind: Platform
//	metadata:
//	  name: my-platform
//	  version: 1.0.0
//	  description: Custom platform for My Company
//	compatibility:
//	  minCoreVersion: "1.0.0"
//	  maxCoreVersion: "2.0.0"
//	provisioners:
//	  - name: my-database
//	    type: postgres
//	    version: 1.0.0
//	    description: Provisions company-specific PostgreSQL databases
//
//	// provisioners/database.go
//	package provisioners
//
//	import (
//	    "context"
//	    "github.com/innominatus/innominatus-core/pkg/sdk"
//	)
//
//	type DatabaseProvisioner struct{}
//
//	func (p *DatabaseProvisioner) Name() string    { return "my-database" }
//	func (p *DatabaseProvisioner) Type() string    { return "postgres" }
//	func (p *DatabaseProvisioner) Version() string { return "1.0.0" }
//
//	func (p *DatabaseProvisioner) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
//	    // Implement provisioning logic
//	    dbName := config.GetString("name")
//	    size := config.GetString("size")
//
//	    // Call your platform's API to create the database
//	    // ...
//
//	    return nil
//	}
//
//	func (p *DatabaseProvisioner) Deprovision(ctx context.Context, resource *sdk.Resource) error {
//	    // Implement deprovisioning logic
//	    return nil
//	}
//
//	func (p *DatabaseProvisioner) GetStatus(ctx context.Context, resource *sdk.Resource) (*sdk.ResourceStatus, error) {
//	    // Return current resource status
//	    return &sdk.ResourceStatus{
//	        State:        sdk.ResourceStateActive,
//	        HealthStatus: "healthy",
//	    }, nil
//	}
//
//	func (p *DatabaseProvisioner) GetHints(ctx context.Context, resource *sdk.Resource) ([]sdk.Hint, error) {
//	    // Return contextual hints
//	    return []sdk.Hint{
//	        sdk.NewURLHint("Admin Console", "https://db.example.com/admin", sdk.IconDatabase),
//	        sdk.NewConnectionStringHint("Connection String", "postgres://user:pass@host/db"),
//	    }, nil
//	}
//
// # Versioning
//
// Platforms follow Semantic Versioning (SemVer):
//
//   - MAJOR version for incompatible API changes
//   - MINOR version for backward-compatible functionality
//   - PATCH version for backward-compatible bug fixes
//
// Example version progression:
//
//	v0.1.0 - Initial development release
//	v0.5.0 - Beta testing with early adopters
//	v1.0.0 - First stable release
//	v1.1.0 - Add new resource type (backward compatible)
//	v1.1.1 - Bug fix in existing provisioner
//	v2.0.0 - Breaking change (e.g., new authentication mechanism)
//
// # Testing
//
// Platform implementations should include:
//
//   - Unit tests for provisioner logic
//   - Integration tests against target platform
//   - Contract tests verifying SDK interface compliance
//
// # Best Practices
//
//   - Keep provisioners stateless; core manages resource lifecycle
//   - Use structured error messages with sdk.SDKError
//   - Provide meaningful hints for user experience
//   - Version your platform independently
//   - Document configuration parameters
//   - Test against multiple core versions
//
// # For More Information
//
// See the full documentation at:
// https://github.com/innominatus/innominatus-core/docs/platform-development-guide.md
package sdk
