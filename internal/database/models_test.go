package database

import (
	"encoding/json"
	"testing"
	"time"
)

// ===== WorkflowStepExecution Tests =====

func TestWorkflowStepExecution_SetGetStepConfig(t *testing.T) {
	step := &WorkflowStepExecution{}

	config := map[string]interface{}{
		"operation": "apply",
		"variables": map[string]interface{}{
			"region": "us-east-1",
			"size":   "large",
		},
	}

	err := step.SetStepConfig(config)
	if err != nil {
		t.Fatalf("SetStepConfig() error = %v", err)
	}

	retrieved := step.GetStepConfig()
	if len(retrieved) != 2 {
		t.Errorf("GetStepConfig() returned %d items, want 2", len(retrieved))
	}

	if retrieved["operation"] != "apply" {
		t.Errorf("GetStepConfig()[operation] = %v, want apply", retrieved["operation"])
	}
}

func TestWorkflowStepExecution_GetStepConfigEmpty(t *testing.T) {
	step := &WorkflowStepExecution{}

	config := step.GetStepConfig()
	if config == nil {
		t.Error("GetStepConfig() returned nil, want empty map")
	}

	if len(config) != 0 {
		t.Errorf("GetStepConfig() length = %d, want 0", len(config))
	}
}

func TestWorkflowStepExecution_CalculateDuration(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	step := &WorkflowStepExecution{
		StartedAt:   &startTime,
		CompletedAt: &endTime,
	}

	step.CalculateDuration()

	if step.DurationMs == nil {
		t.Fatal("CalculateDuration() did not set DurationMs")
	}

	// 5 seconds = 5000ms
	if *step.DurationMs < 4999 || *step.DurationMs > 5001 {
		t.Errorf("CalculateDuration() = %dms, want ~5000ms", *step.DurationMs)
	}
}

func TestWorkflowStepExecution_CalculateDurationNoStart(t *testing.T) {
	endTime := time.Now()

	step := &WorkflowStepExecution{
		CompletedAt: &endTime,
	}

	step.CalculateDuration()

	if step.DurationMs != nil {
		t.Error("CalculateDuration() set DurationMs when StartedAt is nil")
	}
}

func TestWorkflowStepExecution_CalculateDurationNoEnd(t *testing.T) {
	startTime := time.Now()

	step := &WorkflowStepExecution{
		StartedAt: &startTime,
	}

	step.CalculateDuration()

	if step.DurationMs != nil {
		t.Error("CalculateDuration() set DurationMs when CompletedAt is nil")
	}
}

// ===== WorkflowStepConfigJSON Tests =====

func TestWorkflowStepConfigJSON_ValueNil(t *testing.T) {
	var config WorkflowStepConfigJSON

	value, err := config.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	if value != nil {
		t.Errorf("Value() = %v, want nil", value)
	}
}

func TestWorkflowStepConfigJSON_ValueWithData(t *testing.T) {
	config := WorkflowStepConfigJSON{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	value, err := config.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	if value == nil {
		t.Fatal("Value() returned nil")
	}

	// Verify it's valid JSON
	jsonBytes, ok := value.([]byte)
	if !ok {
		t.Fatalf("Value() type = %T, want []byte", value)
	}

	var unmarshaled map[string]interface{}
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshaling JSON error = %v", err)
	}

	if unmarshaled["key1"] != "value1" {
		t.Errorf("Unmarshaled[key1] = %v, want value1", unmarshaled["key1"])
	}
}

func TestWorkflowStepConfigJSON_ScanNil(t *testing.T) {
	var config WorkflowStepConfigJSON

	err := config.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error = %v", err)
	}

	if config != nil {
		t.Errorf("Scan(nil) set config to %v, want nil", config)
	}
}

func TestWorkflowStepConfigJSON_ScanBytes(t *testing.T) {
	jsonData := []byte(`{"key":"value","num":42}`)
	var config WorkflowStepConfigJSON

	err := config.Scan(jsonData)
	if err != nil {
		t.Fatalf("Scan(bytes) error = %v", err)
	}

	if config["key"] != "value" {
		t.Errorf("config[key] = %v, want value", config["key"])
	}

	// JSON numbers are decoded as float64
	if config["num"] != float64(42) {
		t.Errorf("config[num] = %v, want 42", config["num"])
	}
}

func TestWorkflowStepConfigJSON_ScanString(t *testing.T) {
	jsonString := `{"key":"value"}`
	var config WorkflowStepConfigJSON

	err := config.Scan(jsonString)
	if err != nil {
		t.Fatalf("Scan(string) error = %v", err)
	}

	if config["key"] != "value" {
		t.Errorf("config[key] = %v, want value", config["key"])
	}
}

func TestWorkflowStepConfigJSON_ScanInvalidType(t *testing.T) {
	var config WorkflowStepConfigJSON

	// Scanning an int should not error (returns nil for unknown types)
	err := config.Scan(123)
	if err != nil {
		t.Fatalf("Scan(int) error = %v", err)
	}
}

// ===== ResourceInstance Tests =====

func TestResourceInstance_SetGetConfiguration(t *testing.T) {
	resource := &ResourceInstance{}

	config := map[string]interface{}{
		"cpu":    "2",
		"memory": "4Gi",
		"replicas": 3,
	}

	err := resource.SetConfiguration(config)
	if err != nil {
		t.Fatalf("SetConfiguration() error = %v", err)
	}

	retrieved := resource.GetConfiguration()
	if len(retrieved) != 3 {
		t.Errorf("GetConfiguration() returned %d items, want 3", len(retrieved))
	}

	if retrieved["cpu"] != "2" {
		t.Errorf("GetConfiguration()[cpu] = %v, want 2", retrieved["cpu"])
	}
}

func TestResourceInstance_GetConfigurationEmpty(t *testing.T) {
	resource := &ResourceInstance{}

	config := resource.GetConfiguration()
	if config == nil {
		t.Error("GetConfiguration() returned nil, want empty map")
	}

	if len(config) != 0 {
		t.Errorf("GetConfiguration() length = %d, want 0", len(config))
	}
}

func TestResourceInstance_SetGetProviderMetadata(t *testing.T) {
	resource := &ResourceInstance{}

	metadata := map[string]interface{}{
		"provider":     "aws",
		"region":       "us-east-1",
		"instance_id":  "i-1234567890",
	}

	err := resource.SetProviderMetadata(metadata)
	if err != nil {
		t.Fatalf("SetProviderMetadata() error = %v", err)
	}

	retrieved := resource.GetProviderMetadata()
	if len(retrieved) != 3 {
		t.Errorf("GetProviderMetadata() returned %d items, want 3", len(retrieved))
	}

	if retrieved["provider"] != "aws" {
		t.Errorf("GetProviderMetadata()[provider] = %v, want aws", retrieved["provider"])
	}
}

func TestResourceInstance_GetProviderMetadataEmpty(t *testing.T) {
	resource := &ResourceInstance{}

	metadata := resource.GetProviderMetadata()
	if metadata == nil {
		t.Error("GetProviderMetadata() returned nil, want empty map")
	}

	if len(metadata) != 0 {
		t.Errorf("GetProviderMetadata() length = %d, want 0", len(metadata))
	}
}

// ===== Resource State Transition Tests =====

func TestResourceInstance_IsValidStateTransition(t *testing.T) {
	tests := []struct {
		name          string
		currentState  ResourceLifecycleState
		newState      ResourceLifecycleState
		expectedValid bool
	}{
		{
			name:          "requested to provisioning",
			currentState:  ResourceStateRequested,
			newState:      ResourceStateProvisioning,
			expectedValid: true,
		},
		{
			name:          "provisioning to active",
			currentState:  ResourceStateProvisioning,
			newState:      ResourceStateActive,
			expectedValid: true,
		},
		{
			name:          "active to scaling",
			currentState:  ResourceStateActive,
			newState:      ResourceStateScaling,
			expectedValid: true,
		},
		{
			name:          "active to updating",
			currentState:  ResourceStateActive,
			newState:      ResourceStateUpdating,
			expectedValid: true,
		},
		{
			name:          "active to degraded",
			currentState:  ResourceStateActive,
			newState:      ResourceStateDegraded,
			expectedValid: true,
		},
		{
			name:          "active to terminating",
			currentState:  ResourceStateActive,
			newState:      ResourceStateTerminating,
			expectedValid: true,
		},
		{
			name:          "terminating to terminated",
			currentState:  ResourceStateTerminating,
			newState:      ResourceStateTerminated,
			expectedValid: true,
		},
		{
			name:          "failed to provisioning (retry)",
			currentState:  ResourceStateFailed,
			newState:      ResourceStateProvisioning,
			expectedValid: true,
		},
		{
			name:          "invalid: requested to active",
			currentState:  ResourceStateRequested,
			newState:      ResourceStateActive,
			expectedValid: false,
		},
		{
			name:          "invalid: active to requested",
			currentState:  ResourceStateActive,
			newState:      ResourceStateRequested,
			expectedValid: false,
		},
		{
			name:          "invalid: terminated to active",
			currentState:  ResourceStateTerminated,
			newState:      ResourceStateActive,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := &ResourceInstance{
				State: tt.currentState,
			}

			isValid := resource.IsValidStateTransition(tt.newState)
			if isValid != tt.expectedValid {
				t.Errorf("IsValidStateTransition(%v -> %v) = %v, want %v",
					tt.currentState, tt.newState, isValid, tt.expectedValid)
			}
		})
	}
}

// ===== Constant Tests =====

func TestWorkflowStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{WorkflowStatusRunning, "running"},
		{WorkflowStatusCompleted, "completed"},
		{WorkflowStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Constant = %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestStepStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{StepStatusPending, "pending"},
		{StepStatusRunning, "running"},
		{StepStatusCompleted, "completed"},
		{StepStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Constant = %v, want %v", tt.constant, tt.expected)
			}
		})
	}
}

func TestResourceTypeConstants(t *testing.T) {
	if ResourceTypeNative != "native" {
		t.Errorf("ResourceTypeNative = %v, want native", ResourceTypeNative)
	}
	if ResourceTypeDelegated != "delegated" {
		t.Errorf("ResourceTypeDelegated = %v, want delegated", ResourceTypeDelegated)
	}
	if ResourceTypeExternal != "external" {
		t.Errorf("ResourceTypeExternal = %v, want external", ResourceTypeExternal)
	}
}

func TestResourceStateConstants(t *testing.T) {
	states := []ResourceLifecycleState{
		ResourceStateRequested,
		ResourceStateProvisioning,
		ResourceStateActive,
		ResourceStateScaling,
		ResourceStateUpdating,
		ResourceStateDegraded,
		ResourceStateTerminating,
		ResourceStateTerminated,
		ResourceStateFailed,
	}

	// Just verify they're all different and not empty
	stateMap := make(map[ResourceLifecycleState]bool)
	for _, state := range states {
		if state == "" {
			t.Errorf("Found empty state constant")
		}
		if stateMap[state] {
			t.Errorf("Duplicate state constant: %v", state)
		}
		stateMap[state] = true
	}

	if len(stateMap) != 9 {
		t.Errorf("Expected 9 unique states, got %d", len(stateMap))
	}
}

func TestExternalStateConstants(t *testing.T) {
	if ExternalStateWaitingExternal != "WaitingExternal" {
		t.Errorf("ExternalStateWaitingExternal = %v", ExternalStateWaitingExternal)
	}
	if ExternalStateBuildingExternal != "BuildingExternal" {
		t.Errorf("ExternalStateBuildingExternal = %v", ExternalStateBuildingExternal)
	}
	if ExternalStateHealthy != "Healthy" {
		t.Errorf("ExternalStateHealthy = %v", ExternalStateHealthy)
	}
	if ExternalStateError != "Error" {
		t.Errorf("ExternalStateError = %v", ExternalStateError)
	}
	if ExternalStateUnknown != "Unknown" {
		t.Errorf("ExternalStateUnknown = %v", ExternalStateUnknown)
	}
}

// ===== Model JSON Marshaling Tests =====

func TestWorkflowExecution_JSONMarshaling(t *testing.T) {
	now := time.Now()
	errorMsg := "test error"

	exec := &WorkflowExecution{
		ID:              123,
		ApplicationName: "test-app",
		WorkflowName:    "deploy",
		Status:          WorkflowStatusRunning,
		StartedAt:       now,
		ErrorMessage:    &errorMsg,
		TotalSteps:      5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Marshal to JSON
	data, err := json.Marshal(exec)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal back
	var unmarshaled WorkflowExecution
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Verify
	if unmarshaled.ID != exec.ID {
		t.Errorf("ID = %v, want %v", unmarshaled.ID, exec.ID)
	}
	if unmarshaled.ApplicationName != exec.ApplicationName {
		t.Errorf("ApplicationName = %v, want %v", unmarshaled.ApplicationName, exec.ApplicationName)
	}
	if unmarshaled.WorkflowName != exec.WorkflowName {
		t.Errorf("WorkflowName = %v, want %v", unmarshaled.WorkflowName, exec.WorkflowName)
	}
}

func TestResourceInstance_JSONMarshaling(t *testing.T) {
	provider := "gitops"
	refURL := "https://github.com/org/repo/pull/123"

	resource := &ResourceInstance{
		ID:              456,
		ApplicationName: "test-app",
		ResourceName:    "database",
		ResourceType:    "postgres",
		State:           ResourceStateActive,
		HealthStatus:    "healthy",
		Type:            ResourceTypeNative,
		Provider:        &provider,
		ReferenceURL:    &refURL,
	}

	// Marshal to JSON
	data, err := json.Marshal(resource)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// Unmarshal back
	var unmarshaled ResourceInstance
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Verify
	if unmarshaled.ID != resource.ID {
		t.Errorf("ID = %v, want %v", unmarshaled.ID, resource.ID)
	}
	if unmarshaled.State != resource.State {
		t.Errorf("State = %v, want %v", unmarshaled.State, resource.State)
	}
	if *unmarshaled.Provider != provider {
		t.Errorf("Provider = %v, want %v", *unmarshaled.Provider, provider)
	}
}
