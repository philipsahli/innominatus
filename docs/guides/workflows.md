# Workflows

Workflows are the heart of innominatus - multi-step orchestration processes that automate your platform operations.

## What is a Workflow?

A workflow is a sequence of steps that execute infrastructure provisioning, application deployment, and platform operations. Each step can be:

- **Terraform** - Infrastructure provisioning
- **Kubernetes** - Application deployment
- **Ansible** - Configuration management
- **Validation** - Checks and validations
- **Monitoring** - Observability setup

## Workflow Structure

```yaml
apiVersion: orchestrator.innominatus.dev/v1
kind: Workflow
metadata:
  name: my-workflow
  description: Description of what this workflow does

variables:
  ENVIRONMENT: production
  REGION: us-east-1

spec:
  steps:
    - name: step-1
      type: terraform
      path: ./terraform/infrastructure

    - name: step-2
      type: kubernetes
      namespace: production
      when: on_success
```

## Step Types

### Terraform Steps

Provision infrastructure using Terraform.

```yaml
- name: provision-database
  type: terraform
  path: ./terraform/postgres
  outputFile: /tmp/terraform-outputs.json
```

### Kubernetes Steps

Deploy applications to Kubernetes.

```yaml
- name: deploy-app
  type: kubernetes
  namespace: production
  env:
    IMAGE: ${build.image_url}
    VERSION: ${build.version}
```

### Validation Steps

Run checks and validations.

```yaml
- name: health-check
  type: validation
  env:
    ENDPOINT: ${resources.app.url}
    TIMEOUT: "30s"
```

## Variable Interpolation

See [Variable Context](../features/context-variables.md) for complete documentation.

**Quick example:**
```yaml
variables:
  APP_NAME: myapp

steps:
  - name: deploy
    env:
      # Workflow variable
      NAME: ${workflow.APP_NAME}

      # Step output
      VERSION: ${build.version}

      # Resource output
      DB_HOST: ${resources.database.host}
```

## Conditional Execution

See [Conditional Execution](../features/conditional-execution.md) for complete documentation.

```yaml
steps:
  - name: deploy-prod
    type: kubernetes
    when: on_success
    if: ${workflow.ENVIRONMENT} == production

  - name: deploy-dev
    type: kubernetes
    unless: ${workflow.SKIP_DEV} == true
```

## Parallel Execution

See [Parallel Execution](../features/parallel-execution.md) for complete documentation.

```yaml
steps:
  - name: test-unit
    parallel: true
    parallelGroup: 1

  - name: test-integration
    parallel: true
    parallelGroup: 1

  - name: deploy
    parallelGroup: 2
```

## Best Practices

### 1. Use Descriptive Names

```yaml
# Good
- name: provision-postgres-database
- name: deploy-application-to-production
- name: run-database-migrations

# Less clear
- name: step1
- name: deploy
- name: db
```

### 2. Capture Outputs

```yaml
# Good - capture for reuse
- name: build
  type: validation
  outputFile: /tmp/build.json
  # Makes ${build.version} available

# Less flexible
- name: build
  type: validation
  # No outputs captured
```

### 3. Use Workflow Variables

```yaml
# Good - centralized configuration
variables:
  ENVIRONMENT: production
  REGION: us-east-1

steps:
  - name: deploy
    env:
      ENV: ${workflow.ENVIRONMENT}

# Less maintainable
steps:
  - name: deploy
    env:
      ENV: production  # Hardcoded in each step
```

### 4. Implement Error Handling

```yaml
steps:
  - name: deploy
    type: kubernetes

  - name: verify-deployment
    when: on_success

  - name: rollback
    when: on_failure

  - name: cleanup
    when: always
```

## Examples

See the [examples directory](../examples/) for real-world workflows:

- [Basic Workflow](../examples/basic-workflow.md)
- [Database Provisioning](../examples/database.md)
- [Multi-Region Deployment](../examples/multi-region.md)
- [GitOps Integration](../examples/gitops.md)

## Next Steps

- Learn about [Golden Paths](golden-paths.md)
- Explore [Conditional Execution](../features/conditional-execution.md)
- Read about [Variable Context](../features/context-variables.md)
