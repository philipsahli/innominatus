package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"innominatus/internal/types"
	"strings"

	"gopkg.in/yaml.v3"
)

// WorkflowTier represents the three tiers of workflow definitions
type WorkflowTier string

const (
	TierPlatform    WorkflowTier = "platform"
	TierProduct     WorkflowTier = "product"
	TierApplication WorkflowTier = "application"
)

// WorkflowPhase represents when a workflow should execute
type WorkflowPhase string

const (
	PhasePreDeployment  WorkflowPhase = "pre-deployment"
	PhaseDeployment     WorkflowPhase = "deployment"
	PhasePostDeployment WorkflowPhase = "post-deployment"
)

// WorkflowTrigger defines what triggers a workflow
type WorkflowTrigger string

const (
	TriggerAllDeployments    WorkflowTrigger = "all_deployments"
	TriggerFirstDeployment   WorkflowTrigger = "first_deployment"
	TriggerProductDeployment WorkflowTrigger = "product_deployment"
	TriggerManual            WorkflowTrigger = "manual"
)

// PlatformWorkflow represents a platform-level workflow
type PlatformWorkflow struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string        `yaml:"name"`
		Description string        `yaml:"description"`
		Owner       string        `yaml:"owner"`
		Phase       WorkflowPhase `yaml:"phase"`
	} `yaml:"metadata"`
	Spec struct {
		Triggers []WorkflowTrigger `yaml:"triggers"`
		Steps    []types.Step      `yaml:"steps"`
	} `yaml:"spec"`
}

// ProductWorkflow represents a product-specific workflow
type ProductWorkflow struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name        string        `yaml:"name"`
		Description string        `yaml:"description"`
		Product     string        `yaml:"product"`
		Owner       string        `yaml:"owner"`
		Phase       WorkflowPhase `yaml:"phase"`
	} `yaml:"metadata"`
	Spec struct {
		Triggers []WorkflowTrigger `yaml:"triggers"`
		Steps    []types.Step      `yaml:"steps"`
	} `yaml:"spec"`
}

// WorkflowResolver handles resolution and merging of multi-tier workflows
type WorkflowResolver struct {
	workflowsRoot string
	policies      WorkflowPolicies
}

// WorkflowPolicies defines organization-wide workflow policies
type WorkflowPolicies struct {
	RequiredPlatformWorkflows []string `yaml:"requiredPlatformWorkflows"`
	AllowedProductWorkflows   []string `yaml:"allowedProductWorkflows"`
	WorkflowOverrides         struct {
		Platform bool `yaml:"platform"` // Can platform workflows override product workflows
		Product  bool `yaml:"product"`  // Can product workflows override application workflows
	} `yaml:"workflowOverrides"`
	MaxWorkflowDuration string `yaml:"maxWorkflowDuration"`
}

// ResolvedWorkflow represents the final merged workflow for execution
type ResolvedWorkflow struct {
	Name        string
	Description string
	Phase       WorkflowPhase
	Steps       []types.Step
	Sources     map[WorkflowTier][]string // Track which workflows contributed
}

// NewWorkflowResolver creates a new workflow resolver
func NewWorkflowResolver(workflowsRoot string, policies WorkflowPolicies) *WorkflowResolver {
	return &WorkflowResolver{
		workflowsRoot: workflowsRoot,
		policies:      policies,
	}
}

// NewWorkflowResolverFromAdminConfig creates a resolver from admin configuration
func NewWorkflowResolverFromAdminConfig(adminConfig interface{}) *WorkflowResolver {
	// Default policies if admin config is not available
	policies := WorkflowPolicies{
		RequiredPlatformWorkflows: []string{"security-scan", "cost-monitoring"},
		AllowedProductWorkflows:   []string{},
		WorkflowOverrides: struct {
			Platform bool `yaml:"platform"`
			Product  bool `yaml:"product"`
		}{
			Platform: true,
			Product:  true,
		},
		MaxWorkflowDuration: "30m",
	}

	workflowsRoot := "./workflows"

	// For now, just use default settings as admin config interface is complex
	// In a real implementation, this would parse the admin config properly

	return &WorkflowResolver{
		workflowsRoot: workflowsRoot,
		policies:      policies,
	}
}

// ApplicationInstance represents an application deployment instance
type ApplicationInstance struct {
	ID            int64
	Name          string
	Configuration map[string]interface{}
	Resources     []ResourceRef
}

// ResourceRef represents a reference to a resource in an application
type ResourceRef struct {
	ResourceName string
	ResourceType string
}

// ResolveWorkflows resolves and merges workflows for an application deployment
func (r *WorkflowResolver) ResolveWorkflows(app *ApplicationInstance) (map[WorkflowPhase][]ResolvedWorkflow, error) {
	resolved := make(map[WorkflowPhase][]ResolvedWorkflow)

	// Extract product from application metadata
	product := r.extractProductFromApp(app)

	// Load platform workflows
	platformWorkflows, err := r.loadPlatformWorkflows()
	if err != nil {
		return nil, fmt.Errorf("failed to load platform workflows: %w", err)
	}

	// Load product workflows
	productWorkflows, err := r.loadProductWorkflows(product)
	if err != nil {
		return nil, fmt.Errorf("failed to load product workflows for %s: %w", product, err)
	}

	// Generate application workflows from Score spec
	appWorkflows := r.generateApplicationWorkflows(app)

	// Merge workflows by phase
	for _, phase := range []WorkflowPhase{PhasePreDeployment, PhaseDeployment, PhasePostDeployment} {
		phaseWorkflows := []ResolvedWorkflow{}

		// Add platform workflows for this phase
		for _, pw := range platformWorkflows {
			if pw.Metadata.Phase == phase && r.shouldTriggerWorkflow(pw.Spec.Triggers, app) {
				resolved := ResolvedWorkflow{
					Name:        fmt.Sprintf("platform-%s", pw.Metadata.Name),
					Description: pw.Metadata.Description,
					Phase:       phase,
					Steps:       pw.Spec.Steps,
					Sources: map[WorkflowTier][]string{
						TierPlatform: {pw.Metadata.Name},
					},
				}
				phaseWorkflows = append(phaseWorkflows, resolved)
			}
		}

		// Add product workflows for this phase
		for _, pw := range productWorkflows {
			if pw.Metadata.Phase == phase && r.shouldTriggerWorkflow(pw.Spec.Triggers, app) {
				resolved := ResolvedWorkflow{
					Name:        fmt.Sprintf("product-%s-%s", product, pw.Metadata.Name),
					Description: pw.Metadata.Description,
					Phase:       phase,
					Steps:       pw.Spec.Steps,
					Sources: map[WorkflowTier][]string{
						TierProduct: {pw.Metadata.Name},
					},
				}
				phaseWorkflows = append(phaseWorkflows, resolved)
			}
		}

		// Add application workflows for this phase
		for _, aw := range appWorkflows {
			if aw.Phase == phase {
				phaseWorkflows = append(phaseWorkflows, aw)
			}
		}

		if len(phaseWorkflows) > 0 {
			resolved[phase] = phaseWorkflows
		}
	}

	return resolved, nil
}

// loadPlatformWorkflows loads all platform-level workflows
func (r *WorkflowResolver) loadPlatformWorkflows() ([]PlatformWorkflow, error) {
	platformDir := filepath.Join(r.workflowsRoot, "platform")
	if _, err := os.Stat(platformDir); os.IsNotExist(err) {
		return []PlatformWorkflow{}, nil
	}

	var workflows []PlatformWorkflow
	files, err := os.ReadDir(platformDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(platformDir, file.Name())
		// Note: filePath constructed from controlled platformDir and validated file listing
		data, err := os.ReadFile(filepath.Clean(filePath)) // #nosec G304 - path from controlled workflow directory
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		var workflow PlatformWorkflow
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// loadProductWorkflows loads product-specific workflows
func (r *WorkflowResolver) loadProductWorkflows(product string) ([]ProductWorkflow, error) {
	productDir := filepath.Join(r.workflowsRoot, "products", product)
	if _, err := os.Stat(productDir); os.IsNotExist(err) {
		return []ProductWorkflow{}, nil
	}

	var workflows []ProductWorkflow
	files, err := os.ReadDir(productDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(productDir, file.Name())
		// Note: filePath constructed from controlled productDir and validated file listing
		data, err := os.ReadFile(filepath.Clean(filePath)) // #nosec G304 - path from controlled workflow directory
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", filePath, err)
		}

		var workflow ProductWorkflow
		if err := yaml.Unmarshal(data, &workflow); err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		workflows = append(workflows, workflow)
	}

	return workflows, nil
}

// generateApplicationWorkflows generates workflows from Score specification
func (r *WorkflowResolver) generateApplicationWorkflows(app *ApplicationInstance) []ResolvedWorkflow {
	var workflows []ResolvedWorkflow

	// Generate deployment workflow from Score spec
	deploymentSteps := []types.Step{}

	// Add resource provisioning steps
	for _, resource := range app.Resources {
		step := types.Step{
			Name: fmt.Sprintf("provision-%s", resource.ResourceName),
			Type: "resource-provisioning",
		}
		deploymentSteps = append(deploymentSteps, step)
	}

	// Add container deployment step
	deploymentSteps = append(deploymentSteps, types.Step{
		Name:      "deploy-application",
		Type:      "kubernetes",
		Namespace: strings.ToLower(app.Name),
	})

	deploymentWorkflow := ResolvedWorkflow{
		Name:        fmt.Sprintf("app-deployment-%s", app.Name),
		Description: fmt.Sprintf("Application deployment workflow for %s", app.Name),
		Phase:       PhaseDeployment,
		Steps:       deploymentSteps,
		Sources: map[WorkflowTier][]string{
			TierApplication: {"generated-from-score-spec"},
		},
	}

	workflows = append(workflows, deploymentWorkflow)
	return workflows
}

// shouldTriggerWorkflow determines if a workflow should be triggered for the given application
func (r *WorkflowResolver) shouldTriggerWorkflow(triggers []WorkflowTrigger, app *ApplicationInstance) bool {
	for _, trigger := range triggers {
		switch trigger {
		case TriggerAllDeployments:
			return true
		case TriggerFirstDeployment:
			// Check if this is the first deployment (implementation depends on deployment tracking)
			return app.ID == 1 // Simplified logic
		case TriggerProductDeployment:
			// This would be product-specific logic
			return true
		case TriggerManual:
			return false // Manual workflows are not auto-triggered
		}
	}
	return false
}

// extractProductFromApp extracts the product name from application metadata
func (r *WorkflowResolver) extractProductFromApp(app *ApplicationInstance) string {
	// Check for product in metadata
	if metadata, ok := app.Configuration["metadata"].(map[string]interface{}); ok {
		if product, ok := metadata["product"].(string); ok {
			return product
		}
	}

	// Fallback: extract from application name (e.g., "ecommerce-web" -> "ecommerce")
	parts := strings.Split(app.Name, "-")
	if len(parts) > 0 {
		return parts[0]
	}

	return "default"
}

// ValidateWorkflowPolicies validates workflows against organization policies
func (r *WorkflowResolver) ValidateWorkflowPolicies(resolved map[WorkflowPhase][]ResolvedWorkflow) error {
	// Check required platform workflows
	for _, required := range r.policies.RequiredPlatformWorkflows {
		found := false
		for _, phaseWorkflows := range resolved {
			for _, workflow := range phaseWorkflows {
				if sources, ok := workflow.Sources[TierPlatform]; ok {
					for _, source := range sources {
						if source == required {
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return fmt.Errorf("required platform workflow %s not found", required)
		}
	}

	return nil
}

// GetWorkflowSummary provides a summary of resolved workflows
func (r *WorkflowResolver) GetWorkflowSummary(resolved map[WorkflowPhase][]ResolvedWorkflow) map[string]interface{} {
	summary := map[string]interface{}{
		"total_workflows": 0,
		"by_phase":        map[string]int{},
		"by_tier":         map[string]int{},
		"phases":          []string{},
	}

	totalWorkflows := 0
	byTier := map[string]int{
		"platform":    0,
		"product":     0,
		"application": 0,
	}

	for phase, workflows := range resolved {
		summary["by_phase"].(map[string]int)[string(phase)] = len(workflows)
		summary["phases"] = append(summary["phases"].([]string), string(phase))
		totalWorkflows += len(workflows)

		for _, workflow := range workflows {
			for tier := range workflow.Sources {
				byTier[string(tier)]++
			}
		}
	}

	summary["total_workflows"] = totalWorkflows
	summary["by_tier"] = byTier

	return summary
}