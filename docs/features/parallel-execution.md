# Parallel Step Execution

This document describes the parallel step execution feature in innominatus, which allows workflow steps to run concurrently for improved performance.

## Overview

By default, workflow steps execute sequentially (one after another). With parallel execution enabled, independent steps can run concurrently, significantly reducing total workflow execution time.

**Example performance improvement:**
- Sequential: 3 steps × 2 minutes = 6 minutes total
- Parallel: max(2min, 2min, 2min) = 2 minutes total

## Configuration

### Basic Parallel Execution

Mark steps as parallel using the `parallel: true` field:

```yaml
steps:
  - name: validate-syntax
    type: validation
    parallel: true

  - name: security-scan
    type: security
    parallel: true

  - name: policy-check
    type: policy
    parallel: true
```

All steps marked with `parallel: true` will execute concurrently in the same group.

### Explicit Parallel Groups

Use `parallelGroup` to organize steps into multiple parallel execution groups that run sequentially:

```yaml
steps:
  # Group 1: Validation steps (run in parallel)
  - name: validate-syntax
    type: validation
    parallelGroup: 1

  - name: security-scan
    type: security
    parallelGroup: 1

  # Group 2: Deployment steps (run in parallel after Group 1)
  - name: deploy-backend
    type: kubernetes
    parallelGroup: 2

  - name: deploy-frontend
    type: kubernetes
    parallelGroup: 2
```

**Execution order:**
1. Group 1 steps execute in parallel (validate-syntax and security-scan)
2. Wait for all Group 1 steps to complete
3. Group 2 steps execute in parallel (deploy-backend and deploy-frontend)

### Sequential Steps

Steps without `parallel` or `parallelGroup` fields execute sequentially:

```yaml
steps:
  - name: initialize
    type: terraform
    parallel: false  # Explicit sequential

  - name: cleanup
    type: validation  # Implicit sequential (no parallel field)
```

## Execution Behavior

### Mixed Sequential and Parallel

You can mix sequential and parallel steps:

```yaml
steps:
  # Step 1: Sequential (runs first)
  - name: initialize
    type: terraform
    parallel: false

  # Steps 2-3: Parallel (run together after step 1)
  - name: task-a
    type: validation
    parallel: true

  - name: task-b
    type: security
    parallel: true

  # Step 4: Sequential (runs after parallel steps complete)
  - name: finalize
    type: validation
    parallel: false
```

**Execution flow:**
```
initialize (2min)
  ↓
task-a (1min) | task-b (1min)  ← Run in parallel
  ↓
finalize (1min)
  ↓
Total: 4 minutes (instead of 5 minutes if all sequential)
```

### Concurrency Limits

The executor enforces a maximum concurrency limit (default: 5 concurrent steps) to prevent resource exhaustion. Steps beyond this limit will wait for available slots.

### Error Handling

- If any step in a parallel group fails, the entire group fails
- Remaining steps in the group continue to completion
- The workflow stops after the failed group
- Sequential steps after a failure do not execute

## Examples

See `examples/parallel-workflow.yaml` for complete examples:
- Phase-based parallel execution with groups
- Simple parallel execution
- Mixed sequential and parallel workflows

## Best Practices

### When to Use Parallel Execution

**Good use cases:**
- Independent validation checks (syntax, security, policy)
- Deploying multiple independent services
- Provisioning independent resources (databases, caches, storage)
- Running tests in different environments

**Avoid parallelizing:**
- Steps with dependencies (use sequential or groups)
- Steps that modify shared state
- Steps that compete for exclusive resources

### Organizing Parallel Groups

1. **Pre-deployment validations** (Group 1)
   - Syntax validation
   - Security scanning
   - Policy checks

2. **Infrastructure provisioning** (Sequential)
   - Database setup
   - Networking configuration

3. **Application deployment** (Group 2)
   - Backend services
   - Frontend applications
   - Worker processes

4. **Post-deployment tasks** (Sequential)
   - Health checks
   - Monitoring setup
   - Smoke tests

### Performance Optimization

**Calculate potential speedup:**
```
Sequential time = sum(all step durations)
Parallel time = max(group durations) for each group
Speedup = Sequential time / Parallel time
```

**Example:**
- 3 validation steps (2min each) = 6min sequential
- Run in parallel = 2min parallel
- **Speedup: 3x faster**

## Backward Compatibility

Workflows without `parallel` or `parallelGroup` fields execute sequentially (original behavior). This ensures existing workflows continue to work without modification.

## Testing

The implementation includes comprehensive tests to verify:
- Parallel steps actually run concurrently (timing verification)
- Sequential steps run in order
- Mixed workflows execute correctly
- Parallel groups execute in the correct order
- Error handling in parallel execution
- All parallel steps complete successfully

Run tests:
```bash
go test -v ./internal/workflow -run TestParallel
```

## Implementation Details

### Architecture

- **Step Grouping**: `buildStepExecutionGroups()` analyzes workflow steps and creates execution groups
- **Parallel Execution**: `executeStepGroupParallel()` uses goroutines to run steps concurrently
- **Synchronization**: WaitGroups ensure all steps in a group complete before proceeding
- **Error Handling**: Error channels collect failures from parallel steps

### Fields

**Step struct fields:**
```go
type Step struct {
    // ... existing fields ...
    Parallel      bool     // Run this step in parallel
    DependsOn     []string // Future: explicit dependencies
    ParallelGroup int      // Group ID for phased parallel execution
}
```

### Database Tracking

All parallel steps are tracked in the database with:
- Individual step execution records
- Start and completion timestamps
- Status updates (pending → running → completed/failed)
- Error messages for failed steps

## Future Enhancements

### Planned Features

1. **Explicit Dependencies**: Use `dependsOn` field to specify step dependencies
2. **Dynamic Parallelism**: Auto-detect independent steps
3. **Resource Quotas**: Limit parallelism based on resource availability
4. **Dependency Graph Visualization**: Display workflow DAG in UI
5. **Conditional Execution**: Skip steps based on previous results

### Example with Dependencies (Future)

```yaml
steps:
  - name: provision-database
    type: terraform

  - name: run-migrations
    type: database-migration
    dependsOn: [provision-database]  # Waits for database

  - name: deploy-backend
    type: kubernetes
    dependsOn: [run-migrations]

  - name: deploy-frontend
    type: kubernetes
    parallel: true  # Can run in parallel with backend
```

## References

- Gap Analysis: `GAP_ANALYSIS-2025-09-30-2.md` (Gap 4.3.1)
- Implementation: `internal/workflow/executor.go`
- Tests: `internal/workflow/executor_test.go`
- Examples: `examples/parallel-workflow.yaml`
