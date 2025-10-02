package main

import (
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/workflow"
)

// Demonstration of the new three-tier workflow architecture
func main() {
	fmt.Println("🏗️  Three-Tier Workflow Architecture Demo")
	fmt.Println("========================================")

	// Load admin configuration with workflow policies
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		fmt.Printf("Failed to load admin config: %v\n", err)
		return
	}

	fmt.Println("\n📋 Admin Configuration Loaded:")
	fmt.Printf("  Workflows Root: %s\n", adminConfig.WorkflowPolicies.WorkflowsRoot)
	fmt.Printf("  Required Platform Workflows: %v\n", adminConfig.WorkflowPolicies.RequiredPlatformWorkflows)
	fmt.Printf("  Allowed Product Workflows: %v\n", adminConfig.WorkflowPolicies.AllowedProductWorkflows)

	// Create workflow resolver with admin configuration
	resolver := workflow.NewWorkflowResolverFromAdminConfig(adminConfig)

	// Create a sample ecommerce application instance
	app := &workflow.ApplicationInstance{
		ID:   1,
		Name: "ecommerce-web",
		Configuration: map[string]interface{}{
			"metadata": map[string]interface{}{
				"product":    "ecommerce",
				"team":       "ecommerce-frontend-team",
				"costCenter": "engineering",
			},
			"containers": map[string]interface{}{
				"web": map[string]interface{}{
					"image": "nginx:latest",
				},
			},
		},
		Resources: []workflow.ResourceRef{
			{ResourceName: "database", ResourceType: "postgres"},
			{ResourceName: "cache", ResourceType: "redis"},
			{ResourceName: "secrets", ResourceType: "vault-space"},
		},
	}

	fmt.Println("\n🚀 Application Instance:")
	fmt.Printf("  Name: %s\n", app.Name)
	fmt.Printf("  Product: %s\n", app.Configuration["metadata"].(map[string]interface{})["product"])
	fmt.Printf("  Resources: %d\n", len(app.Resources))

	// Resolve workflows for the application
	fmt.Println("\n🔄 Resolving Multi-Tier Workflows...")
	resolvedWorkflows, err := resolver.ResolveWorkflows(app)
	if err != nil {
		fmt.Printf("Failed to resolve workflows: %v\n", err)
		return
	}

	// Display workflow resolution results
	fmt.Println("\n📊 Workflow Resolution Results:")
	for phase, workflows := range resolvedWorkflows {
		fmt.Printf("\n  %s Phase (%d workflows):\n", phase, len(workflows))
		for _, workflow := range workflows {
			fmt.Printf("    🔧 %s (%d steps)\n", workflow.Name, len(workflow.Steps))
			fmt.Printf("       Source Tiers: %v\n", workflow.Sources)
			for i, step := range workflow.Steps {
				fmt.Printf("       Step %d: %s (%s)\n", i+1, step.Name, step.Type)
			}
		}
	}

	// Validate workflows against policies
	fmt.Println("\n✅ Validating Workflow Policies...")
	if err := resolver.ValidateWorkflowPolicies(resolvedWorkflows); err != nil {
		fmt.Printf("❌ Policy validation failed: %v\n", err)
		return
	}
	fmt.Println("✅ All workflow policies validated successfully!")

	// Get workflow summary
	summary := resolver.GetWorkflowSummary(resolvedWorkflows)
	fmt.Println("\n📈 Workflow Summary:")
	fmt.Printf("  Total Workflows: %v\n", summary["total_workflows"])
	fmt.Printf("  Phases: %v\n", summary["phases"])
	fmt.Printf("  By Tier: %v\n", summary["by_tier"])

	fmt.Println("\n🎉 Three-Tier Workflow Resolution Complete!")
	fmt.Println("\nKey Benefits Demonstrated:")
	fmt.Println("  ✅ Platform team controls organization-wide workflows")
	fmt.Println("  ✅ Infrastructure teams define product-specific workflows")
	fmt.Println("  ✅ Application teams focus on Score specifications")
	fmt.Println("  ✅ Automatic workflow resolution and policy enforcement")
	fmt.Println("  ✅ Clear separation of concerns across teams")
}
