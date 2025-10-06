# How to Query Workflow Status

**Goal**: Check the status and details of workflow executions.

**Time**: 1 minute

## Quick Steps

### View All Workflows

**Web UI**:
```
You: "show me recent workflows"
You: "list all workflows"
```

**CLI**:
```bash
./innominatus-ctl chat --one-shot "show me recent workflows"
```

## Expected Response

```
Recent workflows:
1. deploy-app (my-app) - completed in 2m 34s
2. db-lifecycle (database-1) - running (1m 12s elapsed)
3. ephemeral-env (test-env) - completed in 45s
4. observability-setup (monitoring) - failed after 1m 05s
```

## Get Specific Workflow Details

```
You: "show workflow execution 123"
You: "what's the status of workflow 123"
You: "tell me about workflow 123"
```

**Response includes**:
- Workflow name and application
- Current status (pending, running, completed, failed)
- Execution time
- Steps completed
- Error messages (if failed)

## Filter by Status

```
You: "show running workflows"
You: "list failed workflows"
You: "what workflows are pending"
```

## Filter by Application

```
You: "show workflows for my-app"
You: "what workflows ran for database-1"
```

## Understanding Workflow States

- **pending**: Queued, waiting to start
- **running**: Currently executing steps
- **completed**: Successfully finished all steps
- **failed**: Encountered error during execution

## Troubleshooting Failed Workflows

When a workflow fails:

```
You: "show workflow 123"
```

The AI will display:
- Which step failed
- Error message
- Execution logs
- Suggested fixes

Follow up with:
```
You: "why did this fail?"
You: "how do I fix this?"
```

## Related Tasks

- [Monitor Deployments](./monitor-deployments.md)
- [Rollback Deployments](./rollback-deployments.md)
- [View Platform Statistics](./view-statistics.md)
