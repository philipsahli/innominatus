#!/usr/bin/env node

/**
 * innominatus MCP Server
 *
 * Enables Claude AI to interact with innominatus platform orchestration APIs.
 * Provides tools for listing golden paths, executing workflows, managing resources, etc.
 */

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  Tool,
} from "@modelcontextprotocol/sdk/types.js";

// Configuration from environment
const API_BASE = process.env.INNOMINATUS_API_BASE || "http://localhost:8081";
const API_TOKEN = process.env.INNOMINATUS_API_TOKEN || "";

if (!API_TOKEN) {
  console.error("ERROR: INNOMINATUS_API_TOKEN environment variable is required");
  process.exit(1);
}

/**
 * Helper function to make authenticated API requests
 */
async function apiRequest(endpoint: string, options: RequestInit = {}): Promise<any> {
  const url = `${API_BASE}${endpoint}`;

  const response = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "Authorization": `Bearer ${API_TOKEN}`,
      ...options.headers,
    },
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API request failed (${response.status}): ${errorText}`);
  }

  // Handle empty responses
  const text = await response.text();
  return text ? JSON.parse(text) : null;
}

// Initialize MCP server
const server = new Server(
  {
    name: "innominatus-mcp",
    version: "1.0.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// Define available tools
const tools: Tool[] = [
  {
    name: "list_golden_paths",
    description: "List all available golden path workflows. Golden paths are standardized, opinionated workflows for common platform tasks like onboarding teams, deploying applications, or provisioning infrastructure.",
    inputSchema: {
      type: "object",
      properties: {},
      required: [],
    },
  },
  {
    name: "list_providers",
    description: "List all platform providers and their capabilities. Providers are teams that own specific resource types (e.g., database-team owns 'postgres', container-team owns 'kubernetes-namespace').",
    inputSchema: {
      type: "object",
      properties: {},
      required: [],
    },
  },
  {
    name: "get_provider_details",
    description: "Get detailed information about a specific provider including its workflows and resource type capabilities.",
    inputSchema: {
      type: "object",
      properties: {
        provider_name: {
          type: "string",
          description: "Name of the provider (e.g., 'database-team', 'container-team')",
        },
      },
      required: ["provider_name"],
    },
  },
  {
    name: "execute_workflow",
    description: "Execute a workflow (golden path). This starts a new workflow execution with the provided inputs. Returns the execution ID for tracking.",
    inputSchema: {
      type: "object",
      properties: {
        workflow_name: {
          type: "string",
          description: "Name of the workflow to execute (e.g., 'onboard-dev-team', 'provision-postgres')",
        },
        inputs: {
          type: "object",
          description: "Input parameters for the workflow as key-value pairs",
          additionalProperties: true,
        },
      },
      required: ["workflow_name"],
    },
  },
  {
    name: "get_workflow_status",
    description: "Get the current status of a workflow execution including step-by-step progress and any errors.",
    inputSchema: {
      type: "object",
      properties: {
        execution_id: {
          type: "number",
          description: "Workflow execution ID returned from execute_workflow",
        },
      },
      required: ["execution_id"],
    },
  },
  {
    name: "list_workflow_executions",
    description: "List recent workflow executions with their status. Useful for seeing what workflows are running or have completed recently.",
    inputSchema: {
      type: "object",
      properties: {
        limit: {
          type: "number",
          description: "Maximum number of executions to return (default: 20)",
        },
      },
      required: [],
    },
  },
  {
    name: "list_resources",
    description: "List provisioned platform resources (databases, namespaces, buckets, etc.). Can be filtered by type.",
    inputSchema: {
      type: "object",
      properties: {
        type: {
          type: "string",
          description: "Filter by resource type (e.g., 'postgres', 'kubernetes-namespace', 's3-bucket')",
        },
      },
      required: [],
    },
  },
  {
    name: "get_resource_details",
    description: "Get detailed information about a specific resource including its state, properties, and associated workflow.",
    inputSchema: {
      type: "object",
      properties: {
        resource_id: {
          type: "number",
          description: "Resource ID",
        },
      },
      required: ["resource_id"],
    },
  },
  {
    name: "list_specs",
    description: "List deployed Score specifications (applications). Shows what applications are currently deployed on the platform.",
    inputSchema: {
      type: "object",
      properties: {},
      required: [],
    },
  },
  {
    name: "submit_spec",
    description: "Deploy a new Score specification (application). This will automatically provision any required resources (databases, storage, etc.).",
    inputSchema: {
      type: "object",
      properties: {
        spec_yaml: {
          type: "string",
          description: "The Score specification in YAML format",
        },
      },
      required: ["spec_yaml"],
    },
  },
];

// Register tools list handler
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return { tools };
});

// Register tool execution handler
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  try {
    let result: any;

    switch (name) {
      case "list_golden_paths": {
        const providers = await apiRequest("/api/providers");

        // Extract golden paths from all providers
        const goldenPaths = providers.flatMap((provider: any) =>
          (provider.workflows || [])
            .filter((w: any) => w.category === "goldenpath")
            .map((w: any) => ({
              name: w.name,
              provider: provider.name,
              description: w.description,
              tags: w.tags || [],
            }))
        );

        result = {
          count: goldenPaths.length,
          golden_paths: goldenPaths,
        };
        break;
      }

      case "list_providers": {
        const providers = await apiRequest("/api/providers");

        result = {
          count: providers.length,
          providers: providers.map((p: any) => ({
            name: p.name,
            category: p.category,
            description: p.description,
            resource_types: p.capabilities?.resourceTypes || [],
            workflow_count: p.workflows?.length || 0,
          })),
        };
        break;
      }

      case "get_provider_details": {
        const { provider_name } = args as { provider_name: string };
        const provider = await apiRequest(`/api/providers/${encodeURIComponent(provider_name)}`);

        result = {
          name: provider.name,
          version: provider.version,
          category: provider.category,
          description: provider.description,
          capabilities: provider.capabilities,
          workflows: provider.workflows,
        };
        break;
      }

      case "execute_workflow": {
        const { workflow_name, inputs = {} } = args as {
          workflow_name: string;
          inputs?: Record<string, any>;
        };

        const execution = await apiRequest("/api/workflows/execute", {
          method: "POST",
          body: JSON.stringify({
            workflow_name,
            inputs,
          }),
        });

        result = {
          execution_id: execution.id,
          workflow_name: execution.workflow_name,
          status: execution.status,
          started_at: execution.started_at,
          message: `Workflow '${workflow_name}' execution started with ID ${execution.id}`,
        };
        break;
      }

      case "get_workflow_status": {
        const { execution_id } = args as { execution_id: number };
        const execution = await apiRequest(`/api/workflows/${execution_id}`);

        result = {
          execution_id: execution.id,
          workflow_name: execution.workflow_name,
          status: execution.status,
          started_at: execution.started_at,
          completed_at: execution.completed_at,
          error_message: execution.error_message,
          steps: execution.steps?.map((s: any) => ({
            name: s.name,
            type: s.type,
            status: s.status,
            started_at: s.started_at,
            completed_at: s.completed_at,
          })),
        };
        break;
      }

      case "list_workflow_executions": {
        const { limit = 20 } = args as { limit?: number };
        const executions = await apiRequest("/api/workflows");

        result = {
          count: executions.length,
          executions: executions.slice(0, limit).map((e: any) => ({
            execution_id: e.id,
            workflow_name: e.workflow_name,
            status: e.status,
            started_at: e.started_at,
            completed_at: e.completed_at,
          })),
        };
        break;
      }

      case "list_resources": {
        const { type } = args as { type?: string };
        const endpoint = type ? `/api/resources?type=${encodeURIComponent(type)}` : "/api/resources";
        const resources = await apiRequest(endpoint);

        result = {
          count: resources.length,
          resources: resources.map((r: any) => ({
            id: r.id,
            name: r.name,
            type: r.type,
            spec_name: r.spec_name,
            state: r.state,
            workflow_execution_id: r.workflow_execution_id,
          })),
        };
        break;
      }

      case "get_resource_details": {
        const { resource_id } = args as { resource_id: number };
        const resource = await apiRequest(`/api/resources/${resource_id}`);

        result = {
          id: resource.id,
          name: resource.name,
          type: resource.type,
          spec_name: resource.spec_name,
          state: resource.state,
          workflow_execution_id: resource.workflow_execution_id,
          properties: resource.properties,
          error_message: resource.error_message,
          created_at: resource.created_at,
          updated_at: resource.updated_at,
        };
        break;
      }

      case "list_specs": {
        const specs = await apiRequest("/api/specs");

        result = {
          count: specs.length,
          specs: specs.map((s: any) => ({
            name: s.name,
            created_at: s.created_at,
            resources_count: s.resources?.length || 0,
          })),
        };
        break;
      }

      case "submit_spec": {
        const { spec_yaml } = args as { spec_yaml: string };

        const spec = await apiRequest("/api/specs", {
          method: "POST",
          headers: {
            "Content-Type": "application/yaml",
          },
          body: spec_yaml,
        });

        result = {
          name: spec.name,
          message: `Score spec '${spec.name}' submitted successfully`,
          resources_requested: spec.resources?.length || 0,
        };
        break;
      }

      default:
        throw new Error(`Unknown tool: ${name}`);
    }

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  } catch (error: any) {
    return {
      content: [
        {
          type: "text",
          text: `Error: ${error.message}`,
        },
      ],
      isError: true,
    };
  }
});

// Start the server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);

  // Log to stderr (stdout is used for MCP protocol)
  console.error("innominatus MCP server started");
  console.error(`API Base: ${API_BASE}`);
  console.error(`Tools available: ${tools.length}`);
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
