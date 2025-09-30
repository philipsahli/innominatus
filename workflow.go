package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Global variable to store the current Score spec for workflow access
var currentScoreSpec *ScoreSpec

func RunWorkflow(w Workflow, appName string, envType string) error {
	fmt.Printf("Starting workflow with %d steps for app '%s' (env: %s)\n\n", len(w.Steps), appName, envType)

	for i, step := range w.Steps {
		fmt.Printf("Step %d/%d: %s (%s)\n", i+1, len(w.Steps), step.Name, step.Type)

		// Create and start spinner for this step
		spinner := NewSpinner(fmt.Sprintf("Initializing %s step...", step.Type))
		spinner.Start()

		err := runStepWithSpinner(step, appName, envType, spinner)
		if err != nil {
			spinner.Stop(false, fmt.Sprintf("Step '%s' failed: %v", step.Name, err))
			return fmt.Errorf("workflow failed at step '%s': %w", step.Name, err)
		}

		spinner.Stop(true, fmt.Sprintf("Step '%s' completed successfully", step.Name))
		fmt.Println() // Add spacing between steps
	}

	fmt.Println("🎉 Workflow completed successfully!")
	return nil
}

func runStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	switch step.Type {
	case "terraform":
		return runTerraformStepWithSpinner(step, appName, envType, spinner)
	case "ansible":
		return runAnsibleStepWithSpinner(step, appName, envType, spinner)
	case "kubernetes":
		return runKubernetesStepWithSpinner(step, appName, envType, spinner)
	case "terraform-generate":
		return runTerraformGenerateStepWithSpinner(step, appName, envType, spinner)
	case "git-pr":
		return runGitPRStepWithSpinner(step, appName, envType, spinner)
	case "git-check-pr":
		return runGitCheckPRStepWithSpinner(step, appName, envType, spinner)
	case "tfe-status":
		return runTFEStatusStepWithSpinner(step, appName, envType, spinner)
	case "gitea-repo":
		return runGiteaRepoStepWithSpinner(step, appName, envType, spinner)
	case "argocd-app":
		return runArgoCDAppStepWithSpinner(step, appName, envType, spinner)
	case "git-commit-manifests":
		return runGitCommitManifestsStepWithSpinner(step, appName, envType, spinner)
	default:
		return fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// Legacy function for backward compatibility
func runStep(step Step, appName string, envType string) error {
	return runStepWithSpinner(step, appName, envType, nil)
}

func runTerraformStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	// Create app and environment-specific workspace path
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/%s", appName, envType, step.Path)

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Setting up Terraform workspace: %s", workspacePath))
	}

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory %s: %w", workspacePath, err)
	}

	// Copy terraform files from template path to workspace
	templatePath := step.Path
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return fmt.Errorf("terraform template path does not exist: %s", templatePath)
	}

	if spinner != nil {
		spinner.Update("Copying Terraform configuration files...")
	}

	// Copy terraform files (simple approach - copy *.tf files)
	if err := copyTerraformFiles(templatePath, workspacePath); err != nil {
		return fmt.Errorf("failed to copy terraform files: %w", err)
	}

	if spinner != nil {
		spinner.Update("Running terraform init...")
	}

	// Run terraform init
	initCmd := exec.Command("terraform", "init")
	initCmd.Dir = workspacePath
	// Suppress output when using spinner
	if spinner == nil {
		initCmd.Stdout = os.Stdout
		initCmd.Stderr = os.Stderr
	}

	if err := initCmd.Run(); err != nil {
		return fmt.Errorf("terraform init failed: %w", err)
	}

	if spinner != nil {
		spinner.Update("Applying terraform configuration...")
	}

	// Run terraform apply
	applyCmd := exec.Command("terraform", "apply", "-auto-approve")
	applyCmd.Dir = workspacePath
	// Suppress output when using spinner
	if spinner == nil {
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
	}

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	if spinner != nil {
		spinner.Update("Retrieving terraform outputs...")
	}

	// Get terraform outputs
	outputCmd := exec.Command("terraform", "output", "-json")
	outputCmd.Dir = workspacePath
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

// Legacy function for backward compatibility
func runTerraformStep(step Step, appName string, envType string) error {
	return runTerraformStepWithSpinner(step, appName, envType, nil)
}

func runAnsibleStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.Playbook == "" {
		return fmt.Errorf("ansible step requires playbook field")
	}

	// Create app and environment-specific workspace path for ansible
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/ansible", appName, envType)

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Setting up Ansible workspace: %s", workspacePath))
	}

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create ansible workspace directory %s: %w", workspacePath, err)
	}

	// Check if playbook template exists
	if _, err := os.Stat(step.Playbook); os.IsNotExist(err) {
		return fmt.Errorf("ansible playbook template does not exist: %s", step.Playbook)
	}

	if spinner != nil {
		spinner.Update("Copying Ansible playbook...")
	}

	// Copy playbook to workspace (could be enhanced to template substitution)
	playbookName := fmt.Sprintf("%s-%s-playbook.yml", appName, envType)
	workspacePlaybook := fmt.Sprintf("%s/%s", workspacePath, playbookName)

	if err := copyFile(step.Playbook, workspacePlaybook); err != nil {
		return fmt.Errorf("failed to copy playbook to workspace: %w", err)
	}

	if spinner != nil {
		spinner.Update("Running ansible-playbook...")
	}

	// Run ansible-playbook with workspace-specific playbook
	cmd := exec.Command("ansible-playbook", workspacePlaybook)
	cmd.Dir = workspacePath
	// Suppress output when using spinner
	if spinner == nil {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible-playbook failed: %w", err)
	}

	return nil
}

func runKubernetesStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	// Create app and environment-specific workspace path for kubernetes
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/kubernetes", appName, envType)

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Setting up Kubernetes workspace: %s", workspacePath))
	}

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create kubernetes workspace directory %s: %w", workspacePath, err)
	}

	// Determine namespace - use step.namespace or default to app-env pattern
	namespace := step.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", appName, envType)
	}

	if spinner != nil {
		spinner.Update("Generating Kubernetes manifests...")
	}

	manifestPath := fmt.Sprintf("%s/deployment.yaml", workspacePath)

	if err := generateKubernetesManifests(appName, envType, namespace, manifestPath); err != nil {
		return fmt.Errorf("failed to generate kubernetes manifests: %w", err)
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating namespace: %s", namespace))
	}

	// Create namespace first
	createNsCmd := exec.Command("kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml")
	createNsCmd.Dir = workspacePath

	applyNsCmd := exec.Command("kubectl", "apply", "-f", "-")
	applyNsCmd.Dir = workspacePath
	applyNsCmd.Stdin, _ = createNsCmd.StdoutPipe()
	// Suppress output when using spinner
	if spinner == nil {
		applyNsCmd.Stdout = os.Stdout
		applyNsCmd.Stderr = os.Stderr
	}

	if err := createNsCmd.Start(); err != nil {
		// Namespace creation failure is not critical, continue
	} else {
		applyNsCmd.Run() // Don't fail on namespace creation errors
		createNsCmd.Wait()
	}

	if spinner != nil {
		spinner.Update("Applying Kubernetes manifests...")
	}

	// Apply the generated manifests
	applyCmd := exec.Command("kubectl", "apply", "-f", manifestPath, "-n", namespace)
	applyCmd.Dir = workspacePath
	// Suppress output when using spinner
	if spinner == nil {
		applyCmd.Stdout = os.Stdout
		applyCmd.Stderr = os.Stderr
	}

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	if spinner != nil {
		spinner.Update("Checking deployment status...")
	}

	// Get deployment status
	statusCmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
	statusCmd.Dir = workspacePath
	// Always suppress status check output when using spinner
	if spinner == nil {
		statusCmd.Stdout = os.Stdout
		statusCmd.Stderr = os.Stderr
	}
	statusCmd.Run() // Don't fail on status check errors

	return nil
}

func runAnsibleStep(step Step, appName string, envType string) error {
	if step.Playbook == "" {
		return fmt.Errorf("ansible step requires playbook field")
	}

	// Create app and environment-specific workspace path for ansible
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/ansible", appName, envType)
	fmt.Printf("Running Ansible in workspace: %s\n", workspacePath)

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create ansible workspace directory %s: %w", workspacePath, err)
	}

	// Check if playbook template exists
	if _, err := os.Stat(step.Playbook); os.IsNotExist(err) {
		return fmt.Errorf("ansible playbook template does not exist: %s", step.Playbook)
	}

	// Copy playbook to workspace (could be enhanced to template substitution)
	playbookName := fmt.Sprintf("%s-%s-playbook.yml", appName, envType)
	workspacePlaybook := fmt.Sprintf("%s/%s", workspacePath, playbookName)

	if err := copyFile(step.Playbook, workspacePlaybook); err != nil {
		return fmt.Errorf("failed to copy playbook to workspace: %w", err)
	}

	fmt.Printf("Running: ansible-playbook %s\n", workspacePlaybook)

	// Run ansible-playbook with workspace-specific playbook
	cmd := exec.Command("ansible-playbook", workspacePlaybook)
	cmd.Dir = workspacePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ansible-playbook failed: %w", err)
	}

	return nil
}

// copyTerraformFiles copies all .tf files from source to destination directory
func copyTerraformFiles(sourceDir, destDir string) error {
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.tf files
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".tf") {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		// Ensure destination directory exists
		destDirPath := filepath.Dir(destPath)
		if err := os.MkdirAll(destDirPath, 0755); err != nil {
			return err
		}

		// Copy file
		return copyFile(path, destPath)
	})
}

// copyFile copies a single file from source to destination
func copyFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(destination, sourceInfo.Mode())
}

// runKubernetesStep generates Kubernetes manifests from Score spec and deploys them
func runKubernetesStep(step Step, appName string, envType string) error {
	// Create app and environment-specific workspace path for kubernetes
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/kubernetes", appName, envType)
	fmt.Printf("Running Kubernetes deployment in workspace: %s\n", workspacePath)

	// Create workspace directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create kubernetes workspace directory %s: %w", workspacePath, err)
	}

	// Determine namespace - use step.namespace or default to app-env pattern
	namespace := step.Namespace
	if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", appName, envType)
	}

	fmt.Printf("Deploying to Kubernetes namespace: %s\n", namespace)

	// We need access to the original Score spec to translate it
	// For now, we'll create a simple example - in a real implementation,
	// you'd pass the ScoreSpec to this function
	manifestPath := fmt.Sprintf("%s/deployment.yaml", workspacePath)

	if err := generateKubernetesManifests(appName, envType, namespace, manifestPath); err != nil {
		return fmt.Errorf("failed to generate kubernetes manifests: %w", err)
	}

	// Create namespace first
	fmt.Printf("Creating namespace: %s\n", namespace)
	createNsCmd := exec.Command("kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml")
	createNsCmd.Dir = workspacePath

	applyNsCmd := exec.Command("kubectl", "apply", "-f", "-")
	applyNsCmd.Dir = workspacePath
	applyNsCmd.Stdin, _ = createNsCmd.StdoutPipe()
	applyNsCmd.Stdout = os.Stdout
	applyNsCmd.Stderr = os.Stderr

	if err := createNsCmd.Start(); err != nil {
		fmt.Printf("Warning: could not create namespace command: %v\n", err)
	} else {
		if err := applyNsCmd.Run(); err != nil {
			fmt.Printf("Warning: could not create namespace (may already exist): %v\n", err)
		}
		createNsCmd.Wait()
	}

	// Apply the generated manifests
	fmt.Printf("Applying Kubernetes manifests: %s\n", manifestPath)
	applyCmd := exec.Command("kubectl", "apply", "-f", manifestPath, "-n", namespace)
	applyCmd.Dir = workspacePath
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr

	if err := applyCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply failed: %w", err)
	}

	// Get deployment status
	fmt.Println("Checking deployment status...")
	statusCmd := exec.Command("kubectl", "get", "pods", "-n", namespace)
	statusCmd.Dir = workspacePath
	statusCmd.Stdout = os.Stdout
	statusCmd.Stderr = os.Stderr
	statusCmd.Run() // Don't fail on status check errors

	return nil
}

// generateKubernetesManifests creates Kubernetes deployment manifests from Score spec
func generateKubernetesManifests(appName, envType, namespace, outputPath string) error {
	// In a real implementation, you'd translate the actual Score spec
	// For now, we'll create a basic deployment example

	deployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      appName,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app":         appName,
				"environment": envType,
				"managed-by":  "innominatus",
			},
		},
		"spec": map[string]interface{}{
			"replicas": 1,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": appName,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app":         appName,
						"environment": envType,
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "main",
							"image": "nginx:latest", // Default image - would be from Score spec
							"ports": []map[string]interface{}{
								{
									"containerPort": 80,
								},
							},
							"env": []map[string]interface{}{
								{
									"name":  "APP_NAME",
									"value": appName,
								},
								{
									"name":  "ENVIRONMENT",
									"value": envType,
								},
							},
						},
					},
				},
			},
		},
	}

	service := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      appName + "-service",
			"namespace": namespace,
			"labels": map[string]interface{}{
				"app":         appName,
				"environment": envType,
			},
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": appName,
			},
			"ports": []map[string]interface{}{
				{
					"protocol":   "TCP",
					"port":       80,
					"targetPort": 80,
				},
			},
			"type": "ClusterIP",
		},
	}

	// Write manifests to file
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write deployment
	deploymentYaml, err := yaml.Marshal(deployment)
	if err != nil {
		return err
	}

	if _, err := file.Write(deploymentYaml); err != nil {
		return err
	}

	if _, err := file.WriteString("---\n"); err != nil {
		return err
	}

	// Write service
	serviceYaml, err := yaml.Marshal(service)
	if err != nil {
		return err
	}

	if _, err := file.Write(serviceYaml); err != nil {
		return err
	}

	fmt.Printf("Generated Kubernetes manifests: %s\n", outputPath)
	return nil
}

// Spinner represents a CLI spinner
type Spinner struct {
	message string
	active  bool
	mu      sync.Mutex
	done    chan bool
}

// NewSpinner creates a new spinner with the given message
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

// Start begins the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active {
		return
	}
	s.active = true

	go func() {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s", spinChars[i%len(spinChars)], s.message)
				i++
			}
		}
	}()
}

// Stop ends the spinner and shows the result
func (s *Spinner) Stop(success bool, resultMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}
	s.active = false

	close(s.done)
	time.Sleep(110 * time.Millisecond) // Ensure last spinner update is complete

	if success {
		fmt.Printf("\r✅ %s\n", resultMessage)
	} else {
		fmt.Printf("\r❌ %s\n", resultMessage)
	}
}

// Update changes the spinner message while it's running
func (s *Spinner) Update(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// runTerraformGenerateStepWithSpinner generates Terraform files locally based on resource info
func runTerraformGenerateStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.Resource == "" {
		return fmt.Errorf("terraform-generate step requires resource field")
	}
	if step.OutputDir == "" {
		return fmt.Errorf("terraform-generate step requires outputDir field")
	}

	workspacePath := fmt.Sprintf("./workspaces/%s-%s/%s", appName, envType, step.OutputDir)

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating output directory: %s", workspacePath))
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", workspacePath, err)
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Generating Terraform files for resource: %s", step.Resource))
	}

	// Generate a dummy main.tf file with resource info
	resourceName := fmt.Sprintf("%s_%s", step.Resource, appName)
	terraformContent := fmt.Sprintf(`# Generated Terraform configuration for %s
# App: %s, Environment: %s

resource "%s" "%s" {
  # Resource configuration for %s
  name = "%s-%s-%s"

  tags = {
    app         = "%s"
    environment = "%s"
    managed_by  = "innominatus"
  }
}

output "%s_connection_string" {
  description = "Connection string for %s"
  value       = %s.%s.connection_string
  sensitive   = true
}

output "%s_endpoint" {
  description = "Public endpoint for %s"
  value       = %s.%s.endpoint
}

output "%s_id" {
  description = "Resource ID for %s"
  value       = %s.%s.id
}

output "%s_status" {
  description = "Current status of %s"
  value       = %s.%s.status
}
`, step.Resource, appName, envType, step.Resource, resourceName, step.Resource,
		step.Resource, appName, envType, appName, envType,
		step.Resource, step.Resource, step.Resource, resourceName,
		step.Resource, step.Resource, step.Resource, resourceName,
		step.Resource, step.Resource, step.Resource, resourceName,
		step.Resource, step.Resource, step.Resource, resourceName)

	mainTfPath := filepath.Join(workspacePath, "main.tf")
	if err := ioutil.WriteFile(mainTfPath, []byte(terraformContent), 0644); err != nil {
		return fmt.Errorf("failed to write main.tf: %w", err)
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Generated Terraform files in: %s", workspacePath))
	}

	fmt.Printf("Generated Terraform files for resource '%s' in: %s\n", step.Resource, workspacePath)
	return nil
}

// runGitPRStepWithSpinner clones repo, creates branch, commits files, pushes, and opens PR
func runGitPRStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.Repo == "" {
		return fmt.Errorf("git-pr step requires repo field")
	}
	if step.Branch == "" {
		return fmt.Errorf("git-pr step requires branch field")
	}
	if step.CommitMessage == "" {
		return fmt.Errorf("git-pr step requires commitMessage field")
	}

	workspacePath := fmt.Sprintf("./workspaces/%s-%s/git", appName, envType)
	repoPath := filepath.Join(workspacePath, "repo")

	if spinner != nil {
		spinner.Update("Setting up git workspace...")
	}

	// Create workspace directory
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create git workspace: %w", err)
	}

	// Clean up existing repo if it exists
	os.RemoveAll(repoPath)

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Cloning repository: %s", step.Repo))
	}

	// Clone the repository
	cloneCmd := exec.Command("git", "clone", step.Repo, "repo")
	cloneCmd.Dir = workspacePath
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating branch: %s", step.Branch))
	}

	// Create and checkout new branch
	checkoutCmd := exec.Command("git", "checkout", "-b", step.Branch)
	checkoutCmd.Dir = repoPath
	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w", step.Branch, err)
	}

	if spinner != nil {
		spinner.Update("Copying Terraform files to repository...")
	}

	// Copy generated Terraform files from previous step
	tfSourcePath := fmt.Sprintf("./workspaces/%s-%s/tmp-tf", appName, envType)
	tfDestPath := filepath.Join(repoPath, "terraform", appName, envType)

	if err := os.MkdirAll(tfDestPath, 0755); err != nil {
		return fmt.Errorf("failed to create terraform directory in repo: %w", err)
	}

	// Copy .tf files
	if err := copyTerraformFiles(tfSourcePath, tfDestPath); err != nil {
		// If source doesn't exist, create a placeholder file
		placeholderContent := fmt.Sprintf("# Terraform files for %s-%s\n", appName, envType)
		placeholderPath := filepath.Join(tfDestPath, "main.tf")
		if err := ioutil.WriteFile(placeholderPath, []byte(placeholderContent), 0644); err != nil {
			return fmt.Errorf("failed to create placeholder terraform file: %w", err)
		}
	}

	if spinner != nil {
		spinner.Update("Adding files to git...")
	}

	// Add files to git
	addCmd := exec.Command("git", "add", ".")
	addCmd.Dir = repoPath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to add files to git: %w", err)
	}

	if spinner != nil {
		spinner.Update("Committing changes...")
	}

	// Commit changes
	commitCmd := exec.Command("git", "commit", "-m", step.CommitMessage)
	commitCmd.Dir = repoPath
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Pushing branch: %s", step.Branch))
	}

	// Push branch
	pushCmd := exec.Command("git", "push", "origin", step.Branch)
	pushCmd.Dir = repoPath
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	if spinner != nil {
		spinner.Update("Creating pull request...")
	}

	// Create pull request using GitHub CLI
	prCmd := exec.Command("gh", "pr", "create", "--title", step.CommitMessage, "--body", fmt.Sprintf("Automated PR from Score Orchestrator for %s-%s", appName, envType))
	prCmd.Dir = repoPath
	prOutput, err := prCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	prURL := strings.TrimSpace(string(prOutput))
	fmt.Printf("Created pull request: %s\n", prURL)
	return nil
}

// runGitCheckPRStepWithSpinner polls GitHub API until PR is merged
func runGitCheckPRStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.Repo == "" {
		return fmt.Errorf("git-check-pr step requires repo field")
	}
	if step.Branch == "" {
		return fmt.Errorf("git-check-pr step requires branch field")
	}

	workspacePath := fmt.Sprintf("./workspaces/%s-%s/git/repo", appName, envType)

	if spinner != nil {
		spinner.Update("Checking PR status...")
	}

	// Poll for PR status
	maxAttempts := 120 // 10 minutes with 5-second intervals
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if spinner != nil {
			spinner.Update(fmt.Sprintf("Checking PR status... (attempt %d/%d)", attempt+1, maxAttempts))
		}

		// Check PR status using GitHub CLI
		statusCmd := exec.Command("gh", "pr", "status", "--json", "state")
		statusCmd.Dir = workspacePath
		statusOutput, err := statusCmd.Output()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var prStatus struct {
			CurrentBranch struct {
				State string `json:"state"`
			} `json:"currentBranch"`
		}

		if err := json.Unmarshal(statusOutput, &prStatus); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if prStatus.CurrentBranch.State == "MERGED" {
			if spinner != nil {
				spinner.Update("Pull request has been merged!")
			}
			fmt.Printf("Pull request for branch '%s' has been merged\n", step.Branch)
			return nil
		}

		if prStatus.CurrentBranch.State == "CLOSED" {
			return fmt.Errorf("pull request was closed without merging")
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for PR to be merged")
}

// runTFEStatusStepWithSpinner polls TFE API for workspace run status
func runTFEStatusStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.Workspace == "" {
		return fmt.Errorf("tfe-status step requires workspace field")
	}

	tfeToken := os.Getenv("TFE_TOKEN")
	if tfeToken == "" {
		return fmt.Errorf("TFE_TOKEN environment variable is required")
	}

	tfeOrg := os.Getenv("TFE_ORGANIZATION")
	if tfeOrg == "" {
		return fmt.Errorf("TFE_ORGANIZATION environment variable is required")
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Polling TFE workspace: %s", step.Workspace))
	}

	// Poll TFE workspace for run status
	maxAttempts := 120 // 10 minutes with 5-second intervals
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if spinner != nil {
			spinner.Update(fmt.Sprintf("Checking TFE workspace status... (attempt %d/%d)", attempt+1, maxAttempts))
		}

		// Get latest run for workspace
		url := fmt.Sprintf("https://app.terraform.io/api/v2/workspaces/%s/runs?page[size]=1", step.Workspace)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create TFE request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+tfeToken)
		req.Header.Set("Content-Type", "application/vnd.api+json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		var tfeResp struct {
			Data []struct {
				Attributes struct {
					Status string `json:"status"`
				} `json:"attributes"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &tfeResp); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		if len(tfeResp.Data) == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		status := tfeResp.Data[0].Attributes.Status

		if status == "applied" {
			if spinner != nil {
				spinner.Update("TFE run completed successfully! Fetching outputs...")
			}

			// Get workspace outputs
			outputs, err := getTFEWorkspaceOutputs(step.Workspace, tfeToken)
			if err != nil {
				fmt.Printf("TFE workspace '%s' run completed successfully\n", step.Workspace)
				fmt.Printf("Warning: Could not fetch terraform outputs: %v\n", err)
			} else {
				fmt.Printf("TFE workspace '%s' run completed successfully\n", step.Workspace)
				if len(outputs) > 0 {
					fmt.Println("\nTerraform outputs:")
					for key, output := range outputs {
						if output.Sensitive {
							fmt.Printf("  %s: <sensitive>\n", key)
						} else {
							fmt.Printf("  %s: %v\n", key, output.Value)
						}
					}
				} else {
					fmt.Println("No terraform outputs available")
				}
			}
			return nil
		}

		if status == "errored" || status == "canceled" || status == "discarded" {
			return fmt.Errorf("TFE run failed with status: %s", status)
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for TFE run to complete")
}

// TFEOutput represents a Terraform output from TFE API
type TFEOutput struct {
	Value     interface{} `json:"value"`
	Sensitive bool        `json:"sensitive"`
	Type      string      `json:"type"`
}

// runGiteaRepoStepWithSpinner creates or updates a Git repository in Gitea
func runGiteaRepoStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.RepoName == "" {
		return fmt.Errorf("gitea-repo step requires repoName field")
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration to get Gitea settings
	adminConfig, err := loadAdminConfigForWorkflow("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found in admin-config.yaml")
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating repository: %s", step.RepoName))
	}

	// Use owner from step or default to admin username
	owner := step.Owner
	if owner == "" {
		owner = adminConfig.Gitea.Username
	}

	// Create the repository
	if err := createOrUpdateGiteaRepo(adminConfig, step, owner); err != nil {
		return fmt.Errorf("failed to create Gitea repository: %w", err)
	}

	if spinner != nil {
		spinner.Update("Repository created successfully")
	}

	repoURL := fmt.Sprintf("%s/%s/%s", adminConfig.Gitea.URL, owner, step.RepoName)
	fmt.Printf("Gitea repository available at: %s\n", repoURL)
	return nil
}

// createOrUpdateGiteaRepo creates or updates a repository in Gitea via API
func createOrUpdateGiteaRepo(adminConfig *AdminConfigForWorkflow, step Step, owner string) error {
	// Check if repository already exists
	checkURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", adminConfig.Gitea.URL, owner, step.RepoName)

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create check request: %w", err)
	}

	req.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check repository existence: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("Repository %s/%s already exists, skipping creation\n", owner, step.RepoName)
		return nil
	}

	// Repository doesn't exist, create it
	createURL := fmt.Sprintf("%s/api/v1/user/repos", adminConfig.Gitea.URL)
	if owner != adminConfig.Gitea.Username {
		// Use the specified owner as organization name
		createURL = fmt.Sprintf("%s/api/v1/orgs/%s/repos", adminConfig.Gitea.URL, owner)
	}

	description := step.Description
	if description == "" {
		description = fmt.Sprintf("Repository for %s application", step.RepoName)
	}

	payload := map[string]interface{}{
		"name":        step.RepoName,
		"description": description,
		"private":     step.Private,
		"auto_init":   true,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	createReq, err := http.NewRequest("POST", createURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create repository request: %w", err)
	}

	createReq.SetBasicAuth(adminConfig.Gitea.Username, adminConfig.Gitea.Password)
	createReq.Header.Set("Content-Type", "application/json")

	createResp, err := client.Do(createReq)
	if err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(createResp.Body)
		return fmt.Errorf("failed to create repository, status %d: %s", createResp.StatusCode, string(body))
	}

	fmt.Printf("Successfully created repository: %s/%s\n", owner, step.RepoName)
	return nil
}

// AdminConfigForWorkflow represents admin configuration for workflow operations
type AdminConfigForWorkflow struct {
	Gitea struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		OrgName  string `yaml:"orgName"`
	} `yaml:"gitea"`
	ArgoCD struct {
		URL      string `yaml:"url"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"argocd"`
}

// loadAdminConfigForWorkflow loads admin configuration for workflow operations
func loadAdminConfigForWorkflow(configPath string) (*AdminConfigForWorkflow, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read admin config file: %w", err)
	}

	var config AdminConfigForWorkflow
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse admin config: %w", err)
	}

	return &config, nil
}

// runArgoCDAppStepWithSpinner creates an ArgoCD Application for GitOps deployment
func runArgoCDAppStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.AppName == "" {
		step.AppName = fmt.Sprintf("%s-%s", appName, envType)
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration to get ArgoCD settings
	adminConfig, err := loadAdminConfigForWorkflow("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.ArgoCD.URL == "" {
		return fmt.Errorf("argocd configuration not found in admin-config.yaml")
	}

	if spinner != nil {
		spinner.Update(fmt.Sprintf("Creating ArgoCD Application: %s", step.AppName))
	}

	// Determine repository URL
	repoURL := step.RepoURL
	if repoURL == "" && step.RepoName != "" {
		// Auto-generate repo URL from previous gitea-repo step
		owner := step.Owner
		if owner == "" {
			owner = adminConfig.Gitea.Username
		}
		// Use internal service URL for ArgoCD to access Gitea within Kubernetes
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

	// Create the ArgoCD Application
	if err := createOrUpdateArgoCDApp(adminConfig, step, repoURL, project, targetPath, appName, envType); err != nil {
		return fmt.Errorf("failed to create ArgoCD Application: %w", err)
	}

	if spinner != nil {
		spinner.Update("ArgoCD Application created successfully")
	}

	appURL := fmt.Sprintf("%s/applications/%s", adminConfig.ArgoCD.URL, step.AppName)
	fmt.Printf("ArgoCD Application available at: %s\n", appURL)
	fmt.Printf("Repository: %s\n", repoURL)

	// Check if we should wait for sync completion
	// Default to waiting for argocd-app steps unless explicitly disabled
	waitForSync := true
	if step.WaitForSync != nil && *step.WaitForSync == false {
		// Only disable if explicitly set to false
		waitForSync = false
	}

	if waitForSync {
		timeout := step.Timeout
		if timeout <= 0 {
			timeout = 300 // Default 5 minutes
		}

		// Get a fresh token for monitoring
		token, err := authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
		if err != nil {
			return fmt.Errorf("failed to authenticate with ArgoCD for monitoring: %w", err)
		}

		if err := waitForArgoCDSync(step.AppName, adminConfig.ArgoCD.URL, token, timeout, spinner); err != nil {
			return fmt.Errorf("failed waiting for ArgoCD sync: %w", err)
		}
	}

	return nil
}

// createOrUpdateArgoCDApp creates or updates an ArgoCD Application via API
func createOrUpdateArgoCDApp(adminConfig *AdminConfigForWorkflow, step Step, repoURL, project, targetPath, appName, envType string) error {
	// First, authenticate with ArgoCD to get a JWT token
	token, err := authenticateArgoCD(adminConfig.ArgoCD.URL, adminConfig.ArgoCD.Username, adminConfig.ArgoCD.Password)
	if err != nil {
		return fmt.Errorf("failed to authenticate with ArgoCD: %w", err)
	}

	// Check if application already exists
	checkURL := fmt.Sprintf("%s/api/v1/applications/%s", adminConfig.ArgoCD.URL, step.AppName)

	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create check request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to check application existence: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("ArgoCD Application %s already exists, skipping creation\n", step.AppName)
		return nil
	}

	// Determine namespace - use step.namespace or default to app-env pattern
	namespace := step.Namespace
	/*if namespace == "" {
		namespace = fmt.Sprintf("%s-%s", appName, envType)
	}*/

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

	createResp, err := client.Do(createReq)
	if err != nil {
		return fmt.Errorf("failed to create ArgoCD application: %w", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != 200 && createResp.StatusCode != 201 {
		body, _ := ioutil.ReadAll(createResp.Body)
		return fmt.Errorf("failed to create ArgoCD application, status %d: %s", createResp.StatusCode, string(body))
	}

	fmt.Printf("Successfully created ArgoCD Application: %s\n", step.AppName)
	return nil
}

// runGitCommitManifestsStepWithSpinner generates Kubernetes manifests from Score spec and commits to Git
func runGitCommitManifestsStepWithSpinner(step Step, appName string, envType string, spinner *Spinner) error {
	if step.RepoName == "" {
		return fmt.Errorf("git-commit-manifests step requires repoName field")
	}

	if spinner != nil {
		spinner.Update("Loading admin configuration...")
	}

	// Load admin configuration
	adminConfig, err := loadAdminConfigForWorkflow("admin-config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load admin config: %w", err)
	}

	if adminConfig.Gitea.URL == "" {
		return fmt.Errorf("gitea configuration not found in admin-config.yaml")
	}

	// For now, we'll use a template-based approach
	// TODO: Implement proper Score spec parsing from workflow context
	if spinner != nil {
		spinner.Update("Generating manifests from template...")
	}

	// Set defaults
	manifestPath := step.ManifestPath
	if manifestPath == "" {
		manifestPath = "."
	}

	gitBranch := step.GitBranch
	if gitBranch == "" {
		gitBranch = "main"
	}

	owner := step.Owner
	if owner == "" {
		owner = adminConfig.Gitea.Username
	}

	// Create workspace for Git operations
	workspacePath := fmt.Sprintf("./workspaces/%s-%s/git-manifests", appName, envType)
	repoPath := filepath.Join(workspacePath, "repo")

	if spinner != nil {
		spinner.Update("Setting up Git workspace...")
	}

	// Create workspace directory
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Clean up existing repo if it exists
	os.RemoveAll(repoPath)

	// Clone the repository
	repoURL := fmt.Sprintf("http://%s:%s@%s/%s/%s.git",
		adminConfig.Gitea.Username, adminConfig.Gitea.Password,
		strings.TrimPrefix(adminConfig.Gitea.URL, "http://"), owner, step.RepoName)

	if spinner != nil {
		spinner.Update("Cloning repository...")
	}

	cloneCmd := exec.Command("git", "clone", repoURL, "repo")
	cloneCmd.Dir = workspacePath
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Generate Kubernetes manifests
	if spinner != nil {
		spinner.Update("Generating Kubernetes manifests...")
	}

	manifestContent, err := generateKubernetesManifestsFromScore(nil, appName, envType)
	if err != nil {
		return fmt.Errorf("failed to generate manifests: %w", err)
	}

	// Write manifests to repository
	manifestFilePath := filepath.Join(repoPath, manifestPath, "deployment.yaml")
	if manifestPath != "." {
		manifestDir := filepath.Join(repoPath, manifestPath)
		if err := os.MkdirAll(manifestDir, 0755); err != nil {
			return fmt.Errorf("failed to create manifest directory: %w", err)
		}
	}

	if err := ioutil.WriteFile(manifestFilePath, []byte(manifestContent), 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}

	if spinner != nil {
		spinner.Update("Configuring git...")
	}

	// Configure git
	if err := runGitCommand(repoPath, "config", "user.name", "Score Orchestrator"); err != nil {
		return err
	}
	if err := runGitCommand(repoPath, "config", "user.email", "orchestrator@score.dev"); err != nil {
		return err
	}

	if spinner != nil {
		spinner.Update("Committing manifests...")
	}

	// Add and commit files
	if err := runGitCommand(repoPath, "add", "."); err != nil {
		return err
	}

	// Check if there are changes to commit
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = repoPath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(strings.TrimSpace(string(output))) == 0 {
		fmt.Printf("No changes to commit - manifests are up to date\n")
		return nil
	}

	commitMessage := step.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Add Kubernetes manifests for %s\n\nGenerated from Score specification:\n- Deployment with containers and environment variables\n- Service for load balancing\n- Ingress for external access\n\nManaged by Score Orchestrator", appName)
	}

	if err := runGitCommand(repoPath, "commit", "-m", commitMessage); err != nil {
		return err
	}

	if spinner != nil {
		spinner.Update("Pushing to repository...")
	}

	// Push changes
	if err := runGitCommand(repoPath, "push", "origin", gitBranch); err != nil {
		return err
	}

	if spinner != nil {
		spinner.Update("Manifests committed successfully")
	}

	fmt.Printf("Successfully generated and committed Kubernetes manifests to repository\n")
	fmt.Printf("Repository: %s/%s/%s\n", adminConfig.Gitea.URL, owner, step.RepoName)
	fmt.Printf("Manifest path: %s/deployment.yaml\n", manifestPath)

	return nil
}

// loadScoreSpecForManifests loads the current Score specification from the global context
func loadScoreSpecForManifests() (*ScoreSpec, error) {
	// This will need to be passed from the main workflow context
	// For now, we'll read from a known file - this should be improved to use context
	return nil, fmt.Errorf("Score spec context not yet implemented - needs workflow context")
}

// generateKubernetesManifestsFromScore generates Kubernetes YAML from Score specification
func generateKubernetesManifestsFromScore(scoreSpec *ScoreSpec, appName, envType string) (string, error) {
	// For now, use template approach - TODO: implement proper Score spec parsing

	// Use "development" as default to match ArgoCD namespace pattern
	if envType == "default" {
		envType = "development"
	}
	namespace := fmt.Sprintf("%s-%s", appName, envType)

	// For now, return a template - this will be enhanced with actual Score parsing
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
  - port: 80
    targetPort: 80
    protocol: TCP
    name: http
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s-ingress
  namespace: %s
  labels:
    app: %s
    component: web
  annotations:
    kubernetes.io/ingress.class: nginx
spec:
  rules:
  - host: %s.localtest.me
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: %s-service
            port:
              number: 80`

	return fmt.Sprintf(manifestTemplate,
		appName, namespace, appName, envType, // Deployment metadata
		appName, appName, envType, // Deployment spec
		appName, namespace, appName, appName, // Service
		appName, namespace, appName, // Ingress
		appName, appName, // Ingress spec
	), nil
}

// authenticateArgoCD authenticates with ArgoCD and returns a JWT token
func authenticateArgoCD(argocdURL, username, password string) (string, error) {
	loginURL := fmt.Sprintf("%s/api/v1/session", argocdURL)

	loginPayload := map[string]string{
		"username": username,
		"password": password,
	}

	payloadBytes, err := json.Marshal(loginPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login payload: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("authentication failed, status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp struct {
		Token string `json:"token"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResp.Token == "" {
		return "", fmt.Errorf("no token received from ArgoCD")
	}

	return loginResp.Token, nil
}

// getTFEWorkspaceOutputs fetches outputs from a TFE workspace
func getTFEWorkspaceOutputs(workspace, token string) (map[string]TFEOutput, error) {
	url := fmt.Sprintf("https://app.terraform.io/api/v2/workspaces/%s/current-state-version", workspace)

	// First, get the current state version
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create TFE request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/vnd.api+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get state version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("TFE API returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read state version response: %w", err)
	}

	var stateResp struct {
		Data struct {
			ID            string `json:"id"`
			Relationships struct {
				Outputs struct {
					Links struct {
						Related string `json:"related"`
					} `json:"links"`
				} `json:"outputs"`
			} `json:"relationships"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &stateResp); err != nil {
		return nil, fmt.Errorf("failed to parse state version response: %w", err)
	}

	// Get the outputs using the related link
	outputsURL := stateResp.Data.Relationships.Outputs.Links.Related
	if outputsURL == "" {
		return map[string]TFEOutput{}, nil // No outputs available
	}

	outputsReq, err := http.NewRequest("GET", "https://app.terraform.io"+outputsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create outputs request: %w", err)
	}

	outputsReq.Header.Set("Authorization", "Bearer "+token)
	outputsReq.Header.Set("Content-Type", "application/vnd.api+json")

	outputsResp, err := client.Do(outputsReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get outputs: %w", err)
	}
	defer outputsResp.Body.Close()

	if outputsResp.StatusCode != 200 {
		return nil, fmt.Errorf("outputs API returned status %d", outputsResp.StatusCode)
	}

	outputsBody, err := ioutil.ReadAll(outputsResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read outputs response: %w", err)
	}

	var outputsData struct {
		Data []struct {
			Attributes struct {
				Name      string      `json:"name"`
				Value     interface{} `json:"value"`
				Sensitive bool        `json:"sensitive"`
				Type      string      `json:"type"`
			} `json:"attributes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(outputsBody, &outputsData); err != nil {
		return nil, fmt.Errorf("failed to parse outputs response: %w", err)
	}

	outputs := make(map[string]TFEOutput)
	for _, item := range outputsData.Data {
		outputs[item.Attributes.Name] = TFEOutput{
			Value:     item.Attributes.Value,
			Sensitive: item.Attributes.Sensitive,
			Type:      item.Attributes.Type,
		}
	}

	return outputs, nil
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
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			time.Sleep(10 * time.Second)
			continue
		}

		// Parse the response
		body, err := ioutil.ReadAll(resp.Body)
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
