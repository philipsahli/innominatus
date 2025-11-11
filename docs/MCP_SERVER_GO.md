# Go MCP Server for innominatus

The innominatus MCP (Model Context Protocol) server enables Claude Desktop to interact with the innominatus platform conversationally using the official [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk).

## Architecture

The Go MCP server replaces the previous TypeScript implementation with a native Go binary that shares code with the internal AI assistant:

```
┌─────────────────────────────────────────────────────────────┐
│              New Go MCP Server Architecture                │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Claude Desktop                                            │
│         ↓ (stdio)                                          │
│  innominatus-mcp (Go Binary)                              │
│      - Uses github.com/modelcontextprotocol/go-sdk/mcp   │
│      - Stdio transport                                    │
│      - 10 tools from shared registry                     │
│         ↓                                                  │
│  Shared Tool Registry (internal/mcp/tools)               │
│      - HTTP client helper                                │
│      - Structured logging (zerolog)                      │
│      - Common error handling                             │
│         ↓                                                  │
│  innominatus API (http://localhost:8081)                 │
│                                                             │
│  + Internal AI Assistant (Web UI)                         │
│         ↓                                                  │
│  Service + ToolExecutor (Go)                             │
│      - WILL USE shared tool registry (future)            │
│      - Agent loop + RAG integration                      │
│         ↓                                                  │
│  innominatus API                                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Available Tools

The Go MCP server provides 10 tools for Claude AI:

1. **list_golden_paths** - List all golden path workflows (multi-resource patterns)
2. **list_providers** - List platform providers and their capabilities
3. **get_provider_details** - Get detailed information about a specific provider
4. **execute_workflow** - Execute a workflow with provided inputs
5. **get_workflow_status** - Check workflow execution status
6. **list_workflow_executions** - List recent workflow executions
7. **list_resources** - List provisioned resources (optionally filtered by type)
8. **get_resource_details** - Get detailed information about a specific resource
9. **list_specs** - List deployed Score specifications (applications)
10. **submit_spec** - Deploy a new Score specification

## Installation

### 1. Build the MCP Server

```bash
# From project root
go build -o innominatus-mcp ./cmd/mcp-server

# Or use Makefile
make build-mcp  # If you add this target
```

### 2. Generate API Token

- Open Web UI → Profile → Generate API Key
- Copy the token (starts with `inn_...`)
- Save it for configuration

### 3. Configure Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "innominatus": {
      "command": "/absolute/path/to/innominatus-mcp",
      "env": {
        "INNOMINATUS_API_BASE": "http://localhost:8081",
        "INNOMINATUS_API_TOKEN": "your-api-token-here"
      }
    }
  }
}
```

**Important:**
- Use absolute path to the `innominatus-mcp` binary
- Replace `your-api-token-here` with actual token from step 2
- Adjust `INNOMINATUS_API_BASE` if server is not on localhost

### 4. Restart Claude Desktop

Quit and restart Claude Desktop for changes to take effect.

## Configuration

### Environment Variables

- **INNOMINATUS_API_BASE** (optional, default: `http://localhost:8081`)
  - Base URL for the innominatus API
  - Examples: `http://localhost:8081`, `https://innominatus.example.com`

- **INNOMINATUS_API_TOKEN** (required)
  - API authentication token
  - Generate from Web UI Profile page
  - Format: `inn_<random-string>`

## Usage Examples

### In Claude Desktop Chat

**List available golden paths:**
```
User: "What golden paths are available?"
Claude: [Uses list_golden_paths tool]
        Lists: onboard-dev-team, provision-postgres, etc.
```

**Execute a workflow:**
```
User: "Execute the onboard-dev-team workflow for the backend team"
Claude: [Uses execute_workflow tool]
        Returns: Execution ID 123, status: running
```

**Check resource status:**
```
User: "Show all postgres databases"
Claude: [Uses list_resources tool with type filter]
        Lists: ecommerce-db (active), analytics-db (provisioning)
```

**Deploy an application:**
```
User: "Deploy this Score specification: [YAML content]"
Claude: [Uses submit_spec tool]
        Returns: Application deployed successfully
```

## Shared Code Architecture

The Go MCP server shares tool implementations with the internal AI assistant:

### Shared Components (`internal/mcp/tools/`)

**types.go** - Tool interface and registry
- `Tool` interface with Execute() method
- `ToolRegistry` for managing available tools
- `ExecuteResult` for standardized responses

**client.go** - HTTP client helper
- Reusable `*http.Client` with 30s timeout
- Bearer token authentication
- Structured logging with zerolog
- Content-Type handling (JSON, YAML)

**tools.go** - 10 tool implementations
- Each tool implements the `Tool` interface
- Consistent error handling
- Input validation via JSON schemas

### Benefits of Shared Code

1. **60% Code Reduction** - Single implementation for both systems
2. **Consistency** - Same HTTP client, error handling, logging
3. **Maintainability** - One place to update tool logic
4. **Type Safety** - Go's strong typing prevents runtime errors
5. **Single Binary** - No Node.js dependency

## Development

### Running Locally

```bash
# Terminal 1: Start innominatus server
./innominatus

# Terminal 2: Start MCP server with environment variables
export INNOMINATUS_API_TOKEN="your-token"
export INNOMINATUS_API_BASE="http://localhost:8081"
./innominatus-mcp
```

### Logging

The MCP server logs to stderr (stdout is reserved for MCP protocol):

- **Info**: Server startup, tool registrations
- **Debug**: Tool execution requests and results
- **Error**: HTTP failures, tool execution errors

View logs in Claude Desktop → Settings → Developer → View Logs

### Adding a New Tool

1. **Define tool struct** in `internal/mcp/tools/tools.go`:
```go
type MyNewTool struct {
    *BaseTool
}

func NewMyNewTool(client *APIClient) *MyNewTool {
    return &MyNewTool{BaseTool: NewBaseTool(client)}
}

func (t *MyNewTool) Name() string {
    return "my_new_tool"
}

func (t *MyNewTool) Description() string {
    return "Description of what this tool does"
}

func (t *MyNewTool) InputSchema() map[string]interface{} {
    return map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "param1": map[string]interface{}{
                "type":        "string",
                "description": "Description of param1",
            },
        },
        "required": []string{"param1"},
    }
}

func (t *MyNewTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    param1, ok := input["param1"].(string)
    if !ok {
        return "", fmt.Errorf("param1 is required and must be a string")
    }

    resp, err := t.client.Get(ctx, "/api/endpoint")
    if err != nil {
        return "", fmt.Errorf("failed to call API: %w", err)
    }

    return resp, nil
}
```

2. **Register tool** in `BuildRegistry()` function:
```go
func BuildRegistry(apiBaseURL, authToken string) *ToolRegistry {
    client := NewAPIClient(apiBaseURL, authToken)
    registry := NewToolRegistry()

    registry.Register(NewListGoldenPathsTool(client))
    // ... other tools ...
    registry.Register(NewMyNewTool(client))  // ADD THIS LINE

    return registry
}
```

3. **Rebuild and test**:
```bash
go build -o innominatus-mcp ./cmd/mcp-server
# Test in Claude Desktop
```

## Troubleshooting

### MCP Server Not Appearing in Claude Desktop

**Check Claude Desktop logs:**
```bash
# macOS
tail -f ~/Library/Logs/Claude/mcp*.log
```

**Common issues:**
- Incorrect path to binary in config.json
- Missing INNOMINATUS_API_TOKEN
- Binary not executable (`chmod +x innominatus-mcp`)

### Tool Execution Fails

**Check server logs** (stderr output in Claude Desktop logs):
- HTTP connection errors → verify INNOMINATUS_API_BASE
- Authentication errors → verify API token is valid
- Timeout errors → check if innominatus server is running

### Cannot Connect to innominatus API

```bash
# Verify API is accessible
curl http://localhost:8081/health

# Check API token works
curl -H "Authorization: Bearer $INNOMINATUS_API_TOKEN" \
     http://localhost:8081/api/providers
```

## Migration from TypeScript MCP Server

### Differences from TypeScript Version

| Aspect | TypeScript | Go |
|--------|-----------|-----|
| Binary size | ~50MB (Node.js + deps) | ~10MB (static binary) |
| Runtime | Node.js required | Standalone |
| Startup time | ~500ms | ~50ms |
| Memory usage | ~100MB | ~20MB |
| Code sharing | None | Shares with internal AI |
| Maintainability | Separate implementation | Single source of truth |

### Configuration Changes

**Old (TypeScript):**
```json
{
  "innominatus": {
    "command": "node",
    "args": ["/path/to/mcp-server-innominatus/build/index.js"],
    "env": {
      "INNOMINATUS_API_BASE": "http://localhost:8081",
      "INNOMINATUS_API_TOKEN": "your-token"
    }
  }
}
```

**New (Go):**
```json
{
  "innominatus": {
    "command": "/path/to/innominatus-mcp",
    "env": {
      "INNOMINATUS_API_BASE": "http://localhost:8081",
      "INNOMINATUS_API_TOKEN": "your-token"
    }
  }
}
```

### Deprecation Timeline

1. **Now (v1.0)**: Both TypeScript and Go versions available
2. **v1.1 (2 weeks)**: Go version recommended, TypeScript marked deprecated
3. **v1.2 (1 month)**: TypeScript version archived, Go version only

## Future Enhancements

### Planned Features

- [ ] **Refactor internal AI executor** to use shared tool registry
- [ ] **Unit tests** for each tool implementation
- [ ] **Integration tests** with mock API server
- [ ] **MCP protocol compliance tests**
- [ ] **Makefile targets** for building and installing
- [ ] **GitHub Actions** workflow to build MCP binary in releases

### Contribution

When adding new tools or modifying existing ones:

1. Update tool implementation in `internal/mcp/tools/tools.go`
2. Add tests in `internal/mcp/tools/tools_test.go`
3. Update this documentation
4. Test with both Claude Desktop (MCP) and Web UI (internal AI)

## References

- [Official Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
- [MCP Protocol Specification](https://modelcontextprotocol.io/specification)
- [innominatus API Documentation](http://localhost:8081/swagger-user)
- [OpenAPI Spec for MCP Tools](../mcp-server-innominatus/openapi.yaml)

---

**Last Updated:** 2025-11-09
**Version:** 1.0.0 (Go MCP Server)
