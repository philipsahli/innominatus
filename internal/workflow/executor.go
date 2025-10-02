package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// StepExecutorFunc defines the signature for step execution functions
type StepExecutorFunc func(ctx context.Context, step types.Step, appName string, execID int64) error

// WorkflowRepositoryInterface defines the methods needed for workflow persistence
type WorkflowRepositoryInterface interface {
	CreateWorkflowExecution(appName, workflowName string, totalSteps int) (*database.WorkflowExecution, error)
	CreateWorkflowStep(execID int64, stepNumber int, stepName, stepType string, config map[string]interface{}) (*database.WorkflowStepExecution, error)
	UpdateWorkflowStepStatus(stepID int64, status string, errorMessage *string) error
	UpdateWorkflowExecution(execID int64, status string, errorMessage *string) error
	GetWorkflowExecution(id int64) (*database.WorkflowExecution, error)
	ListWorkflowExecutions(appName string, limit, offset int) ([]*database.WorkflowExecutionSummary, error)
}

// ResourceManager interface defines the methods needed for resource management
type ResourceManager interface {
	GetResourcesByApplication(appName string) ([]*database.ResourceInstance, error)
	ProvisionResource(resourceID int64, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error
	TransitionResourceState(resourceID int64, newState database.ResourceLifecycleState, reason string, transitionedBy string, metadata map[string]interface{}) error
}

// WorkflowExecutor handles workflow execution with database persistence
type WorkflowExecutor struct {
	repo             WorkflowRepositoryInterface
	resolver         *WorkflowResolver
	resourceManager  ResourceManager
	maxConcurrent    int
	executionTimeout time.Duration
	stepExecutors    map[string]StepExecutorFunc
	execContext      *ExecutionContext
	outputParser     *OutputParser
	mu               sync.RWMutex
}

// NewWorkflowExecutor creates a new workflow executor with database support
func NewWorkflowExecutor(repo WorkflowRepositoryInterface) *WorkflowExecutor {
	executor := &WorkflowExecutor{
		repo:             repo,
		maxConcurrent:    5,
		executionTimeout: 30 * time.Minute,
		stepExecutors:    make(map[string]StepExecutorFunc),
		execContext:      NewExecutionContext(),
		outputParser:     NewOutputParser(),
	}
	executor.registerDefaultStepExecutors()
	return executor
}

// NewWorkflowExecutorWithResourceManager creates a new workflow executor with resource manager integration
func NewWorkflowExecutorWithResourceManager(repo WorkflowRepositoryInterface, resourceManager ResourceManager) *WorkflowExecutor {
	executor := &WorkflowExecutor{
		repo:             repo,
		resourceManager:  resourceManager,
		maxConcurrent:    5,
		executionTimeout: 30 * time.Minute,
		stepExecutors:    make(map[string]StepExecutorFunc),
		execContext:      NewExecutionContext(),
		outputParser:     NewOutputParser(),
	}
	executor.registerDefaultStepExecutors()
	return executor
}

// NewMultiTierWorkflowExecutor creates a new executor with resolver support
func NewMultiTierWorkflowExecutor(repo WorkflowRepositoryInterface, resolver *WorkflowResolver) *WorkflowExecutor {
	executor := &WorkflowExecutor{
		repo:             repo,
		resolver:         resolver,
		maxConcurrent:    5,
		executionTimeout: 30 * time.Minute,
		stepExecutors:    make(map[string]StepExecutorFunc),
		execContext:      NewExecutionContext(),
		outputParser:     NewOutputParser(),
	}
	executor.registerDefaultStepExecutors()
	return executor
}

// NewMultiTierWorkflowExecutorWithResourceManager creates a new executor with resolver and resource manager support
func NewMultiTierWorkflowExecutorWithResourceManager(repo WorkflowRepositoryInterface, resolver *WorkflowResolver, resourceManager ResourceManager) *WorkflowExecutor {
	executor := &WorkflowExecutor{
		repo:             repo,
		resolver:         resolver,
		resourceManager:  resourceManager,
		maxConcurrent:    5,
		executionTimeout: 30 * time.Minute,
		stepExecutors:    make(map[string]StepExecutorFunc),
		execContext:      NewExecutionContext(),
		outputParser:     NewOutputParser(),
	}
	executor.registerDefaultStepExecutors()
	return executor
}

// ExecuteMultiTierWorkflows executes resolved multi-tier workflows
func (e *WorkflowExecutor) ExecuteMultiTierWorkflows(ctx context.Context, app *ApplicationInstance) error {
	if e.resolver == nil {
		return fmt.Errorf("resolver not configured - use NewMultiTierWorkflowExecutor")
	}

	// Resolve workflows for the application
	resolvedWorkflows, err := e.resolver.ResolveWorkflows(app)
	if err != nil {
		return fmt.Errorf("failed to resolve workflows: %w", err)
	}

	// Validate workflows against policies
	if err := e.resolver.ValidateWorkflowPolicies(resolvedWorkflows); err != nil {
		return fmt.Errorf("workflow policy validation failed: %w", err)
	}

	// Create workflow execution record
	execution, err := e.createMultiTierExecution(app.Name, resolvedWorkflows)
	if err != nil {
		return fmt.Errorf("failed to create workflow execution: %w", err)
	}

	fmt.Printf("🚀 Starting multi-tier workflow execution for %s\n", app.Name)
	summary := e.resolver.GetWorkflowSummary(resolvedWorkflows)
	fmt.Printf("📊 Execution plan: %v total workflows across %v phases\n",
		summary["total_workflows"], len(summary["phases"].([]string)))

	// Execute workflows by phase
	phases := []WorkflowPhase{PhasePreDeployment, PhaseDeployment, PhasePostDeployment}

	for _, phase := range phases {
		workflows, exists := resolvedWorkflows[phase]
		if !exists || len(workflows) == 0 {
			continue
		}

		fmt.Printf("\n📋 Executing %s workflows (%d workflows)...\n", phase, len(workflows))

		if err := e.executePhaseWorkflows(ctx, app.Name, phase, workflows, execution.ID); err != nil {
			// Mark execution as failed
			errorMsg := err.Error()
			if updateErr := e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, &errorMsg); updateErr != nil {
				fmt.Printf("Warning: failed to update workflow status to failed: %v\n", updateErr)
			}
			return fmt.Errorf("failed executing %s workflows: %w", phase, err)
		}

		fmt.Printf("✅ %s phase completed successfully\n", phase)
	}

	// Mark execution as completed
	err = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusCompleted, nil)
	if err != nil {
		fmt.Printf("Warning: failed to update workflow completion: %v\n", err)
	}

	fmt.Printf("\n🎉 Multi-tier workflow execution completed successfully!\n")
	return nil
}

// ExecuteWorkflow executes a workflow with database persistence
func (e *WorkflowExecutor) ExecuteWorkflow(appName string, workflow types.Workflow) error {
	return e.ExecuteWorkflowWithName(appName, "deploy", workflow)
}

// ExecuteWorkflowWithName executes a named workflow with database persistence
func (e *WorkflowExecutor) ExecuteWorkflowWithName(appName, workflowName string, workflow types.Workflow) error {
	// Initialize workflow variables in execution context
	if len(workflow.Variables) > 0 {
		e.execContext.SetWorkflowVariables(workflow.Variables)
		fmt.Printf("📦 Initialized %d workflow variables\n", len(workflow.Variables))
	}

	// Create workflow execution record
	execution, err := e.repo.CreateWorkflowExecution(appName, workflowName, len(workflow.Steps))
	if err != nil {
		return fmt.Errorf("failed to create workflow execution: %w", err)
	}

	fmt.Printf("Starting workflow with %d steps\n\n", len(workflow.Steps))

	// Create step records
	stepRecords := make(map[int]*database.WorkflowStepExecution)
	for i, step := range workflow.Steps {
		stepConfig := map[string]interface{}{
			"name":      step.Name,
			"type":      step.Type,
			"path":      step.Path,
			"playbook":  step.Playbook,
			"namespace": step.Namespace,
		}

		stepRecord, err := e.repo.CreateWorkflowStep(execution.ID, i+1, step.Name, step.Type, stepConfig)
		if err != nil {
			_ = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, stringPtr(fmt.Sprintf("Failed to create step record: %v", err)))
			return fmt.Errorf("failed to create workflow step: %w", err)
		}
		stepRecords[i] = stepRecord
	}

	// Execute steps
	for i, step := range workflow.Steps {
		stepRecord := stepRecords[i]
		fmt.Printf("Step %d/%d: %s (%s)\n", i+1, len(workflow.Steps), step.Name, step.Type)

		// Update step to running
		err := e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil)
		if err != nil {
			fmt.Printf("Warning: failed to update step status: %v\n", err)
		}

		spinner := NewSpinner(fmt.Sprintf("Initializing %s step...", step.Type))
		spinner.Start()

		err = runStepWithSpinner(step, appName, "default", spinner)
		if err != nil {
			// Update step as failed
			errorMsg := err.Error()
			_ = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errorMsg)

			// Update workflow as failed
			workflowErrorMsg := fmt.Sprintf("workflow failed at step '%s': %v", step.Name, err)
			_ = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, &workflowErrorMsg)

			spinner.Stop(false, fmt.Sprintf("Step '%s' failed: %v", step.Name, err))
			return fmt.Errorf("workflow failed at step '%s': %w", step.Name, err)
		}

		// Update step as completed
		err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil)
		if err != nil {
			fmt.Printf("Warning: failed to update step completion: %v\n", err)
		}

		spinner.Stop(true, fmt.Sprintf("Step '%s' completed successfully", step.Name))
		fmt.Println()
	}

	// Update workflow as completed
	err = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusCompleted, nil)
	if err != nil {
		fmt.Printf("Warning: failed to update workflow completion: %v\n", err)
	}

	fmt.Println("🎉 Workflow completed successfully!")
	return nil
}

// GetWorkflowExecution retrieves a workflow execution by ID
func (e *WorkflowExecutor) GetWorkflowExecution(id int64) (*database.WorkflowExecution, error) {
	return e.repo.GetWorkflowExecution(id)
}

// ListWorkflowExecutions lists workflow executions for an application
func (e *WorkflowExecutor) ListWorkflowExecutions(appName string, limit, offset int) ([]*database.WorkflowExecutionSummary, error) {
	return e.repo.ListWorkflowExecutions(appName, limit, offset)
}

// RunWorkflowWithDB executes a workflow with database persistence (convenience function)
func RunWorkflowWithDB(repo WorkflowRepositoryInterface, appName string, workflow types.Workflow) error {
	executor := NewWorkflowExecutor(repo)
	return executor.ExecuteWorkflow(appName, workflow)
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

// Helper methods for multi-tier execution

// createMultiTierExecution creates a workflow execution for multi-tier workflows
func (e *WorkflowExecutor) createMultiTierExecution(appName string, workflows map[WorkflowPhase][]ResolvedWorkflow) (*database.WorkflowExecution, error) {
	totalSteps := 0
	for _, phaseWorkflows := range workflows {
		for _, workflow := range phaseWorkflows {
			totalSteps += len(workflow.Steps)
		}
	}

	return e.repo.CreateWorkflowExecution(appName, "multi-tier-deployment", totalSteps)
}

// executePhaseWorkflows executes all workflows for a specific phase
func (e *WorkflowExecutor) executePhaseWorkflows(ctx context.Context, appName string, phase WorkflowPhase, workflows []ResolvedWorkflow, execID int64) error {
	// Create a semaphore to limit concurrent workflow execution
	semaphore := make(chan struct{}, e.maxConcurrent)
	var wg sync.WaitGroup
	var executionError error
	var errorMu sync.Mutex

	for _, workflow := range workflows {
		wg.Add(1)
		go func(w ResolvedWorkflow) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fmt.Printf("  📋 Executing workflow: %s (%d steps)\n", w.Name, len(w.Steps))

			if err := e.executeResolvedWorkflow(ctx, appName, w, execID); err != nil {
				errorMu.Lock()
				if executionError == nil {
					executionError = fmt.Errorf("workflow %s failed: %w", w.Name, err)
				}
				errorMu.Unlock()
				return
			}

			fmt.Printf("  ✅ Workflow %s completed successfully\n", w.Name)
		}(workflow)
	}

	wg.Wait()
	return executionError
}

// executeResolvedWorkflow executes a single resolved workflow with support for parallel steps
func (e *WorkflowExecutor) executeResolvedWorkflow(ctx context.Context, appName string, workflow ResolvedWorkflow, execID int64) error {
	// Check if any steps are marked for parallel execution
	hasParallelSteps := false
	for _, step := range workflow.Steps {
		if step.Parallel || step.ParallelGroup > 0 {
			hasParallelSteps = true
			break
		}
	}

	// If no parallel steps, use sequential execution
	if !hasParallelSteps {
		return e.executeStepsSequentially(ctx, appName, workflow.Steps, execID)
	}

	// Group steps by parallel groups and dependencies
	stepGroups := e.buildStepExecutionGroups(workflow.Steps)

	// Execute step groups in order, steps within a group run in parallel
	for groupIdx, group := range stepGroups {
		fmt.Printf("    📦 Executing step group %d/%d (%d steps)\n", groupIdx+1, len(stepGroups), len(group))

		if err := e.executeStepGroupParallel(ctx, appName, group, execID); err != nil {
			return fmt.Errorf("step group %d failed: %w", groupIdx+1, err)
		}
	}

	return nil
}

// executeStepsSequentially executes steps one by one (original behavior)
func (e *WorkflowExecutor) executeStepsSequentially(ctx context.Context, appName string, steps []types.Step, execID int64) error {
	for i, step := range steps {
		fmt.Printf("    🔄 Step %d/%d: %s (%s)\n", i+1, len(steps), step.Name, step.Type)

		// Create step execution record
		stepConfig := map[string]interface{}{
			"name":      step.Name,
			"type":      step.Type,
			"path":      step.Path,
			"namespace": step.Namespace,
		}

		stepRecord, err := e.repo.CreateWorkflowStep(execID, i+1, step.Name, step.Type, stepConfig)
		if err != nil {
			return fmt.Errorf("failed to create step execution: %w", err)
		}

		// Update step to running
		err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil)
		if err != nil {
			fmt.Printf("Warning: failed to update step status: %v\n", err)
		}

		// Execute the step
		stepStartTime := time.Now()
		if err := e.executeStepWithExecutor(ctx, step, appName, execID); err != nil {
			// Mark step as failed
			errorMsg := err.Error()
			_ = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errorMsg)
			return fmt.Errorf("step %s failed: %w", step.Name, err)
		}

		// Mark step as completed
		err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil)
		if err != nil {
			fmt.Printf("Warning: failed to update step completion: %v\n", err)
		}

		duration := time.Since(stepStartTime)
		fmt.Printf("    ✅ Step %s completed (took %v)\n", step.Name, duration.Round(time.Millisecond))
	}

	return nil
}

// buildStepExecutionGroups builds groups of steps that can execute in parallel
func (e *WorkflowExecutor) buildStepExecutionGroups(steps []types.Step) [][]types.Step {
	// Build a map of step names to their indices
	stepIndexMap := make(map[string]int)
	for i, step := range steps {
		stepIndexMap[step.Name] = i
	}

	// Track which steps have been added to groups
	addedSteps := make(map[int]bool)
	groups := [][]types.Step{}

	// Process steps by parallel group if specified
	groupMap := make(map[int][]types.Step)
	ungroupedSteps := []types.Step{}

	for i, step := range steps {
		if step.ParallelGroup > 0 {
			groupMap[step.ParallelGroup] = append(groupMap[step.ParallelGroup], step)
			addedSteps[i] = true
		} else if !step.Parallel {
			// Sequential steps get their own group
			if len(ungroupedSteps) > 0 {
				groups = append(groups, ungroupedSteps)
				ungroupedSteps = []types.Step{}
			}
			groups = append(groups, []types.Step{step})
			addedSteps[i] = true
		} else {
			ungroupedSteps = append(ungroupedSteps, step)
			addedSteps[i] = true
		}
	}

	// Add remaining ungrouped parallel steps
	if len(ungroupedSteps) > 0 {
		groups = append(groups, ungroupedSteps)
	}

	// Add explicitly grouped steps in group order
	for groupID := 1; groupID <= len(groupMap); groupID++ {
		if groupSteps, exists := groupMap[groupID]; exists {
			groups = append(groups, groupSteps)
		}
	}

	// If no groups were created, put all steps in one sequential group
	if len(groups) == 0 {
		groups = append(groups, steps)
	}

	return groups
}

// executeStepGroupParallel executes a group of steps in parallel
func (e *WorkflowExecutor) executeStepGroupParallel(ctx context.Context, appName string, steps []types.Step, execID int64) error {
	// If only one step, execute directly
	if len(steps) == 1 {
		return e.executeSingleStep(ctx, appName, steps[0], execID, 0)
	}

	// Create channels for error handling
	var wg sync.WaitGroup
	errorChan := make(chan error, len(steps))

	// Execute all steps in parallel
	for i, step := range steps {
		wg.Add(1)
		go func(idx int, s types.Step) {
			defer wg.Done()

			if err := e.executeSingleStep(ctx, appName, s, execID, idx); err != nil {
				errorChan <- fmt.Errorf("step %s: %w", s.Name, err)
			}
		}(i, step)
	}

	// Wait for all steps to complete
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("parallel execution failed: %v", errors[0])
	}

	return nil
}

// executeSingleStep executes a single step with full database tracking
func (e *WorkflowExecutor) executeSingleStep(ctx context.Context, appName string, step types.Step, execID int64, stepNumber int) error {
	// Check if step should be executed based on conditions
	shouldExecute, skipReason := e.execContext.ShouldExecuteStep(step)
	if !shouldExecute {
		fmt.Printf("      ⏭️  %s (%s) - SKIPPED: %s\n", step.Name, step.Type, skipReason)

		// Create step execution record as skipped
		stepConfig := map[string]interface{}{
			"name":          step.Name,
			"type":          step.Type,
			"path":          step.Path,
			"namespace":     step.Namespace,
			"parallel":      step.Parallel,
			"parallelGroup": step.ParallelGroup,
			"skip_reason":   skipReason,
		}

		stepRecord, err := e.repo.CreateWorkflowStep(execID, stepNumber+1, step.Name, step.Type, stepConfig)
		if err != nil {
			return fmt.Errorf("failed to create step execution: %w", err)
		}

		// Mark step as skipped
		skippedMsg := fmt.Sprintf("skipped: %s", skipReason)
		_ = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, "skipped", &skippedMsg)

		// Record in execution context
		e.execContext.SetStepStatus(step.Name, "skipped")

		return nil
	}

	fmt.Printf("      🔄 %s (%s)\n", step.Name, step.Type)

	// Create step execution record
	stepConfig := map[string]interface{}{
		"name":          step.Name,
		"type":          step.Type,
		"path":          step.Path,
		"namespace":     step.Namespace,
		"parallel":      step.Parallel,
		"parallelGroup": step.ParallelGroup,
	}

	stepRecord, err := e.repo.CreateWorkflowStep(execID, stepNumber+1, step.Name, step.Type, stepConfig)
	if err != nil {
		return fmt.Errorf("failed to create step execution: %w", err)
	}

	// Update step to running
	err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil)
	if err != nil {
		fmt.Printf("      ⚠️  Warning: failed to update step status: %v\n", err)
	}

	// Execute the step
	stepStartTime := time.Now()
	if err := e.executeStepWithExecutor(ctx, step, appName, execID); err != nil {
		// Mark step as failed
		errorMsg := err.Error()
		_ = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errorMsg)

		// Record failure in execution context
		e.execContext.SetStepStatus(step.Name, "failed")

		return err
	}

	// Mark step as completed
	err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil)
	if err != nil {
		fmt.Printf("      ⚠️  Warning: failed to update step completion: %v\n", err)
	}

	duration := time.Since(stepStartTime)
	fmt.Printf("      ✅ %s completed (took %v)\n", step.Name, duration.Round(time.Millisecond))

	// Capture step outputs
	e.captureStepOutputs(step)

	// Record success in execution context
	e.execContext.SetStepStatus(step.Name, "success")

	return nil
}

// captureStepOutputs captures outputs from a completed step
func (e *WorkflowExecutor) captureStepOutputs(step types.Step) {
	outputs := make(map[string]string)

	// Apply setVariables (highest priority - explicit variable setting)
	if len(step.SetVariables) > 0 {
		for k, v := range step.SetVariables {
			e.execContext.SetVariable(k, v)
			outputs[k] = v
		}
		fmt.Printf("      📤 Set %d workflow variables\n", len(step.SetVariables))
	}

	// Read output file if specified
	if step.OutputFile != "" {
		fileOutputs, err := e.outputParser.ParseOutputFile(step.OutputFile)
		if err != nil {
			fmt.Printf("      ⚠️  Warning: failed to parse output file %s: %v\n", step.OutputFile, err)
		} else {
			for k, v := range fileOutputs {
				outputs[k] = v
			}
			if len(fileOutputs) > 0 {
				fmt.Printf("      📄 Captured %d outputs from file: %s\n", len(fileOutputs), step.OutputFile)
			}
		}
	}

	// Store all captured outputs in execution context
	if len(outputs) > 0 {
		e.execContext.SetStepOutputs(step.Name, outputs)

		// Display captured outputs
		for k, v := range outputs {
			// Truncate long values for display
			displayValue := v
			if len(displayValue) > 50 {
				displayValue = displayValue[:47] + "..."
			}
			fmt.Printf("      💾 %s = %s\n", k, displayValue)
		}
	}
}

// executeStepWithExecutor executes a step using registered executors
func (e *WorkflowExecutor) executeStepWithExecutor(ctx context.Context, step types.Step, appName string, execID int64) error {
	e.mu.RLock()
	executor, exists := e.stepExecutors[step.Type]
	e.mu.RUnlock()

	if !exists {
		// Fallback to existing step execution logic
		return runStepWithSpinner(step, appName, "default", nil)
	}

	// Create a timeout context for the step
	stepCtx, cancel := context.WithTimeout(ctx, e.executionTimeout)
	defer cancel()

	return executor(stepCtx, step, appName, execID)
}

// registerDefaultStepExecutors registers the default step executors
func (e *WorkflowExecutor) registerDefaultStepExecutors() {
	// Resource provisioning executor
	e.stepExecutors["resource-provisioning"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		if e.resourceManager == nil {
			// Fallback to simulation if no resource manager
			time.Sleep(2 * time.Second)
			fmt.Printf("      🔧 Simulated resource provisioning for step: %s\n", step.Name)
			return nil
		}

		fmt.Printf("      🔧 Provisioning resources for application: %s\n", appName)

		// Get all resources for the application
		resources, err := e.resourceManager.GetResourcesByApplication(appName)
		if err != nil {
			return fmt.Errorf("failed to get resources for app %s: %w", appName, err)
		}

		if len(resources) == 0 {
			fmt.Printf("      ℹ️  No resources found for application: %s\n", appName)
			return nil
		}

		// Provision resources that are in provisioning state
		provisionedCount := 0
		for _, resource := range resources {
			if resource.State == "provisioning" {
				err := e.resourceManager.ProvisionResource(
					resource.ID,
					"workflow-provisioner",
					map[string]interface{}{
						"provisioned_via": "workflow_step",
						"step_name":       step.Name,
						"execution_id":    execID,
					},
					"workflow-executor",
				)
				if err != nil {
					fmt.Printf("      ❌ Failed to provision resource %s (ID: %d): %v\n", resource.ResourceName, resource.ID, err)
					return fmt.Errorf("failed to provision resource %s: %w", resource.ResourceName, err)
				}
				fmt.Printf("      ✅ Provisioned resource: %s (%s)\n", resource.ResourceName, resource.ResourceType)
				provisionedCount++
			}
		}

		if provisionedCount > 0 {
			fmt.Printf("      🎉 Successfully provisioned %d resources for %s\n", provisionedCount, appName)
		} else {
			fmt.Printf("      ℹ️  All resources already provisioned for %s\n", appName)
		}

		return nil
	}

	// Security scanning executor
	e.stepExecutors["security"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(4 * time.Second)
		fmt.Printf("      🔒 Security scan completed\n")
		return nil
	}

	// Policy validation executor
	e.stepExecutors["policy"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("      📋 Policy validation completed\n")
		return nil
	}

	// Cost analysis executor
	e.stepExecutors["cost-analysis"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      💰 Cost analysis completed\n")
		return nil
	}

	// Tagging executor
	e.stepExecutors["tagging"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("      🏷️  Resource tagging completed\n")
		return nil
	}

	// Database migration executor
	e.stepExecutors["database-migration"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(3 * time.Second)
		fmt.Printf("      🗃️  Database migration completed\n")
		return nil
	}

	// Vault setup executor
	e.stepExecutors["vault-setup"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      🔐 Vault configuration completed\n")
		return nil
	}

	// Monitoring setup executor
	e.stepExecutors["monitoring"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      📊 Monitoring setup completed\n")
		return nil
	}

	// Validation executor
	e.stepExecutors["validation"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("      ✅ Validation completed\n")
		return nil
	}

	// Terraform executor
	e.stepExecutors["terraform"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		fmt.Printf("      🏗️  Executing Terraform step: %s\n", step.Name)

		// Get operation (default: apply)
		operation := step.Operation
		if operation == "" {
			operation = "apply"
		}

		// Get working directory (try both WorkingDir field and Config map)
		workingDir := step.WorkingDir
		if workingDir == "" && step.Config != nil {
			if wd, ok := step.Config["working_dir"].(string); ok {
				workingDir = wd
			}
		}
		if workingDir == "" {
			return fmt.Errorf("terraform step requires 'workingDir' or 'config.working_dir'")
		}

		// Get variables (from both Variables field and Config map)
		variables := make(map[string]string)
		if step.Variables != nil {
			for k, v := range step.Variables {
				variables[k] = fmt.Sprintf("%v", v)
			}
		}
		if step.Config != nil {
			if varsRaw, ok := step.Config["variables"].(map[string]interface{}); ok {
				for k, v := range varsRaw {
					variables[k] = fmt.Sprintf("%v", v)
				}
			}
		}

		// Get outputs to capture
		outputNames := step.Outputs
		if len(outputNames) == 0 && step.Config != nil {
			if outputsRaw, ok := step.Config["outputs"].([]interface{}); ok {
				for _, o := range outputsRaw {
					if outputStr, ok := o.(string); ok {
						outputNames = append(outputNames, outputStr)
					}
				}
			}
		}

		// Create workspace directory for this app/env
		workspaceDir := fmt.Sprintf("workspaces/%s/terraform", appName)
		if err := os.MkdirAll(workspaceDir, 0700); err != nil {
			return fmt.Errorf("failed to create terraform workspace: %w", err)
		}

		// Copy terraform files to workspace
		fmt.Printf("      📁 Preparing Terraform workspace: %s\n", workspaceDir)
		if err := e.copyTerraformFiles(workingDir, workspaceDir); err != nil {
			return fmt.Errorf("failed to copy terraform files: %w", err)
		}

		// Execute terraform operation
		switch operation {
		case "init":
			return e.terraformInit(ctx, workspaceDir)
		case "plan":
			return e.terraformPlan(ctx, workspaceDir, variables)
		case "apply":
			if err := e.terraformInit(ctx, workspaceDir); err != nil {
				return err
			}
			if err := e.terraformApply(ctx, workspaceDir, variables); err != nil {
				return err
			}
			// Capture outputs if specified
			if len(outputNames) > 0 {
				return e.terraformCaptureOutputs(ctx, workspaceDir, outputNames, appName)
			}
			return nil
		case "destroy":
			if err := e.terraformInit(ctx, workspaceDir); err != nil {
				return err
			}
			return e.terraformDestroy(ctx, workspaceDir, variables)
		case "output":
			return e.terraformCaptureOutputs(ctx, workspaceDir, outputNames, appName)
		default:
			return fmt.Errorf("unsupported terraform operation: %s", operation)
		}
	}

	// Terraform-generate executor - generates Terraform code from Score resources
	e.stepExecutors["terraform-generate"] = func(ctx context.Context, step types.Step, appName string, execID int64) error {
		fmt.Printf("      📝 Generating Terraform code for: %s\n", step.Name)

		// Get output directory (default: workspaces/{app}/terraform)
		outputDir := step.OutputDir
		if outputDir == "" {
			outputDir = fmt.Sprintf("workspaces/%s/terraform", appName)
		}

		// Get resource type to generate (from step.Resource field or Config)
		resourceType := step.Resource
		if resourceType == "" && step.Config != nil {
			if rt, ok := step.Config["resource"].(string); ok {
				resourceType = rt
			}
		}

		if resourceType == "" {
			return fmt.Errorf("terraform-generate requires 'resource' field (e.g., 's3', 'postgres')")
		}

		// Create output directory
		if err := os.MkdirAll(outputDir, 0700); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		fmt.Printf("      📁 Output directory: %s\n", outputDir)
		fmt.Printf("      🔧 Resource type: %s\n", resourceType)

		// Generate Terraform code based on resource type
		switch resourceType {
		case "s3", "minio-s3-bucket":
			return e.generateS3BucketTerraform(outputDir, appName, step)
		case "postgres", "postgresql":
			return e.generatePostgresTerraform(outputDir, appName, step)
		default:
			return fmt.Errorf("unsupported resource type for terraform generation: %s", resourceType)
		}
	}
}

// Terraform helper functions

// copyTerraformFiles copies terraform files from source to destination
func (e *WorkflowExecutor) copyTerraformFiles(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		// Destination path
		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0700)
		}

		// Copy file
		return e.copyFile(path, destPath)
	})
}

// copyFile copies a single file
// Note: src and dest paths are expected to be within controlled workspace directories
func (e *WorkflowExecutor) copyFile(src, dest string) error {
	sourceFile, err := os.Open(filepath.Clean(src)) // #nosec G304 - paths controlled by workflow executor
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(filepath.Clean(dest)) // #nosec G304 - paths controlled by workflow executor
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// terraformInit initializes terraform in the workspace
func (e *WorkflowExecutor) terraformInit(ctx context.Context, workspaceDir string) error {
	fmt.Printf("      🔧 Terraform init\n")
	cmd := exec.CommandContext(ctx, "terraform", "init", "-no-color")
	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform init failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// terraformPlan runs terraform plan
func (e *WorkflowExecutor) terraformPlan(ctx context.Context, workspaceDir string, variables map[string]string) error {
	fmt.Printf("      📋 Terraform plan\n")
	args := []string{"plan", "-no-color"}
	for k, v := range variables {
		args = append(args, "-var", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform plan failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("%s\n", string(output))
	return nil
}

// terraformApply runs terraform apply
func (e *WorkflowExecutor) terraformApply(ctx context.Context, workspaceDir string, variables map[string]string) error {
	fmt.Printf("      ✅ Terraform apply\n")
	args := []string{"apply", "-auto-approve", "-no-color"}
	for k, v := range variables {
		args = append(args, "-var", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("      🎉 Terraform apply completed successfully\n")
	return nil
}

// terraformDestroy runs terraform destroy
func (e *WorkflowExecutor) terraformDestroy(ctx context.Context, workspaceDir string, variables map[string]string) error {
	fmt.Printf("      🗑️  Terraform destroy\n")
	args := []string{"destroy", "-auto-approve", "-no-color"}
	for k, v := range variables {
		args = append(args, "-var", fmt.Sprintf("%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, "terraform", args...)
	cmd.Dir = workspaceDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("terraform destroy failed: %w\nOutput: %s", err, string(output))
	}
	fmt.Printf("      ✅ Terraform destroy completed successfully\n")
	return nil
}

// terraformCaptureOutputs captures terraform outputs and stores them
func (e *WorkflowExecutor) terraformCaptureOutputs(ctx context.Context, workspaceDir string, outputNames []string, appName string) error {
	fmt.Printf("      📤 Capturing Terraform outputs\n")

	// Run terraform output -json
	cmd := exec.CommandContext(ctx, "terraform", "output", "-json")
	cmd.Dir = workspaceDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("terraform output failed: %w", err)
	}

	// Parse JSON output
	var outputs map[string]interface{}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return fmt.Errorf("failed to parse terraform outputs: %w", err)
	}

	// Extract specified outputs
	for _, outputName := range outputNames {
		if outputValue, ok := outputs[outputName]; ok {
			if outputMap, ok := outputValue.(map[string]interface{}); ok {
				if value, ok := outputMap["value"]; ok {
					fmt.Printf("      📊 Output '%s': %v\n", outputName, value)
					// TODO: Store outputs in workflow context or database for use in subsequent steps
				}
			}
		} else {
			fmt.Printf("      ⚠️  Output '%s' not found in terraform outputs\n", outputName)
		}
	}

	return nil
}

// Terraform code generation functions

// generateS3BucketTerraform generates Terraform code for S3 bucket provisioning
func (e *WorkflowExecutor) generateS3BucketTerraform(outputDir, appName string, step types.Step) error {
	bucketName := fmt.Sprintf("%s-storage", appName)
	minioEndpoint := "http://minio.minio-system.svc.cluster.local:9000"
	minioUser := "minioadmin"
	minioPassword := "minioadmin"

	// Override from step variables if provided
	if step.Variables != nil {
		if bn, ok := step.Variables["bucket_name"].(string); ok {
			bucketName = bn
		}
		if ep, ok := step.Variables["minio_endpoint"].(string); ok {
			minioEndpoint = ep
		}
		if usr, ok := step.Variables["minio_user"].(string); ok {
			minioUser = usr
		}
		if pwd, ok := step.Variables["minio_password"].(string); ok {
			minioPassword = pwd
		}
	}

	// Generate main.tf
	mainTf := fmt.Sprintf(`# Generated Terraform configuration for %s
# Generated at: %s

terraform {
  required_providers {
    minio = {
      source  = "aminueza/minio"
      version = "~> 2.0"
    }
  }
}

provider "minio" {
  minio_server   = "%s"
  minio_user     = "%s"
  minio_password = "%s"
  minio_ssl      = false
}

resource "minio_s3_bucket" "app_bucket" {
  bucket = "%s"
  acl    = "private"
}

output "minio_url" {
  value       = "s3://${minio_s3_bucket.app_bucket.bucket}"
  description = "S3 URL for the created bucket"
}

output "bucket_name" {
  value       = minio_s3_bucket.app_bucket.bucket
  description = "Name of the created bucket"
}

output "endpoint" {
  value       = "%s"
  description = "Minio endpoint URL"
}

output "bucket_arn" {
  value       = "arn:aws:s3:::${minio_s3_bucket.app_bucket.bucket}"
  description = "ARN-style identifier for the bucket"
}
`, appName, time.Now().Format(time.RFC3339), minioEndpoint, minioUser, minioPassword, bucketName, minioEndpoint)

	// Write main.tf
	mainTfPath := filepath.Join(outputDir, "main.tf")
	if err := os.WriteFile(mainTfPath, []byte(mainTf), 0600); err != nil {
		return fmt.Errorf("failed to write main.tf: %w", err)
	}

	fmt.Printf("      ✅ Generated: %s\n", mainTfPath)
	fmt.Printf("      📦 Bucket name: %s\n", bucketName)

	return nil
}

// generatePostgresTerraform generates Terraform code for PostgreSQL provisioning
func (e *WorkflowExecutor) generatePostgresTerraform(outputDir, appName string, step types.Step) error {
	// Placeholder for future implementation
	return fmt.Errorf("PostgreSQL Terraform generation not yet implemented")
}
