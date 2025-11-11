package e2e

import (
	"context"
	"testing"

	"innominatus/internal/database"
	"innominatus/internal/graph"
	"innominatus/internal/orchestration"
	"innominatus/internal/providers"
	"innominatus/internal/resources"
	"innominatus/internal/workflow"
	providersdk "innominatus/pkg/sdk"

	"github.com/stretchr/testify/suite"
)

// CRUDLifecycleTestSuite tests the complete CRUD lifecycle: CREATE → UPDATE → DELETE
type CRUDLifecycleTestSuite struct {
	suite.Suite
	db               *database.Database
	providerRegistry *providers.Registry
	resolver         *orchestration.Resolver
	resourceManager  *resources.Manager
	workflowExec     *workflow.WorkflowExecutor
	workflowRepo     *database.WorkflowRepository
	resourceRepo     *database.ResourceRepository
	engine           *orchestration.Engine
	graphAdapter     *graph.Adapter
}

// SetupSuite runs once before all tests
func (s *CRUDLifecycleTestSuite) SetupSuite() {
	// Initialize test database with testcontainer
	testDB := database.SetupTestDatabase(s.T())
	s.db = testDB.DB

	// Create test provider registry with CRUD workflows
	s.providerRegistry = providers.NewRegistry()

	// Register test provider with full CRUD support
	crudProvider := &providersdk.Provider{
		Metadata: providersdk.ProviderMetadata{
			Name:     "crud-test-provider",
			Version:  "1.0.0",
			Category: "data",
		},
		Capabilities: providersdk.ProviderCapabilities{
			// Advanced format with operation-specific workflows
			ResourceTypeCapabilities: []providersdk.ResourceTypeCapability{
				{
					Type: "postgres",
					Operations: map[string]providersdk.OperationWorkflow{
						"create": {
							Workflow: "provision-postgres",
						},
						"update": {
							Workflow: "update-postgres",
						},
						"delete": {
							Workflow: "delete-postgres",
						},
					},
				},
				{
					Type:     "postgresql",
					AliasFor: "postgres",
				},
			},
			// Legacy format for backward compatibility
			ResourceTypes: []string{"postgres", "postgresql"},
		},
		Workflows: []providersdk.WorkflowMetadata{
			{
				Name:      "provision-postgres",
				Category:  "provisioner",
				Operation: "create",
				File:      "workflows/provision-postgres.yaml",
			},
			{
				Name:      "update-postgres",
				Category:  "provisioner",
				Operation: "update",
				File:      "workflows/update-postgres.yaml",
			},
			{
				Name:      "delete-postgres",
				Category:  "provisioner",
				Operation: "delete",
				File:      "workflows/delete-postgres.yaml",
			},
		},
	}

	err := s.providerRegistry.RegisterProvider(crudProvider)
	s.Require().NoError(err, "Failed to register CRUD provider")

	// Create resolver
	s.resolver = orchestration.NewResolver(s.providerRegistry)

	// Initialize repositories
	s.workflowRepo = database.NewWorkflowRepository(s.db)
	s.resourceRepo = database.NewResourceRepository(s.db)

	// Initialize graph adapter
	var graphErr error
	s.graphAdapter, graphErr = graph.NewAdapter(s.db.DB())
	s.Require().NoError(graphErr, "Failed to create graph adapter")

	// Initialize workflow executor
	s.workflowExec = workflow.NewWorkflowExecutor(s.workflowRepo)

	// Initialize resource manager
	s.resourceManager = resources.NewManager(s.resourceRepo)
	s.resourceManager.SetGraphAdapter(s.graphAdapter)

	// Initialize orchestration engine
	s.engine = orchestration.NewEngine(
		s.db,
		s.providerRegistry,
		s.workflowRepo,
		s.resourceRepo,
		s.workflowExec,
		s.graphAdapter,
		"../../providers", // Provider directory
	)
}

// TearDownSuite runs once after all tests
func (s *CRUDLifecycleTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
	}
}

// TestCreateOperation tests CREATE operation workflow resolution
func (s *CRUDLifecycleTestSuite) TestCreateOperation() {
	// Create a postgres resource (default operation is CREATE)
	resource, err := s.resourceManager.CreateResourceInstance(
		"test-app",
		"test-postgres",
		"postgres",
		map[string]interface{}{
			"version":  "15",
			"replicas": 2,
		},
	)

	s.Require().NoError(err, "Failed to create resource")
	s.Require().NotNil(resource)
	s.Equal("postgres", resource.ResourceType)
	s.Equal(database.ResourceStateRequested, resource.State)
	s.Nil(resource.DesiredOperation, "DesiredOperation should be nil (defaults to create)")

	// Test resolver picks correct workflow for CREATE
	provider, workflowMeta, err := s.resolver.ResolveWorkflowForOperation("postgres", "create", nil)
	s.Require().NoError(err, "Failed to resolve CREATE workflow")
	s.Equal("crud-test-provider", provider.Metadata.Name)
	s.Equal("provision-postgres", workflowMeta.Name)
	s.Equal("create", workflowMeta.Operation)
}

// TestUpdateOperation tests UPDATE operation workflow resolution
func (s *CRUDLifecycleTestSuite) TestUpdateOperation() {
	// Test resolver picks correct workflow for UPDATE
	provider, workflowMeta, err := s.resolver.ResolveWorkflowForOperation("postgres", "update", nil)
	s.Require().NoError(err, "Failed to resolve UPDATE workflow")
	s.Require().NotNil(provider, "Provider should not be nil")
	s.Equal("crud-test-provider", provider.Metadata.Name)
	s.Equal("update-postgres", workflowMeta.Name)
	s.Equal("update", workflowMeta.Operation)

	// Verify provider supports UPDATE operation
	s.True(provider.SupportsOperation("postgres", "update"), "Provider should support UPDATE operation")
}

// TestDeleteOperation tests DELETE operation workflow resolution
func (s *CRUDLifecycleTestSuite) TestDeleteOperation() {
	// Test resolver picks correct workflow for DELETE
	provider, workflowMeta, err := s.resolver.ResolveWorkflowForOperation("postgres", "delete", nil)
	s.Require().NoError(err, "Failed to resolve DELETE workflow")
	s.Require().NotNil(provider, "Provider should not be nil")
	s.Equal("crud-test-provider", provider.Metadata.Name)
	s.Equal("delete-postgres", workflowMeta.Name)
	s.Equal("delete", workflowMeta.Operation)

	// Verify provider supports DELETE operation
	s.True(provider.SupportsOperation("postgres", "delete"), "Provider should support DELETE operation")
}

// TestAliasResolution tests that aliases resolve to primary resource type operations
func (s *CRUDLifecycleTestSuite) TestAliasResolution() {
	// Test that "postgresql" alias resolves to "postgres" workflows
	provider, workflowMeta, err := s.resolver.ResolveWorkflowForOperation("postgresql", "create", nil)
	s.Require().NoError(err, "Failed to resolve alias CREATE workflow")
	s.Require().NotNil(provider, "Provider should not be nil")
	s.Equal("provision-postgres", workflowMeta.Name, "Alias should resolve to primary type workflow")

	// Test UPDATE via alias
	_, workflowMeta, err = s.resolver.ResolveWorkflowForOperation("postgresql", "update", nil)
	s.Require().NoError(err, "Failed to resolve alias UPDATE workflow")
	s.Equal("update-postgres", workflowMeta.Name)

	// Test DELETE via alias
	_, workflowMeta, err = s.resolver.ResolveWorkflowForOperation("postgresql", "delete", nil)
	s.Require().NoError(err, "Failed to resolve alias DELETE workflow")
	s.Equal("delete-postgres", workflowMeta.Name)
}

// TestUnsupportedOperation tests error handling for unsupported operations
func (s *CRUDLifecycleTestSuite) TestUnsupportedOperation() {
	// Test that READ operation is not supported (not implemented)
	_, _, err := s.resolver.ResolveWorkflowForOperation("postgres", "read", nil)
	s.Error(err, "Should error when operation is not supported")
	s.Contains(err.Error(), "does not support operation", "Error should indicate operation not supported")
}

// TestUnknownResourceType tests error handling for unknown resource types
func (s *CRUDLifecycleTestSuite) TestUnknownResourceType() {
	// Test that unknown resource type errors appropriately
	_, _, err := s.resolver.ResolveWorkflowForOperation("mysql", "create", nil)
	s.Error(err, "Should error for unknown resource type")
	s.Contains(err.Error(), "no provider found", "Error should indicate no provider found")
}

// TestBackwardCompatibility tests that legacy simple resourceTypes format still works
func (s *CRUDLifecycleTestSuite) TestBackwardCompatibility() {
	// Register a legacy provider (simple format)
	legacyProvider := &providersdk.Provider{
		Metadata: providersdk.ProviderMetadata{
			Name:     "legacy-provider",
			Version:  "1.0.0",
			Category: "data",
		},
		Capabilities: providersdk.ProviderCapabilities{
			// Simple format (no operation mapping)
			ResourceTypes: []string{"redis"},
		},
		Workflows: []providersdk.WorkflowMetadata{
			{
				Name:     "provision-redis",
				Category: "provisioner",
				File:     "workflows/redis.yaml",
			},
		},
	}

	err := s.providerRegistry.RegisterProvider(legacyProvider)
	s.Require().NoError(err, "Failed to register legacy provider")

	// Test that CREATE works with legacy format
	provider, workflowMeta, err := s.resolver.ResolveWorkflowForOperation("redis", "create", nil)
	s.Require().NoError(err, "Legacy provider should support CREATE")
	s.Equal("legacy-provider", provider.Metadata.Name)
	s.Equal("provision-redis", workflowMeta.Name)

	// Test that UPDATE/DELETE are not supported with legacy format
	_, _, err = s.resolver.ResolveWorkflowForOperation("redis", "update", nil)
	s.Error(err, "Legacy provider should not support UPDATE")

	_, _, err = s.resolver.ResolveWorkflowForOperation("redis", "delete", nil)
	s.Error(err, "Legacy provider should not support DELETE")
}

// TestFullCRUDLifecycle tests complete CREATE → UPDATE → DELETE flow
func (s *CRUDLifecycleTestSuite) TestFullCRUDLifecycle() {
	ctx := context.Background()
	appName := "crud-test-app"
	resourceName := "crud-postgres"

	// Step 1: CREATE - Create resource in requested state
	s.T().Log("Step 1: CREATE - Creating postgres resource")
	resource, err := s.resourceRepo.CreateResourceInstance(
		appName,
		resourceName,
		"postgres",
		map[string]interface{}{
			"version":  "15",
			"replicas": 2,
		},
	)
	s.Require().NoError(err, "Failed to create resource")
	s.Equal(database.ResourceStateRequested, resource.State)
	s.Nil(resource.DesiredOperation, "Initial operation should be nil (defaults to create)")

	// Verify CREATE workflow resolution
	_, workflowMeta, err := s.resolver.ResolveWorkflowForOperation(resource.ResourceType, "create", nil)
	s.Require().NoError(err)
	s.Equal("provision-postgres", workflowMeta.Name)
	s.T().Logf("✓ CREATE workflow resolved: %s", workflowMeta.Name)

	// Simulate orchestration engine processing (transition to provisioning)
	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateProvisioning,
		"Workflow started",
		"test-engine",
		nil,
	)
	s.Require().NoError(err)

	// Simulate successful provisioning (transition to active)
	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateActive,
		"Resource provisioned successfully",
		"test-engine",
		nil,
	)
	s.Require().NoError(err)

	// Verify resource is active
	resource, err = s.resourceRepo.GetResourceInstance(resource.ID)
	s.Require().NoError(err)
	s.Equal(database.ResourceStateActive, resource.State)
	s.T().Logf("✓ Resource active: %s (ID: %d)", resourceName, resource.ID)

	// Step 2: UPDATE - Update resource configuration
	s.T().Log("Step 2: UPDATE - Scaling postgres replicas")
	updateOp := "update"
	resource.DesiredOperation = &updateOp
	resource.Configuration = map[string]interface{}{
		"replicas": 5, // Scale from 2 to 5
	}

	// Update in database
	query := `UPDATE resource_instances SET desired_operation = $1 WHERE id = $2`
	_, err = s.db.DB().ExecContext(ctx, query, updateOp, resource.ID)
	s.Require().NoError(err)

	// Verify UPDATE workflow resolution
	_, workflowMeta, err = s.resolver.ResolveWorkflowForOperation(resource.ResourceType, "update", nil)
	s.Require().NoError(err)
	s.Equal("update-postgres", workflowMeta.Name)
	s.T().Logf("✓ UPDATE workflow resolved: %s", workflowMeta.Name)

	// Simulate update processing
	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateUpdating,
		"Update in progress",
		"test-engine",
		nil,
	)
	s.Require().NoError(err)

	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateActive,
		"Update completed successfully",
		"test-engine",
		map[string]interface{}{"replicas": 5},
	)
	s.Require().NoError(err)
	s.T().Logf("✓ Resource updated: replicas scaled to 5")

	// Step 3: DELETE - Delete resource
	s.T().Log("Step 3: DELETE - Deleting postgres resource")
	deleteOp := "delete"
	resource.DesiredOperation = &deleteOp

	query = `UPDATE resource_instances SET desired_operation = $1 WHERE id = $2`
	_, err = s.db.DB().ExecContext(ctx, query, deleteOp, resource.ID)
	s.Require().NoError(err)

	// Verify DELETE workflow resolution
	_, workflowMeta, err = s.resolver.ResolveWorkflowForOperation(resource.ResourceType, "delete", nil)
	s.Require().NoError(err)
	s.Equal("delete-postgres", workflowMeta.Name)
	s.T().Logf("✓ DELETE workflow resolved: %s", workflowMeta.Name)

	// Simulate deletion processing
	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateTerminating,
		"Deletion in progress",
		"test-engine",
		nil,
	)
	s.Require().NoError(err)

	err = s.resourceRepo.UpdateResourceInstanceState(
		resource.ID,
		database.ResourceStateTerminated,
		"Resource deleted successfully",
		"test-engine",
		nil,
	)
	s.Require().NoError(err)

	// Verify final state
	resource, err = s.resourceRepo.GetResourceInstance(resource.ID)
	s.Require().NoError(err)
	s.Equal(database.ResourceStateTerminated, resource.State)
	s.T().Logf("✓ Resource deleted: %s (state: %s)", resourceName, resource.State)

	s.T().Log("✅ Full CRUD lifecycle completed successfully!")
}

// TestRunSuite runs the CRUD lifecycle test suite
func TestCRUDLifecycleSuite(t *testing.T) {
	suite.Run(t, new(CRUDLifecycleTestSuite))
}
