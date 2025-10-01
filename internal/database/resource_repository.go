package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"errors"
)

// Common errors
var (
	ErrResourceNotFound = errors.New("resource not found")
)

// ResourceRepository handles resource instance operations
type ResourceRepository struct {
	db *Database
}

// NewResourceRepository creates a new resource repository
func NewResourceRepository(db *Database) *ResourceRepository {
	return &ResourceRepository{db: db}
}

// CreateResourceInstance creates a new resource instance
func (r *ResourceRepository) CreateResourceInstance(applicationName, resourceName, resourceType string, config map[string]interface{}) (*ResourceInstance, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configuration: %w", err)
	}

	query := `
		INSERT INTO resource_instances (application_name, resource_name, resource_type, state, health_status, configuration)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	var resource ResourceInstance
	resource.ApplicationName = applicationName
	resource.ResourceName = resourceName
	resource.ResourceType = resourceType
	resource.State = ResourceStateRequested
	resource.HealthStatus = "unknown"
	resource.Configuration = config

	err = r.db.db.QueryRow(query,
		applicationName, resourceName, resourceType,
		string(ResourceStateRequested), "unknown", configJSON).Scan(
		&resource.ID, &resource.CreatedAt, &resource.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create resource instance: %w", err)
	}

	return &resource, nil
}

// GetResourceInstance retrieves a resource instance by ID
func (r *ResourceRepository) GetResourceInstance(id int64) (*ResourceInstance, error) {
	query := `
		SELECT id, application_name, resource_name, resource_type, state, health_status,
		       configuration, provider_id, provider_metadata, created_at, updated_at,
		       last_health_check, error_message
		FROM resource_instances WHERE id = $1`

	var resource ResourceInstance
	var configJSON, providerMetadataJSON []byte

	err := r.db.db.QueryRow(query, id).Scan(
		&resource.ID, &resource.ApplicationName, &resource.ResourceName,
		&resource.ResourceType, &resource.State, &resource.HealthStatus,
		&configJSON, &resource.ProviderID, &providerMetadataJSON,
		&resource.CreatedAt, &resource.UpdatedAt, &resource.LastHealthCheck,
		&resource.ErrorMessage)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("resource instance not found")
		}
		return nil, fmt.Errorf("failed to get resource instance: %w", err)
	}

	// Unmarshal JSON fields
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &resource.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}
	}

	if len(providerMetadataJSON) > 0 {
		if err := json.Unmarshal(providerMetadataJSON, &resource.ProviderMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider metadata: %w", err)
		}
	}

	return &resource, nil
}

// GetResourceInstanceByName retrieves a resource instance by application and resource name
func (r *ResourceRepository) GetResourceInstanceByName(applicationName, resourceName string) (*ResourceInstance, error) {
	query := `
		SELECT id, application_name, resource_name, resource_type, state, health_status,
		       configuration, provider_id, provider_metadata, created_at, updated_at,
		       last_health_check, error_message
		FROM resource_instances WHERE application_name = $1 AND resource_name = $2`

	var resource ResourceInstance
	var configJSON, providerMetadataJSON []byte

	err := r.db.db.QueryRow(query, applicationName, resourceName).Scan(
		&resource.ID, &resource.ApplicationName, &resource.ResourceName,
		&resource.ResourceType, &resource.State, &resource.HealthStatus,
		&configJSON, &resource.ProviderID, &providerMetadataJSON,
		&resource.CreatedAt, &resource.UpdatedAt, &resource.LastHealthCheck,
		&resource.ErrorMessage)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("resource instance not found")
		}
		return nil, fmt.Errorf("failed to get resource instance: %w", err)
	}

	// Unmarshal JSON fields
	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &resource.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}
	}

	if len(providerMetadataJSON) > 0 {
		if err := json.Unmarshal(providerMetadataJSON, &resource.ProviderMetadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider metadata: %w", err)
		}
	}

	return &resource, nil
}

// ListResourceInstances lists all resource instances for an application
func (r *ResourceRepository) ListResourceInstances(applicationName string) ([]*ResourceInstance, error) {
	query := `
		SELECT id, application_name, resource_name, resource_type, state, health_status,
		       configuration, provider_id, provider_metadata, created_at, updated_at,
		       last_health_check, error_message
		FROM resource_instances WHERE application_name = $1 ORDER BY created_at ASC`

	rows, err := r.db.db.Query(query, applicationName)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource instances: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var resources []*ResourceInstance
	for rows.Next() {
		var resource ResourceInstance
		var configJSON, providerMetadataJSON []byte

		err := rows.Scan(
			&resource.ID, &resource.ApplicationName, &resource.ResourceName,
			&resource.ResourceType, &resource.State, &resource.HealthStatus,
			&configJSON, &resource.ProviderID, &providerMetadataJSON,
			&resource.CreatedAt, &resource.UpdatedAt, &resource.LastHealthCheck,
			&resource.ErrorMessage)

		if err != nil {
			return nil, fmt.Errorf("failed to scan resource instance: %w", err)
		}

		// Unmarshal JSON fields
		if len(configJSON) > 0 {
			if err := json.Unmarshal(configJSON, &resource.Configuration); err != nil {
				return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
			}
		}

		if len(providerMetadataJSON) > 0 {
			if err := json.Unmarshal(providerMetadataJSON, &resource.ProviderMetadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal provider metadata: %w", err)
			}
		}

		resources = append(resources, &resource)
	}

	return resources, nil
}

// UpdateResourceInstanceState updates the state of a resource instance with audit trail
func (r *ResourceRepository) UpdateResourceInstanceState(id int64, newState ResourceLifecycleState, reason, transitionedBy string, metadata map[string]interface{}) error {
	// Start transaction
	tx, err := r.db.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Ignore rollback error as commit supersedes it

	// Get current state
	var currentState string
	err = tx.QueryRow("SELECT state FROM resource_instances WHERE id = $1", id).Scan(&currentState)
	if err != nil {
		return fmt.Errorf("failed to get current state: %w", err)
	}

	// Update resource state
	_, err = tx.Exec("UPDATE resource_instances SET state = $1 WHERE id = $2", string(newState), id)
	if err != nil {
		return fmt.Errorf("failed to update resource state: %w", err)
	}

	// Create state transition record
	metadataJSON, _ := json.Marshal(metadata)
	_, err = tx.Exec(`
		INSERT INTO resource_state_transitions
		(resource_instance_id, from_state, to_state, reason, transitioned_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, currentState, string(newState), reason, transitionedBy, metadataJSON)

	if err != nil {
		return fmt.Errorf("failed to create state transition record: %w", err)
	}

	return tx.Commit()
}

// UpdateResourceInstanceHealth updates the health status of a resource instance
func (r *ResourceRepository) UpdateResourceInstanceHealth(id int64, healthStatus string, errorMessage *string) error {
	query := `
		UPDATE resource_instances
		SET health_status = $1, last_health_check = NOW(), error_message = $2
		WHERE id = $3`

	_, err := r.db.db.Exec(query, healthStatus, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update resource health: %w", err)
	}

	return nil
}

// CreateHealthCheck records a health check result
func (r *ResourceRepository) CreateHealthCheck(resourceID int64, checkType, status string, responseTime *int64, errorMessage *string, metrics map[string]interface{}) error {
	metricsJSON, _ := json.Marshal(metrics)

	query := `
		INSERT INTO resource_health_checks
		(resource_instance_id, check_type, status, response_time, error_message, metrics)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.db.Exec(query, resourceID, checkType, status, responseTime, errorMessage, metricsJSON)
	if err != nil {
		return fmt.Errorf("failed to create health check record: %w", err)
	}

	return nil
}

// GetResourceStateTransitions retrieves state transitions for a resource
func (r *ResourceRepository) GetResourceStateTransitions(resourceID int64, limit int) ([]*ResourceStateTransition, error) {
	query := `
		SELECT id, resource_instance_id, from_state, to_state, reason, transitioned_by, transitioned_at, metadata
		FROM resource_state_transitions
		WHERE resource_instance_id = $1
		ORDER BY transitioned_at DESC
		LIMIT $2`

	rows, err := r.db.db.Query(query, resourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get state transitions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var transitions []*ResourceStateTransition
	for rows.Next() {
		var transition ResourceStateTransition
		var metadataJSON []byte

		err := rows.Scan(
			&transition.ID, &transition.ResourceInstanceID, &transition.FromState,
			&transition.ToState, &transition.Reason, &transition.TransitionedBy,
			&transition.TransitionedAt, &metadataJSON)

		if err != nil {
			return nil, fmt.Errorf("failed to scan state transition: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &transition.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal transition metadata: %w", err)
			}
		}

		transitions = append(transitions, &transition)
	}

	return transitions, nil
}

// DeleteResourceInstance deletes a resource instance and all related data
func (r *ResourceRepository) DeleteResourceInstance(id int64) error {
	query := "DELETE FROM resource_instances WHERE id = $1"
	result, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete resource instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("resource instance not found")
	}

	return nil
}