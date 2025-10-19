package platform

import (
	"fmt"
	"innominatus/pkg/sdk"
	"sync"
)

// Registry manages loaded platforms and their provisioners
type Registry struct {
	mu           sync.RWMutex
	platforms    map[string]*sdk.Platform // name -> platform
	provisioners map[string]sdk.Provisioner // type -> provisioner
}

// NewRegistry creates a new platform registry
func NewRegistry() *Registry {
	return &Registry{
		platforms:    make(map[string]*sdk.Platform),
		provisioners: make(map[string]sdk.Provisioner),
	}
}

// RegisterPlatform registers a platform in the registry
func (r *Registry) RegisterPlatform(platform *sdk.Platform) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate platform name
	if _, exists := r.platforms[platform.Metadata.Name]; exists {
		return fmt.Errorf("platform %s is already registered", platform.Metadata.Name)
	}

	r.platforms[platform.Metadata.Name] = platform
	return nil
}

// RegisterProvisioner registers a provisioner in the registry
func (r *Registry) RegisterProvisioner(provisioner sdk.Provisioner) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	provType := provisioner.Type()

	// Check for duplicate provisioner type
	if existing, exists := r.provisioners[provType]; exists {
		return fmt.Errorf(
			"provisioner type %s is already registered by %s, cannot register %s",
			provType,
			existing.Name(),
			provisioner.Name(),
		)
	}

	r.provisioners[provType] = provisioner
	return nil
}

// GetProvisioner returns a provisioner by type
func (r *Registry) GetProvisioner(provisionerType string) (sdk.Provisioner, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provisioner, exists := r.provisioners[provisionerType]
	if !exists {
		return nil, fmt.Errorf("no provisioner registered for type %s", provisionerType)
	}

	return provisioner, nil
}

// GetPlatform returns a platform by name
func (r *Registry) GetPlatform(name string) (*sdk.Platform, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	platform, exists := r.platforms[name]
	if !exists {
		return nil, fmt.Errorf("platform %s not found", name)
	}

	return platform, nil
}

// ListPlatforms returns all registered platforms
func (r *Registry) ListPlatforms() []*sdk.Platform {
	r.mu.RLock()
	defer r.mu.RUnlock()

	platforms := make([]*sdk.Platform, 0, len(r.platforms))
	for _, platform := range r.platforms {
		platforms = append(platforms, platform)
	}

	return platforms
}

// ListProvisioners returns all registered provisioners
func (r *Registry) ListProvisioners() []sdk.Provisioner {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provisioners := make([]sdk.Provisioner, 0, len(r.provisioners))
	for _, provisioner := range r.provisioners {
		provisioners = append(provisioners, provisioner)
	}

	return provisioners
}

// GetProvisionerTypes returns all registered provisioner types
func (r *Registry) GetProvisionerTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.provisioners))
	for provType := range r.provisioners {
		types = append(types, provType)
	}

	return types
}

// HasProvisioner checks if a provisioner type is registered
func (r *Registry) HasProvisioner(provisionerType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.provisioners[provisionerType]
	return exists
}

// Count returns the number of registered platforms and provisioners
func (r *Registry) Count() (platforms int, provisioners int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.platforms), len(r.provisioners)
}

// Clear removes all platforms and provisioners (useful for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.platforms = make(map[string]*sdk.Platform)
	r.provisioners = make(map[string]sdk.Provisioner)
}
