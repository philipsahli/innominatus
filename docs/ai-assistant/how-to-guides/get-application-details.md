# How to Get Application Details

**Goal**: View detailed information about a specific application.

**Time**: 1 minute

## Quick Steps

```
You: "tell me about my-app"
You: "show details for my-app"
You: "get info on my-app"
```

## Expected Response

```
my-app (production):
• Status: running
• Environment: production
• Created: 2 days ago
• Resources:
  - postgres-db (postgres)
  - redis-cache (redis)
  - app-storage (volume)
  - api-route (route)
• Container: node:18-alpine
• Memory: 512Mi (limit), 256Mi (request)
• CPU: 500m (limit), 100m (request)
```

## View Score Specification

```
You: "show the score spec for my-app"
You: "what's the score specification for my-app"
```

The AI will display the complete YAML spec used for deployment.

## Check Resource Details

### All Resources
```
You: "what resources does my-app use"
You: "list resources for my-app"
```

### Specific Resource Type
```
You: "what databases does my-app have"
You: "show postgres resources for my-app"
```

## View Deployment History

```
You: "show deployment history for my-app"
You: "when was my-app last deployed"
```

## Compare Applications

```
You: "compare my-app and other-app"
You: "what's the difference between my-app and test-app"
```

## Get Status Summary

```
You: "is my-app healthy"
You: "what's the status of my-app"
```

## Understanding the Output

### Status Values
- **running**: Application is operational
- **pending**: Deployment in progress
- **failed**: Deployment or runtime error
- **stopped**: Application is not running

### Resource Types
- **postgres**: PostgreSQL database
- **redis**: Redis cache
- **volume**: Persistent storage
- **route**: Ingress route/URL
- **s3**: Object storage

### Resource Limits
- **Requests**: Minimum guaranteed resources
- **Limits**: Maximum allowed resources

## Related Tasks

- [List Applications](./list-applications.md)
- [Deploy with AI](./deploy-with-ai.md)
- [Query Workflows](./query-workflows.md)
- [View Platform Statistics](./view-statistics.md)
