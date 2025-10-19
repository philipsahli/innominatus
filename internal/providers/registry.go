package providers

import (
	"fmt"
	"innominatus/pkg/sdk"
	"sync"
)

// Registry manages loaded providers and their provisioners
type Registry struct {
	mu           sync.RWMutex
	providers    map[string]*sdk.Provider   // name -> provider
	provisioners map[string]sdk.Provisioner // type -> provisioner
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers:    make(map[string]*sdk.Provider),
		provisioners: make(map[string]sdk.Provisioner),
	}
}

// RegisterProvider registers a provider in the registry
func (r *Registry) RegisterProvider(provider *sdk.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate provider name
	if _, exists := r.providers[provider.Metadata.Name]; exists {
		return fmt.Errorf("provider %s is already registered", provider.Metadata.Name)
	}

	r.providers[provider.Metadata.Name] = provider
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

// GetProvider returns a provider by name
func (r *Registry) GetProvider(name string) (*sdk.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// ListProviders returns all registered providers
func (r *Registry) ListProviders() []*sdk.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*sdk.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	return providers
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

// Count returns the number of registered providers and provisioners
func (r *Registry) Count() (providers int, provisioners int) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.providers), len(r.provisioners)
}

// Clear removes all providers and provisioners (useful for testing)
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers = make(map[string]*sdk.Provider)
	r.provisioners = make(map[string]sdk.Provisioner)
}
