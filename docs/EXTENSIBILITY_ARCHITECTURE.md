# Extensibility Architecture: Platform-as-a-Product Model

**Status:** Proposal
**Version:** 1.0.0
**Last Updated:** 2025-10-18
**Authors:** Platform Team

---

## Executive Summary

This document outlines an architecture for making innominatus extensible, enabling **platform teams** to package and distribute their orchestration logic, resource definitions, and workflow patterns as **versioned products** that can be consumed by enterprise customers.

### Core Principle
**Separate Core Engine from Platform Logic**

innominatus should evolve from a monolithic orchestration tool into a **plugin-based orchestration engine** where platform-specific implementations live in separate, versioned repositories owned by platform teams.

---

## Problem Statement

### Current Limitations

1. **Tight Coupling**: Resource provisioners, golden paths, and workflows are hardcoded in the main repository
2. **No Versioning**: Platform teams cannot version their orchestration logic independently
3. **Testing Challenges**: Platform-specific logic cannot be tested in isolation
4. **Enterprise Adoption**: Each enterprise must fork and customize the core repository
5. **Update Friction**: Core updates risk breaking platform-specific customizations

### Enterprise Requirements

- **Multi-tenancy**: Support multiple platform configurations per innominatus instance
- **Version Control**: Platform teams must use SemVer for their extensions
- **Testability**: Platform logic must be testable independently of core engine
- **Isolation**: Platform team changes should not require core engine changes
- **Discoverability**: Enterprises should discover and install platform packages easily

---

## Proposed Architecture

### Repository Structure

```
innominatus-core/              # Core orchestration engine (this repo)
├── go.mod                      # Semver: v1.x.x
├── internal/engine/            # Workflow execution engine
├── internal/api/               # REST API server
├── internal/database/          # Persistence layer
└── pkg/sdk/                    # Public SDK for platform extensions
    ├── provisioner.go          # Provisioner interface
    ├── workflow.go             # Workflow interface
    └── validator.go            # Spec validator interface

platform-aws/                   # AWS platform implementation (separate repo)
├── go.mod                      # Semver: v2.3.1 (independent versioning)
├── platform.yaml               # Platform manifest
├── provisioners/
│   ├── rds.go                  # RDS database provisioner
│   ├── s3.go                   # S3 bucket provisioner
│   └── eks.go                  # EKS cluster provisioner
├── goldenpaths/
│   ├── deploy-microservice.yaml
│   └── deploy-lambda.yaml
├── resources/
│   ├── postgres.yaml           # Resource definition for AWS RDS
│   └── redis.yaml              # Resource definition for ElastiCache
└── tests/
    ├── provisioner_test.go     # Unit tests
    └── integration_test.go     # Integration tests

platform-azure/                 # Azure platform implementation (separate repo)
├── go.mod                      # Semver: v1.0.5
├── platform.yaml
├── provisioners/
│   ├── cosmosdb.go
│   └── aks.go
└── ...

platform-custom-enterprise/     # Enterprise-specific platform (private repo)
├── go.mod                      # Semver: v0.2.0 (pre-release)
├── platform.yaml
├── provisioners/
│   ├── internal_database.go   # Proprietary database provisioner
│   └── custom_service.go      # Company-specific service
└── ...
```

---

## Core Components

### 1. Platform Manifest (`platform.yaml`)

Each platform repository must contain a manifest describing its capabilities:

```yaml
apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: aws-platform
  version: 2.3.1              # SemVer
  description: AWS cloud platform orchestration
  repository: github.com/innominatus/platform-aws
  license: Apache-2.0

compatibility:
  minCoreVersion: "1.0.0"     # Minimum innominatus-core version
  maxCoreVersion: "2.0.0"     # Maximum compatible core version

provisioners:
  - name: aws-rds
    type: postgres
    version: 2.1.0
    schema: ./provisioners/rds.schema.json

  - name: aws-s3
    type: storage
    version: 2.0.0
    schema: ./provisioners/s3.schema.json

goldenpaths:
  - name: deploy-microservice
    file: ./goldenpaths/deploy-microservice.yaml
    version: 1.5.0

  - name: deploy-lambda
    file: ./goldenpaths/deploy-lambda.yaml
    version: 1.2.0

resources:
  postgres:
    definitionFile: ./resources/postgres.yaml
    provisioner: aws-rds

  redis:
    definitionFile: ./resources/redis.yaml
    provisioner: aws-elasticache

tests:
  unit: go test ./...
  integration: ./tests/run-integration.sh
  e2e: ./tests/run-e2e.sh
```

---

### 2. SDK Interface (`pkg/sdk/`)

**Provisioner Interface:**

```go
// pkg/sdk/provisioner.go
package sdk

type Provisioner interface {
    // Metadata
    Name() string
    Version() string

    // Lifecycle
    Provision(ctx context.Context, resource Resource, config Config) error
    Deprovision(ctx context.Context, resource Resource) error
    GetStatus(ctx context.Context, resource Resource) (Status, error)

    // Validation
    ValidateConfig(config Config) error

    // Schema for config validation
    ConfigSchema() *jsonschema.Schema

    // Resource hints
    GetHints(ctx context.Context, resource Resource) ([]Hint, error)
}

type Resource struct {
    ID              string
    Name            string
    Type            string
    ApplicationName string
    Configuration   map[string]interface{}
}

type Hint struct {
    Type  string // "url", "connection_string", "command", etc.
    Label string
    Value string
    Icon  string
}
```

**Workflow Interface:**

```go
// pkg/sdk/workflow.go
package sdk

type WorkflowDefinition interface {
    Name() string
    Version() string
    Validate(spec ScoreSpec) error
    GenerateSteps(spec ScoreSpec) ([]WorkflowStep, error)
}

type WorkflowStep struct {
    Name       string
    Type       string // "terraform", "ansible", "kubernetes", "provisioner"
    Config     map[string]interface{}
    DependsOn  []string
}
```

---

### 3. Platform Registry

**Central Registry (Future):**

```bash
# Discover available platforms
innominatus-ctl platform discover

# Install a platform
innominatus-ctl platform install github.com/innominatus/platform-aws@v2.3.1

# List installed platforms
innominatus-ctl platform list

# Update platform
innominatus-ctl platform update aws-platform@v2.4.0

# Uninstall platform
innominatus-ctl platform uninstall aws-platform
```

**Platform Installation:**

```yaml
# platforms.yaml (innominatus configuration)
platforms:
  - name: aws
    repository: github.com/innominatus/platform-aws
    version: 2.3.1
    enabled: true

  - name: azure
    repository: github.com/innominatus/platform-azure
    version: 1.0.5
    enabled: false

  - name: custom-enterprise
    repository: github.enterprise.com/platform-team/platform-custom
    version: 0.2.0
    enabled: true
    private: true
    credentials:
      type: github-token
      secretRef: platform-repo-token
```

---

### 4. Semantic Versioning (SemVer)

**Core Engine Versioning:**

- `v1.0.0` → `v1.x.x`: Minor features, bug fixes (backward compatible)
- `v2.0.0`: Breaking changes to SDK interface

**Platform Versioning:**

- `v2.3.1` → `v2.4.0`: New provisioners, golden paths (backward compatible)
- `v3.0.0`: Breaking changes to provisioner implementation

**Compatibility Matrix:**

| Core Version | AWS Platform | Azure Platform | Custom Platform |
|--------------|--------------|----------------|-----------------|
| v1.0.x       | v2.0.0+      | v1.0.0+        | v0.1.0+         |
| v1.5.x       | v2.3.0+      | v1.1.0+        | v0.2.0+         |
| v2.0.x       | v3.0.0+      | v2.0.0+        | v1.0.0+         |

---

## Testing Strategy

### Platform-Level Testing

**1. Unit Tests (Platform Repository)**

```go
// platform-aws/provisioners/rds_test.go
func TestRDSProvisioner_Provision(t *testing.T) {
    provisioner := NewRDSProvisioner()

    resource := sdk.Resource{
        Name: "test-db",
        Type: "postgres",
        Configuration: map[string]interface{}{
            "engine": "postgres",
            "version": "15.4",
            "instance_class": "db.t3.micro",
        },
    }

    err := provisioner.Provision(context.Background(), resource, sdk.Config{})
    assert.NoError(t, err)
}
```

**2. Integration Tests (Platform Repository)**

```bash
# platform-aws/tests/integration_test.sh
#!/bin/bash
# Test RDS provisioning against LocalStack
docker-compose up -d localstack
go test ./... -tags=integration
```

**3. Contract Tests (SDK Compatibility)**

```go
// innominatus-core/tests/sdk_contract_test.go
func TestProvisionerContract(t *testing.T) {
    // Load external platform
    platform := loadPlatform("platform-aws")

    provisioner := platform.GetProvisioner("aws-rds")

    // Verify interface compliance
    assert.Implements(t, (*sdk.Provisioner)(nil), provisioner)

    // Verify schema exists
    schema := provisioner.ConfigSchema()
    assert.NotNil(t, schema)
}
```

**4. E2E Tests (Platform Repository)**

```yaml
# platform-aws/tests/e2e/deploy-microservice.yaml
---
apiVersion: score.dev/v1
metadata:
  name: test-app
resources:
  db:
    type: postgres
    properties:
      engine: postgres
      version: "15.4"
```

```bash
# Run E2E test
innominatus-ctl run deploy-microservice tests/e2e/deploy-microservice.yaml --platform aws --dry-run
```

---

## Migration Path

### Phase 1: Extract SDK (v1.0.0)

1. Create `pkg/sdk/` with public interfaces
2. Refactor existing provisioners to implement SDK interfaces
3. Document SDK usage

**Deliverables:**
- `pkg/sdk/provisioner.go`
- `pkg/sdk/workflow.go`
- `docs/SDK_GUIDE.md`

### Phase 2: Platform Loader (v1.1.0)

1. Implement platform discovery mechanism
2. Add plugin loading from external repositories
3. Support Go plugin architecture or gRPC-based plugins

**Deliverables:**
- `internal/platform/loader.go`
- `platforms.yaml` configuration schema
- Platform manifest validation

### Phase 3: Extract AWS Platform (v1.2.0)

1. Create `platform-aws` repository
2. Move AWS-specific provisioners
3. Add platform manifest
4. Implement testing suite

**Deliverables:**
- Separate repository: `github.com/innominatus/platform-aws`
- CI/CD pipeline for platform repo
- Integration tests

### Phase 4: Registry & Discovery (v2.0.0)

1. Build platform registry (OCI-based or custom)
2. Implement `innominatus-ctl platform` commands
3. Add version compatibility checks

**Deliverables:**
- Platform registry service
- CLI commands for platform management
- Automated compatibility testing

---

## Open Questions

### Technical

1. **Plugin Architecture:**
   - **Go Plugins** (`.so` files): Simple but platform-specific binaries
   - **gRPC Plugins** (Hashicorp plugin pattern): Cross-platform, isolated processes
   - **WebAssembly (WASM)**: Future-proof, sandboxed, but immature Go support
   - **Recommendation:** Start with **gRPC plugins** (Hashicorp model) for isolation and cross-platform support

2. **Platform Distribution:**
   - **OCI Artifacts**: Package platforms as Docker images (GitHub Container Registry, Artifact Registry)
   - **Go Modules**: Distribute as Go packages (`go get github.com/innominatus/platform-aws@v2.3.1`)
   - **Custom Registry**: Build innominatus-specific registry
   - **Recommendation:** Use **OCI artifacts** for distribution, **Go modules** for development

3. **State Management:**
   - Should platforms manage their own state?
   - Core manages resource lifecycle; platforms provide implementation only
   - **Recommendation:** Core owns state, platforms are stateless provisioners

4. **Hot Reload:**
   - Should innominatus support hot-reloading of platform updates without restart?
   - **Recommendation:** v1 requires restart; v2 supports hot reload via gRPC plugin reconnection

### Organizational

1. **Platform Ownership:**
   - Who owns AWS platform? Azure platform?
   - **Recommendation:** Innominatus team owns official platforms; enterprises own custom platforms

2. **Certification:**
   - Should there be a certification process for third-party platforms?
   - **Recommendation:** v1 no certification; v2 introduce security scanning and compatibility badging

3. **Support Model:**
   - How do enterprises get support for custom platforms?
   - **Recommendation:** Core team supports SDK; platform teams support their implementations

4. **Licensing:**
   - Core: Apache 2.0 (permissive)
   - Platforms: Team choice (Apache, MIT, proprietary)

---

## Example: Enterprise Custom Platform

### Scenario: Acme Corp Internal Platform

Acme Corp wants to create a custom platform for their internal database and API gateway:

**Repository:** `github.enterprise.acme.com/platform-team/innominatus-platform-acme`

```yaml
# platform.yaml
apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: acme-internal
  version: 0.5.0
  description: Acme Corp internal platform

compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"

provisioners:
  - name: acme-database
    type: postgres
    version: 0.5.0
    description: Provisions Acme internal PostgreSQL clusters

  - name: acme-api-gateway
    type: api-gateway
    version: 0.3.0
    description: Provisions Acme Kong API gateway instances

goldenpaths:
  - name: deploy-acme-microservice
    file: ./goldenpaths/deploy-acme-microservice.yaml
    version: 0.4.0
```

```go
// provisioners/acme_database.go
package provisioners

import "github.com/innominatus/innominatus-core/pkg/sdk"

type AcmeDatabaseProvisioner struct {
    internalAPI *acme.DatabaseAPI
}

func (p *AcmeDatabaseProvisioner) Provision(ctx context.Context, resource sdk.Resource, config sdk.Config) error {
    // Call Acme's proprietary database provisioning API
    dbInstance, err := p.internalAPI.CreateDatabase(ctx, acme.DatabaseRequest{
        Name: resource.Name,
        Size: config.Get("size").(string),
        HighAvailability: config.Get("ha").(bool),
    })

    if err != nil {
        return fmt.Errorf("failed to provision Acme database: %w", err)
    }

    return nil
}

func (p *AcmeDatabaseProvisioner) GetHints(ctx context.Context, resource sdk.Resource) ([]sdk.Hint, error) {
    return []sdk.Hint{
        {Type: "url", Label: "Acme DB Console", Value: "https://db.acme.com/instance/" + resource.ID, Icon: "database"},
        {Type: "connection_string", Label: "Connection String", Value: "postgres://...", Icon: "lock"},
    }, nil
}
```

**Installation at Acme:**

```yaml
# acme-innominatus-deployment/platforms.yaml
platforms:
  - name: acme-internal
    repository: github.enterprise.acme.com/platform-team/innominatus-platform-acme
    version: 0.5.0
    enabled: true
    private: true
    credentials:
      type: github-token
      secretRef: gh-platform-token
```

**Testing:**

```bash
cd innominatus-platform-acme

# Unit tests
go test ./...

# Integration tests (against Acme staging environment)
export ACME_API_TOKEN=xxx
./tests/integration.sh

# Contract tests (verify SDK compliance)
innominatus-core test-platform ./platform.yaml
```

**Versioning:**

- `v0.1.0` - Initial internal release (alpha)
- `v0.5.0` - Production-ready for Acme microservices
- `v1.0.0` - Stable release for all Acme teams
- `v1.1.0` - Add support for database backups
- `v2.0.0` - Breaking change: new authentication mechanism

---

## Benefits

### For Platform Teams

- **Ownership**: Full control over their platform implementations
- **Versioning**: Independent release cycles using SemVer
- **Testing**: Isolated testing without core engine dependency
- **Innovation**: Rapid iteration on platform features
- **Reusability**: Share platforms across enterprises

### For Enterprise Customers

- **Choice**: Select platforms that match their infrastructure
- **Stability**: Pin to stable platform versions
- **Customization**: Create proprietary platforms for internal systems
- **Updates**: Update core and platforms independently
- **Testability**: Test platform updates before deployment

### For Core Team

- **Focus**: Maintain orchestration engine, not cloud providers
- **Ecosystem**: Enable community-driven platform development
- **Stability**: Reduce breaking changes to core
- **Scalability**: Support unlimited platforms without code bloat

---

## Success Metrics

1. **Time to Platform Development**: <2 weeks from SDK to production platform
2. **Platform Test Coverage**: >80% coverage for each platform repository
3. **Version Compatibility**: Support N-2 core versions for platforms
4. **Enterprise Adoption**: 5+ enterprises with custom platforms within 6 months
5. **Community Platforms**: 10+ community-contributed platforms within 1 year

---

## Next Steps

1. **Immediate (Month 1-2):**
   - [ ] Create RFC for SDK interfaces
   - [ ] Extract `pkg/sdk/` from core
   - [ ] Document SDK usage guide
   - [ ] Create reference platform repository

2. **Short-term (Month 3-4):**
   - [ ] Implement plugin loader (gRPC-based)
   - [ ] Extract AWS platform to separate repo
   - [ ] Build platform manifest validator
   - [ ] Create integration test framework

3. **Medium-term (Month 5-6):**
   - [ ] Build platform registry (OCI-based)
   - [ ] Implement `innominatus-ctl platform` commands
   - [ ] Create certification/security scanning pipeline
   - [ ] Onboard first enterprise custom platform

4. **Long-term (Month 7-12):**
   - [ ] Hot-reload support for platforms
   - [ ] Multi-version platform support
   - [ ] Platform marketplace
   - [ ] Automated compatibility testing

---

## Conclusion

Making innominatus extensible through a **plugin-based platform architecture** will:

✅ Enable **platform teams** to own and version their logic independently
✅ Allow **enterprises** to create custom platforms for proprietary systems
✅ Ensure **testability** at every layer (unit, integration, contract, E2E)
✅ Use **SemVer** for predictable versioning and compatibility
✅ Transform innominatus from a tool into an **orchestration platform**

**This architecture is essential for enterprise adoption and ecosystem growth.**

---

## References

- [Hashicorp Go Plugin System](https://github.com/hashicorp/go-plugin)
- [OCI Artifacts Specification](https://github.com/opencontainers/artifacts)
- [Semantic Versioning 2.0](https://semver.org/)
- [Score Specification](https://docs.score.dev/)
