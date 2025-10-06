# Golden Paths Metadata and Parameters

## Overview

Golden paths in innominatus now support rich metadata and parameter customization, allowing platform teams to provide better documentation, categorization, and runtime configuration for pre-defined workflows.

## Metadata Schema

Each golden path can be defined with the following metadata fields:

### Simple Format (Backward Compatible)

```yaml
goldenpaths:
  my-path: ./workflows/my-workflow.yaml
```

### Full Metadata Format

```yaml
goldenpaths:
  my-path:
    workflow: ./workflows/my-workflow.yaml
    description: Human-readable description of what this golden path does
    category: deployment | cleanup | environment | database | observability
    tags: [tag1, tag2, tag3]
    estimated_duration: 5-10 minutes
    required_params: [param1, param2]
    optional_params:
      param3: default_value
      param4: another_default
```

### Metadata Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `workflow` | string | Yes | Path to the workflow YAML file |
| `description` | string | No | Human-readable description of the golden path |
| `category` | string | No | Category for grouping (deployment, cleanup, etc.) |
| `tags` | array | No | Tags for filtering and searching |
| `estimated_duration` | string | No | Estimated time to complete (e.g., "5-10 minutes") |
| `required_params` | array | No | List of parameters that must be provided |
| `optional_params` | map | No | Parameters with default values |

## Example Configuration

```yaml
goldenpaths:
  deploy-app:
    workflow: ./workflows/deploy-app.yaml
    description: Deploy application with full GitOps pipeline including Git repository and ArgoCD onboarding
    category: deployment
    tags: [deployment, gitops, argocd, production]
    estimated_duration: 5-10 minutes
    required_params: []
    optional_params:
      sync_policy: auto
      namespace_prefix: ""

  ephemeral-env:
    workflow: ./workflows/ephemeral-env.yaml
    description: Create temporary environment for testing with automatic TTL-based cleanup
    category: environment
    tags: [testing, ephemeral, temporary, preview]
    estimated_duration: 3-7 minutes
    required_params: []
    optional_params:
      ttl: 2h
      environment_type: preview

  db-backup:
    workflow: ./workflows/db-backup.yaml
    description: Perform database backup with configurable retention
    category: database
    tags: [database, backup, maintenance]
    estimated_duration: 10-20 minutes
    required_params: [database_name]
    optional_params:
      backup_retention: 7d
      compression: gzip
```

## Using Golden Paths

### List Available Golden Paths

```bash
./innominatus-ctl list-goldenpaths
```

Output shows all metadata:

```
Available Golden Paths (3):
═══════════════════════════════════════════════════════════════

⚙️ deploy-app
   Description: Deploy application with full GitOps pipeline
   Workflow: ./workflows/deploy-app.yaml
   Category: deployment
   Duration: 5-10 minutes
   Tags: deployment, gitops, argocd, production
   Optional Parameters:
      • sync_policy (default: auto)
      • namespace_prefix (default: )
───────────────────────────────────────────────────────────────

ℹ️ Run a golden path: ./innominatus-ctl run <path-name> [score-spec.yaml] [--param key=value]
```

### Run a Golden Path

#### Basic Usage

```bash
# Run with defaults
./innominatus-ctl run deploy-app score-spec.yaml

# Run without Score spec (for paths that don't need it)
./innominatus-ctl run demo-setup
```

#### With Parameter Overrides

```bash
# Override single parameter
./innominatus-ctl run ephemeral-env score-spec.yaml --param ttl=4h

# Override multiple parameters
./innominatus-ctl run db-lifecycle score-spec.yaml \
  --param operation=backup \
  --param backup_retention=30d

# Mix Score spec with parameter overrides
./innominatus-ctl run deploy-app my-app.yaml \
  --param sync_policy=manual \
  --param namespace_prefix=prod-
```

### Parameter Validation

The CLI automatically validates parameters:

1. **Required Parameters**: Must be provided or an error is returned
2. **Optional Parameters**: Use defaults if not provided
3. **Type Validation**: Parameters are validated according to the workflow requirements

Example error for missing required parameter:

```bash
$ ./innominatus-ctl run db-backup --param backup_retention=14d
Error: parameter validation failed: required parameter 'database_name' is missing
```

## Implementation Details

### Configuration Loading

The golden paths configuration loader (`internal/goldenpaths/config.go`) supports both formats:

```go
type GoldenPathMetadata struct {
    Description       string            `yaml:"description"`
    Tags              []string          `yaml:"tags"`
    RequiredParams    []string          `yaml:"required_params"`
    OptionalParams    map[string]string `yaml:"optional_params"`
    WorkflowFile      string            `yaml:"workflow"`
    Category          string            `yaml:"category"`
    EstimatedDuration string            `yaml:"estimated_duration"`
}
```

### Parameter Merging

Parameters are merged in this order (lowest to highest priority):

1. **Default values** from `optional_params`
2. **User-provided values** via `--param` flags

Example:

```yaml
optional_params:
  ttl: 2h
  environment_type: preview
```

```bash
# Uses: ttl=4h (override), environment_type=preview (default)
./innominatus-ctl run ephemeral-env --param ttl=4h
```

### API Methods

```go
// Get metadata for a golden path
metadata, err := config.GetMetadata("deploy-app")

// Validate required parameters
err := config.ValidateParameters("db-backup", params)

// Get parameters merged with defaults
finalParams, err := config.GetParametersWithDefaults("ephemeral-env", params)
```

## Best Practices

### 1. Comprehensive Descriptions

Provide clear descriptions that explain:
- What the golden path does
- When to use it
- What resources it creates/modifies

```yaml
deploy-app:
  description: |
    Deploy application with full GitOps pipeline including Git repository
    creation, ArgoCD onboarding, and automatic sync configuration.
    Use this for production deployments requiring GitOps workflows.
```

### 2. Meaningful Tags

Use tags for:
- **Technology**: `kubernetes`, `terraform`, `ansible`
- **Purpose**: `deployment`, `testing`, `cleanup`
- **Environment**: `production`, `staging`, `development`
- **Features**: `gitops`, `backup`, `monitoring`

### 3. Reasonable Defaults

Set defaults that work for most common use cases:

```yaml
optional_params:
  ttl: 2h              # Short enough for safety
  replicas: 3          # Balanced for HA
  enable_monitoring: true  # Safe to enable by default
```

### 4. Clear Parameter Names

Use descriptive, self-documenting parameter names:

```yaml
# Good
optional_params:
  backup_retention_days: 7
  enable_ssl_verification: true
  max_concurrent_deployments: 5

# Avoid
optional_params:
  retention: 7
  ssl: true
  max: 5
```

### 5. Validation in Workflows

While CLI validates parameters, workflows should also validate:

```yaml
steps:
  - name: validate-ttl
    type: policy
    config:
      script: |
        if [ -z "${ttl}" ]; then
          echo "Error: ttl parameter is required"
          exit 1
        fi
```

## Migration Guide

### Updating Existing Golden Paths

**Before** (simple format):

```yaml
goldenpaths:
  deploy-app: ./workflows/deploy-app.yaml
```

**After** (with metadata):

```yaml
goldenpaths:
  deploy-app:
    workflow: ./workflows/deploy-app.yaml
    description: Deploy application with GitOps
    category: deployment
    tags: [deployment, production]
    estimated_duration: 5-10 minutes
```

Both formats are supported, so you can migrate gradually.

### Adding Parameters to Existing Paths

1. Identify configurable values in your workflow
2. Add them as optional_params with current values as defaults
3. Update workflow to use parameter variables
4. Document the parameters in the description

Example:

```yaml
# Before: hardcoded in workflow
namespace: my-app-prod

# After: parameterized
namespace: my-app-${namespace_suffix}

# In goldenpaths.yaml:
optional_params:
  namespace_suffix: prod
```

## Parameter Validation

### Overview

The Golden Paths parameter validation framework provides comprehensive type checking and constraint enforcement for parameters passed to golden path workflows.

### Parameter Schema Format

**Basic Schema Structure:**

```yaml
goldenpaths:
  example-path:
    workflow: ./workflows/example.yaml
    description: Example golden path
    parameters:
      param_name:
        type: string              # Parameter type
        default: default_value    # Default value (optional)
        description: Description  # Help text for users
        required: false           # Whether parameter is required
```

### Supported Types

#### 1. String

```yaml
app_name:
  type: string
  default: ""
  description: Application name
  pattern: '^[a-z][a-z0-9\-]*$'  # Optional regex pattern
  allowed_values: []             # Optional enum-like restriction
```

**Example usage:**
```bash
./innominatus-ctl run deploy-app score.yaml --param app_name=my-app-123
```

#### 2. Integer

```yaml
replicas:
  type: int
  default: "1"
  description: Number of replicas
  min: 1    # Optional minimum value
  max: 10   # Optional maximum value
```

**Example usage:**
```bash
./innominatus-ctl run ephemeral-env score.yaml --param replicas=3
```

#### 3. Boolean

```yaml
enable_monitoring:
  type: bool
  default: "false"
  description: Enable monitoring stack
```

**Valid values:** `true`, `false`, `yes`, `no`, `1`, `0`, `on`, `off` (case-insensitive)

**Example usage:**
```bash
./innominatus-ctl run observability-setup --param enable_monitoring=true
```

#### 4. Duration

```yaml
ttl:
  type: duration
  default: 2h
  description: Time-to-live for ephemeral environment
  pattern: '^\d+[hmd]$'  # Optional: restrict to hours, minutes, days
```

**Valid values:** Go duration format: `2h`, `30m`, `90s`, `1h30m`, extended formats: `7d` (days), `2w` (weeks)

**Example usage:**
```bash
./innominatus-ctl run ephemeral-env score.yaml --param ttl=4h
```

#### 5. Enum

```yaml
environment_type:
  type: enum
  default: preview
  description: Type of environment
  allowed_values: [preview, staging, development, testing]
```

**Example usage:**
```bash
./innominatus-ctl run ephemeral-env score.yaml --param environment_type=staging
```

### Complete Example with Validation

```yaml
goldenpaths:
  ephemeral-env:
    workflow: ./workflows/ephemeral-env.yaml
    description: Create temporary environment for testing
    category: environment
    tags: [testing, ephemeral, preview]
    estimated_duration: 3-7 minutes
    parameters:
      ttl:
        type: duration
        default: 2h
        description: Time-to-live for ephemeral environment
        pattern: '^\d+[hmd]$'
      environment_type:
        type: enum
        default: preview
        description: Type of ephemeral environment to create
        allowed_values: [preview, staging, development, testing]
      replicas:
        type: int
        default: "1"
        description: Number of application replicas
        min: 1
        max: 10
      enable_monitoring:
        type: bool
        default: "false"
        description: Enable monitoring and observability stack
      namespace_prefix:
        type: string
        default: ""
        description: Optional prefix for namespace
        pattern: '^[a-z0-9\-]*$'
```

### Validation Error Messages

**Invalid duration format:**
```
❌ Parameter validation failed for 'ephemeral-env'
   Parameter:       ttl
   Provided Value:  2x
   Expected Type:   duration
   Constraint:      invalid duration format
   Suggestion:      use format like: 2h, 30m, 90s, 7d
```

**Integer out of range:**
```
❌ Parameter validation failed for 'ephemeral-env'
   Parameter:       replicas
   Provided Value:  15
   Expected Type:   int
   Constraint:      value must be <= 10
```

**Invalid enum value:**
```
❌ Parameter validation failed for 'ephemeral-env'
   Parameter:       environment_type
   Provided Value:  production
   Expected Type:   enum
   Constraint:      value must be one of: preview, staging, development, testing
```

### Migration from Legacy Format

**Old format (deprecated but still supported):**

```yaml
goldenpaths:
  ephemeral-env:
    workflow: ./workflows/ephemeral-env.yaml
    required_params: []
    optional_params:
      ttl: 2h
      environment_type: preview
```

**New format with validation:**

```yaml
goldenpaths:
  ephemeral-env:
    workflow: ./workflows/ephemeral-env.yaml
    parameters:
      ttl:
        type: duration
        default: 2h
        pattern: '^\d+[hmd]$'
      environment_type:
        type: enum
        default: preview
        allowed_values: [preview, staging, development, testing]
```

**Backward Compatibility:**
- If `parameters` is defined, it takes precedence
- If `parameters` is empty/undefined, falls back to `required_params` and `optional_params`
- Both formats can coexist during migration

### Common Validation Patterns

**DNS-safe names:**
```yaml
pattern: '^[a-z][a-z0-9\-]*[a-z0-9]$'
```

**Semantic versioning:**
```yaml
pattern: '^v?\d+\.\d+\.\d+$'
```

**Duration (hours/minutes/days only):**
```yaml
pattern: '^\d+[hmd]$'
```

**Email address:**
```yaml
pattern: '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$'
```

---

## Future Enhancements

Planned improvements:

1. ✅ **Parameter Types** - String, int, bool, duration, enum (implemented)
2. ✅ **Parameter Constraints** - Min/max values, regex patterns (implemented)
3. **Conditional Parameters** - Parameters that depend on other parameters
4. ✅ **Parameter Documentation** - Help text for each parameter (implemented)
5. **Web UI Integration** - Form-based parameter input in web interface
6. **Parameter Templates** - Reusable parameter sets across golden paths

---

**See Also:**
- [CLI Overview](README.md)
- [Output Formatting](output-formatting.md)
- [User Guide](../user-guide/README.md)

---

*Last updated: 2025-10-06*
