package workflow

import (
	"innominatus/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflowAnalyzer(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	assert.NotNil(t, analyzer)
	assert.NotNil(t, analyzer.stepDurations)
	assert.NotNil(t, analyzer.resourceTypes)

	// Test that default durations are loaded
	assert.True(t, len(analyzer.stepDurations) > 0)
	assert.True(t, len(analyzer.resourceTypes) > 0)
}

func TestAnalyzeSpec_SimpleSpec(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		APIVersion: "score.dev/v1b1",
		Metadata: types.Metadata{
			Name: "test-app",
		},
		Containers: map[string]types.Container{
			"web": {
				Image: "nginx:latest",
			},
		},
		Resources: map[string]types.Resource{
			"db": {
				Type: "postgres",
				Params: map[string]interface{}{
					"version": "13",
				},
			},
		},
	}

	analysis, err := analyzer.AnalyzeSpec(spec)
	require.NoError(t, err)
	require.NotNil(t, analysis)

	// Test basic analysis structure
	assert.Equal(t, spec, analysis.Spec)
	assert.NotNil(t, analysis.Dependencies)
	assert.NotNil(t, analysis.ExecutionPlan)
	assert.NotNil(t, analysis.ResourceGraph)
	assert.NotNil(t, analysis.Summary)

	// Test summary values
	assert.Greater(t, analysis.Summary.TotalSteps, 0)
	assert.Equal(t, 1, analysis.Summary.TotalResources)
	assert.NotEmpty(t, analysis.Summary.RiskLevel)
	assert.Greater(t, analysis.Summary.ComplexityScore, 0)
}

func TestAnalyzeSpec_WithWorkflows(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		APIVersion: "score.dev/v1b1",
		Metadata: types.Metadata{
			Name: "test-app",
		},
		Containers: map[string]types.Container{
			"web": {Image: "nginx:latest"},
		},
		Workflows: map[string]types.Workflow{
			"deploy": {
				Steps: []types.Step{
					{Name: "setup-infra", Type: "terraform", Path: "./terraform"},
					{Name: "deploy-app", Type: "kubernetes", Namespace: "test-app"},
					{Name: "run-tests", Type: "validation"},
				},
			},
		},
	}

	analysis, err := analyzer.AnalyzeSpec(spec)
	require.NoError(t, err)

	// Should include workflow steps in analysis
	assert.Greater(t, len(analysis.Dependencies), 0)

	// Find our custom steps
	stepNames := make([]string, len(analysis.Dependencies))
	for i, dep := range analysis.Dependencies {
		stepNames[i] = dep.StepName
	}

	assert.Contains(t, stepNames, "setup-infra")
	assert.Contains(t, stepNames, "deploy-app")
	assert.Contains(t, stepNames, "run-tests")
}

func TestAnalyzeResources(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
			"cache": {Type: "redis"},
			"storage": {Type: "volume"},
		},
	}

	graph, err := analyzer.analyzeResources(spec)
	require.NoError(t, err)

	// Should have application + 3 resources = 4 nodes
	assert.Len(t, graph.Nodes, 4)

	// Should have edges connecting app to resources
	assert.Greater(t, len(graph.Edges), 0)

	// Check that nodes have correct types
	nodeTypes := make(map[string]string)
	for _, node := range graph.Nodes {
		nodeTypes[node.Name] = node.Type
	}

	assert.Equal(t, "application", nodeTypes["test-app"])
	assert.Equal(t, "postgres", nodeTypes["db"])
	assert.Equal(t, "redis", nodeTypes["cache"])
	assert.Equal(t, "volume", nodeTypes["storage"])
}

func TestBuildParallelGroups(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	dependencies := []DependencyAnalysis{
		{
			StepName:          "step1",
			StepType:          "validation",
			DependsOn:         []string{},
			CanRunInParallel:  true,
			EstimatedDuration: 1 * time.Minute,
		},
		{
			StepName:          "step2",
			StepType:          "security",
			DependsOn:         []string{},
			CanRunInParallel:  true,
			EstimatedDuration: 2 * time.Minute,
		},
		{
			StepName:          "step3",
			StepType:          "terraform",
			DependsOn:         []string{"step1", "step2"},
			CanRunInParallel:  false,
			EstimatedDuration: 5 * time.Minute,
		},
	}

	groups := analyzer.buildParallelGroups(dependencies)

	// The function should create groups based on dependencies
	// First group: steps with no dependencies (step1, step2)
	// Second group: steps that depend on the first group (step3)
	assert.Greater(t, len(groups), 0, "Should create at least one group")

	if len(groups) >= 1 {
		// Check that steps are distributed across groups
		totalSteps := 0
		for _, group := range groups {
			totalSteps += len(group.Steps)
		}
		assert.Equal(t, 3, totalSteps, "All steps should be included")

		// First group should contain steps without dependencies
		firstGroup := groups[0]
		assert.Greater(t, len(firstGroup.Steps), 0, "First group should have steps")

		// Check that dependencies are properly handled
		stepNames := make([]string, len(firstGroup.Steps))
		for i, step := range firstGroup.Steps {
			stepNames[i] = step.Name
		}

		// At minimum, step1 and step2 should be able to run first
		// (since they have no dependencies)
		foundIndependentStep := false
		for _, name := range stepNames {
			if name == "step1" || name == "step2" {
				foundIndependentStep = true
				break
			}
		}
		assert.True(t, foundIndependentStep, "First group should contain steps without dependencies")
	}
}

func TestBuildParallelGroups_CircularDependency(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	// Create circular dependency: step1 -> step2 -> step3 -> step1
	dependencies := []DependencyAnalysis{
		{
			StepName:         "step1",
			DependsOn:        []string{"step3"},
			EstimatedDuration: 1 * time.Minute,
		},
		{
			StepName:         "step2",
			DependsOn:        []string{"step1"},
			EstimatedDuration: 1 * time.Minute,
		},
		{
			StepName:         "step3",
			DependsOn:        []string{"step2"},
			EstimatedDuration: 1 * time.Minute,
		},
	}

	// Should handle circular dependencies gracefully
	groups := analyzer.buildParallelGroups(dependencies)

	// Should still create groups (may include unresolved dependencies in final group)
	assert.Greater(t, len(groups), 0)

	// Total steps should be preserved
	totalSteps := 0
	for _, group := range groups {
		totalSteps += len(group.Steps)
	}
	assert.Equal(t, 3, totalSteps)
}

func TestGenerateImpliedWorkflow(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	resourceGraph := ResourceGraph{
		Nodes: []ResourceNode{
			{ID: "app-1", Name: "test-app", Type: "application", Level: 0},
			{ID: "resource-1", Name: "db", Type: "postgres", Level: 1},
		},
		Edges: []ResourceEdge{
			{From: "app-1", To: "resource-1", DependencyType: "requires"},
		},
	}

	dependencies := analyzer.generateImpliedWorkflow(spec, resourceGraph)

	// Should generate standard workflow phases
	assert.Greater(t, len(dependencies), 0)

	// Check that we have pre-deployment, deployment, and post-deployment steps
	phases := make(map[string]int)
	for _, dep := range dependencies {
		phases[dep.Phase]++
	}

	assert.Greater(t, phases["pre-deployment"], 0)
	assert.Greater(t, phases["deployment"], 0)
	assert.Greater(t, phases["post-deployment"], 0)
}

func TestGetStepDuration(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	tests := []struct {
		stepType string
		want     time.Duration
	}{
		{"validation", 30 * time.Second},
		{"security", 2 * time.Minute},
		{"terraform", 5 * time.Minute},
		{"kubernetes", 3 * time.Minute},
		{"unknown-type", 2 * time.Minute}, // default
	}

	for _, tt := range tests {
		t.Run(tt.stepType, func(t *testing.T) {
			duration := analyzer.getStepDuration(tt.stepType)
			assert.Equal(t, tt.want, duration)
		})
	}
}

func TestDeterminePhase(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	tests := []struct {
		stepType string
		workflow string
		want     string
	}{
		{"validation", "deploy", "pre-deployment"},
		{"security", "deploy", "pre-deployment"},
		{"policy", "deploy", "pre-deployment"},
		{"terraform", "deploy", "deployment"},
		{"kubernetes", "deploy", "deployment"},
		{"resource-provisioning", "deploy", "deployment"},
		{"monitoring", "deploy", "post-deployment"},
		{"health-check", "deploy", "post-deployment"},
		{"unknown", "deploy", "deployment"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.stepType, func(t *testing.T) {
			phase := analyzer.determinePhase(tt.stepType, tt.workflow)
			assert.Equal(t, tt.want, phase)
		})
	}
}

func TestGenerateWarnings(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db1": {Type: "postgres"},
			"db2": {Type: "postgres"},
			"db3": {Type: "postgres"},
			"db4": {Type: "postgres"},
			"db5": {Type: "postgres"},
			"db6": {Type: "postgres"}, // 6 resources should trigger warning
		},
	}

	dependencies := []DependencyAnalysis{
		{
			StepName:          "long-step",
			StepType:          "terraform",
			CanRunInParallel:  false,
			EstimatedDuration: 6 * time.Minute, // Long duration + sequential
		},
	}

	warnings := analyzer.generateWarnings(spec, dependencies)

	// Should warn about long sequential step
	assert.Greater(t, len(warnings), 0)

	found := false
	for _, warning := range warnings {
		if contains(warning, "long duration") && contains(warning, "cannot run in parallel") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should warn about long sequential step")

	// Should warn about high number of resources
	found = false
	for _, warning := range warnings {
		if contains(warning, "High number of resources") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should warn about high number of resources")
}

func TestGenerateRecommendations(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
	}

	dependencies := []DependencyAnalysis{
		{StepName: "step1", CanRunInParallel: true},
		{StepName: "step2", CanRunInParallel: true},
		{StepName: "step3", CanRunInParallel: false},
	}

	plan := ExecutionPlan{
		TotalTime: 20 * time.Minute, // Long execution time
	}

	recommendations := analyzer.generateRecommendations(spec, dependencies, plan)

	// Should recommend parallelization
	assert.Greater(t, len(recommendations), 0)

	found := false
	for _, rec := range recommendations {
		if contains(rec, "parallel") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should recommend parallelization")

	// Should recommend optimization for long execution time
	found = false
	for _, rec := range recommendations {
		if contains(rec, "execution time is high") {
			found = true
			break
		}
	}
	assert.True(t, found, "Should recommend optimization for long execution time")
}

func TestCreateSummary(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "test-app"},
		Resources: map[string]types.Resource{
			"db": {Type: "postgres"},
		},
	}

	dependencies := []DependencyAnalysis{
		{StepName: "step1"},
		{StepName: "step2"},
		{StepName: "step3"},
	}

	plan := ExecutionPlan{
		TotalTime:   10 * time.Minute,
		MaxParallel: 2,
	}

	summary := analyzer.createSummary(spec, dependencies, plan)

	assert.Equal(t, 3, summary.TotalSteps)
	assert.Equal(t, 1, summary.TotalResources)
	assert.Equal(t, 2, summary.ParallelSteps)
	assert.Equal(t, 10*time.Minute, summary.EstimatedTime)
	assert.Greater(t, summary.ComplexityScore, 0)
	assert.NotEmpty(t, summary.RiskLevel)
}

func TestAnalyzeSpec_EmptySpec(t *testing.T) {
	analyzer := NewWorkflowAnalyzer()

	spec := &types.ScoreSpec{
		Metadata: types.Metadata{Name: "empty-app"},
	}

	analysis, err := analyzer.AnalyzeSpec(spec)
	require.NoError(t, err)

	// Should handle empty spec gracefully
	assert.Equal(t, 0, analysis.Summary.TotalResources)
	assert.Greater(t, analysis.Summary.TotalSteps, 0) // Should still have implied workflow
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    (len(s) > len(substr) && (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}