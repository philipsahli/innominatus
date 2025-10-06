# AI Assistant Documentation

**The AI-powered conversational interface for innominatus platform operations**

---

## Overview

The innominatus AI Assistant is a conversational interface that combines:
- **Live platform data** via tool calling (applications, workflows, resources)
- **Documentation knowledge** via RAG (Retrieval-Augmented Generation)
- **Natural language understanding** powered by Anthropic Claude and OpenAI GPT

Instead of remembering CLI commands or navigating complex APIs, ask questions in plain English:
- "list my applications"
- "show me failed workflows for demo-app"
- "generate a score spec for a python app with postgres"
- "what are golden paths?"

---

## Quick Start

### 🧑‍💻 For Users (Developers)

**You want to use the AI assistant to interact with the platform.**

👉 **[User Guide: AI Assistant](../user-guide/ai-assistant.md)** - Complete guide for platform users

**Quick access:**
- **Web UI:** Login → Click "AI Assistant" (🤖 icon)
- **CLI:** `./innominatus-ctl chat`

### ⚙️ For Platform Teams

**You want to configure and operate the AI assistant for your organization.**

👉 **[Platform Team Guide: AI Assistant](../platform-team-guide/ai-assistant.md)** - Configuration, monitoring, and operations

**Quick setup:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
./innominatus
curl http://localhost:8081/api/ai/health
```

---

## What Can It Do?

### Platform Operations (via Tool Calling)

**List Resources:**
- "list my applications"
- "show all workflows"
- "what resources are deployed"
- "get platform statistics"

**Get Details:**
- "tell me about demo-app"
- "show workflow execution 123"
- "what's the status of my-app"

**Deploy & Manage:**
- "deploy this spec" (after generating)
- "delete test-app"

**Monitoring:**
- "show failed workflows"
- "what workflows are running"
- "show me workflows for demo-app"

### Knowledge Base (via RAG)

**Explain Concepts:**
- "what are golden paths?"
- "explain the demo environment"
- "how does OIDC authentication work"

**Generate Specifications:**
- "create a score spec for a python app with postgres"
- "generate a spec for node.js with redis"

**Guidance:**
- "how do I deploy an app?"
- "what's the difference between deploy-app and POST /api/specs?"

---

## Documentation Structure

This documentation follows the [Diátaxis framework](https://diataxis.fr/) with four types of content:

### 📘 Tutorials (Learning-Oriented)

**Goal**: Teach you how to use the AI assistant through hands-on examples

- **[Getting Started](tutorials/getting-started.md)** - Your first 10 minutes with the AI assistant

### 📗 How-To Guides (Task-Oriented)

**Goal**: Step-by-step instructions for specific tasks

- **[Query Workflow Status](how-to-guides/query-workflows.md)** - Check workflow execution status
- **[View Platform Statistics](how-to-guides/view-statistics.md)** - Get platform-wide metrics
- **[Get Application Details](how-to-guides/get-application-details.md)** - Deep dive into specific apps
- **[Deploy with AI](how-to-guides/deploy-with-ai.md)** - Use AI to generate and deploy specs
- **[Generate Score Specs](how-to-guides/generate-specs.md)** - AI-powered spec generation
- **[List Applications](how-to-guides/list-applications.md)** - Query deployed applications

### 📙 Explanations (Understanding-Oriented)

**Goal**: Understand how the AI assistant works under the hood

- **[Tool Calling Architecture](explanations/tool-calling-architecture.md)** - How AI uses tools to interact with platform

### 📕 Reference (Information-Oriented)

**Goal**: Technical specifications and API documentation

- **[Tools Reference](reference/tools-reference.md)** - Complete documentation of all 8 AI tools
- **[API Reference](reference/api-reference.md)** - HTTP API endpoints for AI chat
- **[CLI Reference](reference/cli-reference.md)** - Command-line interface documentation
- **[Configuration](reference/configuration.md)** - AI assistant configuration options

---

## Key Features

### 🤖 Dual-Model Support

The AI assistant supports both **Anthropic Claude** and **OpenAI GPT**:

**Anthropic Claude** (Recommended):
- Superior reasoning for complex multi-step operations
- Better tool calling capabilities
- Default provider for production deployments

**OpenAI GPT**:
- Alternative provider for specific use cases
- Configurable via environment variables

**Configuration:**
```bash
# Use Anthropic (default)
export ANTHROPIC_API_KEY="sk-ant-..."
export AI_PROVIDER="anthropic"

# Or use OpenAI
export OPENAI_API_KEY="sk-..."
export AI_PROVIDER="openai"
```

### 🛠️ Tool Calling

The AI can execute 8 platform operations:

| Tool | Purpose |
|------|---------|
| `list_applications` | List all deployed applications |
| `get_application` | Get details for specific application |
| `deploy_application` | Deploy a Score specification |
| `delete_application` | Remove an application and its resources |
| `list_workflows` | View workflow executions (filter by status, app) |
| `get_workflow` | Get detailed workflow execution info |
| `list_resources` | View platform resources (databases, caches, volumes) |
| `get_dashboard_stats` | Platform-wide statistics |

**See:** [Tools Reference](reference/tools-reference.md) for complete documentation.

### 📚 RAG (Retrieval-Augmented Generation)

The AI has access to the complete innominatus documentation:
- User guides
- Platform team guides
- API documentation
- Workflow guides
- Configuration references

When you ask conceptual questions, the AI retrieves relevant documentation and provides accurate, contextual answers.

### 🔄 Multi-Turn Conversations

The AI maintains conversation context for natural follow-ups:

```
You: "list my applications"
AI: "You have 3 applications: demo-app, test-service, api-gateway"

You: "tell me about the first one"
AI: "demo-app (production): 5 resources, created 2 days ago..."

You: "show me its workflows"
AI: "demo-app workflows: deploy-app (completed), db-lifecycle (running)..."
```

### 🔐 Authentication

All AI operations use your authentication token:
- **Web UI**: Session-based authentication
- **CLI**: API key via `$IDP_API_KEY` environment variable

The AI can only perform actions you're authorized for.

---

## Architecture Overview

```
┌─────────────┐
│    User     │
└──────┬──────┘
       │
       │ Natural Language
       │ "list my applications"
       ↓
┌──────────────────────┐
│   AI Assistant       │
│  (Claude/GPT)        │
│                      │
│  ┌────────────────┐  │
│  │ System Prompt  │  │
│  │ + Tools        │  │
│  │ + RAG Context  │  │
│  └────────────────┘  │
└──────┬───────────────┘
       │
       │ Tool Calls
       │ list_applications()
       ↓
┌──────────────────────┐
│  Tool Executor       │
│                      │
│  HTTP Client         │
└──────┬───────────────┘
       │
       │ HTTP Request
       │ GET /api/specs
       ↓
┌──────────────────────┐
│  innominatus API     │
│                      │
│  ┌────────────────┐  │
│  │ Auth/Authz     │  │
│  │ Business Logic │  │
│  │ Database       │  │
│  └────────────────┘  │
└──────┬───────────────┘
       │
       │ JSON Response
       ↓
     [Data]
       │
       ↓
┌──────────────────────┐
│   AI Assistant       │
│  (Formats Response)  │
└──────┬───────────────┘
       │
       │ Structured Text
       │ "You have 3 applications..."
       ↓
┌─────────────┐
│    User     │
└─────────────┘
```

**Key Components:**

1. **AI Assistant** (`internal/ai/chat.go`) - Orchestrates multi-turn conversations
2. **Tool Definitions** (`internal/ai/tools.go`) - 8 available platform operations
3. **Tool Executor** (`internal/ai/executor.go`) - Executes HTTP API calls
4. **RAG System** (`internal/ai/rag.go`) - Documentation retrieval
5. **API Handlers** (`internal/server/handlers.go`) - HTTP endpoints

**See:** [Tool Calling Architecture](explanations/tool-calling-architecture.md) for detailed explanation.

---

## Use Cases

### Developer Self-Service

**Scenario**: Deploy a new application

```
Developer: "I need to deploy a python app with postgres"

AI: "I'll generate a Score specification for you:
[generates spec]

Would you like me to deploy it?"

Developer: "yes"

AI: "Application deployed. Workflow #45 is running."
```

### Platform Monitoring

**Scenario**: Check platform health

```
Operator: "show me failed workflows from today"

AI: "2 failed workflows:
• deploy-app (demo-app) - failed at provision step
• db-lifecycle (api-gateway) - database connection timeout

Would you like details for either workflow?"
```

### Documentation Discovery

**Scenario**: Learn about features

```
User: "what are golden paths?"

AI: "Golden paths are pre-defined workflows for common use cases:
• deploy-app: Full application deployment
• ephemeral-env: Temporary test environments
• db-lifecycle: Database management
• observability-setup: Monitoring setup

See /goldenpaths to run a golden path workflow."
```

---

## Configuration

### Environment Variables

```bash
# Required: LLM Provider API Keys
export ANTHROPIC_API_KEY="sk-ant-..."  # For Anthropic Claude
export OPENAI_API_KEY="sk-..."         # For OpenAI GPT

# Optional: Provider Selection
export AI_PROVIDER="anthropic"  # or "openai" (default: anthropic)

# Optional: Model Configuration
export AI_MODEL="claude-3-5-sonnet-20241022"  # or "gpt-4o"
export AI_MAX_TOKENS="2048"                   # Response length limit
export AI_TEMPERATURE="0.7"                   # Creativity (0.0-1.0)
```

### System Prompt Customization

**Location**: `internal/ai/chat.go` - `systemPrompt` constant

**Purpose**: Control AI behavior, tone, and guidelines

**Example Customizations:**
- Add domain-specific terminology
- Enforce company policies
- Customize response style
- Add safety guidelines

**See:** [Configuration Reference](reference/configuration.md) for details.

---

## Security Considerations

### Authentication & Authorization

**All tool calls use user's auth token:**
- User can only perform actions they're authorized for
- No privilege escalation
- Full audit trail in API logs

### Tool Permissions

**Read-Only Tools** (Safe):
- `list_applications`, `get_application`
- `list_workflows`, `get_workflow`
- `list_resources`, `get_dashboard_stats`

**Write Tools** (Careful):
- `deploy_application` - Creates resources
- `delete_application` - Destructive action

**Best Practice**: Add confirmation prompts for destructive actions (future enhancement).

### Data Privacy

**AI Provider Communication:**
- User messages sent to Anthropic/OpenAI APIs
- Platform data included in tool results
- No persistent storage by AI providers (per their policies)

**Recommendation**: Review data privacy policies for:
- [Anthropic Privacy Policy](https://www.anthropic.com/privacy)
- [OpenAI Privacy Policy](https://openai.com/privacy)

---

## Troubleshooting

### "AI service is not enabled"

**Cause**: Missing API keys

**Fix**:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENAI_API_KEY="sk-..."
./innominatus  # Restart server
```

### "Failed to fetch" (Web UI)

**Cause**: Not logged in or session expired

**Fix**:
1. Refresh page
2. Login again
3. Retry AI request

### "Error executing tool: 401 Unauthorized"

**Cause**: Invalid or expired auth token

**Fix (CLI)**:
```bash
export IDP_API_KEY="your-valid-key"
./innominatus-ctl chat
```

**Fix (Web UI)**: Logout and login again

### Slow Responses

**Cause**: Multiple LLM API calls + tool execution

**Typical latency**: 2-4 seconds per request

**Optimization**:
- Reduce `AI_MAX_TOKENS` for faster responses
- Use more specific queries (fewer tool calls needed)

### AI Repeating Actions

**Cause**: Agent loop executing same tool multiple times

**Protection**: Max 5 iterations enforced

**Fix**: Rephrase query to be more specific

---

## Performance

### Response Times

**Typical breakdown:**
- LLM API call: 1-2 seconds
- Tool execution: 100-500ms
- Total: 2-4 seconds

**With multiple tools:**
- Can increase to 5-8 seconds

### Token Usage

**Per request:**
- Input: 500-1000 tokens (system prompt + conversation)
- Output: 100-800 tokens (response)

**Cost estimates:**
- Anthropic Claude: ~$0.005-0.02 USD per request
- OpenAI GPT-4: ~$0.01-0.03 USD per request

### Rate Limiting

**innominatus API**: Rate limiting applies to tool execution

**LLM Provider**: Follow provider rate limits
- Anthropic: Tier-based limits
- OpenAI: Tier-based limits

---

## Roadmap

### Near-Term

- [ ] Streaming responses (SSE)
- [ ] Confirmation prompts for destructive actions
- [ ] Tool result caching (30s TTL)
- [ ] Conversation export (markdown format)

### Future

- [ ] Parallel tool execution
- [ ] Custom tool definitions (user-defined)
- [ ] Multi-user conversation context
- [ ] Voice input/output support
- [ ] Mobile app integration

---

## Contributing

### Adding New Tools

**Steps:**
1. Define tool in `internal/ai/tools.go`
2. Implement executor in `internal/ai/executor.go`
3. Add tests in `internal/ai/executor_test.go`
4. Update [Tools Reference](reference/tools-reference.md)
5. Submit PR

**Example**: See [Tool Calling Architecture](explanations/tool-calling-architecture.md#extensibility)

### Improving Documentation

**Structure**: Follow Diátaxis framework
- Tutorials: Learning-oriented
- How-to guides: Task-oriented
- Explanations: Understanding-oriented
- Reference: Information-oriented

**Location**: `docs/ai-assistant/`

---

## Support

### For Users

**Questions about using the AI assistant?**
- Check [Getting Started Tutorial](tutorials/getting-started.md)
- Review [How-To Guides](how-to-guides/)
- Ask in [GitHub Discussions](https://github.com/philipsahli/innominatus/discussions)

### For Platform Teams

**Questions about deployment/configuration?**
- Review [Configuration Reference](reference/configuration.md)
- Check [Platform Team Guide](../platform-team-guide/)
- Open [GitHub Issue](https://github.com/philipsahli/innominatus/issues)

### For Contributors

**Want to improve the AI assistant?**
- Review [Tool Calling Architecture](explanations/tool-calling-architecture.md)
- Check [Contributing Guide](../../CONTRIBUTING.md)
- Join [GitHub Discussions](https://github.com/philipsahli/innominatus/discussions)

---

## See Also

- **[Main README](../../README.md)** - innominatus project overview
- **[User Guide](../user-guide/)** - For application developers
- **[Platform Team Guide](../platform-team-guide/)** - For platform engineers
- **[Development Guide](../development/)** - For contributors

---

**Built with ❤️ using Anthropic Claude and OpenAI GPT**
