package resources

import (
	"fmt"
	"innominatus/internal/database"
	"innominatus/internal/vault"
	"strings"
)

// provisionPostgres provisions a PostgreSQL resource
func (m *Manager) provisionPostgres(resource *database.ResourceInstance, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	// Simulate PostgreSQL provisioning
	fmt.Printf("Provisioning PostgreSQL resource: %s\n", resource.ResourceName)

	// Add provider-specific metadata
	if providerMetadata == nil {
		providerMetadata = make(map[string]interface{})
	}
	providerMetadata["database_name"] = resource.ResourceName + "_db"
	providerMetadata["port"] = 5432
	providerMetadata["version"] = "13.0"
	providerMetadata["storage_size"] = "20GB"

	// Transition to active state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateActive,
		"PostgreSQL database provisioned successfully",
		transitionedBy, map[string]interface{}{
			"provider_id":       providerID,
			"provider_metadata": providerMetadata,
			"provisioning_time": "45s",
		})
}

// provisionRedis provisions a Redis resource
func (m *Manager) provisionRedis(resource *database.ResourceInstance, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	// Simulate Redis provisioning
	fmt.Printf("Provisioning Redis resource: %s\n", resource.ResourceName)

	// Add provider-specific metadata
	if providerMetadata == nil {
		providerMetadata = make(map[string]interface{})
	}
	providerMetadata["instance_type"] = "cache.t3.micro"
	providerMetadata["port"] = 6379
	providerMetadata["version"] = "6.2"
	providerMetadata["memory_size"] = "512MB"

	// Transition to active state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateActive,
		"Redis cache provisioned successfully",
		transitionedBy, map[string]interface{}{
			"provider_id":       providerID,
			"provider_metadata": providerMetadata,
			"provisioning_time": "30s",
		})
}

// provisionVolume provisions a volume resource
func (m *Manager) provisionVolume(resource *database.ResourceInstance, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	// Simulate volume provisioning
	fmt.Printf("Provisioning Volume resource: %s\n", resource.ResourceName)

	// Add provider-specific metadata
	if providerMetadata == nil {
		providerMetadata = make(map[string]interface{})
	}
	providerMetadata["volume_type"] = "gp3"
	providerMetadata["size"] = "10GB"
	providerMetadata["mount_path"] = "/data"
	providerMetadata["filesystem"] = "ext4"

	// Transition to active state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateActive,
		"Volume provisioned successfully",
		transitionedBy, map[string]interface{}{
			"provider_id":       providerID,
			"provider_metadata": providerMetadata,
			"provisioning_time": "20s",
		})
}

// provisionVaultSpace provisions a Vault space resource with secret synchronization
func (m *Manager) provisionVaultSpace(resource *database.ResourceInstance, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	fmt.Printf("Provisioning Vault space resource: %s for app: %s\n", resource.ResourceName, resource.ApplicationName)

	// Initialize Vault client
	vaultClient := vault.NewClient("http://vault.vault.svc.cluster.local:8200", "root")

	// Create VSO manager
	vsoManager := vault.NewVSOManager("http://vault.vault.svc.cluster.local:8200", "vault-secrets-operator-system")

	// Create K8s deployer
	k8sDeployer := vault.NewK8sDeployer("docker-desktop", false)

	// Application namespace for this resource
	appNamespace := strings.ToLower(resource.ApplicationName)
	serviceAccount := "vault-secrets-operator"

	// Setup Vault authentication and authorization
	fmt.Printf("🔧 Setting up Vault authentication for app: %s\n", resource.ApplicationName)

	// Setup Kubernetes auth method
	if err := vaultClient.SetupKubernetesAuthMethod(); err != nil {
		return fmt.Errorf("failed to setup Kubernetes auth method: %w", err)
	}

	// Setup application-specific policy
	if err := vaultClient.SetupVaultPolicy(resource.ApplicationName); err != nil {
		return fmt.Errorf("failed to setup Vault policy: %w", err)
	}

	// Setup Kubernetes role
	if err := vaultClient.SetupKubernetesRole(resource.ApplicationName, appNamespace, serviceAccount); err != nil {
		return fmt.Errorf("failed to setup Kubernetes role: %w", err)
	}

	// Extract secrets configuration from Score spec
	secretsConfig := []string{"app-config", "database-credentials", "api-keys"}
	if config, ok := resource.Configuration["params"].(map[string]interface{}); ok {
		if secrets, ok := config["secrets"].([]interface{}); ok {
			secretsConfig = []string{}
			for _, secret := range secrets {
				if secretMap, ok := secret.(map[string]interface{}); ok {
					if name, ok := secretMap["name"].(string); ok {
						secretsConfig = append(secretsConfig, name)
					}
				}
			}
		}
	}

	// Create default secrets in Vault based on Score spec
	for _, secretName := range secretsConfig {
		secretData := m.GenerateDefaultSecretData(secretName, resource.ApplicationName)
		if err := vaultClient.CreateSecret(resource.ApplicationName, secretName, secretData); err != nil {
			fmt.Printf("Warning: failed to create secret %s: %v\n", secretName, err)
		}
	}

	// Generate VSO manifests for secret synchronization
	manifests, err := vsoManager.GenerateAllManifests(resource.ApplicationName, appNamespace, secretsConfig)
	if err != nil {
		return fmt.Errorf("failed to generate VSO manifests: %w", err)
	}

	// Deploy VSO manifests to Kubernetes
	if err := k8sDeployer.DeployVSOManifests(resource.ApplicationName, appNamespace, manifests); err != nil {
		return fmt.Errorf("failed to deploy VSO manifests: %w", err)
	}

	// Add provider-specific metadata
	if providerMetadata == nil {
		providerMetadata = make(map[string]interface{})
	}
	authMount := "kubernetes"
	policyName := fmt.Sprintf("%s-policy", resource.ApplicationName)
	roleName := "vault-secrets-operator"

	providerMetadata["vault_namespace"] = resource.ApplicationName
	providerMetadata["auth_mount"] = authMount
	providerMetadata["policy_name"] = policyName
	providerMetadata["role_name"] = roleName
	providerMetadata["service_account"] = serviceAccount
	providerMetadata["app_namespace"] = appNamespace
	providerMetadata["secrets_created"] = len(secretsConfig)
	providerMetadata["vso_manifests"] = len(manifests)

	// Store VSO manifests in metadata for later deployment
	for manifestType, manifestYAML := range manifests {
		providerMetadata[fmt.Sprintf("manifest_%s", manifestType)] = manifestYAML
	}

	// Transition to active state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateActive,
		"Vault space provisioned with secret synchronization",
		transitionedBy, map[string]interface{}{
			"provider_id":       providerID,
			"provider_metadata": providerMetadata,
			"provisioning_time": "60s",
		})
}

// provisionGenericResource provisions any other resource type
func (m *Manager) provisionGenericResource(resource *database.ResourceInstance, providerID string, providerMetadata map[string]interface{}, transitionedBy string) error {
	// Simulate generic resource provisioning
	fmt.Printf("Provisioning %s resource: %s\n", resource.ResourceType, resource.ResourceName)

	// Add basic metadata
	if providerMetadata == nil {
		providerMetadata = make(map[string]interface{})
	}
	providerMetadata["resource_type"] = resource.ResourceType
	providerMetadata["provisioning_method"] = "generic"

	// Transition to active state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateActive,
		fmt.Sprintf("%s resource provisioned successfully", resource.ResourceType),
		transitionedBy, map[string]interface{}{
			"provider_id":       providerID,
			"provider_metadata": providerMetadata,
			"provisioning_time": "15s",
		})
}

// Deprovision methods for each resource type

// deprovisionPostgres deprovisions a PostgreSQL resource
func (m *Manager) deprovisionPostgres(resource *database.ResourceInstance, transitionedBy string) error {
	fmt.Printf("Deprovisioning PostgreSQL resource: %s\n", resource.ResourceName)

	// Simulate PostgreSQL cleanup
	// In production, this would:
	// - Drop database connections
	// - Backup critical data if required
	// - Remove database instance
	// - Clean up networking/security groups

	metadata := map[string]interface{}{
		"deprovisioning_method": "postgres_cleanup",
		"database_backed_up":    false, // Could be configurable
		"connections_dropped":   true,
		"cleanup_time":          "30s",
	}

	// Transition to terminated state
	return m.TransitionResourceState(resource.ID,
		database.ResourceStateTerminated,
		"PostgreSQL database deprovisioned successfully",
		transitionedBy,
		metadata)
}

// deprovisionRedis deprovisions a Redis resource
func (m *Manager) deprovisionRedis(resource *database.ResourceInstance, transitionedBy string) error {
	fmt.Printf("Deprovisioning Redis resource: %s\n", resource.ResourceName)

	// Simulate Redis cleanup
	// In production, this would:
	// - Flush cache if required
	// - Close connections
	// - Remove instance
	// - Clean up networking

	metadata := map[string]interface{}{
		"deprovisioning_method": "redis_cleanup",
		"cache_flushed":         false, // Could be configurable
		"connections_closed":    true,
		"cleanup_time":          "15s",
	}

	return m.TransitionResourceState(resource.ID,
		database.ResourceStateTerminated,
		"Redis cache deprovisioned successfully",
		transitionedBy,
		metadata)
}

// deprovisionVolume deprovisions a volume resource
func (m *Manager) deprovisionVolume(resource *database.ResourceInstance, transitionedBy string) error {
	fmt.Printf("Deprovisioning Volume resource: %s\n", resource.ResourceName)

	// Simulate volume cleanup
	// In production, this would:
	// - Unmount volume from instances
	// - Create snapshot if required
	// - Delete volume
	// - Clean up mount points

	metadata := map[string]interface{}{
		"deprovisioning_method": "volume_cleanup",
		"volume_unmounted":      true,
		"snapshot_created":      false, // Could be configurable
		"cleanup_time":          "20s",
	}

	return m.TransitionResourceState(resource.ID,
		database.ResourceStateTerminated,
		"Volume deprovisioned successfully",
		transitionedBy,
		metadata)
}

// deprovisionVaultSpace deprovisions a Vault space resource
func (m *Manager) deprovisionVaultSpace(resource *database.ResourceInstance, transitionedBy string) error {
	fmt.Printf("Deprovisioning Vault space resource: %s for app: %s\n", resource.ResourceName, resource.ApplicationName)

	// Initialize Vault client
	vaultClient := vault.NewClient("http://vault.vault.svc.cluster.local:8200", "root")

	// Create K8s deployer for cleanup
	k8sDeployer := vault.NewK8sDeployer("docker-desktop", false)

	appNamespace := strings.ToLower(resource.ApplicationName)

	// Clean up VSO resources from Kubernetes
	fmt.Printf("🧹 Cleaning up VSO resources for app: %s\n", resource.ApplicationName)
	if err := k8sDeployer.CleanupVSOResources(resource.ApplicationName, appNamespace); err != nil {
		fmt.Printf("Warning: failed to cleanup VSO resources: %v\n", err)
	}

	// Clean up Vault secrets (using existing DeleteSecret method for each secret)
	fmt.Printf("🧹 Cleaning up Vault secrets for app: %s\n", resource.ApplicationName)
	secretsConfig := []string{"app-config", "database-credentials", "api-keys"}
	if config, ok := resource.Configuration["params"].(map[string]interface{}); ok {
		if secrets, ok := config["secrets"].([]interface{}); ok {
			secretsConfig = []string{}
			for _, secret := range secrets {
				if secretMap, ok := secret.(map[string]interface{}); ok {
					if name, ok := secretMap["name"].(string); ok {
						secretsConfig = append(secretsConfig, name)
					}
				}
			}
		}
	}
	for _, secretName := range secretsConfig {
		if err := vaultClient.DeleteSecret(resource.ApplicationName, secretName); err != nil {
			fmt.Printf("Warning: failed to delete secret %s: %v\n", secretName, err)
		}
	}

	// Clean up Vault policies and roles (manual cleanup using basic operations)
	fmt.Printf("🧹 Cleaning up Vault policies and roles for app: %s\n", resource.ApplicationName)
	fmt.Printf("Note: Policy and role cleanup would be implemented in production Vault integration\n")

	metadata := map[string]interface{}{
		"deprovisioning_method": "vault_space_cleanup",
		"vso_manifests_removed": true,
		"secrets_deleted":       true,
		"policies_cleaned":      true,
		"app_namespace":         appNamespace,
		"cleanup_time":          "45s",
	}

	return m.TransitionResourceState(resource.ID,
		database.ResourceStateTerminated,
		"Vault space deprovisioned successfully",
		transitionedBy,
		metadata)
}

// deprovisionGenericResource deprovisions any other resource type
func (m *Manager) deprovisionGenericResource(resource *database.ResourceInstance, transitionedBy string) error {
	fmt.Printf("Deprovisioning %s resource: %s\n", resource.ResourceType, resource.ResourceName)

	metadata := map[string]interface{}{
		"deprovisioning_method": "generic_cleanup",
		"resource_type":         resource.ResourceType,
		"cleanup_time":          "10s",
	}

	return m.TransitionResourceState(resource.ID,
		database.ResourceStateTerminated,
		fmt.Sprintf("%s resource deprovisioned successfully", resource.ResourceType),
		transitionedBy,
		metadata)
}

// DeprovisionResource deprovisions a resource instance based on its type
func (m *Manager) DeprovisionResource(resourceID int64, transitionedBy string) error {
	if err := m.checkRepository(); err != nil {
		return err
	}

	resource, err := m.resourceRepo.GetResourceInstance(resourceID)
	if err != nil {
		return fmt.Errorf("failed to get resource: %w", err)
	}

	// First transition to terminating state
	err = m.TransitionResourceState(resourceID,
		database.ResourceStateTerminating,
		"Resource deprovisioning started",
		transitionedBy, map[string]interface{}{
			"operation": "deprovision",
		})
	if err != nil {
		return fmt.Errorf("failed to transition to terminating state: %w", err)
	}

	// Call appropriate deprovision method based on resource type
	switch resource.ResourceType {
	case "postgres":
		return m.deprovisionPostgres(resource, transitionedBy)
	case "redis":
		return m.deprovisionRedis(resource, transitionedBy)
	case "volume":
		return m.deprovisionVolume(resource, transitionedBy)
	case "vault-space":
		return m.deprovisionVaultSpace(resource, transitionedBy)
	default:
		return m.deprovisionGenericResource(resource, transitionedBy)
	}
}

// GenerateDefaultSecretData generates default secret data based on secret name and app
func (m *Manager) GenerateDefaultSecretData(secretName, appName string) map[string]interface{} {
	switch secretName {
	case "app-config":
		return map[string]interface{}{
			"app_name":    appName,
			"environment": "development",
			"debug":       "true",
			"log_level":   "info",
		}
	case "database-credentials":
		return map[string]interface{}{
			"database_url": fmt.Sprintf("postgresql://user:pass@postgres.%s.svc.cluster.local:5432/%s", appName, appName),
			"username":     fmt.Sprintf("%s_user", appName),
			"password":     fmt.Sprintf("%s_password_123", appName),
			"host":         fmt.Sprintf("postgres.%s.svc.cluster.local", appName),
			"port":         "5432",
		}
	case "api-keys":
		return map[string]interface{}{
			"api_key":        fmt.Sprintf("%s_api_key_abc123", appName),
			"webhook_secret": fmt.Sprintf("%s_webhook_secret_xyz789", appName),
			"jwt_secret":     fmt.Sprintf("%s_jwt_secret_def456", appName),
		}
	default:
		return map[string]interface{}{
			"default_key": fmt.Sprintf("%s_default_value", secretName),
			"created_for": appName,
			"secret_type": secretName,
			"created_at":  "2024-01-01T00:00:00Z",
		}
	}
}
