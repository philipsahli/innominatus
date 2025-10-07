# AI Assistant Configuration & Operations

**Enable conversational AI for your platform users**

---

## Overview

The innominatus AI Assistant provides a natural language interface for platform operations. Instead of training users on CLI commands and API endpoints, they can ask questions in plain English.

**For Platform Teams, this means:**
- Reduced support burden (self-service questions)
- Faster user onboarding
- Better platform adoption
- Comprehensive audit trails

**User Benefits:**
- Natural language queries ("list my applications")
- Live platform data access
- Documentation retrieval
- Spec generation assistance

---

## Quick Setup

### Prerequisites

**Required:**
- innominatus server v0.2.0+
- API key from Anthropic or OpenAI
- PostgreSQL database (for audit logs)

**Optional:**
- OIDC authentication (for user tracking)
- Prometheus metrics (for monitoring)

### Basic Configuration

```bash
# 1. Set API key (choose one provider)
export ANTHROPIC_API_KEY="sk-ant-..."  # Recommended
# OR
export OPENAI_API_KEY="sk-..."

# 2. Optional: Select provider explicitly
export AI_PROVIDER="anthropic"  # or "openai" (default: anthropic)

# 3. Start server
./innominatus

# 4. Verify AI is enabled
curl http://localhost:8081/api/ai/health
# Response: {"status": "ok", "provider": "anthropic"}
```

### Kubernetes Deployment

**Helm values:**
```yaml
# values.yaml
ai:
  enabled: true
  provider: "anthropic"  # or "openai"

  # API keys (use secrets in production)
  anthropicApiKey: "sk-ant-..."
  # OR
  openaiApiKey: "sk-..."

  # Optional: Model configuration
  model: "claude-3-5-sonnet-20241022"
  maxTokens: 2048
  temperature: 0.7
```

**Install:**
```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --values values.yaml
```

---

## Provider Selection

### Anthropic Claude (Recommended)

**Best for:**
- Complex multi-step operations
- Superior reasoning capabilities
- Production deployments

**Models:**
- `claude-3-5-sonnet-20241022` (default, recommended)
- `claude-3-opus-20240229` (highest capability)
- `claude-3-haiku-20240307` (fastest, lowest cost)

**Pricing (as of 2025):**
- Sonnet: $3/MTok input, $15/MTok output
- ~$0.005-0.02 per request

**Configuration:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
export AI_PROVIDER="anthropic"
export AI_MODEL="claude-3-5-sonnet-20241022"
```

**Get API Key:** https://console.anthropic.com/

### OpenAI GPT

**Best for:**
- Alternative provider
- Specific use cases requiring GPT
- Organizations with existing OpenAI contracts

**Models:**
- `gpt-4o` (default, recommended)
- `gpt-4-turbo` (high capability)
- `gpt-3.5-turbo` (faster, lower cost)

**Pricing (as of 2025):**
- GPT-4o: $5/MTok input, $15/MTok output
- ~$0.01-0.03 per request

**Configuration:**
```bash
export OPENAI_API_KEY="sk-..."
export AI_PROVIDER="openai"
export AI_MODEL="gpt-4o"
```

**Get API Key:** https://platform.openai.com/api-keys

---

## Configuration Reference

### Environment Variables

```bash
# Provider Selection (Required)
ANTHROPIC_API_KEY="sk-ant-..."     # Anthropic API key
OPENAI_API_KEY="sk-..."            # OpenAI API key
AI_PROVIDER="anthropic"            # Provider: "anthropic" or "openai"

# Model Configuration (Optional)
AI_MODEL="claude-3-5-sonnet-20241022"  # Model name
AI_MAX_TOKENS="2048"                   # Max response length (100-4096)
AI_TEMPERATURE="0.7"                   # Creativity (0.0-1.0)

# API Endpoints (Optional - for custom deployments)
ANTHROPIC_API_URL="https://api.anthropic.com"
OPENAI_API_URL="https://api.openai.com"

# Feature Flags (Optional)
AI_ENABLE_RAG="true"                   # Enable documentation retrieval
AI_ENABLE_TOOLS="true"                 # Enable tool calling
AI_MAX_ITERATIONS="5"                  # Max tool calling iterations
```

### Configuration File (Future)

**Planned for v0.3.0:**
```yaml
# config/ai-config.yaml
ai:
  provider: anthropic
  model: claude-3-5-sonnet-20241022
  maxTokens: 2048
  temperature: 0.7

  tools:
    enabled: true
    maxIterations: 5

  rag:
    enabled: true
    chunkSize: 1000

  rateLimiting:
    requestsPerMinute: 60
    requestsPerHour: 1000
```

---

## Security Configuration

### Authentication

**All AI operations require user authentication:**

**Web UI:**
- Session-based authentication
- User must be logged in
- Auth token passed automatically

**CLI:**
- Requires `$IDP_API_KEY` environment variable
- User must have valid API key

**API:**
```bash
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"message": "list my applications"}'
```

### Authorization

**AI operations use user permissions:**
- Users can only see their authorized applications
- Deploy/delete actions respect RBAC
- No privilege escalation possible

**Implementation:**
All AI tool calls use the user's auth token when calling internal APIs.

### Data Privacy

**What's sent to AI providers:**
- User messages (questions/prompts)
- Platform data returned by tools (app names, workflow status, etc.)
- Documentation chunks (for RAG)

**What's NOT sent:**
- User credentials
- API keys
- Sensitive configuration (secrets, passwords)

**AI Provider Policies:**
- **Anthropic:** No training on user data, 30-day retention for safety
- **OpenAI:** No training on API data (enterprise), 30-day retention

**Best Practices:**
1. Review provider data policies: [Anthropic](https://www.anthropic.com/privacy) | [OpenAI](https://openai.com/privacy)
2. Consider on-premise LLM deployment for highly sensitive environments
3. Enable audit logging for all AI interactions
4. Inform users about AI data handling

### Audit Logging

**All AI interactions are logged:**

**Log Location:** Server logs (JSON format)

**Logged Events:**
```json
{
  "timestamp": "2025-10-06T16:30:00Z",
  "event": "ai_chat",
  "user": "user@example.com",
  "message": "list my applications",
  "tools_used": ["list_applications"],
  "response_tokens": 456,
  "duration_ms": 2340
}
```

**Query logs:**
```bash
# Find AI usage by user
cat /var/log/innominatus/server.log | jq 'select(.event == "ai_chat" and .user == "user@example.com")'

# Count AI requests
cat /var/log/innominatus/server.log | jq 'select(.event == "ai_chat")' | wc -l

# Average response time
cat /var/log/innominatus/server.log | jq -s 'map(select(.event == "ai_chat")) | map(.duration_ms) | add / length'
```

---

## Monitoring & Metrics

### Prometheus Metrics

**Available at:** `http://localhost:8081/metrics`

**AI-Specific Metrics:**
```prometheus
# Request count
innominatus_ai_requests_total{provider="anthropic", status="success"}

# Response time
innominatus_ai_response_duration_seconds{provider="anthropic", quantile="0.95"}

# Token usage
innominatus_ai_tokens_total{provider="anthropic", type="input"}
innominatus_ai_tokens_total{provider="anthropic", type="output"}

# Tool usage
innominatus_ai_tool_calls_total{tool_name="list_applications", status="success"}

# Errors
innominatus_ai_errors_total{provider="anthropic", error_type="rate_limit"}
```

**Example queries:**
```promql
# AI requests per minute
rate(innominatus_ai_requests_total[5m])

# Average response time
avg(innominatus_ai_response_duration_seconds)

# Most used tools
topk(5, sum by (tool_name) (innominatus_ai_tool_calls_total))

# Error rate
rate(innominatus_ai_errors_total[5m]) / rate(innominatus_ai_requests_total[5m])
```

### Grafana Dashboard

**Import dashboard:** `dashboards/ai-assistant-dashboard.json` (future)

**Panels:**
- AI request rate (requests/min)
- Average response time (p50, p95, p99)
- Token usage (input/output)
- Tool usage breakdown
- Error rate by type
- Cost estimation (based on token usage)

### Health Checks

**Endpoint:** `GET /api/ai/health`

**Response (healthy):**
```json
{
  "status": "ok",
  "provider": "anthropic",
  "model": "claude-3-5-sonnet-20241022",
  "tools_enabled": true,
  "rag_enabled": true
}
```

**Response (unhealthy):**
```json
{
  "status": "error",
  "error": "API key not configured",
  "provider": "none"
}
```

**Monitoring:**
```bash
# Check every 60 seconds
watch -n 60 'curl -s http://localhost:8081/api/ai/health | jq'

# Alert if unhealthy
curl -s http://localhost:8081/api/ai/health | jq -e '.status == "ok"' || echo "AI service unhealthy!"
```

---

## Cost Management

### Cost Estimation

**Anthropic Claude (Sonnet):**
- Input: $3 per million tokens
- Output: $15 per million tokens
- Average request: 500 input + 200 output = ~$0.004

**OpenAI GPT-4o:**
- Input: $5 per million tokens
- Output: $15 per million tokens
- Average request: 500 input + 200 output = ~$0.006

**Monthly estimates (1000 users, 10 requests/user/day):**
- Anthropic: ~$1,200/month
- OpenAI: ~$1,800/month

### Cost Optimization

**1. Reduce Max Tokens:**
```bash
export AI_MAX_TOKENS="1024"  # Default: 2048
# Saves ~50% on output tokens
```

**2. Use Cheaper Models:**
```bash
# Anthropic Haiku (10x cheaper than Sonnet)
export AI_MODEL="claude-3-haiku-20240307"

# OpenAI GPT-3.5 Turbo (20x cheaper than GPT-4o)
export AI_MODEL="gpt-3.5-turbo"
```

**3. Enable Caching (Future):**
```bash
export AI_ENABLE_CACHE="true"
export AI_CACHE_TTL="300"  # 5 minutes
```

**4. Rate Limiting:**
```bash
# Limit AI requests per user
export AI_RATE_LIMIT="60"  # requests per hour per user
```

### Monitoring Costs

**Prometheus query:**
```promql
# Estimated hourly cost
(
  sum(rate(innominatus_ai_tokens_total{type="input"}[1h])) * 0.000003 +
  sum(rate(innominatus_ai_tokens_total{type="output"}[1h])) * 0.000015
) * 3600
```

**Alert on high costs:**
```yaml
# Prometheus alert
- alert: HighAICosts
  expr: |
    (
      sum(rate(innominatus_ai_tokens_total{type="input"}[1h])) * 0.000003 +
      sum(rate(innominatus_ai_tokens_total{type="output"}[1h])) * 0.000015
    ) * 3600 * 24 * 30 > 2000
  annotations:
    summary: "AI costs exceeding $2000/month"
```

---

## Troubleshooting

### AI Service Not Starting

**Symptom:** `/api/ai/health` returns error

**Common causes:**

**1. Missing API key:**
```bash
# Check
echo $ANTHROPIC_API_KEY

# Fix
export ANTHROPIC_API_KEY="sk-ant-..."
./innominatus  # Restart
```

**2. Invalid API key:**
```bash
# Test manually
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model": "claude-3-5-sonnet-20241022", "max_tokens": 10, "messages": [{"role": "user", "content": "hi"}]}'

# Should return valid response, not 401
```

**3. Network connectivity:**
```bash
# Check API endpoint reachable
curl -I https://api.anthropic.com
# Should return 200 or 405 (not timeout)
```

### Slow Responses

**Symptom:** AI requests taking >10 seconds

**Diagnosis:**
```bash
# Check API latency
time curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"message": "list my applications"}'
```

**Common causes:**

**1. Multiple tool calls:**
- Each tool call adds 1-2 seconds
- Check logs for tool usage

**2. High max_tokens:**
```bash
# Reduce for faster responses
export AI_MAX_TOKENS="1024"
```

**3. Provider rate limiting:**
- Check server logs for 429 errors
- Implement request queuing

**4. Network latency:**
- Check network path to AI provider
- Consider regional API endpoints

### High Error Rates

**Check Prometheus:**
```promql
rate(innominatus_ai_errors_total[5m])
```

**Common errors:**

**401 Unauthorized:**
- Invalid or expired API key
- Regenerate key from provider console

**429 Rate Limited:**
- Exceeded provider rate limits
- Implement exponential backoff
- Upgrade provider tier

**500 Internal Server Error:**
- Check server logs
- May indicate tool execution failures

**503 Service Unavailable:**
- Provider having issues
- Check status pages: [Anthropic](https://status.anthropic.com) | [OpenAI](https://status.openai.com)

---

## Advanced Topics

### Custom System Prompt

**Location:** `internal/ai/chat.go` - `systemPrompt` constant

**Customize for:**
- Company-specific terminology
- Additional guidelines
- Custom tone/style
- Domain expertise

**Example customization:**
```go
const systemPrompt = `You are an expert AI assistant for Acme Corp's internal developer platform.

Key terminology:
- "golden paths" are called "standard workflows" at Acme
- Our primary environment is "prod" not "production"

Company guidelines:
- Always recommend infrastructure-as-code approaches
- Emphasize security best practices
- Mention cost implications for expensive resources

Response style:
- Professional but friendly
- Include cost estimates when deploying resources
- Link to internal wiki when referencing processes
`
```

### Adding Custom Tools

**Steps:**

**1. Define tool in `internal/ai/tools.go`:**
```go
{
    Name: "check_costs",
    Description: "Get cost breakdown for an application",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "app_name": {"type": "string"},
        },
        "required": []string{"app_name"},
    },
}
```

**2. Implement executor in `internal/ai/executor.go`:**
```go
case "check_costs":
    appName := input["app_name"].(string)
    return e.checkApplicationCosts(ctx, appName)
```

**3. Restart server:**
```bash
./innominatus
```

**See:** [AI Assistant Architecture](../ai-assistant/explanations/tool-calling-architecture.md#extensibility)

### On-Premise LLM Deployment

**For highly sensitive environments:**

**Options:**
1. **Anthropic Enterprise:** On-premise Claude deployment
2. **Azure OpenAI:** Bring-your-own-key in your Azure tenant
3. **Self-hosted:** LLaMA, Mistral, etc. (requires code changes)

**Requirements:**
- Minimum 40GB GPU VRAM for 70B parameter models
- OpenAI-compatible API endpoint
- Update `ANTHROPIC_API_URL` or `OPENAI_API_URL`

**Contact:** Enterprise support for custom deployment assistance

---

## User Onboarding

### Introducing AI Assistant to Users

**1. Internal announcement:**
```markdown
🤖 New: AI Assistant for innominatus

You can now use natural language to interact with our platform!

Try it:
- Web UI: Click the AI Assistant icon
- CLI: `innominatus-ctl chat`

Examples:
- "list my applications"
- "show failed workflows"
- "generate a score spec for python"

Questions? See the [User Guide](docs/user-guide/ai-assistant.md)
```

**2. Training materials:**
- Quick start video (2-3 minutes)
- Written guide: [User Guide](../user-guide/ai-assistant.md)
- Internal wiki page with examples

**3. Office hours:**
- Weekly Q&A sessions
- Slack/Teams channel for questions
- Collect feedback for improvements

### User Feedback Collection

**Add feedback mechanism:**
```bash
# Future feature: feedback API
POST /api/ai/feedback
{
  "message_id": "msg_123",
  "rating": "helpful",  # helpful, not_helpful, incorrect
  "comment": "Great response!"
}
```

**Track metrics:**
- User adoption rate (% of users trying AI)
- Engagement (requests per user per day)
- Satisfaction (thumbs up/down feedback)

---

## Roadmap

### v0.3.0 (Planned)
- [ ] Streaming responses (SSE)
- [ ] Conversation export (markdown)
- [ ] Tool result caching (30s TTL)
- [ ] Configuration file support

### v0.4.0 (Future)
- [ ] Confirmation prompts for destructive actions
- [ ] Parallel tool execution
- [ ] Cost optimization (request batching)
- [ ] Custom tool definitions (user-provided)

### v1.0.0 (Future)
- [ ] Multi-user conversation context
- [ ] Voice input/output
- [ ] Mobile app integration
- [ ] On-premise LLM support (LLaMA, Mistral)

---

## See Also

- **[User Guide - AI Assistant](../user-guide/ai-assistant.md)** - End-user documentation
- **[AI Assistant Architecture](../ai-assistant/explanations/tool-calling-architecture.md)** - Technical deep-dive
- **[AI Tools Reference](../ai-assistant/reference/tools-reference.md)** - All 8 available tools
- **[Operations Guide](operations.md)** - Platform operations and maintenance

**Full Documentation:** [docs/ai-assistant/](../ai-assistant/README.md)

---

*Enable self-service platform operations with conversational AI.* 🤖
