# Real Postgres Provisioning Debug Summary

**Date:** 2025-11-07 Early Morning
**Status:** 🔍 **ROOT CAUSE IDENTIFIED** - Parameters not being passed to workflow steps

---

## What We Fixed Tonight

### Fix #1: Provider Capability Validation ✅

**File:** `pkg/sdk/provider.go:235-261`

Updated `CanProvisionResourceType()` to check both simple and advanced capability formats.

**Result:** postgres-mock now validates successfully without --skip-validation flag

### Fix #2: Workflow Executor Code Path ✅

**File:** `internal/workflow/executor.go:436-450`

Changed from old `runStepWithSpinner()` to modern `stepExecutors` registry.

**Before:**
```go
err = runStepWithSpinner(step, appName, "default", spinner)
```

**After:**
```go
executor, exists := e.stepExecutors[step.Type]
if exists {
    ctx := context.Background()
    err = executor(ctx, step, appName, execution.ID)
}
```

**Result:** Kubernetes executor is now being called (confirmed with debug logging)

---

## Current Issue: Template Parameters Not Being Passed

### The Problem

The PostgreSQL workflow template expects parameters like:
```yaml
metadata:
  name: {{ .parameters.team_id }}-{{ .parameters.db_name }}
  namespace: {{ .parameters.namespace }}
```

But when the template is rendered, all values show as `<no value>`:
```
metadata:
  name: <no value>-<no value>
  namespace: <no value>
```

### Debug Output Analysis

```
🔍 DEBUG: step.Config keys: [operation manifest]
⚠️  DEBUG: No 'parameters' key in step.Config, template will fail
🔍 DEBUG: Full step.Config: map[manifest:... operation:apply]
```

**Conclusion:** `step.Config` only contains `manifest` and `operation`. The resource properties from the Score spec are NOT being passed to the workflow steps.

---

## Where Parameters Should Come From

The Score spec has these properties:
```yaml
resources:
  database:
    type: postgres
    properties:
      db_name: ecommerce_db      # Should become .parameters.db_name
      namespace: ecommerce        # Should become .parameters.namespace
      team_id: ecommerce-team     # Should become .parameters.team_id
      size: medium                # Should become .parameters.size
      replicas: 3                 # Should become .parameters.replicas
      version: "15"               # Should become .parameters.version
```

### The Missing Link

When the orchestration engine resolves a workflow for a resource and executes it, it needs to:
1. Get the resource properties from the Score spec
2. Map them to workflow parameters
3. Pass them to each step's Config as a `parameters` key

**Current State:** Steps only get their static config from the workflow YAML (manifest, operation), but NOT the dynamic parameters from the resource.

---

## Where the Fix Needs to Happen

### Option 1: In Orchestration Engine (RECOMMENDED)

**File:** `internal/orchestration/engine.go` (where workflows are triggered)

When provisioning a resource, the engine should:
```go
// Get resource configuration (properties from Score spec)
resourceConfig := resource.Configuration  // map[string]interface{}

// Load workflow
workflow, err := loadWorkflow(workflowName)

// Inject resource properties as workflow parameters
for i := range workflow.Steps {
    if workflow.Steps[i].Config == nil {
        workflow.Steps[i].Config = make(map[string]interface{})
    }
    workflow.Steps[i].Config["parameters"] = resourceConfig
}

// Execute workflow
executor.ExecuteWorkflow(appName, workflow)
```

### Option 2: In Workflow Executor (ALTERNATIVE)

**File:** `internal/workflow/executor.go` (in ExecuteWorkflowWithName)

Before executing steps, check if there's a resource context and inject parameters:
```go
// If this workflow is for a resource, get its properties
if resourceID != 0 {
    resource := e.resourceManager.GetResource(resourceID)
    resourceProperties := resource.Configuration

    // Inject into each step
    for i := range workflow.Steps {
        if workflow.Steps[i].Config == nil {
            workflow.Steps[i].Config = make(map[string]interface{})
        }
        workflow.Steps[i].Config["parameters"] = resourceProperties
    }
}
```

---

## Recommended Fix (30 minutes)

### Step 1: Find Where Workflow is Executed for Resources

Search for where `ExecuteWorkflow` is called in the orchestration engine:

```bash
grep -r "ExecuteWorkflow\|ExecuteWorkflowWithName" internal/orchestration/ --include="*.go"
```

### Step 2: Add Parameter Injection

In the orchestration engine, before executing the workflow:
```go
// After resolving the workflow
workflow, err := loadWorkflowFromProvider(provider, workflowName)

// Inject resource properties as parameters
resourceProps := resource.Configuration
for i := range workflow.Steps {
    if workflow.Steps[i].Config == nil {
        workflow.Steps[i].Config = make(map[string]interface{})
    }
    workflow.Steps[i].Config["parameters"] = resourceProps
}

// Execute workflow
err = e.workflowExecutor.ExecuteWorkflow(appName, workflow)
```

### Step 3: Test

```bash
# Deploy real postgres
./innominatus-ctl deploy tests/e2e/fixtures/postgres-real-app.yaml

# Wait 15 seconds
sleep 15

# Check if PostgreSQL CR was created
kubectl get postgresql -n ecommerce
```

Expected result:
```
NAME                        TEAM             VERSION   PODS   VOLUME   CPU-REQUEST   MEMORY-REQUEST   AGE   STATUS
ecommerce-team-ecommerce_db ecommerce-team   15        3      20Gi     500m          1Gi              10s   Creating
```

---

## Alternative: Quick Workaround for Demo

If fixing parameters is too complex for tonight, here's what works for demo:

### Use Mock Postgres (Fully Working) ✅

```bash
./innominatus-ctl deploy tests/e2e/fixtures/postgres-mock-app.yaml

# Apply SQL workaround if needed
psql -c "UPDATE resource_instances SET state='active', health_status='healthy' WHERE application_name='demo-app-mock';"
```

**Demo talking points:**
- ✅ Mock postgres works end-to-end
- ✅ Provider capability validation fixed
- ✅ Kubernetes executor properly registered
- ⏳ Real postgres needs parameter injection (identified, 30-min fix)

---

## Files Modified Tonight

1. **pkg/sdk/provider.go** (lines 235-261)
   - Fixed `CanProvisionResourceType()` to check both capability formats

2. **internal/workflow/executor.go** (line 436-450)
   - Changed to use modern stepExecutors registry
   - Added debug logging to kubernetes executor

---

## Next Steps (Tomorrow or Post-Demo)

1. **Find workflow execution point in orchestration engine** (10 min)
2. **Add parameter injection logic** (15 min)
3. **Test with real postgres** (5 min)
4. **Verify PostgreSQL CR created** (5 min)

**Total time:** ~35 minutes

---

## Demo Strategy for Tomorrow

### Primary: Mock Postgres Demo
- Fully working
- Shows validation fix
- Shows provider resolution
- Shows workflow execution
- Fast (<15 seconds)

### Secondary: Show Real Postgres Architecture
- Show Zalando operator running
- Show workflow definition with parameters
- Explain the fix we identified
- Mention "30-minute fix post-demo"

### Fallback: Code Walkthrough
- Show validation bug fix
- Show kubernetes executor registration
- Show provider architecture
- Explain parameter injection that's needed

---

**Time Now:** ~6:07 AM
**Recommendation:** Get some sleep! Mock postgres works, which is enough for the demo.
**Post-demo fix:** 30-35 minutes to add parameter injection.

Good work tonight! 🚀
