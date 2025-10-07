# How to Deploy Applications with AI

**Goal**: Deploy an application using AI-generated or existing Score specifications.

**Time**: 2-3 minutes

## Option 1: Deploy with AI-Generated Spec

### Step 1: Generate the Spec

```
You: "generate a score spec for a python fastapi app with postgres"
```

The AI will create a complete Score specification.

### Step 2: Review the Spec

The AI shows you the generated YAML. Review it for:
- Correct application name
- Appropriate resource limits
- Required dependencies

### Step 3: Deploy

```
You: "deploy this spec"
```

The AI will use the `deploy_application` tool to deploy it.

## Option 2: Deploy Existing Spec

### If You Have a YAML File

**Web UI**: Copy and paste the spec content, then:
```
You: "deploy this spec:
<paste your yaml here>
"
```

**CLI**: Use the generate-spec command:
```bash
./innominatus-ctl chat --generate-spec "Node.js app with Redis" -o my-app.yaml

# Then deploy via API
curl -X POST http://localhost:8081/api/specs \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/yaml" \
  --data-binary @my-app.yaml
```

## Deployment Response

Expected response:
```
Application deployed successfully.
Workflow execution #45 started.
Status: running
```

## Verify Deployment

Check the deployment:
```
You: "show workflow 45"
You: "what's the status of my-app"
```

## Troubleshooting

### "Deployment failed"
The AI will show the error message. Common issues:
- Invalid YAML syntax
- Missing required fields
- Resource conflicts

Ask: "what went wrong?" for explanation.

### Deployment is Slow
```
You: "show me running workflows"
```

Check if other deployments are in progress.

## Best Practices

1. **Review before deploying**: Always check the generated spec
2. **Use golden paths**: For production, prefer `./innominatus-ctl run deploy-app spec.yaml`
3. **Test in staging**: Deploy to staging environment first
4. **Monitor workflows**: Track deployment progress with workflow queries

## Related Tasks

- [Generate Score Specifications](./generate-specs.md)
- [Monitor Deployments](./monitor-deployments.md)
- [Rollback Deployments](./rollback-deployments.md)
