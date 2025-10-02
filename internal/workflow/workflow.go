package workflow

import (
	"encoding/json"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/types"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Spinner struct {
	message string
	active  bool
	mu      sync.Mutex
	done    chan bool
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

func (s *Spinner) Start() {
	s.mu.Lock()
	if s.active {
		s.mu.Unlock()
		return
	}
	s.active = true
	s.mu.Unlock()

	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				if s.active {
					fmt.Printf("\r%s %s", chars[i%len(chars)], s.message)
				}
				s.mu.Unlock()
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

func (s *Spinner) Stop(success bool, resultMessage string) {
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return
	}
	s.active = false
	s.mu.Unlock()

	s.done <- true

	if success {
		fmt.Printf("\r✅ %s\n", resultMessage)
	} else {
		fmt.Printf("\r❌ %s\n", resultMessage)
	}
}

func (s *Spinner) Update(message string) {
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

func RunWorkflow(w types.Workflow, appName string, envType string) error {
	fmt.Printf("Starting workflow with %d steps for app '%s' (env: %s)\n\n", len(w.Steps), appName, envType)

	for i, step := range w.Steps {
		fmt.Printf("Step %d/%d: %s (%s)\n", i+1, len(w.Steps), step.Name, step.Type)

		spinner := NewSpinner(fmt.Sprintf("Initializing %s step...", step.Type))
		spinner.Start()

		err := runStepWithSpinner(step, appName, envType, spinner)
		if err != nil {
			spinner.Stop(false, fmt.Sprintf("Step '%s' failed: %v", step.Name, err))
			return fmt.Errorf("workflow failed at step '%s': %w", step.Name, err)
		}

		spinner.Stop(true, fmt.Sprintf("Step '%s' completed successfully", step.Name))
		fmt.Println()
	}

	fmt.Println("🎉 Workflow completed successfully!")
	return nil
}

//nolint:unused // Legacy implementation kept for reference
func runStep(step types.Step) error {
	switch step.Type {
	case "terraform":
		return runTerraformStep(step)
	case "ansible":
		return runAnsibleStep(step)
	case "kubernetes":
		return runKubernetesStep(step)
	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

func runStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	switch step.Type {
	case "terraform":
		return runTerraformStepWithSpinner(step, appName, envType, spinner)
	case "ansible":
		return runAnsibleStepWithSpinner(step, appName, envType, spinner)
	case "kubernetes":
		return runKubernetesStepWithSpinner(step, appName, envType, spinner)
	case "gitea-repo":
		return runGiteaRepoStepWithSpinner(step, appName, envType, spinner)
	case "argocd-app":
		return runArgoCDAppStepWithSpinner(step, appName, envType, spinner)
	case "git-commit-manifests":
		return runGitCommitManifestsStepWithSpinner(step, appName, envType, spinner)
	case "dummy":
		return runDummyStepWithSpinner(step, appName, envType, spinner)
	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

//nolint:unused // Legacy implementation kept for reference
func runTerraformStep(step types.Step) error {
	fmt.Printf("Running Terraform in path: %s\n", step.Path)

	// Check if directory exists
	if _, err := os.Stat(step.Path); os.IsNotExist(err) {
		return fmt.Errorf("terraform path does not exist: %s", step.Path)
	}

	// Run terraform init
	fmt.Println("Running: terraform init")
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = step.Path
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr

	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Run terraform apply
	fmt.Println("Running: terraform apply -auto-approve")
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = step.Path
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	// Get terraform outputs
	fmt.Println("Getting terraform outputs...")
	outputCmd := exec.Command("terraform", "output", "-json")
	outputCmd.Dir = step.Path
	outputCmd.Stderr = os.Stderr

	output, err := outputCmd.Output()
	if err != nil {
		fmt.Printf("Warning: could not get terraform outputs: %v\n", err)
		return nil // Don't fail the step for output errors
	}

	if len(output) > 0 {
		fmt.Println("Terraform outputs:")
		var outputs map[string]interface{}
		if err := json.Unmarshal(output, &outputs); err == nil {
			for key, value := range outputs {
				fmt.Printf("  %s: %v\n", key, value)
			}
		} else {
			fmt.Printf("  %s\n", string(output))
		}
	}

	return nil
}

func runTerraformStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	spinner.Update("Checking Terraform path...")

	// Check if directory exists
	if _, err := os.Stat(step.Path); os.IsNotExist(err) {
		return fmt.Errorf("terraform path does not exist: %s", step.Path)
	}

	// Run terraform init
	spinner.Update("Running terraform init...")
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = step.Path
	if spinner == nil {
		initCmd.Stdout = os.Stdout
		initCmd.Stderr = os.Stderr
	}

	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Run terraform apply
	spinner.Update("Applying terraform configuration...")
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = step.Path
	if spinner == nil {
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
	}

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	// Get terraform outputs
	spinner.Update("Retrieving terraform outputs...")
	outputCmd := exec.Command("terraform", "output", "-json")
	outputCmd.Dir = step.Path
	if spinner == nil {
		outputCmd.Stderr = os.Stderr
	}

	output, err := outputCmd.Output()
	if err != nil {
		// Don't fail the step for output errors
		return nil
	}

	if len(output) > 0 && spinner == nil {
		fmt.Println("Terraform outputs:")
		var outputs map[string]interface{}
		if err := json.Unmarshal(output, &outputs); err == nil {
			for key, value := range outputs {
				fmt.Printf("  %s: %v\n", key, value)
			}
		} else {
			fmt.Printf("  %s\n", string(output))
		}
	}

	return nil
}

//nolint:unused // Legacy implementation kept for reference
func runAnsibleStep(step types.Step) error {
	if step.Playbook == "" {
		return fmt.Errorf("ansible step requires playbook field")
	}

	fmt.Printf("Running Ansible playbook: %s\n", step.Playbook)

	// Check if playbook exists
	if _, err := os.Stat(step.Playbook); os.IsNotExist(err) {
		return fmt.Errorf("ansible playbook does not exist: %s", step.Playbook)
	}

	// Run ansible-playbook
	fmt.Printf("Running: ansible-playbook %s\n", step.Playbook)
	cmd := exec.Command("ansible-playbook", step.Playbook) // #nosec G204 - ansible playbook from validated workflow definition
	if step.Path != "" {
		cmd.Dir = step.Path
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible-playbook failed: %w", err)
	}

	return nil
}

func runAnsibleStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	if step.Playbook == "" {
		return fmt.Errorf("ansible step requires playbook field")
	}

	spinner.Update("Checking Ansible playbook...")

	// Check if playbook exists
	if _, err := os.Stat(step.Playbook); os.IsNotExist(err) {
		return fmt.Errorf("ansible playbook does not exist: %s", step.Playbook)
	}

	// Run ansible-playbook
	spinner.Update("Running ansible-playbook...")
	cmd := exec.Command("ansible-playbook", step.Playbook) // #nosec G204 - playbook from validated workflow definition
	if step.Path != "" {
		cmd.Dir = step.Path
	}
	if spinner == nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible-playbook failed: %w", err)
	}

	return nil
}

//nolint:unused // Legacy implementation kept for reference
func runKubernetesStep(step types.Step) error {
	fmt.Printf("Running Kubernetes deployment")
	if step.Namespace != "" {
		fmt.Printf(" in namespace: %s", step.Namespace)
	}
	fmt.Println()

	// Placeholder for actual kubernetes deployment logic
	fmt.Println("✅ Kubernetes deployment completed successfully")
	return nil
}

// generateKubernetesManifests generates Kubernetes manifests from Score spec information
func generateKubernetesManifests(appName string, namespace string, step types.Step) string {
	// For now, generate a basic nginx deployment
	// In a full implementation, this should parse the Score spec and generate appropriate manifests

	manifests := fmt.Sprintf(`---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
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
      - name: web
        image: nginx:1.25
        ports:
        - containerPort: 80
          protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  selector:
    app: %s
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
  type: ClusterIP
`, appName, namespace, appName, appName, appName, appName, namespace, appName, appName)

	return manifests
}

func runKubernetesStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	spinner.Update("Setting up Kubernetes deployment...")

	namespace := step.Namespace
	if namespace == "" {
		namespace = appName
	}

	spinner.Update(fmt.Sprintf("Creating namespace: %s", namespace))

	// Create namespace if it doesn't exist
	createNsCmd := exec.Command("kubectl", "create", "namespace", namespace) // #nosec G204 - namespace from workflow config
	output, err := createNsCmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "AlreadyExists") {
		return fmt.Errorf("failed to create namespace: %w, output: %s", err, string(output))
	}

	spinner.Update(fmt.Sprintf("Deploying to namespace: %s", namespace))

	// Generate Kubernetes manifests
	manifests := generateKubernetesManifests(appName, namespace, step)

	spinner.Update("Applying Kubernetes manifests...")

	// Apply manifests using kubectl
	applyCmd := exec.Command("kubectl", "apply", "-f", "-", "-n", namespace) // #nosec G204 - namespace from workflow config
	applyCmd.Stdin = strings.NewReader(manifests)
	output, err = applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply manifests: %w, output: %s", err, string(output))
	}

	spinner.Update("Waiting for deployment to be ready...")

	// Wait for deployment to be ready (with timeout)
	// #nosec G204 - appName and namespace from validated workflow definition
	waitCmd := exec.Command("kubectl", "wait", "--for=condition=available",
		"--timeout=120s",
		fmt.Sprintf("deployment/%s", appName),
		"-n", namespace)
	output, err = waitCmd.CombinedOutput()
	if err != nil {
		// Don't fail if wait times out, just log it
		fmt.Printf("\nWarning: Deployment readiness check: %s\n", string(output))
	}

	spinner.Update("Checking deployment status...")

	// Verify pods are running
	getPodsCmd := exec.Command("kubectl", "get", "pods", "-n", namespace) // #nosec G204 - namespace from workflow config
	output, err = getPodsCmd.CombinedOutput()
	if err != nil {
		fmt.Printf("\nWarning: Could not get pods: %v\n", err)
	} else {
		fmt.Printf("\nPods in namespace %s:\n%s\n", namespace, string(output))
	}

	return nil
}

// runGiteaRepoStepWithSpinner creates a repository in Gitea
func runGiteaRepoStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	if step.RepoName == "" {
		return fmt.Errorf("gitea-repo step requires repoName field")
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found in admin-config.yaml")
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating repository: %s", step.RepoName))
	}

	// Create repository via Gitea API
	owner := step.Owner
	if owner == "" {
		owner = adminConfig.Gitea.Username
	}

	repoData := map[string]interface{}{
		"name":        step.RepoName,
		"description": step.Description,
		"private":     step.Private,
		"auto_init":   true,
	}

	repoJSON, err := json.Marshal(repoData)
	if err != nil {
		return fmt.Errorf("failed to marshal repository data: %w", err)
	}

	createURL := fmt.Sprintf("%s/api/v1/user/repos", adminConfig.Gitea.URL)
	if owner != adminConfig.Gitea.Username {
		// Use the specified owner as organization name
		createURL = fmt.Sprintf("%s/api/v1/orgs/%s/repos", adminConfig.Gitea.URL, owner)
	}
	req, err := http.NewRequest("POST", createURL, strings.NewReader(string(repoJSON)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 409 {
		fmt.Printf("Repository %s/%s already exists, skipping creation\n", owner, step.RepoName)
	} else if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create repository, status %d: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Gitea repository available at: %s/%s/%s\n", adminConfig.Gitea.URL, owner, step.RepoName)
	return nil
}

// runArgoCDAppStepWithSpinner creates an ArgoCD Application with sync waiting
func runArgoCDAppStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	if step.AppName == "" {
		step.AppName = fmt.Sprintf("%s-%s", appName, envType)
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.ArgoCD.URL == "" {
		return fmt.Errorf("argocd configuration not found in admin-config.yaml")
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating ArgoCD Application: %s", step.AppName))
	}

	// Authenticate with ArgoCD to get JWT token
	token, err := authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with ArgoCD: %w", err)
	}

	// Determine repository URL
	repoURL := step.RepoURL
	if repoURL == "" && step.RepoName != "" {
		owner := step.Owner
		if owner == "" {
			owner = adminConfig.Gitea.Username
		}
		// Use internal service URL for ArgoCD
		internalGiteaURL := "http://gitea-http.gitea.svc.cluster.local:3000"
		repoURL = fmt.Sprintf("%s/%s/%s.git", internalGiteaURL, owner, step.RepoName)
	}

	if repoURL == "" {
		return fmt.Errorf("argocd-app step requires either repoURL or repoName field")
	}

	// Set defaults
	project := step.Project
	if project == "" {
		project = "default"
	}

	targetPath := step.TargetPath
	if targetPath == "" {
		targetPath = "."
	}

	namespace := step.Namespace

	// Determine sync policy
	syncPolicy := map[string]interface{}{}
	if step.SyncPolicy == "auto" {
		syncPolicy = map[string]interface{}{
			"automated": map[string]interface{}{
				"prune":    true,
				"selfHeal": true,
			},
			"syncOptions": []string{
				"CreateNamespace=true",
			},
		}
	}

	// Create ArgoCD Application manifest
	application := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      step.AppName,
			"namespace": "argocd",
			"labels": map[string]interface{}{
				"app":         appName,
				"environment": envType,
				"managed-by":  "innominatus",
			},
		},
		"spec": map[string]interface{}{
			"project": project,
			"source": map[string]interface{}{
				"repoURL":        repoURL,
				"targetRevision": "HEAD",
				"path":           targetPath,
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": namespace,
			},
			"syncPolicy": syncPolicy,
		},
	}

	// Convert to JSON
	appJSON, err := json.Marshal(application)
	if err != nil {
		return fmt.Errorf("failed to marshal application: %w", err)
	}

	// Create the application
	createURL := fmt.Sprintf("%s/api/v1/applications", adminConfig.ArgoCD.URL)
	createReq, err := http.NewRequest("POST", createURL, strings.NewReader(string(appJSON)))
	if err != nil {
		return fmt.Errorf("failed to create application request: %w", err)
	}

	createReq.Header.Set("Authorization", "Bearer "+token)
	createReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	createResp, err := client.Do(createReq)
	if err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}
	defer func() { _ = createResp.Body.Close() }()

	if createResp.StatusCode != 200 && createResp.StatusCode != 201 {
		body, _ := io.ReadAll(createResp.Body)
		return fmt.Errorf("failed to create ArgoCD application, status %d: %s", createResp.StatusCode, string(body))
	}

	appURL := fmt.Sprintf("%s/applications/%s", adminConfig.ArgoCD.URL, step.AppName)
	fmt.Printf("ArgoCD Application available at: %s\n", appURL)
	fmt.Printf("Repository: %s\n", repoURL)

	// Check if we should wait for sync completion
	waitForSync := step.WaitForSync == nil || *step.WaitForSync

	if waitForSync {
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = 300 // Default 5 minutes
		}

		if err := waitForArgoCDSync(step.AppName, adminConfig.ArgoCD.URL, token, timeout, spinner); err != nil {
			return fmt.Errorf("failed waiting for ArgoCD sync: %w", err)
		}
	}

	return nil
}

// runGitCommitManifestsStepWithSpinner generates and commits Kubernetes manifests
func runGitCommitManifestsStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	if step.RepoName == "" {
		return fmt.Errorf("git-commit-manifests step requires repoName field")
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration
	adminConfig, err := admin.LoadAdminConfig("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found in admin-config.yaml")
	}

	if spinner != nil {
		spinner.Update("Generating manifests from template...")
	}

	// Use "development" as default to match ArgoCD namespace pattern
	if envType == "default" {
		envType = "development"
	}
	namespace := fmt.Sprintf("%s-%s", appName, envType)

	// Generate Kubernetes manifests (simplified template)
	manifestTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
    environment: %s
    managed-by: innominatus
spec:
  replicas: 2
  selector:
    matchLabels:
      app: %s
      component: web
  template:
    metadata:
      labels:
        app: %s
        component: web
        environment: %s
    spec:
      containers:
      - name: web
        image: nginx:latest
        ports:
        - containerPort: 80
          name: http
        env:
        - name: APP_NAME
          value: "platform-config"
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "100m"
---
apiVersion: v1
kind: Service
metadata:
  name: %s-service
  namespace: %s
  labels:
    app: %s
    component: web
spec:
  selector:
    app: %s
    component: web
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: ClusterIP`

	manifestContent := fmt.Sprintf(manifestTemplate,
		appName, namespace, appName, envType,
		appName, appName, envType,
		appName, namespace, appName, appName)

	// Set defaults
	gitBranch := step.GitBranch
	if gitBranch == "" {
		gitBranch = "main"
	}

	// Clone repository and commit manifests
	owner := step.Owner
	if owner == "" {
		owner = adminConfig.Gitea.Username
	}

	if spinner != nil {
		spinner.Update("Cloning repository...")
	}

	// Create temporary directory for git operations
	tmpDir := fmt.Sprintf("/tmp/score-repo-%s", step.RepoName)
	_ = os.RemoveAll(tmpDir) // Clean up any existing directory

	// Clone repository
	repoURL := fmt.Sprintf("%s/%s/%s.git", adminConfig.Gitea.URL, owner, step.RepoName)
	cloneCmd := exec.Command("git", "clone", repoURL, tmpDir) // #nosec G204 - repo URL from admin config and workflow step
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Write manifest file
	manifestFilePath := filepath.Join(tmpDir, "deployment.yaml")
	if err := os.WriteFile(manifestFilePath, []byte(manifestContent), 0600); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	if spinner != nil {
		spinner.Update("Configuring git...")
	}

	// Configure git
	if err := runGitCommand(tmpDir, "config", "user.name", "Score Orchestrator"); err != nil {
		return err
	}
	if err := runGitCommand(tmpDir, "config", "user.email", "orchestrator@score.dev"); err != nil {
		return err
	}

	if spinner != nil {
		spinner.Update("Committing manifests...")
	}

	// Add and commit files
	if err := runGitCommand(tmpDir, "add", "."); err != nil {
		return err
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = tmpDir
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		fmt.Printf("No changes to commit - manifests are up to date\n")
		_ = os.RemoveAll(tmpDir)
		return nil
	}

	commitMessage := step.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Add Kubernetes manifests for %s\n\nGenerated from Score specification", appName)
	}

	if err := runGitCommand(tmpDir, "commit", "-m", commitMessage); err != nil {
		return err
	}

	if spinner != nil {
		spinner.Update("Pushing to repository...")
	}

	// Push changes
	if err := runGitCommand(tmpDir, "push", "origin", gitBranch); err != nil {
		return err
	}

	fmt.Printf("Successfully generated and committed Kubernetes manifests to repository\n")
	fmt.Printf("Repository: %s/%s/%s\n", adminConfig.Gitea.URL, owner, step.RepoName)

	// Clean up temporary directory
	_ = os.RemoveAll(tmpDir)
	return nil
}

// Helper functions

// authenticateArgoCD authenticates with ArgoCD and returns a JWT token
func authenticateArgoCD(argoCDURL, username, password string) (string, error) {
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	loginJSON, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login data: %w", err)
	}

	loginURL := fmt.Sprintf("%s/api/v1/session", argoCDURL)
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(string(loginJSON)))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed, status %d: %s", resp.StatusCode, string(body))
	}

	var authResponse struct {
		Token string `json:"token"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &authResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	return authResponse.Token, nil
}

// waitForArgoCDSync waits for an ArgoCD application to sync and become healthy
func waitForArgoCDSync(appName, argoCDURL, token string, timeoutSeconds int, spinner *Spinner) error {
	client := &http.Client{Timeout: 30 * time.Second}
	statusURL := fmt.Sprintf("%s/api/v1/applications/%s", argoCDURL, appName)

	startTime := time.Now()
	timeout := time.Duration(timeoutSeconds) * time.Second

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Waiting for initial sync (timeout: %v)...", timeout))
	}

	for {
		// Check if we've exceeded the timeout
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for ArgoCD sync after %v", timeout)
		}

		// Get application status
		req, err := http.NewRequest("GET", statusURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create status request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			// Retry on network errors
			time.Sleep(10 * time.Second)
			continue
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			time.Sleep(10 * time.Second)
			continue
		}

		// Parse the response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		var appStatus struct {
			Status struct {
				Sync struct {
					Status string `json:"status"`
				} `json:"sync"`
				Health struct {
					Status string `json:"status"`
				} `json:"health"`
			} `json:"status"`
		}

		if err := json.Unmarshal(body, &appStatus); err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		syncStatus := appStatus.Status.Sync.Status
		healthStatus := appStatus.Status.Health.Status

		if spinner != nil {
			spinner.Update(fmt.Sprintf("Sync Status: %s, Health: %s", syncStatus, healthStatus))
		}

		// Check if sync is complete and healthy
		if syncStatus == "Synced" && healthStatus == "Healthy" {
			if spinner != nil {
				spinner.Update("ArgoCD Application synced successfully (Status: Synced, Health: Healthy)")
			}
			fmt.Printf("✅ ArgoCD Application synced successfully (Status: %s, Health: %s)\n", syncStatus, healthStatus)
			return nil
		}

		// Check for failed states
		if syncStatus == "OutOfSync" && healthStatus == "Degraded" {
			return fmt.Errorf("ArgoCD application failed to sync (Status: %s, Health: %s)", syncStatus, healthStatus)
		}

		// Wait before next check
		time.Sleep(10 * time.Second)
	}
}

// runGitCommand executes a git command in the specified directory
func runGitCommand(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %s: %s", strings.Join(args, " "), string(output))
	}

	return nil
}

func runDummyStepWithSpinner(step types.Step, appName string, envType string, spinner *Spinner) error {
	spinner.Update("Running dummy step...")
	time.Sleep(1 * time.Second)
	spinner.Update("Dummy step processing...")
	time.Sleep(1 * time.Second)
	return nil
}
