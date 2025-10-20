# AI Assistant Configuration Reference

**Purpose**: Complete reference for configuring the AI assistant service.

## Environment Variables

### Required Variables

#### `OPENAI_API_KEY`
- **Purpose**: API key for OpenAI embedding model
- **Used For**: RAG (Retrieval-Augmented Generation) document embeddings
- **Model**: `text-embedding-3-small`
- **How to Set**:
  ```bash
  export OPENAI_API_KEY="sk-..."
  ```
- **Validation**: Server will log error if missing when AI features are used

#### `ANTHROPIC_API_KEY`
- **Purpose**: API key for Anthropic Claude LLM
- **Used For**: Chat completions and tool calling
- **Model**: `claude-3-5-sonnet-20241022`
- **How to Set**:
  ```bash
  export ANTHROPIC_API_KEY="sk-ant-..."
  ```
- **Validation**: Server will log error if missing when AI features are used

### Optional Variables

#### `AI_ENABLED`
- **Purpose**: Explicitly enable/disable AI features
- **Default**: `true` (enabled if API keys are present)
- **Values**: `true`, `false`
- **How to Set**:
  ```bash
  export AI_ENABLED=false
  ```

#### `RAG_TOP_K`
- **Purpose**: Number of document chunks to retrieve from RAG
- **Default**: `3`
- **Range**: 1-10
- **How to Set**:
  ```bash
  export RAG_TOP_K=5
  ```

#### `RAG_MIN_SCORE`
- **Purpose**: Minimum similarity score for RAG results (0.0-1.0)
- **Default**: `0.3`
- **Range**: 0.0-1.0
- **How to Set**:
  ```bash
  export RAG_MIN_SCORE=0.5
  ```

#### `AI_MAX_TOKENS`
- **Purpose**: Maximum tokens in AI responses
- **Default**: `800` (concise responses)
- **Range**: 100-4000
- **How to Set**:
  ```bash
  export AI_MAX_TOKENS=1500
  ```
- **Note**: Higher values = longer, more detailed responses

#### `AI_TEMPERATURE`
- **Purpose**: LLM temperature for creativity vs consistency
- **Default**: `0.7`
- **Range**: 0.0-1.0
- **How to Set**:
  ```bash
  export AI_TEMPERATURE=0.3
  ```
- **Values**:
  - `0.0-0.3`: More deterministic, factual
  - `0.4-0.7`: Balanced (recommended)
  - `0.8-1.0`: More creative, varied

#### `AI_MAX_ITERATIONS`
- **Purpose**: Maximum agent loop iterations for tool calling
- **Default**: `5`
- **Range**: 1-10
- **How to Set**:
  ```bash
  export AI_MAX_ITERATIONS=3
  ```
- **Note**: Prevents infinite loops if AI keeps requesting tools

## System Prompt Customization

The system prompt defines the AI's behavior, style, and capabilities. It's located in:
```
/Users/philipsahli/projects/innominatus/internal/ai/chat.go
```

### Function: `buildSystemPromptWithTools()`

**Current Prompt** (lines 238-274):
```go
func buildSystemPromptWithTools() string {
    return `You are an expert AI assistant for innominatus...

Guidelines:
- **IMPORTANT: Keep responses very brief and concise (2-3 sentences maximum)**
- Use bullet points instead of long paragraphs
- When using tools, just present the results - don't over-explain
- Only provide detailed explanations when explicitly asked
- Be direct and to the point
...`
}
```

### Customization Guidelines

#### 1. Response Length
To make responses longer/shorter, modify:
```go
- **IMPORTANT: Keep responses very brief and concise (2-3 sentences maximum)**
```
Change to:
```go
- **IMPORTANT: Provide detailed explanations (5-7 sentences)**
```

Also adjust `MaxTokens` in `chat.go` line 74:
```go
MaxTokens: 800,  // Change to 2000 for longer responses
```

#### 2. Response Style
To change tone or formality:
```go
// Current: Direct and concise
- Be direct and to the point

// Alternative: Friendly and explanatory
- Be friendly and helpful, explaining concepts clearly
```

#### 3. Tool Usage Behavior
To change when/how tools are used:
```go
// Current
- Use tools when the user wants to perform actions or view current state

// Alternative: More proactive
- Proactively use tools to gather context before answering
```

#### 4. Documentation Usage
To adjust RAG reliance:
```go
// Current
- Use documentation context only when necessary

// Alternative: Documentation-first
- Always check documentation context before answering
```

### Applying Changes

After modifying the system prompt:

1. **Rebuild server**:
   ```bash
   go build -o innominatus cmd/server/main.go
   ```

2. **Restart server**:
   ```bash
   ./innominatus
   ```

3. **Test changes**:
   ```bash
   ./innominatus-ctl chat --one-shot "list my applications"
   ```

## Knowledge Base Configuration

The AI assistant loads documentation from the `docs/` directory into an in-memory vector database.

### Indexed Documents

**Location**: `/Users/philipsahli/projects/innominatus/docs/`

**Included Files**:
- `README.md`
- `FEATURES.md`
- `GOLDEN_PATHS_METADATA.md`
- `HEALTH_MONITORING.md`
- `OIDC_AUTHENTICATION.md`
- `docs/ai-assistant/**/*.md`

**Excluded**:
- `.git/`
- `node_modules/`
- Binary files
- Hidden files (`.*)

### Document Processing

**Location**: `internal/ai/service.go` - `NewService()` function

**Process**:
1. Walk `docs/` directory recursively
2. Read markdown files (`.md`)
3. Split into chunks (1000 characters with 200 character overlap)
4. Generate embeddings using OpenAI `text-embedding-3-small`
5. Store in in-memory vector database

**Chunk Size**: 1000 characters (configurable in code)
**Chunk Overlap**: 200 characters (configurable in code)

### Adding Custom Documentation

To add custom documentation to the knowledge base:

1. **Create markdown file** in `docs/`:
   ```bash
   echo "# Custom Guide\n\nContent here..." > docs/custom-guide.md
   ```

2. **Restart server** to re-index:
   ```bash
   ./innominatus
   ```

3. **Verify indexing** in logs:
   ```
   Successfully loaded 25 documents into knowledge base
   ```

4. **Test retrieval**:
   ```bash
   ./innominatus-ctl chat --one-shot "tell me about the custom guide"
   ```

### Document Metadata

Each document includes metadata for better retrieval:
- `source`: File path (e.g., `docs/FEATURES.md`)
- `type`: Document type (`markdown`)
- `title`: Extracted from first H1 header

## API Configuration

### Base URL

**Internal**: `http://localhost:8081`
- Used by tool executors for internal API calls
- Configurable in `internal/ai/chat.go` line 128:
  ```go
  apiBaseURL := "http://localhost:8081"
  ```

**External**: Set by deployment environment
- For Kubernetes: Use service DNS
- For Docker: Use container networking

### Authentication

**Token Extraction**: `internal/ai/handlers.go` line 18-25
```go
authHeader := r.Header.Get("Authorization")
var authToken string
if authHeader != "" {
    if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
        authToken = authHeader[7:]
    }
}
```

**Token Usage**: Passed to all tool executors for API calls

## Performance Tuning

### Response Time

**Factors**:
- LLM API latency (~1-3 seconds)
- RAG retrieval time (~100-300ms)
- Tool execution time (varies by tool)
- Network latency

**Optimization**:
- Reduce `RAG_TOP_K` to retrieve fewer documents (faster)
- Reduce `AI_MAX_TOKENS` for shorter responses (faster)
- Lower `AI_MAX_ITERATIONS` to limit tool calling loops

### Memory Usage

**Knowledge Base**: ~5-10MB for 100 documents
**Per Request**: ~1-2MB (conversation history)

**Optimization**:
- Limit document size (split large docs)
- Reduce chunk overlap (default 200 chars)

### Concurrent Requests

**Limits**:
- No built-in concurrency limits
- Limited by LLM API rate limits (Anthropic: 50 req/min)

**Recommendation**: Implement rate limiting for production

## Logging

### AI Service Log Levels

The AI service uses structured logging with standardized message formats for comprehensive observability.

**Components**:
- `internal/ai/service.go` - Service initialization, status
- `internal/ai/chat.go` - Chat interactions, agent loop, tool calling
- `internal/ai/knowledge.go` - Document loading, knowledge base
- `internal/ai/executor.go` - Internal API tool execution

**Debug Level** (`LOG_LEVEL=debug`):
- RAG retrieval parameters (query, top_k, min_score) and results
- Agent loop iterations with token usage tracking
- Individual tool executions with parameters and results
- Document loading details (per-file processing, filtering)
- HTTP request/response details for internal API calls

**Info Level** (`LOG_LEVEL=info`) - Default:
- Service initialization status
- Knowledge base loaded with document count
- Major operational milestones

**Warn Level**:
- Failed RAG retrievals (non-fatal, continues without context)
- Missing authentication tokens for tool execution
- Document loading warnings

**Error Level**:
- AI service initialization failures
- Tool execution failures
- Critical API errors
- Spec generation failures

### Log Message Format

All AI logs follow a consistent pattern:

- **Errors**: `"Failed to <action>"` (e.g., "Failed to retrieve RAG context")
- **Success**: Past tense (e.g., "Loaded knowledge base", "Executed tool")
- **In-progress**: Present participle (e.g., "Loading documentation files")
- **Structure**: Fields always before `.Msg()`, errors use `.Err(err)` first

### Example AI Logs

**Service Initialization**:
```json
{
  "level": "info",
  "llm_provider": "anthropic",
  "llm_model": "claude-sonnet-4-5",
  "embedding_provider": "openai",
  "embedding_model": "text-embedding-3-small",
  "message": "Initialized AI service",
  "timestamp": "2025-10-08T10:30:00Z"
}
```

**Knowledge Base Loading**:
```json
{
  "level": "info",
  "document_count": 25,
  "message": "Loaded knowledge base",
  "timestamp": "2025-10-08T10:30:01Z"
}
```

**RAG Retrieval**:
```json
{
  "level": "debug",
  "query": "how to deploy an app",
  "top_k": 3,
  "min_score": 0.3,
  "results_count": 3,
  "context_length": 1500,
  "message": "Retrieved RAG context",
  "timestamp": "2025-10-08T10:30:15Z"
}
```

**Agent Loop**:
```json
{
  "level": "debug",
  "iteration": 2,
  "prompt_tokens": 450,
  "completion_tokens": 125,
  "total_tokens": 575,
  "cumulative_tokens": 1250,
  "tool_uses": 1,
  "message": "Received LLM response",
  "timestamp": "2025-10-08T10:30:16Z"
}
```

**Tool Execution**:
```json
{
  "level": "debug",
  "iteration": 2,
  "tool_index": 1,
  "total_tools": 1,
  "tool_name": "list_applications",
  "tool_id": "toolu_abc123",
  "message": "Executing tool",
  "timestamp": "2025-10-08T10:30:17Z"
}
```

**Agent Loop Completion**:
```json
{
  "level": "debug",
  "iterations": 3,
  "total_tokens": 1250,
  "has_spec": false,
  "citations_count": 2,
  "message": "Agent loop completed",
  "timestamp": "2025-10-08T10:30:18Z"
}
```

**Failed RAG Retrieval** (Warn):
```json
{
  "level": "warn",
  "error": "connection timeout",
  "query": "example query",
  "message": "Failed to retrieve RAG context",
  "timestamp": "2025-10-08T10:30:15Z"
}
```

**Tool Execution Error**:
```json
{
  "level": "error",
  "error": "API request failed with status 401",
  "tool_name": "deploy_application",
  "tool_id": "toolu_xyz789",
  "message": "Failed to execute tool",
  "timestamp": "2025-10-08T10:30:20Z"
}
```

### Log Queries for Troubleshooting

**Loki Query Examples**:

Find all failed operations:
```
{job="innominatus"} | json | level="error" | message=~"Failed to.*"
```

Track token usage over time:
```
{job="innominatus"} | json | total_tokens > 0
  | line_format "{{.total_tokens}} tokens"
```

Debug tool execution flow:
```
{job="innominatus"} | json | tool_name!=""
  | line_format "{{.tool_name}}: {{.message}}"
```

Monitor RAG retrieval performance:
```
{job="innominatus"} | json | message=~".*RAG context"
  | line_format "{{.results_count}} results, {{.context_length}} chars"
```

Find agent loops with high token usage:
```
{job="innominatus"} | json | message="Agent loop completed" | total_tokens > 2000
```

Track knowledge base loading:
```
{job="innominatus"} | json | message="Loaded knowledge base"
  | line_format "{{.document_count}} documents loaded"
```

### Token Usage Monitoring

**Why Monitor Tokens**: LLM API costs are based on token consumption

**Key Metrics**:
- `prompt_tokens`: Input tokens (context + user message)
- `completion_tokens`: Output tokens (AI response)
- `total_tokens`: Sum of prompt and completion tokens
- `cumulative_tokens`: Total across all iterations in agent loop

**Cost Tracking Queries**:
```
# Sum tokens per hour
sum_over_time({job="innominatus"} | json | total_tokens[1h])

# Average tokens per request
avg_over_time({job="innominatus"} | json | message="Agent loop completed" | total_tokens[5m])

# Identify expensive requests
{job="innominatus"} | json | total_tokens > 3000
```

### Debugging Common Issues

**Issue: "No auth token available for tool execution"**
- **Log Level**: Warn
- **Cause**: User not authenticated or API key missing
- **Impact**: Tools will fail with 401/403
- **Fix**: Ensure user logged in (web UI) or API key set (CLI)

**Issue: "Failed to retrieve RAG context"**
- **Log Level**: Warn
- **Cause**: OpenAI API error or network issue
- **Impact**: AI continues without documentation context
- **Fix**: Check OpenAI API key and connectivity

**Issue: "Reached maximum agent loop iterations"**
- **Log Level**: Warn
- **Cause**: Too many tool calls without completion
- **Impact**: Response may be incomplete
- **Fix**: Reduce `AI_MAX_ITERATIONS` or simplify request

## Health Checks

### AI Service Status

**Endpoint**: `GET /api/ai/status`

**Response**:
```json
{
  "enabled": true,
  "llm_provider": "anthropic",
  "embedding_model": "openai",
  "documents_loaded": 25,
  "status": "ready"
}
```

**Status Values**:
- `ready`: AI service is operational
- `not_configured`: Missing API keys
- `error`: Service initialization failed

### Testing Configuration

```bash
# Check AI status
curl http://localhost:8081/api/ai/status

# Test chat endpoint
curl -X POST http://localhost:8081/api/ai/chat \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "list my applications"}'
```

## Troubleshooting

### "AI service is not enabled"

**Cause**: Missing API keys or `AI_ENABLED=false`

**Fix**:
```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
export AI_ENABLED=true
./innominatus
```

### "Failed to retrieve RAG context"

**Cause**: OpenAI API error or network issue

**Impact**: AI will continue without documentation context

**Fix**: Check OpenAI API key and network connectivity

### Responses are too long/short

**Fix**: Adjust `AI_MAX_TOKENS`:
```bash
export AI_MAX_TOKENS=800   # Shorter responses
export AI_MAX_TOKENS=2000  # Longer responses
```

And modify system prompt (see "Response Length" section above).

### Tool calling not working

**Symptoms**: AI describes actions but doesn't execute them

**Cause**: Authentication failure (no auth token)

**Fix**: Ensure user is logged in (web UI) or API key is set (CLI)

## See Also

- [Tools Reference](./tools-reference.md) - Available AI tools
- [API Reference](./api-reference.md) - HTTP endpoints
- [Tool Calling Architecture](../explanations/tool-calling-architecture.md) - Implementation details
