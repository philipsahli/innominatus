package providers

import (
	"context"
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/resources"
	"innominatus/pkg/sdk"
)

// GiteaAdapter adapts the existing GiteaProvisioner to the SDK Provisioner interface
type GiteaAdapter struct {
	provisioner *resources.GiteaProvisioner
	repo        *database.ResourceRepository
}

// NewGiteaAdapter creates a new Gitea adapter
func NewGiteaAdapter(repo *database.ResourceRepository) *GiteaAdapter {
	return &GiteaAdapter{
		provisioner: resources.NewGiteaProvisioner(repo),
		repo:        repo,
	}
}

func (a *GiteaAdapter) Name() string    { return "gitea-repo" }
func (a *GiteaAdapter) Type() string    { return "gitea-repo" }
func (a *GiteaAdapter) Version() string { return "1.0.0" }

func (a *GiteaAdapter) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
	// Convert SDK Resource to database.ResourceInstance
	dbResource := sdkResourceToDatabaseResource(resource)

	// Convert SDK Config to map[string]interface{}
	configMap := config.AsMap()

	// Call existing provisioner
	return a.provisioner.Provision(dbResource, configMap, "platform-adapter")
}

func (a *GiteaAdapter) Deprovision(ctx context.Context, resource *sdk.Resource) error {
	dbResource := sdkResourceToDatabaseResource(resource)
	return a.provisioner.Deprovision(dbResource)
}

func (a *GiteaAdapter) GetStatus(ctx context.Context, resource *sdk.Resource) (*sdk.ResourceStatus, error) {
	dbResource := sdkResourceToDatabaseResource(resource)

	statusMap, err := a.provisioner.GetStatus(dbResource)
	if err != nil {
		return nil, err
	}

	// Convert status map to SDK ResourceStatus
	status := &sdk.ResourceStatus{
		State: resource.State,
	}

	if state, ok := statusMap["state"].(string); ok {
		status.HealthStatus = state

		// Map internal states to SDK states
		switch state {
		case "active":
			status.State = sdk.ResourceStateActive
		case "not_found":
			status.State = sdk.ResourceStateTerminated
		case "error":
			status.State = sdk.ResourceStateFailed
		}
	}

	if errMsg, ok := statusMap["error"].(string); ok {
		status.Message = errMsg
	}

	return status, nil
}

func (a *GiteaAdapter) GetHints(ctx context.Context, resource *sdk.Resource) ([]sdk.Hint, error) {
	// Get resource from database to fetch hints
	dbResource, err := a.repo.GetResourceInstance(resource.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	// Convert database hints to SDK hints
	hints := make([]sdk.Hint, len(dbResource.Hints))
	for i, dbHint := range dbResource.Hints {
		hints[i] = sdk.Hint{
			Type:  dbHint.Type,
			Label: dbHint.Label,
			Value: dbHint.Value,
			Icon:  dbHint.Icon,
		}
	}

	return hints, nil
}

// KubernetesAdapter adapts the existing KubernetesProvisioner to the SDK Provisioner interface
type KubernetesAdapter struct {
	provisioner *resources.KubernetesProvisioner
	repo        *database.ResourceRepository
}

// NewKubernetesAdapter creates a new Kubernetes adapter
func NewKubernetesAdapter(repo *database.ResourceRepository) *KubernetesAdapter {
	return &KubernetesAdapter{
		provisioner: resources.NewKubernetesProvisioner(repo),
		repo:        repo,
	}
}

func (a *KubernetesAdapter) Name() string    { return "kubernetes" }
func (a *KubernetesAdapter) Type() string    { return "kubernetes" }
func (a *KubernetesAdapter) Version() string { return "1.0.0" }

func (a *KubernetesAdapter) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
	dbResource := sdkResourceToDatabaseResource(resource)
	configMap := config.AsMap()
	return a.provisioner.Provision(dbResource, configMap, "platform-adapter")
}

func (a *KubernetesAdapter) Deprovision(ctx context.Context, resource *sdk.Resource) error {
	dbResource := sdkResourceToDatabaseResource(resource)
	return a.provisioner.Deprovision(dbResource)
}

func (a *KubernetesAdapter) GetStatus(ctx context.Context, resource *sdk.Resource) (*sdk.ResourceStatus, error) {
	dbResource := sdkResourceToDatabaseResource(resource)

	statusMap, err := a.provisioner.GetStatus(dbResource)
	if err != nil {
		return nil, err
	}

	status := &sdk.ResourceStatus{
		State: resource.State,
	}

	if state, ok := statusMap["state"].(string); ok {
		status.HealthStatus = state
		switch state {
		case "active":
			status.State = sdk.ResourceStateActive
		case "not_found":
			status.State = sdk.ResourceStateTerminated
		case "error":
			status.State = sdk.ResourceStateFailed
		}
	}

	return status, nil
}

func (a *KubernetesAdapter) GetHints(ctx context.Context, resource *sdk.Resource) ([]sdk.Hint, error) {
	dbResource, err := a.repo.GetResourceInstance(resource.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	hints := make([]sdk.Hint, len(dbResource.Hints))
	for i, dbHint := range dbResource.Hints {
		hints[i] = sdk.Hint{
			Type:  dbHint.Type,
			Label: dbHint.Label,
			Value: dbHint.Value,
			Icon:  dbHint.Icon,
		}
	}

	return hints, nil
}

// ArgoCDAdapter adapts the existing ArgoCDProvisioner to the SDK Provisioner interface
type ArgoCDAdapter struct {
	provisioner *resources.ArgoCDProvisioner
	repo        *database.ResourceRepository
}

// NewArgoCDAdapter creates a new ArgoCD adapter
func NewArgoCDAdapter(repo *database.ResourceRepository) *ArgoCDAdapter {
	return &ArgoCDAdapter{
		provisioner: resources.NewArgoCDProvisioner(repo),
		repo:        repo,
	}
}

func (a *ArgoCDAdapter) Name() string    { return "argocd-app" }
func (a *ArgoCDAdapter) Type() string    { return "argocd-app" }
func (a *ArgoCDAdapter) Version() string { return "1.0.0" }

func (a *ArgoCDAdapter) Provision(ctx context.Context, resource *sdk.Resource, config sdk.Config) error {
	dbResource := sdkResourceToDatabaseResource(resource)
	configMap := config.AsMap()
	return a.provisioner.Provision(dbResource, configMap, "platform-adapter")
}

func (a *ArgoCDAdapter) Deprovision(ctx context.Context, resource *sdk.Resource) error {
	dbResource := sdkResourceToDatabaseResource(resource)
	return a.provisioner.Deprovision(dbResource)
}

func (a *ArgoCDAdapter) GetStatus(ctx context.Context, resource *sdk.Resource) (*sdk.ResourceStatus, error) {
	dbResource := sdkResourceToDatabaseResource(resource)

	statusMap, err := a.provisioner.GetStatus(dbResource)
	if err != nil {
		return nil, err
	}

	status := &sdk.ResourceStatus{
		State: resource.State,
	}

	if state, ok := statusMap["state"].(string); ok {
		status.HealthStatus = state
		switch state {
		case "active":
			status.State = sdk.ResourceStateActive
		case "not_found":
			status.State = sdk.ResourceStateTerminated
		case "error":
			status.State = sdk.ResourceStateFailed
		}
	}

	return status, nil
}

func (a *ArgoCDAdapter) GetHints(ctx context.Context, resource *sdk.Resource) ([]sdk.Hint, error) {
	dbResource, err := a.repo.GetResourceInstance(resource.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}

	hints := make([]sdk.Hint, len(dbResource.Hints))
	for i, dbHint := range dbResource.Hints {
		hints[i] = sdk.Hint{
			Type:  dbHint.Type,
			Label: dbHint.Label,
			Value: dbHint.Value,
			Icon:  dbHint.Icon,
		}
	}

	return hints, nil
}

// Helper function to convert SDK Resource to database ResourceInstance
func sdkResourceToDatabaseResource(resource *sdk.Resource) *database.ResourceInstance {
	// Convert string fields to pointers
	var providerID *string
	if resource.ProviderID != "" {
		providerID = &resource.ProviderID
	}

	var errorMessage *string
	if resource.ErrorMessage != "" {
		errorMessage = &resource.ErrorMessage
	}

	return &database.ResourceInstance{
		ID:              resource.ID,
		ApplicationName: resource.ApplicationName,
		ResourceName:    resource.ResourceName,
		ResourceType:    resource.ResourceType,
		State:           database.ResourceLifecycleState(resource.State),
		HealthStatus:    resource.HealthStatus,
		ProviderID:      providerID,
		CreatedAt:       resource.CreatedAt,
		UpdatedAt:       resource.UpdatedAt,
		ErrorMessage:    errorMessage,
	}
}
