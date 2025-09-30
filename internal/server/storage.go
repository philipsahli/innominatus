package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"innominatus/internal/types"
	"sync"
	"time"
)

type StoredSpec struct {
	Spec *types.ScoreSpec `json:"spec"`
	Team string           `json:"team"`
	CreatedAt time.Time   `json:"created_at"`
	CreatedBy string      `json:"created_by"`
}

type Storage struct {
	specs           map[string]*StoredSpec
	environments    map[string]*Environment
	resourceInstances map[int64]*LocalResourceInstance
	resourcesByApp  map[string][]int64 // application_name -> resource IDs
	nextResourceID  int64
	mu              sync.RWMutex
	dataDir         string // Directory to store persistent data
}

type Environment struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	TTL       string            `json:"ttl"`
	CreatedAt time.Time         `json:"created_at"`
	Status    string            `json:"status"`
	Resources map[string]string `json:"resources"`
}

// LocalResourceInstance represents a resource instance stored locally
type LocalResourceInstance struct {
	ID                  int64                  `json:"id"`
	ApplicationName     string                 `json:"application_name"`
	ResourceName        string                 `json:"resource_name"`
	ResourceType        string                 `json:"resource_type"`
	State               string                 `json:"state"`
	HealthStatus        string                 `json:"health_status"`
	Configuration       map[string]interface{} `json:"configuration"`
	ProviderID          *string                `json:"provider_id,omitempty"`
	ProviderMetadata    map[string]interface{} `json:"provider_metadata,omitempty"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	LastHealthCheck     *time.Time             `json:"last_health_check,omitempty"`
	ErrorMessage        *string                `json:"error_message,omitempty"`
}

func NewStorage() *Storage {
	dataDir := "data"
	storage := &Storage{
		specs:           make(map[string]*StoredSpec),
		environments:    make(map[string]*Environment),
		resourceInstances: make(map[int64]*LocalResourceInstance),
		resourcesByApp:  make(map[string][]int64),
		nextResourceID:  1,
		dataDir:         dataDir,
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create data directory: %v\n", err)
	}

	// Load existing data
	if err := storage.loadFromDisk(); err != nil {
		fmt.Printf("Warning: Failed to load existing data: %v\n", err)
	}

	return storage
}

func (s *Storage) AddSpec(name string, spec *types.ScoreSpec, team string, createdBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	storedSpec := &StoredSpec{
		Spec:      spec,
		Team:      team,
		CreatedAt: time.Now(),
		CreatedBy: createdBy,
	}

	s.specs[name] = storedSpec

	// Create ephemeral environment if specified
	if spec.Environment != nil && spec.Environment.Type == "ephemeral" {
		env := &Environment{
			Name:      name,
			Type:      spec.Environment.Type,
			TTL:       spec.Environment.TTL,
			CreatedAt: time.Now(),
			Status:    "creating",
			Resources: make(map[string]string),
		}

		// Simulate resource creation
		for resourceName, resource := range spec.Resources {
			env.Resources[resourceName] = resource.Type
		}

		s.environments[name] = env
	}

	// Create sample resource instances for each resource in the spec
	for resourceName, resource := range spec.Resources {
		config := make(map[string]interface{})

		// Add some sample configuration based on resource type
		switch resource.Type {
		case "postgres":
			config["database"] = name + "_db"
			config["username"] = name + "_user"
		case "redis":
			config["memory_limit"] = "256MB"
		case "volume":
			config["size"] = "10GB"
		case "route":
			config["host"] = name + ".example.com"
		}

		// Add the resource instance
		resourceInstance := &LocalResourceInstance{
			ID:              s.nextResourceID,
			ApplicationName: name,
			ResourceName:    resourceName,
			ResourceType:    resource.Type,
			State:          "active",
			HealthStatus:   "healthy",
			Configuration:  config,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		// Store the resource
		s.resourceInstances[s.nextResourceID] = resourceInstance

		// Add to app mapping
		s.resourcesByApp[name] = append(s.resourcesByApp[name], s.nextResourceID)

		// Increment next ID
		s.nextResourceID++
	}

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return nil
}

func (s *Storage) GetSpec(name string) (*types.ScoreSpec, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storedSpec, exists := s.specs[name]
	if !exists {
		return nil, false
	}
	return storedSpec.Spec, true
}

func (s *Storage) GetStoredSpec(name string) (*StoredSpec, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	storedSpec, exists := s.specs[name]
	return storedSpec, exists
}

func (s *Storage) ListSpecs() map[string]*types.ScoreSpec {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*types.ScoreSpec)
	for name, storedSpec := range s.specs {
		result[name] = storedSpec.Spec
	}
	return result
}

func (s *Storage) ListSpecsByTeam(team string) map[string]*types.ScoreSpec {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*types.ScoreSpec)
	for name, storedSpec := range s.specs {
		if storedSpec.Team == team {
			result[name] = storedSpec.Spec
		}
	}
	return result
}

func (s *Storage) ListStoredSpecs() map[string]*StoredSpec {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*StoredSpec)
	for name, storedSpec := range s.specs {
		result[name] = storedSpec
	}
	return result
}

func (s *Storage) DeleteSpec(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.specs[name]; !exists {
		return fmt.Errorf("spec '%s' not found", name)
	}

	// Delete associated resource instances
	if resourceIDs, exists := s.resourcesByApp[name]; exists {
		for _, id := range resourceIDs {
			delete(s.resourceInstances, id)
		}
		delete(s.resourcesByApp, name)
	}

	delete(s.specs, name)
	delete(s.environments, name)

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return nil
}

func (s *Storage) GetEnvironment(name string) (*Environment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	env, exists := s.environments[name]
	return env, exists
}

func (s *Storage) ListEnvironments() map[string]*Environment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*Environment)
	for name, env := range s.environments {
		result[name] = env
	}
	return result
}

// Persistence methods

type StorageData struct {
	Specs             map[string]*StoredSpec                 `json:"specs"`
	Environments      map[string]*Environment               `json:"environments"`
	ResourceInstances map[int64]*LocalResourceInstance      `json:"resource_instances"`
	ResourcesByApp    map[string][]int64                    `json:"resources_by_app"`
	NextResourceID    int64                                 `json:"next_resource_id"`
}

// saveToDisk saves the current storage state to disk
func (s *Storage) saveToDisk() error {
	data := StorageData{
		Specs:             s.specs,
		Environments:      s.environments,
		ResourceInstances: s.resourceInstances,
		ResourcesByApp:    s.resourcesByApp,
		NextResourceID:    s.nextResourceID,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal storage data: %w", err)
	}

	filePath := filepath.Join(s.dataDir, "storage.json")
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write storage file: %w", err)
	}

	return nil
}

// loadFromDisk loads storage state from disk
func (s *Storage) loadFromDisk() error {
	filePath := filepath.Join(s.dataDir, "storage.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, start with empty storage
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read storage file: %w", err)
	}

	var storageData StorageData
	if err := json.Unmarshal(data, &storageData); err != nil {
		return fmt.Errorf("failed to unmarshal storage data: %w", err)
	}

	// Load data into memory
	if storageData.Specs != nil {
		s.specs = storageData.Specs
		fmt.Printf("📦 Loaded %d deployed applications from disk\n", len(s.specs))
	}

	if storageData.Environments != nil {
		s.environments = storageData.Environments
		fmt.Printf("🌍 Loaded %d environments from disk\n", len(s.environments))
	}

	if storageData.ResourceInstances != nil {
		s.resourceInstances = storageData.ResourceInstances
		fmt.Printf("🔧 Loaded %d resource instances from disk\n", len(s.resourceInstances))
	}

	if storageData.ResourcesByApp != nil {
		s.resourcesByApp = storageData.ResourcesByApp
	}

	if storageData.NextResourceID > 0 {
		s.nextResourceID = storageData.NextResourceID
	}

	return nil
}

// Resource management methods for local storage

// AddResource creates a new resource instance and stores it locally
func (s *Storage) AddResource(appName, resourceName, resourceType string, config map[string]interface{}) (*LocalResourceInstance, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create new resource instance
	resource := &LocalResourceInstance{
		ID:              s.nextResourceID,
		ApplicationName: appName,
		ResourceName:    resourceName,
		ResourceType:    resourceType,
		State:          "requested",
		HealthStatus:   "unknown",
		Configuration:  config,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Store the resource
	s.resourceInstances[s.nextResourceID] = resource

	// Add to app mapping
	s.resourcesByApp[appName] = append(s.resourcesByApp[appName], s.nextResourceID)

	// Increment next ID
	s.nextResourceID++

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return resource, nil
}

// GetResourcesByApplication returns all resource instances for a specific application
func (s *Storage) GetResourcesByApplication(appName string) ([]*LocalResourceInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resourceIDs, exists := s.resourcesByApp[appName]
	if !exists {
		return []*LocalResourceInstance{}, nil
	}

	resources := make([]*LocalResourceInstance, 0, len(resourceIDs))
	for _, id := range resourceIDs {
		if resource, exists := s.resourceInstances[id]; exists {
			resources = append(resources, resource)
		}
	}

	return resources, nil
}

// GetResource returns a specific resource instance by ID
func (s *Storage) GetResource(id int64) (*LocalResourceInstance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, exists := s.resourceInstances[id]
	if !exists {
		return nil, fmt.Errorf("resource with ID %d not found", id)
	}

	return resource, nil
}

// UpdateResourceHealth updates the health status of a resource
func (s *Storage) UpdateResourceHealth(id int64, healthStatus string, errorMessage *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource, exists := s.resourceInstances[id]
	if !exists {
		return fmt.Errorf("resource with ID %d not found", id)
	}

	resource.HealthStatus = healthStatus
	resource.ErrorMessage = errorMessage
	now := time.Now()
	resource.LastHealthCheck = &now
	resource.UpdatedAt = now

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return nil
}

// UpdateResourceState updates the state of a resource
func (s *Storage) UpdateResourceState(id int64, newState string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource, exists := s.resourceInstances[id]
	if !exists {
		return fmt.Errorf("resource with ID %d not found", id)
	}

	resource.State = newState
	resource.UpdatedAt = time.Now()

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return nil
}

// DeleteResource removes a resource instance
func (s *Storage) DeleteResource(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource, exists := s.resourceInstances[id]
	if !exists {
		return fmt.Errorf("resource with ID %d not found", id)
	}

	// Remove from app mapping
	appName := resource.ApplicationName
	if resourceIDs, exists := s.resourcesByApp[appName]; exists {
		for i, resourceID := range resourceIDs {
			if resourceID == id {
				s.resourcesByApp[appName] = append(resourceIDs[:i], resourceIDs[i+1:]...)
				break
			}
		}
		// If no more resources for this app, remove the app entry
		if len(s.resourcesByApp[appName]) == 0 {
			delete(s.resourcesByApp, appName)
		}
	}

	// Remove resource instance
	delete(s.resourceInstances, id)

	// Save to disk
	if err := s.saveToDisk(); err != nil {
		fmt.Printf("Warning: Failed to save data to disk: %v\n", err)
	}

	return nil
}