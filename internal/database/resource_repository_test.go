package database

import (
	"testing"
)

// ===== ResourceRepository Tests =====

func setupTestResourceRepo(t *testing.T) *ResourceRepository {
	db, err := NewDatabase()
	if err != nil {
		t.Skipf("Database connection failed: %v", err)
	}

	repo := NewResourceRepository(db)
	return repo
}

func TestNewResourceRepository(t *testing.T) {
	db, err := NewDatabase()
	if err != nil {
		t.Skipf("Database connection failed: %v", err)
	}

	repo := NewResourceRepository(db)
	if repo == nil {
		t.Fatal("NewResourceRepository() returned nil")
	}

	if repo.db == nil {
		t.Error("ResourceRepository db is nil")
	}
}

func TestResourceRepository_CreateResourceInstance(t *testing.T) {
	repo := setupTestResourceRepo(t)

	config := map[string]interface{}{
		"size":   "10Gi",
		"region": "us-east-1",
	}

	resource, err := repo.CreateResourceInstance("test-app", "postgres-db", "postgres", config)
	if err != nil {
		t.Fatalf("CreateResourceInstance() error = %v", err)
	}

	if resource.ID == 0 {
		t.Error("CreateResourceInstance() returned resource with ID 0")
	}

	if resource.ApplicationName != "test-app" {
		t.Errorf("ApplicationName = %v, want test-app", resource.ApplicationName)
	}

	if resource.ResourceName != "postgres-db" {
		t.Errorf("ResourceName = %v, want postgres-db", resource.ResourceName)
	}

	if resource.ResourceType != "postgres" {
		t.Errorf("ResourceType = %v, want postgres", resource.ResourceType)
	}

	if resource.State != ResourceStateRequested {
		t.Errorf("State = %v, want %v", resource.State, ResourceStateRequested)
	}

	if resource.HealthStatus != "unknown" {
		t.Errorf("HealthStatus = %v, want unknown", resource.HealthStatus)
	}

	if resource.Configuration["size"] != "10Gi" {
		t.Errorf("Configuration[size] = %v, want 10Gi", resource.Configuration["size"])
	}
}

func TestResourceRepository_GetResourceInstance(t *testing.T) {
	repo := setupTestResourceRepo(t)

	config := map[string]interface{}{
		"replicas": 3,
		"version":  "14.5",
	}

	created, _ := repo.CreateResourceInstance("test-app", "redis", "redis", config)

	// Get resource by ID
	retrieved, err := repo.GetResourceInstance(created.ID)
	if err != nil {
		t.Fatalf("GetResourceInstance() error = %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, created.ID)
	}

	if retrieved.ResourceName != "redis" {
		t.Errorf("ResourceName = %v, want redis", retrieved.ResourceName)
	}

	// Verify configuration was properly unmarshaled
	if retrieved.Configuration["replicas"] != float64(3) { // JSON unmarshals numbers as float64
		t.Errorf("Configuration[replicas] = %v, want 3", retrieved.Configuration["replicas"])
	}
}

func TestResourceRepository_GetResourceInstance_NotFound(t *testing.T) {
	repo := setupTestResourceRepo(t)

	_, err := repo.GetResourceInstance(999999)
	if err == nil {
		t.Error("GetResourceInstance() should return error for non-existent ID")
	}
}

func TestResourceRepository_GetResourceInstanceByName(t *testing.T) {
	repo := setupTestResourceRepo(t)

	config := map[string]interface{}{
		"bucket": "my-bucket",
	}

	created, _ := repo.CreateResourceInstance("test-app", "storage", "s3", config)

	// Get resource by name
	retrieved, err := repo.GetResourceInstanceByName("test-app", "storage")
	if err != nil {
		t.Fatalf("GetResourceInstanceByName() error = %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf("ID = %v, want %v", retrieved.ID, created.ID)
	}

	if retrieved.Configuration["bucket"] != "my-bucket" {
		t.Errorf("Configuration[bucket] = %v, want my-bucket", retrieved.Configuration["bucket"])
	}
}

func TestResourceRepository_GetResourceInstanceByName_NotFound(t *testing.T) {
	repo := setupTestResourceRepo(t)

	_, err := repo.GetResourceInstanceByName("nonexistent-app", "nonexistent-resource")
	if err == nil {
		t.Error("GetResourceInstanceByName() should return error for non-existent resource")
	}
}

func TestResourceRepository_ListResourceInstances(t *testing.T) {
	repo := setupTestResourceRepo(t)

	// Create multiple resources for same app
	_, _ = repo.CreateResourceInstance("list-app", "resource-1", "postgres", map[string]interface{}{})
	_, _ = repo.CreateResourceInstance("list-app", "resource-2", "redis", map[string]interface{}{})
	_, _ = repo.CreateResourceInstance("list-app", "resource-3", "s3", map[string]interface{}{})

	resources, err := repo.ListResourceInstances("list-app")
	if err != nil {
		t.Fatalf("ListResourceInstances() error = %v", err)
	}

	if len(resources) < 3 {
		t.Errorf("ListResourceInstances() count = %v, want >= 3", len(resources))
	}

	// Verify resources are ordered by created_at ASC
	if len(resources) >= 2 {
		if resources[1].CreatedAt.Before(resources[0].CreatedAt) {
			t.Error("Resources not ordered by created_at ASC")
		}
	}
}

func TestResourceRepository_ListResourceInstances_EmptyApp(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resources, err := repo.ListResourceInstances("empty-app-no-resources")
	if err != nil {
		t.Fatalf("ListResourceInstances() error = %v", err)
	}

	if len(resources) != 0 {
		t.Errorf("ListResourceInstances() for empty app should return 0 resources, got %v", len(resources))
	}
}

func TestResourceRepository_UpdateResourceInstanceState(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	// Update state to provisioning
	metadata := map[string]interface{}{
		"provider": "aws",
		"region":   "us-east-1",
	}

	err := repo.UpdateResourceInstanceState(
		resource.ID,
		ResourceStateProvisioning,
		"starting provisioning",
		"system",
		metadata,
	)
	if err != nil {
		t.Fatalf("UpdateResourceInstanceState() error = %v", err)
	}

	// Verify state was updated
	updated, _ := repo.GetResourceInstance(resource.ID)
	if updated.State != ResourceStateProvisioning {
		t.Errorf("State = %v, want %v", updated.State, ResourceStateProvisioning)
	}

	// Verify state transition was recorded
	transitions, err := repo.GetResourceStateTransitions(resource.ID, 10)
	if err != nil {
		t.Fatalf("GetResourceStateTransitions() error = %v", err)
	}

	if len(transitions) != 1 {
		t.Fatalf("Expected 1 state transition, got %v", len(transitions))
	}

	if transitions[0].FromState != ResourceStateRequested {
		t.Errorf("FromState = %v, want %v", transitions[0].FromState, ResourceStateRequested)
	}

	if transitions[0].ToState != ResourceStateProvisioning {
		t.Errorf("ToState = %v, want %v", transitions[0].ToState, ResourceStateProvisioning)
	}

	if transitions[0].Reason != "starting provisioning" {
		t.Errorf("Reason = %v, want 'starting provisioning'", transitions[0].Reason)
	}

	if transitions[0].TransitionedBy != "system" {
		t.Errorf("TransitionedBy = %v, want system", transitions[0].TransitionedBy)
	}
}

func TestResourceRepository_UpdateResourceInstanceHealth(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	// Update health status
	err := repo.UpdateResourceInstanceHealth(resource.ID, "healthy", nil)
	if err != nil {
		t.Fatalf("UpdateResourceInstanceHealth() error = %v", err)
	}

	// Verify health was updated
	updated, err := repo.GetResourceInstance(resource.ID)
	if err != nil {
		t.Fatalf("GetResourceInstance() error = %v", err)
	}

	if updated.HealthStatus != "healthy" {
		t.Errorf("HealthStatus = %v, want healthy", updated.HealthStatus)
	}

	if updated.LastHealthCheck == nil {
		t.Error("LastHealthCheck should be set after health update")
	}
}

func TestResourceRepository_UpdateResourceInstanceHealth_WithError(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	errorMsg := "connection timeout"
	err := repo.UpdateResourceInstanceHealth(resource.ID, "unhealthy", &errorMsg)
	if err != nil {
		t.Fatalf("UpdateResourceInstanceHealth() error = %v", err)
	}

	updated, err := repo.GetResourceInstance(resource.ID)
	if err != nil {
		t.Fatalf("GetResourceInstance() error = %v", err)
	}

	if updated.HealthStatus != "unhealthy" {
		t.Errorf("HealthStatus = %v, want unhealthy", updated.HealthStatus)
	}

	if updated.ErrorMessage == nil || *updated.ErrorMessage != errorMsg {
		t.Errorf("ErrorMessage = %v, want %v", updated.ErrorMessage, errorMsg)
	}
}

func TestResourceRepository_UpdateResourceHints(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	hints := []ResourceHint{
		{
			Type:  "url",
			Label: "Performance Optimization",
			Value: "Consider increasing buffer pool size",
			Icon:  "info",
		},
		{
			Type:  "url",
			Label: "Security Warning",
			Value: "SSL encryption is disabled",
			Icon:  "warning",
		},
	}

	err := repo.UpdateResourceHints(resource.ID, hints)
	if err != nil {
		t.Fatalf("UpdateResourceHints() error = %v", err)
	}

	// Verify hints were updated
	updated, err := repo.GetResourceInstance(resource.ID)
	if err != nil {
		t.Fatalf("GetResourceInstance() error = %v", err)
	}

	if len(updated.Hints) != 2 {
		t.Errorf("Hints count = %v, want 2", len(updated.Hints))
	}

	if updated.Hints[0].Type != "url" {
		t.Errorf("Hints[0].Type = %v, want url", updated.Hints[0].Type)
	}

	if updated.Hints[1].Icon != "warning" {
		t.Errorf("Hints[1].Icon = %v, want warning", updated.Hints[1].Icon)
	}
}

func TestResourceRepository_CreateHealthCheck(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "api", "service", map[string]interface{}{})

	responseTime := int64(150)
	metrics := map[string]interface{}{
		"cpu":    "45%",
		"memory": "512MB",
	}

	err := repo.CreateHealthCheck(resource.ID, "http", "healthy", &responseTime, nil, metrics)
	if err != nil {
		t.Fatalf("CreateHealthCheck() error = %v", err)
	}

	// Note: We don't have a GetHealthChecks method, so we just verify no error
}

func TestResourceRepository_CreateHealthCheck_WithError(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "api", "service", map[string]interface{}{})

	errorMsg := "connection refused"
	err := repo.CreateHealthCheck(resource.ID, "tcp", "unhealthy", nil, &errorMsg, map[string]interface{}{})
	if err != nil {
		t.Fatalf("CreateHealthCheck() error = %v", err)
	}
}

func TestResourceRepository_GetResourceStateTransitions(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	// Create multiple state transitions
	_ = repo.UpdateResourceInstanceState(resource.ID, ResourceStateProvisioning, "start", "system", nil)
	_ = repo.UpdateResourceInstanceState(resource.ID, ResourceStateActive, "ready", "system", nil)
	_ = repo.UpdateResourceInstanceState(resource.ID, ResourceStateDegraded, "high load", "monitor", nil)

	transitions, err := repo.GetResourceStateTransitions(resource.ID, 10)
	if err != nil {
		t.Fatalf("GetResourceStateTransitions() error = %v", err)
	}

	if len(transitions) < 3 {
		t.Errorf("GetResourceStateTransitions() count = %v, want >= 3", len(transitions))
	}

	// Verify transitions are ordered by transitioned_at DESC (most recent first)
	if len(transitions) >= 2 {
		if transitions[1].TransitionedAt.After(transitions[0].TransitionedAt) {
			t.Error("Transitions not ordered by transitioned_at DESC")
		}
	}
}

func TestResourceRepository_GetResourceStateTransitions_Limit(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "db", "postgres", map[string]interface{}{})

	// Create 5 transitions
	for i := 0; i < 5; i++ {
		state := ResourceStateProvisioning
		if i%2 == 0 {
			state = ResourceStateActive
		}
		_ = repo.UpdateResourceInstanceState(resource.ID, state, "test", "system", nil)
	}

	// Get only 2 most recent
	transitions, err := repo.GetResourceStateTransitions(resource.ID, 2)
	if err != nil {
		t.Fatalf("GetResourceStateTransitions() error = %v", err)
	}

	if len(transitions) != 2 {
		t.Errorf("GetResourceStateTransitions(limit=2) count = %v, want 2", len(transitions))
	}
}

func TestResourceRepository_DeleteResourceInstance(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "temp-db", "postgres", map[string]interface{}{})

	// Delete resource
	err := repo.DeleteResourceInstance(resource.ID)
	if err != nil {
		t.Fatalf("DeleteResourceInstance() error = %v", err)
	}

	// Verify resource was deleted
	_, err = repo.GetResourceInstance(resource.ID)
	if err == nil {
		t.Error("GetResourceInstance() should return error for deleted resource")
	}
}

func TestResourceRepository_DeleteResourceInstance_NotFound(t *testing.T) {
	repo := setupTestResourceRepo(t)

	err := repo.DeleteResourceInstance(999999)
	if err == nil {
		t.Error("DeleteResourceInstance() should return error for non-existent resource")
	}
}

func TestResourceRepository_UpdateExternalResourceState(t *testing.T) {
	repo := setupTestResourceRepo(t)

	resource, _ := repo.CreateResourceInstance("test-app", "external-svc", "service", map[string]interface{}{})

	// Update external state
	err := repo.UpdateExternalResourceState(resource.ID, ExternalStateHealthy, "https://external.example.com/resource/123")
	if err != nil {
		t.Fatalf("UpdateExternalResourceState() error = %v", err)
	}

	// Verify update
	updated, err := repo.GetResourceInstance(resource.ID)
	if err != nil {
		t.Fatalf("GetResourceInstance() error = %v", err)
	}

	if updated.ExternalState == nil || *updated.ExternalState != ExternalStateHealthy {
		t.Errorf("ExternalState = %v, want %v", updated.ExternalState, ExternalStateHealthy)
	}

	if updated.ReferenceURL == nil || *updated.ReferenceURL != "https://external.example.com/resource/123" {
		t.Errorf("ReferenceURL = %v, want https://external.example.com/resource/123", updated.ReferenceURL)
	}

	if updated.LastSync == nil {
		t.Error("LastSync should be set after external state update")
	}
}

func TestResourceRepository_UpdateExternalResourceState_NotFound(t *testing.T) {
	repo := setupTestResourceRepo(t)

	err := repo.UpdateExternalResourceState(999999, ExternalStateHealthy, "https://example.com")
	if err != ErrResourceNotFound {
		t.Errorf("UpdateExternalResourceState() error = %v, want ErrResourceNotFound", err)
	}
}

func TestResourceRepository_GetDelegatedResources(t *testing.T) {
	repo := setupTestResourceRepo(t)

	// Create a mix of resource types
	native1, _ := repo.CreateResourceInstance("delegated-app", "native-1", "postgres", map[string]interface{}{})
	// Set type to native
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeNative, native1.ID)

	delegated1, _ := repo.CreateResourceInstance("delegated-app", "delegated-1", "argocd", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeDelegated, delegated1.ID)

	delegated2, _ := repo.CreateResourceInstance("delegated-app", "delegated-2", "vault", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeDelegated, delegated2.ID)

	// Get only delegated resources
	resources, err := repo.GetDelegatedResources("delegated-app")
	if err != nil {
		t.Fatalf("GetDelegatedResources() error = %v", err)
	}

	// Should only return delegated resources
	foundNative := false
	foundDelegated := 0

	for _, r := range resources {
		if r.Type == ResourceTypeNative {
			foundNative = true
		}
		if r.Type == ResourceTypeDelegated {
			foundDelegated++
		}
	}

	if foundNative {
		t.Error("GetDelegatedResources() returned native resource")
	}

	if foundDelegated < 2 {
		t.Errorf("GetDelegatedResources() returned %d delegated resources, want >= 2", foundDelegated)
	}
}

func TestResourceRepository_FilterResourcesByType_WithApp(t *testing.T) {
	repo := setupTestResourceRepo(t)

	// Create resources
	r1, _ := repo.CreateResourceInstance("filter-app", "r1", "postgres", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeNative, r1.ID)

	r2, _ := repo.CreateResourceInstance("filter-app", "r2", "redis", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeDelegated, r2.ID)

	// Filter by app and type
	resources, err := repo.FilterResourcesByType("filter-app", ResourceTypeNative)
	if err != nil {
		t.Fatalf("FilterResourcesByType() error = %v", err)
	}

	// Verify only native resources returned
	for _, r := range resources {
		if r.ApplicationName != "filter-app" {
			t.Errorf("Resource app = %v, want filter-app", r.ApplicationName)
		}
		if r.Type != ResourceTypeNative {
			t.Errorf("Resource type = %v, want %v", r.Type, ResourceTypeNative)
		}
	}
}

func TestResourceRepository_FilterResourcesByType_WithoutApp(t *testing.T) {
	repo := setupTestResourceRepo(t)

	// Create resources in different apps
	r1, _ := repo.CreateResourceInstance("app1", "r1", "postgres", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeExternal, r1.ID)

	r2, _ := repo.CreateResourceInstance("app2", "r2", "redis", map[string]interface{}{})
	_, _ = repo.db.db.Exec("UPDATE resource_instances SET type = $1 WHERE id = $2", ResourceTypeExternal, r2.ID)

	// Filter by type only (no app filter)
	resources, err := repo.FilterResourcesByType("", ResourceTypeExternal)
	if err != nil {
		t.Fatalf("FilterResourcesByType() error = %v", err)
	}

	// Verify all returned resources are external type
	externalCount := 0
	for _, r := range resources {
		if r.Type == ResourceTypeExternal {
			externalCount++
		}
	}

	if externalCount < 2 {
		t.Errorf("FilterResourcesByType() returned %d external resources, want >= 2", externalCount)
	}
}

func TestResourceRepository_ErrResourceNotFound(t *testing.T) {
	if ErrResourceNotFound == nil {
		t.Error("ErrResourceNotFound should be defined")
	}

	if ErrResourceNotFound.Error() != "resource not found" {
		t.Errorf("ErrResourceNotFound message = %v, want 'resource not found'", ErrResourceNotFound.Error())
	}
}
