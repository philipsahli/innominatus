package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/events"
	"innominatus/internal/graph"
	"innominatus/internal/logging"
	"innominatus/internal/providers"
	"innominatus/internal/types"
	"innominatus/internal/workflow"
	"innominatus/pkg/sdk"
	"os"
	"path/filepath"
	"time"

	graphSDK "github.com/philipsahli/innominatus-graph/pkg/graph"
	"gopkg.in/yaml.v3"
)

// Engine is the event-driven orchestration engine
// It polls for pending resources and automatically triggers provider workflows
type Engine struct {
	db           *database.Database
	registry     *providers.Registry
	resolver     *Resolver
	resourceRepo *database.ResourceRepository
	workflowRepo *database.WorkflowRepository
	workflowExec *workflow.WorkflowExecutor
	graphAdapter *graph.Adapter
	eventBus     events.EventBus
	providersDir string
	pollInterval time.Duration
	stopChan     chan struct{}
	logger       *logging.ZerologAdapter
}

// NewEngine creates a new orchestration engine
func NewEngine(
	db *database.Database,
	registry *providers.Registry,
	workflowRepo *database.WorkflowRepository,
	resourceRepo *database.ResourceRepository,
	workflowExec *workflow.WorkflowExecutor,
	graphAdapter *graph.Adapter,
	providersDir string,
) *Engine {
	return &Engine{
		db:           db,
		registry:     registry,
		resolver:     NewResolver(registry),
		workflowRepo: workflowRepo,
		resourceRepo: resourceRepo,
		workflowExec: workflowExec,
		graphAdapter: graphAdapter,
		providersDir: providersDir,
		pollInterval: 5 * time.Second,
		stopChan:     make(chan struct{}),
		logger:       logging.NewStructuredLogger("orchestration"),
	}
}

// SetEventBus sets the event bus for publishing orchestration events
func (e *Engine) SetEventBus(bus events.EventBus) {
	e.eventBus = bus
	e.logger.Info("Event bus configured for orchestration engine")
}

// Start begins the orchestration engine polling loop
func (e *Engine) Start(ctx context.Context) {
	e.logger.InfoWithFields("Starting orchestration engine", map[string]interface{}{
		"poll_interval": e.pollInterval.String(),
	})

	ticker := time.NewTicker(e.pollInterval)
	defer ticker.Stop()

	// Initial poll on startup
	e.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			e.logger.Info("Orchestration engine stopped by context")
			return
		case <-e.stopChan:
			e.logger.Info("Orchestration engine stopped")
			return
		case <-ticker.C:
			e.poll(ctx)
		}
	}
}

// Stop gracefully stops the orchestration engine
func (e *Engine) Stop() {
	close(e.stopChan)
}

// poll checks for pending resources and triggers provisioning workflows
func (e *Engine) poll(ctx context.Context) {
	// First, check for requested/pending resources
	e.pollPendingResources(ctx)

	// Second, recover orphaned provisioning resources (stuck without workflow_execution_id)
	e.recoverOrphanedResources(ctx)

	// Third, CRITICAL FIX: update resource state based on workflow completion
	e.pollProvisioningResources(ctx)
}

// pollPendingResources polls for requested/pending resources without workflow execution
func (e *Engine) pollPendingResources(ctx context.Context) {
	// Query for pending resources without workflow execution
	query := `
		SELECT id, application_name, resource_name, resource_type, state,
		       configuration, provider_id, workflow_execution_id,
		       created_at, updated_at
		FROM resource_instances
		WHERE state IN ('requested', 'pending')
		AND workflow_execution_id IS NULL
		ORDER BY created_at ASC
		LIMIT 100
	`

	rows, err := e.db.DB().QueryContext(ctx, query)
	if err != nil {
		e.logger.ErrorWithFields("Failed to query pending resources", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var resources []*database.ResourceInstance
	for rows.Next() {
		var resource database.ResourceInstance
		var providerID *string
		var workflowExecutionID *int64
		var configJSON []byte

		err := rows.Scan(
			&resource.ID,
			&resource.ApplicationName,
			&resource.ResourceName,
			&resource.ResourceType,
			&resource.State,
			&configJSON,
			&providerID,
			&workflowExecutionID,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		)
		if err != nil {
			e.logger.ErrorWithFields("Failed to scan resource row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Parse configuration JSON
		if len(configJSON) > 0 {
			var config map[string]interface{}
			if err := json.Unmarshal(configJSON, &config); err != nil {
				e.logger.WarnWithFields("Failed to parse resource configuration", map[string]interface{}{
					"resource_id": resource.ID,
					"error":       err.Error(),
				})
			} else {
				resource.Configuration = config
			}
		}

		resource.ProviderID = providerID
		resource.WorkflowExecutionID = workflowExecutionID

		resources = append(resources, &resource)
	}

	if err := rows.Err(); err != nil {
		e.logger.ErrorWithFields("Error iterating resource rows", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(resources) == 0 {
		return // No pending resources
	}

	e.logger.InfoWithFields("Found pending resources", map[string]interface{}{
		"count": len(resources),
	})

	// Process each pending resource
	for _, resource := range resources {
		err := e.processResource(ctx, resource)
		if err != nil {
			e.logger.ErrorWithFields("Failed to process resource", map[string]interface{}{
				"resource_id":   resource.ID,
				"resource_name": resource.ResourceName,
				"resource_type": resource.ResourceType,
				"app_name":      resource.ApplicationName,
				"error":         err.Error(),
			})

			// Publish resource failed event
			if e.eventBus != nil {
				e.eventBus.Publish(events.NewEvent(
					events.EventTypeResourceFailed,
					resource.ApplicationName,
					"orchestration-engine",
					map[string]interface{}{
						"resource_id":   resource.ID,
						"resource_name": resource.ResourceName,
						"resource_type": resource.ResourceType,
						"error":         err.Error(),
					},
				))
			}

			// Update resource to failed state
			errorMsg := err.Error()
			_ = e.resourceRepo.UpdateResourceInstanceState(
				resource.ID,
				database.ResourceStateFailed,
				fmt.Sprintf("Failed to provision: %s", errorMsg),
				"orchestration-engine",
				nil,
			)
		}
	}
}

// processResource handles a single pending resource
func (e *Engine) processResource(ctx context.Context, resource *database.ResourceInstance) error {
	e.logger.InfoWithFields("Processing pending resource", map[string]interface{}{
		"resource_id":   resource.ID,
		"resource_name": resource.ResourceName,
		"resource_type": resource.ResourceType,
		"app_name":      resource.ApplicationName,
	})

	// Publish resource provisioning event
	if e.eventBus != nil {
		e.eventBus.Publish(events.NewEvent(
			events.EventTypeResourceProvisioning,
			resource.ApplicationName,
			"orchestration-engine",
			map[string]interface{}{
				"resource_id":   resource.ID,
				"resource_name": resource.ResourceName,
				"resource_type": resource.ResourceType,
			},
		))
	}

	// Step 1: Determine operation (create, update, delete)
	operation := "create" // Default operation
	if resource.DesiredOperation != nil && *resource.DesiredOperation != "" {
		operation = *resource.DesiredOperation
	}

	// Extract workflow tags for disambiguation
	var tags []string
	if len(resource.WorkflowTags) > 0 {
		tags = resource.WorkflowTags
	}

	e.logger.InfoWithFields("Processing resource with operation", map[string]interface{}{
		"resource_id":   resource.ID,
		"resource_type": resource.ResourceType,
		"operation":     operation,
		"tags":          tags,
	})

	// Step 2: Check for explicit workflow override
	var provider *sdk.Provider
	var workflowMeta *sdk.WorkflowMetadata
	var err error

	if resource.WorkflowOverride != nil && *resource.WorkflowOverride != "" {
		// User explicitly specified which workflow to use
		workflowName := *resource.WorkflowOverride
		e.logger.InfoWithFields("Using workflow override", map[string]interface{}{
			"workflow_name": workflowName,
		})

		// Still need to resolve provider for this resource type
		provider, _, err = e.resolver.ResolveWorkflowForOperation(resource.ResourceType, operation, tags)
		if err != nil {
			return fmt.Errorf("failed to resolve provider for workflow override: %w", err)
		}

		// Find the specified workflow in the provider
		workflowMeta = e.resolver.FindWorkflowByName(provider, workflowName)
		if workflowMeta == nil {
			return fmt.Errorf("workflow override '%s' not found in provider '%s'", workflowName, provider.Metadata.Name)
		}
	} else {
		// Standard resolution based on operation
		provider, workflowMeta, err = e.resolver.ResolveWorkflowForOperation(resource.ResourceType, operation, tags)
		if err != nil {
			return fmt.Errorf("failed to resolve provider: %w", err)
		}
	}

	e.logger.InfoWithFields("Resolved provider for resource", map[string]interface{}{
		"resource_type": resource.ResourceType,
		"operation":     operation,
		"provider_name": provider.Metadata.Name,
		"workflow_name": workflowMeta.Name,
	})

	// Publish provider resolved event
	if e.eventBus != nil {
		e.eventBus.Publish(events.NewEvent(
			events.EventTypeProviderResolved,
			resource.ApplicationName,
			"orchestration-engine",
			map[string]interface{}{
				"resource_id":   resource.ID,
				"resource_name": resource.ResourceName,
				"resource_type": resource.ResourceType,
				"provider_name": provider.Metadata.Name,
				"workflow_name": workflowMeta.Name,
			},
		))
	}

	// Step 2: Load the workflow YAML
	workflowDef, err := e.loadWorkflowFromProvider(provider, workflowMeta)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Step 3: Build workflow inputs from resource configuration
	workflowInputs := e.buildWorkflowInputs(resource, workflowDef)

	// Step 4: Execute workflow
	err = e.workflowExec.ExecuteWorkflowWithName(
		resource.ApplicationName,
		workflowMeta.Name,
		*workflowDef,
		workflowInputs,
	)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}

	// Step 5: Get the created workflow execution ID
	execution, err := e.workflowRepo.GetLatestWorkflowExecution(resource.ApplicationName, workflowMeta.Name)
	if err != nil {
		return fmt.Errorf("failed to get workflow execution: %w", err)
	}

	// Step 6: Update resource instance with provider and workflow info
	err = e.updateResourceWithProvisioningInfo(ctx, resource, provider, execution.ID)
	if err != nil {
		return fmt.Errorf("failed to update resource: %w", err)
	}

	// Step 7: Update graph with provider node and edges
	err = e.updateGraphWithProvider(ctx, resource, provider, workflowMeta.Name, execution.ID)
	if err != nil {
		e.logger.ErrorWithFields("Failed to update graph", map[string]interface{}{
			"resource_id": resource.ID,
			"error":       err.Error(),
		})
		// Non-fatal error, continue
	}

	e.logger.InfoWithFields("Successfully initiated resource provisioning", map[string]interface{}{
		"resource_id":           resource.ID,
		"resource_name":         resource.ResourceName,
		"provider_name":         provider.Metadata.Name,
		"workflow_execution_id": execution.ID,
	})

	return nil
}

// buildWorkflowInputs creates workflow variables from resource configuration
func (e *Engine) buildWorkflowInputs(resource *database.ResourceInstance, workflow *types.Workflow) map[string]string {
	inputs := make(map[string]string)

	// Add resource metadata
	inputs["app_name"] = resource.ApplicationName
	inputs["resource_name"] = resource.ResourceName
	inputs["resource_type"] = resource.ResourceType

	// Convert resource configuration to string inputs
	for key, value := range resource.Configuration {
		if strValue, ok := value.(string); ok {
			inputs[key] = strValue
		} else {
			inputs[key] = fmt.Sprintf("%v", value)
		}
	}

	// Add any default workflow variables that aren't overridden
	for key, value := range workflow.Variables {
		if _, exists := inputs[key]; !exists {
			inputs[key] = value
		}
	}

	return inputs
}

// updateResourceWithProvisioningInfo updates the resource instance with provider and workflow info
func (e *Engine) updateResourceWithProvisioningInfo(
	ctx context.Context,
	resource *database.ResourceInstance,
	provider *sdk.Provider,
	workflowExecutionID int64,
) error {
	providerID := provider.Metadata.Name

	query := `
		UPDATE resource_instances
		SET
			state = $1,
			provider_id = $2,
			workflow_execution_id = $3,
			updated_at = $4
		WHERE id = $5
	`

	_, err := e.db.DB().ExecContext(
		ctx,
		query,
		database.ResourceStateProvisioning,
		providerID,
		workflowExecutionID,
		time.Now(),
		resource.ID,
	)

	return err
}

// updateGraphWithProvider adds provider node and edges to the graph
func (e *Engine) updateGraphWithProvider(
	ctx context.Context,
	resource *database.ResourceInstance,
	provider *sdk.Provider,
	workflowName string,
	workflowExecutionID int64,
) error {
	if e.graphAdapter == nil {
		return nil // Graph adapter not configured
	}

	// Create provider node
	providerNode := &graphSDK.Node{
		ID:   fmt.Sprintf("provider:%s", provider.Metadata.Name),
		Name: provider.Metadata.Name,
		Type: "provider",
		Properties: map[string]interface{}{
			"version":  provider.Metadata.Version,
			"category": provider.Metadata.Category,
		},
	}

	err := e.graphAdapter.AddNode(resource.ApplicationName, providerNode)
	if err != nil {
		return fmt.Errorf("failed to add provider node: %w", err)
	}

	// Add edge: resource → provider (type: "requires")
	resourceNodeID := fmt.Sprintf("resource:%s:%s", resource.ApplicationName, resource.ResourceName)
	providerNodeID := fmt.Sprintf("provider:%s", provider.Metadata.Name)

	resourceToProviderEdge := &graphSDK.Edge{
		FromNodeID: resourceNodeID,
		ToNodeID:   providerNodeID,
		Type:       "requires",
	}

	err = e.graphAdapter.AddEdge(resource.ApplicationName, resourceToProviderEdge)
	if err != nil {
		return fmt.Errorf("failed to add resource→provider edge: %w", err)
	}

	// Add edge: provider → workflow (type: "executes")
	workflowNodeID := fmt.Sprintf("workflow:%s:%d", workflowName, workflowExecutionID)

	providerToWorkflowEdge := &graphSDK.Edge{
		FromNodeID: providerNodeID,
		ToNodeID:   workflowNodeID,
		Type:       "executes",
	}

	err = e.graphAdapter.AddEdge(resource.ApplicationName, providerToWorkflowEdge)
	if err != nil {
		return fmt.Errorf("failed to add provider→workflow edge: %w", err)
	}

	return nil
}

// loadWorkflowFromProvider loads a workflow YAML file from a provider
func (e *Engine) loadWorkflowFromProvider(provider *sdk.Provider, workflowMeta *sdk.WorkflowMetadata) (*types.Workflow, error) {
	// Construct workflow file path
	// Workflow file path is relative to provider directory
	providerDir := filepath.Join(e.providersDir, provider.Metadata.Name)
	workflowPath := filepath.Join(providerDir, workflowMeta.File)

	// Read workflow file
	// #nosec G304 -- workflow path is constructed from validated provider config
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file %s: %w", workflowPath, err)
	}

	// Parse workflow YAML
	var workflow types.Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	return &workflow, nil
}

// recoverOrphanedResources recovers resources stuck in provisioning state without workflow_execution_id
// This can happen if a resource was transitioned to provisioning but the workflow never started
func (e *Engine) recoverOrphanedResources(ctx context.Context) {
	// Query for orphaned provisioning resources (provisioning state but no workflow)
	// Only recover resources that have been stuck for more than 30 seconds
	query := `
		SELECT id, application_name, resource_name, resource_type, state,
		       configuration, provider_id, workflow_execution_id,
		       created_at, updated_at
		FROM resource_instances
		WHERE state = 'provisioning'
		AND workflow_execution_id IS NULL
		AND updated_at < NOW() - INTERVAL '30 seconds'
		ORDER BY created_at ASC
		LIMIT 50
	`

	rows, err := e.db.DB().QueryContext(ctx, query)
	if err != nil {
		e.logger.ErrorWithFields("Failed to query orphaned resources", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	var resources []*database.ResourceInstance
	for rows.Next() {
		var resource database.ResourceInstance
		var providerID *string
		var workflowExecutionID *int64
		var configJSON []byte

		err := rows.Scan(
			&resource.ID,
			&resource.ApplicationName,
			&resource.ResourceName,
			&resource.ResourceType,
			&resource.State,
			&configJSON,
			&providerID,
			&workflowExecutionID,
			&resource.CreatedAt,
			&resource.UpdatedAt,
		)
		if err != nil {
			e.logger.ErrorWithFields("Failed to scan orphaned resource row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		// Parse configuration JSON
		if len(configJSON) > 0 {
			var config map[string]interface{}
			if err := json.Unmarshal(configJSON, &config); err != nil {
				e.logger.WarnWithFields("Failed to parse orphaned resource configuration", map[string]interface{}{
					"resource_id": resource.ID,
					"error":       err.Error(),
				})
			} else {
				resource.Configuration = config
			}
		}

		resource.ProviderID = providerID
		resource.WorkflowExecutionID = workflowExecutionID

		resources = append(resources, &resource)
	}

	if err := rows.Err(); err != nil {
		e.logger.ErrorWithFields("Error iterating orphaned resource rows", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(resources) == 0 {
		return // No orphaned resources
	}

	e.logger.InfoWithFields("Found orphaned provisioning resources", map[string]interface{}{
		"count": len(resources),
	})

	// Reset orphaned resources back to requested state so they can be picked up again
	for _, resource := range resources {
		e.logger.WarnWithFields("Recovering orphaned resource", map[string]interface{}{
			"resource_id":   resource.ID,
			"resource_name": resource.ResourceName,
			"resource_type": resource.ResourceType,
			"app_name":      resource.ApplicationName,
			"stuck_since":   resource.UpdatedAt,
		})

		// Reset state to requested
		err := e.resourceRepo.UpdateResourceInstanceState(
			resource.ID,
			database.ResourceStateRequested,
			"Recovered from orphaned provisioning state",
			"orchestration-engine",
			nil,
		)
		if err != nil {
			e.logger.ErrorWithFields("Failed to recover orphaned resource", map[string]interface{}{
				"resource_id": resource.ID,
				"error":       err.Error(),
			})
			continue
		}

		e.logger.InfoWithFields("Successfully recovered orphaned resource", map[string]interface{}{
			"resource_id":   resource.ID,
			"resource_name": resource.ResourceName,
		})
	}
}

// pollProvisioningResources checks for provisioning resources and updates their state based on workflow completion
// CRITICAL FIX: This ensures resources transition to 'active' or 'failed' based on workflow status
func (e *Engine) pollProvisioningResources(ctx context.Context) {
	// Query for resources in provisioning state with workflow execution
	query := `
		SELECT ri.id, ri.application_name, ri.resource_name, ri.resource_type, ri.state,
		       ri.workflow_execution_id, we.status, we.error_message
		FROM resource_instances ri
		INNER JOIN workflow_executions we ON ri.workflow_execution_id = we.id
		WHERE ri.state = 'provisioning'
		AND ri.workflow_execution_id IS NOT NULL
		AND we.status IN ('completed', 'failed')
		ORDER BY ri.created_at ASC
		LIMIT 100
	`

	rows, err := e.db.DB().QueryContext(ctx, query)
	if err != nil {
		e.logger.ErrorWithFields("Failed to query provisioning resources", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	defer rows.Close()

	type resourceWorkflowStatus struct {
		resourceID          int64
		appName             string
		resourceName        string
		resourceType        string
		currentState        database.ResourceLifecycleState
		workflowExecutionID int64
		workflowStatus      string
		errorMessage        *string
	}

	var resources []resourceWorkflowStatus
	for rows.Next() {
		var rws resourceWorkflowStatus
		err := rows.Scan(
			&rws.resourceID,
			&rws.appName,
			&rws.resourceName,
			&rws.resourceType,
			&rws.currentState,
			&rws.workflowExecutionID,
			&rws.workflowStatus,
			&rws.errorMessage,
		)
		if err != nil {
			e.logger.ErrorWithFields("Failed to scan provisioning resource row", map[string]interface{}{
				"error": err.Error(),
			})
			continue
		}

		resources = append(resources, rws)
	}

	if err := rows.Err(); err != nil {
		e.logger.ErrorWithFields("Error iterating provisioning resource rows", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if len(resources) == 0 {
		return // No resources to update
	}

	e.logger.InfoWithFields("Found provisioning resources with completed workflows", map[string]interface{}{
		"count": len(resources),
	})

	// Update resource state based on workflow status
	for _, rws := range resources {
		var newState database.ResourceLifecycleState
		var reason string

		if rws.workflowStatus == "completed" {
			newState = database.ResourceStateActive
			reason = "Resource provisioning completed successfully"

			e.logger.InfoWithFields("Marking resource as active after successful workflow", map[string]interface{}{
				"resource_id":           rws.resourceID,
				"resource_name":         rws.resourceName,
				"resource_type":         rws.resourceType,
				"workflow_execution_id": rws.workflowExecutionID,
			})
		} else if rws.workflowStatus == "failed" {
			newState = database.ResourceStateFailed
			reason = "Resource provisioning failed"
			if rws.errorMessage != nil {
				reason = fmt.Sprintf("Resource provisioning failed: %s", *rws.errorMessage)
			}

			e.logger.WarnWithFields("Marking resource as failed after workflow failure", map[string]interface{}{
				"resource_id":           rws.resourceID,
				"resource_name":         rws.resourceName,
				"resource_type":         rws.resourceType,
				"workflow_execution_id": rws.workflowExecutionID,
				"error":                 rws.errorMessage,
			})
		} else {
			continue // Skip unknown states
		}

		// Update resource state
		err := e.resourceRepo.UpdateResourceInstanceState(
			rws.resourceID,
			newState,
			reason,
			"orchestration-engine",
			nil,
		)
		if err != nil {
			e.logger.ErrorWithFields("Failed to update resource state", map[string]interface{}{
				"resource_id": rws.resourceID,
				"error":       err.Error(),
			})
			continue
		}

		e.logger.InfoWithFields("Successfully updated resource state", map[string]interface{}{
			"resource_id":   rws.resourceID,
			"resource_name": rws.resourceName,
			"new_state":     newState,
		})

		// Publish state change event
		if e.eventBus != nil {
			var eventType events.EventType
			if newState == database.ResourceStateActive {
				eventType = events.EventTypeResourceActive
			} else {
				eventType = events.EventTypeResourceFailed
			}

			e.eventBus.Publish(events.NewEvent(
				eventType,
				rws.appName,
				"orchestration-engine",
				map[string]interface{}{
					"resource_id":           rws.resourceID,
					"resource_name":         rws.resourceName,
					"resource_type":         rws.resourceType,
					"workflow_execution_id": rws.workflowExecutionID,
					"new_state":             string(newState),
				},
			))
		}
	}
}
