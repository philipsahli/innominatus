package demo

// #nosec G204 - Demo installer executes kubectl and helm with controlled parameters for local demo setup only

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"gopkg.in/yaml.v3"
)

// Installer handles Helm and Kubernetes operations for demo components
type Installer struct {
	kubeContext string
	dryRun      bool
}

// NewInstaller creates a new installer instance
func NewInstaller(kubeContext string, dryRun bool) *Installer {
	return &Installer{
		kubeContext: kubeContext,
		dryRun:      dryRun,
	}
}

// VerifyKubeContext checks if the specified Kubernetes context exists and is accessible
func (i *Installer) VerifyKubeContext() error {
	fmt.Printf("🔍 Verifying Kubernetes context: %s\n", i.kubeContext)

	// Check if context exists
	cmd := exec.Command("kubectl", "config", "get-contexts", i.kubeContext)  // #nosec G204 - kubectl with controlled context
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Kubernetes context '%s' not found. Make sure Docker Desktop is running", i.kubeContext)
	}

	// Test connectivity
	cmd = exec.Command("kubectl", "--context", i.kubeContext, "cluster-info")  // #nosec G204 - kubectl with controlled context
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("unable to connect to Kubernetes cluster with context '%s'", i.kubeContext)
	}

	fmt.Printf("✅ Kubernetes context verified\n")
	return nil
}

// CreateNamespace creates a namespace if it doesn't exist
func (i *Installer) CreateNamespace(namespace string) error {
	fmt.Printf("📁 Creating namespace: %s\n", namespace)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would create namespace: %s\n", namespace)
		return nil
	}

	cmd := exec.Command("kubectl", "--context", i.kubeContext, "create", "namespace", namespace)  // #nosec G204 - kubectl with controlled namespace
	output, err := cmd.CombinedOutput()

	// Ignore error if namespace already exists
	if err != nil && !strings.Contains(string(output), "already exists") {
		return fmt.Errorf("failed to create namespace %s: %v\nOutput: %s", namespace, err, string(output))
	}

	return nil
}

// AddHelmRepo adds a Helm repository
func (i *Installer) AddHelmRepo(repoName, repoURL string) error {
	fmt.Printf("📦 Adding Helm repository: %s (helm repo add %s %s)\n", repoName, repoName, repoURL)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would add Helm repo: %s -> %s\n", repoName, repoURL)
		return nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "helm", "repo", "add", repoName, repoURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add Helm repository %s: %v\nOutput: %s", repoName, err, string(output))
	}
	fmt.Printf("✅ Helm repository %s added successfully\n", repoName)

	// Update repo with timeout - make this non-blocking and optional
	fmt.Printf("🔄 Updating Helm repositories (helm repo update)\n")
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer updateCancel()

	cmd = exec.CommandContext(updateCtx, "helm", "repo", "update")
	updateOutput, err := cmd.CombinedOutput()
	if err != nil {
		// Don't fail if repo update times out - the repo add might have succeeded
		fmt.Printf("Warning: Helm repo update failed (continuing anyway): %v\nOutput: %s\n", err, string(updateOutput))
	} else {
		fmt.Printf("✅ Helm repositories updated successfully\n")
	}

	return nil
}

// InstallComponent installs a single component using Helm
func (i *Installer) InstallComponent(component DemoComponent) error {
	fmt.Printf("🚀 Installing component: %s\n", component.Name)
	fmt.Printf("   Chart: %s/%s version %s\n", component.Repo, component.Chart, component.Version)

	// Create namespace first
	if err := i.CreateNamespace(component.Namespace); err != nil {
		return err
	}

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would install: %s/%s version %s\n",
			component.Repo, component.Chart, component.Version)
		return nil
	}

	// Extract repo name from URL
	repoName := i.getRepoName(component.Repo)
	fmt.Printf("   Using repo name: %s\n", repoName)

	// Add repository
	fmt.Printf("   Adding Helm repository...\n")
	if err := i.AddHelmRepo(repoName, component.Repo); err != nil {
		return err
	}
	fmt.Printf("   Repository added successfully\n")

	// Create values file
	valuesFile, err := i.createValuesFile(component)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(valuesFile) }()

	// Install or upgrade with Helm
	chartRef := fmt.Sprintf("%s/%s", repoName, component.Chart)
	helmCmd := []string{"helm", "upgrade", "--install", component.Name,
		chartRef,
		"--version", component.Version,
		"--namespace", component.Namespace,
		"--kube-context", i.kubeContext,
		"--values", valuesFile,
		"--wait",
		"--timeout", "10m"}

	fmt.Printf("   Executing: %s\n", strings.Join(helmCmd, " "))

	// Create context with timeout for helm install
	installCtx, installCancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer installCancel()

	cmd := exec.CommandContext(installCtx, "helm", "upgrade", "--install", component.Name,  // #nosec G204 - helm with controlled parameters
		chartRef,
		"--version", component.Version,
		"--namespace", component.Namespace,
		"--kube-context", i.kubeContext,
		"--values", valuesFile,
		"--wait",
		"--timeout", "10m")

	fmt.Printf("   Starting Helm installation (timeout: 15 minutes)...\n")
	fmt.Printf("   This may take several minutes for database initialization...\n")
	fmt.Printf("   Progress: ")

	// Start the command
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start helm install for %s: %v", component.Name, err)
	}

	// Monitor progress in a goroutine
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Show progress while waiting
	statusTicker := time.NewTicker(30 * time.Second)
	dotTicker := time.NewTicker(1 * time.Second)
	defer statusTicker.Stop()
	defer dotTicker.Stop()

	for {
		select {
		case err := <-done:
			fmt.Printf("\n   ✅ Installation completed successfully\n")
			if err != nil {
				// Get output if command failed
				output, _ := cmd.CombinedOutput()
				return fmt.Errorf("failed to install %s: %v\nOutput: %s", component.Name, err, string(output))
			}
			return nil
		case <-dotTicker.C:
			fmt.Printf(".")
		case <-statusTicker.C:
			fmt.Printf("\n   ⏳ Still installing %s... (checking pods)\n", component.Name)
			// Show pod status
			statusCmd := exec.Command("kubectl", "get", "pods", "-n", component.Namespace, "--no-headers")  // #nosec G204 - kubectl for pod status
			if statusOutput, err := statusCmd.Output(); err == nil {
				lines := strings.Split(strings.TrimSpace(string(statusOutput)), "\n")
				if len(lines) > 0 && lines[0] != "" {
					fmt.Printf("   Pods: %d found\n", len(lines))
				}
			}
			fmt.Printf("   Progress: ")
		case <-installCtx.Done():
			fmt.Printf("\n")
			_ = cmd.Process.Kill()
			return fmt.Errorf("helm install for %s timed out after 15 minutes", component.Name)
		}
	}
}

// UninstallComponent uninstalls a component using Helm
func (i *Installer) UninstallComponent(component DemoComponent) error {
	fmt.Printf("🗑️  Uninstalling component: %s\n", component.Name)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would uninstall: %s from namespace %s\n",
			component.Name, component.Namespace)
		return nil
	}

	// Uninstall Helm release
	cmd := exec.Command("helm", "uninstall", component.Name,  // #nosec G204 - helm uninstall command
		"--namespace", component.Namespace,
		"--kube-context", i.kubeContext)

	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "not found") {
		return fmt.Errorf("failed to uninstall %s: %v\nOutput: %s", component.Name, err, string(output))
	}

	fmt.Printf("✅ %s uninstalled\n", component.Name)
	return nil
}

// DeleteNamespace deletes a namespace and all its resources
func (i *Installer) DeleteNamespace(namespace string) error {
	fmt.Printf("🗑️  Deleting namespace: %s\n", namespace)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would delete namespace: %s\n", namespace)
		return nil
	}

	cmd := exec.Command("kubectl", "--context", i.kubeContext, "delete", "namespace", namespace, "--ignore-not-found=true")  // #nosec G204 - kubectl delete namespace
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete namespace %s: %v", namespace, err)
	}

	return nil
}

// WaitForDeployment waits for a deployment to be ready
func (i *Installer) WaitForDeployment(namespace, deploymentName string, timeout time.Duration) error {
	fmt.Printf("⏳ Waiting for deployment %s/%s to be ready...\n", namespace, deploymentName)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would wait for deployment: %s/%s\n", namespace, deploymentName)
		return nil
	}

	cmd := exec.Command("kubectl", "--context", i.kubeContext, "wait",  // #nosec G204 - kubectl wait command
		"--for=condition=Available",
		"--timeout="+timeout.String(),
		"-n", namespace,
		"deployment/"+deploymentName)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deployment %s/%s did not become ready within %v", namespace, deploymentName, timeout)
	}

	fmt.Printf("✅ Deployment %s/%s is ready\n", namespace, deploymentName)
	return nil
}

// DeployManifest deploys a Kubernetes manifest from YAML content
func (i *Installer) DeployManifest(namespace, name, yamlContent string) error {
	fmt.Printf("📄 Deploying manifest: %s\n", name)

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would deploy manifest: %s to namespace %s\n", name, namespace)
		return nil
	}

	// Create namespace first
	if err := i.CreateNamespace(namespace); err != nil {
		return err
	}

	// Write manifest to temporary file
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-*.yaml", name))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		return fmt.Errorf("failed to write manifest: %v", err)
	}
	_ = tmpFile.Close()

	// Apply manifest
	cmd := exec.Command("kubectl", "--context", i.kubeContext, "apply", "-f", tmpFile.Name(), "-n", namespace)  // #nosec G204 - kubectl apply command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply manifest %s: %v\nOutput: %s", name, err, string(output))
	}

	fmt.Printf("✅ Manifest %s deployed\n", name)
	return nil
}

// CheckHelmRelease checks if a Helm release exists
func (i *Installer) CheckHelmRelease(releaseName, namespace string) (bool, error) {
	cmd := exec.Command("helm", "list", "-n", namespace, "--kube-context", i.kubeContext, "-q")  // #nosec G204 - helm list command
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	releases := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, release := range releases {
		if strings.TrimSpace(release) == releaseName {
			return true, nil
		}
	}

	return false, nil
}

// getRepoName extracts a repository name from URL
func (i *Installer) getRepoName(repoURL string) string {
	// Simple extraction - take the last part of the domain
	parts := strings.Split(repoURL, "/")
	for _, part := range parts {
		if strings.Contains(part, ".") && !strings.HasPrefix(part, "http") {
			return strings.Split(part, ".")[0]
		}
	}
	return "repo"
}

// createValuesFile creates a temporary values file for Helm
func (i *Installer) createValuesFile(component DemoComponent) (string, error) {
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("%s-values-*.yaml", component.Name))
	if err != nil {
		return "", fmt.Errorf("failed to create temp values file: %v", err)
	}

	// Convert values to YAML
	yamlData, err := yaml.Marshal(component.Values)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to marshal values to YAML: %v", err)
	}

	if _, err := tmpFile.Write(yamlData); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write values file: %v", err)
	}

	_ = tmpFile.Close()
	return tmpFile.Name(), nil
}

// InstallKubernetesDashboard installs Kubernetes Dashboard and creates admin user
func (i *Installer) InstallKubernetesDashboard() error {
	fmt.Printf("🚀 Installing Kubernetes Dashboard...\n")

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would install Kubernetes Dashboard\n")
		return nil
	}

	// Create namespace first
	if err := i.CreateNamespace("kubernetes-dashboard"); err != nil {
		return err
	}

	// Download and apply official dashboard manifest
	dashboardURL := "https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml"
	cmd := exec.Command("kubectl", "--context", i.kubeContext, "apply", "-f", dashboardURL)  // #nosec G204 - kubectl apply for dashboard
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install dashboard: %v\nOutput: %s", err, string(output))
	}

	// Create admin user ServiceAccount
	adminUserManifest := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: admin-user
  namespace: kubernetes-dashboard
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admin-user
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: admin-user
  namespace: kubernetes-dashboard
`

	if err := i.DeployManifest("kubernetes-dashboard", "admin-user", adminUserManifest); err != nil {
		return err
	}

	// Create Ingress for Dashboard
	dashboardIngress := `
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: kubernetes-dashboard-ingress
  namespace: kubernetes-dashboard
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/backend-protocol: "HTTPS"
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
spec:
  rules:
  - host: k8s.localtest.me
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: kubernetes-dashboard
            port:
              number: 443
`

	if err := i.DeployManifest("kubernetes-dashboard", "dashboard-ingress", dashboardIngress); err != nil {
		return err
	}

	fmt.Printf("✅ Kubernetes Dashboard installed successfully\n")
	return nil
}

// InstallDemoApp installs the demo application
func (i *Installer) InstallDemoApp() error {
	fmt.Printf("🚀 Installing Demo Application...\n")

	if i.dryRun {
		fmt.Printf("   [DRY RUN] Would install Demo Application\n")
		return nil
	}

	// Demo app deployment (same as in git.go but applied directly)
	demoAppManifest := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-app
  namespace: demo
spec:
  replicas: 2
  selector:
    matchLabels:
      app: demo-app
  template:
    metadata:
      labels:
        app: demo-app
    spec:
      containers:
      - name: nginx
        image: nginx:1.21
        ports:
        - containerPort: 80
        volumeMounts:
        - name: html
          mountPath: /usr/share/nginx/html
      volumes:
      - name: html
        configMap:
          name: demo-app-html
---
apiVersion: v1
kind: Service
metadata:
  name: demo-app-service
  namespace: demo
spec:
  selector:
    app: demo-app
  ports:
  - port: 80
    targetPort: 80
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-app-html
  namespace: demo
data:
  index.html: |
    <!DOCTYPE html>
    <html>
    <head>
        <title>OpenAlps Demo</title>
        <style>
            body { font-family: Arial, sans-serif; margin: 40px; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; }
            .container { text-align: center; padding: 60px; }
            h1 { font-size: 3em; margin-bottom: 20px; }
            p { font-size: 1.2em; }
            .links { margin-top: 40px; }
            .link { display: inline-block; margin: 10px; padding: 15px 30px; background: rgba(255,255,255,0.2); text-decoration: none; color: white; border-radius: 5px; }
        </style>
    </head>
    <body>
        <div class="container">
            <h1>🚀 OpenAlps Demo Environment</h1>
            <p>Welcome to your demo platform! This application was deployed from a Score specification.</p>
            <div class="links">
                <a href="http://gitea.localtest.me" class="link">📚 Gitea</a>
                <a href="http://argocd.localtest.me" class="link">🔄 ArgoCD</a>
                <a href="http://vault.localtest.me" class="link">🔒 Vault</a>
                <a href="http://grafana.localtest.me" class="link">📊 Grafana</a>
                <a href="http://prometheus.localtest.me" class="link">📈 Prometheus</a>
                <a href="http://k8s.localtest.me" class="link">🎛️ Dashboard</a>
            </div>
        </div>
    </body>
    </html>
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: demo-app-ingress
  namespace: demo
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: demo.localtest.me
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: demo-app-service
            port:
              number: 80
`

	return i.DeployManifest("demo", "demo-app", demoAppManifest)
}