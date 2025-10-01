# Conditional Step Execution

This document describes conditional step execution in innominatus, allowing workflows to skip or execute steps based on runtime conditions.

## Overview

Conditional execution enables dynamic workflow behavior by allowing steps to:
- Run only when specific conditions are met
- Skip execution based on environment variables
- Execute based on previous step results
- Support manual approval gates

## Condition Types

### 1. `when` - Simple Keywords

The `when` field accepts predefined keywords for common scenarios:

```yaml
steps:
  - name: cleanup-on-failure
    type: validation
    when: on_failure  # Only runs if previous steps failed
```

**Supported keywords:**
- `always` - Always execute (default behavior without conditions)
- `on_success` / `success` - Execute only if all previous steps succeeded
- `on_failure` / `failure` - Execute only if any previous step failed
- `manual` - Requires manual approval (currently skipped)

### 2. `if` - Conditional Expression

The `if` field accepts expressions that must evaluate to `true` for the step to run:

```yaml
steps:
  - name: deploy-to-production
    type: kubernetes
    if: $ENVIRONMENT == production
    env:
      ENVIRONMENT: production
```

### 3. `unless` - Negative Conditional

The `unless` field accepts expressions that must evaluate to `false` for the step to run:

```yaml
steps:
  - name: run-tests
    type: validation
    unless: $SKIP_TESTS == true
    env:
      SKIP_TESTS: "false"
```

## Expression Syntax

### Boolean Literals

```yaml
if: true      # Always runs
if: false     # Never runs
```

### Variable Checks

Check if a variable exists and is non-empty:

```yaml
if: DEPLOY_ENABLED
env:
  DEPLOY_ENABLED: "true"
```

### Comparison Operators

**Numeric comparisons:**
```yaml
if: $REPLICAS > 3
if: $CPU_LIMIT >= 2
if: $MEMORY < 1024
```

**String comparisons:**
```yaml
if: $ENV == production
if: $VERSION != v1.0.0
```

**Supported operators:** `==`, `!=`, `<`, `>`, `<=`, `>=`

### String Operations

**Contains:**
```yaml
if: $MESSAGE contains error
if: production contains prod
```

**Starts with:**
```yaml
if: $BRANCH_NAME startsWith feature/
if: production startsWith prod
```

**Ends with:**
```yaml
if: $FILENAME endsWith .yaml
if: config.json endsWith .json
```

**Regex matching:**
```yaml
if: $VERSION matches ^v[0-9]+\.[0-9]+\.[0-9]+$
if: test123 matches test[0-9]+
```

### Step Status References

Check the result of previous steps:

```yaml
steps:
  - name: build
    type: validation

  - name: deploy-if-build-succeeded
    type: kubernetes
    if: build.success

  - name: notify-on-build-failure
    type: notification
    if: build.failed
```

**Status checks:**
- `step_name.success` / `step_name.succeeded` - Step completed successfully
- `step_name.failed` / `step_name.failure` - Step failed
- `step_name.skipped` - Step was skipped

### Variable Substitution

Use `$VAR` or `${VAR}` syntax for variable substitution:

```yaml
steps:
  - name: conditional-deploy
    type: kubernetes
    if: ${APP_NAME}-${ENVIRONMENT} == myapp-production
    env:
      APP_NAME: myapp
      ENVIRONMENT: production
```

## Examples

### Example 1: Environment-Based Deployment

```yaml
steps:
  - name: deploy-dev
    type: kubernetes
    namespace: dev
    if: $ENVIRONMENT == development
    env:
      ENVIRONMENT: development

  - name: deploy-staging
    type: kubernetes
    namespace: staging
    if: $ENVIRONMENT == staging
    env:
      ENVIRONMENT: staging

  - name: deploy-production
    type: kubernetes
    namespace: production
    if: $ENVIRONMENT == production
    env:
      ENVIRONMENT: production
```

### Example 2: Cleanup on Failure

```yaml
steps:
  - name: provision-infrastructure
    type: terraform

  - name: deploy-application
    type: kubernetes

  - name: cleanup-on-failure
    type: terraform
    when: on_failure
    path: ./terraform/cleanup
```

### Example 3: Skip Tests Conditionally

```yaml
steps:
  - name: build
    type: validation

  - name: run-unit-tests
    type: validation
    unless: $SKIP_TESTS == true
    env:
      SKIP_TESTS: "false"

  - name: run-integration-tests
    type: validation
    unless: $SKIP_TESTS == true
    if: $RUN_INTEGRATION == true
    env:
      SKIP_TESTS: "false"
      RUN_INTEGRATION: "true"
```

### Example 4: Feature Flags

```yaml
steps:
  - name: deploy-feature-a
    type: kubernetes
    if: $FEATURE_A_ENABLED == true
    env:
      FEATURE_A_ENABLED: "true"

  - name: deploy-feature-b
    type: kubernetes
    if: $FEATURE_B_ENABLED == true
    env:
      FEATURE_B_ENABLED: "false"
```

### Example 5: Version-Based Logic

```yaml
steps:
  - name: deploy-canary
    type: kubernetes
    if: $VERSION matches ^v[0-9]+\.[0-9]+\.0$
    env:
      VERSION: v2.1.0

  - name: deploy-full
    type: kubernetes
    unless: $VERSION matches ^v[0-9]+\.[0-9]+\.0$
    env:
      VERSION: v2.1.1
```

### Example 6: Combined Conditions

```yaml
steps:
  - name: security-scan
    type: security

  - name: deploy-to-production
    type: kubernetes
    when: on_success          # Only if all previous steps succeeded
    if: $ENVIRONMENT == production
    unless: $MAINTENANCE_MODE == true
    env:
      ENVIRONMENT: production
      MAINTENANCE_MODE: "false"
```

### Example 7: Notification Based on Results

```yaml
steps:
  - name: run-tests
    type: validation

  - name: notify-success
    type: notification
    when: on_success
    if: run-tests.success

  - name: notify-failure
    type: notification
    when: on_failure
    if: run-tests.failed
```

## Complete Workflow Example

See `examples/conditional-workflow.yaml` for a comprehensive example with multiple condition types.

## Execution Behavior

### Evaluation Order

Conditions are evaluated in this order:
1. **`when`** - Simple keyword check
2. **`unless`** - Must be false to continue
3. **`if`** - Must be true to continue

If any condition fails, the step is skipped.

### Skipped Steps

When a step is skipped:
- Marked with status `"skipped"` in the database
- Skip reason is recorded
- Displayed with ⏭️ icon in output
- Does not cause workflow failure
- Recorded in execution context for subsequent conditions

Example output:
```
⏭️  deploy-to-production (kubernetes) - SKIPPED: if condition '$ENVIRONMENT == production' is false
```

### Success vs Failure

- **Success:** Step executed and completed without error
- **Failed:** Step executed but returned an error
- **Skipped:** Step did not execute due to conditions

### Execution Context

The execution context tracks:
- Previous step statuses (`success`, `failed`, `skipped`)
- Previous step outputs (for future use)
- Environment variables
- Overall workflow status

## Environment Variables

### Step-Level Variables

Define variables specific to a step:

```yaml
steps:
  - name: my-step
    type: kubernetes
    if: $DEPLOY == true
    env:
      DEPLOY: "true"
      NAMESPACE: production
```

### Workflow-Level Variables (Future)

Currently, variables must be defined per step. Future enhancement will support workflow-level variables.

## Best Practices

### 1. Use `when` for Simple Cases

```yaml
# Good
- name: cleanup
  type: validation
  when: on_failure

# Overkill
- name: cleanup
  type: validation
  if: previous-step.failed
```

### 2. Combine Conditions for Complex Logic

```yaml
- name: deploy
  type: kubernetes
  when: on_success              # All previous steps OK
  if: $ENVIRONMENT == production  # Production environment
  unless: $MAINTENANCE_MODE == true  # Not in maintenance
```

### 3. Use Variable Checks for Feature Flags

```yaml
- name: new-feature
  type: kubernetes
  if: $ENABLE_NEW_FEATURE
  env:
    ENABLE_NEW_FEATURE: "true"
```

### 4. Reference Previous Steps for Dependencies

```yaml
- name: deploy-backend
  type: kubernetes

- name: deploy-frontend
  type: kubernetes
  if: deploy-backend.success
```

### 5. Use `unless` for Skip Logic

```yaml
# Good - clear intent to skip
- name: run-tests
  type: validation
  unless: $SKIP_TESTS

# Less clear
- name: run-tests
  type: validation
  if: $SKIP_TESTS != true
```

## Limitations

### Current Limitations

1. **No logical operators:** Cannot use `AND`, `OR`, `NOT` in expressions
2. **No complex expressions:** Only simple comparisons supported
3. **No nested conditions:** Cannot nest if/unless statements
4. **No workflow-level variables:** Variables must be per-step

### Workarounds

**Multiple conditions:**
```yaml
# Instead of: if: $A == true AND $B == false
# Use separate fields:
- name: my-step
  if: $A == true
  unless: $B == true
```

**Complex logic:**
```yaml
# Instead of complex expression
# Use multiple steps with conditions
- name: check-a
  type: validation
  if: $CONDITION_A

- name: check-b
  type: validation
  if: $CONDITION_B
```

## Future Enhancements

### Planned Features

1. **Logical operators:** Support `AND`, `OR`, `NOT` in expressions
2. **Workflow-level variables:** Define variables once for all steps
3. **Output references:** Use previous step outputs in conditions
4. **Custom functions:** Support custom condition functions
5. **Approval gates:** Interactive manual approval workflow
6. **Timeout conditions:** Skip after timeout

### Example (Future)

```yaml
workflow:
  variables:
    ENVIRONMENT: production
    DEPLOY_ENABLED: true

steps:
  - name: deploy
    type: kubernetes
    if: $ENVIRONMENT == production AND $DEPLOY_ENABLED
    outputs:
      - DEPLOYMENT_ID

  - name: verify
    type: validation
    if: $DEPLOY.DEPLOYMENT_ID != ""
    when: on_success
```

## Testing

Run condition evaluation tests:

```bash
go test -v ./internal/workflow -run TestExecutionContext
```

## Troubleshooting

### Step Always Skips

**Check:**
1. Variable is defined in `env` section
2. Variable name matches exactly (case-sensitive)
3. Comparison operator is correct
4. Value comparison uses correct type (string vs numeric)

**Debug:**
```yaml
# Add explicit env variable
env:
  DEBUG: "true"

# Use simple condition first
if: true  # Should always run
```

### Condition Not Evaluating Correctly

**Common issues:**
- Variable not defined: `if: $MISSING_VAR` → skipped
- String vs numeric: `"10" > 5` → numeric comparison works
- Quote handling: Use variables without quotes in conditions

**Example:**
```yaml
# Correct
if: $COUNT > 5
env:
  COUNT: "10"

# Incorrect
if: "$COUNT" > "5"  # Compares strings, not numbers
```

### Step Status Not Available

Make sure the referenced step:
1. Appears before the conditional step
2. Has a `name` field
3. Has already executed (not in same parallel group)

## References

- Implementation: `internal/workflow/conditions.go`
- Tests: `internal/workflow/conditions_test.go`
- Integration: `internal/workflow/executor.go`
- Examples: `examples/conditional-workflow.yaml`
- Gap Analysis: `GAP_ANALYSIS-2025-09-30-2.md` (Gap 4.1.1)
