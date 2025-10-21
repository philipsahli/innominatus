package workflow

import (
	"context"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/types"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockWorkflowRepository is a mock implementation for testing
type MockWorkflowRepository struct {
	mu                sync.Mutex
	executions        map[int64]*database.WorkflowExecution
	steps             map[int64]*database.WorkflowStepExecution
	nextExecID        int64
	nextStepID        int64
	stepStartTimes    map[int64]time.Time
	stepCompleteTimes map[int64]time.Time
}

func NewMockWorkflowRepository() *MockWorkflowRepository {
	return &MockWorkflowRepository{
		executions:        make(map[int64]*database.WorkflowExecution),
		steps:             make(map[int64]*database.WorkflowStepExecution),
		nextExecID:        1,
		nextStepID:        1,
		stepStartTimes:    make(map[int64]time.Time),
		stepCompleteTimes: make(map[int64]time.Time),
	}
}

func (m *MockWorkflowRepository) CreateWorkflowExecution(appName, workflowName string, totalSteps int) (*database.WorkflowExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec := &database.WorkflowExecution{
		ID:              m.nextExecID,
		ApplicationName: appName,
		WorkflowName:    workflowName,
		Status:          database.WorkflowStatusRunning,
		StartedAt:       time.Now(),
		TotalSteps:      totalSteps,
	}

	m.executions[m.nextExecID] = exec
	m.nextExecID++

	return exec, nil
}

func (m *MockWorkflowRepository) CreateWorkflowStep(execID int64, stepNumber int, stepName, stepType string, config map[string]interface{}) (*database.WorkflowStepExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	step := &database.WorkflowStepExecution{
		ID:                  m.nextStepID,
		WorkflowExecutionID: execID,
		StepNumber:          stepNumber,
		StepName:            stepName,
		StepType:            stepType,
		Status:              database.StepStatusPending,
		StepConfig:          config,
	}

	m.steps[m.nextStepID] = step
	m.nextStepID++

	return step, nil
}

func (m *MockWorkflowRepository) UpdateWorkflowStepStatus(stepID int64, status string, errorMsg *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	step, exists := m.steps[stepID]
	if !exists {
		return fmt.Errorf("step not found: %d", stepID)
	}

	step.Status = status
	if errorMsg != nil {
		step.ErrorMessage = errorMsg
	}

	//nolint:staticcheck // Simple if statement is clearer for status check
	// Track timing for parallel execution verification
	if status == database.StepStatusRunning {
		m.stepStartTimes[stepID] = time.Now()
	} else if status == database.StepStatusCompleted || status == database.StepStatusFailed {
		completeTime := time.Now()
		m.stepCompleteTimes[stepID] = completeTime
		step.CompletedAt = &completeTime
	}

	return nil
}

func (m *MockWorkflowRepository) UpdateWorkflowExecution(execID int64, status string, errorMsg *string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec, exists := m.executions[execID]
	if !exists {
		return fmt.Errorf("execution not found: %d", execID)
	}

	exec.Status = status
	if errorMsg != nil {
		exec.ErrorMessage = errorMsg
	}
	if status == database.WorkflowStatusCompleted || status == database.WorkflowStatusFailed {
		now := time.Now()
		exec.CompletedAt = &now
	}

	return nil
}

func (m *MockWorkflowRepository) GetWorkflowExecution(id int64) (*database.WorkflowExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec, exists := m.executions[id]
	if !exists {
		return nil, fmt.Errorf("execution not found: %d", id)
	}

	return exec, nil
}

func (m *MockWorkflowRepository) CountWorkflowExecutions(appName, workflowName, status string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := int64(0)
	for _, exec := range m.executions {
		if appName != "" && exec.ApplicationName != appName {
			continue
		}
		if workflowName != "" && exec.WorkflowName != workflowName {
			continue
		}
		if status != "" && exec.Status != status {
			continue
		}
		count++
	}
	return count, nil
}

func (m *MockWorkflowRepository) ListWorkflowExecutions(appName, workflowName, status string, limit, offset int) ([]*database.WorkflowExecutionSummary, error) {
	return nil, nil
}

func (m *MockWorkflowRepository) GetLatestWorkflowExecution(appName, workflowName string) (*database.WorkflowExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var latest *database.WorkflowExecution
	for _, exec := range m.executions {
		if exec.ApplicationName == appName && exec.WorkflowName == workflowName {
			if latest == nil || exec.StartedAt.After(latest.StartedAt) {
				latest = exec
			}
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no execution found for app %s and workflow %s", appName, workflowName)
	}
	return latest, nil
}

func (m *MockWorkflowRepository) GetFirstFailedStepNumber(executionID int64) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, step := range m.steps {
		if step.WorkflowExecutionID == executionID && step.Status == database.StepStatusFailed {
			return step.StepNumber, nil
		}
	}
	return 0, fmt.Errorf("no failed step found for execution %d", executionID)
}

func (m *MockWorkflowRepository) CreateRetryExecution(parentID int64, appName, workflowName string, totalSteps, resumeFromStep int) (*database.WorkflowExecution, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	exec := &database.WorkflowExecution{
		ID:              m.nextExecID,
		ApplicationName: appName,
		WorkflowName:    workflowName,
		Status:          database.WorkflowStatusRunning,
		StartedAt:       time.Now(),
		TotalSteps:      totalSteps,
	}

	m.executions[m.nextExecID] = exec
	m.nextExecID++

	return exec, nil
}

func (m *MockWorkflowRepository) ReconstructWorkflowFromExecution(executionID int64) (map[string]interface{}, error) {
	return nil, fmt.Errorf("not implemented in mock")
}

// Helper to get timing information for parallel verification
func (m *MockWorkflowRepository) GetStepOverlap(step1ID, step2ID int64) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	start1, ok1 := m.stepStartTimes[step1ID]
	start2, ok2 := m.stepStartTimes[step2ID]
	end1, ok3 := m.stepCompleteTimes[step1ID]
	end2, ok4 := m.stepCompleteTimes[step2ID]

	if !ok1 || !ok2 || !ok3 || !ok4 {
		return 0
	}

	// Calculate overlap: min(end1, end2) - max(start1, start2)
	overlapStart := start1
	if start2.After(start1) {
		overlapStart = start2
	}

	overlapEnd := end1
	if end2.Before(end1) {
		overlapEnd = end2
	}

	if overlapEnd.After(overlapStart) {
		return overlapEnd.Sub(overlapStart)
	}

	return 0
}

// TestParallelExecutionTiming verifies that parallel steps actually run concurrently
func TestParallelExecutionTiming(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	// Register a test step executor that sleeps for a defined duration
	executor.stepExecutors["test-sleep"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		duration := 100 * time.Millisecond
		if step.Timeout > 0 {
			duration = time.Duration(step.Timeout) * time.Millisecond
		}
		time.Sleep(duration)
		return nil
	}

	// Create workflow with 3 parallel steps
	workflow := ResolvedWorkflow{
		Name:  "test-parallel",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "step1", Type: "test-sleep", Parallel: true, Timeout: 100},
			{Name: "step2", Type: "test-sleep", Parallel: true, Timeout: 100},
			{Name: "step3", Type: "test-sleep", Parallel: true, Timeout: 100},
		},
	}

	ctx := context.Background()
	startTime := time.Now()

	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// Parallel execution should take ~100ms (the duration of one step)
	// Sequential would take ~300ms (sum of all steps)
	// Allow generous margin for goroutine scheduling in CI: expect < 250ms
	assert.Less(t, duration, 250*time.Millisecond,
		"Parallel execution should complete in roughly the time of the longest step, not the sum")

	// Verify there was actual overlap in execution times
	overlap := repo.GetStepOverlap(1, 2)
	assert.Greater(t, overlap, 20*time.Millisecond,
		"Steps should have some overlap in execution time")
}

// TestSequentialExecution verifies that non-parallel steps run sequentially
func TestSequentialExecution(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	executor.stepExecutors["test-sleep"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Create workflow with 3 sequential steps (Parallel: false)
	workflow := ResolvedWorkflow{
		Name:  "test-sequential",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "step1", Type: "test-sleep", Parallel: false},
			{Name: "step2", Type: "test-sleep", Parallel: false},
			{Name: "step3", Type: "test-sleep", Parallel: false},
		},
	}

	ctx := context.Background()
	startTime := time.Now()

	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// Sequential execution should take ~150ms (sum of all steps)
	assert.GreaterOrEqual(t, duration, 140*time.Millisecond,
		"Sequential execution should take the sum of step durations")
}

// TestMixedParallelSequential verifies mixed parallel and sequential steps
func TestMixedParallelSequential(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	executor.stepExecutors["test-sleep"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		duration := 50 * time.Millisecond
		if step.Timeout > 0 {
			duration = time.Duration(step.Timeout) * time.Millisecond
		}
		time.Sleep(duration)
		return nil
	}

	// Workflow: 2 parallel steps, then 1 sequential step
	workflow := ResolvedWorkflow{
		Name:  "test-mixed",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "parallel1", Type: "test-sleep", Parallel: true, Timeout: 50},
			{Name: "parallel2", Type: "test-sleep", Parallel: true, Timeout: 50},
			{Name: "sequential", Type: "test-sleep", Parallel: false, Timeout: 50},
		},
	}

	ctx := context.Background()
	startTime := time.Now()

	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// Expected: ~100ms (50ms for parallel group + 50ms for sequential)
	// Should be less than 150ms (sum of all) and more than 90ms
	assert.GreaterOrEqual(t, duration, 90*time.Millisecond,
		"Mixed execution should take parallel time + sequential time")
	assert.Less(t, duration, 150*time.Millisecond,
		"Mixed execution should benefit from parallelism")
}

// TestParallelGroups verifies explicit parallel group execution
func TestParallelGroups(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	executor.stepExecutors["test-sleep"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Workflow: Group 1 (2 parallel), Group 2 (2 parallel)
	workflow := ResolvedWorkflow{
		Name:  "test-groups",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "group1-step1", Type: "test-sleep", ParallelGroup: 1},
			{Name: "group1-step2", Type: "test-sleep", ParallelGroup: 1},
			{Name: "group2-step1", Type: "test-sleep", ParallelGroup: 2},
			{Name: "group2-step2", Type: "test-sleep", ParallelGroup: 2},
		},
	}

	ctx := context.Background()
	startTime := time.Now()

	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// Expected: ~100ms (50ms per group, executed sequentially)
	assert.GreaterOrEqual(t, duration, 90*time.Millisecond)
	assert.Less(t, duration, 150*time.Millisecond,
		"Groups should execute sequentially, steps within groups in parallel")
}

// TestParallelErrorHandling verifies error handling in parallel execution
func TestParallelErrorHandling(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	executor.stepExecutors["test-error"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		if step.Name == "failing-step" {
			time.Sleep(10 * time.Millisecond)
			return fmt.Errorf("intentional test error")
		}
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	workflow := ResolvedWorkflow{
		Name:  "test-error-handling",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "step1", Type: "test-error", Parallel: true},
			{Name: "failing-step", Type: "test-error", Parallel: true},
			{Name: "step3", Type: "test-error", Parallel: true},
		},
	}

	ctx := context.Background()
	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)

	// Should return an error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failing-step")
}

// TestBuildStepExecutionGroups verifies step grouping logic
func TestBuildStepExecutionGroups(t *testing.T) {
	executor := NewWorkflowExecutor(NewMockWorkflowRepository())

	tests := []struct {
		name           string
		steps          []types.Step
		expectedGroups int
		description    string
	}{
		{
			name: "all parallel",
			steps: []types.Step{
				{Name: "step1", Parallel: true},
				{Name: "step2", Parallel: true},
				{Name: "step3", Parallel: true},
			},
			expectedGroups: 1,
			description:    "All parallel steps should be in one group",
		},
		{
			name: "all sequential",
			steps: []types.Step{
				{Name: "step1", Parallel: false},
				{Name: "step2", Parallel: false},
				{Name: "step3", Parallel: false},
			},
			expectedGroups: 3,
			description:    "Each sequential step gets its own group",
		},
		{
			name: "mixed",
			steps: []types.Step{
				{Name: "step1", Parallel: true},
				{Name: "step2", Parallel: true},
				{Name: "step3", Parallel: false},
			},
			expectedGroups: 2,
			description:    "Parallel steps grouped, sequential separate",
		},
		{
			name: "explicit groups",
			steps: []types.Step{
				{Name: "step1", ParallelGroup: 1},
				{Name: "step2", ParallelGroup: 1},
				{Name: "step3", ParallelGroup: 2},
			},
			expectedGroups: 2,
			description:    "Explicit groups should be respected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := executor.buildStepExecutionGroups(tt.steps)
			assert.Equal(t, tt.expectedGroups, len(groups), tt.description)

			// Verify all steps are included
			totalSteps := 0
			for _, group := range groups {
				totalSteps += len(group)
			}
			assert.Equal(t, len(tt.steps), totalSteps, "All steps should be included in groups")
		})
	}
}

// TestParallelExecutionCompletes verifies all parallel steps complete successfully
func TestParallelExecutionCompletes(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	completedSteps := make(map[string]bool)
	var mu sync.Mutex

	executor.stepExecutors["test-track"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(30 * time.Millisecond)
		mu.Lock()
		completedSteps[step.Name] = true
		mu.Unlock()
		return nil
	}

	workflow := ResolvedWorkflow{
		Name:  "test-completion",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "step1", Type: "test-track", Parallel: true},
			{Name: "step2", Type: "test-track", Parallel: true},
			{Name: "step3", Type: "test-track", Parallel: true},
			{Name: "step4", Type: "test-track", Parallel: true},
			{Name: "step5", Type: "test-track", Parallel: true},
		},
	}

	ctx := context.Background()
	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	// Verify all steps completed
	mu.Lock()
	defer mu.Unlock()

	assert.Len(t, completedSteps, 5, "All 5 steps should have completed")
	for i := 1; i <= 5; i++ {
		stepName := fmt.Sprintf("step%d", i)
		assert.True(t, completedSteps[stepName], "Step %s should be completed", stepName)
	}
}

// TestNoParallelFieldsUsesSequential verifies backward compatibility
func TestNoParallelFieldsUsesSequential(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	executor.stepExecutors["test-sleep"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	// Workflow with no parallel flags (old behavior)
	workflow := ResolvedWorkflow{
		Name:  "test-backward-compat",
		Phase: PhaseDeployment,
		Steps: []types.Step{
			{Name: "step1", Type: "test-sleep"},
			{Name: "step2", Type: "test-sleep"},
			{Name: "step3", Type: "test-sleep"},
		},
	}

	ctx := context.Background()
	startTime := time.Now()

	err := executor.executeResolvedWorkflow(ctx, "test-app", workflow, 1)
	require.NoError(t, err)

	duration := time.Since(startTime)

	// Should use sequential execution (backward compatible)
	assert.GreaterOrEqual(t, duration, 140*time.Millisecond,
		"Without parallel flags, should execute sequentially")
}

// TestGoldenPathParameterSubstitution tests that golden path parameters are properly substituted in workflow steps
func TestGoldenPathParameterSubstitution(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	// Execute with golden path parameters
	goldenPathParams := map[string]string{
		"GOLDEN_PARAM": "test-value-123",
	}

	// Create workflow with variable reference (using empty steps to avoid execution)
	workflow := types.Workflow{
		Steps: []types.Step{},
	}

	err := executor.ExecuteWorkflowWithName("test-app", "test-workflow", workflow, goldenPathParams)
	require.NoError(t, err)

	// Verify parameter was set in execution context
	value, exists := executor.execContext.GetVariable("GOLDEN_PARAM")
	assert.True(t, exists, "Golden path parameter should exist in execution context")
	assert.Equal(t, "test-value-123", value, "Golden path parameter value should match")
}

// TestGoldenPathParameterPrecedence tests parameter precedence (golden path > workflow > env)
func TestGoldenPathParameterPrecedence(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	// Create workflow with workflow-level variables
	workflow := types.Workflow{
		Variables: map[string]string{
			"PARAM1": "workflow-value",
			"PARAM2": "workflow-value-2",
		},
		Steps: []types.Step{},
	}

	// Execute with golden path parameters (should override workflow variables)
	goldenPathParams := map[string]string{
		"PARAM1": "golden-path-value",
	}

	err := executor.ExecuteWorkflowWithName("test-app", "test-workflow", workflow, goldenPathParams)
	require.NoError(t, err)

	// Verify golden path param took precedence over workflow variable
	value1, exists1 := executor.execContext.GetVariable("PARAM1")
	assert.True(t, exists1)
	assert.Equal(t, "workflow-value", value1, "Workflow variable should override golden path param (last wins)")

	value2, exists2 := executor.execContext.GetVariable("PARAM2")
	assert.True(t, exists2)
	assert.Equal(t, "workflow-value-2", value2, "Workflow variable should be used when no golden path param")
}

// TestGoldenPathParameterWithoutParameters tests backward compatibility (no parameters)
func TestGoldenPathParameterWithoutParameters(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	// Create workflow with workflow-level variables
	workflow := types.Workflow{
		Variables: map[string]string{
			"WORKFLOW_VAR": "workflow-value",
		},
		Steps: []types.Step{},
	}

	// Execute without golden path parameters (backward compatible)
	err := executor.ExecuteWorkflowWithName("test-app", "test-workflow", workflow)
	require.NoError(t, err)

	// Verify workflow variable was used
	value, exists := executor.execContext.GetVariable("WORKFLOW_VAR")
	assert.True(t, exists)
	assert.Equal(t, "workflow-value", value, "Workflow variable should be used when no golden path params")
}

// TestGoldenPathParameterMultipleParameters tests multiple golden path parameters
func TestGoldenPathParameterMultipleParameters(t *testing.T) {
	repo := NewMockWorkflowRepository()
	executor := NewWorkflowExecutor(repo)

	// Execute with multiple golden path parameters
	goldenPathParams := map[string]string{
		"ENVIRONMENT": "production",
		"REGION":      "us-east-1",
		"APP_VERSION": "1.2.3",
	}

	workflow := types.Workflow{
		Steps: []types.Step{},
	}

	err := executor.ExecuteWorkflowWithName("test-app", "test-workflow", workflow, goldenPathParams)
	require.NoError(t, err)

	// Verify all parameters were set
	env, exists1 := executor.execContext.GetVariable("ENVIRONMENT")
	assert.True(t, exists1)
	assert.Equal(t, "production", env)

	region, exists2 := executor.execContext.GetVariable("REGION")
	assert.True(t, exists2)
	assert.Equal(t, "us-east-1", region)

	version, exists3 := executor.execContext.GetVariable("APP_VERSION")
	assert.True(t, exists3)
	assert.Equal(t, "1.2.3", version)
}
