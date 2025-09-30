package workflow

import (
	"fmt"
	"innominatus/internal/types"
	"time"
)

// WorkflowAnalysis represents the analysis of a workflow specification
type WorkflowAnalysis struct {
	Spec           *types.ScoreSpec      `json:"spec"`
	Dependencies   []DependencyAnalysis  `json:"dependencies"`
	ExecutionPlan  ExecutionPlan         `json:"executionPlan"`
	EstimatedTime  time.Duration         `json:"estimatedTime"`
	ResourceGraph  ResourceGraph         `json:"resourceGraph"`
	Warnings       []string              `json:"warnings"`
	Recommendations []string             `json:"recommendations"`
	Summary        AnalysisSummary       `json:"summary"`
}

// DependencyAnalysis represents dependencies between workflow steps
type DependencyAnalysis struct {
	StepName     string   `json:"stepName"`
	StepType     string   `json:"stepType"`
	DependsOn    []string `json:"dependsOn"`
	Blocks       []string `json:"blocks"`
	CanRunInParallel bool `json:"canRunInParallel"`
	EstimatedDuration time.Duration `json:"estimatedDuration"`
	Phase        string   `json:"phase"`
}

// ExecutionPlan represents the planned execution order and parallelization
type ExecutionPlan struct {
	Phases    []ExecutionPhase `json:"phases"`
	TotalTime time.Duration    `json:"totalTime"`
	MaxParallel int            `json:"maxParallel"`
}

// ExecutionPhase represents a phase of execution with parallel groups
type ExecutionPhase struct {
	Name         string              `json:"name"`
	Order        int                 `json:"order"`
	ParallelGroups []ParallelGroup   `json:"parallelGroups"`
	EstimatedTime time.Duration      `json:"estimatedTime"`
}

// ParallelGroup represents steps that can run in parallel
type ParallelGroup struct {
	Steps         []StepExecution   `json:"steps"`
	EstimatedTime time.Duration     `json:"estimatedTime"`
}

// StepExecution represents a step in the execution plan
type StepExecution struct {
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Order         int           `json:"order"`
	EstimatedTime time.Duration `json:"estimatedTime"`
	Resources     []string      `json:"resources"`
	Status        string        `json:"status"`
}

// ResourceGraph represents the dependency graph of resources
type ResourceGraph struct {
	Nodes []ResourceNode `json:"nodes"`
	Edges []ResourceEdge `json:"edges"`
}

// ResourceNode represents a resource in the dependency graph
type ResourceNode struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Level    int                    `json:"level"`
	Metadata map[string]interface{} `json:"metadata"`
}

// ResourceEdge represents a dependency between resources
type ResourceEdge struct {
	From         string `json:"from"`
	To           string `json:"to"`
	DependencyType string `json:"dependencyType"`
}

// AnalysisSummary provides high-level summary of the analysis
type AnalysisSummary struct {
	TotalSteps       int           `json:"totalSteps"`
	TotalResources   int           `json:"totalResources"`
	ParallelSteps    int           `json:"parallelSteps"`
	CriticalPath     []string      `json:"criticalPath"`
	EstimatedTime    time.Duration `json:"estimatedTime"`
	ComplexityScore  int           `json:"complexityScore"`
	RiskLevel        string        `json:"riskLevel"`
}

// WorkflowAnalyzer analyzes workflow specifications and builds dependency graphs
type WorkflowAnalyzer struct {
	stepDurations map[string]time.Duration
	resourceTypes map[string]ResourceTypeInfo
}

// ResourceTypeInfo contains metadata about resource types
type ResourceTypeInfo struct {
	ProvisionTime  time.Duration
	Dependencies   []string
	Complexity     int
}

// NewWorkflowAnalyzer creates a new workflow analyzer
func NewWorkflowAnalyzer() *WorkflowAnalyzer {
	return &WorkflowAnalyzer{
		stepDurations: getDefaultStepDurations(),
		resourceTypes: getDefaultResourceTypes(),
	}
}

// AnalyzeSpec analyzes a Score specification and returns detailed workflow analysis
func (a *WorkflowAnalyzer) AnalyzeSpec(spec *types.ScoreSpec) (*WorkflowAnalysis, error) {
	analysis := &WorkflowAnalysis{
		Spec:            spec,
		Dependencies:    []DependencyAnalysis{},
		Warnings:        []string{},
		Recommendations: []string{},
	}

	// Analyze resources and their dependencies
	resourceGraph, err := a.analyzeResources(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze resources: %w", err)
	}
	analysis.ResourceGraph = resourceGraph

	// Analyze workflow dependencies
	dependencies, err := a.analyzeDependencies(spec, resourceGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze dependencies: %w", err)
	}
	analysis.Dependencies = dependencies

	// Create execution plan
	executionPlan, err := a.createExecutionPlan(dependencies)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution plan: %w", err)
	}
	analysis.ExecutionPlan = executionPlan
	analysis.EstimatedTime = executionPlan.TotalTime

	// Generate warnings and recommendations
	analysis.Warnings = a.generateWarnings(spec, dependencies)
	analysis.Recommendations = a.generateRecommendations(spec, dependencies, executionPlan)

	// Create summary
	analysis.Summary = a.createSummary(spec, dependencies, executionPlan)

	return analysis, nil
}

// analyzeResources analyzes the resources defined in the spec
func (a *WorkflowAnalyzer) analyzeResources(spec *types.ScoreSpec) (ResourceGraph, error) {
	graph := ResourceGraph{
		Nodes: []ResourceNode{},
		Edges: []ResourceEdge{},
	}

	level := 0
	nodeID := 0

	// Add application container as a node
	nodeID++
	appNode := ResourceNode{
		ID:   fmt.Sprintf("app-%d", nodeID),
		Name: spec.Metadata.Name,
		Type: "application",
		Level: level,
		Metadata: map[string]interface{}{
			"containers": len(spec.Containers),
		},
	}
	graph.Nodes = append(graph.Nodes, appNode)

	// Add resources as nodes
	for resourceName, resource := range spec.Resources {
		nodeID++
		level++
		resourceNode := ResourceNode{
			ID:   fmt.Sprintf("resource-%d", nodeID),
			Name: resourceName,
			Type: resource.Type,
			Level: level,
			Metadata: map[string]interface{}{
				"params": resource.Params,
			},
		}
		graph.Nodes = append(graph.Nodes, resourceNode)

		// Create edge from app to resource
		edge := ResourceEdge{
			From:           appNode.ID,
			To:             resourceNode.ID,
			DependencyType: "requires",
		}
		graph.Edges = append(graph.Edges, edge)
	}

	// Analyze resource dependencies (simplified logic)
	a.analyzeResourceDependencies(&graph)

	return graph, nil
}

// analyzeResourceDependencies analyzes dependencies between resources
func (a *WorkflowAnalyzer) analyzeResourceDependencies(graph *ResourceGraph) {
	// Simple dependency rules
	dependencyRules := map[string][]string{
		"postgres": {"volume"},
		"redis":    {"volume"},
		"route":    {"service"},
		"service":  {"deployment"},
	}

	for i, node := range graph.Nodes {
		if deps, exists := dependencyRules[node.Type]; exists {
			for j, depNode := range graph.Nodes {
				if i != j {
					for _, depType := range deps {
						if depNode.Type == depType {
							edge := ResourceEdge{
								From:           depNode.ID,
								To:             node.ID,
								DependencyType: "prerequisite",
							}
							graph.Edges = append(graph.Edges, edge)
							// Update levels based on dependencies
							if depNode.Level >= node.Level {
								graph.Nodes[i].Level = depNode.Level + 1
							}
						}
					}
				}
			}
		}
	}
}

// analyzeDependencies analyzes workflow step dependencies
func (a *WorkflowAnalyzer) analyzeDependencies(spec *types.ScoreSpec, resourceGraph ResourceGraph) ([]DependencyAnalysis, error) {
	var dependencies []DependencyAnalysis

	// If spec has workflows defined, analyze them
	if spec.Workflows != nil {
		for workflowName, workflow := range spec.Workflows {
			for i, step := range workflow.Steps {
				dep := DependencyAnalysis{
					StepName:          step.Name,
					StepType:          step.Type,
					DependsOn:         []string{},
					Blocks:            []string{},
					CanRunInParallel:  false,
					EstimatedDuration: a.getStepDuration(step.Type),
					Phase:             a.determinePhase(step.Type, workflowName),
				}

				// Analyze step dependencies
				dep.DependsOn = a.getStepDependencies(step, workflow.Steps[:i])
				dep.Blocks = a.getStepsBlocked(step, workflow.Steps[i+1:])
				dep.CanRunInParallel = len(dep.DependsOn) == 0

				dependencies = append(dependencies, dep)
			}
		}
	} else {
		// Generate implied workflow from resources
		dependencies = a.generateImpliedWorkflow(spec, resourceGraph)
	}

	return dependencies, nil
}

// generateImpliedWorkflow generates workflow dependencies from Score spec resources
func (a *WorkflowAnalyzer) generateImpliedWorkflow(spec *types.ScoreSpec, resourceGraph ResourceGraph) []DependencyAnalysis {
	var dependencies []DependencyAnalysis

	// Pre-deployment phase
	dependencies = append(dependencies, DependencyAnalysis{
		StepName:          "validate-spec",
		StepType:          "validation",
		DependsOn:         []string{},
		Blocks:            []string{"provision-resources"},
		CanRunInParallel:  true,
		EstimatedDuration: 30 * time.Second,
		Phase:             "pre-deployment",
	})

	dependencies = append(dependencies, DependencyAnalysis{
		StepName:          "security-scan",
		StepType:          "security",
		DependsOn:         []string{},
		Blocks:            []string{"provision-resources"},
		CanRunInParallel:  true,
		EstimatedDuration: 2 * time.Minute,
		Phase:             "pre-deployment",
	})

	// Deployment phase - resource provisioning based on dependency levels
	levels := a.getResourceLevels(resourceGraph)
	for level := 0; level < len(levels); level++ {
		for _, node := range levels[level] {
			if node.Type != "application" {
				stepName := fmt.Sprintf("provision-%s", node.Name)
				var dependsOn []string

				// Find dependencies from previous levels
				for _, edge := range resourceGraph.Edges {
					if edge.To == node.ID {
						for _, depNode := range resourceGraph.Nodes {
							if depNode.ID == edge.From && depNode.Type != "application" {
								dependsOn = append(dependsOn, fmt.Sprintf("provision-%s", depNode.Name))
							}
						}
					}
				}

				dependencies = append(dependencies, DependencyAnalysis{
					StepName:          stepName,
					StepType:          "resource-provisioning",
					DependsOn:         dependsOn,
					Blocks:            []string{"deploy-application"},
					CanRunInParallel:  len(dependsOn) == 0,
					EstimatedDuration: a.getResourceProvisionTime(node.Type),
					Phase:             "deployment",
				})
			}
		}
	}

	// Application deployment
	var resourceDeps []string
	for _, node := range resourceGraph.Nodes {
		if node.Type != "application" {
			resourceDeps = append(resourceDeps, fmt.Sprintf("provision-%s", node.Name))
		}
	}

	dependencies = append(dependencies, DependencyAnalysis{
		StepName:          "deploy-application",
		StepType:          "kubernetes",
		DependsOn:         resourceDeps,
		Blocks:            []string{"setup-monitoring"},
		CanRunInParallel:  false,
		EstimatedDuration: 3 * time.Minute,
		Phase:             "deployment",
	})

	// Post-deployment phase
	dependencies = append(dependencies, DependencyAnalysis{
		StepName:          "setup-monitoring",
		StepType:          "monitoring",
		DependsOn:         []string{"deploy-application"},
		Blocks:            []string{},
		CanRunInParallel:  false,
		EstimatedDuration: 2 * time.Minute,
		Phase:             "post-deployment",
	})

	dependencies = append(dependencies, DependencyAnalysis{
		StepName:          "health-check",
		StepType:          "validation",
		DependsOn:         []string{"deploy-application"},
		Blocks:            []string{},
		CanRunInParallel:  true,
		EstimatedDuration: 1 * time.Minute,
		Phase:             "post-deployment",
	})

	return dependencies
}

// createExecutionPlan creates an optimized execution plan with parallelization
func (a *WorkflowAnalyzer) createExecutionPlan(dependencies []DependencyAnalysis) (ExecutionPlan, error) {
	plan := ExecutionPlan{
		Phases:      []ExecutionPhase{},
		MaxParallel: 0,
	}

	// Group by phase
	phaseGroups := make(map[string][]DependencyAnalysis)
	for _, dep := range dependencies {
		phaseGroups[dep.Phase] = append(phaseGroups[dep.Phase], dep)
	}

	// Create execution phases
	phaseOrder := []string{"pre-deployment", "deployment", "post-deployment"}
	totalTime := time.Duration(0)

	for i, phaseName := range phaseOrder {
		if steps, exists := phaseGroups[phaseName]; exists {
			phase := ExecutionPhase{
				Name:           phaseName,
				Order:          i + 1,
				ParallelGroups: []ParallelGroup{},
			}

			// Build parallel groups using topological sort
			groups := a.buildParallelGroups(steps)
			phase.ParallelGroups = groups

			// Calculate phase time (max of parallel groups)
			phaseTime := time.Duration(0)
			for _, group := range groups {
				if group.EstimatedTime > phaseTime {
					phaseTime = group.EstimatedTime
				}
			}
			phase.EstimatedTime = phaseTime
			totalTime += phaseTime

			// Track max parallel steps
			for _, group := range groups {
				if len(group.Steps) > plan.MaxParallel {
					plan.MaxParallel = len(group.Steps)
				}
			}

			plan.Phases = append(plan.Phases, phase)
		}
	}

	plan.TotalTime = totalTime
	return plan, nil
}

// buildParallelGroups builds parallel execution groups from dependencies
func (a *WorkflowAnalyzer) buildParallelGroups(dependencies []DependencyAnalysis) []ParallelGroup {
	var groups []ParallelGroup
	remaining := make([]DependencyAnalysis, len(dependencies))
	copy(remaining, dependencies)
	completed := make(map[string]bool)

	maxIterations := 100 // Safety limit to prevent infinite loops
	iteration := 0

	for len(remaining) > 0 && iteration < maxIterations {
		iteration++
		var group ParallelGroup
		maxTime := time.Duration(0)
		newRemaining := []DependencyAnalysis{}
		progressMade := false

		for _, dep := range remaining {
			canRun := true
			// Check if all dependencies are completed
			for _, depName := range dep.DependsOn {
				if !completed[depName] {
					canRun = false
					break
				}
			}

			if canRun {
				step := StepExecution{
					Name:          dep.StepName,
					Type:          dep.StepType,
					Order:         len(groups) + 1,
					EstimatedTime: dep.EstimatedDuration,
					Status:        "pending",
				}
				group.Steps = append(group.Steps, step)
				completed[dep.StepName] = true
				progressMade = true

				if dep.EstimatedDuration > maxTime {
					maxTime = dep.EstimatedDuration
				}
			} else {
				newRemaining = append(newRemaining, dep)
			}
		}

		if len(group.Steps) > 0 {
			group.EstimatedTime = maxTime
			groups = append(groups, group)
		}

		// If no progress was made and there are still remaining dependencies,
		// there might be circular dependencies or unresolvable dependencies
		if !progressMade && len(newRemaining) > 0 {
			// Add remaining steps as a final group to prevent infinite loop
			finalGroup := ParallelGroup{Steps: []StepExecution{}, EstimatedTime: 0}
			for _, dep := range newRemaining {
				step := StepExecution{
					Name:          dep.StepName,
					Type:          dep.StepType,
					Order:         len(groups) + 1,
					EstimatedTime: dep.EstimatedDuration,
					Status:        "pending",
				}
				finalGroup.Steps = append(finalGroup.Steps, step)
				if dep.EstimatedDuration > finalGroup.EstimatedTime {
					finalGroup.EstimatedTime = dep.EstimatedDuration
				}
			}
			if len(finalGroup.Steps) > 0 {
				groups = append(groups, finalGroup)
			}
			break
		}

		remaining = newRemaining
	}

	return groups
}

// Helper methods

func (a *WorkflowAnalyzer) getStepDuration(stepType string) time.Duration {
	if duration, exists := a.stepDurations[stepType]; exists {
		return duration
	}
	return 2 * time.Minute // default
}

func (a *WorkflowAnalyzer) getResourceProvisionTime(resourceType string) time.Duration {
	if info, exists := a.resourceTypes[resourceType]; exists {
		return info.ProvisionTime
	}
	return 3 * time.Minute // default
}

func (a *WorkflowAnalyzer) determinePhase(stepType, workflowName string) string {
	validationTypes := []string{"validation", "security", "policy"}
	deploymentTypes := []string{"terraform", "kubernetes", "resource-provisioning"}
	postDeploymentTypes := []string{"monitoring", "health-check"}

	for _, t := range validationTypes {
		if stepType == t {
			return "pre-deployment"
		}
	}
	for _, t := range deploymentTypes {
		if stepType == t {
			return "deployment"
		}
	}
	for _, t := range postDeploymentTypes {
		if stepType == t {
			return "post-deployment"
		}
	}
	return "deployment"
}

func (a *WorkflowAnalyzer) getStepDependencies(step types.Step, previousSteps []types.Step) []string {
	var deps []string

	// Simple dependency rules based on step types
	dependencyRules := map[string][]string{
		"kubernetes":     {"terraform", "resource-provisioning"},
		"monitoring":     {"kubernetes"},
		"health-check":   {"kubernetes"},
		"argocd-app":     {"gitea-repo", "git-commit-manifests"},
	}

	if stepDeps, exists := dependencyRules[step.Type]; exists {
		for _, prevStep := range previousSteps {
			for _, depType := range stepDeps {
				if prevStep.Type == depType {
					deps = append(deps, prevStep.Name)
				}
			}
		}
	}

	return deps
}

func (a *WorkflowAnalyzer) getStepsBlocked(step types.Step, futureSteps []types.Step) []string {
	var blocked []string

	// Steps that typically block others
	blockingRules := map[string][]string{
		"terraform":          {"kubernetes"},
		"resource-provisioning": {"kubernetes"},
		"gitea-repo":         {"git-commit-manifests", "argocd-app"},
		"git-commit-manifests": {"argocd-app"},
	}

	if blockedTypes, exists := blockingRules[step.Type]; exists {
		for _, futureStep := range futureSteps {
			for _, blockedType := range blockedTypes {
				if futureStep.Type == blockedType {
					blocked = append(blocked, futureStep.Name)
				}
			}
		}
	}

	return blocked
}

func (a *WorkflowAnalyzer) getResourceLevels(graph ResourceGraph) [][]ResourceNode {
	maxLevel := 0
	for _, node := range graph.Nodes {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	levels := make([][]ResourceNode, maxLevel+1)
	for _, node := range graph.Nodes {
		levels[node.Level] = append(levels[node.Level], node)
	}

	return levels
}

func (a *WorkflowAnalyzer) generateWarnings(spec *types.ScoreSpec, dependencies []DependencyAnalysis) []string {
	var warnings []string

	// Check for long-running sequential steps
	for _, dep := range dependencies {
		if dep.EstimatedDuration > 5*time.Minute && !dep.CanRunInParallel {
			warnings = append(warnings, fmt.Sprintf("Step '%s' has long duration (%v) and cannot run in parallel", dep.StepName, dep.EstimatedDuration))
		}
	}

	// Check for missing validation steps
	hasValidation := false
	for _, dep := range dependencies {
		if dep.StepType == "validation" || dep.StepType == "security" {
			hasValidation = true
			break
		}
	}
	if !hasValidation {
		warnings = append(warnings, "No validation or security steps found in workflow")
	}

	// Check for resource complexity
	if len(spec.Resources) > 5 {
		warnings = append(warnings, fmt.Sprintf("High number of resources (%d) may increase complexity", len(spec.Resources)))
	}

	return warnings
}

func (a *WorkflowAnalyzer) generateRecommendations(spec *types.ScoreSpec, dependencies []DependencyAnalysis, plan ExecutionPlan) []string {
	var recommendations []string

	// Suggest parallelization opportunities
	parallelizable := 0
	for _, dep := range dependencies {
		if dep.CanRunInParallel {
			parallelizable++
		}
	}
	if parallelizable > 1 {
		recommendations = append(recommendations, fmt.Sprintf("Consider running %d steps in parallel to reduce total execution time", parallelizable))
	}

	// Suggest monitoring if missing
	hasMonitoring := false
	for _, dep := range dependencies {
		if dep.StepType == "monitoring" {
			hasMonitoring = true
			break
		}
	}
	if !hasMonitoring {
		recommendations = append(recommendations, "Consider adding monitoring setup for better observability")
	}

	// Suggest optimization if execution time is long
	if plan.TotalTime > 15*time.Minute {
		recommendations = append(recommendations, "Total execution time is high - consider optimizing resource provisioning or using pre-built images")
	}

	return recommendations
}

func (a *WorkflowAnalyzer) createSummary(spec *types.ScoreSpec, dependencies []DependencyAnalysis, plan ExecutionPlan) AnalysisSummary {
	// Find critical path (longest sequential path)
	criticalPath := a.findCriticalPath(dependencies)

	// Calculate complexity score
	complexityScore := len(dependencies)*2 + len(spec.Resources)*3
	if len(criticalPath) > 5 {
		complexityScore += 10
	}

	// Determine risk level
	riskLevel := "low"
	if complexityScore > 30 {
		riskLevel = "high"
	} else if complexityScore > 15 {
		riskLevel = "medium"
	}

	return AnalysisSummary{
		TotalSteps:      len(dependencies),
		TotalResources:  len(spec.Resources),
		ParallelSteps:   plan.MaxParallel,
		CriticalPath:    criticalPath,
		EstimatedTime:   plan.TotalTime,
		ComplexityScore: complexityScore,
		RiskLevel:       riskLevel,
	}
}

func (a *WorkflowAnalyzer) findCriticalPath(dependencies []DependencyAnalysis) []string {
	// Simplified critical path calculation
	// In a real implementation, this would use graph algorithms
	var path []string

	// Find steps with no dependencies (starting points)
	for _, dep := range dependencies {
		if len(dep.DependsOn) == 0 {
			path = append(path, dep.StepName)
			break
		}
	}

	// Follow the longest dependency chain
	for i := 0; i < len(dependencies); i++ {
		longestNext := ""
		longestDuration := time.Duration(0)

		for _, dep := range dependencies {
			for _, depName := range dep.DependsOn {
				if len(path) > 0 && depName == path[len(path)-1] {
					if dep.EstimatedDuration > longestDuration {
						longestNext = dep.StepName
						longestDuration = dep.EstimatedDuration
					}
				}
			}
		}

		if longestNext != "" {
			path = append(path, longestNext)
		} else {
			break
		}
	}

	return path
}

// Default configurations
func getDefaultStepDurations() map[string]time.Duration {
	return map[string]time.Duration{
		"validation":             30 * time.Second,
		"security":               2 * time.Minute,
		"policy":                 1 * time.Minute,
		"terraform":              5 * time.Minute,
		"ansible":                3 * time.Minute,
		"kubernetes":             3 * time.Minute,
		"resource-provisioning":  4 * time.Minute,
		"monitoring":             2 * time.Minute,
		"health-check":           1 * time.Minute,
		"gitea-repo":             30 * time.Second,
		"git-commit-manifests":   1 * time.Minute,
		"argocd-app":             2 * time.Minute,
		"vault-setup":            2 * time.Minute,
		"database-migration":     3 * time.Minute,
		"cost-analysis":          2 * time.Minute,
		"tagging":                1 * time.Minute,
	}
}

func getDefaultResourceTypes() map[string]ResourceTypeInfo {
	return map[string]ResourceTypeInfo{
		"postgres": {
			ProvisionTime: 5 * time.Minute,
			Dependencies:  []string{"volume"},
			Complexity:    3,
		},
		"redis": {
			ProvisionTime: 3 * time.Minute,
			Dependencies:  []string{"volume"},
			Complexity:    2,
		},
		"volume": {
			ProvisionTime: 2 * time.Minute,
			Dependencies:  []string{},
			Complexity:    1,
		},
		"route": {
			ProvisionTime: 1 * time.Minute,
			Dependencies:  []string{"service"},
			Complexity:    1,
		},
		"service": {
			ProvisionTime: 30 * time.Second,
			Dependencies:  []string{},
			Complexity:    1,
		},
	}
}