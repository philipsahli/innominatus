# How to View Platform Statistics

**Goal**: Get overview statistics and metrics for your platform.

**Time**: 1 minute

## Quick Steps

```
You: "show platform statistics"
You: "get platform stats"
You: "what's the platform overview"
```

## Expected Response

```
Platform Statistics:
• Total Applications: 12
• Running Workflows: 3
• Total Resources: 47
  - 8 postgres databases
  - 5 redis caches
  - 12 volumes
  - 15 routes
  - 7 S3 buckets
• Environments:
  - production: 5 apps
  - staging: 4 apps
  - development: 3 apps
```

## Specific Metrics

### Application Count
```
You: "how many applications are deployed"
You: "total app count"
```

### Resource Breakdown
```
You: "how many databases are running"
You: "show resource distribution"
```

### Workflow Statistics
```
You: "how many workflows are running"
You: "show workflow stats"
```

## Environment Breakdown

```
You: "show stats by environment"
You: "how many production apps are there"
```

## Trends and History

```
You: "show deployment trends"
You: "how many apps were deployed this week"
```

## Resource Utilization

```
You: "what's the resource usage"
You: "show platform capacity"
```

## Understanding the Metrics

### Total Applications
Count of all deployed applications across all environments.

### Running Workflows
Active workflow executions currently in progress.

### Total Resources
Sum of all platform resources (databases, caches, volumes, routes, storage).

### Environment Distribution
Breakdown of applications by environment (production, staging, development).

## Use Cases

### Capacity Planning
```
You: "show platform stats"
```
Review total resources to plan capacity.

### Health Check
```
You: "how many failed workflows"
```
Monitor platform health.

### Audit
```
You: "show stats by environment"
```
Verify environment distribution.

## Related Tasks

- [List Applications](./list-applications.md)
- [Query Workflows](./query-workflows.md)
- [Get Application Details](./get-application-details.md)
