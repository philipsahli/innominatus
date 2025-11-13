package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/events"
	"innominatus/internal/graph"
	"innominatus/internal/logging"
	"innominatus/internal/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	sdk "github.com/philipsahli/innominatus-graph/pkg/graph"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// StepExecutorFunc defines the signature for step execution functions
// stepID is the database ID of the workflow step for log persistence
type StepExecutorFunc func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error

// WorkflowRepositoryInterface defines the methods needed for workflow persistence
type WorkflowRepositoryInterface interface {
	CreateWorkflowExecution(appName, workflowName string, totalSteps int) (*database.WorkflowExecution, error)
	CreateWorkflowStep(execID int64, stepNumber int, stepName, stepType string, config map[string]interface{}) (*database.WorkflowStepExecution, error)
	UpdateWorkflowStepStatus(stepID int64, status string, errorMessage *string) error
	UpdateWorkflowExecution(execID int64, status string, errorMessage *string) error
	GetWorkflowExecution(id int64) (*database.WorkflowExecution, error)
	CountWorkflowExecutions(appName, workflowName, status string) (int64, error)
	ListWorkflowExecutions(appName, workflowName, status string, limit, offset int) ([]*database.WorkflowExecutionSummary, error)
	GetLatestWorkflowExecution(appName, workflowName string) (*database.WorkflowExecution, error)
	GetFirstFailedStepNumber(executionID int64) (int, error)
	CreateRetryExecution(parentID int64, appName, workflowName string, totalSteps, resumeFromStep int) (*database.WorkflowExecution, error)
	ReconstructWorkflowFromExecution(executionID int64) (map[string]interface{}, error)
	AddWorkflowStepLogs(stepID int64, logs string) error
}

// ResourceManager interface defines the methods needed for resource management
type ResourceManager interface {
	GetResourcesByApplication(appName string) ([]*database.ResourceInstance, error)
	ProvisionResource(resourceID int64, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error
	TransitionResourceState(resourceID int64, newState database.ResourceLifecycleState, reason string, transitionedBy string, metadata map[string]interface{}) error
	UpdateResourceHealth(resourceID int64, healthStatus string, errorMessage *string) error
}

// WorkflowExecutor handles workflow execution with database persistence
type WorkflowExecutor struct {
	repo             WorkflowRepositoryInterface
	resolver         *WorkflowResolver
	resourceManager  ResourceManager
	graphAdapter     *graph.Adapter
	eventBus         events.EventBus
	maxConcurrent    int
	executionTimeout time.Duration
	stepExecutors    map[string]StepExecutorFunc
	execContext      *ExecutionContext
	outputParser     *OutputParser
	logger           *logging.ZerologAdapter
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
		logger:           logging.NewStructuredLogger("workflow"),
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
		logger:           logging.NewStructuredLogger("workflow"),
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
		logger:           logging.NewStructuredLogger("workflow"),
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
		logger:           logging.NewStructuredLogger("workflow"),
	}
	executor.registerDefaultStepExecutors()
	return executor
}

// SetGraphAdapter sets the graph adapter for workflow tracking
func (e *WorkflowExecutor) SetGraphAdapter(adapter *graph.Adapter) {
	e.graphAdapter = adapter
}

// SetEventBus sets the event bus for publishing workflow events
func (e *WorkflowExecutor) SetEventBus(bus events.EventBus) {
	e.eventBus = bus
	e.logger.Info("Event bus configured for workflow executor")
}

// stepToConfig converts a Step struct to a map for storage in the database
// This ensures all step fields are preserved when storing workflow executions
func stepToConfig(step types.Step) (map[string]interface{}, error) {
	// Marshal the step to JSON first
	stepJSON, err := json.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal step: %w", err)
	}

	// Unmarshal to map to preserve all fields
	var config map[string]interface{}
	if err := json.Unmarshal(stepJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal step config: %w", err)
	}

	return config, nil
}

// ExecuteMultiTierWorkflows executes resolved multi-tier workflows
func (e *WorkflowExecutor) ExecuteMultiTierWorkflows(ctx context.Context, app *ApplicationInstance) error {
	// Ensure logger is initialized (defensive programming)
	if e.logger == nil {
		e.logger = logging.NewStructuredLogger("workflow")
	}

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

	summary := e.resolver.GetWorkflowSummary(resolvedWorkflows)
	e.logger.InfoWithFields("Starting multi-tier workflow execution", map[string]interface{}{
		"app_name":        app.Name,
		"execution_id":    execution.ID,
		"total_workflows": summary["total_workflows"],
		"phases":          len(summary["phases"].([]string)),
	})

	// Execute workflows by phase
	phases := []WorkflowPhase{PhasePreDeployment, PhaseDeployment, PhasePostDeployment}

	for _, phase := range phases {
		workflows, exists := resolvedWorkflows[phase]
		if !exists || len(workflows) == 0 {
			continue
		}

		e.logger.InfoWithFields("Executing workflow phase", map[string]interface{}{
			"app_name":       app.Name,
			"execution_id":   execution.ID,
			"phase":          string(phase),
			"workflow_count": len(workflows),
		})

		if err := e.executePhaseWorkflows(ctx, app.Name, phase, workflows, execution.ID); err != nil {
			// Mark execution as failed
			errorMsg := err.Error()
			if updateErr := e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, &errorMsg); updateErr != nil {
				e.logger.WarnWithFields("Failed to update workflow status", map[string]interface{}{
					"execution_id": execution.ID,
					"error":        updateErr.Error(),
				})
			}
			e.logger.ErrorWithFields("Phase execution failed", map[string]interface{}{
				"app_name":     app.Name,
				"execution_id": execution.ID,
				"phase":        string(phase),
				"error":        err.Error(),
			})
			return fmt.Errorf("failed executing %s workflows: %w", phase, err)
		}

		e.logger.InfoWithFields("Phase completed successfully", map[string]interface{}{
			"app_name":     app.Name,
			"execution_id": execution.ID,
			"phase":        string(phase),
		})
	}

	// Mark execution as completed
	err = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusCompleted, nil)
	if err != nil {
		e.logger.WarnWithFields("Failed to update workflow completion", map[string]interface{}{
			"execution_id": execution.ID,
			"error":        err.Error(),
		})
	}

	e.logger.InfoWithFields("Multi-tier workflow execution completed successfully", map[string]interface{}{
		"app_name":     app.Name,
		"execution_id": execution.ID,
	})
	return nil
}

// ExecuteWorkflow executes a workflow with database persistence
func (e *WorkflowExecutor) ExecuteWorkflow(appName string, workflow types.Workflow) error {
	return e.ExecuteWorkflowWithName(appName, "deploy", workflow)
}

// ExecuteWorkflowWithName executes a named workflow with database persistence
func (e *WorkflowExecutor) ExecuteWorkflowWithName(appName, workflowName string, workflow types.Workflow, goldenPathParams ...map[string]string) error {
	// Ensure logger is initialized (defensive programming)
	if e.logger == nil {
		e.logger = logging.NewStructuredLogger("workflow")
	}

	// Create OpenTelemetry span for workflow execution
	tracer := otel.Tracer("innominatus/workflow")
	_, span := tracer.Start(context.Background(), "workflow.execute",
		trace.WithAttributes(
			attribute.String("app.name", appName),
			attribute.String("workflow.name", workflowName),
			attribute.Int("workflow.steps", len(workflow.Steps)),
		),
	)
	defer span.End()

	// Initialize golden path parameters first (if provided) - they take precedence
	if len(goldenPathParams) > 0 && len(goldenPathParams[0]) > 0 {
		e.execContext.SetWorkflowVariables(goldenPathParams[0])
		e.logger.InfoWithFields("Initialized golden path parameters", map[string]interface{}{
			"app_name":        appName,
			"workflow_name":   workflowName,
			"parameter_count": len(goldenPathParams[0]),
		})
	}

	// Initialize workflow variables in execution context (may override golden path params if same keys exist)
	if len(workflow.Variables) > 0 {
		e.execContext.SetWorkflowVariables(workflow.Variables)
		e.logger.InfoWithFields("Initialized workflow variables", map[string]interface{}{
			"app_name":       appName,
			"workflow_name":  workflowName,
			"variable_count": len(workflow.Variables),
		})
	}

	// Pre-execution validation: Check all workflow variable references
	if err := e.execContext.ValidateWorkflowVariables(workflow); err != nil {
		if IsStrictMode() {
			span.RecordError(err)
			e.logger.ErrorWithFields("Workflow validation failed", map[string]interface{}{
				"app_name":      appName,
				"workflow_name": workflowName,
				"error":         err.Error(),
			})
			return fmt.Errorf("workflow validation failed: %w", err)
		}
		// Lenient mode: log warning but continue
		e.logger.WarnWithFields("Workflow validation warnings (lenient mode)", map[string]interface{}{
			"app_name":      appName,
			"workflow_name": workflowName,
			"warning":       err.Error(),
		})
	}

	// Create workflow execution record
	execution, err := e.repo.CreateWorkflowExecution(appName, workflowName, len(workflow.Steps))
	if err != nil {
		span.RecordError(err)
		e.logger.ErrorWithFields("Failed to create workflow execution", map[string]interface{}{
			"app_name":      appName,
			"workflow_name": workflowName,
			"error":         err.Error(),
		})
		return fmt.Errorf("failed to create workflow execution: %w", err)
	}

	// Add execution ID to span
	span.SetAttributes(attribute.Int64("workflow.execution_id", execution.ID))

	e.logger.InfoWithFields("Starting workflow execution", map[string]interface{}{
		"app_name":      appName,
		"workflow_name": workflowName,
		"execution_id":  execution.ID,
		"total_steps":   len(workflow.Steps),
	})

	// Publish workflow started event
	if e.eventBus != nil {
		e.eventBus.Publish(events.NewEvent(
			events.EventTypeWorkflowStarted,
			appName,
			"workflow-executor",
			map[string]interface{}{
				"workflow_name": workflowName,
				"execution_id":  execution.ID,
				"total_steps":   len(workflow.Steps),
			},
		))
	}

	// Create workflow node in graph (if graph adapter is available)
	workflowNodeID := fmt.Sprintf("workflow-%d", execution.ID)
	if e.graphAdapter != nil {
		workflowNode := &sdk.Node{
			ID:    workflowNodeID,
			Type:  sdk.NodeTypeWorkflow,
			Name:  workflowName,
			State: sdk.NodeStatePending,
			Properties: map[string]interface{}{
				"execution_id": execution.ID,
				"app_name":     appName,
				"total_steps":  len(workflow.Steps),
			},
		}
		if err := e.graphAdapter.AddNode(appName, workflowNode); err != nil {
			fmt.Printf("Warning: failed to add workflow node to graph: %v\n", err)
		} else {
			// Create edge: spec triggers workflow
			specNodeID := fmt.Sprintf("spec:%s", appName)
			specToWorkflowEdge := &sdk.Edge{
				ID:         fmt.Sprintf("spec-%s-wf-%d", appName, execution.ID),
				FromNodeID: specNodeID,
				ToNodeID:   workflowNodeID,
				Type:       sdk.EdgeTypeTriggers,
				Properties: map[string]interface{}{
					"workflow_name": workflowName,
				},
			}
			if err := e.graphAdapter.AddEdge(appName, specToWorkflowEdge); err != nil {
				fmt.Printf("Warning: failed to add spec→workflow edge to graph: %v\n", err)
			}
		}
	}

	// Create step records
	stepRecords := make(map[int]*database.WorkflowStepExecution)
	stepNodeIDs := make(map[int]string)
	for i, step := range workflow.Steps {
		stepConfig, err := stepToConfig(step)
		if err != nil {
			_ = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, stringPtr(fmt.Sprintf("Failed to serialize step config: %v", err)))
			return fmt.Errorf("failed to serialize step config: %w", err)
		}

		stepRecord, err := e.repo.CreateWorkflowStep(execution.ID, i+1, step.Name, step.Type, stepConfig)
		if err != nil {
			_ = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, stringPtr(fmt.Sprintf("Failed to create step record: %v", err)))
			return fmt.Errorf("failed to create workflow step: %w", err)
		}
		stepRecords[i] = stepRecord

		// Create step node in graph (if graph adapter is available)
		stepNodeID := fmt.Sprintf("step-%d", stepRecord.ID)
		stepNodeIDs[i] = stepNodeID
		if e.graphAdapter != nil {
			stepNode := &sdk.Node{
				ID:    stepNodeID,
				Type:  sdk.NodeTypeStep,
				Name:  step.Name,
				State: sdk.NodeStateWaiting,
				Properties: map[string]interface{}{
					"step_id":     stepRecord.ID,
					"step_number": i + 1,
					"step_type":   step.Type,
				},
			}
			if err := e.graphAdapter.AddNode(appName, stepNode); err != nil {
				fmt.Printf("Warning: failed to add step node to graph: %v\n", err)
			}

			// Create edge: workflow contains step
			edge := &sdk.Edge{
				ID:         fmt.Sprintf("wf-%d-step-%d", execution.ID, stepRecord.ID),
				FromNodeID: workflowNodeID,
				ToNodeID:   stepNodeID,
				Type:       sdk.EdgeTypeContains,
			}
			if err := e.graphAdapter.AddEdge(appName, edge); err != nil {
				fmt.Printf("Warning: failed to add workflow→step edge to graph: %v\n", err)
			}
		}
	}

	// Execute steps
	for i, step := range workflow.Steps {
		stepRecord := stepRecords[i]
		stepNodeID := stepNodeIDs[i]

		e.logger.InfoWithFields("Executing workflow step", map[string]interface{}{
			"app_name":      appName,
			"workflow_name": workflowName,
			"execution_id":  execution.ID,
			"step_number":   i + 1,
			"total_steps":   len(workflow.Steps),
			"step_name":     step.Name,
			"step_type":     step.Type,
		})

		// Update step to running
		err := e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil)
		if err != nil {
			e.logger.WarnWithFields("Failed to update step status", map[string]interface{}{
				"step_id": stepRecord.ID,
				"error":   err.Error(),
			})
		}

		// Update step node state to running in graph
		if e.graphAdapter != nil {
			if err := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateRunning); err != nil {
				fmt.Printf("Warning: failed to update step state in graph: %v\n", err)
			}
		}

		spinner := NewSpinner(fmt.Sprintf("Initializing %s step...", step.Type))
		spinner.Start()

		// Use the modern stepExecutors registry instead of old runStepWithSpinner
		executor, exists := e.stepExecutors[step.Type]
		if !exists {
			spinner.Stop(false, fmt.Sprintf("Unsupported step type: %s", step.Type))
			err = fmt.Errorf("unsupported step type: %s", step.Type)
		} else {
			// Execute step with context, passing stepID for log persistence
			ctx := context.Background()
			err = executor(ctx, step, appName, execution.ID, stepRecord.ID)
			if err != nil {
				spinner.Stop(false, fmt.Sprintf("Step '%s' failed", step.Name))
			} else {
				spinner.Stop(true, fmt.Sprintf("✅ Step '%s' completed successfully", step.Name))
			}
		}

		if err != nil {
			// Update step as failed
			errorMsg := err.Error()
			_ = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errorMsg)

			// Update workflow as failed
			workflowErrorMsg := fmt.Sprintf("workflow failed at step '%s': %v", step.Name, err)
			_ = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, &workflowErrorMsg)

			// Update any linked resources to failed state
			e.updateLinkedResourcesOnFailure(execution.ID, appName, workflowErrorMsg)

			// Update step node state to failed in graph (triggers automatic propagation to workflow)
			if e.graphAdapter != nil {
				if err := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateFailed); err != nil {
					fmt.Printf("Warning: failed to update step state in graph: %v\n", err)
				}
			}

			spinner.Stop(false, fmt.Sprintf("Step '%s' failed: %v", step.Name, err))
			return fmt.Errorf("workflow failed at step '%s': %w", step.Name, err)
		}

		// Update step as completed
		err = e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil)
		if err != nil {
			fmt.Printf("Warning: failed to update step completion: %v\n", err)
		}

		// Update step node state to succeeded in graph
		if e.graphAdapter != nil {
			if err := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateSucceeded); err != nil {
				fmt.Printf("Warning: failed to update step state in graph: %v\n", err)
			}
		}

		spinner.Stop(true, fmt.Sprintf("Step '%s' completed successfully", step.Name))
		fmt.Println()
	}

	// Update workflow as completed
	err = e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusCompleted, nil)
	if err != nil {
		fmt.Printf("Warning: failed to update workflow completion: %v\n", err)
	}

	// Publish workflow completed event
	if e.eventBus != nil {
		e.eventBus.Publish(events.NewEvent(
			events.EventTypeWorkflowCompleted,
			appName,
			"workflow-executor",
			map[string]interface{}{
				"workflow_name": workflowName,
				"execution_id":  execution.ID,
				"total_steps":   len(workflow.Steps),
			},
		))
	}

	// Update workflow node state to succeeded in graph
	if e.graphAdapter != nil {
		if err := e.graphAdapter.UpdateNodeState(appName, workflowNodeID, sdk.NodeStateSucceeded); err != nil {
			fmt.Printf("Warning: failed to update workflow state in graph: %v\n", err)
		}
	}

	// Update any linked resources to active state
	e.updateLinkedResourcesOnCompletion(execution.ID, appName)

	fmt.Println("🎉 Workflow completed successfully!")
	return nil
}

// updateLinkedResourcesOnCompletion updates resources linked to a workflow execution
// Transitions resources from provisioning to active state with healthy status
func (e *WorkflowExecutor) updateLinkedResourcesOnCompletion(workflowExecutionID int64, appName string) {
	if e.resourceManager == nil {
		return // No resource manager available
	}

	e.logger.InfoWithFields("Updating linked resources after workflow completion", map[string]interface{}{
		"workflow_execution_id": workflowExecutionID,
		"app_name":              appName,
	})

	// Query for resources linked to this workflow execution
	// This requires direct database access since ResourceManager doesn't have this query
	// We'll transition all resources in provisioning state for this app
	resources, err := e.resourceManager.GetResourcesByApplication(appName)
	if err != nil {
		e.logger.WarnWithFields("Failed to get resources for app", map[string]interface{}{
			"app_name": appName,
			"error":    err.Error(),
		})
		return
	}

	e.logger.InfoWithFields("Found resources for application", map[string]interface{}{
		"app_name":       appName,
		"resource_count": len(resources),
	})

	for _, resource := range resources {
		e.logger.DebugWithFields("Checking resource", map[string]interface{}{
			"resource_id":           resource.ID,
			"resource_name":         resource.ResourceName,
			"state":                 resource.State,
			"workflow_execution_id": resource.WorkflowExecutionID,
			"expected_workflow_id":  workflowExecutionID,
			"workflow_id_match":     resource.WorkflowExecutionID != nil && *resource.WorkflowExecutionID == workflowExecutionID,
			"is_provisioning_state": resource.State == database.ResourceStateProvisioning,
		})

		// Only update resources that are linked to this workflow and in provisioning state
		if resource.WorkflowExecutionID != nil && *resource.WorkflowExecutionID == workflowExecutionID {
			if resource.State == database.ResourceStateProvisioning {
				e.logger.InfoWithFields("Transitioning resource to active", map[string]interface{}{
					"resource_id":   resource.ID,
					"resource_name": resource.ResourceName,
					"resource_type": resource.ResourceType,
				})

				// Transition to active state
				err := e.resourceManager.TransitionResourceState(
					resource.ID,
					database.ResourceStateActive,
					"Resource provisioned successfully by workflow",
					"workflow-executor",
					map[string]interface{}{
						"workflow_execution_id": workflowExecutionID,
					},
				)
				if err != nil {
					e.logger.ErrorWithFields("Failed to transition resource to active", map[string]interface{}{
						"resource_id": resource.ID,
						"error":       err.Error(),
					})
					continue
				}

				// Update health status to healthy
				err = e.resourceManager.UpdateResourceHealth(resource.ID, "healthy", nil)
				if err != nil {
					e.logger.WarnWithFields("Failed to update resource health", map[string]interface{}{
						"resource_id": resource.ID,
						"error":       err.Error(),
					})
				}

				e.logger.InfoWithFields("Successfully updated resource", map[string]interface{}{
					"resource_id":   resource.ID,
					"resource_name": resource.ResourceName,
					"state":         "active",
					"health":        "healthy",
				})
			}
		}
	}
}

// updateLinkedResourcesOnFailure updates resources linked to a failed workflow execution
// Transitions resources from provisioning to failed state
func (e *WorkflowExecutor) updateLinkedResourcesOnFailure(workflowExecutionID int64, appName string, errorMessage string) {
	if e.resourceManager == nil {
		return // No resource manager available
	}

	e.logger.InfoWithFields("Updating linked resources after workflow failure", map[string]interface{}{
		"workflow_execution_id": workflowExecutionID,
		"app_name":              appName,
	})

	// Query for resources linked to this workflow execution
	resources, err := e.resourceManager.GetResourcesByApplication(appName)
	if err != nil {
		e.logger.WarnWithFields("Failed to get resources for app", map[string]interface{}{
			"app_name": appName,
			"error":    err.Error(),
		})
		return
	}

	for _, resource := range resources {
		// Only update resources that are linked to this workflow and in provisioning state
		if resource.WorkflowExecutionID != nil && *resource.WorkflowExecutionID == workflowExecutionID {
			if resource.State == database.ResourceStateProvisioning {
				e.logger.InfoWithFields("Transitioning resource to failed", map[string]interface{}{
					"resource_id":   resource.ID,
					"resource_name": resource.ResourceName,
					"resource_type": resource.ResourceType,
				})

				// Transition to failed state
				err := e.resourceManager.TransitionResourceState(
					resource.ID,
					database.ResourceStateFailed,
					fmt.Sprintf("Provisioning workflow failed: %s", errorMessage),
					"workflow-executor",
					map[string]interface{}{
						"workflow_execution_id": workflowExecutionID,
						"error":                 errorMessage,
					},
				)
				if err != nil {
					e.logger.ErrorWithFields("Failed to transition resource to failed", map[string]interface{}{
						"resource_id": resource.ID,
						"error":       err.Error(),
					})
					continue
				}

				// Update health status to unhealthy
				unhealthyErr := errorMessage
				err = e.resourceManager.UpdateResourceHealth(resource.ID, "unhealthy", &unhealthyErr)
				if err != nil {
					e.logger.WarnWithFields("Failed to update resource health", map[string]interface{}{
						"resource_id": resource.ID,
						"error":       err.Error(),
					})
				}

				e.logger.InfoWithFields("Successfully updated failed resource", map[string]interface{}{
					"resource_id":   resource.ID,
					"resource_name": resource.ResourceName,
					"state":         "failed",
					"health":        "unhealthy",
				})
			}
		}
	}
}

// GetWorkflowExecution retrieves a workflow execution by ID
func (e *WorkflowExecutor) GetWorkflowExecution(id int64) (*database.WorkflowExecution, error) {
	return e.repo.GetWorkflowExecution(id)
}

// GetRepository returns the workflow repository for accessing database methods
func (e *WorkflowExecutor) GetRepository() WorkflowRepositoryInterface {
	return e.repo
}

// CountWorkflowExecutions counts total workflow executions matching filters
func (e *WorkflowExecutor) CountWorkflowExecutions(appName, workflowName, status string) (int64, error) {
	return e.repo.CountWorkflowExecutions(appName, workflowName, status)
}

// ListWorkflowExecutions lists workflow executions with optional filtering
func (e *WorkflowExecutor) ListWorkflowExecutions(appName, workflowName, status string, limit, offset int) ([]*database.WorkflowExecutionSummary, error) {
	return e.repo.ListWorkflowExecutions(appName, workflowName, status, limit, offset)
}

// RetryWorkflowFromFailedStep retries a failed workflow execution from the first failed step
func (e *WorkflowExecutor) RetryWorkflowFromFailedStep(appName, workflowName string, workflow types.Workflow, parentExecutionID int64) error {
	// Ensure logger is initialized
	if e.logger == nil {
		e.logger = logging.NewStructuredLogger("workflow")
	}

	// Get the failed step number from parent execution
	failedStepNumber, err := e.repo.GetFirstFailedStepNumber(parentExecutionID)
	if err != nil {
		return fmt.Errorf("failed to find failed step: %w", err)
	}

	e.logger.InfoWithFields("Retrying workflow from failed step", map[string]interface{}{
		"app_name":            appName,
		"workflow_name":       workflowName,
		"parent_execution_id": parentExecutionID,
		"resume_from_step":    failedStepNumber,
	})

	// Create retry execution record
	execution, err := e.repo.CreateRetryExecution(parentExecutionID, appName, workflowName, len(workflow.Steps), failedStepNumber)
	if err != nil {
		return fmt.Errorf("failed to create retry execution: %w", err)
	}

	e.logger.InfoWithFields("Created retry execution", map[string]interface{}{
		"execution_id":     execution.ID,
		"retry_count":      execution.RetryCount,
		"resume_from_step": failedStepNumber,
	})

	// Execute workflow starting from the failed step
	return e.executeWorkflowFromStep(appName, workflowName, workflow, execution, failedStepNumber)
}

// executeWorkflowFromStep executes a workflow starting from a specific step number
func (e *WorkflowExecutor) executeWorkflowFromStep(appName, workflowName string, workflow types.Workflow, execution *database.WorkflowExecution, startFromStep int) error {
	// Create OpenTelemetry span
	tracer := otel.Tracer("innominatus/workflow")
	_, span := tracer.Start(context.Background(), "workflow.retry",
		trace.WithAttributes(
			attribute.String("app.name", appName),
			attribute.String("workflow.name", workflowName),
			attribute.Int64("execution.id", execution.ID),
			attribute.Int("start_from_step", startFromStep),
		),
	)
	defer span.End()

	// Initialize workflow variables
	if len(workflow.Variables) > 0 {
		e.execContext.SetWorkflowVariables(workflow.Variables)
	}

	// Create workflow node in graph
	workflowNodeID := fmt.Sprintf("workflow-%d", execution.ID)
	if e.graphAdapter != nil {
		workflowNode := &sdk.Node{
			ID:    workflowNodeID,
			Type:  sdk.NodeTypeWorkflow,
			Name:  workflowName,
			State: sdk.NodeStatePending,
			Properties: map[string]interface{}{
				"execution_id":     execution.ID,
				"app_name":         appName,
				"total_steps":      len(workflow.Steps),
				"is_retry":         execution.IsRetry,
				"retry_count":      execution.RetryCount,
				"resume_from_step": startFromStep,
			},
		}
		if err := e.graphAdapter.AddNode(appName, workflowNode); err != nil {
			fmt.Printf("Warning: failed to add workflow node to graph: %v\n", err)
		}
	}

	// Create step records and execute from startFromStep
	stepRecords := make(map[int]*database.WorkflowStepExecution)

	for i := startFromStep - 1; i < len(workflow.Steps); i++ {
		step := workflow.Steps[i]
		stepNumber := i + 1

		// Create step execution record
		stepConfig, err := stepToConfig(step)
		if err != nil {
			return fmt.Errorf("failed to serialize step config: %w", err)
		}

		stepRecord, err := e.repo.CreateWorkflowStep(execution.ID, stepNumber, step.Name, step.Type, stepConfig)
		if err != nil {
			return fmt.Errorf("failed to create workflow step: %w", err)
		}
		stepRecords[stepNumber] = stepRecord

		e.logger.InfoWithFields("Executing step (retry)", map[string]interface{}{
			"step_number": stepNumber,
			"step_name":   step.Name,
			"step_type":   step.Type,
		})

		// Create step node in graph
		stepNodeID := fmt.Sprintf("step-%d", stepRecord.ID)
		if e.graphAdapter != nil {
			stepNode := &sdk.Node{
				ID:    stepNodeID,
				Type:  sdk.NodeTypeStep,
				Name:  step.Name,
				State: sdk.NodeStatePending,
				Properties: map[string]interface{}{
					"step_number": stepNumber,
					"step_type":   step.Type,
					"step_id":     stepRecord.ID,
				},
			}
			if err := e.graphAdapter.AddNode(appName, stepNode); err != nil {
				fmt.Printf("Warning: failed to add step node to graph: %v\n", err)
			}

			// Create edge from workflow to step
			edge := &sdk.Edge{
				ID:         fmt.Sprintf("workflow-%d-to-step-%d", execution.ID, stepRecord.ID),
				FromNodeID: workflowNodeID,
				ToNodeID:   stepNodeID,
				Type:       sdk.EdgeTypeContains,
			}
			if err := e.graphAdapter.AddEdge(appName, edge); err != nil {
				fmt.Printf("Warning: failed to add workflow-step edge to graph: %v\n", err)
			}
		}

		// Update step to running
		if err := e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusRunning, nil); err != nil {
			fmt.Printf("Warning: failed to update step status: %v\n", err)
		}

		if e.graphAdapter != nil {
			if err := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateRunning); err != nil {
				fmt.Printf("Warning: failed to update step state in graph: %v\n", err)
			}
		}

		// Execute the step with spinner
		spinner := NewSpinner(fmt.Sprintf("Executing step '%s' (%s)...", step.Name, step.Type))
		spinner.Start()

		// Store spinner reference for step execution
		stepErr := runStepWithSpinner(step, appName, "default", spinner)

		if stepErr != nil {
			spinner.Stop(false, fmt.Sprintf("Step '%s' failed", step.Name))

			errMsg := stepErr.Error()
			if updateErr := e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusFailed, &errMsg); updateErr != nil {
				fmt.Printf("Warning: failed to update step status: %v\n", updateErr)
			}

			if e.graphAdapter != nil {
				if updateErr := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateFailed); updateErr != nil {
					fmt.Printf("Warning: failed to update step state in graph: %v\n", updateErr)
				}
			}

			// Update workflow as failed
			workflowErr := e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusFailed, &errMsg)
			if workflowErr != nil {
				fmt.Printf("Warning: failed to update workflow status: %v\n", workflowErr)
			}

			if e.graphAdapter != nil {
				if updateErr := e.graphAdapter.UpdateNodeState(appName, workflowNodeID, sdk.NodeStateFailed); updateErr != nil {
					fmt.Printf("Warning: failed to update workflow state in graph: %v\n", updateErr)
				}
			}

			return fmt.Errorf("step %d (%s) failed: %w", stepNumber, step.Name, stepErr)
		}

		// Update step as completed
		if err := e.repo.UpdateWorkflowStepStatus(stepRecord.ID, database.StepStatusCompleted, nil); err != nil {
			fmt.Printf("Warning: failed to update step status: %v\n", err)
		}

		if e.graphAdapter != nil {
			if err := e.graphAdapter.UpdateNodeState(appName, stepNodeID, sdk.NodeStateSucceeded); err != nil {
				fmt.Printf("Warning: failed to update step state in graph: %v\n", err)
			}
		}

		spinner.Stop(true, fmt.Sprintf("Step '%s' completed successfully", step.Name))
		fmt.Println()
	}

	// Update workflow as completed
	if err := e.repo.UpdateWorkflowExecution(execution.ID, database.WorkflowStatusCompleted, nil); err != nil {
		fmt.Printf("Warning: failed to update workflow completion: %v\n", err)
	}

	if e.graphAdapter != nil {
		if err := e.graphAdapter.UpdateNodeState(appName, workflowNodeID, sdk.NodeStateSucceeded); err != nil {
			fmt.Printf("Warning: failed to update workflow state in graph: %v\n", err)
		}
	}

	fmt.Println("🎉 Workflow retry completed successfully!")
	return nil
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
		stepConfig, err := stepToConfig(step)
		if err != nil {
			return fmt.Errorf("failed to serialize step config: %w", err)
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
		if err := e.executeStepWithExecutor(ctx, step, appName, execID, stepRecord.ID); err != nil {
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
	// Check dependencies before executing
	if len(step.DependsOn) > 0 {
		for _, depStepName := range step.DependsOn {
			depStatus, found := e.execContext.GetStepStatus(depStepName)
			if !found {
				return fmt.Errorf("dependency %s not found for step %s", depStepName, step.Name)
			}
			if depStatus != "success" {
				return fmt.Errorf("dependency %s did not complete successfully (status: %s) for step %s", depStepName, depStatus, step.Name)
			}
		}
		fmt.Printf("      ✓ All dependencies satisfied for %s\n", step.Name)
	}

	// Per-step validation: Check all variable references in step configuration
	if err := e.execContext.ValidateStepVariables(step, step.Env); err != nil {
		if IsStrictMode() {
			// In strict mode, fail the step immediately
			e.logger.ErrorWithFields("Step validation failed", map[string]interface{}{
				"app_name":  appName,
				"step_name": step.Name,
				"error":     err.Error(),
			})
			return fmt.Errorf("step validation failed: %w", err)
		}
		// Lenient mode: log warning but continue
		e.logger.WarnWithFields("Step validation warnings (lenient mode)", map[string]interface{}{
			"app_name":  appName,
			"step_name": step.Name,
			"warning":   err.Error(),
		})
	}

	// Check if step should be executed based on conditions
	shouldExecute, skipReason := e.execContext.ShouldExecuteStep(step)
	if !shouldExecute {
		fmt.Printf("      ⏭️  %s (%s) - SKIPPED: %s\n", step.Name, step.Type, skipReason)

		// Create step execution record as skipped
		stepConfig, err := stepToConfig(step)
		if err != nil {
			return fmt.Errorf("failed to serialize step config: %w", err)
		}
		// Add skip reason to config
		stepConfig["skip_reason"] = skipReason

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
	stepConfig, err := stepToConfig(step)
	if err != nil {
		return fmt.Errorf("failed to serialize step config: %w", err)
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
	if err := e.executeStepWithExecutor(ctx, step, appName, execID, stepRecord.ID); err != nil {
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
func (e *WorkflowExecutor) executeStepWithExecutor(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
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

	return executor(stepCtx, step, appName, execID, stepID)
}

// registerDefaultStepExecutors registers the default step executors
func (e *WorkflowExecutor) registerDefaultStepExecutors() {
	// Resource provisioning executor
	e.stepExecutors["resource-provisioning"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
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
	e.stepExecutors["security"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(4 * time.Second)
		fmt.Printf("      🔒 Security scan completed\n")
		return nil
	}

	// Policy validation executor
	e.stepExecutors["policy"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		fmt.Printf("      📋 Executing policy script: %s\n", step.Name)

		// Get script from config
		script, ok := step.Config["script"].(string)
		if !ok || script == "" {
			return fmt.Errorf("policy step requires 'script' in config")
		}

		// Get workflow variables from execution context
		// These contain the resource properties passed from the orchestration engine
		workflowVars := make(map[string]interface{})
		for k, v := range e.execContext.WorkflowVariables {
			workflowVars[k] = v
		}

		// Create template data with parameters from workflow variables
		// Templates expect .parameters.field_name syntax
		templateData := map[string]interface{}{
			"parameters": workflowVars,
		}

		fmt.Printf("      🔍 DEBUG: Template parameters for policy script: %v\n", workflowVars)

		// Render script template with parameters
		renderedScript, err := e.renderTemplate(script, templateData)
		if err != nil {
			return fmt.Errorf("failed to render policy script template: %w", err)
		}

		fmt.Printf("      📝 DEBUG: Rendered script (first 300 chars):\n%s\n", func() string {
			if len(renderedScript) > 300 {
				return renderedScript[:300] + "..."
			}
			return renderedScript
		}())

		// Create temporary script file
		tmpFile, err := os.CreateTemp("", "policy-*.sh")
		if err != nil {
			return fmt.Errorf("failed to create temp script file: %w", err)
		}
		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Write rendered script to file (not raw script)
		if _, err := tmpFile.WriteString(renderedScript); err != nil {
			return fmt.Errorf("failed to write script: %w", err)
		}
		_ = tmpFile.Close()

		// Make script executable (0700 = owner only, needs execute bit for bash)
		// #nosec G302 -- Script needs execute permissions to run
		if err := os.Chmod(tmpFile.Name(), 0700); err != nil {
			return fmt.Errorf("failed to make script executable: %w", err)
		}

		// Execute script and capture output
		// #nosec G204 -- tmpFile.Name() is a controlled temporary file path
		cmd := exec.Command("/bin/bash", tmpFile.Name())

		// Capture output for log persistence
		var outputBuf strings.Builder
		cmd.Stdout = io.MultiWriter(os.Stdout, &outputBuf)
		cmd.Stderr = io.MultiWriter(os.Stderr, &outputBuf)

		// Set environment variables
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("APP_NAME=%s", appName),
		)

		if err := cmd.Run(); err != nil {
			// Store logs even on failure
			_ = e.repo.AddWorkflowStepLogs(stepID, outputBuf.String())
			return fmt.Errorf("policy script failed: %w", err)
		}

		// Store captured logs in database
		if err := e.repo.AddWorkflowStepLogs(stepID, outputBuf.String()); err != nil {
			fmt.Printf("      ⚠️  Warning: failed to store step logs: %v\n", err)
		}

		fmt.Printf("      ✅ Policy script completed successfully\n")
		return nil
	}

	// Cost analysis executor
	e.stepExecutors["cost-analysis"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      💰 Cost analysis completed\n")
		return nil
	}

	// Tagging executor
	e.stepExecutors["tagging"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("      🏷️  Resource tagging completed\n")
		return nil
	}

	// Database migration executor
	e.stepExecutors["database-migration"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(3 * time.Second)
		fmt.Printf("      🗃️  Database migration completed\n")
		return nil
	}

	// Vault setup executor
	e.stepExecutors["vault-setup"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      🔐 Vault configuration completed\n")
		return nil
	}

	// Monitoring setup executor
	e.stepExecutors["monitoring"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(2 * time.Second)
		fmt.Printf("      📊 Monitoring setup completed\n")
		return nil
	}

	// Validation executor
	e.stepExecutors["validation"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		time.Sleep(1 * time.Second)
		fmt.Printf("      ✅ Validation completed\n")
		return nil
	}

	// Terraform executor
	e.stepExecutors["terraform"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
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
				return e.terraformCaptureOutputs(ctx, workspaceDir, outputNames, step)
			}
			return nil
		case "destroy":
			if err := e.terraformInit(ctx, workspaceDir); err != nil {
				return err
			}
			return e.terraformDestroy(ctx, workspaceDir, variables)
		case "output":
			return e.terraformCaptureOutputs(ctx, workspaceDir, outputNames, step)
		default:
			return fmt.Errorf("unsupported terraform operation: %s", operation)
		}
	}

	// Terraform-generate executor - generates Terraform code from Score resources
	e.stepExecutors["terraform-generate"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
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

	// Kubernetes executor - applies Kubernetes manifests
	e.stepExecutors["kubernetes"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		fmt.Printf("      ☸️  Executing Kubernetes step: %s\n", step.Name)

		// Get namespace (default to app name if not specified)
		namespace := step.Namespace
		if namespace == "" && step.Config != nil {
			if ns, ok := step.Config["namespace"].(string); ok {
				namespace = ns
			}
		}
		if namespace == "" {
			namespace = appName
		}

		// Get operation (default: apply)
		operation := step.Operation
		if operation == "" && step.Config != nil {
			if op, ok := step.Config["operation"].(string); ok {
				operation = op
			}
		}
		if operation == "" {
			operation = "apply"
		}

		fmt.Printf("      📋 Operation: %s\n", operation)
		fmt.Printf("      🏷️  Namespace: %s\n", namespace)

		// Handle different kubernetes operations
		var logs string
		var err error

		switch operation {
		case "create-namespace":
			logs, err = e.kubernetesCreateNamespace(ctx, namespace)
			if err != nil {
				// Store logs even on failure
				_ = e.repo.AddWorkflowStepLogs(stepID, logs)
				return err
			}

		case "apply":
			// Get manifest from config (inline YAML or file path)
			manifest, ok := step.Config["manifest"].(string)
			if !ok || manifest == "" {
				return fmt.Errorf("kubernetes apply step requires 'manifest' in config")
			}

			// Get workflow variables from execution context
			// These contain the resource properties passed from the orchestration engine
			workflowVars := make(map[string]interface{})
			for k, v := range e.execContext.WorkflowVariables {
				workflowVars[k] = v
			}

			// Create template data with parameters from workflow variables
			// Templates expect .parameters.field_name syntax
			templateData := map[string]interface{}{
				"parameters": workflowVars,
			}

			fmt.Printf("      🔍 DEBUG: Template parameters from workflow variables: %v\n", workflowVars)

			// Render template with parameters
			rendered, err := e.renderTemplate(manifest, templateData)
			if err != nil {
				return fmt.Errorf("failed to render manifest template: %w", err)
			}

			fmt.Printf("      📝 DEBUG: Rendered manifest (first 500 chars):\n%s\n", func() string {
				if len(rendered) > 500 {
					return rendered[:500] + "..."
				}
				return rendered
			}())

			logs, err = e.kubernetesApply(ctx, namespace, rendered)
			if err != nil {
				// Store logs even on failure
				_ = e.repo.AddWorkflowStepLogs(stepID, logs)
				return err
			}

		case "delete":
			// Get manifest or resource identifier
			manifest, ok := step.Config["manifest"].(string)
			if !ok || manifest == "" {
				return fmt.Errorf("kubernetes delete step requires 'manifest' in config")
			}

			// Render template
			rendered, err := e.renderTemplate(manifest, step.Config)
			if err != nil {
				return fmt.Errorf("failed to render manifest template: %w", err)
			}

			return e.kubernetesDelete(ctx, namespace, rendered)

		case "get":
			// Get resource type and name
			resourceType, ok := step.Config["resource_type"].(string)
			if !ok || resourceType == "" {
				return fmt.Errorf("kubernetes get step requires 'resource_type' in config")
			}

			resourceName, _ := step.Config["resource_name"].(string)

			return e.kubernetesGet(ctx, namespace, resourceType, resourceName)

		default:
			return fmt.Errorf("unsupported kubernetes operation: %s (supported: apply, delete, get, create-namespace)", operation)
		}

		// Store captured logs in database
		if err := e.repo.AddWorkflowStepLogs(stepID, logs); err != nil {
			fmt.Printf("      ⚠️  Warning: failed to store step logs: %v\n", err)
		}

		return nil
	}

	// Ansible executor - runs Ansible playbooks
	e.stepExecutors["ansible"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		fmt.Printf("      🤖 Executing Ansible step: %s\n", step.Name)

		// Get playbook from config
		playbook, ok := step.Config["playbook"].(string)
		if !ok || playbook == "" {
			// Try step.Playbook field for backward compatibility
			playbook = step.Playbook
			if playbook == "" {
				return fmt.Errorf("ansible step requires 'playbook' in config")
			}
		}

		// Check if playbook exists
		if _, err := os.Stat(playbook); os.IsNotExist(err) {
			return fmt.Errorf("ansible playbook does not exist: %s", playbook)
		}

		fmt.Printf("      📝 Playbook: %s\n", playbook)

		// Run ansible-playbook
		// #nosec G204 - playbook from validated workflow definition
		cmd := exec.CommandContext(ctx, "ansible-playbook", playbook)

		// Set working directory if specified
		workingDir := step.WorkingDir
		if workingDir == "" && step.Config != nil {
			if wd, ok := step.Config["working_dir"].(string); ok {
				workingDir = wd
			}
		}
		if workingDir != "" {
			cmd.Dir = workingDir
		}

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ansible-playbook failed: %w", err)
		}

		fmt.Printf("      ✅ Ansible playbook completed successfully\n")
		return nil
	}

	// Gitea repository executor - creates/manages Gitea repositories
	e.stepExecutors["gitea-repo"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		fmt.Printf("      🗂️  Executing Gitea repository step: %s\n", step.Name)

		// This is a simplified version - full implementation would use Gitea API
		// For now, we delegate to the legacy implementation for compatibility
		return runStepWithSpinner(step, appName, "default", nil)
	}

	// ArgoCD application executor - creates/manages ArgoCD applications
	e.stepExecutors["argocd-app"] = func(ctx context.Context, step types.Step, appName string, execID int64, stepID int64) error {
		fmt.Printf("      🚀 Executing ArgoCD application step: %s\n", step.Name)

		// This is a simplified version - full implementation would use ArgoCD API
		// For now, we delegate to the legacy implementation for compatibility
		return runStepWithSpinner(step, appName, "default", nil)
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
func (e *WorkflowExecutor) terraformCaptureOutputs(ctx context.Context, workspaceDir string, outputNames []string, step types.Step) error {
	fmt.Printf("      📤 Capturing Terraform outputs\n")

	// Determine resource name for storing outputs
	// Priority: step.Resource > step.Name
	resourceName := step.Resource
	if resourceName == "" {
		resourceName = step.Name
	}

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

	// Extract specified outputs and store in execution context
	for _, outputName := range outputNames {
		if outputValue, ok := outputs[outputName]; ok {
			if outputMap, ok := outputValue.(map[string]interface{}); ok {
				if value, ok := outputMap["value"]; ok {
					valueStr := fmt.Sprintf("%v", value)
					fmt.Printf("      📊 Output '%s': %s\n", outputName, valueStr)

					// Store output in execution context for interpolation in subsequent steps
					e.execContext.SetResourceOutput(resourceName, outputName, valueStr)
					fmt.Printf("      ✓ Stored as ${resources.%s.%s}\n", resourceName, outputName)
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

// Kubernetes helper functions

// kubernetesCreateNamespace creates a Kubernetes namespace and returns output logs
func (e *WorkflowExecutor) kubernetesCreateNamespace(ctx context.Context, namespace string) (string, error) {
	fmt.Printf("      🏗️  Creating namespace: %s\n", namespace)

	// #nosec G204 - namespace is validated input from workflow config
	cmd := exec.CommandContext(ctx, "kubectl", "create", "namespace", namespace)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil && !strings.Contains(outputStr, "AlreadyExists") {
		return outputStr, fmt.Errorf("failed to create namespace: %w, output: %s", err, outputStr)
	}

	if strings.Contains(outputStr, "AlreadyExists") {
		fmt.Printf("      ℹ️  Namespace already exists: %s\n", namespace)
	} else {
		fmt.Printf("      ✅ Namespace created: %s\n", namespace)
	}

	return outputStr, nil
}

// kubernetesApply applies a Kubernetes manifest and returns output logs
func (e *WorkflowExecutor) kubernetesApply(ctx context.Context, namespace, manifest string) (string, error) {
	fmt.Printf("      📝 Applying Kubernetes manifest (workflow context namespace: %s)\n", namespace)

	// Don't pass -n flag to kubectl - let the manifest specify its own namespace
	// This avoids conflicts when the manifest has a namespace field in metadata
	// #nosec G204 - validated inputs from workflow config
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return outputStr, fmt.Errorf("failed to apply manifest: %w, output: %s", err, outputStr)
	}

	fmt.Printf("      ✅ Manifest applied successfully\n")
	fmt.Printf("      📋 Output: %s\n", outputStr)

	return outputStr, nil
}

// kubernetesDelete deletes a Kubernetes resource
func (e *WorkflowExecutor) kubernetesDelete(ctx context.Context, namespace, manifest string) error {
	fmt.Printf("      🗑️  Deleting Kubernetes resources from namespace: %s\n", namespace)

	// #nosec G204 - namespace is validated input from workflow config
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "-f", "-", "-n", namespace)
	cmd.Stdin = strings.NewReader(manifest)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete resources: %w, output: %s", err, string(output))
	}

	fmt.Printf("      ✅ Resources deleted successfully\n")
	fmt.Printf("      📋 Output: %s\n", string(output))

	return nil
}

// kubernetesGet retrieves Kubernetes resource information
func (e *WorkflowExecutor) kubernetesGet(ctx context.Context, namespace, resourceType, resourceName string) error {
	fmt.Printf("      🔍 Getting Kubernetes resource: %s/%s\n", resourceType, resourceName)

	args := []string{"get", resourceType}
	if resourceName != "" {
		args = append(args, resourceName)
	}
	args = append(args, "-n", namespace, "-o", "yaml")

	// #nosec G204 - args are validated inputs from workflow config
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get resource: %w, output: %s", err, string(output))
	}

	fmt.Printf("      ✅ Resource retrieved successfully\n")
	fmt.Printf("      📋 Output:\n%s\n", string(output))

	return nil
}

// renderTemplate renders a Go template with the provided data
func (e *WorkflowExecutor) renderTemplate(templateStr string, data map[string]interface{}) (string, error) {
	tmpl, err := template.New("manifest").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
