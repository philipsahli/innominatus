# AI Assistant CLI Reference

**Purpose**: Complete command-line interface reference for the AI assistant.

## Overview

The innominatus CLI (`innominatus-ctl`) provides chat commands for interacting with the AI assistant from the terminal.

## Prerequisites

- innominatus server running (`./innominatus`)
- API key configured: `export IDP_API_KEY=<your-key>`
- CLI built: `go build -o innominatus-ctl cmd/cli/main.go`

## Commands

---

### chat

**Description**: Start an interactive chat session with the AI assistant.

**Usage**:
```bash
./innominatus-ctl chat [flags]
```

**Flags**:
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--one-shot` | string | - | Send a single message and exit |
| `--generate-spec` | string | - | Generate Score spec from description |
| `-o, --output` | string | - | Save generated spec to file |
| `--api-url` | string | `http://localhost:8081` | API base URL |
| `--api-key` | string | `$IDP_API_KEY` | API authentication key |

**Interactive Mode**:
```bash
./innominatus-ctl chat

> list my applications
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources

> tell me about demo-app
demo-app (production):
• Status: running
• 5 resources: 2 postgres, 1 redis, 1 volume, 1 route
• Created: 2 days ago

> exit
```

**One-Shot Mode**:
```bash
./innominatus-ctl chat --one-shot "list my applications"
```

Output:
```
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources
```

**Generate Spec Mode**:
```bash
./innominatus-ctl chat --generate-spec "node.js app with postgres" -o my-app.yaml
```

Output:
```
Score specification saved to my-app.yaml
```

File contents (`my-app.yaml`):
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

**Custom API URL**:
```bash
./innominatus-ctl chat --api-url https://innominatus.company.com --one-shot "list apps"
```

**Custom API Key**:
```bash
./innominatus-ctl chat --api-key "custom-key-123" --one-shot "list apps"
```

---

## Interactive Commands

When in interactive chat mode, these commands are available:

### exit / quit

**Description**: Exit the chat session

**Usage**:
```
> exit
> quit
```

### clear

**Description**: Clear the terminal screen

**Usage**:
```
> clear
```

### help

**Description**: Show help message with available commands

**Usage**:
```
> help
```

---

## Environment Variables

### IDP_API_KEY

**Description**: API authentication token

**Required**: Yes (unless provided via `--api-key`)

**How to Set**:
```bash
export IDP_API_KEY="your-api-key-here"
```

**How to Get**:
- Web UI: Navigate to Profile → Generate API Key
- CLI: Not available (must use Web UI)

### IDP_API_URL

**Description**: API base URL

**Required**: No

**Default**: `http://localhost:8081`

**How to Set**:
```bash
export IDP_API_URL="https://innominatus.company.com"
```

**Override**: Use `--api-url` flag to override

---

## Examples

### Basic Workflow

```bash
# Set API key
export IDP_API_KEY="your-key"

# Start interactive chat
./innominatus-ctl chat

# Ask questions
> list my applications
> tell me about demo-app
> show recent workflows
> generate a score spec for python app with postgres

# Exit
> exit
```

### Quick Queries

```bash
# One-shot queries (no interactive session)
./innominatus-ctl chat --one-shot "list apps"
./innominatus-ctl chat --one-shot "show workflows"
./innominatus-ctl chat --one-shot "platform stats"
```

### Spec Generation Workflow

```bash
# Generate spec
./innominatus-ctl chat --generate-spec "java spring boot with mysql" -o spring-app.yaml

# Review spec
cat spring-app.yaml

# Deploy via golden path
./innominatus-ctl run deploy-app spring-app.yaml
```

### Scripting with AI

```bash
#!/bin/bash
# Script to check platform status

export IDP_API_KEY="your-key"

# Get app count
APP_COUNT=$(./innominatus-ctl chat --one-shot "how many applications" | grep -oP '\d+')

echo "Total applications: $APP_COUNT"

# Get running workflows
./innominatus-ctl chat --one-shot "show running workflows"
```

### CI/CD Integration

```yaml
# .github/workflows/deploy.yml
name: Deploy with AI

on: push

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Generate Score Spec
        run: |
          export IDP_API_KEY="${{ secrets.IDP_API_KEY }}"
          ./innominatus-ctl chat \
            --generate-spec "production node.js app with postgres and redis" \
            -o deployment.yaml

      - name: Deploy Application
        run: |
          ./innominatus-ctl run deploy-app deployment.yaml
```

---

## Output Formats

### Text Output (Default)

Standard text output with markdown formatting:
```
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
```

### YAML Output (Spec Generation)

When generating specs with `-o` flag:
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: app-name
...
```

### Error Output

Errors are printed to stderr:
```
Error: AI service is not enabled
Error: Invalid API key
Error: Failed to connect to server
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (API error, network failure, etc.) |
| 2 | Invalid arguments or flags |
| 3 | Authentication failure |

**Examples**:
```bash
./innominatus-ctl chat --one-shot "list apps"
echo $?  # 0 (success)

./innominatus-ctl chat --api-key "invalid" --one-shot "list apps"
echo $?  # 3 (auth failure)
```

---

## Common Use Cases

### 1. Quick Platform Status Check

```bash
./innominatus-ctl chat --one-shot "platform stats"
```

### 2. List All Applications

```bash
./innominatus-ctl chat --one-shot "list my applications"
```

### 3. Check Specific Application

```bash
./innominatus-ctl chat --one-shot "tell me about demo-app"
```

### 4. Monitor Workflows

```bash
./innominatus-ctl chat --one-shot "show recent workflows"
./innominatus-ctl chat --one-shot "show running workflows"
./innominatus-ctl chat --one-shot "show failed workflows"
```

### 5. Generate and Deploy

```bash
# Generate spec
./innominatus-ctl chat --generate-spec "python fastapi with postgres" -o api.yaml

# Deploy
./innominatus-ctl run deploy-app api.yaml

# Monitor
./innominatus-ctl chat --one-shot "show workflows for api"
```

### 6. Ask Documentation Questions

```bash
./innominatus-ctl chat --one-shot "what are golden paths?"
./innominatus-ctl chat --one-shot "how do I deploy an application?"
./innominatus-ctl chat --one-shot "explain the demo environment"
```

---

## Troubleshooting

### "Failed to connect to server"

**Cause**: Server not running or wrong URL

**Fix**:
```bash
# Check server is running
curl http://localhost:8081/health

# Or start server
./innominatus
```

### "Authentication failed"

**Cause**: Missing or invalid API key

**Fix**:
```bash
# Check API key is set
echo $IDP_API_KEY

# Generate new key via Web UI
# Then set it:
export IDP_API_KEY="new-key"
```

### "AI service is not enabled"

**Cause**: Server missing AI API keys

**Fix**: Configure server with API keys:
```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
./innominatus
```

### No response / Timeout

**Cause**: LLM API slow or network issue

**Fix**: Wait longer or check network connectivity

### Markdown not rendering

**Note**: CLI outputs plain text with markdown syntax. For rendered markdown, use Web UI.

---

## Comparison: CLI vs Web UI

| Feature | CLI | Web UI |
|---------|-----|--------|
| Interactive chat | ✅ | ✅ |
| One-shot queries | ✅ | ❌ |
| Spec generation | ✅ | ✅ |
| Markdown rendering | ❌ (plain text) | ✅ (rendered) |
| Tool calling | ✅ | ✅ |
| Conversation history | ❌ | ✅ |
| Save specs to file | ✅ | ⚠️ (copy/paste) |
| Scriptable | ✅ | ❌ |
| CI/CD integration | ✅ | ❌ |

**When to Use CLI**:
- Scripting and automation
- CI/CD pipelines
- Quick terminal queries
- Spec generation and saving

**When to Use Web UI**:
- Exploratory conversations
- Better markdown formatting
- Conversation history
- Visual feedback

---

## Development Mode

For development, run CLI directly with Go:

```bash
# Interactive chat
go run cmd/cli/main.go chat

# One-shot query
go run cmd/cli/main.go chat --one-shot "list apps"

# Generate spec
go run cmd/cli/main.go chat --generate-spec "node app" -o test.yaml
```

---

## See Also

- [Getting Started Tutorial](../tutorials/getting-started.md) - Learn AI assistant basics
- [API Reference](./api-reference.md) - HTTP API for integrations
- [Tools Reference](./tools-reference.md) - Available AI tools
- [Configuration Reference](./configuration.md) - Environment variables
