# Extensibility Architecture - Implementation Plan

## Overview
This document outlines the implementation plan for the extensibility architecture described in `EXTENSIBILITY_ARCHITECTURE.md`.

## Implementation Phases

### Phase 1: Extract SDK Interfaces (v1.0.0) - **Current Focus**
**Timeline:** Week 1-2
**Status:** In Progress

#### 1.1 Create SDK Package Structure
- [ ] Create `pkg/sdk/` directory (public API)
- [ ] Define `Provisioner` interface
- [ ] Define `Resource` interface
- [ ] Define `Hint` interface
- [ ] Define `Config` interface
- [ ] Define `ValidationResult` interface

#### 1.2 SDK Interfaces

**Core Interfaces:**
```go
// pkg/sdk/provisioner.go
type Provisioner interface {
    Name() string
    Type() string
    Version() string
    Provision(ctx context.Context, resource *Resource, config Config) error
    Deprovision(ctx context.Context, resource *Resource) error
    GetStatus(ctx context.Context, resource *Resource) (*ResourceStatus, error)
    GetHints(ctx context.Context, resource *Resource) ([]Hint, error)
}

// pkg/sdk/resource.go
type Resource struct {
    ID              int64
    ApplicationName string
    ResourceName    string
    ResourceType    string
    State           string
    Configuration   Config
}

// pkg/sdk/hint.go
type Hint struct {
    Type  string  // "url", "command", "dashboard", "connection_string"
    Label string
    Value string
    Icon  string
}

// pkg/sdk/config.go
type Config interface {
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    GetMap(key string) map[string]interface{}
}
```

#### 1.3 Platform Manifest Support
- [ ] Define `Platform` struct
- [ ] Create YAML parser for `platform.yaml`
- [ ] Implement manifest validation
- [ ] Support version compatibility checks

#### 1.4 Backward Compatibility
- [ ] Create adapters for existing provisioners (Gitea, Kubernetes, ArgoCD)
- [ ] Ensure existing code continues to work
- [ ] Add deprecation warnings for internal APIs

---

### Phase 2: Platform Loader (v1.1.0) - **Next**
**Timeline:** Week 3-4
**Status:** Planned

#### 2.1 Platform Registry
- [ ] Create `internal/platform/registry.go`
- [ ] Implement platform registration system
- [ ] Add platform lookup by type
- [ ] Support multiple platforms per instance

#### 2.2 Platform Loader
- [ ] Create `internal/platform/loader.go`
- [ ] Load `platform.yaml` from filesystem
- [ ] Validate platform manifest
- [ ] Register provisioners from platform
- [ ] Handle platform initialization

#### 2.3 Configuration
- [ ] Add `platforms` section to `admin-config.yaml`
- [ ] Support platform-specific configuration
- [ ] Environment variable overrides

---

### Phase 3: Example Platform (Proof of Concept)
**Timeline:** Week 5
**Status:** Planned

#### 3.1 Create `platform-builtin` Package
- [ ] Move existing provisioners to `platform-builtin/`
- [ ] Create `platform-builtin/platform.yaml`
- [ ] Implement SDK interfaces for:
  - [ ] GiteaProvisioner
  - [ ] KubernetesProvisioner
  - [ ] ArgoCDProvisioner

#### 3.2 Testing
- [ ] Unit tests for SDK interfaces
- [ ] Integration tests for platform loader
- [ ] End-to-end test with builtin platform

---

### Phase 4: Documentation & Examples (Future)
**Timeline:** Week 6
**Status:** Planned

#### 4.1 SDK Documentation
- [ ] GoDoc comments for all public APIs
- [ ] Example platform implementation
- [ ] Platform development guide

#### 4.2 Migration Guide
- [ ] Guide for creating custom platforms
- [ ] Best practices for platform development
- [ ] Testing strategies

---

## Implementation Details

### 1. SDK Package Structure

```
pkg/
└── sdk/
    ├── provisioner.go      # Provisioner interface
    ├── resource.go         # Resource types
    ├── config.go           # Configuration interface
    ├── hint.go             # Resource hints
    ├── status.go           # Resource status
    ├── platform.go         # Platform metadata
    └── validation.go       # Validation types
```

### 2. Platform Manifest Format

```yaml
apiVersion: innominatus.io/v1
kind: Platform
metadata:
  name: builtin
  version: 1.0.0
  description: Built-in platform with Gitea, Kubernetes, ArgoCD

compatibility:
  minCoreVersion: "1.0.0"
  maxCoreVersion: "2.0.0"

provisioners:
  - name: gitea-repo
    type: gitea-repo
    version: 1.0.0
    description: Gitea repository provisioner

  - name: kubernetes
    type: kubernetes
    version: 1.0.0
    description: Kubernetes deployment provisioner

  - name: argocd-app
    type: argocd-app
    version: 1.0.0
    description: ArgoCD application provisioner
```

### 3. Platform Loader Flow

```
1. Server Startup
   ↓
2. Load admin-config.yaml
   ↓
3. Discover platform.yaml files
   ↓
4. Parse and validate manifests
   ↓
5. Initialize platforms
   ↓
6. Register provisioners with core
   ↓
7. Server ready
```

### 4. Adapter Pattern (Backward Compatibility)

```go
// Existing provisioner
type GiteaProvisioner struct {
    repo *database.ResourceRepository
}

// SDK adapter
type GiteaProvisionerAdapter struct {
    inner *GiteaProvisioner
}

func (a *GiteaProvisionerAdapter) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
    // Convert SDK types to internal types
    dbResource := convertToDBResource(resource)
    configMap := config.AsMap()

    // Call existing provisioner
    return a.inner.Provision(dbResource, configMap, "platform")
}
```

---

## Success Criteria

### Phase 1 Complete When:
- [ ] SDK interfaces defined and documented
- [ ] Existing provisioners work via adapters
- [ ] No breaking changes to current API
- [ ] All tests passing

### Phase 2 Complete When:
- [ ] Platform manifest can be loaded
- [ ] Provisioners registered dynamically
- [ ] Multiple platforms can coexist
- [ ] Configuration system supports platforms

### Phase 3 Complete When:
- [ ] Built-in platform extracted
- [ ] Example custom platform created
- [ ] End-to-end tests passing

---

## Risks & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing code | High | Use adapter pattern for backward compatibility |
| Complex migration | Medium | Incremental approach with feature flags |
| Performance overhead | Low | SDK is interface-based, minimal overhead |
| Documentation burden | Medium | Auto-generate GoDoc, include examples |

---

## Testing Strategy

### Unit Tests
- SDK interface compliance tests
- Platform manifest parsing tests
- Adapter conversion tests

### Integration Tests
- Platform loader with builtin platform
- Multi-platform registration
- Resource provisioning via SDK

### End-to-End Tests
- Deploy application using platform
- Verify hints returned correctly
- Test platform isolation

---

## Next Steps

1. **Create SDK package structure** - Define interfaces
2. **Implement adapters** - Wrap existing provisioners
3. **Create platform manifest** - Define builtin platform
4. **Build platform loader** - Load and register platforms
5. **Test end-to-end** - Verify full workflow

---

## Timeline Summary

- **Week 1-2**: Phase 1 (SDK extraction)
- **Week 3-4**: Phase 2 (Platform loader)
- **Week 5**: Phase 3 (Example platform)
- **Week 6**: Documentation & polish

**Target Completion:** 6 weeks
