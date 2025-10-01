package vault

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// K8sDeployer handles deployment of VSO manifests to Kubernetes
type K8sDeployer struct {
	kubeContext string
	dryRun      bool
}

// NewK8sDeployer creates a new Kubernetes deployer
func NewK8sDeployer(kubeContext string, dryRun bool) *K8sDeployer {
	return &K8sDeployer{
		kubeContext: kubeContext,
		dryRun:      dryRun,
	}
}

// DeployVSOManifests deploys all VSO manifests for an application
func (k *K8sDeployer) DeployVSOManifests(appName, appNamespace string, manifests map[string]string) error {
	fmt.Printf("🚀 Deploying VSO manifests for app: %s in namespace: %s\n", appName, appNamespace)

	if k.dryRun {
		fmt.Printf("   [DRY RUN] Would deploy %d manifests for app: %s\n", len(manifests), appName)
		return nil
	}

	// Create namespace first
	if err := k.createNamespace(appNamespace); err != nil {
		return fmt.Errorf("failed to create namespace %s: %w", appNamespace, err)
	}

	// Create service account for VSO
	if err := k.createServiceAccount(appNamespace, "vault-secrets-operator"); err != nil {
		return fmt.Errorf("failed to create service account: %w", err)
	}

	// Deploy manifests in order
	deployOrder := []string{"service-account", "vault-connection", "vault-auth"}

	// Deploy ordered manifests first
	for _, manifestType := range deployOrder {
		if manifest, exists := manifests[manifestType]; exists {
			if err := k.deployManifest(appNamespace, manifestType, manifest); err != nil {
				return fmt.Errorf("failed to deploy %s: %w", manifestType, err)
			}
		}
	}

	// Deploy remaining manifests (secrets)
	for manifestType, manifest := range manifests {
		if !contains(deployOrder, manifestType) {
			if err := k.deployManifest(appNamespace, manifestType, manifest); err != nil {
				return fmt.Errorf("failed to deploy %s: %w", manifestType, err)
			}
		}
	}

	fmt.Printf("✅ Successfully deployed all VSO manifests for app: %s\n", appName)
	return nil
}

// DeployManifest deploys a single manifest to Kubernetes
func (k *K8sDeployer) deployManifest(namespace, manifestType, yamlContent string) error {
	fmt.Printf("   📄 Deploying %s manifest to namespace: %s\n", manifestType, namespace)

	// Write manifest to temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("vso-%s-*.yaml", manifestType))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	_ = tmpFile.Close()

	// Apply manifest using kubectl
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "apply", "-f", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply manifest %s: %v\nOutput: %s", manifestType, err, string(output))
	}

	fmt.Printf("   ✅ %s deployed successfully\n", manifestType)
	return nil
}

// CreateNamespace creates a namespace if it doesn't exist
func (k *K8sDeployer) createNamespace(namespace string) error {
	fmt.Printf("📁 Ensuring namespace exists: %s\n", namespace)

	cmd := exec.Command("kubectl", "--context", k.kubeContext, "create", "namespace", namespace)
	output, err := cmd.CombinedOutput()

	// Ignore error if namespace already exists
	if err != nil && !strings.Contains(string(output), "already exists") {
		return fmt.Errorf("failed to create namespace %s: %v\nOutput: %s", namespace, err, string(output))
	}

	return nil
}

// createServiceAccount creates a service account if it doesn't exist
func (k *K8sDeployer) createServiceAccount(namespace, serviceAccountName string) error {
	fmt.Printf("👤 Ensuring ServiceAccount exists: %s/%s\n", namespace, serviceAccountName)

	serviceAccountYAML := fmt.Sprintf(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/component: vault-secrets-operator
    app.kubernetes.io/managed-by: idp-orchestrator`, serviceAccountName, namespace)

	// Write ServiceAccount manifest to temporary file
	tmpFile, err := os.CreateTemp("", "serviceaccount-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(serviceAccountYAML); err != nil {
		return fmt.Errorf("failed to write ServiceAccount manifest: %w", err)
	}
	_ = tmpFile.Close()

	// Apply ServiceAccount using kubectl
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "apply", "-f", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create ServiceAccount %s: %v\nOutput: %s", serviceAccountName, err, string(output))
	}

	fmt.Printf("✅ ServiceAccount %s created/updated\n", serviceAccountName)
	return nil
}

// WaitForVSOSync waits for VSO to sync secrets
func (k *K8sDeployer) WaitForVSOSync(appName, appNamespace string, secretNames []string, timeout time.Duration) error {
	fmt.Printf("⏳ Waiting for VSO to sync secrets for app: %s (timeout: %v)\n", appName, timeout)

	if k.dryRun {
		fmt.Printf("   [DRY RUN] Would wait for VSO sync of %d secrets\n", len(secretNames))
		return nil
	}

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for VSO to sync secrets")
			}

			allReady := true
			for _, secretName := range secretNames {
				ready, err := k.isSecretReady(appNamespace, secretName)
				if err != nil {
					fmt.Printf("   Warning: failed to check secret %s: %v\n", secretName, err)
					continue
				}
				if !ready {
					allReady = false
					fmt.Printf("   ⏳ Secret %s not yet synced...\n", secretName)
					break
				}
			}

			if allReady {
				fmt.Printf("✅ All secrets synced successfully for app: %s\n", appName)
				return nil
			}

		case <-time.After(timeout):
			return fmt.Errorf("timeout waiting for VSO to sync secrets")
		}
	}
}

// IsSecretReady checks if a Kubernetes secret exists and has data
func (k *K8sDeployer) isSecretReady(namespace, secretName string) (bool, error) {
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "get", "secret", secretName, "-n", namespace, "-o", "jsonpath={.data}")
	output, err := cmd.Output()
	if err != nil {
		// Secret doesn't exist yet
		return false, nil
	}

	// Check if secret has data
	return len(strings.TrimSpace(string(output))) > 2, nil // More than just "{}"
}

// GetVSOSecretStatus gets the status of VSO-managed secrets
func (k *K8sDeployer) GetVSOSecretStatus(appName, appNamespace string) (map[string]string, error) {
	fmt.Printf("📊 Getting VSO secret status for app: %s\n", appName)

	if k.dryRun {
		return map[string]string{
			"app-config":            "synced",
			"database-credentials":  "synced",
			"api-keys":             "synced",
		}, nil
	}

	status := make(map[string]string)

	// Get all VaultStaticSecret resources for this app
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "get", "vaultstaticsecret",
		"-n", appNamespace,
		"-l", fmt.Sprintf("app=%s", appName),
		"-o", "jsonpath={range .items[*]}{.metadata.name}{':'}{.status.lastGeneration}{':'}{.status.secretMAC}{'\\n'}{end}")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get VaultStaticSecret status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			secretName := parts[0]
			// Simple status determination - in production would check more fields
			if parts[1] != "" && parts[2] != "" {
				status[secretName] = "synced"
			} else {
				status[secretName] = "pending"
			}
		}
	}

	return status, nil
}

// CleanupVSOResources removes all VSO resources for an application
func (k *K8sDeployer) CleanupVSOResources(appName, appNamespace string) error {
	fmt.Printf("🗑️  Cleaning up VSO resources for app: %s\n", appName)

	if k.dryRun {
		fmt.Printf("   [DRY RUN] Would cleanup VSO resources for app: %s\n", appName)
		return nil
	}

	// Delete VaultStaticSecret resources
	cmd := exec.Command("kubectl", "--context", k.kubeContext, "delete", "vaultstaticsecret",
		"-n", appNamespace,
		"-l", fmt.Sprintf("app=%s", appName))
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to delete VaultStaticSecret resources: %v\nOutput: %s\n", err, string(output))
	}

	// Delete VaultAuth
	vaultAuthName := fmt.Sprintf("%s-vault-auth", appName)
	cmd = exec.Command("kubectl", "--context", k.kubeContext, "delete", "vaultauth", vaultAuthName, "-n", appNamespace)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to delete VaultAuth %s: %v\nOutput: %s\n", vaultAuthName, err, string(output))
	}

	// Delete VaultConnection
	vaultConnectionName := fmt.Sprintf("%s-vault-connection", appName)
	cmd = exec.Command("kubectl", "--context", k.kubeContext, "delete", "vaultconnection", vaultConnectionName, "-n", appNamespace)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to delete VaultConnection %s: %v\nOutput: %s\n", vaultConnectionName, err, string(output))
	}

	// Delete ServiceAccount
	serviceAccountName := fmt.Sprintf("%s-vault-sa", appName)
	cmd = exec.Command("kubectl", "--context", k.kubeContext, "delete", "serviceaccount", serviceAccountName, "-n", appNamespace)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to delete ServiceAccount %s: %v\nOutput: %s\n", serviceAccountName, err, string(output))
	}

	fmt.Printf("✅ VSO resources cleanup completed for app: %s\n", appName)
	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}