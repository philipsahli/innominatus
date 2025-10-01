# Workflow Context and Variable Sharing

This document describes the workflow context system for sharing variables and passing data between steps in innominatus workflows.

## Overview

Workflow context enables:
- **Workflow-level variables** shared across all steps
- **Step output capture** from files or stdout
- **Variable passing** between steps
- **Dynamic workflows** that adapt based on previous step results

## Features

### 1. Workflow Variables

Define variables at the workflow level that are accessible to all steps:

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: deployment-workflow

variables:
  ENVIRONMENT: production
  REGION: us-east-1
  APP_NAME: myapp

spec:
  steps:
    - name: deploy
      type: kubernetes
      if: $ENVIRONMENT == production
```

### 2. Step Outputs

Capture outputs from step execution:

```yaml
steps:
  - name: build
    type: validation
    outputs:
      - version
      - build_id
    outputFile: /tmp/build-outputs.json
```

### 3. Set Variables

Explicitly set workflow variables from a step:

```yaml
steps:
  - name: configure
    type: validation
    setVariables:
      DATABASE_URL: postgresql://localhost:5432/mydb
      API_KEY: secret-key-123
```

### 4. Variable References

Reference variables and outputs in conditions and configurations:

```yaml
steps:
  - name: deploy-backend
    type: kubernetes
    if: ${build.version} matches ^v[0-9]+

  - name: notify
    type: monitoring
    env:
      VERSION: ${build.version}
      ENVIRONMENT: ${workflow.ENVIRONMENT}
```

### 5. Resource Parameter Interpolation

Use workflow variables and step outputs in Score resource parameters:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: myapp

variables:
  ENVIRONMENT: production
  REGION: us-east-1

resources:
  database:
    type: postgres
    params:
      name: "myapp-db-${workflow.ENVIRONMENT}"
      region: "${workflow.REGION}"
      version: "${db-config.pg_version}"
      tags:
        - "env:${workflow.ENVIRONMENT}"
        - "app:myapp"
        - "version:${build.version}"
```

Resource parameters support full variable interpolation including:
- Workflow variables: `${workflow.VAR}`
- Step outputs: `${step.output}`
- Resource outputs: `${resources.name.attr}` **NEW**
- Nested maps and arrays
- Mixed with static values

## Variable Syntax

### Reference Formats

innominatus supports **three types of variable references**:

**1. Workflow variables:**
```yaml
$VAR_NAME              # Simple reference
${VAR_NAME}            # Explicit reference
${workflow.VAR_NAME}   # Explicit workflow variable (recommended)
```

**2. Step outputs:**
```yaml
${stepName.outputKey}  # Reference output from previous step
$stepName.outputKey    # Short form
```

**3. Resource outputs:** (NEW)
```yaml
${resources.resourceName.attribute}  # Reference provisioned resource attributes
$resources.resourceName.attribute    # Short form
```

### All Three Syntaxes Together

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: complete-example

variables:
  APP_NAME: myapp
  ENVIRONMENT: production

spec:
  steps:
    - name: build
      type: validation
      outputFile: /tmp/build.json
      # Outputs: version, image_url

    - name: provision-database
      type: terraform
      path: ./terraform/postgres
      # Creates resource outputs: db.host, db.port, db.name

    - name: deploy
      type: kubernetes
      env:
        # Workflow variable
        APP_NAME: ${workflow.APP_NAME}
        ENVIRONMENT: ${workflow.ENVIRONMENT}

        # Step output
        VERSION: ${build.version}
        IMAGE: ${build.image_url}

        # Resource output
        DATABASE_URL: "postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}"

        # All three combined
        FULL_CONFIG: "app=${workflow.APP_NAME},version=${build.version},db=${resources.database.host}"
```

**Step environment (scoped to step):**
```yaml
env:
  LOCAL_VAR: value
if: $LOCAL_VAR == value
```

### Variable Precedence

Variables are resolved in this order (highest to lowest priority):
1. **Step env** - Variables defined in step's `env` field
2. **Workflow variables** - Variables defined in `workflow.variables`
3. **Context environment** - System-level environment variables
4. **System environment** - OS environment variables

Example:
```yaml
variables:
  ENV: workflow-value

steps:
  - name: test
    type: validation
    env:
      ENV: step-value  # This takes precedence
    if: $ENV == step-value  # TRUE
```

## Output Capture

### Output File Formats

#### JSON Format

```yaml
steps:
  - name: build
    type: validation
    outputFile: /tmp/outputs.json
```

**File content:**
```json
{
  "version": "1.0.0",
  "build_id": "12345",
  "artifact_url": "https://artifacts.example.com/app-1.0.0.tar.gz"
}
```

#### Key=Value Format

```yaml
steps:
  - name: build
    type: validation
    outputFile: /tmp/outputs.env
```

**File content:**
```bash
VERSION=1.0.0
BUILD_ID=12345
ARTIFACT_URL=https://artifacts.example.com/app-1.0.0.tar.gz

# Comments are supported
DEBUG=false
```

### Stdout Parsing

innominatus supports multiple stdout output formats:

#### GitHub Actions Style

```bash
echo "::set-output name=version::1.0.0"
echo "::set-output name=build_id::12345"
```

#### Environment Variable Style

```bash
echo "OUTPUT_VERSION=1.0.0"
echo "OUTPUT_BUILD_ID=12345"
```

#### Named Outputs

```yaml
steps:
  - name: get-version
    type: validation
    outputs:
      - version
```

The step's last non-empty stdout line will be captured as the `version` output.

### Set Variables

Explicitly set workflow variables without output files:

```yaml
steps:
  - name: setup
    type: validation
    setVariables:
      DATABASE_URL: postgresql://localhost:5432/db
      CACHE_ENABLED: "true"
      MAX_CONNECTIONS: "100"
```

These variables become available to all subsequent steps.

## Complete Examples

### Example 1: Build and Deploy Pipeline

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: build-deploy-pipeline

variables:
  ENVIRONMENT: production
  REGION: us-east-1

spec:
  steps:
    # Step 1: Build application and capture version
    - name: build
      type: validation
      outputFile: /tmp/build-outputs.json
      # Outputs: version, build_id, artifact_url

    # Step 2: Run tests (only if build succeeded)
    - name: test
      type: validation
      if: build.success
      env:
        VERSION: ${build.version}

    # Step 3: Deploy to staging
    - name: deploy-staging
      type: kubernetes
      namespace: staging
      if: $ENVIRONMENT != production
      env:
        VERSION: ${build.version}
        BUILD_ID: ${build.build_id}

    # Step 4: Deploy to production
    - name: deploy-production
      type: kubernetes
      namespace: production
      if: $ENVIRONMENT == production
      env:
        VERSION: ${build.version}
        ARTIFACT: ${build.artifact_url}

    # Step 5: Notify deployment
    - name: notify
      type: monitoring
      when: on_success
      env:
        MESSAGE: "Deployed version ${build.version} to ${workflow.ENVIRONMENT}"
```

### Example 2: Database Setup with Connection Passing

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: database-setup

spec:
  steps:
    # Step 1: Provision database
    - name: provision-db
      type: terraform
      path: ./terraform/database
      outputFile: /tmp/terraform-outputs.json
      # Outputs: db_host, db_port, db_name

    # Step 2: Set connection string
    - name: configure
      type: validation
      setVariables:
        DB_URL: "postgresql://${provision-db.db_host}:${provision-db.db_port}/${provision-db.db_name}"

    # Step 3: Run migrations
    - name: migrate
      type: validation
      env:
        DATABASE_URL: ${workflow.DB_URL}

    # Step 4: Deploy application with connection
    - name: deploy
      type: kubernetes
      env:
        DB_HOST: ${provision-db.db_host}
        DB_PORT: ${provision-db.db_port}
        DB_NAME: ${provision-db.db_name}
```

### Example 3: Feature Flag Workflow

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: feature-deployment

variables:
  FEATURE_A: "true"
  FEATURE_B: "false"
  DEPLOYMENT_MODE: canary

spec:
  steps:
    - name: deploy-core
      type: kubernetes
      setVariables:
        CORE_VERSION: "2.0.0"

    - name: deploy-feature-a
      type: kubernetes
      if: ${workflow.FEATURE_A} == true
      env:
        CORE_VERSION: ${workflow.CORE_VERSION}

    - name: deploy-feature-b
      type: kubernetes
      if: ${workflow.FEATURE_B} == true
      env:
        CORE_VERSION: ${workflow.CORE_VERSION}

    - name: verify-deployment
      type: validation
      env:
        MODE: ${workflow.DEPLOYMENT_MODE}
        VERSION: ${workflow.CORE_VERSION}
```

### Example 4: Multi-Environment Deployment

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: multi-env-deployment

variables:
  VERSION: "1.0.0"
  TARGET_ENV: staging

spec:
  steps:
    - name: build
      type: validation
      env:
        VERSION: ${workflow.VERSION}
      outputFile: /tmp/build.json
      # Outputs: artifact_id

    - name: deploy-dev
      type: kubernetes
      namespace: development
      if: ${workflow.TARGET_ENV} == dev
      env:
        ARTIFACT_ID: ${build.artifact_id}

    - name: deploy-staging
      type: kubernetes
      namespace: staging
      if: ${workflow.TARGET_ENV} == staging
      env:
        ARTIFACT_ID: ${build.artifact_id}

    - name: deploy-production
      type: kubernetes
      namespace: production
      if: ${workflow.TARGET_ENV} == production
      env:
        ARTIFACT_ID: ${build.artifact_id}

    - name: update-version
      type: validation
      setVariables:
        DEPLOYED_VERSION: ${workflow.VERSION}
        DEPLOYED_ENV: ${workflow.TARGET_ENV}
        DEPLOYED_AT: ${build.artifact_id}
```

### Example 5: Conditional Deployment Based on Test Results

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: test-and-deploy

spec:
  steps:
    - name: run-tests
      type: validation
      outputFile: /tmp/test-results.json
      # Outputs: passed, failed, coverage

    - name: check-quality
      type: validation
      setVariables:
        QUALITY_GATE: "passed"
      if: ${run-tests.coverage} >= 80

    - name: deploy
      type: kubernetes
      if: ${workflow.QUALITY_GATE} == passed
      env:
        TEST_COVERAGE: ${run-tests.coverage}
        TESTS_PASSED: ${run-tests.passed}

    - name: notify-success
      type: monitoring
      when: on_success
      if: deploy.success
      env:
        COVERAGE: ${run-tests.coverage}

    - name: notify-failure
      type: monitoring
      when: on_failure
      env:
        REASON: "Quality gate failed or deployment failed"
```

## Resource Parameter Interpolation

### Overview

Resource parameters in Score specifications support full variable interpolation. This enables dynamic resource configuration based on workflow state, environment, and previous step outputs.

### Interpolation Features

**Supported in resource params:**
- Workflow variables: `${workflow.VAR}`
- Step outputs: `${step.output}`
- Resource outputs: `${resources.name.attr}` **NEW**
- Nested maps and objects
- Arrays and lists
- Mixed with static values

### Three Ways to Reference Data

**1. Workflow Variables** - Configuration and constants
```yaml
${workflow.ENVIRONMENT}  # "production"
${workflow.REGION}       # "us-east-1"
```

**2. Step Outputs** - Results from previous steps
```yaml
${build.version}         # "2.5.0" from build step
${provision-db.db_host}  # "db.example.com" from provisioning
```

**3. Resource Outputs** - Attributes from provisioned resources
```yaml
${resources.database.host}  # Direct reference to database resource
${resources.cache.endpoint} # Cache endpoint from resource
```

### Score Spec Example

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: myapp

variables:
  ENVIRONMENT: production
  REGION: us-east-1

resources:
  database:
    type: postgres
    params:
      # Simple interpolation
      name: "myapp-db-${workflow.ENVIRONMENT}"
      region: "${workflow.REGION}"

      # Reference step outputs
      version: "${db-config.pg_version}"
      instance_class: "${sizing.db_instance}"

      # Nested configuration
      backup:
        enabled: true
        retention_days: "${sizing.backup_retention}"
        window: "02:00-04:00"

      # Array with interpolation
      tags:
        - "env:${workflow.ENVIRONMENT}"
        - "region:${workflow.REGION}"
        - "version:${build.version}"
        - "managed-by:innominatus"

  cache:
    type: redis
    params:
      name: "myapp-cache-${workflow.ENVIRONMENT}"
      region: "${workflow.REGION}"
      # Nested object with mixed values
      cluster:
        node_type: "${sizing.cache_node_type}"
        replicas: 3  # Static value
        version: "7.0"
```

### How It Works

When a workflow processes a Score specification:

1. **Workflow Execution**: Steps run and populate the execution context with:
   - Workflow variables (defined in `variables:`)
   - Step outputs (from `outputFile` and `setVariables`)

2. **Resource Processing**: Before resources are provisioned:
   - Resource params are recursively traversed
   - String values containing `${...}` are interpolated
   - Variables are resolved from execution context
   - Non-string values (numbers, booleans) pass through unchanged

3. **Variable Precedence**: Same as step environment variables:
   - Step env > Workflow variables > Context env > System env

### Practical Example

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: payment-api

variables:
  APP_NAME: payment-api
  ENVIRONMENT: production
  REGION: us-west-2

workflows:
  deploy:
    steps:
      # Step 1: Build application
      - name: build
        type: validation
        outputFile: /tmp/build.json
        # Outputs: version, image_url, commit_sha

      # Step 2: Determine infrastructure sizing
      - name: sizing
        type: validation
        setVariables:
          DB_INSTANCE: "db.r5.xlarge"
          CACHE_NODES: "3"
          APP_REPLICAS: "5"
        env:
          ENVIRONMENT: ${workflow.ENVIRONMENT}

      # Step 3: Provision TLS certificate
      - name: tls
        type: validation
        outputFile: /tmp/tls.json
        # Outputs: cert_arn, cert_domain

      # Resources are processed after steps complete
      # with full access to all variables and outputs

resources:
  database:
    type: postgres
    params:
      identifier: "${workflow.APP_NAME}-${workflow.ENVIRONMENT}"
      instance_class: "${sizing.DB_INSTANCE}"
      region: "${workflow.REGION}"
      backup_retention_period: 30
      tags:
        Name: "${workflow.APP_NAME}-database"
        Environment: "${workflow.ENVIRONMENT}"
        Version: "${build.version}"

  cache:
    type: redis
    params:
      cluster_id: "${workflow.APP_NAME}-cache-${workflow.ENVIRONMENT}"
      node_type: "cache.r5.large"
      num_cache_nodes: "${sizing.CACHE_NODES}"
      engine_version: "7.0"

  route:
    type: route
    params:
      # Build full hostname from variables
      host: "${workflow.APP_NAME}.${workflow.REGION}.example.com"
      port: 443
      certificate_arn: "${tls.cert_arn}"

      # Can use outputs in nested config
      backend:
        protocol: "HTTPS"
        port: 8443
        health_check: "/health"
```

### Use Cases

**1. Environment-Specific Configuration**
```yaml
variables:
  ENVIRONMENT: staging

resources:
  database:
    type: postgres
    params:
      # Different naming per environment
      name: "myapp-${workflow.ENVIRONMENT}"
      # Conditional sizing via step outputs
      instance_class: "${sizing.db_instance}"
```

**2. Multi-Region Deployment**
```yaml
variables:
  REGION: eu-west-1

resources:
  database:
    type: postgres
    params:
      region: "${workflow.REGION}"
      replica_regions:
        - "${workflow.SECONDARY_REGION}"
      endpoint: "db.${workflow.REGION}.example.com"
```

**3. Version Tagging**
```yaml
resources:
  all-resources:
    params:
      tags:
        - "version:${build.version}"
        - "commit:${build.commit_sha}"
        - "deployed:${deployment.timestamp}"
```

**4. Dynamic Connection Strings**
```yaml
resources:
  application:
    params:
      environment:
        DATABASE_URL: "postgresql://${db.host}:${db.port}/${db.name}"
        CACHE_URL: "redis://${cache.endpoint}:${cache.port}"
        API_VERSION: "${build.version}"
```

### Best Practices

**1. Use workflow variables for configuration:**
```yaml
# Good - centralized configuration
variables:
  ENVIRONMENT: production
  REGION: us-east-1

resources:
  db:
    params:
      name: "app-${workflow.ENVIRONMENT}"
```

**2. Reference step outputs for dynamic values:**
```yaml
# Good - uses actual provisioned values
steps:
  - name: provision-cert
    outputFile: /tmp/cert.json

resources:
  route:
    params:
      certificate_arn: "${provision-cert.cert_arn}"
```

**3. Keep static values as-is:**
```yaml
# Good - clear distinction
resources:
  db:
    params:
      name: "${workflow.APP_NAME}-db"  # Dynamic
      version: "15"                     # Static
      backup_enabled: true              # Static
```

## Best Practices

### 1. Use Workflow Variables for Constants

```yaml
# Good
variables:
  REGION: us-east-1
  ENVIRONMENT: production

# Less maintainable - repeated in each step
steps:
  - name: step1
    env:
      REGION: us-east-1
```

### 2. Capture Important Outputs

```yaml
# Good - capture for reuse
- name: build
  type: validation
  outputFile: /tmp/build.json

# Less flexible - no reuse
- name: build
  type: validation
```

### 3. Use SetVariables for Computed Values

```yaml
# Good - computed once, used everywhere
- name: configure
  type: validation
  setVariables:
    DATABASE_URL: "postgresql://${db.host}:${db.port}/${db.name}"

# Bad - recompute in each step
- name: app1
  env:
    URL: "postgresql://${db.host}:${db.port}/${db.name}"
- name: app2
  env:
    URL: "postgresql://${db.host}:${db.port}/${db.name}"
```

### 4. Use Explicit References for Clarity

```yaml
# Clear intent
if: ${workflow.ENVIRONMENT} == production

# Less clear
if: $ENVIRONMENT == production
```

### 5. Document Output Contracts

```yaml
- name: build
  type: validation
  outputFile: /tmp/build.json
  # Outputs expected:
  # - version: Semantic version (e.g., "1.0.0")
  # - build_id: Unique build identifier
  # - artifact_url: Download URL for artifact
```

## Output Display

When steps complete, captured outputs are displayed:

```
✅ build completed (took 2.5s)
📤 Set 2 workflow variables
📄 Captured 3 outputs from file: /tmp/build.json
💾 version = 1.0.0
💾 build_id = 12345
💾 artifact_url = https://artifacts.example.com/app...
```

## Limitations

### Current Limitations

1. **No stdout capture for custom executors** - Only output files and setVariables supported currently
2. **No nested object support** - JSON objects are flattened to strings
3. **No array support** - JSON arrays converted to strings
4. **No computed expressions** - Cannot compute values in setVariables (use external scripts)

### Workarounds

**Complex outputs:**
```bash
# In your script, write simple key=value
echo "VERSION=$(compute_version)" > /tmp/outputs.env
echo "URL=$(build_url)" >> /tmp/outputs.env
```

**Nested data:**
```bash
# Flatten nested JSON to top-level keys
jq -r 'to_entries | map("\(.key)=\(.value)") | .[]' complex.json > outputs.env
```

## Future Enhancements

### Planned Features

1. **Stdout capture for all step types**
2. **Expression evaluation in setVariables**
3. **Nested object access** (e.g., `${build.metadata.version}`)
4. **Array indexing** (e.g., `${build.tags[0]}`)
5. **Output transformation functions**
6. **Secret masking in output display**

### Example (Future)

```yaml
steps:
  - name: compute
    type: validation
    setVariables:
      FULL_URL: "${protocol}://${host}:${port}/${path}"  # Expression eval
      VERSION_MAJOR: "${version | split('.') | first}"    # Transformation

  - name: use-nested
    if: ${build.metadata.version} == 1.0.0  # Nested access
```

## Testing

Run context and output tests:

```bash
go test -v ./internal/workflow -run "TestOutputParser|TestExecutionContext"
```

## See Also

- [Conditional Execution](CONDITIONAL_EXECUTION.md) - Use variables in conditions
- [Parallel Execution](PARALLEL_EXECUTION.md) - Run steps concurrently
- Examples: `examples/context-workflow.yaml`

## References

- Implementation: `internal/workflow/outputs.go`
- Context: `internal/workflow/conditions.go`
- Integration: `internal/workflow/executor.go`
- Tests: `internal/workflow/outputs_test.go`
- Gap Analysis: `GAP_ANALYSIS-2025-09-30-2.md` (Gap 4.2.1)
