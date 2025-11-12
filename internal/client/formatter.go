package client

import (
	"fmt"
	"strings"
	"time"
)

// WatchFormatter formats events for CLI watch output
type WatchFormatter struct {
	verbose bool
	showAll bool // Show all events vs just important ones
}

// NewWatchFormatter creates a new watch formatter
func NewWatchFormatter(verbose, showAll bool) *WatchFormatter {
	return &WatchFormatter{
		verbose: verbose,
		showAll: showAll,
	}
}

// FormatEvent formats an event for display
func (f *WatchFormatter) FormatEvent(event Event) string {
	// Skip certain events if not in show-all mode
	if !f.showAll {
		switch event.Type {
		case "connected", "broadcast":
			return "" // Skip connection events
		}
	}

	timestamp := event.Timestamp.Format("15:04:05")
	icon := f.getIcon(event.Type)

	var message string
	switch event.Type {
	case "spec.created":
		message = fmt.Sprintf("Score specification created for %s", event.AppName)

	case "spec.validated":
		message = "Score specification validated"

	case "resource.created":
		resourceName := f.getString(event.Data, "resource_name")
		resourceType := f.getString(event.Data, "resource_type")
		message = fmt.Sprintf("Resource created: %s (%s)", resourceName, resourceType)

	case "resource.requested":
		resourceName := f.getString(event.Data, "resource_name")
		message = fmt.Sprintf("Resource requested: %s", resourceName)

	case "resource.provisioning":
		resourceName := f.getString(event.Data, "resource_name")
		resourceType := f.getString(event.Data, "resource_type")
		message = fmt.Sprintf("Provisioning resource: %s (%s)", resourceName, resourceType)

	case "resource.active":
		resourceName := f.getString(event.Data, "resource_name")
		message = fmt.Sprintf("Resource active: %s", resourceName)

	case "resource.failed":
		resourceName := f.getString(event.Data, "resource_name")
		errorMsg := f.getString(event.Data, "error")
		message = fmt.Sprintf("Resource failed: %s - %s", resourceName, errorMsg)

	case "provider.resolved":
		providerName := f.getString(event.Data, "provider_name")
		resourceType := f.getString(event.Data, "resource_type")
		workflowName := f.getString(event.Data, "workflow_name")
		message = fmt.Sprintf("Provider resolved: %s for %s (workflow: %s)", providerName, resourceType, workflowName)

	case "workflow.created":
		workflowName := f.getString(event.Data, "workflow_name")
		message = fmt.Sprintf("Workflow created: %s", workflowName)

	case "workflow.started":
		workflowName := f.getString(event.Data, "workflow_name")
		totalSteps := f.getInt(event.Data, "total_steps")
		message = fmt.Sprintf("Workflow started: %s (%d steps)", workflowName, totalSteps)

	case "workflow.completed":
		workflowName := f.getString(event.Data, "workflow_name")
		message = fmt.Sprintf("Workflow completed: %s", workflowName)

	case "workflow.failed":
		workflowName := f.getString(event.Data, "workflow_name")
		errorMsg := f.getString(event.Data, "error")
		message = fmt.Sprintf("Workflow failed: %s - %s", workflowName, errorMsg)

	case "step.started":
		stepName := f.getString(event.Data, "step_name")
		message = fmt.Sprintf("Step started: %s", stepName)

	case "step.completed":
		stepName := f.getString(event.Data, "step_name")
		message = fmt.Sprintf("Step completed: %s", stepName)

	case "step.failed":
		stepName := f.getString(event.Data, "step_name")
		errorMsg := f.getString(event.Data, "error")
		message = fmt.Sprintf("Step failed: %s - %s", stepName, errorMsg)

	case "step.progress":
		stepName := f.getString(event.Data, "step_name")
		progress := f.getString(event.Data, "progress")
		message = fmt.Sprintf("Step progress: %s - %s", stepName, progress)

	case "deployment.started":
		message = fmt.Sprintf("Deployment started for %s", event.AppName)

	case "deployment.completed":
		message = "Deployment completed successfully!"

	case "deployment.failed":
		errorMsg := f.getString(event.Data, "error")
		message = fmt.Sprintf("Deployment failed: %s", errorMsg)

	default:
		if f.showAll {
			message = fmt.Sprintf("%s event", event.Type)
		} else {
			return "" // Skip unknown events
		}
	}

	// Format output
	output := fmt.Sprintf("[%s] %s %s", timestamp, icon, message)

	// Add verbose details if enabled
	if f.verbose && len(event.Data) > 0 {
		output += f.formatDetails(event.Data)
	}

	return output
}

// getIcon returns an icon for the event type
func (f *WatchFormatter) getIcon(eventType string) string {
	switch {
	case strings.HasSuffix(eventType, ".created"):
		return "📝"
	case strings.HasSuffix(eventType, ".validated"):
		return "✅"
	case strings.HasSuffix(eventType, ".started"):
		return "🚀"
	case strings.HasSuffix(eventType, ".completed"):
		return "✅"
	case strings.HasSuffix(eventType, ".failed"):
		return "❌"
	case strings.HasSuffix(eventType, ".active"):
		return "🟢"
	case strings.HasSuffix(eventType, ".provisioning"):
		return "⏳"
	case strings.HasSuffix(eventType, ".resolved"):
		return "🔍"
	case strings.HasSuffix(eventType, ".progress"):
		return "⏳"
	default:
		return "ℹ️"
	}
}

// getString safely gets a string from event data
func (f *WatchFormatter) getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", val)
	}
	return ""
}

// getInt safely gets an int from event data
func (f *WatchFormatter) getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

// formatDetails formats event data details
func (f *WatchFormatter) formatDetails(data map[string]interface{}) string {
	var details []string
	for k, v := range data {
		// Skip certain verbose fields
		if k == "resource_id" || k == "execution_id" || k == "workflow_execution_id" {
			continue
		}
		details = append(details, fmt.Sprintf("%s=%v", k, v))
	}
	if len(details) > 0 {
		return "\n    " + strings.Join(details, ", ")
	}
	return ""
}

// PrintHeader prints a watch session header
func (f *WatchFormatter) PrintHeader(appName string) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("🔍 Watching deployment: %s\n", appName)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// PrintFooter prints a watch session footer
func (f *WatchFormatter) PrintFooter(success bool, duration time.Duration) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if success {
		fmt.Printf("✅ Deployment completed in %v\n", duration.Round(time.Second))
	} else {
		fmt.Printf("❌ Deployment failed after %v\n", duration.Round(time.Second))
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}
