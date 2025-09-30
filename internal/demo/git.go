package demo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitManager handles Git operations for the demo environment
type GitManager struct {
	giteaURL     string
	username     string
	password     string
	repoName     string
	workDir      string
}

// NewGitManager creates a new Git manager
func NewGitManager(giteaURL, username, password, repoName string) *GitManager {
	return &GitManager{
		giteaURL: giteaURL,
		username: username,
		password: password,
		repoName: repoName,
		workDir:  "",
	}
}

// SeedRepository creates and seeds the platform-config repository
func (g *GitManager) SeedRepository() error {
	fmt.Printf("🌱 Seeding platform-config repository...\n")

	// Wait for Gitea to be ready
	if err := g.waitForGitea(); err != nil {
		return fmt.Errorf("Gitea not ready: %v", err)
	}

	// Create temporary working directory
	workDir, err := os.MkdirTemp("", "platform-config-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	g.workDir = workDir
	defer os.RemoveAll(workDir)

	// Check if repository already exists
	repoExists, err := g.checkRepositoryExists()
	if err != nil {
		return fmt.Errorf("failed to check if repository exists: %v", err)
	}

	if repoExists {
		fmt.Printf("📂 Repository already exists, updating...\n")
		if err := g.updateRepository(); err != nil {
			return err
		}
	} else {
		fmt.Printf("📂 Creating new repository...\n")
		if err := g.createRepository(); err != nil {
			return err
		}
	}

	return nil
}

// waitForGitea waits for Gitea to be ready
func (g *GitManager) waitForGitea() error {
	fmt.Printf("⏳ Waiting for Gitea to be ready...\n")

	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		if i > 0 {
			time.Sleep(10 * time.Second)
		}

		// Try to access Gitea
		cmd := exec.Command("curl", "-f", "-s", fmt.Sprintf("http://%s/api/v1/version", g.giteaURL))
		if err := cmd.Run(); err == nil {
			fmt.Printf("✅ Gitea is ready\n")
			return nil
		}

		fmt.Printf("   Retry %d/%d...\n", i+1, maxRetries)
	}

	return fmt.Errorf("Gitea did not become ready within timeout")
}

// checkRepositoryExists checks if the repository already exists
func (g *GitManager) checkRepositoryExists() (bool, error) {
	repoURL := fmt.Sprintf("http://%s:%s@%s/%s/%s.git",
		g.username, g.password, g.giteaURL, g.username, g.repoName)

	cmd := exec.Command("git", "ls-remote", repoURL)
	err := cmd.Run()
	return err == nil, nil
}

// createRepository creates a new repository and seeds it
func (g *GitManager) createRepository() error {
	// Initialize git repo
	if err := g.runGitCommand(g.workDir, "init"); err != nil {
		return err
	}

	// Configure git
	if err := g.runGitCommand(g.workDir, "config", "user.name", "OpenAlps Demo"); err != nil {
		return err
	}
	if err := g.runGitCommand(g.workDir, "config", "user.email", "demo@openalps.local"); err != nil {
		return err
	}

	// Create manifests
	if err := g.createManifests(); err != nil {
		return err
	}

	// Add files
	if err := g.runGitCommand(g.workDir, "add", "."); err != nil {
		return err
	}

	// Commit
	if err := g.runGitCommand(g.workDir, "commit", "-m", "Initial commit: OpenAlps demo environment"); err != nil {
		return err
	}

	// Create repository in Gitea via API
	if err := g.createGiteaRepository(); err != nil {
		return err
	}

	// Add remote and push
	repoURL := fmt.Sprintf("http://%s:%s@%s/%s/%s.git",
		g.username, g.password, g.giteaURL, g.username, g.repoName)

	if err := g.runGitCommand(g.workDir, "remote", "add", "origin", repoURL); err != nil {
		return err
	}

	if err := g.runGitCommand(g.workDir, "push", "-u", "origin", "main"); err != nil {
		return err
	}

	fmt.Printf("✅ Repository created and seeded\n")
	return nil
}

// updateRepository updates an existing repository
func (g *GitManager) updateRepository() error {
	// Clone existing repository
	repoURL := fmt.Sprintf("http://%s:%s@%s/%s/%s.git",
		g.username, g.password, g.giteaURL, g.username, g.repoName)

	if err := g.runGitCommand("", "clone", repoURL, g.workDir); err != nil {
		return err
	}

	// Configure git
	if err := g.runGitCommand(g.workDir, "config", "user.name", "OpenAlps Demo"); err != nil {
		return err
	}
	if err := g.runGitCommand(g.workDir, "config", "user.email", "demo@openalps.local"); err != nil {
		return err
	}

	// Update manifests
	if err := g.createManifests(); err != nil {
		return err
	}

	// Check if there are changes
	if err := g.runGitCommand(g.workDir, "add", "."); err != nil {
		return err
	}

	// Check git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = g.workDir
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %v", err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		fmt.Printf("📂 Repository is up to date\n")
		return nil
	}

	// Commit changes
	if err := g.runGitCommand(g.workDir, "commit", "-m", "Update OpenAlps demo environment"); err != nil {
		return err
	}

	// Push changes
	if err := g.runGitCommand(g.workDir, "push"); err != nil {
		return err
	}

	fmt.Printf("✅ Repository updated\n")
	return nil
}

// createGiteaRepository creates a repository in Gitea via API
func (g *GitManager) createGiteaRepository() error {
	apiURL := fmt.Sprintf("http://%s/api/v1/user/repos", g.giteaURL)

	payload := fmt.Sprintf(`{
		"name": "%s",
		"description": "OpenAlps Demo Platform Configuration",
		"private": false,
		"auto_init": false
	}`, g.repoName)

	cmd := exec.Command("curl", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-u", fmt.Sprintf("%s:%s", g.username, g.password),
		"-d", payload,
		apiURL)

	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "already exists") {
		return fmt.Errorf("failed to create repository: %v\nOutput: %s", err, string(output))
	}

	return nil
}

// createManifests creates all the necessary manifest files
func (g *GitManager) createManifests() error {
	fmt.Printf("📄 Creating manifests...\n")

	// Create directory structure
	dirs := []string{
		"apps",
		"apps/infrastructure",
		"apps/monitoring",
		"apps/demo",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(g.workDir, dir), 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	// Create root app of apps
	if err := g.createRootApp(); err != nil {
		return err
	}

	// Create individual application manifests
	if err := g.createApplicationManifests(); err != nil {
		return err
	}

	// Create demo app manifests
	if err := g.createDemoAppManifests(); err != nil {
		return err
	}

	return nil
}

// createRootApp creates the root app-of-apps manifest
func (g *GitManager) createRootApp() error {
	rootApp := `apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: root-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: http://gitea.localtest.me/admin/platform-config.git
    targetRevision: HEAD
    path: apps
  destination:
    server: https://kubernetes.default.svc
    namespace: argocd
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
`

	return g.writeFile("root-app.yaml", rootApp)
}

// createApplicationManifests creates ArgoCD Application manifests for each component
func (g *GitManager) createApplicationManifests() error {
	// Create individual app manifests
	apps := []struct {
		name      string
		namespace string
		path      string
	}{
		{"gitea-app", "gitea", "apps/infrastructure"},
		{"vault-app", "vault", "apps/infrastructure"},
		{"prometheus-app", "monitoring", "apps/monitoring"},
		{"grafana-app", "monitoring", "apps/monitoring"},
		{"demo-app", "demo", "apps/demo"},
	}

	for _, app := range apps {
		manifest := fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: argocd
spec:
  project: default
  source:
    repoURL: http://gitea.localtest.me/admin/platform-config.git
    targetRevision: HEAD
    path: %s
  destination:
    server: https://kubernetes.default.svc
    namespace: %s
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
`, app.name, app.path, app.namespace)

		filename := filepath.Join("apps", fmt.Sprintf("%s.yaml", app.name))
		if err := g.writeFile(filename, manifest); err != nil {
			return err
		}
	}

	return nil
}

// createDemoAppManifests creates the demo application manifests
func (g *GitManager) createDemoAppManifests() error {
	// Demo app deployment
	deployment := `apiVersion: apps/v1
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

	return g.writeFile("apps/demo/demo-app.yaml", deployment)
}

// writeFile writes content to a file relative to the work directory
func (g *GitManager) writeFile(filename, content string) error {
	fullPath := filepath.Join(g.workDir, filename)

	// Ensure directory exists
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %v", filename, err)
	}

	return nil
}

// runGitCommand runs a git command in the specified directory
func (g *GitManager) runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %v\nOutput: %s", err, string(output))
	}

	return nil
}