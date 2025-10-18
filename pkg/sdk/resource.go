package sdk

import "time"

// Resource represents a resource instance being provisioned or managed
// This is the core data structure passed to provisioners
type Resource struct {
	// ID is the unique database identifier
	ID int64 `json:"id"`

	// ApplicationName is the name of the application this resource belongs to
	ApplicationName string `json:"application_name"`

	// ResourceName is the name of this specific resource instance
	ResourceName string `json:"resource_name"`

	// ResourceType identifies the type of resource (postgres, redis, etc.)
	ResourceType string `json:"resource_type"`

	// State is the current lifecycle state of the resource
	State ResourceState `json:"state"`

	// HealthStatus indicates the health of the resource
	HealthStatus string `json:"health_status"`

	// Configuration contains resource-specific configuration
	Configuration Config `json:"configuration"`

	// ProviderID is the external identifier from the platform provider
	ProviderID string `json:"provider_id,omitempty"`

	// ProviderMetadata contains platform-specific metadata
	ProviderMetadata map[string]interface{} `json:"provider_metadata,omitempty"`

	// Hints are contextual links and commands for the resource
	Hints []Hint `json:"hints,omitempty"`

	// CreatedAt is the timestamp when the resource was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is the timestamp when the resource was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// ErrorMessage contains error details if provisioning failed
	ErrorMessage string `json:"error_message,omitempty"`
}

// ResourceState represents the lifecycle state of a resource
type ResourceState string

const (
	// ResourceStateRequested indicates the resource has been requested but not yet provisioned
	ResourceStateRequested ResourceState = "requested"

	// ResourceStateProvisioning indicates the resource is currently being provisioned
	ResourceStateProvisioning ResourceState = "provisioning"

	// ResourceStateActive indicates the resource is successfully provisioned and active
	ResourceStateActive ResourceState = "active"

	// ResourceStateScaling indicates the resource is being scaled
	ResourceStateScaling ResourceState = "scaling"

	// ResourceStateUpdating indicates the resource is being updated
	ResourceStateUpdating ResourceState = "updating"

	// ResourceStateDegraded indicates the resource is active but degraded
	ResourceStateDegraded ResourceState = "degraded"

	// ResourceStateTerminating indicates the resource is being terminated
	ResourceStateTerminating ResourceState = "terminating"

	// ResourceStateTerminated indicates the resource has been terminated
	ResourceStateTerminated ResourceState = "terminated"

	// ResourceStateFailed indicates the resource provisioning or operation failed
	ResourceStateFailed ResourceState = "failed"
)

// ResourceStatus contains the current status of a resource
// Returned by Provisioner.GetStatus()
type ResourceStatus struct {
	// State is the current lifecycle state
	State ResourceState `json:"state"`

	// HealthStatus indicates the health (healthy, degraded, unhealthy, unknown)
	HealthStatus string `json:"health_status"`

	// Message provides additional context about the status
	Message string `json:"message,omitempty"`

	// Metadata contains platform-specific status information
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// LastChecked is when this status was last verified
	LastChecked time.Time `json:"last_checked"`
}

// IsHealthy returns true if the resource is in a healthy state
func (s *ResourceStatus) IsHealthy() bool {
	return s.HealthStatus == "healthy" || s.HealthStatus == "ok"
}

// IsActive returns true if the resource is in an active state
func (r *Resource) IsActive() bool {
	return r.State == ResourceStateActive
}

// IsFailed returns true if the resource is in a failed state
func (r *Resource) IsFailed() bool {
	return r.State == ResourceStateFailed
}

// IsTerminated returns true if the resource has been terminated
func (r *Resource) IsTerminated() bool {
	return r.State == ResourceStateTerminated
}
