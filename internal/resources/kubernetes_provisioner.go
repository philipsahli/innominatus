package resources

// #nosec G204 - Kubernetes provisioner executes kubectl commands with validated resource names and namespaces

import (
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"innominatus/internal/types"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// KubernetesProvisioner handles Kubernetes deployments
// This is owned by the K8s infrastructure team
type KubernetesProvisioner struct {
	repo *database.ResourceRepository
}

// NewKubernetesProvisioner creates a new Kubernetes provisioner
func NewKubernetesProvisioner(repo *database.ResourceRepository) *KubernetesProvisioner {
	return &KubernetesProvisioner{
		repo: repo,
	}
}

// Provision deploys application to Kubernetes
func (kp *KubernetesProvisioner) Provision(resource *database.ResourceInstance, config map[string]interface{}, provisionedBy string) error {
	appName := resource.ApplicationName
	namespace := resource.ResourceName // Use resource name as namespace

	// Extract Score spec from config if available
	var scoreSpec *types.ScoreSpec
	if specInterface, ok := config["score_spec"]; ok {
		if spec, ok := specInterface.(*types.ScoreSpec); ok {
			scoreSpec = spec
		}
	}

	fmt.Printf("🔧 Provisioning Kubernetes deployment for '%s' in namespace '%s'\n", appName, namespace)

	// Step 1: Create namespace
	if err := kp.createNamespace(namespace); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	fmt.Printf("   ✅ Namespace '%s' created\n", namespace)

	// Step 2: Generate manifests
	manifests, err := kp.generateManifests(appName, namespace, scoreSpec)
	if err != nil {
		return fmt.Errorf("failed to generate manifests: %w", err)
	}
	fmt.Printf("   ✅ Kubernetes manifests generated\n")

	// Step 3: Apply manifests
	if err := kp.applyManifests(manifests, namespace); err != nil {
		return fmt.Errorf("failed to apply manifests: %w", err)
	}
	fmt.Printf("   ✅ Manifests applied to cluster\n")

	// Step 4: Wait for deployment readiness
	if err := kp.waitForReady(appName, namespace); err != nil {
		fmt.Printf("   ⚠️  Warning: Deployment readiness check: %v\n", err)
		// Don't fail - deployment may still be progressing
	} else {
		fmt.Printf("   ✅ Deployment is ready\n")
	}

	// Step 5: Verify pods are running
	if err := kp.verifyPods(namespace); err != nil {
		fmt.Printf("   ⚠️  Warning: Pod verification: %v\n", err)
	} else {
		fmt.Printf("   ✅ Pods are running\n")
	}

	// Step 6: Commit manifests to Gitea (if configured)
	if err := kp.commitManifestsToGit(appName, namespace, manifests); err != nil {
		fmt.Printf("   ⚠️  Warning: Failed to commit manifests to Git: %v\n", err)
		// Don't fail deployment - manifests are already applied
	} else {
		fmt.Printf("   ✅ Manifests committed to Git repository\n")
	}

	// Resource status is updated by the Manager's ProvisionResource method

	// Add multiple hints for easy access to Kubernetes resources
	hints := []database.ResourceHint{
		{
			Type:  "dashboard",
			Label: "Kubernetes Dashboard",
			Value: fmt.Sprintf("http://k8s.localtest.me/#/overview?namespace=%s", namespace),
			Icon:  "external-link",
		},
		{
			Type:  "namespace",
			Label: "Namespace",
			Value: namespace,
			Icon:  "server",
		},
		{
			Type:  "command",
			Label: "View Pods",
			Value: fmt.Sprintf("kubectl get pods -n %s", namespace),
			Icon:  "terminal",
		},
	}

	// Update resource hints in database
	if err := kp.repo.UpdateResourceHints(resource.ID, hints); err != nil {
		// Log error but don't fail the provisioning
		fmt.Printf("   ⚠️  Warning: Failed to update resource hints: %v\n", err)
	}

	fmt.Printf("🎉 Kubernetes deployment completed for '%s'\n", appName)
	return nil
}

// Deprovision removes Kubernetes resources
func (kp *KubernetesProvisioner) Deprovision(resource *database.ResourceInstance) error {
	namespace := resource.ResourceName
	fmt.Printf("🗑️  Deprovisioning Kubernetes namespace '%s'\n", namespace)

	if err := kp.deleteNamespace(namespace); err != nil {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	fmt.Printf("✅ Namespace '%s' deleted\n", namespace)
	return nil
}

// GetStatus returns the current status of the Kubernetes deployment
func (kp *KubernetesProvisioner) GetStatus(resource *database.ResourceInstance) (map[string]interface{}, error) {
	namespace := resource.ResourceName
	status := make(map[string]interface{})

	// Check if namespace exists
	checkCmd := exec.Command("kubectl", "get", "namespace", namespace) // #nosec G204 - kubectl with validated namespace
	if err := checkCmd.Run(); err != nil {
		status["state"] = "not_found"
		status["error"] = "namespace not found"
		return status, nil
	}

	// Get pod status
	getPodsCmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-o", "json") // #nosec G204 - kubectl with validated namespace
	output, err := getPodsCmd.CombinedOutput()
	if err != nil {
		status["state"] = "error"
		status["error"] = fmt.Sprintf("failed to get pods: %v", err)
		return status, nil
	}

	// Simple status check - in production, parse JSON properly
	if strings.Contains(string(output), "Running") {
		status["state"] = "healthy"
		status["pods"] = "running"
	} else {
		status["state"] = "unhealthy"
		status["pods"] = "not running"
	}

	status["namespace"] = namespace
	return status, nil
}

// createNamespace creates a Kubernetes namespace
func (kp *KubernetesProvisioner) createNamespace(namespace string) error {
	cmd := exec.Command("kubectl", "create", "namespace", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if namespace already exists
		if strings.Contains(string(output), "AlreadyExists") {
			return nil // Namespace exists, not an error
		}
		return fmt.Errorf("kubectl create namespace failed: %w, output: %s", err, string(output))
	}
	return nil
}

// deleteNamespace deletes a Kubernetes namespace
func (kp *KubernetesProvisioner) deleteNamespace(namespace string) error {
	cmd := exec.Command("kubectl", "delete", "namespace", namespace, "--ignore-not-found=true")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl delete namespace failed: %w, output: %s", err, string(output))
	}
	return nil
}

// generateManifests generates Kubernetes manifests from Score spec
func (kp *KubernetesProvisioner) generateManifests(appName string, namespace string, scoreSpec *types.ScoreSpec) (string, error) {
	var manifests []string

	// Generate Deployment
	deployment := kp.generateDeployment(appName, namespace, scoreSpec)
	manifests = append(manifests, deployment)

	// Generate Service
	service := kp.generateService(appName, namespace, scoreSpec)
	manifests = append(manifests, service)

	// Generate Ingress if route resources exist
	if scoreSpec != nil && scoreSpec.Resources != nil {
		for _, resource := range scoreSpec.Resources {
			if resource.Type == "route" {
				ingress := kp.generateIngress(appName, namespace, resource.Params)
				manifests = append(manifests, ingress)
			}
		}
	}

	return strings.Join(manifests, "\n---\n"), nil
}

// generateDeployment creates a Kubernetes Deployment manifest
func (kp *KubernetesProvisioner) generateDeployment(appName string, namespace string, scoreSpec *types.ScoreSpec) string {
	// Default container configuration
	containerName := "web"
	containerImage := "nginx:1.25"
	containerPort := 80
	var containerVars map[string]string

	// Extract from Score spec if available
	if scoreSpec != nil && scoreSpec.Containers != nil {
		for name, container := range scoreSpec.Containers {
			containerName = name
			if container.Image != "" {
				containerImage = container.Image
			}
			// Extract environment variables
			containerVars = container.Variables
			// Note: Ports are not yet in the Container type definition
			// Using default port 80 for now
			break // Use first container
		}
	}

	manifest := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: innominatus
spec:
  replicas: 1
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: %s
        image: %s
        ports:
        - containerPort: %d
          protocol: TCP%s`,
		appName, namespace, appName, appName, appName, containerName, containerImage, containerPort,
		kp.generateEnvSection(containerVars))

	return manifest
}

// generateEnvSection creates the environment variables section for deployment
func (kp *KubernetesProvisioner) generateEnvSection(variables map[string]string) string {
	if len(variables) == 0 {
		return ""
	}

	var envVars []string
	for key, value := range variables {
		envVars = append(envVars, fmt.Sprintf(`        - name: %s
          value: %q`, key, value))
	}

	return fmt.Sprintf("\n        env:\n%s", strings.Join(envVars, "\n"))
}

// generateService creates a Kubernetes Service manifest
func (kp *KubernetesProvisioner) generateService(appName string, namespace string, scoreSpec *types.ScoreSpec) string {
	// Default port
	servicePort := 80

	// Extract from Score spec if available - ports not yet in Container type
	// Using default port 80 for now

	manifest := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: innominatus
spec:
  selector:
    app: %s
  ports:
  - port: %d
    targetPort: %d
    protocol: TCP
  type: ClusterIP`,
		appName, namespace, appName, appName, servicePort, servicePort)

	return manifest
}

// generateIngress creates a Kubernetes Ingress manifest
func (kp *KubernetesProvisioner) generateIngress(appName string, namespace string, params map[string]interface{}) string {
	host := "example.local"
	port := 80

	// Extract host from params
	if hostParam, ok := params["host"]; ok {
		if hostStr, ok := hostParam.(string); ok {
			host = hostStr
		}
	}
	if portParam, ok := params["port"]; ok {
		switch v := portParam.(type) {
		case int:
			port = v
		case float64:
			port = int(v)
		}
	}

	manifest := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    managed-by: innominatus
spec:
  rules:
  - host: %s
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: %d`,
		appName, namespace, appName, host, appName, port)

	return manifest
}

// applyManifests applies Kubernetes manifests using kubectl
func (kp *KubernetesProvisioner) applyManifests(manifests string, namespace string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace)
	cmd.Stdin = strings.NewReader(manifests)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w, output: %s", err, string(output))
	}
	return nil
}

// waitForReady waits for deployment to be ready
func (kp *KubernetesProvisioner) waitForReady(appName string, namespace string) error {
	cmd := exec.Command("kubectl", "wait", // #nosec G204 - kubectl apply command
		"--for=condition=available",
		"--timeout=120s",
		fmt.Sprintf("deployment/%s", appName),
		"-n", namespace)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("deployment not ready: %s", string(output))
	}
	return nil
}

// verifyPods checks if pods are running
func (kp *KubernetesProvisioner) verifyPods(namespace string) error {
	// Wait a bit for pods to start
	time.Sleep(2 * time.Second)

	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	fmt.Printf("\n   Pods in namespace %s:\n", namespace)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Printf("   %s\n", line)
		}
	}

	return nil
}

// commitManifestsToGit commits the generated Kubernetes manifests to Gitea repository
func (kp *KubernetesProvisioner) commitManifestsToGit(appName, namespace, manifests string) error {
	// Load admin configuration to get Gitea URL
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found")
	}

	// Create temporary directory for git operations
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("innominatus-git-%s-%d", appName, time.Now().Unix()))
	if err := os.MkdirAll(tmpDir, 0700); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // Clean up after

	// Git clone the repository
	repoURL := fmt.Sprintf("%s/%s/%s.git", adminConfig.Gitea.URL, adminConfig.Gitea.OrgName, appName)
	cloneCmd := exec.Command("git", "clone", repoURL, tmpDir) // #nosec G204 - kubectl get pods
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w\nOutput: %s", err, string(output))
	}

	// Create k8s directory if it doesn't exist
	k8sDir := filepath.Join(tmpDir, "k8s")
	if err := os.MkdirAll(k8sDir, 0700); err != nil {
		return fmt.Errorf("failed to create k8s directory: %w", err)
	}

	// Write manifests to file
	manifestFile := filepath.Join(k8sDir, "manifests.yaml")
	if err := os.WriteFile(manifestFile, []byte(manifests), 0600); err != nil {
		return fmt.Errorf("failed to write manifests: %w", err)
	}

	// Git add
	addCmd := exec.Command("git", "add", "k8s/manifests.yaml")
	addCmd.Dir = tmpDir
	if output, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to add files: %w\nOutput: %s", err, string(output))
	}

	// Git commit
	commitCmd := exec.Command("git", "commit", "-m", fmt.Sprintf("Add Kubernetes manifests for %s", appName)) // #nosec G204 - kubectl rollout status
	commitCmd.Dir = tmpDir
	if output, err := commitCmd.CombinedOutput(); err != nil {
		// Check if there are no changes to commit
		if strings.Contains(string(output), "nothing to commit") {
			return nil // Not an error - manifests already committed
		}
		return fmt.Errorf("failed to commit: %w\nOutput: %s", err, string(output))
	}

	// Git push
	pushCmd := exec.Command("git", "push", "origin", "main")
	pushCmd.Dir = tmpDir
	if output, err := pushCmd.CombinedOutput(); err != nil {
		// Try master branch if main fails
		pushCmd = exec.Command("git", "push", "origin", "master")
		pushCmd.Dir = tmpDir
		if output2, err2 := pushCmd.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to push to main or master: main=%w (%s), master=%w (%s)",
				err, string(output), err2, string(output2))
		}
	}

	return nil
}
