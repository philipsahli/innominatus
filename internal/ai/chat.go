package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai/llm"
	"github.com/philipsahli/innominatus-ai-sdk/pkg/platformai/rag"
	"github.com/rs/zerolog/log"
)

// Chat handles chat interactions with the AI assistant with tool calling support
func (s *Service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI service is not enabled")
	}

	// Retrieve relevant context from RAG
	log.Debug().
		Str("query", req.Message).
		Int("top_k", 3).
		Float64("min_score", 0.3).
		Msg("Retrieving RAG context")

	ragResponse, err := s.sdk.RAG().Retrieve(ctx, rag.RetrieveRequest{
		Query:    req.Message,
		TopK:     3,
		MinScore: 0.3,
	})
	if err != nil {
		log.Warn().
			Err(err).
			Str("query", req.Message).
			Msg("Failed to retrieve RAG context")
		ragResponse = &rag.RetrieveResponse{Context: ""}
	} else {
		log.Debug().
			Int("results_count", len(ragResponse.Results)).
			Int("context_length", len(ragResponse.Context)).
			Msg("Retrieved RAG context")
	}

	// Extract citations from RAG results
	var citations []string
	for _, result := range ragResponse.Results {
		if source, ok := result.Document.Metadata["source"]; ok {
			citations = append(citations, source)
		}
	}

	if len(citations) > 0 {
		log.Debug().
			Strs("citations", citations).
			Msg("Extracted citations")
	}

	// Build system prompt with tool awareness
	systemPrompt := buildSystemPromptWithTools()

	// Fetch and inject current user context if auth token is available
	if req.AuthToken != "" {
		executor := NewToolExecutor("http://localhost:8081", req.AuthToken)
		userInfo, err := executor.getCurrentUser(ctx)
		if err == nil {
			systemPrompt += "\n\n**Current User Context:**\n" +
				"You are currently assisting the following user:\n" +
				userInfo + "\n\n" +
				"All tool executions will be performed with this user's permissions and identity. " +
				"If the user attempts an action they don't have permission for, the tool will fail with an authorization error."
		} else {
			log.Warn().
				Err(err).
				Msg("Failed to fetch current user context for AI assistant")
		}
	}

	// Prepare initial messages with context
	contextMessage := ragResponse.Context
	if contextMessage != "" {
		contextMessage = "Here is relevant context from the documentation:\n\n" + contextMessage + "\n\n"
	}

	// Build messages array starting with conversation history
	messages := []llm.Message{}

	// Add previous conversation history if provided
	if len(req.ConversationHistory) > 0 {
		for _, msg := range req.ConversationHistory {
			// Build content including spec if present
			content := msg.Content
			if msg.Spec != "" {
				content += "\n\n[Generated Score Specification]:\n```yaml\n" + msg.Spec + "\n```"
			}

			messages = append(messages, llm.Message{
				Role: msg.Role,
				Content: []llm.ContentBlock{
					{
						Type: "text",
						Text: content,
					},
				},
			})
		}
	}

	// Add current user message
	messages = append(messages, llm.Message{
		Role: "user",
		Content: []llm.ContentBlock{
			{
				Type: "text",
				Text: contextMessage + req.Message,
			},
		},
	})

	// Get available tools
	tools := GetAvailableTools()

	// Agent loop: keep calling LLM until it stops requesting tools
	maxIterations := 10 // Increased from 5 to support complex multi-tool queries
	totalTokens := 0
	var toolResults []string

	log.Debug().
		Int("max_iterations", maxIterations).
		Int("available_tools", len(tools)).
		Msg("Starting agent loop")

	for i := 0; i < maxIterations; i++ {
		log.Debug().
			Int("iteration", i+1).
			Int("messages_count", len(messages)).
			Msg("Processing agent iteration")

		// Call LLM with tools
		llmResponse, err := s.sdk.LLM().GenerateWithTools(ctx, llm.GenerateWithToolsRequest{
			SystemPrompt: systemPrompt,
			Messages:     messages,
			Temperature:  0.7,
			MaxTokens:    800, // Reduced for more concise responses
			Tools:        tools,
		})
		if err != nil {
			log.Error().
				Err(err).
				Int("iteration", i+1).
				Msg("Failed to generate response")
			return nil, fmt.Errorf("failed to generate AI response: %w", err)
		}

		totalTokens += llmResponse.Usage.TotalTokens

		log.Debug().
			Int("iteration", i+1).
			Int("prompt_tokens", llmResponse.Usage.PromptTokens).
			Int("completion_tokens", llmResponse.Usage.CompletionTokens).
			Int("total_tokens", llmResponse.Usage.TotalTokens).
			Int("cumulative_tokens", totalTokens).
			Int("tool_uses", len(llmResponse.ToolUses)).
			Msg("Received LLM response")

		// If no tool uses, we have a final response
		if len(llmResponse.ToolUses) == 0 {
			// Check if response contains a Score spec
			generatedSpec := extractYAMLSpec(llmResponse.Text)

			log.Debug().
				Int("iterations", i+1).
				Int("total_tokens", totalTokens).
				Bool("has_spec", generatedSpec != "").
				Int("citations_count", len(citations)).
				Msg("Agent loop completed")

			return &ChatResponse{
				Message:       llmResponse.Text,
				GeneratedSpec: generatedSpec,
				Citations:     citations,
				TokensUsed:    totalTokens,
				Timestamp:     time.Now(),
			}, nil
		}

		// Add assistant's response with tool uses to conversation
		assistantContent := []llm.ContentBlock{}
		if llmResponse.Text != "" {
			assistantContent = append(assistantContent, llm.ContentBlock{
				Type: "text",
				Text: llmResponse.Text,
			})
		}
		for _, toolUse := range llmResponse.ToolUses {
			// Ensure input is at least an empty object, never nil
			input := toolUse.Input
			if input == nil {
				input = map[string]interface{}{}
			}

			assistantContent = append(assistantContent, llm.ContentBlock{
				Type:  "tool_use",
				ID:    toolUse.ID,
				Name:  toolUse.Name,
				Input: input,
			})
		}
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: assistantContent,
		})

		// Execute tools and add results
		userContent := []llm.ContentBlock{}
		for idx, toolUse := range llmResponse.ToolUses {
			// Create tool executor (using internal API base URL and auth token from request)
			apiBaseURL := "http://localhost:8081"
			authToken := req.AuthToken
			if authToken == "" {
				log.Warn().
					Str("tool", toolUse.Name).
					Msg("No auth token available for tool execution")
			}

			log.Debug().
				Int("iteration", i+1).
				Int("tool_index", idx+1).
				Int("total_tools", len(llmResponse.ToolUses)).
				Str("tool_name", toolUse.Name).
				Str("tool_id", toolUse.ID).
				Msg("Executing tool")

			executor := NewToolExecutor(apiBaseURL, authToken)
			result, err := executor.ExecuteTool(ctx, toolUse.Name, toolUse.Input)

			var resultContent string
			if err != nil {
				log.Error().
					Err(err).
					Str("tool_name", toolUse.Name).
					Str("tool_id", toolUse.ID).
					Msg("Failed to execute tool")
				resultContent = fmt.Sprintf("Error executing tool: %v", err)
			} else {
				log.Debug().
					Str("tool_name", toolUse.Name).
					Str("tool_id", toolUse.ID).
					Int("result_length", len(result)).
					Msg("Executed tool")
				resultContent = result
				// Append tool result for display (not used in this iteration but may be useful for debugging)
				_ = append(toolResults, fmt.Sprintf("Tool %s: %s", toolUse.Name, result))
			}

			userContent = append(userContent, llm.ContentBlock{
				Type:      "tool_result",
				ToolUseID: toolUse.ID,
				Content:   resultContent,
				IsError:   err != nil,
			})
		}

		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userContent,
		})
	}

	// If we hit max iterations, return what we have
	log.Warn().
		Int("max_iterations", maxIterations).
		Int("total_tokens", totalTokens).
		Msg("Reached maximum agent loop iterations")

	return &ChatResponse{
		Message:    "I've executed the requested actions, but the conversation exceeded the maximum number of iterations. Please try breaking your request into smaller parts.",
		Citations:  citations,
		TokensUsed: totalTokens,
		Timestamp:  time.Now(),
	}, nil
}

// GenerateSpec generates a Score specification from a description
func (s *Service) GenerateSpec(ctx context.Context, req GenerateSpecRequest) (*GenerateSpecResponse, error) {
	if !s.enabled {
		return nil, fmt.Errorf("AI service is not enabled")
	}

	log.Debug().
		Str("description", req.Description).
		Msg("Generating Score specification")

	// Build a specific prompt for spec generation
	prompt := fmt.Sprintf(`Generate a complete Score specification based on the following description:

%s

The Score spec should be valid YAML following the Score specification format. Include:
- metadata (name, annotations)
- containers with appropriate resource limits
- required resources (database, cache, etc.)
- service configuration

Respond with:
1. The complete YAML Score specification in a code block
2. A brief explanation of the key components

YAML Spec:`, req.Description)

	// Retrieve relevant examples from RAG
	log.Debug().
		Str("query", req.Description+" Score specification example").
		Msg("Retrieving RAG examples")

	ragResponse, err := s.sdk.RAG().Retrieve(ctx, rag.RetrieveRequest{
		Query:    req.Description + " Score specification example",
		TopK:     2,
		MinScore: 0.3,
	})
	if err != nil {
		log.Warn().
			Err(err).
			Msg("Failed to retrieve RAG examples")
		ragResponse = &rag.RetrieveResponse{Context: ""}
	} else {
		log.Debug().
			Int("results_count", len(ragResponse.Results)).
			Msg("Retrieved RAG examples")
	}

	// Generate spec using LLM
	llmResponse, err := s.sdk.LLM().GenerateWithContext(ctx, llm.GenerateRequest{
		SystemPrompt: buildSpecGenerationSystemPrompt(),
		UserPrompt:   prompt,
		Temperature:  0.3, // Lower temperature for more consistent output
		MaxTokens:    3000,
	}, ragResponse.Context)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to generate specification")
		return nil, fmt.Errorf("failed to generate spec: %w", err)
	}

	log.Debug().
		Int("prompt_tokens", llmResponse.Usage.PromptTokens).
		Int("completion_tokens", llmResponse.Usage.CompletionTokens).
		Int("total_tokens", llmResponse.Usage.TotalTokens).
		Msg("Received LLM response")

	// Extract YAML spec from response
	spec := extractYAMLSpec(llmResponse.Text)
	if spec == "" {
		log.Error().
			Msg("Failed to extract YAML specification")
		return nil, fmt.Errorf("failed to extract YAML spec from AI response")
	}

	log.Debug().
		Int("spec_length", len(spec)).
		Int("total_tokens", llmResponse.Usage.TotalTokens).
		Msg("Generated Score specification")

	// Extract explanation (text before or after the YAML block)
	explanation := extractExplanation(llmResponse.Text, spec)

	// Extract citations
	var citations []string
	for _, result := range ragResponse.Results {
		if source, ok := result.Document.Metadata["source"]; ok {
			citations = append(citations, source)
		}
	}

	return &GenerateSpecResponse{
		Spec:        spec,
		Explanation: explanation,
		Citations:   citations,
		TokensUsed:  llmResponse.Usage.TotalTokens,
	}, nil
}

// buildSystemPromptWithTools creates the system prompt for the AI assistant with tool awareness
func buildSystemPromptWithTools() string {
	return `You are an expert AI assistant for innominatus, a Score-based platform orchestration tool for Internal Developer Platforms (IDPs).

Your role is to help developers and platform engineers with:
- Understanding innominatus features and capabilities
- Creating Score specifications for their applications
- Configuring workflows and golden paths
- Troubleshooting deployment issues
- Following best practices for platform engineering
- Performing actions on the platform (list apps, deploy, delete, view workflows, etc.)

You have access to tools that allow you to interact with the innominatus platform:
- list_applications: View all deployed applications
- get_application: Get details of a specific application
- deploy_application: Deploy a Score specification
- delete_application: Remove an application
- list_workflows: View workflow executions
- get_workflow: Get workflow execution details
- list_resources: View platform resources
- get_dashboard_stats: Get platform statistics

When the user asks to perform an action (like "list my applications" or "deploy this spec"), use the appropriate tool.
When answering questions about the platform, use the provided context from the documentation.

**IMPORTANT - Working with previously generated specs:**
- When you see a "[Generated Score Specification]:" block in the conversation history, that is a spec you previously created
- If the user says "deploy that" or "deploy it" or "use that spec", they are referring to the most recent spec in the conversation
- Extract the YAML content from the conversation history and pass it to the deploy_application tool
- Do NOT ask the user to provide the spec again - use what you already generated

Guidelines:
- **IMPORTANT: Keep responses very brief and concise (2-3 sentences maximum)**
- **IMPORTANT: Answer concisely and stop after addressing the user's main question**
- **Avoid chaining more than 3-4 tools unless absolutely necessary**
- **If you need more context, ask the user rather than exploring further**
- Use bullet points instead of long paragraphs
- When using tools, just present the results - don't over-explain
- Only provide detailed explanations when explicitly asked
- Be direct and to the point
- Use tools when the user wants to perform actions or view current state
- Use documentation context only when necessary
- If generating Score specs, ensure they are valid YAML

If you don't know something or the context doesn't contain the information, say so clearly.`
}

// buildSpecGenerationSystemPrompt creates a specialized prompt for spec generation
func buildSpecGenerationSystemPrompt() string {
	return `You are an expert at generating Score specifications for applications.

Score is a platform-agnostic workload specification that describes how to run a workload.

When generating a Score spec:
- Follow the official Score specification format
- Include all required fields (apiVersion, metadata, containers)
- Add appropriate resource requests/limits
- Configure necessary dependencies (database, cache, etc.)
- Use sensible defaults based on the application type
- Include relevant annotations and labels
- Ensure the YAML is valid and properly formatted

Always respond with:
1. A complete, valid YAML Score specification in a code block
2. A brief explanation of the key components and why they were chosen`
}

// extractYAMLSpec extracts YAML code blocks from AI response
func extractYAMLSpec(text string) string {
	// Look for ```yaml or ```yml code blocks
	patterns := []string{"```yaml", "```yml", "```YAML", "```YML"}

	for _, pattern := range patterns {
		start := strings.Index(text, pattern)
		if start == -1 {
			continue
		}

		// Find the end of the code block
		start += len(pattern)
		end := strings.Index(text[start:], "```")
		if end == -1 {
			continue
		}

		// Extract and clean the YAML
		yaml := strings.TrimSpace(text[start : start+end])
		return yaml
	}

	return ""
}

// extractExplanation extracts the explanation text from the AI response
func extractExplanation(fullText, spec string) string {
	// Remove the YAML spec from the full text to get the explanation
	explanation := strings.ReplaceAll(fullText, "```yaml\n"+spec+"\n```", "")
	explanation = strings.ReplaceAll(explanation, "```yml\n"+spec+"\n```", "")
	explanation = strings.TrimSpace(explanation)

	// If explanation is too short, return a default
	if len(explanation) < 20 {
		return "Generated Score specification based on your requirements."
	}

	return explanation
}
