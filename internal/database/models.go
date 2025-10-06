package database

import (
	"encoding/json"
	"time"
)

// WorkflowExecution represents a specific execution instance of a workflow template.
// This tracks the runtime execution of a workflow definition with status, timing, and step progress.
// Workflow definitions/templates are stored as YAML files (e.g., workflows/deploy-app.yaml)
// while executions are runtime instances stored in the database.
type WorkflowExecution struct {
	ID              int64      `json:"id" db:"id"`
	ApplicationName string     `json:"application_name" db:"application_name"`
	WorkflowName    string     `json:"workflow_name" db:"workflow_name"` // References the template name
	Status          string     `json:"status" db:"status"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ErrorMessage    *string    `json:"error_message,omitempty" db:"error_message"`
	TotalSteps      int        `json:"total_steps" db:"total_steps"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// Related data (not stored in DB directly)
	Steps []*WorkflowStepExecution `json:"steps,omitempty"`
}

// WorkflowStepExecution represents the execution of a single step within a workflow execution.
// Each step corresponds to a step definition in the workflow template (e.g., terraform, kubernetes, ansible)
type WorkflowStepExecution struct {
	ID                  int64                  `json:"id" db:"id"`
	WorkflowExecutionID int64                  `json:"workflow_execution_id" db:"workflow_execution_id"`
	StepNumber          int                    `json:"step_number" db:"step_number"`
	StepName            string                 `json:"step_name" db:"step_name"`
	StepType            string                 `json:"step_type" db:"step_type"`
	Status              string                 `json:"status" db:"status"`
	StartedAt           *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt         *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs          *int64                 `json:"duration_ms,omitempty" db:"duration_ms"`
	ErrorMessage        *string                `json:"error_message,omitempty" db:"error_message"`
	StepConfig          map[string]interface{} `json:"step_config,omitempty" db:"step_config"`
	OutputLogs          *string                `json:"output_logs,omitempty" db:"output_logs"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
}

// Workflow execution status constants
const (
	WorkflowStatusRunning   = "running"
	WorkflowStatusCompleted = "completed"
	WorkflowStatusFailed    = "failed"
)

// Workflow step status constants
const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusCompleted = "completed"
	StepStatusFailed    = "failed"
)

// SetStepConfig converts step configuration to JSON for database storage
func (s *WorkflowStepExecution) SetStepConfig(config map[string]interface{}) error {
	s.StepConfig = config
	return nil
}

// GetStepConfig returns the step configuration
func (s *WorkflowStepExecution) GetStepConfig() map[string]interface{} {
	if s.StepConfig == nil {
		return make(map[string]interface{})
	}
	return s.StepConfig
}

// CalculateDuration calculates the duration if both start and end times are available
func (s *WorkflowStepExecution) CalculateDuration() {
	if s.StartedAt != nil && s.CompletedAt != nil {
		duration := s.CompletedAt.Sub(*s.StartedAt).Milliseconds()
		s.DurationMs = &duration
	}
}

// WorkflowExecutionSummary provides a summary view for listing workflows
type WorkflowExecutionSummary struct {
	ID              int64      `json:"id"`
	ApplicationName string     `json:"application_name"`
	WorkflowName    string     `json:"workflow_name"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	TotalSteps      int        `json:"total_steps"`
	CompletedSteps  int        `json:"completed_steps"`
	FailedSteps     int        `json:"failed_steps"`
	Duration        *int64     `json:"duration_ms,omitempty"`
}

// WorkflowStepConfigJSON handles JSON marshaling for step configuration
type WorkflowStepConfigJSON map[string]interface{}

// Value implements the driver Valuer interface for database storage
func (c WorkflowStepConfigJSON) Value() (interface{}, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements the sql Scanner interface for database retrieval
func (c *WorkflowStepConfigJSON) Scan(value interface{}) error {
	if value == nil {
		*c = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	default:
		return nil
	}
}

// ResourceInstance represents a managed resource with lifecycle tracking
type ResourceInstance struct {
	ID                  int64                  `json:"id" db:"id"`
	ApplicationName     string                 `json:"application_name" db:"application_name"`
	ResourceName        string                 `json:"resource_name" db:"resource_name"`
	ResourceType        string                 `json:"resource_type" db:"resource_type"`
	State               ResourceLifecycleState `json:"state" db:"state"`
	HealthStatus        string                 `json:"health_status" db:"health_status"`
	Configuration       map[string]interface{} `json:"configuration" db:"configuration"`
	ProviderID          *string                `json:"provider_id,omitempty" db:"provider_id"`
	ProviderMetadata    map[string]interface{} `json:"provider_metadata,omitempty" db:"provider_metadata"`
	Type                string                 `json:"type" db:"type"`                               // "native" or "delegated"
	Provider            *string                `json:"provider,omitempty" db:"provider"`             // e.g., "gitops", "terraform-enterprise"
	ReferenceURL        *string                `json:"reference_url,omitempty" db:"reference_url"`   // PR URL, external ID, or build link
	ExternalState       *string                `json:"external_state,omitempty" db:"external_state"` // External system state
	LastSync            *time.Time             `json:"last_sync,omitempty" db:"last_sync"`           // Last synchronization time
	WorkflowExecutionID *int64                 `json:"workflow_execution_id,omitempty" db:"workflow_execution_id"`
	CreatedAt           time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at" db:"updated_at"`
	LastHealthCheck     *time.Time             `json:"last_health_check,omitempty" db:"last_health_check"`
	ErrorMessage        *string                `json:"error_message,omitempty" db:"error_message"`

	// Related data (not stored in DB directly)
	Dependencies     []string                   `json:"dependencies,omitempty"`
	StateTransitions []*ResourceStateTransition `json:"state_transitions,omitempty"`
}

// ResourceLifecycleState represents the current state of a resource
type ResourceLifecycleState string

const (
	ResourceStateRequested    ResourceLifecycleState = "requested"
	ResourceStateProvisioning ResourceLifecycleState = "provisioning"
	ResourceStateActive       ResourceLifecycleState = "active"
	ResourceStateScaling      ResourceLifecycleState = "scaling"
	ResourceStateUpdating     ResourceLifecycleState = "updating"
	ResourceStateDegraded     ResourceLifecycleState = "degraded"
	ResourceStateTerminating  ResourceLifecycleState = "terminating"
	ResourceStateTerminated   ResourceLifecycleState = "terminated"
	ResourceStateFailed       ResourceLifecycleState = "failed"
)

// Resource type constants
const (
	ResourceTypeNative    = "native"    // Directly managed by orchestrator
	ResourceTypeDelegated = "delegated" // Managed by external system (GitOps, Terraform Enterprise)
	ResourceTypeExternal  = "external"  // Read-only reference to external resource
)

// External state constants for delegated resources
const (
	ExternalStateWaitingExternal  = "WaitingExternal"  // Waiting for external system to start
	ExternalStateBuildingExternal = "BuildingExternal" // External system is building/provisioning
	ExternalStateHealthy          = "Healthy"          // External resource is healthy
	ExternalStateError            = "Error"            // External resource has errors
	ExternalStateUnknown          = "Unknown"          // External state is unknown
)

// ResourceStateTransition tracks state changes for audit trail
type ResourceStateTransition struct {
	ID                 int64                  `json:"id" db:"id"`
	ResourceInstanceID int64                  `json:"resource_instance_id" db:"resource_instance_id"`
	FromState          ResourceLifecycleState `json:"from_state" db:"from_state"`
	ToState            ResourceLifecycleState `json:"to_state" db:"to_state"`
	Reason             string                 `json:"reason" db:"reason"`
	TransitionedBy     string                 `json:"transitioned_by" db:"transitioned_by"`
	TransitionedAt     time.Time              `json:"transitioned_at" db:"transitioned_at"`
	Metadata           map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// ResourceHealthCheck represents health check results
type ResourceHealthCheck struct {
	ID                 int64                  `json:"id" db:"id"`
	ResourceInstanceID int64                  `json:"resource_instance_id" db:"resource_instance_id"`
	CheckType          string                 `json:"check_type" db:"check_type"`
	Status             string                 `json:"status" db:"status"` // healthy, degraded, unhealthy
	CheckedAt          time.Time              `json:"checked_at" db:"checked_at"`
	ResponseTime       *int64                 `json:"response_time,omitempty" db:"response_time"`
	ErrorMessage       *string                `json:"error_message,omitempty" db:"error_message"`
	Metrics            map[string]interface{} `json:"metrics,omitempty" db:"metrics"`
}

// ResourceDependency tracks dependencies between resources
type ResourceDependency struct {
	ID                 int64     `json:"id" db:"id"`
	ResourceInstanceID int64     `json:"resource_instance_id" db:"resource_instance_id"`
	DependsOnID        int64     `json:"depends_on_id" db:"depends_on_id"`
	DependencyType     string    `json:"dependency_type" db:"dependency_type"` // hard, soft, optional
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
}

// Resource lifecycle state validation
var ValidResourceStateTransitions = map[ResourceLifecycleState][]ResourceLifecycleState{
	ResourceStateRequested: {
		ResourceStateProvisioning,
		ResourceStateFailed,
	},
	ResourceStateProvisioning: {
		ResourceStateActive,
		ResourceStateFailed,
	},
	ResourceStateActive: {
		ResourceStateScaling,
		ResourceStateUpdating,
		ResourceStateDegraded,
		ResourceStateTerminating,
		ResourceStateFailed,
	},
	ResourceStateScaling: {
		ResourceStateActive,
		ResourceStateFailed,
	},
	ResourceStateUpdating: {
		ResourceStateActive,
		ResourceStateFailed,
	},
	ResourceStateDegraded: {
		ResourceStateActive,
		ResourceStateTerminating,
		ResourceStateFailed,
	},
	ResourceStateTerminating: {
		ResourceStateTerminated,
		ResourceStateFailed,
	},
	ResourceStateFailed: {
		ResourceStateProvisioning,
		ResourceStateTerminating,
	},
}

// IsValidStateTransition checks if a state transition is valid
func (r *ResourceInstance) IsValidStateTransition(newState ResourceLifecycleState) bool {
	validStates, exists := ValidResourceStateTransitions[r.State]
	if !exists {
		return false
	}

	for _, validState := range validStates {
		if validState == newState {
			return true
		}
	}
	return false
}

// SetConfiguration converts resource configuration to JSON for database storage
func (r *ResourceInstance) SetConfiguration(config map[string]interface{}) error {
	r.Configuration = config
	return nil
}

// GetConfiguration returns the resource configuration
func (r *ResourceInstance) GetConfiguration() map[string]interface{} {
	if r.Configuration == nil {
		return make(map[string]interface{})
	}
	return r.Configuration
}

// SetProviderMetadata converts provider metadata to JSON for database storage
func (r *ResourceInstance) SetProviderMetadata(metadata map[string]interface{}) error {
	r.ProviderMetadata = metadata
	return nil
}

// GetProviderMetadata returns the provider metadata
func (r *ResourceInstance) GetProviderMetadata() map[string]interface{} {
	if r.ProviderMetadata == nil {
		return make(map[string]interface{})
	}
	return r.ProviderMetadata
}
