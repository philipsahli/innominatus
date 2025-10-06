# AI Tools Reference

**Purpose**: Complete reference for all AI assistant tools that interact with the platform.

The AI assistant has access to 8 tools for platform interaction. These tools are automatically invoked when the AI determines they're needed to answer your question or perform an action.

## Tool Execution Flow

1. User asks a question or requests an action
2. AI determines which tool(s) are needed
3. AI calls tool with appropriate parameters
4. Tool executes HTTP API request to platform
5. AI receives result and formats response for user

All tools use the user's authentication token from the current session.

---

## list_applications

**Description**: List all deployed applications in the platform.

**When AI Uses This**:
- "list my applications"
- "show all apps"
- "what applications are deployed"
- "which apps are running"

**Parameters**: None

**Returns**:
```json
{
  "applications": [
    {
      "name": "demo-app",
      "environment": "production",
      "status": "running",
      "resource_count": 5
    }
  ]
}
```

**API Endpoint**: `GET /api/specs`

**Example Response to User**:
```
You have 3 applications:
• demo-app (production) - 5 resources
• test-service (staging) - 3 resources
• api-gateway (production) - 8 resources
```

---

## get_application

**Description**: Get detailed information about a specific application including Score specification, resources, and deployment status.

**When AI Uses This**:
- "tell me about my-app"
- "show details for demo-app"
- "what resources does api-gateway use"

**Parameters**:
- `app_name` (string, required): The name of the application to retrieve

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "app_name": {
      "type": "string",
      "description": "The name of the application to retrieve"
    }
  },
  "required": ["app_name"]
}
```

**Returns**:
```json
{
  "name": "demo-app",
  "environment": "production",
  "status": "running",
  "score_spec": "apiVersion: score.dev/v1b1...",
  "resources": [
    {"name": "postgres-db", "type": "postgres"},
    {"name": "redis-cache", "type": "redis"}
  ],
  "created_at": "2025-10-04T10:30:00Z"
}
```

**API Endpoint**: `GET /api/specs/{app_name}`

**Example Response to User**:
```
demo-app (production):
• Status: running
• 5 resources: 2 postgres, 1 redis, 1 volume, 1 route
• Created: 2 days ago
```

---

## deploy_application

**Description**: Deploy a Score specification to the platform.

**When AI Uses This**:
- "deploy this spec"
- "deploy the application"
- (After generating a spec) "deploy it"

**Parameters**:
- `score_spec` (string, required): The complete Score specification in YAML format
- `environment` (string, optional): Target environment (default: "development")

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "score_spec": {
      "type": "string",
      "description": "Complete Score specification in YAML format"
    },
    "environment": {
      "type": "string",
      "description": "Target environment (production, staging, development)"
    }
  },
  "required": ["score_spec"]
}
```

**Returns**:
```json
{
  "message": "Application deployed successfully",
  "workflow_id": 45,
  "status": "running"
}
```

**API Endpoint**: `POST /api/specs`

**Example Response to User**:
```
Application deployed successfully.
Workflow execution #45 started.
Status: running
```

---

## delete_application

**Description**: Remove an application and all its resources from the platform.

**When AI Uses This**:
- "delete my-app"
- "remove demo-app"
- "uninstall test-service"

**Parameters**:
- `app_name` (string, required): The name of the application to delete

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "app_name": {
      "type": "string",
      "description": "The name of the application to delete"
    }
  },
  "required": ["app_name"]
}
```

**Returns**:
```json
{
  "message": "Application deleted successfully",
  "app_name": "demo-app"
}
```

**API Endpoint**: `DELETE /api/specs/{app_name}`

**Example Response to User**:
```
demo-app has been deleted successfully.
All associated resources have been removed.
```

---

## list_workflows

**Description**: View all workflow executions with their status and execution time.

**When AI Uses This**:
- "show workflows"
- "list recent workflows"
- "what workflows are running"
- "show failed workflows"

**Parameters**:
- `status` (string, optional): Filter by status (pending, running, completed, failed)
- `app_name` (string, optional): Filter by application name

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "status": {
      "type": "string",
      "description": "Filter by workflow status"
    },
    "app_name": {
      "type": "string",
      "description": "Filter by application name"
    }
  },
  "required": []
}
```

**Returns**:
```json
{
  "workflows": [
    {
      "id": 123,
      "name": "deploy-app",
      "app_name": "demo-app",
      "status": "completed",
      "duration": "2m 34s",
      "started_at": "2025-10-06T10:00:00Z"
    }
  ]
}
```

**API Endpoint**: `GET /api/workflows?status={status}&app={app_name}`

**Example Response to User**:
```
Recent workflows:
1. deploy-app (demo-app) - completed (2m 34s)
2. db-lifecycle (api-gateway) - running (1m 12s)
3. ephemeral-env (test-env) - completed (45s)
```

---

## get_workflow

**Description**: Get detailed information about a specific workflow execution including steps, logs, and errors.

**When AI Uses This**:
- "show workflow 123"
- "what's the status of workflow 45"
- "tell me about workflow execution 67"

**Parameters**:
- `workflow_id` (integer, required): The ID of the workflow execution

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "workflow_id": {
      "type": "integer",
      "description": "The ID of the workflow execution to retrieve"
    }
  },
  "required": ["workflow_id"]
}
```

**Returns**:
```json
{
  "id": 123,
  "name": "deploy-app",
  "app_name": "demo-app",
  "status": "completed",
  "steps": [
    {"name": "validate", "status": "completed"},
    {"name": "provision", "status": "completed"},
    {"name": "deploy", "status": "completed"}
  ],
  "started_at": "2025-10-06T10:00:00Z",
  "completed_at": "2025-10-06T10:02:34Z",
  "duration": "2m 34s"
}
```

**API Endpoint**: `GET /api/workflows/{workflow_id}`

**Example Response to User**:
```
Workflow #123 (deploy-app):
• Application: demo-app
• Status: completed
• Duration: 2m 34s
• Steps: validate ✓, provision ✓, deploy ✓
```

---

## list_resources

**Description**: View all platform resources (databases, caches, volumes, routes, storage) across all applications.

**When AI Uses This**:
- "show all resources"
- "list platform resources"
- "what databases are running"
- "show all postgres instances"

**Parameters**:
- `type` (string, optional): Filter by resource type (postgres, redis, volume, route, s3)
- `app_name` (string, optional): Filter by application name

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "description": "Filter by resource type"
    },
    "app_name": {
      "type": "string",
      "description": "Filter by application name"
    }
  },
  "required": []
}
```

**Returns**:
```json
{
  "resources": [
    {
      "name": "demo-app-db",
      "type": "postgres",
      "app_name": "demo-app",
      "status": "running",
      "properties": {
        "version": "15"
      }
    }
  ]
}
```

**API Endpoint**: `GET /api/resources?type={type}&app={app_name}`

**Example Response to User**:
```
Platform Resources (47 total):
• Postgres: 8 instances
• Redis: 5 instances
• Volumes: 12 instances
• Routes: 15 instances
• S3 Buckets: 7 instances
```

---

## get_dashboard_stats

**Description**: Get platform-wide statistics including application count, resource distribution, and workflow metrics.

**When AI Uses This**:
- "show platform statistics"
- "get platform stats"
- "platform overview"
- "how many apps are deployed"

**Parameters**: None

**Returns**:
```json
{
  "total_applications": 12,
  "running_workflows": 3,
  "total_resources": 47,
  "resource_breakdown": {
    "postgres": 8,
    "redis": 5,
    "volume": 12,
    "route": 15,
    "s3": 7
  },
  "environment_distribution": {
    "production": 5,
    "staging": 4,
    "development": 3
  }
}
```

**API Endpoint**: `GET /api/stats`

**Example Response to User**:
```
Platform Statistics:
• Total Applications: 12
• Running Workflows: 3
• Total Resources: 47
• Environments: production (5), staging (4), development (3)
```

---

## Tool Definitions Source Code

Tools are defined in `/Users/philipsahli/projects/innominatus/internal/ai/tools.go`

Tool execution logic is in `/Users/philipsahli/projects/innominatus/internal/ai/executor.go`

## Authentication

All tools use the user's authentication token from the current session. The token is automatically extracted from the `Authorization: Bearer <token>` header and passed to tool executors.

If a tool execution fails with authentication errors, check:
- User is logged in (web UI)
- API key is set (CLI): `export IDP_API_KEY=<your-key>`

## Error Handling

Tools return errors in a structured format:
```json
{
  "error": "Application not found",
  "code": "NOT_FOUND"
}
```

The AI assistant will interpret these errors and provide helpful guidance to the user.

## Adding New Tools

To add a new tool:
1. Define the tool in `internal/ai/tools.go`
2. Implement the executor in `internal/ai/executor.go`
3. Update this documentation
4. Restart the server

See [Tool Calling Architecture](../explanations/tool-calling-architecture.md) for implementation details.
