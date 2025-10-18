package sdk_test

import (
	"testing"

	"innominatus/pkg/sdk"
)

func TestMapConfig(t *testing.T) {
	// Create config
	data := map[string]interface{}{
		"name":     "test-db",
		"size":     "large",
		"port":     5432,
		"replicas": 3,
		"enabled":  true,
		"metadata": map[string]interface{}{
			"owner": "platform-team",
			"env":   "production",
		},
		"tags": []interface{}{"production", "critical"},
	}

	config := sdk.NewMapConfig(data)

	// Test GetString
	if config.GetString("name") != "test-db" {
		t.Errorf("Expected name='test-db', got '%s'", config.GetString("name"))
	}

	// Test GetInt
	if config.GetInt("port") != 5432 {
		t.Errorf("Expected port=5432, got %d", config.GetInt("port"))
	}

	if config.GetInt("replicas") != 3 {
		t.Errorf("Expected replicas=3, got %d", config.GetInt("replicas"))
	}

	// Test GetBool
	if !config.GetBool("enabled") {
		t.Error("Expected enabled=true, got false")
	}

	// Test GetMap
	metadata := config.GetMap("metadata")
	if metadata["owner"] != "platform-team" {
		t.Errorf("Expected metadata.owner='platform-team', got '%v'", metadata["owner"])
	}

	// Test GetSlice
	tags := config.GetSlice("tags")
	if len(tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(tags))
	}

	// Test Has
	if !config.Has("name") {
		t.Error("Expected Has('name')=true, got false")
	}

	if config.Has("nonexistent") {
		t.Error("Expected Has('nonexistent')=false, got true")
	}

	// Test Keys
	keys := config.Keys()
	if len(keys) != 7 {
		t.Errorf("Expected 7 keys, got %d", len(keys))
	}

	// Test AsMap
	allConfig := config.AsMap()
	if allConfig["name"] != "test-db" {
		t.Error("AsMap() should return all data")
	}
}

func TestResourceState(t *testing.T) {
	// Test resource state constants
	states := []sdk.ResourceState{
		sdk.ResourceStateRequested,
		sdk.ResourceStateProvisioning,
		sdk.ResourceStateActive,
		sdk.ResourceStateScaling,
		sdk.ResourceStateUpdating,
		sdk.ResourceStateDegraded,
		sdk.ResourceStateTerminating,
		sdk.ResourceStateTerminated,
		sdk.ResourceStateFailed,
	}

	expectedStates := []string{
		"requested",
		"provisioning",
		"active",
		"scaling",
		"updating",
		"degraded",
		"terminating",
		"terminated",
		"failed",
	}

	for i, state := range states {
		if string(state) != expectedStates[i] {
			t.Errorf("Expected state '%s', got '%s'", expectedStates[i], state)
		}
	}
}

func TestResourceHelpers(t *testing.T) {
	// Test IsActive
	resource := &sdk.Resource{State: sdk.ResourceStateActive}
	if !resource.IsActive() {
		t.Error("Expected IsActive()=true for active resource")
	}

	// Test IsFailed
	resource.State = sdk.ResourceStateFailed
	if !resource.IsFailed() {
		t.Error("Expected IsFailed()=true for failed resource")
	}

	// Test IsTerminated
	resource.State = sdk.ResourceStateTerminated
	if !resource.IsTerminated() {
		t.Error("Expected IsTerminated()=true for terminated resource")
	}
}

func TestResourceStatus(t *testing.T) {
	// Test healthy status
	status := &sdk.ResourceStatus{
		State:        sdk.ResourceStateActive,
		HealthStatus: "healthy",
	}

	if !status.IsHealthy() {
		t.Error("Expected IsHealthy()=true for healthy status")
	}

	// Test unhealthy status
	status.HealthStatus = "degraded"
	if status.IsHealthy() {
		t.Error("Expected IsHealthy()=false for degraded status")
	}

	// Test "ok" status
	status.HealthStatus = "ok"
	if !status.IsHealthy() {
		t.Error("Expected IsHealthy()=true for ok status")
	}
}

func TestHintHelpers(t *testing.T) {
	// Test NewURLHint
	urlHint := sdk.NewURLHint("Dashboard", "https://example.com", sdk.IconExternalLink)
	if urlHint.Type != sdk.HintTypeURL {
		t.Errorf("Expected type='%s', got '%s'", sdk.HintTypeURL, urlHint.Type)
	}
	if urlHint.Label != "Dashboard" {
		t.Errorf("Expected label='Dashboard', got '%s'", urlHint.Label)
	}
	if urlHint.Icon != sdk.IconExternalLink {
		t.Errorf("Expected icon='%s', got '%s'", sdk.IconExternalLink, urlHint.Icon)
	}

	// Test NewCommandHint
	cmdHint := sdk.NewCommandHint("Get Pods", "kubectl get pods", sdk.IconTerminal)
	if cmdHint.Type != sdk.HintTypeCommand {
		t.Errorf("Expected type='%s', got '%s'", sdk.HintTypeCommand, cmdHint.Type)
	}

	// Test NewConnectionStringHint
	connHint := sdk.NewConnectionStringHint("Database", "postgres://localhost/db")
	if connHint.Type != sdk.HintTypeConnectionString {
		t.Errorf("Expected type='%s', got '%s'", sdk.HintTypeConnectionString, connHint.Type)
	}
	if connHint.Icon != sdk.IconLock {
		t.Errorf("Expected icon='%s', got '%s'", sdk.IconLock, connHint.Icon)
	}

	// Test NewDashboardHint
	dashHint := sdk.NewDashboardHint("Admin Panel", "https://admin.example.com")
	if dashHint.Type != sdk.HintTypeDashboard {
		t.Errorf("Expected type='%s', got '%s'", sdk.HintTypeDashboard, dashHint.Type)
	}
}

func TestPlatformValidation(t *testing.T) {
	// Valid platform
	validPlatform := &sdk.Platform{
		APIVersion: "innominatus.io/v1",
		Kind:       "Platform",
		Metadata: sdk.PlatformMetadata{
			Name:    "test-platform",
			Version: "1.0.0",
		},
		Compatibility: sdk.PlatformCompatibility{
			MinCoreVersion: "1.0.0",
			MaxCoreVersion: "2.0.0",
		},
		Provisioners: []sdk.ProvisionerMetadata{
			{
				Name:    "test-provisioner",
				Type:    "postgres",
				Version: "1.0.0",
			},
		},
	}

	if err := validPlatform.Validate(); err != nil {
		t.Errorf("Expected valid platform to pass validation, got error: %v", err)
	}

	// Invalid platform - missing name
	invalidPlatform := &sdk.Platform{
		APIVersion: "innominatus.io/v1",
		Kind:       "Platform",
		Metadata: sdk.PlatformMetadata{
			Version: "1.0.0",
		},
		Compatibility: sdk.PlatformCompatibility{
			MinCoreVersion: "1.0.0",
		},
		Provisioners: []sdk.ProvisionerMetadata{
			{Name: "test", Type: "postgres", Version: "1.0.0"},
		},
	}

	if err := invalidPlatform.Validate(); err == nil {
		t.Error("Expected invalid platform to fail validation")
	}
}

func TestPlatformProvisionerLookup(t *testing.T) {
	platform := &sdk.Platform{
		Provisioners: []sdk.ProvisionerMetadata{
			{Name: "postgres-provisioner", Type: "postgres", Version: "1.0.0"},
			{Name: "redis-provisioner", Type: "redis", Version: "1.0.0"},
		},
	}

	// Test GetProvisionerByType
	prov := platform.GetProvisionerByType("postgres")
	if prov == nil {
		t.Error("Expected to find provisioner by type 'postgres'")
	}
	if prov != nil && prov.Name != "postgres-provisioner" {
		t.Errorf("Expected name='postgres-provisioner', got '%s'", prov.Name)
	}

	// Test GetProvisionerByName
	prov = platform.GetProvisionerByName("redis-provisioner")
	if prov == nil {
		t.Error("Expected to find provisioner by name 'redis-provisioner'")
	}
	if prov != nil && prov.Type != "redis" {
		t.Errorf("Expected type='redis', got '%s'", prov.Type)
	}

	// Test not found
	prov = platform.GetProvisionerByType("nonexistent")
	if prov != nil {
		t.Error("Expected nil for nonexistent provisioner type")
	}
}

func TestSDKErrors(t *testing.T) {
	// Test ErrProvisionFailed
	err := sdk.ErrProvisionFailed("database creation failed")
	if err.Code != sdk.ErrCodeProvisionFailed {
		t.Errorf("Expected code='%s', got '%s'", sdk.ErrCodeProvisionFailed, err.Code)
	}

	// Test ErrInvalidConfig
	err = sdk.ErrInvalidConfig("missing required field: %s", "name")
	if err.Message != "missing required field: name" {
		t.Errorf("Expected formatted message, got '%s'", err.Message)
	}

	// Test error string
	errStr := err.Error()
	if errStr == "" {
		t.Error("Expected non-empty error string")
	}
}
