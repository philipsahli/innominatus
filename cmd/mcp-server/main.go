package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"innominatus/internal/mcp/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup structured logging to stderr (stdio is for MCP protocol)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Get configuration from environment
	apiBase := os.Getenv("INNOMINATUS_API_BASE")
	if apiBase == "" {
		apiBase = "http://localhost:8081"
	}

	apiToken := os.Getenv("INNOMINATUS_API_TOKEN")
	if apiToken == "" {
		log.Fatal().Msg("INNOMINATUS_API_TOKEN environment variable is required")
	}

	log.Info().
		Str("api_base", apiBase).
		Msg("Starting innominatus MCP server")

	// Create tool registry with all 10 tools
	registry := tools.BuildRegistry(apiBase, apiToken)

	// Create MCP server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "innominatus",
			Version: "1.0.0",
		},
		nil, // No special options needed
	)

	// Register all tools from registry
	for _, tool := range registry.List() {
		// Capture tool in closure
		t := tool

		// Convert our tool schema to json.RawMessage
		schemaBytes, err := json.Marshal(t.InputSchema())
		if err != nil {
			log.Fatal().Err(err).Str("tool", t.Name()).Msg("Failed to marshal tool schema")
		}

		// Create MCP tool definition
		mcpTool := &mcp.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: json.RawMessage(schemaBytes),
		}

		// Create handler function
		handler := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			log.Debug().
				Str("tool", t.Name()).
				RawJSON("args", req.Params.Arguments).
				Msg("Executing tool")

			// Convert JSON arguments to map
			var args map[string]interface{}
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				log.Error().
					Err(err).
					Str("tool", t.Name()).
					Msg("Failed to unmarshal tool arguments")
				return nil, fmt.Errorf("invalid arguments: %w", err)
			}

			// Execute tool
			result, err := t.Execute(ctx, args)
			if err != nil {
				log.Error().
					Err(err).
					Str("tool", t.Name()).
					Msg("Tool execution failed")
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						&mcp.TextContent{Text: fmt.Sprintf("Error: %s", err.Error())},
					},
					IsError: true,
				}, nil
			}

			log.Debug().
				Str("tool", t.Name()).
				Str("result", result).
				Msg("Tool execution successful")

			// Return result as text content
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: result},
				},
			}, nil
		}

		// Register tool
		server.AddTool(mcpTool, handler)

		log.Info().
			Str("tool", t.Name()).
			Str("description", t.Description()).
			Msg("Registered tool")
	}

	// Create stdio transport
	transport := &mcp.StdioTransport{}

	// Connect server to stdio transport
	ctx := context.Background()
	log.Info().Msg("MCP server ready - listening on stdio")

	_, err := server.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect server to stdio transport")
	}

	// Block forever - the server runs asynchronously
	select {}
}
