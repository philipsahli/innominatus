# Using the AI Assistant

**Ask questions in natural language instead of memorizing commands**

---

## Overview

The innominatus AI Assistant lets you interact with the platform using conversational language. Instead of remembering CLI commands or API endpoints, just ask:

- "list my applications"
- "show me failed workflows"
- "generate a score spec for a python app with postgres"
- "what are golden paths?"

The AI combines:
- **Live platform data** - Your applications, workflows, and resources
- **Documentation knowledge** - Guides, references, and best practices
- **Natural language** - No need to memorize syntax

---

## Getting Started

### Access via Web UI

1. Open your innominatus instance (e.g., http://localhost:8081)
2. Login with your credentials
3. Click **"AI Assistant"** in the navigation menu (🤖 icon)
4. Start typing your question

### Access via CLI

```bash
# Interactive chat session
./innominatus-ctl chat

# One-shot question
./innominatus-ctl chat --one-shot "list my applications"

# Save conversation to file
./innominatus-ctl chat --output chat-log.txt
```

---

## What You Can Ask

### List Your Resources

**Applications:**
```
"list my applications"
"show all apps"
"what applications do I have"
```

**Workflows:**
```
"show recent workflows"
"list all workflows"
"what workflows are running"
```

**Resources:**
```
"show all resources"
"what databases are deployed"
"list postgres instances"
```

### Get Specific Details

**Application Details:**
```
"tell me about demo-app"
"show details for my-api"
"what resources does demo-app use"
```

**Workflow Status:**
```
"show workflow execution 123"
"what's the status of workflow 45"
"why did workflow 67 fail"
```

**Platform Statistics:**
```
"show platform statistics"
"how many apps are deployed"
"get platform overview"
```

### Generate Score Specifications

**Basic Apps:**
```
"generate a score spec for a python app"
"create a spec for node.js with redis"
"make a spec for a go service with postgres"
```

**With Specific Requirements:**
```
"generate a score spec for a python app with:
- postgres database version 15
- redis cache
- 512MB memory limit
- exposed on /api route"
```

**After Generation:**
```
"deploy it"
"save it to my-app.yaml"
```

### Learn About Features

**Concepts:**
```
"what are golden paths?"
"explain the demo environment"
"how does deployment work"
```

**How-To:**
```
"how do I deploy an app?"
"how do I create an ephemeral environment?"
"how do I check workflow status?"
```

---

## Example Conversations

### Deploying Your First App

```
You: "I need to deploy a python app with postgres"

AI: "I'll generate a Score specification for you:

apiVersion: score.dev/v1b1
metadata:
  name: python-app

containers:
  app:
    image: python:3.11-slim
    resources:
      requests:
        memory: "256Mi"
        cpu: "100m"

resources:
  database:
    type: postgres
    properties:
      version: "15"

Would you like me to deploy this?"

You: "yes, deploy it"

AI: "Application deployed successfully.
Workflow execution #45 started.
Status: running

You can check progress with: 'show workflow 45'"
```

### Checking Application Status

```
You: "list my applications"

AI: "You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources"

You: "tell me about demo-app"

AI: "demo-app (production):
• Status: running
• 5 resources: 2 postgres, 1 redis, 1 volume, 1 route
• Created: 2 days ago
• Last workflow: deploy-app (completed)"

You: "show me its workflows"

AI: "demo-app workflows:
1. deploy-app - completed (2m 34s)
2. db-lifecycle - running (1m 12s)
3. observability-setup - completed (45s)"
```

### Troubleshooting Failed Workflows

```
You: "show me failed workflows"

AI: "2 failed workflows:
1. deploy-app (test-app) - failed at provision step
   Error: Database connection timeout
2. db-lifecycle (api-gateway) - failed at backup step
   Error: S3 bucket not accessible

Would you like details for either workflow?"

You: "show me details for workflow 1"

AI: "Workflow #123 (deploy-app):
• Application: test-app
• Status: failed
• Duration: 1m 23s
• Steps:
  ✓ validate (completed)
  ✓ create-namespace (completed)
  ✗ provision-database (failed)

Error: Database connection timeout after 60s
Suggestion: Check database credentials and network connectivity"
```

---

## Understanding AI Responses

### When Tools Are Used

The AI uses **tools** to fetch live platform data when you:
- Ask to "list", "show", "get" something
- Request current status or state
- Want to see what's deployed

**Indicators:**
- Response includes specific data (app names, workflow IDs)
- Shows current timestamps and status
- Lists actual resources

### When Documentation Is Used

The AI retrieves **documentation** when you:
- Ask "what is...", "how does...", "why..."
- Request explanations or concepts
- Need guidance or best practices

**Indicators:**
- Response explains concepts
- Includes examples and patterns
- Provides step-by-step instructions

### Multi-Turn Context

The AI remembers your conversation:

```
You: "list my applications"
AI: "You have 3 applications: demo-app, test-service, api-gateway"

You: "tell me about the first one"  ← AI knows you mean demo-app
AI: "demo-app (production): 5 resources..."

You: "show its workflows"  ← AI still knows you mean demo-app
AI: "demo-app workflows: deploy-app (completed)..."
```

---

## Tips for Better Results

### Be Specific

❌ **Vague:** "show me stuff"
✅ **Specific:** "list my applications in production"

❌ **Vague:** "what's wrong"
✅ **Specific:** "show me failed workflows for demo-app"

### Use Natural Language

You don't need exact commands:

```
✅ "list my applications"
✅ "show all my apps"
✅ "what applications do I have"
✅ "can you list the apps"
```

All of these work!

### Ask Follow-Up Questions

Don't start over - continue the conversation:

```
You: "list my applications"
AI: [shows 3 apps]

You: "which one uses the most resources?"
AI: [analyzes and responds]

You: "show me its workflows"
AI: [shows workflows for that app]
```

### Request Specific Formats

```
"generate a score spec for python with postgres"
"show me workflows as a table"
"list applications with their resource counts"
```

---

## Common Use Cases

### Daily Development Workflow

**Morning Check:**
```
"show platform statistics"
"list my applications"
"show recent workflows"
```

**During Development:**
```
"generate a spec for [your app]"
"deploy it to staging"
"show deployment status"
```

**Debugging:**
```
"show failed workflows"
"why did workflow 123 fail"
"check logs for demo-app"
```

### Learning the Platform

**Discover Features:**
```
"what are golden paths?"
"how does OIDC authentication work?"
"what resources can I use in score specs?"
```

**Get Examples:**
```
"show me an example score spec"
"how do I add a postgres database?"
"what's a good memory limit for a python app?"
```

### Quick Operations

**Status Checks:**
```
"is my-app running?"
"show workflow status for 45"
"what's using the most resources?"
```

**Quick Deployments:**
```
"deploy my-app.yaml"
"update demo-app configuration"
"rollback api-gateway"  (if supported)
```

---

## Troubleshooting

### "AI service is not enabled"

**Cause:** AI assistant not configured by Platform Team

**What to do:**
1. Contact your Platform Team
2. Request AI assistant access
3. They need to set `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`

### "Failed to fetch" (Web UI)

**Cause:** Session expired or not logged in

**Fix:**
1. Refresh the page
2. Login again
3. Retry your question

### "Error executing tool: 401 Unauthorized" (CLI)

**Cause:** Missing or invalid API key

**Fix:**
```bash
# Get API key from Platform Team or generate one
export IDP_API_KEY="your-api-key"

# Retry
./innominatus-ctl chat
```

### Responses are slow

**Normal:** AI responses take 2-4 seconds (includes API calls)

**Too slow (>10 seconds)?**
- Check your internet connection
- Contact Platform Team (may be rate limiting or API issues)

### AI doesn't understand my question

**Try:**
- Rephrase more specifically
- Break into smaller questions
- Use keywords like "list", "show", "get", "create"

**Examples:**
- ❌ "stuff" → ✅ "applications"
- ❌ "thing broken" → ✅ "failed workflows"
- ❌ "make it work" → ✅ "deploy this spec"

---

## Privacy & Security

### What's Sent to AI Providers

**Your messages** and **platform data** are sent to:
- Anthropic (if using Claude)
- OpenAI (if using GPT)

**Not stored persistently** by AI providers (per their policies)

### Authentication

**All operations use your credentials:**
- Web UI: Your session token
- CLI: Your API key (`$IDP_API_KEY`)

**You can only:**
- See applications you have access to
- Deploy to environments you're authorized for
- Delete resources you own

The AI cannot perform actions beyond your permissions.

### Audit Trail

**All AI actions are logged:**
- API calls made by tools
- Deployments triggered
- Resources accessed

Your Platform Team can see AI activity in audit logs.

---

## Getting Help

### In-App Help

**Web UI:**
- Click the "?" icon in the AI assistant
- Shows quick tips and examples

**CLI:**
```bash
./innominatus-ctl chat --help
```

### Ask the AI

```
"how do I use the AI assistant?"
"show me examples of what I can ask"
"what commands are available?"
```

The AI can help you learn how to use it!

### Contact Platform Team

**For issues:**
- AI not responding
- Permission errors
- Feature requests

**They can:**
- Check AI service status
- Review your access permissions
- Enable additional features

---

## See Also

- **[CLI Reference](cli-reference.md)** - All CLI commands including `chat`
- **[First Deployment](first-deployment.md)** - Deploy your first application
- **[Troubleshooting Guide](troubleshooting.md)** - Common issues and fixes
- **[Score Specifications](score-specifications.md)** - Writing Score specs

**Full AI Assistant Documentation:** [docs/ai-assistant/](../ai-assistant/README.md)

---

*The AI assistant makes platform operations as easy as having a conversation.* 💬
