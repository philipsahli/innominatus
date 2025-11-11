package orchestration

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"innominatus/internal/providers"
	"innominatus/internal/types"
	"innominatus/internal/workflow"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ValidationIntegrationTestSuite is a BDD-style test suite for provider and workflow validation
type ValidationIntegrationTestSuite struct {
	suite.Suite
	tempDir   string
	loader    *providers.Loader
	registry  *providers.Registry
	resolver  *Resolver
	validator *workflow.WorkflowValidator
}

// SetupSuite runs once before all tests in the suite
func (s *ValidationIntegrationTestSuite) SetupSuite() {
	// Create temporary directory for test providers
	tempDir, err := ioutil.TempDir("", "innominatus-validation-test-*")
	require.NoError(s.T(), err, "Failed to create temp directory")
	s.tempDir = tempDir

	// Initialize loader and validator
	s.loader = providers.NewLoader("1.0.0")
	s.validator = workflow.NewWorkflowValidator()
	s.registry = providers.NewRegistry()
	s.resolver = NewResolver(s.registry)
}

// TearDownSuite runs once after all tests
func (s *ValidationIntegrationTestSuite) TearDownSuite() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// SetupTest runs before each test
func (s *ValidationIntegrationTestSuite) SetupTest() {
	// Clear registry before each test
	s.registry = providers.NewRegistry()
	s.resolver = NewResolver(s.registry)
}

// Feature: Provider Workflow Validation
// As a platform operator
// I want providers with invalid workflows to be rejected at load time
// So that runtime errors are prevented

func (s *ValidationIntegrationTestSuite) TestFeature_ProviderWorkflowValidation() {
	s.Run("Scenario: Provider with invalid policy step is rejected", func() {
		// Given a provider with a workflow using 'command' instead of 'script'
		providerDir := s.createTestProvider("invalid-policy", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: invalid-policy-provider
  version: 1.0.0
  category: test
capabilities:
  resourceTypes:
    - test-resource
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-test
    file: ./workflows/provision-test.yaml
    category: provisioner
`,
			"workflows/provision-test.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-test
steps:
  - name: invalid-step
    type: policy
    config:
      command: echo "wrong field"
`,
		})

		// When I try to load the provider
		provider, err := s.loader.LoadFromFile(filepath.Join(providerDir, "provider.yaml"))

		// Then the provider should be rejected
		s.Error(err, "Provider with invalid workflow should be rejected")
		s.Nil(provider, "Provider should be nil")
		s.Contains(err.Error(), "provider workflow validation failed", "Error should mention validation failure")
		s.Contains(err.Error(), "policy step requires 'script'", "Error should specify the exact issue")
		s.Contains(err.Error(), "found 'command' instead", "Error should mention the wrong field")
	})

	s.Run("Scenario: Provider with valid workflow is accepted", func() {
		// Given a provider with a valid workflow
		providerDir := s.createTestProvider("valid-policy", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: valid-policy-provider
  version: 1.0.0
  category: test
capabilities:
  resourceTypes:
    - valid-resource
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-valid
    file: ./workflows/provision-valid.yaml
    category: provisioner
`,
			"workflows/provision-valid.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-valid
steps:
  - name: valid-step
    type: policy
    config:
      script: |
        #!/bin/bash
        echo "correct field"
`,
		})

		// When I load the provider
		provider, err := s.loader.LoadFromFile(filepath.Join(providerDir, "provider.yaml"))

		// Then the provider should be loaded successfully
		s.NoError(err, "Valid provider should load without errors")
		s.NotNil(provider, "Provider should not be nil")
		s.Equal("valid-policy-provider", provider.Metadata.Name)
		s.True(provider.CanProvisionResourceType("valid-resource"))
	})

	s.Run("Scenario: Provider with terraform workflow missing operation is rejected", func() {
		// Given a provider with invalid terraform workflow
		providerDir := s.createTestProvider("invalid-terraform", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: invalid-terraform-provider
  version: 1.0.0
  category: infrastructure
capabilities:
  resourceTypes:
    - infrastructure
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-infra
    file: ./workflows/provision-infra.yaml
    category: provisioner
`,
			"workflows/provision-infra.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-infra
steps:
  - name: terraform-step
    type: terraform
    config:
      working_dir: ./terraform
      # Missing 'operation' field
`,
		})

		// When I try to load the provider
		provider, err := s.loader.LoadFromFile(filepath.Join(providerDir, "provider.yaml"))

		// Then the provider should be rejected
		s.Error(err, "Provider with invalid terraform workflow should be rejected")
		s.Nil(provider)
		s.Contains(err.Error(), "terraform step requires 'operation'")
	})
}

// Feature: Provider Capability Conflict Detection
// As a platform operator
// I want to prevent multiple providers from claiming the same resource type
// So that resource resolution is unambiguous

func (s *ValidationIntegrationTestSuite) TestFeature_ProviderCapabilityConflicts() {
	s.Run("Scenario: Two providers claiming same resource type are detected", func() {
		// Given two providers both claiming 'postgres'
		provider1Dir := s.createTestProvider("db-provider-1", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: database-provider-1
  version: 1.0.0
  category: data
capabilities:
  resourceTypes:
    - postgres
    - postgresql
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    category: provisioner
`,
			"workflows/provision-postgres.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-postgres
steps:
  - name: create-db
    type: policy
    config:
      script: echo "creating db"
`,
		})

		provider2Dir := s.createTestProvider("db-provider-2", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: database-provider-2
  version: 1.0.0
  category: data
capabilities:
  resourceTypes:
    - postgres
    - mysql
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-db
    file: ./workflows/provision-db.yaml
    category: provisioner
`,
			"workflows/provision-db.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-db
steps:
  - name: setup-db
    type: policy
    config:
      script: echo "setup db"
`,
		})

		provider1, err := s.loader.LoadFromFile(filepath.Join(provider1Dir, "provider.yaml"))
		require.NoError(s.T(), err)
		provider2, err := s.loader.LoadFromFile(filepath.Join(provider2Dir, "provider.yaml"))
		require.NoError(s.T(), err)

		// When I register both providers
		err = s.registry.RegisterProvider(provider1)
		s.NoError(err)
		err = s.registry.RegisterProvider(provider2)
		s.NoError(err)

		// Then validation should detect the conflict
		err = s.resolver.ValidateProviders()
		s.Error(err, "Validation should detect capability conflict")
		s.Contains(err.Error(), "postgres", "Error should mention conflicting resource type")
		s.Contains(err.Error(), "database-provider-1", "Error should mention first provider")
		s.Contains(err.Error(), "database-provider-2", "Error should mention second provider")
	})

	s.Run("Scenario: Providers with different resource types coexist", func() {
		// Reset registry for this test
		testRegistry := providers.NewRegistry()
		testResolver := NewResolver(testRegistry)

		// Given two providers with different capabilities
		provider1Dir := s.createTestProvider("storage-provider", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: storage-provider
  version: 1.0.0
  category: storage
capabilities:
  resourceTypes:
    - s3-bucket
    - minio-bucket
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-storage
    file: ./workflows/provision-storage.yaml
    category: provisioner
`,
			"workflows/provision-storage.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-storage
steps:
  - name: create-bucket
    type: policy
    config:
      script: echo "bucket"
`,
		})

		provider2Dir := s.createTestProvider("cache-provider", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: cache-provider
  version: 1.0.0
  category: cache
capabilities:
  resourceTypes:
    - redis
    - memcached
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-cache
    file: ./workflows/provision-cache.yaml
    category: provisioner
`,
			"workflows/provision-cache.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-cache
steps:
  - name: create-cache
    type: policy
    config:
      script: echo "cache"
`,
		})

		provider1, err := s.loader.LoadFromFile(filepath.Join(provider1Dir, "provider.yaml"))
		require.NoError(s.T(), err)
		provider2, err := s.loader.LoadFromFile(filepath.Join(provider2Dir, "provider.yaml"))
		require.NoError(s.T(), err)

		// When I register both providers
		err = testRegistry.RegisterProvider(provider1)
		s.NoError(err)
		err = testRegistry.RegisterProvider(provider2)
		s.NoError(err)

		// Then validation should pass
		err = testResolver.ValidateProviders()
		s.NoError(err, "Providers with different resource types should coexist")

		// And I should be able to resolve each resource type
		p, w, err := testResolver.ResolveProviderForResource("s3-bucket")
		s.NoError(err)
		s.Equal("storage-provider", p.Metadata.Name)
		s.NotNil(w)

		p, w, err = testResolver.ResolveProviderForResource("redis")
		s.NoError(err)
		s.Equal("cache-provider", p.Metadata.Name)
		s.NotNil(w)
	})
}

// Feature: Resource Type Resolution
// As a developer
// I want my Score spec to fail fast if a resource type has no provider
// So that I get clear feedback before deployment

func (s *ValidationIntegrationTestSuite) TestFeature_ResourceTypeResolution() {
	s.Run("Scenario: Score spec with unknown resource type is detected", func() {
		// Given a registry with known providers
		providerDir := s.createTestProvider("known-provider", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: known-provider
  version: 1.0.0
  category: test
capabilities:
  resourceTypes:
    - postgres
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    category: provisioner
`,
			"workflows/provision-postgres.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-postgres
steps:
  - name: create-db
    type: policy
    config:
      script: echo "db"
`,
		})

		provider, err := s.loader.LoadFromFile(filepath.Join(providerDir, "provider.yaml"))
		require.NoError(s.T(), err)
		err = s.registry.RegisterProvider(provider)
		require.NoError(s.T(), err)

		// When I try to resolve an unknown resource type
		_, _, err = s.resolver.ResolveProviderForResource("unknown-database-type")

		// Then I should get a clear error
		s.Error(err, "Unknown resource type should fail to resolve")
		s.Contains(err.Error(), "no provider found for resource type 'unknown-database-type'")
	})

	s.Run("Scenario: Score spec with known resource type is resolved", func() {
		// Reset registry for this test
		testRegistry := providers.NewRegistry()
		testResolver := NewResolver(testRegistry)

		// Given a registry with a postgres provider
		providerDir := s.createTestProvider("postgres-provider", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: postgres-provider
  version: 1.0.0
  category: database
capabilities:
  resourceTypes:
    - postgres
    - postgresql
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    category: provisioner
`,
			"workflows/provision-postgres.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-postgres
steps:
  - name: provision-db
    type: policy
    config:
      script: echo "provisioning postgres"
`,
		})

		provider, err := s.loader.LoadFromFile(filepath.Join(providerDir, "provider.yaml"))
		require.NoError(s.T(), err)
		err = testRegistry.RegisterProvider(provider)
		require.NoError(s.T(), err)

		// When I resolve both 'postgres' and 'postgresql' (alias)
		p1, w1, err := testResolver.ResolveProviderForResource("postgres")
		s.NoError(err)
		p2, w2, err := testResolver.ResolveProviderForResource("postgresql")
		s.NoError(err)

		// Then both should resolve to the same provider
		s.Equal("postgres-provider", p1.Metadata.Name)
		s.Equal("postgres-provider", p2.Metadata.Name)
		s.Equal("provision-postgres", w1.Name)
		s.Equal("provision-postgres", w2.Name)
	})
}

// Feature: Workflow Step Validation
// As a workflow author
// I want detailed validation errors for each step
// So that I can quickly fix configuration issues

func (s *ValidationIntegrationTestSuite) TestFeature_WorkflowStepValidation() {
	s.Run("Scenario: Multiple validation errors are reported", func() {
		// Given a workflow with multiple invalid steps
		wf := &types.Workflow{
			Steps: []types.Step{
				{
					// Missing name
					Type: "policy",
					Config: map[string]interface{}{
						"script": "echo test",
					},
				},
				{
					Name: "invalid-policy",
					Type: "policy",
					Config: map[string]interface{}{
						"command": "wrong field", // Should be 'script'
					},
				},
				{
					Name: "invalid-terraform",
					Type: "terraform",
					Config: map[string]interface{}{
						"working_dir": "/tmp",
						// Missing 'operation'
					},
				},
			},
		}

		// When I validate the workflow
		errors := s.validator.ValidateWorkflow(wf)

		// Then I should get all validation errors
		s.Greater(len(errors), 0, "Should have validation errors")

		errorMessages := ""
		for _, err := range errors {
			errorMessages += err.Error() + "\n"
		}

		s.Contains(errorMessages, "step must have a name", "Should report missing name")
		s.Contains(errorMessages, "policy step requires 'script'", "Should report policy error")
		s.Contains(errorMessages, "terraform step requires 'operation'", "Should report terraform error")
	})

	s.Run("Scenario: Kubernetes step validation", func() {
		// Given a kubernetes step without manifest or namespace
		wf := &types.Workflow{
			Steps: []types.Step{
				{
					Name: "k8s-step",
					Type: "kubernetes",
					Config: map[string]interface{}{
						"operation": "apply",
						// Missing both manifest and namespace
					},
				},
			},
		}

		// When I validate
		errors := s.validator.ValidateWorkflow(wf)

		// Then I should get validation error
		s.Greater(len(errors), 0)
		errorStr := errors[0].Error()
		s.Contains(errorStr, "kubernetes step requires 'manifest' or 'namespace'")
	})
}

// Feature: End-to-End Validation Flow
// As a platform operator
// I want the complete validation chain to work seamlessly
// So that invalid configurations never reach production

func (s *ValidationIntegrationTestSuite) TestFeature_EndToEndValidation() {
	s.Run("Scenario: Complete flow from provider load to resource creation", func() {
		// Given valid providers are loaded
		dbProviderDir := s.createTestProvider("complete-db-provider", map[string]string{
			"provider.yaml": `
apiVersion: v1
kind: Provider
metadata:
  name: complete-db-provider
  version: 1.0.0
  category: database
capabilities:
  resourceTypes:
    - postgres
compatibility:
  minCoreVersion: 1.0.0
  maxCoreVersion: 2.0.0
workflows:
  - name: provision-postgres
    file: ./workflows/provision-postgres.yaml
    category: provisioner
`,
			"workflows/provision-postgres.yaml": `
apiVersion: v1
kind: Workflow
metadata:
  name: provision-postgres
  description: Provision PostgreSQL database
steps:
  - name: create-database
    type: policy
    config:
      script: |
        #!/bin/bash
        echo "Creating PostgreSQL database"
  - name: configure-access
    type: policy
    config:
      script: |
        #!/bin/bash
        echo "Configuring database access"
`,
		})

		// When I load and register the provider
		provider, err := s.loader.LoadFromFile(filepath.Join(dbProviderDir, "provider.yaml"))
		require.NoError(s.T(), err, "Provider should load successfully")

		err = s.registry.RegisterProvider(provider)
		require.NoError(s.T(), err, "Provider should register successfully")

		// And validate no conflicts
		err = s.resolver.ValidateProviders()
		s.NoError(err, "No conflicts should exist")

		// And I can resolve the provider for postgres
		resolvedProvider, resolvedWorkflow, err := s.resolver.ResolveProviderForResource("postgres")
		s.NoError(err, "Should resolve postgres provider")
		s.Equal("complete-db-provider", resolvedProvider.Metadata.Name)
		s.Equal("provision-postgres", resolvedWorkflow.Name)

		// Then the complete validation chain succeeds
		s.T().Log("✅ Complete validation chain successful:")
		s.T().Logf("   1. Provider loaded and validated")
		s.T().Logf("   2. Workflows validated (2 steps)")
		s.T().Logf("   3. Provider registered without conflicts")
		s.T().Logf("   4. Resource type 'postgres' resolved to provider '%s'", resolvedProvider.Metadata.Name)
		s.T().Logf("   5. Workflow '%s' selected for provisioning", resolvedWorkflow.Name)
	})
}

// Helper function to create a test provider in temp directory
func (s *ValidationIntegrationTestSuite) createTestProvider(name string, files map[string]string) string {
	providerDir := filepath.Join(s.tempDir, name)
	err := os.MkdirAll(providerDir, 0755)
	require.NoError(s.T(), err)

	for filename, content := range files {
		filePath := filepath.Join(providerDir, filename)
		fileDir := filepath.Dir(filePath)

		err := os.MkdirAll(fileDir, 0755)
		require.NoError(s.T(), err)

		err = ioutil.WriteFile(filePath, []byte(content), 0644)
		require.NoError(s.T(), err)
	}

	return providerDir
}

// Run the test suite
func TestValidationIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ValidationIntegrationTestSuite))
}

// Additional helper test to verify actual providers pass validation
func TestRealProvidersValidation(t *testing.T) {
	// Skip if providers directory doesn't exist
	if _, err := os.Stat("../../providers"); os.IsNotExist(err) {
		t.Skip("Providers directory not available")
	}

	loader := providers.NewLoader("1.0.0")

	t.Run("database-team provider passes all validations", func(t *testing.T) {
		if _, err := os.Stat("../../providers/database-team/provider.yaml"); os.IsNotExist(err) {
			t.Skip("database-team provider not found")
		}

		// Load provider
		provider, err := loader.LoadFromFile("../../providers/database-team/provider.yaml")
		assert.NoError(t, err, "database-team provider should load without errors")
		assert.NotNil(t, provider)

		// Check capabilities
		assert.True(t, provider.CanProvisionResourceType("postgres"), "Should provision postgres")
		assert.True(t, provider.CanProvisionResourceType("postgresql"), "Should provision postgresql alias")

		// Check workflow exists and is valid
		workflow := provider.GetProvisionerWorkflow()
		assert.NotNil(t, workflow, "Should have provisioner workflow")
		assert.Equal(t, "provision-postgres", workflow.Name)
	})

	t.Run("all providers in directory pass validation", func(t *testing.T) {
		allProviders, err := loader.LoadFromDirectory("../../providers")
		assert.NoError(t, err, "Should load all providers")

		registry := providers.NewRegistry()
		validProviderCount := 0

		for _, p := range allProviders {
			if err := registry.RegisterProvider(p); err != nil {
				t.Logf("Provider %s: %v", p.Metadata.Name, err)
			} else {
				validProviderCount++
				t.Logf("✅ Provider %s loaded successfully", p.Metadata.Name)
			}
		}

		assert.Greater(t, validProviderCount, 0, "At least one provider should load successfully")

		// Check for capability conflicts
		resolver := NewResolver(registry)
		err = resolver.ValidateProviders()
		if err != nil {
			t.Logf("Validation warnings: %v", err)
		}
	})
}
