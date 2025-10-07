# Tutorial: Getting Started with the AI Assistant

**Learning goal**: By the end of this tutorial, you'll understand how to use the innominatus AI assistant to manage your applications.

**Time**: 10 minutes

**Prerequisites**:
- innominatus server running
- API keys configured (`OPENAI_API_KEY` and `ANTHROPIC_API_KEY`)
- At least one application deployed (or access to demo environment)

## What You'll Learn

- How to access the AI assistant
- Basic conversation patterns
- Using tools to interact with your platform
- Understanding AI responses

## Step 1: Access the AI Assistant

### Option A: Web UI
1. Open your browser to `http://localhost:8081`
2. Login with your credentials
3. Click "AI Assistant" in the navigation menu (🤖 icon)

### Option B: CLI
```bash
# Interactive chat
./innominatus-ctl chat

# One-shot question
./innominatus-ctl chat --one-shot "list my applications"
```

## Step 2: Your First Question

Let's start with a simple question to list your applications:

**You**: `list my applications`

**AI**:
```
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources
```

**What happened?**
The AI recognized your intent, used the `list_applications` tool, fetched data from the API, and presented it in a readable format.

## Step 3: Get Application Details

Now let's dig deeper into one application:

**You**: `tell me about demo-app`

**AI**:
```
demo-app (production):
• 5 resources: 2 postgres, 1 redis, 1 volume, 1 route
• Created: 2 days ago
• Status: running
• Environment: production
```

**What happened?**
The AI used the `get_application` tool with parameter `app_name: "demo-app"` to fetch detailed information.

## Step 4: View Workflows

Check what's happening on your platform:

**You**: `show me recent workflows`

**AI**:
```
Recent workflows:
1. deploy-app (demo-app) - completed (2m 34s)
2. db-lifecycle (api-gateway) - running (1m 12s)
3. ephemeral-env (test-env) - completed (45s)
```

## Step 5: Ask Questions About Features

The AI can also answer questions using documentation:

**You**: `what are golden paths?`

**AI**:
```
Golden paths are pre-defined workflows that solve common developer needs:
• deploy-app: Full application deployment
• ephemeral-env: Temporary test environments
• db-lifecycle: Database management
• observability-setup: Monitoring setup

See /goldenpaths for all available paths.
```

**What happened?**
The AI retrieved relevant documentation using RAG (Retrieval-Augmented Generation) and provided a concise answer.

## Step 6: Generate a Score Specification

Ask the AI to create a Score spec:

**You**: `generate a score spec for a node.js app with postgres`

**AI**:
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

💡 Tip: Use `/save spec.yaml` in CLI to save the spec.

## Understanding AI Responses

### Response Elements

1. **Direct answers**: Short, concise responses (2-3 sentences)
2. **Bullet points**: Used for lists and structured data
3. **Citations**: Sources from documentation (shown at bottom)
4. **Tool usage**: Indicated by data fetched from platform

### When Tools Are Used

The AI automatically uses tools when you:
- Ask to "list", "show", "get", "view" something
- Request to "deploy", "delete", "create" something
- Ask "what's running" or "what's the status"

### When Documentation Is Used

The AI retrieves documentation when you:
- Ask "what is...", "how does...", "why..."
- Request explanations or concepts
- Need guidance or best practices

## Common Patterns

### List Resources
```
"list my applications"
"show me workflows"
"what resources are deployed"
"get platform statistics"
```

### Get Details
```
"tell me about demo-app"
"show workflow execution 123"
"what's the status of my-app"
```

### Generate Specs
```
"create a score spec for..."
"generate a spec for python app with redis"
```

### Ask Questions
```
"how do I deploy an app?"
"what golden paths are available?"
"explain the demo environment"
```

## Next Steps

Now that you understand the basics:

1. **Tutorial**: [Using AI for Deployments](./deployment-tutorial.md)
2. **How-to**: [Query Workflow Status](../how-to-guides/query-workflows.md)
3. **Reference**: [Available Tools](../reference/tools-reference.md)
4. **Explanation**: [How Tool Calling Works](../explanations/tool-calling-architecture.md)

## Troubleshooting

### "AI service is not enabled"
- Ensure `OPENAI_API_KEY` and `ANTHROPIC_API_KEY` are set
- Restart the server after setting environment variables

### "Failed to fetch"
- Check that you're logged in (web UI)
- Verify server is running on `http://localhost:8081`

### Responses are too long
The AI is configured to be concise. If responses are too long, this may be a configuration issue. See [Configuration Reference](../reference/configuration.md).

## Summary

You've learned how to:
✅ Access the AI assistant (web UI and CLI)
✅ Ask questions and get concise answers
✅ Use tools to interact with your platform
✅ Generate Score specifications
✅ Understand when tools vs documentation are used

The AI assistant combines live platform data with documentation knowledge to help you work more efficiently!
