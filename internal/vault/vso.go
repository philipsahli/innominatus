package vault
// #nosec G204 - Demo/vault components execute commands with controlled parameters

import (
	"fmt"
	"gopkg.in/yaml.v3"
)

// VSOManager handles Vault Secrets Operator CRD generation
type VSOManager struct {
	vaultAddress string
	namespace    string
}

// NewVSOManager creates a new VSO manager
func NewVSOManager(vaultAddress, namespace string) *VSOManager {
	return &VSOManager{
		vaultAddress: vaultAddress,
		namespace:    namespace,
	}
}

// VaultConnection represents a VaultConnection CRD
type VaultConnection struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		Address string            `yaml:"address"`
		Headers map[string]string `yaml:"headers,omitempty"`
	} `yaml:"spec"`
}

// VaultAuth represents a VaultAuth CRD
type VaultAuth struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		VaultConnectionRef string `yaml:"vaultConnectionRef"`
		Method             string `yaml:"method"`
		Mount              string `yaml:"mount"`
		Kubernetes         struct {
			Role           string   `yaml:"role"`
			ServiceAccount string   `yaml:"serviceAccount"`
			Audiences      []string `yaml:"audiences"`
		} `yaml:"kubernetes"`
	} `yaml:"spec"`
}

// VaultStaticSecret represents a VaultStaticSecret CRD
type VaultStaticSecret struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		VaultAuthRef       string `yaml:"vaultAuthRef"`
		Mount              string `yaml:"mount"`
		Type               string `yaml:"type"`
		Path               string `yaml:"path"`
		RefreshAfter       string `yaml:"refreshAfter"`
		Destination        struct {
			Name   string            `yaml:"name"`
			Create bool              `yaml:"create"`
			Labels map[string]string `yaml:"labels,omitempty"`
		} `yaml:"destination"`
	} `yaml:"spec"`
}

// GenerateVaultConnection creates a VaultConnection manifest for an application
func (v *VSOManager) GenerateVaultConnection(appName, appNamespace string) (string, error) {
	fmt.Printf("🔗 Generating VaultConnection for app: %s in namespace: %s\n", appName, appNamespace)

	conn := VaultConnection{
		APIVersion: "secrets.hashicorp.com/v1beta1",
		Kind:       "VaultConnection",
	}
	conn.Metadata.Name = fmt.Sprintf("%s-vault-connection", appName)
	conn.Metadata.Namespace = appNamespace
	conn.Spec.Address = v.vaultAddress
	conn.Spec.Headers = map[string]string{}

	yamlData, err := yaml.Marshal(conn)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VaultConnection: %w", err)
	}

	return string(yamlData), nil
}

// GenerateVaultAuth creates a VaultAuth manifest for an application
func (v *VSOManager) GenerateVaultAuth(appName, appNamespace, serviceAccount string) (string, error) {
	fmt.Printf("🔑 Generating VaultAuth for app: %s with SA: %s\n", appName, serviceAccount)

	auth := VaultAuth{
		APIVersion: "secrets.hashicorp.com/v1beta1",
		Kind:       "VaultAuth",
	}
	auth.Metadata.Name = fmt.Sprintf("%s-vault-auth", appName)
	auth.Metadata.Namespace = appNamespace
	auth.Spec.VaultConnectionRef = fmt.Sprintf("%s-vault-connection", appName)
	auth.Spec.Method = "kubernetes"
	auth.Spec.Mount = fmt.Sprintf("kubernetes-%s", appName)
	auth.Spec.Kubernetes.Role = fmt.Sprintf("%s-role", appName)
	auth.Spec.Kubernetes.ServiceAccount = serviceAccount
	auth.Spec.Kubernetes.Audiences = []string{"vault"}

	yamlData, err := yaml.Marshal(auth)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VaultAuth: %w", err)
	}

	return string(yamlData), nil
}

// GenerateVaultStaticSecret creates a VaultStaticSecret manifest for a specific secret
func (v *VSOManager) GenerateVaultStaticSecret(appName, appNamespace, secretName, vaultPath string) (string, error) {
	fmt.Printf("🔒 Generating VaultStaticSecret for app: %s, secret: %s\n", appName, secretName)

	secret := VaultStaticSecret{
		APIVersion: "secrets.hashicorp.com/v1beta1",
		Kind:       "VaultStaticSecret",
	}
	secret.Metadata.Name = fmt.Sprintf("%s-%s", appName, secretName)
	secret.Metadata.Namespace = appNamespace
	secret.Spec.VaultAuthRef = fmt.Sprintf("%s-vault-auth", appName)
	secret.Spec.Mount = "secret"
	secret.Spec.Type = "kv-v2"
	secret.Spec.Path = vaultPath
	secret.Spec.RefreshAfter = "30s"
	secret.Spec.Destination.Name = secretName
	secret.Spec.Destination.Create = true
	secret.Spec.Destination.Labels = map[string]string{
		"app":                       appName,
		"managed-by":                "vault-secrets-operator",
		"idp-orchestrator/app-name": appName,
	}

	yamlData, err := yaml.Marshal(secret)
	if err != nil {
		return "", fmt.Errorf("failed to marshal VaultStaticSecret: %w", err)
	}

	return string(yamlData), nil
}

// GenerateServiceAccount creates a ServiceAccount manifest for Vault access
func (v *VSOManager) GenerateServiceAccount(appName, appNamespace string) (string, error) {
	fmt.Printf("👤 Generating ServiceAccount for app: %s\n", appName)

	serviceAccount := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ServiceAccount",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-vault-sa", appName),
			"namespace": appNamespace,
			"labels": map[string]string{
				"app":                       appName,
				"component":                 "vault-auth",
				"idp-orchestrator/app-name": appName,
			},
		},
	}

	yamlData, err := yaml.Marshal(serviceAccount)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ServiceAccount: %w", err)
	}

	return string(yamlData), nil
}

// GenerateAllManifests creates all required manifests for an application's Vault integration
func (v *VSOManager) GenerateAllManifests(appName, appNamespace string, secrets []string) (map[string]string, error) {
	fmt.Printf("📝 Generating all Vault manifests for app: %s\n", appName)

	manifests := make(map[string]string)
	serviceAccountName := fmt.Sprintf("%s-vault-sa", appName)

	// Generate ServiceAccount
	serviceAccount, err := v.GenerateServiceAccount(appName, appNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ServiceAccount: %w", err)
	}
	manifests["service-account"] = serviceAccount

	// Generate VaultConnection
	vaultConnection, err := v.GenerateVaultConnection(appName, appNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to generate VaultConnection: %w", err)
	}
	manifests["vault-connection"] = vaultConnection

	// Generate VaultAuth
	vaultAuth, err := v.GenerateVaultAuth(appName, appNamespace, serviceAccountName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate VaultAuth: %w", err)
	}
	manifests["vault-auth"] = vaultAuth

	// Generate VaultStaticSecret for each secret
	for _, secretName := range secrets {
		vaultPath := fmt.Sprintf("applications/%s/%s", appName, secretName)
		staticSecret, err := v.GenerateVaultStaticSecret(appName, appNamespace, secretName, vaultPath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate VaultStaticSecret for %s: %w", secretName, err)
		}
		manifests[fmt.Sprintf("secret-%s", secretName)] = staticSecret
	}

	fmt.Printf("✅ Generated %d manifests for app: %s\n", len(manifests), appName)
	return manifests, nil
}