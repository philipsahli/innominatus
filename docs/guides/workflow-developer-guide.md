# Workflow Developer Guide

A comprehensive reference for workflow developers building automation pipelines with innominatus.

## Table of Contents

- [Workflow Structure](#workflow-structure)
- [Step Types](#step-types)
- [Variable Interpolation](#variable-interpolation)
- [Conditional Execution](#conditional-execution)
- [Parallel Execution](#parallel-execution)
- [Step Dependencies](#step-dependencies)
- [Output Capture](#output-capture)
- [Error Handling](#error-handling)
- [Best Practices](#best-practices)

## Workflow Structure

Every workflow follows this basic structure:

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: my-workflow
  description: What this workflow does

variables:
  ENVIRONMENT: production
  REGION: us-east-1

spec:
  steps:
    - name: step-1
      type: terraform
      # step configuration
```

## Step Types

### Terraform
Infrastructure provisioning with Terraform:

```yaml
- name: provision-database
  type: terraform
  resource: database
  config:
    operation: apply
    working_dir: ./terraform/postgres
    variables:
      instance_class: db.t3.micro
      backup_retention: 7
    outputs:
      - db_host
      - db_port
      - db_name
```

**Operations:** `init`, `plan`, `apply`, `destroy`, `output`

### Kubernetes
Deploy applications to Kubernetes:

```yaml
- name: deploy-app
  type: kubernetes
  namespace: production
  env:
    IMAGE: ${build.image_url}
    VERSION: ${build.version}
    DATABASE_URL: postgresql://${resources.database.host}:5432/mydb
```

### Ansible
Configuration management with Ansible:

```yaml
- name: configure-servers
  type: ansible
  config:
    playbook: ./ansible/configure.yml
    inventory: ./ansible/inventory.ini
    extra_vars:
      app_version: ${build.version}
      environment: ${workflow.ENVIRONMENT}
```

### Validation
Run checks, tests, or custom scripts:

```yaml
- name: health-check
  type: validation
  env:
    ENDPOINT: https://api.example.com/health
    TIMEOUT: "30s"
```

### Monitoring
Setup observability and notifications:

```yaml
- name: setup-alerts
  type: monitoring
  env:
    DASHBOARD_NAME: ${workflow.APP_NAME}
    ENVIRONMENT: ${workflow.ENVIRONMENT}
```

## Variable Interpolation

innominatus supports three types of variable references:

### 1. Workflow Variables

Defined at the workflow level, available to all steps:

```yaml
variables:
  APP_NAME: myapp
  ENVIRONMENT: production
  REGION: us-east-1

steps:
  - name: deploy
    env:
      NAME: ${workflow.APP_NAME}
      ENV: ${workflow.ENVIRONMENT}
```

### 2. Step Outputs

Capture and reuse outputs from previous steps:

```yaml
steps:
  - name: build
    type: validation
    outputFile: /tmp/build.json
    # Creates: ${build.version}, ${build.image_url}

  - name: deploy
    type: kubernetes
    env:
      IMAGE: ${build.image_url}
      VERSION: ${build.version}
```

### 3. Resource Outputs

Reference attributes from provisioned resources:

```yaml
steps:
  - name: provision-database
    type: terraform
    resource: database
    config:
      outputs:
        - host
        - port
        - name

  - name: deploy-app
    type: kubernetes
    env:
      DATABASE_URL: postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}
      REDIS_URL: redis://${resources.cache.endpoint}:6379
```

### Syntax Shortcuts

```yaml
# All these are equivalent:
${workflow.ENVIRONMENT}
${ENVIRONMENT}
$ENVIRONMENT

# Step outputs:
${build.version}
$build.version

# Resource outputs:
${resources.database.host}
$resources.database.host
```

## Conditional Execution

### When Conditions

Control when steps execute based on previous step status:

```yaml
steps:
  - name: deploy
    type: kubernetes
    when: on_success  # Default: only if all previous steps succeeded

  - name: verify
    when: on_success  # Only after successful steps

  - name: rollback
    when: on_failure  # Only if previous step failed

  - name: cleanup
    when: always  # Always runs regardless of status

  - name: approval-gate
    when: manual  # Requires manual approval (future)
```

### If Conditions

Execute steps conditionally based on variable values:

```yaml
steps:
  - name: deploy-to-production
    type: kubernetes
    if: ${workflow.ENVIRONMENT} == production

  - name: deploy-to-staging
    type: kubernetes
    if: ${workflow.ENVIRONMENT} == staging

  - name: enable-debug
    type: kubernetes
    unless: ${workflow.ENVIRONMENT} == production
```

### Comparison Operators

```yaml
# Equality
if: ${workflow.ENVIRONMENT} == production
if: ${build.version} != v1.0.0

# Numeric comparisons
if: ${workflow.REPLICAS} > 5
if: ${test.coverage} >= 80
if: ${workflow.CPU_LIMIT} <= 2

# String operations
if: ${workflow.BRANCH} contains hotfix
if: ${workflow.BRANCH} startsWith feature/
if: ${workflow.CONFIG_FILE} endsWith .yaml

# Regex matching
if: ${workflow.VERSION} matches ^v[0-9]+\.[0-9]+\.[0-9]+$
```

### Combined Conditions

Use both `when`, `if`, and `unless` together:

```yaml
- name: deploy-production
  type: kubernetes
  when: on_success          # All previous steps succeeded
  if: ${workflow.ENVIRONMENT} == production
  unless: ${workflow.MAINTENANCE_MODE} == true
```

## Parallel Execution

### Basic Parallel Steps

Run independent steps concurrently:

```yaml
steps:
  # These three run in parallel
  - name: validate-syntax
    type: validation
    parallel: true

  - name: security-scan
    type: security
    parallel: true

  - name: policy-check
    type: policy
    parallel: true

  # Runs after all parallel steps complete
  - name: deploy
    type: kubernetes
    parallel: false
```

### Parallel Groups

Organize steps into sequential phases with parallel execution within each phase:

```yaml
steps:
  # Phase 1: Validation (parallel)
  - name: validate-syntax
    type: validation
    parallelGroup: 1

  - name: security-scan
    type: security
    parallelGroup: 1

  # Phase 2: Infrastructure (sequential)
  - name: provision-database
    type: terraform

  # Phase 3: Deployment (parallel)
  - name: deploy-backend
    type: kubernetes
    parallelGroup: 2

  - name: deploy-frontend
    type: kubernetes
    parallelGroup: 2

  - name: deploy-worker
    type: kubernetes
    parallelGroup: 2

  # Phase 4: Verification (sequential)
  - name: health-check
    type: validation
```

**Execution flow:**
```
Group 1: validate-syntax | security-scan  (parallel)
           ↓
        provision-database                 (sequential)
           ↓
Group 2: deploy-backend | deploy-frontend | deploy-worker  (parallel)
           ↓
        health-check                       (sequential)
```

## Step Dependencies

Explicitly declare dependencies between steps:

```yaml
steps:
  - name: provision-infrastructure
    type: terraform

  - name: deploy-application
    type: kubernetes
    dependsOn:
      - provision-infrastructure

  - name: run-tests
    type: validation
    dependsOn:
      - deploy-application

  - name: setup-monitoring
    type: monitoring
    dependsOn:
      - deploy-application
      - run-tests
```

**Dependency enforcement:**
- Steps wait for all dependencies to complete successfully
- If any dependency fails, the step is skipped
- Dependencies must complete with status "success"

## Output Capture

### From Files

Capture outputs from JSON or key=value files:

```yaml
- name: build
  type: validation
  outputFile: /tmp/build.json
  # File contains:
  # {
  #   "version": "1.0.0",
  #   "image_url": "registry.example.com/app:1.0.0",
  #   "commit_sha": "abc123"
  # }
```

Access outputs in subsequent steps:
```yaml
- name: deploy
  env:
    VERSION: ${build.version}
    IMAGE: ${build.image_url}
    COMMIT: ${build.commit_sha}
```

### Set Variables Explicitly

Set workflow variables without output files:

```yaml
- name: configure
  type: validation
  setVariables:
    DATABASE_URL: postgresql://localhost:5432/mydb
    CACHE_ENABLED: "true"
    MAX_CONNECTIONS: "100"

- name: deploy
  env:
    DB_URL: ${workflow.DATABASE_URL}
    CACHE: ${workflow.CACHE_ENABLED}
```

### Terraform Outputs

Automatically capture Terraform outputs:

```yaml
- name: provision-s3
  type: terraform
  resource: storage
  config:
    operation: apply
    working_dir: ./terraform/s3
    outputs:
      - bucket_name
      - bucket_arn
      - bucket_url

- name: deploy-app
  type: kubernetes
  env:
    S3_BUCKET: ${resources.storage.bucket_name}
    S3_ARN: ${resources.storage.bucket_arn}
```

## Error Handling

### Cleanup on Failure

```yaml
steps:
  - name: deploy
    type: kubernetes

  - name: verify
    type: validation
    when: on_success

  - name: rollback
    type: kubernetes
    when: on_failure

  - name: cleanup
    when: always
```

### Conditional Rollback

```yaml
steps:
  - name: deploy-canary
    type: kubernetes

  - name: run-smoke-tests
    type: validation
    when: on_success

  - name: rollback-canary
    type: kubernetes
    when: on_failure
    if: deploy-canary.success

  - name: promote-to-production
    type: kubernetes
    when: on_success
    if: run-smoke-tests.success
```

## Best Practices

### 1. Use Descriptive Step Names

```yaml
# Good
- name: provision-postgres-database
- name: deploy-application-to-production
- name: run-integration-tests

# Avoid
- name: step1
- name: deploy
- name: test
```

### 2. Organize with Parallel Groups

```yaml
# Good - clear phases
steps:
  # Pre-deployment validation
  - name: validate-syntax
    parallelGroup: 1
  - name: security-scan
    parallelGroup: 1

  # Infrastructure
  - name: provision-database
    type: terraform

  # Application deployment
  - name: deploy-backend
    parallelGroup: 2
  - name: deploy-frontend
    parallelGroup: 2

  # Post-deployment
  - name: health-check
    type: validation
```

### 3. Centralize Configuration

```yaml
# Good - workflow variables
variables:
  ENVIRONMENT: production
  REGION: us-east-1
  REPLICAS: "5"

steps:
  - name: deploy
    env:
      ENV: ${workflow.ENVIRONMENT}
      REGION: ${workflow.REGION}

# Avoid - repeated values
steps:
  - name: deploy-backend
    env:
      REGION: us-east-1  # Repeated
  - name: deploy-frontend
    env:
      REGION: us-east-1  # Repeated
```

### 4. Capture Important Outputs

```yaml
# Good - capture for reuse
- name: build
  type: validation
  outputFile: /tmp/build.json

- name: provision-database
  type: terraform
  resource: database
  config:
    outputs:
      - host
      - port
      - name

# Then use everywhere:
- name: deploy
  env:
    VERSION: ${build.version}
    DB_HOST: ${resources.database.host}
```

### 5. Handle Errors Explicitly

```yaml
steps:
  - name: deploy
    type: kubernetes

  - name: verify-deployment
    type: validation
    when: on_success

  - name: rollback-deployment
    type: kubernetes
    when: on_failure

  - name: notify-success
    type: monitoring
    when: on_success

  - name: notify-failure
    type: monitoring
    when: on_failure

  - name: cleanup-resources
    when: always
```

### 6. Use Environment-Based Conditionals

```yaml
variables:
  ENVIRONMENT: production

steps:
  - name: deploy-with-high-resources
    type: kubernetes
    if: ${workflow.ENVIRONMENT} == production
    env:
      REPLICAS: "10"
      CPU: "4"

  - name: deploy-with-low-resources
    type: kubernetes
    unless: ${workflow.ENVIRONMENT} == production
    env:
      REPLICAS: "2"
      CPU: "1"
```

### 7. Document Output Contracts

```yaml
- name: build
  type: validation
  outputFile: /tmp/build.json
  # Expected outputs:
  # - version: Semantic version (e.g., "2.1.0")
  # - image_url: Full image URL with tag
  # - commit_sha: Git commit hash (7 chars)
  # - build_timestamp: ISO 8601 timestamp
```

## Complete Example

Here's a complete workflow demonstrating multiple capabilities:

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: production-deployment
  description: Complete production deployment with validation and monitoring

variables:
  APP_NAME: payment-api
  ENVIRONMENT: production
  REGION: us-west-2

spec:
  steps:
    # Phase 1: Pre-deployment validation (parallel)
    - name: validate-syntax
      type: validation
      parallelGroup: 1

    - name: security-scan
      type: security
      parallelGroup: 1

    - name: policy-check
      type: policy
      parallelGroup: 1

    # Phase 2: Build application
    - name: build
      type: validation
      outputFile: /tmp/build.json
      env:
        APP_NAME: ${workflow.APP_NAME}
      # Outputs: version, image_url, commit_sha

    # Phase 3: Provision infrastructure (parallel)
    - name: provision-database
      type: terraform
      resource: database
      parallelGroup: 2
      config:
        operation: apply
        working_dir: ./terraform/postgres
        outputs:
          - host
          - port
          - name

    - name: provision-cache
      type: terraform
      resource: cache
      parallelGroup: 2
      config:
        operation: apply
        working_dir: ./terraform/redis
        outputs:
          - endpoint
          - port

    # Phase 4: Deploy application
    - name: deploy-application
      type: kubernetes
      namespace: production
      dependsOn:
        - provision-database
        - provision-cache
      env:
        APP_NAME: ${workflow.APP_NAME}
        VERSION: ${build.version}
        IMAGE: ${build.image_url}
        DATABASE_URL: postgresql://${resources.database.host}:${resources.database.port}/${resources.database.name}
        REDIS_URL: redis://${resources.cache.endpoint}:${resources.cache.port}

    # Phase 5: Verification
    - name: health-check
      type: validation
      when: on_success
      env:
        ENDPOINT: https://${workflow.APP_NAME}.example.com/health

    - name: smoke-tests
      type: validation
      when: on_success
      env:
        VERSION: ${build.version}

    # Phase 6: Monitoring setup
    - name: setup-monitoring
      type: monitoring
      when: on_success
      if: ${workflow.ENVIRONMENT} == production
      env:
        APP_NAME: ${workflow.APP_NAME}
        VERSION: ${build.version}
        ENVIRONMENT: ${workflow.ENVIRONMENT}

    # Phase 7: Error handling
    - name: rollback
      type: kubernetes
      when: on_failure
      namespace: production

    - name: notify-success
      type: monitoring
      when: on_success
      env:
        MESSAGE: "Deployed ${workflow.APP_NAME} version ${build.version} to ${workflow.ENVIRONMENT}"

    - name: notify-failure
      type: monitoring
      when: on_failure
      env:
        MESSAGE: "Deployment failed for ${workflow.APP_NAME}"

    - name: cleanup
      type: validation
      when: always
```

## Reference Documentation

For detailed documentation on specific features:

- **Parallel Execution:** [docs/features/parallel-execution.md](../features/parallel-execution.md)
- **Conditional Execution:** [docs/features/conditional-execution.md](../features/conditional-execution.md)
- **Context & Variables:** [docs/features/context-variables.md](../features/context-variables.md)
- **Health Monitoring:** [docs/features/health-monitoring.md](../features/health-monitoring.md)

## Testing Workflows

Test your workflows locally:

```bash
# Validate workflow syntax
./innominatus-ctl validate my-workflow.yaml

# Run workflow
./innominatus-ctl run deploy-app score-spec.yaml

# Check workflow status
./innominatus-ctl status my-app
```

## Next Steps

- Explore [Golden Paths](golden-paths.md) for pre-defined workflow templates
- Learn about [Score Integration](score-integration.md) for application deployment
- Review [examples/](../../examples/) directory for real-world workflows
