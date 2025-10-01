package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"innominatus/internal/admin"
	"innominatus/internal/database"
	"innominatus/internal/demo"
	"innominatus/internal/errors"
	"innominatus/internal/goldenpaths"
	"innominatus/internal/graph"
	"innominatus/internal/types"
	"innominatus/internal/users"
	"innominatus/internal/validation"
	"innominatus/internal/workflow"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

func (c *Client) ListCommand(showDetails bool) error {
	formatter := NewOutputFormatter()
	specs, err := c.ListSpecs()
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		formatter.PrintEmptyState("No applications deployed")
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("Deployed Applications (%d):", len(specs)))

	// Fetch workflows if details are requested
	var allWorkflows []interface{}
	if showDetails {
		formatter.PrintInfo(fmt.Sprintf("%s Fetching workflow data for detailed view...", SymbolSearch))
		workflows, err := c.ListWorkflows("")
		if err != nil {
			formatter.PrintWarning(fmt.Sprintf("Could not fetch workflow data: %v", err))
		} else {
			allWorkflows = workflows
			formatter.PrintSuccess(fmt.Sprintf("Found %d workflow executions", len(allWorkflows)))
		}
	}

	for name, spec := range specs {
		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolApp, fmt.Sprintf("Application: %s", name))

		// Show metadata
		if spec.Metadata != nil {
			if apiVersion, ok := spec.Metadata["APIVersion"].(string); ok {
				formatter.PrintKeyValue(1, "API Version", apiVersion)
			}
		}

		// Show containers
		if len(spec.Containers) > 0 {
			formatter.PrintSection(1, SymbolContainer, fmt.Sprintf("Containers (%d):", len(spec.Containers)))
			for containerName, container := range spec.Containers {
				if containerMap, ok := container.(map[string]interface{}); ok {
					image := "unknown"
					if img, ok := containerMap["Image"].(string); ok {
						image = img
					}
					formatter.PrintItem(2, SymbolBullet, fmt.Sprintf("%s: %s", containerName, image))

					// Show container variables
					if variables, ok := containerMap["Variables"].(map[string]interface{}); ok && len(variables) > 0 {
						fmt.Printf("        Variables:\n")
						for key, value := range variables {
							formatter.PrintKeyValue(3, key, value)
						}
					}
				}
			}
		}

		// Show resources with detailed information
		if len(spec.Resources) > 0 {
			formatter.PrintSection(1, SymbolResource, fmt.Sprintf("Resources (%d):", len(spec.Resources)))
			for resourceName, resource := range spec.Resources {
				if resourceMap, ok := resource.(map[string]interface{}); ok {
					resourceType := "unknown"
					if rType, ok := resourceMap["Type"].(string); ok {
						resourceType = rType
					}
					formatter.PrintItem(2, SymbolBullet, fmt.Sprintf("%s (%s)", resourceName, resourceType))

					// Show resource parameters
					if params, ok := resourceMap["Params"].(map[string]interface{}); ok && len(params) > 0 {
						fmt.Printf("        Parameters:\n")
						for key, value := range params {
							formatter.PrintKeyValue(3, key, value)
						}
					}
				}
			}
		} else {
			formatter.PrintSection(1, SymbolResource, "Resources: None")
		}

		// Show environment information
		if spec.Environment != nil {
			formatter.PrintSection(1, SymbolEnv, "Environment:")
			if envType, ok := spec.Environment["type"].(string); ok {
				formatter.PrintKeyValue(2, "Type", envType)
			}
			if ttl, ok := spec.Environment["ttl"].(string); ok {
				formatter.PrintKeyValue(2, "TTL", ttl)
			}
		}

		// Show dependency graph
		if len(spec.Graph) > 0 {
			formatter.PrintSection(1, SymbolLink, "Dependencies:")
			for container, dependencies := range spec.Graph {
				for _, resource := range dependencies {
					formatter.PrintItem(2, "", fmt.Sprintf("%s %s %s", container, SymbolArrow, resource))
				}
			}
		}

		// Show detailed information if requested
		if showDetails {
			formatter.PrintInfo("   📋 Details enabled - showing additional information:")
			c.showDetailedInfo(name, spec, allWorkflows)
		}

		formatter.PrintDivider(1)
	}

	formatter.PrintCount("application(s) deployed", len(specs))
	return nil
}

func (c *Client) StatusCommand(name string) error {
	spec, err := c.GetSpec(name)
	if err != nil {
		return err
	}

	// Display application info
	if metadata, ok := spec.Metadata["Name"].(string); ok {
		fmt.Printf("Application: %s\n", metadata)
	}

	fmt.Printf("\nResources (%d):\n", len(spec.Resources))
	for resourceName, resource := range spec.Resources {
		if resourceMap, ok := resource.(map[string]interface{}); ok {
			if resourceType, ok := resourceMap["Type"].(string); ok {
				fmt.Printf("  - %s (type: %s)\n", resourceName, resourceType)

				// Show parameters if present
				if params, ok := resourceMap["Params"].(map[string]interface{}); ok && len(params) > 0 {
					for key, value := range params {
						fmt.Printf("    %s: %v\n", key, value)
					}
				}
			}
		}
	}

	// Display environment info
	if spec.Environment != nil {
		fmt.Printf("\nEnvironment:\n")
		if envType, ok := spec.Environment["type"].(string); ok {
			fmt.Printf("  Type: %s\n", envType)
		}
		if ttl, ok := spec.Environment["ttl"].(string); ok {
			fmt.Printf("  TTL: %s\n", ttl)
		}
	}

	// Display dependency graph
	fmt.Printf("\nDependency Graph:\n")
	for container, dependencies := range spec.Graph {
		for _, resource := range dependencies {
			fmt.Printf("  %s -> %s\n", container, resource)
		}
	}

	return nil
}

func (c *Client) ValidateCommand(filename string, explain bool, format string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Use new Score validator for detailed validation
	if explain {
		return c.ValidateWithExplanation(filename, format)
	}

	// Quick validation for backward compatibility
	var spec types.ScoreSpec
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Basic validation
	if spec.Metadata.Name == "" {
		return fmt.Errorf("validation failed: metadata.name is required")
	}

	if len(spec.Containers) == 0 {
		return fmt.Errorf("validation failed: at least one container is required")
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess("Score spec is valid")
	formatter.PrintKeyValue(1, "Application", spec.Metadata.Name)
	formatter.PrintKeyValue(1, "API Version", spec.APIVersion)
	formatter.PrintKeyValue(1, "Containers", len(spec.Containers))
	formatter.PrintKeyValue(1, "Resources", len(spec.Resources))

	if spec.Environment != nil {
		formatter.PrintKeyValue(1, "Environment", fmt.Sprintf("%s (TTL: %s)", spec.Environment.Type, spec.Environment.TTL))
	}

	// Show dependency analysis
	graph := graph.BuildGraph(&spec)
	if len(graph) > 0 {
		formatter.PrintEmpty()
		formatter.PrintSubHeader("Dependencies detected:")
		for container, dependencies := range graph {
			for _, resource := range dependencies {
				formatter.PrintItem(1, "", fmt.Sprintf("%s %s %s", container, SymbolArrow, resource))
			}
		}
	}

	return nil
}

func (c *Client) ValidateWithExplanation(filename string, format string) error {
	validator, err := validation.NewScoreValidator(filename)
	if err != nil {
		return fmt.Errorf("failed to create validator: %w", err)
	}

	validationErrors, err := validator.Validate()
	if err != nil && len(validationErrors) == 0 {
		// Fatal error during validation setup
		return err
	}

	// Format and display results
	formatter := validation.NewExplanationFormatter(validationErrors)

	switch format {
	case "json":
		fmt.Println(formatter.ExportJSON())
	case "simple":
		fmt.Print(formatter.FormatSimple())
	default:
		fmt.Print(formatter.Format())
	}

	// Return error if validation failed
	hasErrors := false
	for _, valErr := range validationErrors {
		if valErr.Severity == errors.SeverityError || valErr.Severity == errors.SeverityFatal {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		return fmt.Errorf("validation failed with %d error(s)", len(validationErrors))
	}

	return nil
}

func (c *Client) EnvironmentsCommand() error {
	formatter := NewOutputFormatter()
	environments, err := c.ListEnvironments()
	if err != nil {
		return err
	}

	if len(environments) == 0 {
		formatter.PrintEmptyState("No active environments")
		return nil
	}

	formatter.PrintHeader("Active Environments:")
	for name, env := range environments {
		formatter.PrintSection(1, SymbolEnv, fmt.Sprintf("%s (%s)", name, env.Type))
		formatter.PrintKeyValue(2, "TTL", env.TTL)
		formatter.PrintKeyValue(2, "Status", env.Status)
		formatter.PrintKeyValue(2, "Created", formatter.FormatTime(env.CreatedAt))
		formatter.PrintKeyValue(2, "Resources", len(env.Resources))
	}

	return nil
}

func (c *Client) DeleteCommand(name string) error {
	formatter := NewOutputFormatter()
	// Complete application deletion (infrastructure + database records)
	err := c.DeleteApplication(name)
	if err != nil {
		return err
	}

	formatter.PrintSuccess(fmt.Sprintf("Successfully deleted application '%s' and all its resources", name))
	return nil
}

func (c *Client) DeprovisionCommand(name string) error {
	formatter := NewOutputFormatter()
	// Infrastructure teardown with audit trail preserved
	err := c.DeprovisionApplication(name)
	if err != nil {
		return err
	}

	formatter.PrintSuccess(fmt.Sprintf("Successfully deprovisioned infrastructure for application '%s'", name))
	formatter.PrintInfo("Application metadata and audit trail preserved in database")
	return nil
}

func (c *Client) AdminCommand(user *users.User, args []string) error {
	if !user.IsAdmin() {
		return fmt.Errorf("access denied: admin role required")
	}

	if len(args) == 0 {
		return fmt.Errorf("admin command requires a subcommand")
	}

	subcommand := args[0]

	switch subcommand {
	case "show":
		config, err := admin.LoadAdminConfig("admin-config.yaml")
		if err != nil {
			return fmt.Errorf("failed to load admin config: %w", err)
		}

		config.PrintConfig()
		return nil

	case "add-user":
		return c.addUserCommand(args[1:])

	case "list-users":
		return c.listUsersCommand()

	case "delete-user":
		if len(args) < 2 {
			return fmt.Errorf("delete-user command requires a username")
		}
		return c.deleteUserCommand(args[1])

	case "generate-api-key":
		return c.generateAPIKeyCommand(user, args[1:])

	case "list-api-keys":
		return c.listAPIKeysCommand(user, args[1:])

	case "revoke-api-key":
		return c.revokeAPIKeyCommand(user, args[1:])

	default:
		return fmt.Errorf("unknown admin subcommand '%s'. Available: show, add-user, list-users, delete-user, generate-api-key, list-api-keys, revoke-api-key", subcommand)
	}
}

func (c *Client) addUserCommand(args []string) error {
	fs := flag.NewFlagSet("add-user", flag.ContinueOnError)
	username := fs.String("username", "", "Username for new user")
	password := fs.String("password", "", "Password for new user")
	team := fs.String("team", "", "Team for new user")
	role := fs.String("role", "user", "Role for new user (user|admin)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" || *password == "" || *team == "" {
		return fmt.Errorf("username, password, and team are required")
	}

	if *role != "user" && *role != "admin" {
		return fmt.Errorf("role must be 'user' or 'admin'")
	}

	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	err = store.AddUser(*username, *password, *team, *role)
	if err != nil {
		return err
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("User '%s' added successfully (%s, %s)", *username, *team, *role))
	return nil
}

func (c *Client) listUsersCommand() error {
	formatter := NewOutputFormatter()
	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	if len(store.Users) == 0 {
		formatter.PrintEmptyState("No users found")
		return nil
	}

	formatter.PrintHeader("Users:")
	for _, user := range store.Users {
		formatter.PrintItem(1, "", fmt.Sprintf("%s (%s, %s)", user.Username, user.Team, user.Role))
	}

	return nil
}

func (c *Client) deleteUserCommand(username string) error {
	formatter := NewOutputFormatter()
	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	err = store.DeleteUser(username)
	if err != nil {
		return err
	}

	formatter.PrintSuccess(fmt.Sprintf("User '%s' deleted successfully", username))
	return nil
}

// ListGoldenPathsCommand lists all available golden paths with metadata
func (c *Client) ListGoldenPathsCommand() error {
	formatter := NewOutputFormatter()
	config, err := goldenpaths.LoadGoldenPaths()
	if err != nil {
		return fmt.Errorf("failed to load golden paths: %w", err)
	}

	paths := config.ListPaths()
	if len(paths) == 0 {
		formatter.PrintEmptyState("No golden paths configured")
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("Available Golden Paths (%d):", len(paths)))

	for _, pathName := range paths {
		metadata, err := config.GetMetadata(pathName)
		if err != nil {
			formatter.PrintWarning(fmt.Sprintf("Failed to load metadata for '%s': %v", pathName, err))
			continue
		}

		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolWorkflow, pathName)

		// Description
		if metadata.Description != "" {
			formatter.PrintKeyValue(1, "Description", metadata.Description)
		}

		// Workflow file
		formatter.PrintKeyValue(1, "Workflow", metadata.WorkflowFile)

		// Category and duration
		if metadata.Category != "" {
			formatter.PrintKeyValue(1, "Category", metadata.Category)
		}
		if metadata.EstimatedDuration != "" {
			formatter.PrintKeyValue(1, "Duration", metadata.EstimatedDuration)
		}

		// Tags
		if len(metadata.Tags) > 0 {
			formatter.PrintKeyValue(1, "Tags", strings.Join(metadata.Tags, ", "))
		}

		// Required parameters
		if len(metadata.RequiredParams) > 0 {
			formatter.PrintSection(1, "", "Required Parameters:")
			for _, param := range metadata.RequiredParams {
				formatter.PrintItem(2, SymbolBullet, param)
			}
		}

		// Optional parameters with defaults
		if len(metadata.OptionalParams) > 0 {
			formatter.PrintSection(1, "", "Optional Parameters:")
			for param, defaultValue := range metadata.OptionalParams {
				formatter.PrintItem(2, SymbolBullet, fmt.Sprintf("%s (default: %s)", param, defaultValue))
			}
		}

		formatter.PrintDivider(0)
	}

	formatter.PrintEmpty()
	formatter.PrintInfo("Run a golden path: ./innominatus-ctl run <path-name> [score-spec.yaml] [--param key=value]")

	return nil
}

// RunGoldenPathCommand executes a golden path workflow with parameter overrides
func (c *Client) RunGoldenPathCommand(pathName string, scoreFile string, params map[string]string) error {
	formatter := NewOutputFormatter()

	// Load golden paths configuration
	config, err := goldenpaths.LoadGoldenPaths()
	if err != nil {
		return fmt.Errorf("failed to load golden paths: %w", err)
	}

	// Get metadata for the golden path
	metadata, err := config.GetMetadata(pathName)
	if err != nil {
		return err
	}

	// Validate that the workflow file exists
	if err := config.ValidatePaths(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Validate required parameters
	if err := config.ValidateParameters(pathName, params); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	// Merge with defaults for optional parameters
	finalParams, err := config.GetParametersWithDefaults(pathName, params)
	if err != nil {
		return fmt.Errorf("failed to merge parameters: %w", err)
	}

	formatter.PrintInfo(fmt.Sprintf("Running golden path '%s' with workflow: %s", pathName, metadata.WorkflowFile))

	// Show active parameters if any
	if len(finalParams) > 0 {
		formatter.PrintSection(0, "", "Active Parameters:")
		for key, value := range finalParams {
			formatter.PrintKeyValue(1, key, value)
		}
	}

	// Load and parse the Score spec if provided
	if scoreFile != "" {
		formatter.PrintInfo(fmt.Sprintf("Using Score spec: %s", scoreFile))
		scoreData, err := os.ReadFile(scoreFile)
		if err != nil {
			return fmt.Errorf("failed to read Score spec %s: %w", scoreFile, err)
		}

		var spec types.ScoreSpec
		err = yaml.Unmarshal(scoreData, &spec)
		if err != nil {
			return fmt.Errorf("failed to parse Score spec: %w", err)
		}

		formatter.PrintSuccess(fmt.Sprintf("Loaded Score spec for application: %s", spec.Metadata.Name))
	}

	// Execute the workflow using the existing RunWorkflow function
	// TODO: Pass finalParams to runWorkflow for parameter substitution in workflow steps
	err = c.runWorkflow(metadata.WorkflowFile, scoreFile)
	if err != nil {
		return fmt.Errorf("failed to execute golden path workflow: %w", err)
	}

	formatter.PrintSuccess(fmt.Sprintf("Golden path '%s' completed successfully", pathName))
	return nil
}

// runWorkflow executes a workflow via the server API with real resource provisioning
func (c *Client) runWorkflow(workflowFile string, scoreFile string) error {
	formatter := NewOutputFormatter()

	// Extract workflow name from file path
	workflowName := filepath.Base(workflowFile)
	workflowName = strings.TrimSuffix(workflowName, ".yaml")
	workflowName = strings.TrimSuffix(workflowName, ".yml")

	formatter.PrintInfo(fmt.Sprintf("Executing golden path workflow: %s", workflowName))

	// Load the Score specification if provided
	var scoreData []byte
	var err error
	if scoreFile != "" {
		scoreData, err = os.ReadFile(scoreFile)
		if err != nil {
			return fmt.Errorf("failed to read Score file: %w", err)
		}
		formatter.PrintSuccess(fmt.Sprintf("Loaded Score specification: %s", scoreFile))
	}

	// Ensure we have authentication
	if c.token == "" {
		return fmt.Errorf("authentication required: please login first with './innominatus-ctl login'")
	}

	// Make API request to server for golden path execution
	url := fmt.Sprintf("%s/api/workflows/golden-paths/%s/execute", c.baseURL, workflowName)

	var req *http.Request
	if scoreData != nil {
		req, err = http.NewRequest("POST", url, bytes.NewBuffer(scoreData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/yaml")
	} else {
		req, err = http.NewRequest("POST", url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("workflow execution failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Display execution results
	if message, ok := response["message"].(string); ok {
		formatter.PrintSuccess(message)
	}

	if appName, ok := response["app_name"].(string); ok {
		formatter.PrintKeyValue(1, "Application", appName)
	}

	if workflowID, ok := response["workflow_id"].(float64); ok {
		formatter.PrintKeyValue(1, "Workflow ID", fmt.Sprintf("%.0f", workflowID))
	}

	if resourcesCreated, ok := response["resources_created"].(float64); ok && resourcesCreated > 0 {
		formatter.PrintKeyValue(1, "Resources created", fmt.Sprintf("%.0f", resourcesCreated))
	}

	if resourcesProvisioned, ok := response["resources_provisioned"].(float64); ok && resourcesProvisioned > 0 {
		formatter.PrintKeyValue(1, "Resources provisioned", fmt.Sprintf("%.0f", resourcesProvisioned))
	}

	formatter.PrintSuccess("Golden path workflow execution completed with resource provisioning")
	return nil
}

// DemoTimeCommand installs/reconciles the demo environment
func (c *Client) DemoTimeCommand() error {
	// Create demo environment configuration
	env := demo.NewDemoEnvironment()

	// Create installer
	installer := demo.NewInstaller(env.KubeContext, false)

	// Create health checker
	healthChecker := demo.NewHealthChecker(30 * time.Second)

	// Create git manager
	gitManager := demo.NewGitManager("gitea.localtest.me", "giteaadmin", "admin", "platform-config")

	// Create Grafana manager
	grafanaManager := demo.NewGrafanaManager("http://grafana.localtest.me", "admin", "admin")

	// Create cheat sheet
	cheatSheet := demo.NewCheatSheet(env)

	// Print welcome message
	cheatSheet.PrintWelcome()

	// Kill any processes using port 8081 to prevent conflicts
	cheatSheet.PrintProgress("Cleaning up port 8081...")
	if err := killProcessesOnPort(8081); err != nil {
		fmt.Printf("Warning: Failed to clean port 8081: %v\n", err)
	}

	// Verify Kubernetes context
	if err := installer.VerifyKubeContext(); err != nil {
		cheatSheet.PrintError("Kubernetes Context Verification", err)
		return err
	}

	// Install components
	cheatSheet.PrintProgress("Installing demo environment components...")

	for _, component := range env.GetHelmComponents() {
		cheatSheet.PrintProgress(fmt.Sprintf("Installing %s...", component.Name))
		if err := installer.InstallComponent(component); err != nil {
			cheatSheet.PrintError(fmt.Sprintf("Installing %s", component.Name), err)
			return err
		}
	}

	// Install Kubernetes Dashboard
	cheatSheet.PrintProgress("Installing Kubernetes Dashboard...")
	if err := installer.InstallKubernetesDashboard(); err != nil {
		cheatSheet.PrintError("Installing Kubernetes Dashboard", err)
		return err
	}

	// Install Demo App
	cheatSheet.PrintProgress("Installing Demo Application...")
	if err := installer.InstallDemoApp(); err != nil {
		cheatSheet.PrintError("Installing Demo Application", err)
		return err
	}

	// Wait for services to be healthy
	cheatSheet.PrintProgress("Waiting for services to become healthy...")
	if err := healthChecker.WaitForHealthy(env, 30, 10*time.Second); err != nil {
		cheatSheet.PrintError("Health Check", err)
		return err
	}

	// Seed Git repository
	cheatSheet.PrintProgress("Seeding Git repository...")
	if err := gitManager.SeedRepository(); err != nil {
		cheatSheet.PrintError("Git Repository Seeding", err)
		return err
	}

	// Install Grafana dashboards
	cheatSheet.PrintProgress("Installing Grafana dashboards...")
	if err := grafanaManager.InstallClusterHealthDashboard(); err != nil {
		cheatSheet.PrintError("Grafana Dashboard Installation", err)
		return err
	}

	// Print installation complete
	cheatSheet.PrintInstallationComplete()

	// Print status and credentials
	healthResults := healthChecker.CheckAll(env)
	cheatSheet.PrintStatus(healthResults)
	cheatSheet.PrintCredentials()
	cheatSheet.PrintQuickStart()
	cheatSheet.PrintCommands()
	cheatSheet.PrintFooter()

	return nil
}

// DemoNukeCommand uninstalls and cleans the demo environment
func (c *Client) DemoNukeCommand() error {
	// Create demo environment configuration
	env := demo.NewDemoEnvironment()

	// Create installer
	installer := demo.NewInstaller(env.KubeContext, false)

	// Create cheat sheet
	cheatSheet := demo.NewCheatSheet(env)

	cheatSheet.PrintProgress("Uninstalling demo environment...")

	// Uninstall components in reverse order
	components := env.GetHelmComponents()
	for i := len(components) - 1; i >= 0; i-- {
		component := components[i]
		cheatSheet.PrintProgress(fmt.Sprintf("Uninstalling %s...", component.Name))
		if err := installer.UninstallComponent(component); err != nil {
			fmt.Printf("Warning: Failed to uninstall %s: %v\n", component.Name, err)
		}
	}

	// Delete namespaces
	namespaces := []string{"demo", "monitoring", "vault", "argocd", "gitea", "minio-system", "ingress-nginx", "kubernetes-dashboard"}
	for _, namespace := range namespaces {
		cheatSheet.PrintProgress(fmt.Sprintf("Deleting namespace %s...", namespace))
		if err := installer.DeleteNamespace(namespace); err != nil {
			fmt.Printf("Warning: Failed to delete namespace %s: %v\n", namespace, err)
		}
	}

	// Clean database using same defaults as server startup
	cheatSheet.PrintProgress("Cleaning database tables...")
	db, err := database.NewDatabase()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not connect to database: %v\n", err)
		fmt.Printf("   Skipping database cleanup. Database may not be running.\n")
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "idp_orchestrator"
		}
		fmt.Printf("   To manually clean: psql -d %s -c \"TRUNCATE TABLE workflow_executions, resource_instances CASCADE;\"\n", dbName)
	} else {
		defer db.Close()
		if err := db.CleanDatabase(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to clean database: %v\n", err)
		} else {
			cheatSheet.PrintProgress("✓ Database tables cleaned")
		}
	}

	// Print nuke complete
	cheatSheet.PrintNukeComplete()

	return nil
}

// DemoStatusCommand checks demo environment health and displays status
func (c *Client) DemoStatusCommand() error {
	// Create demo environment configuration
	env := demo.NewDemoEnvironment()

	// Create health checker
	healthChecker := demo.NewHealthChecker(10 * time.Second)

	// Create cheat sheet
	cheatSheet := demo.NewCheatSheet(env)

	// Check all component health
	healthResults := healthChecker.CheckAll(env)

	// Print compact status
	cheatSheet.PrintCompactStatus(healthResults)

	// Print detailed status
	cheatSheet.PrintStatus(healthResults)

	// Print credentials
	cheatSheet.PrintCredentials()

	// Print quick start guide
	cheatSheet.PrintQuickStart()

	// Print useful commands
	cheatSheet.PrintCommands()

	return nil
}

// killProcessesOnPort kills any processes listening on the specified port
func killProcessesOnPort(port int) error {
	// Use lsof to find processes using the port
	cmd := exec.Command("lsof", "-ti:"+strconv.Itoa(port))
	output, err := cmd.Output()
	if err != nil {
		// If lsof fails, the port might be free
		return nil
	}

	// Parse PIDs from output
	pids := strings.Fields(strings.TrimSpace(string(output)))
	if len(pids) == 0 {
		return nil
	}

	// Kill each process
	for _, pid := range pids {
		killCmd := exec.Command("kill", "-9", pid)
		if err := killCmd.Run(); err != nil {
			fmt.Printf("Warning: Failed to kill process %s: %v\n", pid, err)
		} else {
			fmt.Printf("✅ Killed process %s using port %d\n", pid, port)
		}
	}

	return nil
}

// showDetailedInfo displays detailed information for an application including ArgoCD URLs and workflow links
func (c *Client) showDetailedInfo(name string, spec *SpecResponse, allWorkflows []interface{}) {
	fmt.Printf("\n📋 Detailed Information for %s:\n", name)

	// ArgoCD URLs
	fmt.Printf("  🔗 ArgoCD Application: https://argocd.localhost/applications/%s\n", name)
	fmt.Printf("  🌐 ArgoCD Dashboard: https://argocd.localhost/applications\n")

	// Git Repository (if available in metadata)
	if spec.Metadata != nil {
		if repo, ok := spec.Metadata["repository"].(string); ok && repo != "" {
			fmt.Printf("  📦 Git Repository: %s\n", repo)
		}
	}

	// API Endpoints
	fmt.Printf("  🔧 API Endpoint: %s/api/specs/%s\n", c.baseURL, name)

	// Workflow information
	workflows := c.filterWorkflowsByApp(allWorkflows, name)
	if len(workflows) > 0 {
		fmt.Printf("  ⚙️  Workflow Executions (%d):\n", len(workflows))
		c.displayWorkflowSummary(workflows, name)
	} else {
		fmt.Printf("  ⚙️  No workflow executions found\n")
	}
}

// filterWorkflowsByApp filters workflows that belong to a specific application
func (c *Client) filterWorkflowsByApp(allWorkflows []interface{}, appName string) []interface{} {
	var filtered []interface{}

	for _, workflow := range allWorkflows {
		if wf, ok := workflow.(map[string]interface{}); ok {
			if app, exists := wf["app"]; exists && app == appName {
				filtered = append(filtered, workflow)
			}
		}
	}

	return filtered
}

// ListWorkflowsCommand lists all workflow executions with optional filtering by application
func (c *Client) ListWorkflowsCommand(appName string) error {
	workflows, err := c.ListWorkflows(appName)
	if err != nil {
		return err
	}

	if len(workflows) == 0 {
		if appName != "" {
			fmt.Printf("No workflow executions found for application '%s'\n", appName)
		} else {
			fmt.Println("No workflow executions found")
		}
		return nil
	}

	title := "All Workflow Executions"
	if appName != "" {
		title = fmt.Sprintf("Workflow Executions for '%s'", appName)
	}

	fmt.Printf("%s (%d):\n", title, len(workflows))
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for i, workflow := range workflows {
		if wf, ok := workflow.(map[string]interface{}); ok {
			// Extract workflow information
			workflowID := "unknown"
			if id, exists := wf["id"]; exists {
				workflowID = fmt.Sprintf("%v", id)
			}

			status := "unknown"
			if s, exists := wf["status"]; exists {
				status = fmt.Sprintf("%v", s)
			}

			app := "unknown"
			if a, exists := wf["app"]; exists {
				app = fmt.Sprintf("%v", a)
			}

			execTime := "unknown"
			if t, exists := wf["execution_time"]; exists {
				execTime = fmt.Sprintf("%v", t)
			}

			startTime := "unknown"
			if st, exists := wf["start_time"]; exists {
				startTime = fmt.Sprintf("%v", st)
			}

			workflowType := "unknown"
			if wt, exists := wf["workflow_type"]; exists {
				workflowType = fmt.Sprintf("%v", wt)
			}

			// Determine status emoji
			statusEmoji := "❓"
			switch status {
			case "completed", "success":
				statusEmoji = "✅"
			case "failed", "error":
				statusEmoji = "❌"
			case "running", "in_progress":
				statusEmoji = "🔄"
			case "pending":
				statusEmoji = "⏳"
			}

			fmt.Printf("\n%s Workflow #%d\n", statusEmoji, i+1)
			fmt.Printf("   ID: %s\n", workflowID)
			fmt.Printf("   Status: %s\n", status)
			if appName == "" {
				fmt.Printf("   Application: %s\n", app)
			}
			fmt.Printf("   Type: %s\n", workflowType)
			fmt.Printf("   Started: %s\n", startTime)
			fmt.Printf("   Execution Time: %s\n", execTime)

			// Show steps if available
			if steps, exists := wf["steps"]; exists {
				if stepsList, ok := steps.([]interface{}); ok && len(stepsList) > 0 {
					fmt.Printf("   Steps (%d):\n", len(stepsList))
					for j, step := range stepsList {
						if stepMap, ok := step.(map[string]interface{}); ok {
							stepName := "unnamed"
							if name, exists := stepMap["name"]; exists {
								stepName = fmt.Sprintf("%v", name)
							}
							stepStatus := "unknown"
							if status, exists := stepMap["status"]; exists {
								stepStatus = fmt.Sprintf("%v", status)
							}
							stepEmoji := "❓"
							switch stepStatus {
							case "completed", "success":
								stepEmoji = "✅"
							case "failed", "error":
								stepEmoji = "❌"
							case "running", "in_progress":
								stepEmoji = "🔄"
							case "pending":
								stepEmoji = "⏳"
							}
							fmt.Printf("      %d. %s %s (%s)\n", j+1, stepEmoji, stepName, stepStatus)
						}
					}
				}
			}

			// Show error message if workflow failed
			if errorMsg, exists := wf["error"]; exists && errorMsg != nil {
				fmt.Printf("   Error: %v\n", errorMsg)
			}

			// Show API link
			fmt.Printf("   🔗 API Link: %s/api/workflows/%s\n", c.baseURL, workflowID)

			fmt.Println("   ───────────────────────────────────────────")
		}
	}

	fmt.Printf("\nTotal: %d workflow execution(s)\n", len(workflows))
	return nil
}

// displayWorkflowSummary shows a summary of workflow executions for an application
func (c *Client) displayWorkflowSummary(workflows []interface{}, appName string) {
	for i, workflow := range workflows {
		if wf, ok := workflow.(map[string]interface{}); ok {
			status := "unknown"
			if s, exists := wf["status"]; exists {
				status = fmt.Sprintf("%v", s)
			}

			execTime := "unknown"
			if t, exists := wf["execution_time"]; exists {
				execTime = fmt.Sprintf("%v", t)
			}

			workflowID := "unknown"
			if id, exists := wf["id"]; exists {
				workflowID = fmt.Sprintf("%v", id)
			}

			fmt.Printf("    %d. Status: %s | Time: %s | ID: %s\n", i+1, status, execTime, workflowID)
			fmt.Printf("       🔗 Workflow Link: %s/workflows/%s\n", c.baseURL, workflowID)
		}
	}
}

// generateAPIKeyCommand generates a new API key for a user
func (c *Client) generateAPIKeyCommand(user *users.User, args []string) error {
	fs := flag.NewFlagSet("generate-api-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username to generate API key for")
	keyName := fs.String("name", "", "Name for the API key")
	expiryDays := fs.Int("expiry-days", 0, "Number of days until expiry (required, must be > 0)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Default to current user if not admin
	if *username == "" {
		*username = user.Username
	} else if !user.IsAdmin() && *username != user.Username {
		return fmt.Errorf("access denied: only admins can generate API keys for other users")
	}

	if *keyName == "" {
		return fmt.Errorf("API key name is required")
	}

	if *expiryDays <= 0 {
		return fmt.Errorf("expiry-days is required and must be greater than 0")
	}

	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	apiKey, err := store.GenerateAPIKey(*username, *keyName, *expiryDays)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Generated API key for user '%s'\n", *username)
	fmt.Printf("   Name: %s\n", apiKey.Name)
	fmt.Printf("   Key: %s\n", apiKey.Key)
	fmt.Printf("   Created: %s\n", apiKey.CreatedAt.Format(time.RFC3339))
	fmt.Printf("   Expires: %s\n", apiKey.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("\n💡 Store this API key securely. You can use it with:\n")
	fmt.Printf("   export IDP_API_KEY=%s\n", apiKey.Key)
	fmt.Printf("   ./innominatus-ctl list\n")

	return nil
}

// listAPIKeysCommand lists API keys for a user
func (c *Client) listAPIKeysCommand(user *users.User, args []string) error {
	fs := flag.NewFlagSet("list-api-keys", flag.ContinueOnError)
	username := fs.String("username", "", "Username to list API keys for")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Default to current user if not admin
	if *username == "" {
		*username = user.Username
	} else if !user.IsAdmin() && *username != user.Username {
		return fmt.Errorf("access denied: only admins can list API keys for other users")
	}

	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	apiKeys, err := store.ListAPIKeys(*username)
	if err != nil {
		return err
	}

	if len(apiKeys) == 0 {
		fmt.Printf("No API keys found for user '%s'\n", *username)
		return nil
	}

	fmt.Printf("API Keys for user '%s' (%d):\n", *username, len(apiKeys))
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for i, key := range apiKeys {
		status := "✅ Active"
		if time.Now().After(key.ExpiresAt) {
			status = "❌ Expired"
		}

		fmt.Printf("\n%d. %s (%s)\n", i+1, key.Name, status)
		fmt.Printf("   Key: %s...%s\n", key.Key[:8], key.Key[len(key.Key)-8:])
		fmt.Printf("   Created: %s\n", key.CreatedAt.Format(time.RFC3339))
		fmt.Printf("   Expires: %s\n", key.ExpiresAt.Format(time.RFC3339))
		if !key.LastUsedAt.IsZero() {
			fmt.Printf("   Last Used: %s\n", key.LastUsedAt.Format(time.RFC3339))
		} else {
			fmt.Printf("   Last Used: Never\n")
		}
	}

	return nil
}

// revokeAPIKeyCommand revokes an API key
func (c *Client) revokeAPIKeyCommand(user *users.User, args []string) error {
	fs := flag.NewFlagSet("revoke-api-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username to revoke API key for")
	keyName := fs.String("name", "", "Name of the API key to revoke")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Default to current user if not admin
	if *username == "" {
		*username = user.Username
	} else if !user.IsAdmin() && *username != user.Username {
		return fmt.Errorf("access denied: only admins can revoke API keys for other users")
	}

	if *keyName == "" {
		return fmt.Errorf("API key name is required")
	}

	store, err := users.LoadUsers()
	if err != nil {
		return fmt.Errorf("failed to load users: %w", err)
	}

	err = store.RevokeAPIKey(*username, *keyName)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Revoked API key '%s' for user '%s'\n", *keyName, *username)
	return nil
}

// ListResourcesCommand lists all resource instances with optional filtering by application
func (c *Client) ListResourcesCommand(appName string) error {
	resources, err := c.ListResources(appName)
	if err != nil {
		return err
	}

	if len(resources) == 0 {
		if appName != "" {
			fmt.Printf("No resource instances found for application '%s'\n", appName)
		} else {
			fmt.Println("No resource instances found")
		}
		return nil
	}

	title := "All Resource Instances"
	if appName != "" {
		title = fmt.Sprintf("Resource Instances for '%s'", appName)
	}

	totalCount := 0
	for _, resourceList := range resources {
		totalCount += len(resourceList)
	}

	fmt.Printf("%s (%d):\n", title, totalCount)
	fmt.Println("═══════════════════════════════════════════════════════════════")

	for applicationName, resourceList := range resources {
		if len(resourceList) == 0 {
			continue
		}

		fmt.Printf("\n📦 Application: %s (%d resources)\n", applicationName, len(resourceList))
		fmt.Println("───────────────────────────────────────────")

		for i, resource := range resourceList {
			// Determine status emoji based on state and health
			statusEmoji := "❓"
			switch resource.State {
			case "active":
				if resource.HealthStatus == "healthy" {
					statusEmoji = "✅"
				} else if resource.HealthStatus == "degraded" {
					statusEmoji = "⚠️"
				} else {
					statusEmoji = "❌"
				}
			case "provisioning", "scaling", "updating":
				statusEmoji = "🔄"
			case "requested", "pending":
				statusEmoji = "⏳"
			case "terminating":
				statusEmoji = "🗑️"
			case "terminated":
				statusEmoji = "💀"
			case "failed":
				statusEmoji = "❌"
			}

			fmt.Printf("\n%s Resource #%d\n", statusEmoji, i+1)
			fmt.Printf("   ID: %d\n", resource.ID)
			fmt.Printf("   Name: %s\n", resource.ResourceName)
			fmt.Printf("   Type: %s\n", resource.ResourceType)
			fmt.Printf("   State: %s\n", resource.State)
			fmt.Printf("   Health: %s\n", resource.HealthStatus)
			fmt.Printf("   Created: %s\n", resource.CreatedAt.Format(time.RFC3339))
			fmt.Printf("   Updated: %s\n", resource.UpdatedAt.Format(time.RFC3339))

			// Show last health check if available
			if resource.LastHealthCheck != nil {
				fmt.Printf("   Last Health Check: %s\n", resource.LastHealthCheck.Format(time.RFC3339))
			}

			// Show provider information if available
			if resource.ProviderID != nil {
				fmt.Printf("   Provider ID: %s\n", *resource.ProviderID)
			}

			// Show configuration if present
			if len(resource.Configuration) > 0 {
				fmt.Printf("   Configuration:\n")
				for key, value := range resource.Configuration {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}

			// Show provider metadata if present
			if len(resource.ProviderMetadata) > 0 {
				fmt.Printf("   Provider Metadata:\n")
				for key, value := range resource.ProviderMetadata {
					fmt.Printf("      %s: %v\n", key, value)
				}
			}

			// Show error message if resource is in failed state
			if resource.ErrorMessage != nil && *resource.ErrorMessage != "" {
				fmt.Printf("   Error: %s\n", *resource.ErrorMessage)
			}

			// Show API link
			fmt.Printf("   🔗 API Link: %s/api/resources/%d\n", c.baseURL, resource.ID)
		}

		fmt.Println("   ───────────────────────────────────────────")
	}

	fmt.Printf("\nTotal: %d resource instance(s) across %d application(s)\n", totalCount, len(resources))
	return nil
}

// AnalyzeCommand analyzes a Score specification for workflow dependencies and execution plan
func (c *Client) AnalyzeCommand(filename string) error {
	// Read the Score specification file
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	// Parse the Score spec
	var spec types.ScoreSpec
	err = yaml.Unmarshal(data, &spec)
	if err != nil {
		return fmt.Errorf("failed to parse Score specification: %w", err)
	}

	if spec.Metadata.Name == "" {
		return fmt.Errorf("Score specification must have metadata.name")
	}

	fmt.Printf("🔍 Analyzing workflow for '%s'...\n\n", spec.Metadata.Name)

	// Create local workflow analyzer for offline analysis
	analyzer := workflow.NewWorkflowAnalyzer()
	analysis, err := analyzer.AnalyzeSpec(&spec)
	if err != nil {
		return fmt.Errorf("failed to analyze workflow: %w", err)
	}

	// Display analysis results
	c.displayWorkflowAnalysis(analysis)

	return nil
}

// displayWorkflowAnalysis displays workflow analysis results in a formatted manner
func (c *Client) displayWorkflowAnalysis(analysis *workflow.WorkflowAnalysis) {
	// Display summary
	fmt.Printf("📊 Workflow Analysis Summary\n")
	fmt.Printf("   ═══════════════════════════════════════════\n\n")

	fmt.Printf("   📈 Complexity Score: %d (%s risk)\n",
		analysis.Summary.ComplexityScore, analysis.Summary.RiskLevel)
	fmt.Printf("   ⏱️  Estimated Time: %v\n", analysis.Summary.EstimatedTime)
	fmt.Printf("   📝 Total Steps: %d\n", analysis.Summary.TotalSteps)
	fmt.Printf("   🔗 Total Resources: %d\n", analysis.Summary.TotalResources)
	fmt.Printf("   ⚡ Max Parallel Steps: %d\n", analysis.Summary.ParallelSteps)

	if len(analysis.Summary.CriticalPath) > 0 {
		fmt.Printf("   🎯 Critical Path: %s\n", strings.Join(analysis.Summary.CriticalPath, " → "))
	}

	// Display execution plan
	fmt.Printf("\n🚀 Execution Plan\n")
	fmt.Printf("   ═══════════════════════════════════════════\n\n")

	for i, phase := range analysis.ExecutionPlan.Phases {
		fmt.Printf("   Phase %d: %s (%v)\n", phase.Order, toTitle(phase.Name), phase.EstimatedTime)

		for j, group := range phase.ParallelGroups {
			if len(group.Steps) == 1 {
				step := group.Steps[0]
				fmt.Printf("     └─ %s (%s) - %v\n", step.Name, step.Type, step.EstimatedTime)
			} else {
				fmt.Printf("     └─ Parallel Group %d (%v):\n", j+1, group.EstimatedTime)
				for _, step := range group.Steps {
					fmt.Printf("        ├─ %s (%s) - %v\n", step.Name, step.Type, step.EstimatedTime)
				}
			}
		}

		if i < len(analysis.ExecutionPlan.Phases)-1 {
			fmt.Printf("\n")
		}
	}

	// Display resource graph
	if len(analysis.ResourceGraph.Nodes) > 0 {
		fmt.Printf("\n🔗 Resource Dependencies\n")
		fmt.Printf("   ═══════════════════════════════════════════\n\n")

		// Group nodes by level
		levelGroups := make(map[int][]workflow.ResourceNode)
		for _, node := range analysis.ResourceGraph.Nodes {
			levelGroups[node.Level] = append(levelGroups[node.Level], node)
		}

		maxLevel := 0
		for level := range levelGroups {
			if level > maxLevel {
				maxLevel = level
			}
		}

		for level := 0; level <= maxLevel; level++ {
			if nodes, exists := levelGroups[level]; exists {
				fmt.Printf("   Level %d: ", level)
				nodeNames := make([]string, len(nodes))
				for i, node := range nodes {
					nodeNames[i] = fmt.Sprintf("%s (%s)", node.Name, node.Type)
				}
				fmt.Printf("%s\n", strings.Join(nodeNames, ", "))
			}
		}

		if len(analysis.ResourceGraph.Edges) > 0 {
			fmt.Printf("\n   Dependencies:\n")
			for _, edge := range analysis.ResourceGraph.Edges {
				fromNode := c.findNodeByID(analysis.ResourceGraph.Nodes, edge.From)
				toNode := c.findNodeByID(analysis.ResourceGraph.Nodes, edge.To)
				if fromNode != nil && toNode != nil {
					fmt.Printf("     %s → %s (%s)\n", fromNode.Name, toNode.Name, edge.DependencyType)
				}
			}
		}
	}

	// Display dependency analysis
	if len(analysis.Dependencies) > 0 {
		fmt.Printf("\n📋 Step Dependencies\n")
		fmt.Printf("   ═══════════════════════════════════════════\n\n")

		for _, dep := range analysis.Dependencies {
			fmt.Printf("   🔧 %s (%s) - %v\n", dep.StepName, dep.StepType, dep.EstimatedDuration)

			if len(dep.DependsOn) > 0 {
				fmt.Printf("      ⬅️  Depends on: %s\n", strings.Join(dep.DependsOn, ", "))
			}

			if len(dep.Blocks) > 0 {
				fmt.Printf("      ➡️  Blocks: %s\n", strings.Join(dep.Blocks, ", "))
			}

			if dep.CanRunInParallel {
				fmt.Printf("      ⚡ Can run in parallel\n")
			} else {
				fmt.Printf("      🔒 Must run sequentially\n")
			}

			fmt.Printf("      📅 Phase: %s\n", dep.Phase)
			fmt.Printf("\n")
		}
	}

	// Display warnings
	if len(analysis.Warnings) > 0 {
		fmt.Printf("⚠️  Warnings\n")
		fmt.Printf("   ═══════════════════════════════════════════\n\n")

		for _, warning := range analysis.Warnings {
			fmt.Printf("   ⚠️  %s\n", warning)
		}
		fmt.Printf("\n")
	}

	// Display recommendations
	if len(analysis.Recommendations) > 0 {
		fmt.Printf("💡 Recommendations\n")
		fmt.Printf("   ═══════════════════════════════════════════\n\n")

		for _, rec := range analysis.Recommendations {
			fmt.Printf("   💡 %s\n", rec)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("✅ Analysis complete! Use this information to optimize your deployment workflow.\n")
}

// findNodeByID finds a resource node by its ID
func (c *Client) findNodeByID(nodes []workflow.ResourceNode, id string) *workflow.ResourceNode {
	for _, node := range nodes {
		if node.ID == id {
			return &node
		}
	}
	return nil
}

// LogsCommand displays workflow execution logs with various options
func (c *Client) LogsCommand(workflowID string, options LogsOptions) error {
	// Get detailed workflow execution information
	workflowDetail, err := c.GetWorkflowDetail(workflowID)
	if err != nil {
		return err
	}

	// Display workflow header information
	if !options.StepOnly {
		c.displayWorkflowHeader(workflowDetail)
	}

	// Display step logs based on options
	if options.Step != "" {
		// Show logs for specific step
		return c.displayStepLogs(workflowDetail, options.Step, options)
	} else {
		// Show logs for all steps
		return c.displayAllStepLogs(workflowDetail, options)
	}
}

// LogsOptions contains options for the logs command
type LogsOptions struct {
	Step     string // Specific step name to show logs for
	StepOnly bool   // Only show step logs, skip workflow header
	Tail     int    // Number of lines to show from end of logs (0 = all)
	Follow   bool   // Follow logs in real-time (for running workflows)
	Verbose  bool   // Show additional metadata
}

// displayWorkflowHeader shows workflow execution summary
func (c *Client) displayWorkflowHeader(workflow *WorkflowExecutionDetail) {
	statusEmoji := "❓"
	switch workflow.Status {
	case "completed":
		statusEmoji = "✅"
	case "failed":
		statusEmoji = "❌"
	case "running":
		statusEmoji = "🔄"
	case "pending":
		statusEmoji = "⏳"
	}

	fmt.Printf("%s Workflow Execution #%d\n", statusEmoji, workflow.ID)
	fmt.Printf("═══════════════════════════════════════════════════════════════\n")
	fmt.Printf("Application: %s\n", workflow.ApplicationName)
	fmt.Printf("Workflow: %s\n", workflow.WorkflowName)
	fmt.Printf("Status: %s\n", workflow.Status)
	fmt.Printf("Started: %s\n", workflow.StartedAt.Format(time.RFC3339))

	if workflow.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", workflow.CompletedAt.Format(time.RFC3339))
		duration := workflow.CompletedAt.Sub(workflow.StartedAt)
		fmt.Printf("Duration: %v\n", duration)
	}

	fmt.Printf("Total Steps: %d\n", workflow.TotalSteps)

	if workflow.ErrorMessage != nil && *workflow.ErrorMessage != "" {
		fmt.Printf("Error: %s\n", *workflow.ErrorMessage)
	}

	fmt.Printf("\n")
}

// displayStepLogs shows logs for a specific step
func (c *Client) displayStepLogs(workflow *WorkflowExecutionDetail, stepName string, options LogsOptions) error {
	var targetStep *WorkflowStepDetail
	for _, step := range workflow.Steps {
		if step.StepName == stepName {
			targetStep = &step
			break
		}
	}

	if targetStep == nil {
		return fmt.Errorf("step '%s' not found in workflow. Available steps: %s",
			stepName, c.getAvailableStepNames(workflow.Steps))
	}

	c.displaySingleStepLogs(targetStep, options)
	return nil
}

// displayAllStepLogs shows logs for all steps
func (c *Client) displayAllStepLogs(workflow *WorkflowExecutionDetail, options LogsOptions) error {
	if len(workflow.Steps) == 0 {
		fmt.Println("No steps found in this workflow execution.")
		return nil
	}

	for i, step := range workflow.Steps {
		if i > 0 {
			fmt.Printf("\n")
		}
		c.displaySingleStepLogs(&step, options)
	}

	return nil
}

// displaySingleStepLogs shows logs for a single step
func (c *Client) displaySingleStepLogs(step *WorkflowStepDetail, options LogsOptions) {
	statusEmoji := "❓"
	switch step.Status {
	case "completed":
		statusEmoji = "✅"
	case "failed":
		statusEmoji = "❌"
	case "running":
		statusEmoji = "🔄"
	case "pending":
		statusEmoji = "⏳"
	}

	fmt.Printf("%s Step %d: %s (%s)\n", statusEmoji, step.StepNumber, step.StepName, step.StepType)

	if options.Verbose {
		fmt.Printf("   ID: %d\n", step.ID)
		fmt.Printf("   Status: %s\n", step.Status)
		fmt.Printf("   Started: %s\n", step.StartedAt.Format(time.RFC3339))

		if step.CompletedAt != nil {
			fmt.Printf("   Completed: %s\n", step.CompletedAt.Format(time.RFC3339))
			if step.DurationMs != nil {
				duration := time.Duration(*step.DurationMs) * time.Millisecond
				fmt.Printf("   Duration: %v\n", duration)
			}
		}

		if step.ErrorMessage != nil && *step.ErrorMessage != "" {
			fmt.Printf("   Error: %s\n", *step.ErrorMessage)
		}
	}

	// Display logs
	if step.OutputLogs != nil && *step.OutputLogs != "" {
		fmt.Printf("   Logs:\n")
		logs := *step.OutputLogs

		// Apply tail option if specified
		if options.Tail > 0 {
			lines := strings.Split(logs, "\n")
			if len(lines) > options.Tail {
				lines = lines[len(lines)-options.Tail:]
				logs = strings.Join(lines, "\n")
			}
		}

		// Format and display logs with indentation
		lines := strings.Split(logs, "\n")
		for _, line := range lines {
			if line != "" {
				fmt.Printf("      %s\n", line)
			}
		}
	} else {
		fmt.Printf("   Logs: No output logs available\n")
	}

	fmt.Printf("   ───────────────────────────────────────────\n")
}

// getAvailableStepNames returns a comma-separated list of available step names
func (c *Client) getAvailableStepNames(steps []WorkflowStepDetail) string {
	var names []string
	for _, step := range steps {
		names = append(names, step.StepName)
	}
	return strings.Join(names, ", ")
}

// toTitle converts a string to title case (replacement for deprecated strings.Title)
func toTitle(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
