# Platform Extension Guide

## Overview

This guide explains how to create custom platform extensions for innominatus, enabling you to add support for proprietary infrastructure, cloud providers, or custom resource types.

## What is a Platform?

A **platform** is a collection of provisioners that extend innominatus with support for specific infrastructure types. Platforms are:

- **Independent**: Developed, versioned, and tested separately from the core
- **Composable**: Multiple platforms can coexist in a single innominatus instance
- **Versioned**: Follow semantic versioning for clear compatibility
- **Distributable**: Shared as Go modules or OCI artifacts

## Architecture

```
innominatus-core (v1.x.x)
├── pkg/sdk/           # Public SDK for platform developers
├── internal/platform/ # Platform loader and registry
└── platforms/
    ├── builtin/       # Built-in platform (Gitea, K8s, ArgoCD)
    ├── aws/           # AWS platform (optional, separate repo)
    ├── azure/         # Azure platform (optional, separate repo)
    └── custom/        # Your custom platform
```

## Quick Start

### 1. Create Platform Repository

```bash
# Create new platform repository
mkdir innominatus-platform-mycompany
cd innominatus-platform-mycompany

# Initialize Go module
go mod init github.com/mycompany/innominatus-platform-mycompany

# Add SDK dependency
go get github.com/philipsahli/innominatus-core/pkg/sdk
```

### 2. Create Platform Manifest

Create `platform.yaml`:

```yaml
apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: mycompany-platform
  version: 1.0.0
  description: MyCompany internal platform for IDP
  author: Platform Team
  homepage: https://github.com/mycompany/innominatus-platform-mycompany
  license: MIT
  tags:
    - aws
    - database
    - storage

compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"

provisioners:
  - name: mycompany-database
    type: postgres
    version: 1.0.0
    description: Provisions managed PostgreSQL databases
    tags:
      - database
      - postgres
      - rds

  - name: mycompany-storage
    type: s3-bucket
    version: 1.0.0
    description: Provisions S3 buckets with company policies
    tags:
      - storage
      - s3
      - aws

configuration:
  # Platform-specific defaults
  aws_region: us-east-1
  default_tags:
    managed_by: innominatus
    team: platform
```

### 3. Implement Provisioner

Create `provisioners/database.go`:

```go
package provisioners

import (
	"context"
	"fmt"
	"github.com/philipsahli/innominatus-core/pkg/sdk"
)

type DatabaseProvisioner struct {
	// Add your dependencies (AWS SDK clients, etc.)
	awsClient *AWSClient
}

func NewDatabaseProvisioner(awsClient *AWSClient) *DatabaseProvisioner {
	return &DatabaseProvisioner{
		awsClient: awsClient,
	}
}

// Name returns the provisioner name
func (p *DatabaseProvisioner) Name() string {
	return "mycompany-database"
}

// Type returns the provisioner type
func (p *DatabaseProvisioner) Type() string {
	return "postgres"
}

// Version returns the provisioner version
func (p *DatabaseProvisioner) Version() string {
	return "1.0.0"
}

// Provision creates a new database
func (p *DatabaseProvisioner) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
	// Extract configuration
	dbName := config.GetString("name")
	dbSize := config.GetString("size")
	highAvailability := config.GetBool("ha")

	// Validate required parameters
	if dbName == "" {
		return sdk.ErrInvalidConfig("database name is required")
	}

	// Call AWS API to create database
	instance, err := p.awsClient.CreateDatabase(ctx, DatabaseRequest{
		Name:              dbName,
		InstanceClass:     dbSize,
		HighAvailability:  highAvailability,
		Engine:            "postgres",
		EngineVersion:     config.GetString("version"),
	})

	if err != nil {
		return sdk.ErrProvisionFailed("failed to create database: %v", err)
	}

	// Store provider ID for future operations
	resource.ProviderID = instance.ID

	return nil
}

// Deprovision deletes a database
func (p *DatabaseProvisioner) Deprovision(ctx context.Context, resource *sdk.Resource) error {
	if resource.ProviderID == "" {
		return sdk.ErrInvalidResource("provider ID is required")
	}

	if err := p.awsClient.DeleteDatabase(ctx, resource.ProviderID); err != nil {
		return sdk.ErrDeprovisionFailed("failed to delete database: %v", err)
	}

	return nil
}

// GetStatus returns the current status of the database
func (p *DatabaseProvisioner) GetStatus(ctx context.Context, resource *sdk.Resource) (*sdk.ResourceStatus, error) {
	if resource.ProviderID == "" {
		return nil, sdk.ErrInvalidResource("provider ID is required")
	}

	instance, err := p.awsClient.GetDatabase(ctx, resource.ProviderID)
	if err != nil {
		if IsNotFoundError(err) {
			return &sdk.ResourceStatus{
				State:        sdk.ResourceStateTerminated,
				HealthStatus: "not_found",
			}, nil
		}
		return nil, sdk.ErrStatusCheckFailed("failed to get database status: %v", err)
	}

	// Map AWS status to SDK state
	var state sdk.ResourceState
	var health string

	switch instance.Status {
	case "creating":
		state = sdk.ResourceStateProvisioning
		health = "provisioning"
	case "available":
		state = sdk.ResourceStateActive
		health = "healthy"
	case "backing-up":
		state = sdk.ResourceStateUpdating
		health = "healthy"
	case "deleting":
		state = sdk.ResourceStateTerminating
		health = "terminating"
	case "failed":
		state = sdk.ResourceStateFailed
		health = "failed"
	default:
		state = sdk.ResourceStateDegraded
		health = "unknown"
	}

	return &sdk.ResourceStatus{
		State:        state,
		HealthStatus: health,
		Message:      fmt.Sprintf("Database status: %s", instance.Status),
	}, nil
}

// GetHints returns contextual hints for the database
func (p *DatabaseProvisioner) GetHints(ctx context.Context, resource *sdk.Resource) ([]sdk.Hint, error) {
	if resource.ProviderID == "" {
		return []sdk.Hint{}, nil
	}

	instance, err := p.awsClient.GetDatabase(ctx, resource.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	hints := []sdk.Hint{
		sdk.NewURLHint(
			"AWS Console",
			fmt.Sprintf("https://console.aws.amazon.com/rds/home?region=us-east-1#database:id=%s", instance.ID),
			sdk.IconExternalLink,
		),
		sdk.NewConnectionStringHint(
			"Connection String",
			fmt.Sprintf("postgres://%s:%d/%s", instance.Endpoint, instance.Port, resource.ResourceName),
		),
		sdk.NewDashboardHint(
			"Monitoring Dashboard",
			fmt.Sprintf("https://monitoring.mycompany.com/rds/%s", instance.ID),
		),
	}

	return hints, nil
}
```

### 4. Create Tests

Create `provisioners/database_test.go`:

```go
package provisioners_test

import (
	"context"
	"testing"
	"github.com/philipsahli/innominatus-core/pkg/sdk"
	"github.com/mycompany/innominatus-platform-mycompany/provisioners"
)

func TestDatabaseProvision(t *testing.T) {
	// Create mock AWS client
	mockClient := NewMockAWSClient()

	// Create provisioner
	prov := provisioners.NewDatabaseProvisioner(mockClient)

	// Create test resource
	resource := &sdk.Resource{
		ID:              1,
		ApplicationName: "test-app",
		ResourceName:    "test-db",
		ResourceType:    "postgres",
		State:           sdk.ResourceStateRequested,
		Configuration:   sdk.NewMapConfig(map[string]interface{}{
			"name":    "test-db",
			"size":    "db.t3.small",
			"ha":      true,
			"version": "14.0",
		}),
	}

	// Test provision
	ctx := context.Background()
	err := prov.Provision(ctx, resource, resource.Configuration)
	if err != nil {
		t.Fatalf("Failed to provision: %v", err)
	}

	// Verify resource has provider ID
	if resource.ProviderID == "" {
		t.Error("Expected provider ID to be set")
	}

	// Test GetStatus
	status, err := prov.GetStatus(ctx, resource)
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if status.State != sdk.ResourceStateActive {
		t.Errorf("Expected state=Active, got %v", status.State)
	}

	// Test GetHints
	hints, err := prov.GetHints(ctx, resource)
	if err != nil {
		t.Fatalf("Failed to get hints: %v", err)
	}

	if len(hints) == 0 {
		t.Error("Expected at least one hint")
	}

	// Test Deprovision
	err = prov.Deprovision(ctx, resource)
	if err != nil {
		t.Fatalf("Failed to deprovision: %v", err)
	}
}
```

### 5. Register Platform

Create `main.go` (if distributing as standalone binary):

```go
package main

import (
	"fmt"
	"github.com/philipsahli/innominatus-core/pkg/sdk"
	"github.com/mycompany/innominatus-platform-mycompany/provisioners"
)

func main() {
	// Load platform manifest
	platform, err := sdk.LoadPlatformFromFile("platform.yaml")
	if err != nil {
		panic(fmt.Sprintf("Failed to load platform: %v", err))
	}

	fmt.Printf("Platform %s v%s loaded successfully\n",
		platform.Metadata.Name,
		platform.Metadata.Version)
}
```

### 6. Build and Distribute

```bash
# Run tests
go test ./...

# Build binary
go build -o platform-mycompany

# Create git tag for versioning
git tag v1.0.0
git push origin v1.0.0

# Or build as Go module (no binary needed)
go mod tidy
```

## Installation

### Option 1: Directory Installation

```bash
# Copy platform to innominatus platforms directory
cp -r innominatus-platform-mycompany /path/to/innominatus/platforms/mycompany/

# innominatus will automatically discover platform.yaml files
```

### Option 2: Go Module

```go
// Import platform in innominatus
import (
	mycompany "github.com/mycompany/innominatus-platform-mycompany"
)

// Register provisioners at runtime
registry.RegisterProvisioner(mycompany.NewDatabaseProvisioner(...))
```

### Option 3: OCI Artifact (Future)

```bash
# Pull platform from registry
innominatus-ctl platform install mycompany/platform:v1.0.0

# List installed platforms
innominatus-ctl platform list

# Update platform
innominatus-ctl platform update mycompany/platform:v1.1.0
```

## Configuration

Platforms can provide default configuration that can be overridden in `admin-config.yaml`:

```yaml
# admin-config.yaml
platforms:
  mycompany-platform:
    aws_region: us-west-2
    default_tags:
      environment: production
      cost_center: engineering
```

## Versioning Strategy

Follow Semantic Versioning (SemVer):

### Version Format

`MAJOR.MINOR.PATCH` (e.g., `v2.3.1`)

- **MAJOR**: Incompatible changes to provisioner interface
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Example Progression

```
v0.1.0  - Initial development
v0.5.0  - Beta testing
v1.0.0  - First stable release
v1.1.0  - Add S3 bucket provisioner (new feature)
v1.1.1  - Fix database connection string bug
v2.0.0  - Change provisioner config schema (breaking change)
```

### Compatibility Matrix

```yaml
# platform.yaml
compatibility:
  minCoreVersion: "1.0.0"  # Minimum innominatus core version
  maxCoreVersion: "2.0.0"  # Maximum innominatus core version
```

**Example:**
- Platform `v1.0.0` requires core `1.x.x`
- Platform `v2.0.0` requires core `2.x.x`
- Core `v1.5.0` can run platform `v1.0.0` to `v1.9.9`

## SDK Reference

### Core Interfaces

#### Provisioner Interface

```go
type Provisioner interface {
    Name() string
    Type() string
    Version() string
    Provision(ctx context.Context, resource *Resource, config Config) error
    Deprovision(ctx context.Context, resource *Resource) error
    GetStatus(ctx context.Context, resource *Resource) (*ResourceStatus, error)
    GetHints(ctx context.Context, resource *Resource) ([]Hint, error)
}
```

#### Config Interface

```go
type Config interface {
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetFloat(key string) float64
    GetMap(key string) map[string]interface{}
    GetSlice(key string) []interface{}
    Has(key string) bool
    Keys() []string
    AsMap() map[string]interface{}
}
```

### Resource States

```go
const (
    ResourceStateRequested    ResourceState = "requested"
    ResourceStateProvisioning ResourceState = "provisioning"
    ResourceStateActive       ResourceState = "active"
    ResourceStateScaling      ResourceState = "scaling"
    ResourceStateUpdating     ResourceState = "updating"
    ResourceStateDegraded     ResourceState = "degraded"
    ResourceStateTerminating  ResourceState = "terminating"
    ResourceStateTerminated   ResourceState = "terminated"
    ResourceStateFailed       ResourceState = "failed"
)
```

### Error Types

```go
// Provisioning errors
sdk.ErrProvisionFailed("message")
sdk.ErrDeprovisionFailed("message")

// Configuration errors
sdk.ErrInvalidConfig("message")
sdk.ErrInvalidResource("message")

// Operational errors
sdk.ErrStatusCheckFailed("message")
sdk.ErrNotFound("message")
sdk.ErrAlreadyExists("message")
sdk.ErrTimeout("message")
sdk.ErrUnauthorized("message")
```

### Hint Helpers

```go
// URL hint (dashboard, documentation)
sdk.NewURLHint(label, url, icon)

// Command hint (kubectl, psql)
sdk.NewCommandHint(label, command, icon)

// Connection string hint (database, cache)
sdk.NewConnectionStringHint(label, connectionString)

// Dashboard hint (monitoring, admin panel)
sdk.NewDashboardHint(label, url)
```

### Available Icons

```go
const (
    IconExternalLink = "external-link"
    IconDatabase     = "database"
    IconLock         = "lock"
    IconTerminal     = "terminal"
    IconGitBranch    = "git-branch"
    IconDownload     = "download"
    IconSettings     = "settings"
    IconCloud        = "cloud"
    IconServer       = "server"
)
```

## Best Practices

### 1. Keep Provisioners Stateless

**Don't:**
```go
type BadProvisioner struct {
    cache map[string]*Resource  // Stateful - BAD
}
```

**Do:**
```go
type GoodProvisioner struct {
    apiClient APIClient  // Stateless client - GOOD
}
```

### 2. Use Structured Errors

**Don't:**
```go
return fmt.Errorf("failed")  // Generic error
```

**Do:**
```go
return sdk.ErrProvisionFailed("database creation failed: %v", err)
```

### 3. Provide Meaningful Hints

**Don't:**
```go
hints := []sdk.Hint{
    {Type: "url", Label: "Link", Value: "http://..."},
}
```

**Do:**
```go
hints := []sdk.Hint{
    sdk.NewURLHint("AWS Console", "https://console.aws.amazon.com/...", sdk.IconCloud),
    sdk.NewConnectionStringHint("PostgreSQL", "postgres://..."),
    sdk.NewDashboardHint("Monitoring", "https://grafana.mycompany.com/..."),
}
```

### 4. Handle Context Cancellation

```go
func (p *Provisioner) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
    // Check context before long operations
    select {
    case <-ctx.Done():
        return sdk.ErrTimeout("provision operation cancelled")
    default:
    }

    // Do work...
    result, err := p.apiClient.CreateResource(ctx, ...)
    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return sdk.ErrTimeout("provision timeout: %v", err)
        }
        return sdk.ErrProvisionFailed("failed to create: %v", err)
    }

    return nil
}
```

### 5. Validate Configuration

```go
func (p *Provisioner) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
    // Validate required fields
    name := config.GetString("name")
    if name == "" {
        return sdk.ErrInvalidConfig("name is required")
    }

    // Validate allowed values
    size := config.GetString("size")
    validSizes := []string{"small", "medium", "large"}
    if !contains(validSizes, size) {
        return sdk.ErrInvalidConfig("invalid size: must be one of %v", validSizes)
    }

    // Continue with provisioning...
}
```

### 6. Test Platform Implementations

```go
// Test successful provision
func TestProvisionSuccess(t *testing.T) { ... }

// Test provision failure
func TestProvisionFailure(t *testing.T) { ... }

// Test invalid configuration
func TestInvalidConfig(t *testing.T) { ... }

// Test context cancellation
func TestContextCancellation(t *testing.T) { ... }

// Test GetStatus for all states
func TestGetStatusAllStates(t *testing.T) { ... }
```

## Examples

### Example 1: AWS RDS Database

See full implementation in [examples/aws-platform/](../examples/aws-platform/)

### Example 2: Azure Storage Account

See full implementation in [examples/azure-platform/](../examples/azure-platform/)

### Example 3: HashiCorp Vault Secret

See full implementation in [examples/vault-platform/](../examples/vault-platform/)

## Troubleshooting

### Platform Not Loaded

**Problem:** Platform manifest not discovered

**Solution:**
1. Check `platform.yaml` is in correct location (`platforms/*/platform.yaml`)
2. Verify YAML syntax is valid
3. Check compatibility version constraints
4. Review server logs for loading errors

### Provisioner Not Found

**Problem:** `no provisioner registered for type X`

**Solution:**
1. Check provisioner is listed in `platform.yaml`
2. Verify provisioner type matches resource type
3. Ensure provisioner is registered in registry

### Version Compatibility Error

**Problem:** `platform requires core version >= X.Y.Z`

**Solution:**
1. Update innominatus core to compatible version
2. Or update platform to newer version with broader compatibility

### Configuration Errors

**Problem:** `INVALID_CONFIG: missing required field`

**Solution:**
1. Check Score spec includes all required configuration
2. Review platform documentation for required fields
3. Validate configuration before deployment

## Next Steps

1. Review [SDK Reference](../pkg/sdk/doc.go) for complete API
2. Explore [Example Platforms](../examples/)
3. Read [EXTENSIBILITY_ARCHITECTURE.md](EXTENSIBILITY_ARCHITECTURE.md) for design details
4. Join community discussions for platform development

---

**Happy Platform Building!**

For questions or contributions, visit:
- GitHub: https://github.com/philipsahli/innominatus-core
- Documentation: https://docs.innominatus.io
