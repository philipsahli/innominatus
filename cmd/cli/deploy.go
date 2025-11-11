package main

import (
	"context"
	"fmt"
	clientpkg "innominatus/internal/client"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	watch        bool
	watchVerbose bool
	watchAll     bool
	timeout      time.Duration
)

var deployCmd = &cobra.Command{
	Use:   "deploy <score-file.yaml>",
	Short: "Deploy a Score specification",
	Long: `Deploy a Score specification to the platform.

With the -w/--watch flag, this command will stream real-time deployment events
and show the progress of resource provisioning and workflow execution.

Examples:
  # Deploy a Score spec
  innominatus-ctl deploy myapp.yaml

  # Deploy with real-time watch
  innominatus-ctl deploy myapp.yaml -w

  # Deploy with verbose watch output
  innominatus-ctl deploy myapp.yaml -w --verbose

  # Deploy with custom timeout
  innominatus-ctl deploy myapp.yaml -w --timeout 10m
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specFile := args[0]

		// Read spec file
		specData, err := os.ReadFile(specFile)
		if err != nil {
			return fmt.Errorf("failed to read spec file: %w", err)
		}

		// Parse app name from spec
		appName, err := extractAppName(specData)
		if err != nil {
			return fmt.Errorf("failed to extract app name from spec: %w", err)
		}

		// Submit spec to server
		fmt.Printf("📤 Submitting Score specification: %s\n", appName)
		err = client.DeploySpec(specData)
		if err != nil {
			return fmt.Errorf("failed to deploy spec: %w", err)
		}

		if !watch {
			fmt.Printf("✅ Spec submitted successfully!\n")
			fmt.Printf("\nTo watch deployment progress, use:\n")
			fmt.Printf("  innominatus-ctl deploy %s -w\n", specFile)
			return nil
		}

		// Watch mode - stream real-time events
		fmt.Printf("✅ Spec submitted, starting watch mode...\n")
		return watchDeployment(appName)
	},
}

func init() {
	deployCmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch deployment progress in real-time")
	deployCmd.Flags().BoolVar(&watchVerbose, "verbose", false, "Show verbose event details")
	deployCmd.Flags().BoolVar(&watchAll, "all", false, "Show all events (including internal)")
	deployCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "Deployment timeout")
	rootCmd.AddCommand(deployCmd)
}

func watchDeployment(appName string) error {
	// Create SSE client
	sseClient := clientpkg.NewSSEClient(serverURL, clientpkg.GetAPIKey())

	// Create formatter
	formatter := clientpkg.NewWatchFormatter(watchVerbose, watchAll)

	// Print header
	formatter.PrintHeader(appName)

	// Track deployment state
	startTime := time.Now()
	deploymentComplete := false
	deploymentFailed := false

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Event handler
	eventHandler := func(event clientpkg.Event) error {
		// Format and print event
		output := formatter.FormatEvent(event)
		if output != "" {
			fmt.Println(output)
		}

		// Check for completion
		switch event.Type {
		case "deployment.completed":
			deploymentComplete = true
			return fmt.Errorf("deployment completed") // Signal to stop streaming

		case "deployment.failed":
			deploymentFailed = true
			return fmt.Errorf("deployment failed") // Signal to stop streaming

		case "workflow.completed":
			// Check if this is the final workflow
			// For now, we'll consider any workflow completion as potential completion
			// In a real implementation, you'd track all expected workflows

		case "workflow.failed", "resource.failed":
			deploymentFailed = true
			return fmt.Errorf("deployment failed") // Signal to stop streaming
		}

		return nil
	}

	// Stream events
	err := sseClient.StreamEvents(ctx, appName, eventHandler)

	// Calculate duration
	duration := time.Since(startTime)

	// Print footer
	if deploymentComplete {
		formatter.PrintFooter(true, duration)
		return nil
	}

	if deploymentFailed {
		formatter.PrintFooter(false, duration)
		return fmt.Errorf("deployment failed")
	}

	if err != nil && err.Error() == "deployment completed" {
		formatter.PrintFooter(true, duration)
		return nil
	}

	if err != nil && err.Error() == "deployment failed" {
		formatter.PrintFooter(false, duration)
		return fmt.Errorf("deployment failed")
	}

	if ctx.Err() == context.DeadlineExceeded {
		formatter.PrintFooter(false, duration)
		return fmt.Errorf("deployment timeout after %v", timeout)
	}

	if err != nil {
		formatter.PrintFooter(false, duration)
		return fmt.Errorf("watch error: %w", err)
	}

	formatter.PrintFooter(true, duration)
	return nil
}

func extractAppName(specData []byte) (string, error) {
	// Parse the spec data as YAML to extract app name
	var spec struct {
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}

	if err := yaml.Unmarshal(specData, &spec); err != nil {
		return "", fmt.Errorf("failed to parse spec YAML: %w", err)
	}

	if spec.Metadata.Name == "" {
		return "", fmt.Errorf("spec metadata.name is required")
	}

	return spec.Metadata.Name, nil
}
