# How to Generate Score Specifications

**Goal**: Use AI to generate Score specifications for your applications.

**Time**: 2 minutes

## Basic Generation

### Simple Application

```
You: "generate a score spec for a node.js app"
You: "create a spec for python fastapi application"
```

### With Dependencies

```
You: "generate a score spec for a python app with postgres"
You: "create a spec for node.js app with redis and postgres"
```

### Specific Requirements

```
You: "generate a score spec for a java spring boot app with 2GB memory and postgres database"
```

## Expected Response

The AI generates a complete YAML specification:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: nodejs-app

containers:
  app:
    image: node:18-alpine
    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"
      limits:
        memory: "512Mi"
        cpu: "500m"

resources:
  database:
    type: postgres
    properties:
      version: "15"
```

## Customizing Generated Specs

### Adjust Resource Limits

```
You: "increase memory to 1GB"
You: "add more CPU resources"
```

### Add Dependencies

```
You: "add redis cache to this spec"
You: "include volume storage"
```

### Change Container Image

```
You: "use node:20 instead"
You: "switch to python:3.11-slim"
```

## Saving Specs

### Web UI
Copy the YAML from the response and save locally.

### CLI
```bash
# Generate and save in one command
./innominatus-ctl chat --generate-spec "node.js app with postgres" -o my-app.yaml

# Or redirect output
./innominatus-ctl chat --one-shot "generate a score spec for python app with redis" > app.yaml
```

## Deploying Generated Specs

After generation:

```
You: "deploy this spec"
```

Or via CLI:
```bash
./innominatus-ctl run deploy-app my-app.yaml
```

## Best Practices

1. **Be specific**: Include language version, dependencies, resource needs
2. **Review before deploying**: Always check the generated YAML
3. **Iterate**: Ask for modifications if needed
4. **Save specs**: Keep generated specs in version control

## Example Workflows

### Generate + Review + Deploy
```
You: "generate a score spec for go app with postgres and redis"
[Review the output]
You: "increase memory to 1GB"
[Review again]
You: "deploy this spec"
```

### Generate for Different Environments
```
You: "generate a score spec for production node.js app with 4GB memory"
You: "now generate a staging version with 1GB memory"
```

## Common Patterns

### Microservice
```
You: "generate a score spec for a microservice with postgres, redis, and message queue"
```

### Static Site
```
You: "generate a score spec for a static site with nginx"
```

### Background Worker
```
You: "generate a score spec for a python celery worker with redis"
```

## Related Tasks

- [Deploy with AI](./deploy-with-ai.md)
- [List Applications](./list-applications.md)
- [Get Application Details](./get-application-details.md)
