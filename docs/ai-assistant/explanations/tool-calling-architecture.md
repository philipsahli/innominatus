# Tool Calling Architecture

**Purpose**: Understand how the AI assistant uses tools to interact with the platform.

## Overview

The AI assistant uses **Anthropic's tool calling** (also known as function calling) to perform actions on the innominatus platform. Instead of just answering questions from documentation, the AI can execute real platform operations like listing applications, deploying specs, and checking workflow status.

## Why Tool Calling?

### Problem: Static Documentation-Only AI

**Before tool calling**, the AI could only:
- Answer questions from static documentation
- Generate Score specifications
- Provide explanations and guidance

**Limitations**:
- No access to live platform data (current apps, workflows, resources)
- Couldn't perform actions (deploy, delete, monitor)
- Responses were generic, not personalized to user's actual state

### Solution: Dynamic Tool Calling

**With tool calling**, the AI can:
- Query live platform state ("list my applications")
- Perform actions ("deploy this spec")
- Provide personalized, data-driven responses
- Chain multiple operations together

**Example**:
```
User: "what applications do I have"

Without tools: "You can see your applications by running 'innominatus-ctl list'"

With tools: "You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources"
```

## Architecture Components

### 1. Tool Definitions

**Location**: `internal/ai/tools.go`

**Purpose**: Define available tools for the AI with names, descriptions, and input schemas.

**Structure**:
```go
type Tool struct {
    Name        string                 // e.g., "list_applications"
    Description string                 // When to use this tool
    InputSchema map[string]interface{} // JSON Schema for parameters
}
```

**Example Tool**:
```go
{
    Name: "get_application",
    Description: "Get detailed information about a specific application...",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "app_name": map[string]interface{}{
                "type": "string",
                "description": "The name of the application to retrieve",
            },
        },
        "required": []string{"app_name"},
    },
}
```

**Why JSON Schema?**: Anthropic's API requires input validation schemas to ensure the AI provides properly structured parameters.

### 2. Tool Executor

**Location**: `internal/ai/executor.go`

**Purpose**: Execute tool calls by making HTTP requests to the innominatus API.

**Structure**:
```go
type ToolExecutor struct {
    apiBaseURL string  // e.g., "http://localhost:8081"
    authToken  string  // User's Bearer token
}

func (e *ToolExecutor) ExecuteTool(ctx context.Context, toolName string, input map[string]interface{}) (string, error)
```

**Why HTTP API Calls?**:
- Reuses existing API logic (validation, authorization, business logic)
- Maintains separation of concerns (AI service doesn't access database directly)
- Enables consistent authentication and audit trails

### 3. Agent Loop

**Location**: `internal/ai/chat.go` - `Chat()` function

**Purpose**: Orchestrate multi-turn conversations between AI, tools, and user.

**Flow**:
```
1. User sends message
2. AI analyzes message + available tools
3. AI decides which tool(s) to use (if any)
4. Tools execute → return results
5. AI receives results
6. AI formulates response (may request more tools)
7. Repeat until AI has final answer
8. Return response to user
```

**Why Loop?**: Some queries require multiple tool calls. Example:
```
User: "show me failed workflows for demo-app"

AI calls:
1. list_workflows(status="failed") → Get all failed workflows
2. Filters for "demo-app" in results
3. Formats and returns
```

### 4. Conversation History

**Structure**: Array of messages with roles and content blocks

**Message Types**:
- `user`: User's question or tool results
- `assistant`: AI's response or tool requests

**Content Block Types**:
- `text`: Plain text message
- `tool_use`: AI requesting to use a tool
- `tool_result`: Result from tool execution

**Example Conversation**:
```json
[
  {
    "role": "user",
    "content": [{"type": "text", "text": "list my applications"}]
  },
  {
    "role": "assistant",
    "content": [
      {"type": "text", "text": "I'll list your applications."},
      {"type": "tool_use", "id": "tool_1", "name": "list_applications", "input": {}}
    ]
  },
  {
    "role": "user",
    "content": [
      {"type": "tool_result", "tool_use_id": "tool_1", "content": "{\"applications\": [...]}"}
    ]
  },
  {
    "role": "assistant",
    "content": [
      {"type": "text", "text": "You have 3 applications:\n• demo-app..."}
    ]
  }
]
```

## End-to-End Flow

### Example: "list my applications"

**Step 1: User Request**
```http
POST /api/ai/chat
{
  "message": "list my applications"
}
```

**Step 2: Build Initial Conversation**
```go
messages := []llm.Message{
    {
        Role: "user",
        Content: []llm.ContentBlock{
            {Type: "text", Text: "list my applications"},
        },
    },
}
```

**Step 3: Call LLM with Tools**
```go
llmResponse, err := s.sdk.LLM().GenerateWithTools(ctx, llm.GenerateWithToolsRequest{
    SystemPrompt: "You are an expert AI assistant...",
    Messages:     messages,
    Tools:        GetAvailableTools(), // 8 available tools
})
```

**Step 4: LLM Decides to Use Tool**

Response from Anthropic:
```json
{
  "content": [
    {"type": "text", "text": "I'll list your applications."},
    {"type": "tool_use", "id": "toolu_123", "name": "list_applications", "input": {}}
  ],
  "stop_reason": "tool_use"
}
```

**Step 5: Add AI Response to Conversation**
```go
messages = append(messages, llm.Message{
    Role: "assistant",
    Content: []llm.ContentBlock{
        {Type: "text", Text: "I'll list your applications."},
        {Type: "tool_use", ID: "toolu_123", Name: "list_applications", Input: {}},
    },
})
```

**Step 6: Execute Tool**
```go
executor := NewToolExecutor("http://localhost:8081", authToken)
result, err := executor.ExecuteTool(ctx, "list_applications", {})
// result: "{\"applications\": [{\"name\": \"demo-app\", ...}]}"
```

**Step 7: Add Tool Result to Conversation**
```go
messages = append(messages, llm.Message{
    Role: "user",
    Content: []llm.ContentBlock{
        {
            Type:      "tool_result",
            ToolUseID: "toolu_123",
            Content:   result,
        },
    },
})
```

**Step 8: Call LLM Again with Tool Results**
```go
llmResponse, err := s.sdk.LLM().GenerateWithTools(ctx, llm.GenerateWithToolsRequest{
    SystemPrompt: "You are an expert AI assistant...",
    Messages:     messages,
    Tools:        GetAvailableTools(),
})
```

**Step 9: LLM Formats Final Response**

Response from Anthropic:
```json
{
  "content": [
    {"type": "text", "text": "You have 3 applications:\n• demo-app (production) - 5 resources\n• test-service (staging) - 3 resources\n• api-gateway (production) - 8 resources"}
  ],
  "stop_reason": "end_turn"
}
```

**Step 10: Return to User**
```json
{
  "message": "You have 3 applications:\n• demo-app...",
  "tokens_used": 456,
  "timestamp": "2025-10-06T16:30:00Z"
}
```

## Design Decisions

### Why Anthropic's Tool Calling vs Custom Parsing?

**Option 1: Custom Parsing** (Not chosen)
```
AI: "I will now list applications using the API..."
[Parse response for intent]
[Execute action]
```

**Problems**:
- Fragile regex/parsing
- AI might describe actions without clear intent
- Difficult to extract parameters
- Prone to errors

**Option 2: Anthropic Tool Calling** (Chosen)
```
AI: [requests tool: "list_applications" with parameters: {}]
[Execute tool]
AI: [formats results]
```

**Advantages**:
- Structured tool requests (no parsing)
- Built-in parameter validation
- AI reasoning is explicit
- Reliable and extensible

### Why Agent Loop Instead of Single-Shot?

**Single-Shot** (Not chosen):
```
User → AI → Tool → Response
```

**Problem**: Can only execute one tool per request

**Agent Loop** (Chosen):
```
User → AI → Tool → AI → Tool → AI → Response
```

**Advantage**: Supports complex queries requiring multiple tools:
```
User: "deploy my-app and show its status"

AI:
1. deploy_application("my-app") → workflow_id: 45
2. get_workflow(45) → status: running
3. Format: "Deployed my-app. Workflow #45 is running."
```

### Why Max 5 Iterations?

**Protection against infinite loops**:
```go
maxIterations := 5
```

**Scenario**:
- AI requests tool A
- Tool A returns data
- AI requests tool B
- Tool B returns data
- AI requests tool A again (loop)
- ...

**Solution**: Limit to 5 iterations, then return:
```
"I've executed the requested actions, but the conversation exceeded the maximum number of iterations."
```

### Why HTTP API Calls Instead of Direct Database Access?

**Direct Database** (Not chosen):
```go
// Tool executor queries database directly
apps := db.Query("SELECT * FROM applications")
```

**Problems**:
- Bypasses authentication/authorization
- Duplicates business logic
- No audit trail
- Breaks separation of concerns

**HTTP API Calls** (Chosen):
```go
// Tool executor calls internal API
resp := http.Get("/api/specs")
```

**Advantages**:
- Reuses existing auth/authz logic
- Maintains single source of truth for business logic
- Full audit trail (all API calls logged)
- Clean separation: AI service → API → Database

## Tool Categories

### Read-Only Tools (Safe)

**Purpose**: Query platform state without modifications

**Tools**:
- `list_applications`
- `get_application`
- `list_workflows`
- `get_workflow`
- `list_resources`
- `get_dashboard_stats`

**Characteristics**:
- No side effects
- Can be called repeatedly
- Low risk

### Write Tools (Careful)

**Purpose**: Modify platform state

**Tools**:
- `deploy_application`
- `delete_application`

**Characteristics**:
- Have side effects
- Require user confirmation (best practice)
- Higher risk

**Future Enhancement**: Add confirmation prompts for destructive actions:
```
AI: "This will delete demo-app and all its resources. Are you sure? (yes/no)"
```

## Error Handling

### Tool Execution Failures

**Scenario**: API returns 404 or 500

**Handling**:
```go
result, err := executor.ExecuteTool(ctx, toolUse.Name, toolUse.Input)

var resultContent string
if err != nil {
    resultContent = fmt.Sprintf("Error executing tool: %v", err)
} else {
    resultContent = result
}

userContent = append(userContent, llm.ContentBlock{
    Type:      "tool_result",
    ToolUseID: toolUse.ID,
    Content:   resultContent,
    IsError:   err != nil,
})
```

**AI Response**: AI receives error and can explain to user:
```
User: "tell me about nonexistent-app"
AI calls: get_application("nonexistent-app")
Result: "Error: Application not found"
AI: "The application 'nonexistent-app' was not found. You can list all applications with 'list my applications'."
```

### Authentication Failures

**Scenario**: No auth token or invalid token

**Handling**:
```go
authToken := req.AuthToken
if authToken == "" {
    fmt.Printf("Warning: No auth token available for tool execution\n")
}
```

**Result**: Tool calls fail with 401/403, AI explains authentication issue

## Performance Considerations

### Latency

**Total latency** = LLM API + Tool execution + Network

**Typical breakdown**:
- LLM API call 1: ~1-2 seconds
- Tool execution: ~100-500ms
- LLM API call 2: ~1-2 seconds
- **Total**: ~2-4 seconds

**Optimization strategies**:
- Reduce `MaxTokens` for faster LLM responses
- Cache tool results (not implemented)
- Batch tool calls (not supported by Anthropic)

### Token Usage

**Input tokens**:
- System prompt: ~300 tokens
- User message: ~50-200 tokens
- Tool definitions: ~500 tokens
- Conversation history: grows with each turn

**Output tokens**:
- AI response: ~100-800 tokens (limited by `MaxTokens`)

**Cost per request**: ~$0.005-0.02 USD (varies by token count)

## Security Considerations

### Authentication

**All tool calls use user's auth token**:
```go
executor := NewToolExecutor(apiBaseURL, req.AuthToken)
```

**Benefits**:
- User can only perform actions they're authorized for
- Audit trail links actions to users
- No privilege escalation

### Tool Permissions

**Future enhancement**: Per-tool permission checks
```go
// Pseudocode
if tool.IsDestructive() && !user.HasPermission("admin") {
    return "You don't have permission to use this tool"
}
```

### Input Validation

**Handled by JSON Schema** in tool definitions:
```go
"required": ["app_name"]
```

Anthropic validates inputs before calling tools.

## Extensibility

### Adding New Tools

**Steps**:
1. Define tool in `internal/ai/tools.go`
2. Implement executor in `internal/ai/executor.go`
3. Restart server (tools loaded at startup)

**Example**: Add "rollback_application" tool
```go
// In tools.go
{
    Name: "rollback_application",
    Description: "Rollback an application to previous version",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "app_name": {"type": "string"},
            "version": {"type": "string", "description": "Version to rollback to"},
        },
        "required": []string{"app_name"},
    },
}

// In executor.go
case "rollback_application":
    appName := input["app_name"].(string)
    version := input["version"].(string)
    return e.rollbackApplication(ctx, appName, version)
```

No changes to agent loop or conversation handling needed!

## Comparison with Other Approaches

### vs. LangChain Agents

**LangChain**: Python framework with agent abstractions

**innominatus**: Native Go implementation with Anthropic SDK

**Tradeoffs**:
- LangChain: More features, Python ecosystem
- innominatus: Simpler, no external dependencies, type-safe

### vs. OpenAI Function Calling

**OpenAI**: Similar tool calling concept

**Anthropic**: More structured, better for complex multi-step reasoning

**Why Anthropic?**: Claude's reasoning capabilities excel at multi-turn tool calling

## Future Enhancements

### 1. Streaming Responses

**Current**: Wait for full response

**Future**: Stream tokens as they're generated
```
AI: "You have 3 applications..."
[Streaming]
```

### 2. Tool Result Caching

**Current**: Re-execute tools every time

**Future**: Cache tool results for 30 seconds
```
list_applications → cache for 30s
```

### 3. Parallel Tool Execution

**Current**: Execute tools sequentially

**Future**: Execute independent tools in parallel
```
// Execute simultaneously
list_applications()
get_dashboard_stats()
```

### 4. Confirmation Prompts

**Current**: Destructive actions execute immediately

**Future**: Require confirmation
```
AI: "This will delete demo-app. Confirm? (yes/no)"
User: "yes"
AI: [executes delete_application]
```

## See Also

- [Tools Reference](../reference/tools-reference.md) - Complete tool documentation
- [RAG System Design](./rag-system-design.md) - Knowledge base architecture
- [Agent Loop Pattern](./agent-loop-pattern.md) - Multi-turn conversations
- [Authentication Flow](./authentication-flow.md) - Auth token handling
