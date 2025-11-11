# innominatus MCP Server

Model Context Protocol (MCP) server for [innominatus](https://github.com/philipsahli/innominatus) platform orchestration. Enables Claude AI to interact with innominatus APIs conversationally.

## Features

**10 MCP Tools:**
- `list_golden_paths` - List all golden path workflows
- `list_providers` - List platform providers and capabilities
- `get_provider_details` - Get detailed provider information
- `execute_workflow` - Execute a workflow
- `get_workflow_status` - Check workflow execution status
- `list_workflow_executions` - List recent workflow executions
- `list_resources` - List provisioned resources
- `get_resource_details` - Get resource details
- `list_specs` - List deployed Score specifications
- `submit_spec` - Deploy a new Score specification

## Installation

### Prerequisites

- Node.js 18+ installed
- innominatus server running (default: http://localhost:8081)
- API token for authentication

### 1. Install Dependencies

```bash
cd mcp-server-innominatus
npm install
```

### 2. Build TypeScript

```bash
npm run build
```

This compiles `src/index.ts` to `build/index.js`.

### 3. Generate API Token

Generate an API token from the innominatus Web UI:

1. Open http://localhost:8081
2. Navigate to **Profile** page
3. Click **Generate API Key**
4. Copy the token (starts with `inn_...`)

### 4. Configure Claude Desktop

Edit Claude Desktop configuration file:

**macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows:** `%APPDATA%\Claude\claude_desktop_config.json`

Add the innominatus MCP server:

```json
{
  "mcpServers": {
    "innominatus": {
      "command": "node",
      "args": [
        "/Users/philipsahli/projects/innominatus/mcp-server-innominatus/build/index.js"
      ],
      "env": {
        "INNOMINATUS_API_BASE": "http://localhost:8081",
        "INNOMINATUS_API_TOKEN": "inn_your_actual_token_here"
      }
    }
  }
}
```

**Important:**
- Replace `/Users/philipsahli/projects/innominatus` with your actual path
- Replace `inn_your_actual_token_here` with your API token

### 5. Restart Claude Desktop

Close and reopen Claude Desktop for the configuration to take effect.

## Usage

Once configured, you can ask Claude questions about your innominatus platform:

### Example Conversations

**List Golden Paths:**
```
User: "What golden paths are available?"

Claude: [Uses list_golden_paths tool]
Available golden paths:
- onboard-dev-team (identity-team)
- provision-postgres (database-team)
- provision-gitea-repo (container-team)
...
```

**Execute Workflow:**
```
User: "Execute the onboard-dev-team workflow for platform-team"

Claude: [Uses execute_workflow tool]
Workflow execution started:
- Execution ID: 123
- Status: running
```

**Check Status:**
```
User: "What's the status of workflow execution 123?"

Claude: [Uses get_workflow_status tool]
Workflow execution 123 (onboard-dev-team):
- Status: completed
- Started: 2025-10-30T20:15:00Z
- Completed: 2025-10-30T20:18:45Z
- All 5 steps completed successfully
```

**List Resources:**
```
User: "Show all postgres resources"

Claude: [Uses list_resources tool with type filter]
Found 3 postgres resources:
- ecommerce-db (active)
- analytics-db (provisioning)
- staging-db (active)
```

**Deploy Application:**
```
User: "Deploy an application called 'my-app' with a postgres database"

Claude: [Uses submit_spec tool]
I'll create a Score spec for your application...
[Submits spec and confirms deployment]
```

## Tool Reference

### list_golden_paths

List all golden path workflows.

**Arguments:** None

**Returns:**
```json
{
  "count": 5,
  "golden_paths": [
    {
      "name": "onboard-dev-team",
      "provider": "identity-team",
      "description": "Complete team onboarding",
      "tags": ["onboarding", "team"]
    }
  ]
}
```

### execute_workflow

Execute a workflow with inputs.

**Arguments:**
```json
{
  "workflow_name": "onboard-dev-team",
  "inputs": {
    "team_name": "platform-team",
    "github_org": "my-company"
  }
}
```

**Returns:**
```json
{
  "execution_id": 123,
  "workflow_name": "onboard-dev-team",
  "status": "running",
  "started_at": "2025-10-30T20:15:00Z"
}
```

### list_resources

List provisioned resources, optionally filtered by type.

**Arguments:**
```json
{
  "type": "postgres"  // optional
}
```

**Returns:**
```json
{
  "count": 3,
  "resources": [
    {
      "id": 1,
      "name": "ecommerce-db",
      "type": "postgres",
      "spec_name": "ecommerce-backend",
      "state": "active"
    }
  ]
}
```

## Development

### Watch Mode

For development with auto-rebuild:

```bash
npm run watch
```

### Testing Manually

Test the MCP server directly:

```bash
export INNOMINATUS_API_TOKEN="your-token"
npm test
```

This runs the server and you can send MCP protocol messages via stdin.

### Debugging

The MCP server logs to stderr (stdout is reserved for MCP protocol):

```bash
# Check logs when running in Claude Desktop
tail -f ~/Library/Logs/Claude/mcp*.log
```

## Troubleshooting

### "API request failed (401)"

**Cause:** Invalid or missing API token

**Solution:**
1. Generate a new API token from Web UI Profile page
2. Update `INNOMINATUS_API_TOKEN` in claude_desktop_config.json
3. Restart Claude Desktop

### "API request failed (connection refused)"

**Cause:** innominatus server not running

**Solution:**
```bash
# Start innominatus server
cd /path/to/innominatus
./innominatus
```

### "Tool not found"

**Cause:** MCP server not loaded by Claude Desktop

**Solution:**
1. Check claude_desktop_config.json syntax (valid JSON)
2. Verify file path to build/index.js is correct
3. Check Claude Desktop logs for errors
4. Restart Claude Desktop

### "Cannot find module '@modelcontextprotocol/sdk'"

**Cause:** Dependencies not installed

**Solution:**
```bash
cd mcp-server-innominatus
npm install
npm run build
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `INNOMINATUS_API_BASE` | No | `http://localhost:8081` | Base URL of innominatus API |
| `INNOMINATUS_API_TOKEN` | **Yes** | - | API authentication token |

## Architecture

```
Claude Desktop
    ↓
MCP Protocol (stdio)
    ↓
mcp-server-innominatus
    ↓
HTTP/REST
    ↓
innominatus API Server (port 8081)
    ↓
PostgreSQL Database
```

## Security Notes

- API token is stored in Claude Desktop config (plain text)
- Ensure file permissions restrict access: `chmod 600 ~/Library/Application\ Support/Claude/claude_desktop_config.json`
- API token has same permissions as the user who generated it
- Tokens can be revoked from Web UI Profile page

## Contributing

This MCP server is part of the innominatus project. For issues or contributions:

- **GitHub:** https://github.com/philipsahli/innominatus
- **Issues:** https://github.com/philipsahli/innominatus/issues

## License

MIT - See main innominatus project for details.

## Related Documentation

- [innominatus Documentation](../README.md)
- [CLAUDE.md](../CLAUDE.md) - Development guide
- [QUICKREF.md](../QUICKREF.md) - Command reference
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification
