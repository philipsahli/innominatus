package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"innominatus/internal/ai"

	"github.com/chzyer/readline"
)

// ChatCommand implements interactive chat with the AI assistant
func ChatCommand() error {
	fmt.Println("================================================================================")
	fmt.Println("🤖 innominatus AI Assistant - Interactive Chat")
	fmt.Println("================================================================================")
	fmt.Println()

	// Initialize AI service
	ctx := context.Background()
	aiService, err := ai.NewServiceFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AI service: %w", err)
	}

	if !aiService.IsEnabled() {
		return fmt.Errorf("AI service is not enabled. Please set OPENAI_API_KEY and ANTHROPIC_API_KEY environment variables")
	}

	// Get status
	status := aiService.GetStatus(ctx)
	fmt.Printf("✅ %s\n", status.Message)
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  /help       - Show this help message")
	fmt.Println("  /clear      - Clear the screen")
	fmt.Println("  /save FILE  - Save last generated spec to a file")
	fmt.Println("  /exit, /quit, Ctrl+C - Exit the chat")
	fmt.Println()
	fmt.Println("Type your question or ask me to generate a Score specification...")
	fmt.Println("================================================================================")
	fmt.Println()

	// Setup readline
	rl, err := readline.New("💬 You: ")
	if err != nil {
		return fmt.Errorf("failed to create readline: %w", err)
	}
	defer func() {
		if err := rl.Close(); err != nil {
			fmt.Printf("Error closing readline: %v\n", err)
		}
	}()

	var lastSpec string

	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF, Ctrl+C, Ctrl+D
			fmt.Println("\nGoodbye! 👋")
			return nil
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		// Handle commands
		if strings.HasPrefix(input, "/") {
			parts := strings.Fields(input)
			cmd := parts[0]

			switch cmd {
			case "/exit", "/quit":
				fmt.Println("Goodbye! 👋")
				return nil

			case "/help":
				printHelp()
				continue

			case "/clear":
				print("\033[H\033[2J") // Clear screen
				continue

			case "/save":
				if len(parts) < 2 {
					fmt.Println("❌ Usage: /save <filename>")
					continue
				}
				if lastSpec == "" {
					fmt.Println("❌ No spec has been generated yet")
					continue
				}
				filename := parts[1]
				if err := os.WriteFile(filename, []byte(lastSpec), 0600); err != nil {
					fmt.Printf("❌ Failed to save spec: %v\n", err)
				} else {
					fmt.Printf("✅ Spec saved to %s\n", filename)
				}
				continue

			default:
				fmt.Printf("❌ Unknown command: %s (type /help for available commands)\n", cmd)
				continue
			}
		}

		// Check if user wants to generate a spec
		lowerInput := strings.ToLower(input)
		isSpecRequest := strings.Contains(lowerInput, "generate") ||
			strings.Contains(lowerInput, "create") && (strings.Contains(lowerInput, "spec") || strings.Contains(lowerInput, "score"))

		if isSpecRequest {
			// Generate spec
			fmt.Println("\n🔄 Generating specification...")
			specResp, err := aiService.GenerateSpec(ctx, ai.GenerateSpecRequest{
				Description: input,
			})
			if err != nil {
				fmt.Printf("\n❌ Error: %v\n\n", err)
				continue
			}

			fmt.Printf("\n🤖 AI Response:\n\n%s\n\n", specResp.Explanation)
			fmt.Println("📄 Generated Score Specification:")
			fmt.Println("```yaml")
			fmt.Println(specResp.Spec)
			fmt.Println("```")
			fmt.Println()

			if len(specResp.Citations) > 0 {
				fmt.Println("📚 Sources used:")
				for _, citation := range specResp.Citations {
					fmt.Printf("   • %s\n", citation)
				}
				fmt.Println()
			}

			fmt.Printf("💡 Tip: Use '/save spec.yaml' to save this specification\n\n")

			lastSpec = specResp.Spec
		} else {
			// Regular chat
			fmt.Println()
			chatResp, err := aiService.Chat(ctx, ai.ChatRequest{
				Message: input,
			})
			if err != nil {
				fmt.Printf("❌ Error: %v\n\n", err)
				continue
			}

			fmt.Printf("🤖 AI: %s\n\n", chatResp.Message)

			if chatResp.GeneratedSpec != "" {
				fmt.Println("📄 Generated Score Specification:")
				fmt.Println("```yaml")
				fmt.Println(chatResp.GeneratedSpec)
				fmt.Println("```")
				fmt.Println()
				fmt.Printf("💡 Tip: Use '/save spec.yaml' to save this specification\n\n")
				lastSpec = chatResp.GeneratedSpec
			}

			if len(chatResp.Citations) > 0 {
				fmt.Println("📚 Sources:")
				for _, citation := range chatResp.Citations {
					fmt.Printf("   • %s\n", citation)
				}
				fmt.Println()
			}
		}
	}
}

// OneShotCommand executes a single question and exits
func OneShotCommand(question string) error {
	ctx := context.Background()

	// Initialize AI service
	aiService, err := ai.NewServiceFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AI service: %w", err)
	}

	if !aiService.IsEnabled() {
		return fmt.Errorf("AI service is not enabled. Set OPENAI_API_KEY and ANTHROPIC_API_KEY")
	}

	// Send question
	resp, err := aiService.Chat(ctx, ai.ChatRequest{
		Message: question,
	})
	if err != nil {
		return fmt.Errorf("failed to get AI response: %w", err)
	}

	// Print response
	fmt.Println(resp.Message)

	if resp.GeneratedSpec != "" {
		fmt.Println("\nGenerated Score Specification:")
		fmt.Println("```yaml")
		fmt.Println(resp.GeneratedSpec)
		fmt.Println("```")
	}

	return nil
}

// GenerateSpecCommand generates a spec and saves it to a file
func GenerateSpecCommand(description string, outputFile string) error {
	ctx := context.Background()

	// Initialize AI service
	aiService, err := ai.NewServiceFromEnv(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize AI service: %w", err)
	}

	if !aiService.IsEnabled() {
		return fmt.Errorf("AI service is not enabled. Set OPENAI_API_KEY and ANTHROPIC_API_KEY")
	}

	// Generate spec
	fmt.Printf("🔄 Generating specification for: %s\n", description)
	resp, err := aiService.GenerateSpec(ctx, ai.GenerateSpecRequest{
		Description: description,
	})
	if err != nil {
		return fmt.Errorf("failed to generate spec: %w", err)
	}

	// Save to file
	if err := os.WriteFile(outputFile, []byte(resp.Spec), 0600); err != nil {
		return fmt.Errorf("failed to save spec: %w", err)
	}

	fmt.Printf("✅ Score specification saved to: %s\n", outputFile)
	fmt.Printf("\n📝 Explanation:\n%s\n", resp.Explanation)

	if len(resp.Citations) > 0 {
		fmt.Println("\n📚 Sources used:")
		for _, citation := range resp.Citations {
			fmt.Printf("   • %s\n", citation)
		}
	}

	return nil
}

// printHelp prints the help message
func printHelp() {
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println("🤖 innominatus AI Assistant - Help")
	fmt.Println("================================================================================")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  /help          - Show this help message")
	fmt.Println("  /clear         - Clear the screen")
	fmt.Println("  /save FILE     - Save last generated spec to a file")
	fmt.Println("  /exit, /quit   - Exit the chat")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  • Ask questions about innominatus features")
	fmt.Println("  • Request Score spec generation (e.g., 'Generate a spec for a Node.js app')")
	fmt.Println("  • Get help with workflows, golden paths, and configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  💬 'How do I deploy a microservice with PostgreSQL?'")
	fmt.Println("  💬 'Generate a Score spec for a Python FastAPI app with Redis'")
	fmt.Println("  💬 'What golden paths are available?'")
	fmt.Println("  💬 'How do I configure OIDC authentication?'")
	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Println()
}
