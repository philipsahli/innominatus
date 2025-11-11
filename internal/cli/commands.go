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
	"innominatus/internal/security"
	"innominatus/internal/types"
	"innominatus/internal/users"
	"innominatus/internal/validation"
	"innominatus/internal/workflow"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	// Validate file path to prevent path traversal
	cleanPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if err := security.ValidateFilePath(cleanPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	data, err := os.ReadFile(cleanPath) // #nosec G304 - path validated above
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

func (c *Client) AdminCommand(args []string) error {
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
		return c.generateAPIKeyCommand(args[1:])

	case "list-api-keys":
		return c.listAPIKeysCommand(args[1:])

	case "revoke-api-key":
		return c.revokeAPIKeyCommand(args[1:])
	case "user-api-keys":
		return c.userAPIKeysCommand(args[1:])
	case "user-generate-key":
		return c.userGenerateKeyCommand(args[1:])
	case "user-revoke-key":
		return c.userRevokeKeyCommand(args[1:])

	default:
		return fmt.Errorf("unknown admin subcommand '%s'. Available: show, add-user, list-users, delete-user, generate-api-key, list-api-keys, revoke-api-key, user-api-keys, user-generate-key, user-revoke-key", subcommand)
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

	// Use API instead of direct file access
	err := c.CreateUser(*username, *password, *team, *role)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("User '%s' added successfully (%s, %s)", *username, *team, *role))
	return nil
}

func (c *Client) listUsersCommand() error {
	formatter := NewOutputFormatter()

	// Use API instead of direct file access
	users, err := c.ListUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) == 0 {
		formatter.PrintEmptyState("No users found")
		return nil
	}

	formatter.PrintHeader("Users:")
	for _, user := range users {
		formatter.PrintItem(1, "", fmt.Sprintf("%s (%s, %s)", user.Username, user.Team, user.Role))
	}

	return nil
}

func (c *Client) deleteUserCommand(username string) error {
	formatter := NewOutputFormatter()

	// Use API instead of direct file access
	err := c.DeleteUser(username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	formatter.PrintSuccess(fmt.Sprintf("User '%s' deleted successfully", username))
	return nil
}

// Admin API key management commands

func (c *Client) userAPIKeysCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("username is required")
	}
	username := args[0]

	keys, err := c.AdminGetAPIKeys(username)
	if err != nil {
		return fmt.Errorf("failed to get API keys: %w", err)
	}

	formatter := NewOutputFormatter()
	if len(keys) == 0 {
		formatter.PrintEmptyState(fmt.Sprintf("No API keys found for user '%s'", username))
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("API Keys for %s:", username))
	for _, key := range keys {
		formatter.PrintEmpty()
		if name, ok := key["name"].(string); ok {
			formatter.PrintSection(1, SymbolWorkflow, name)
		}
		if keyVal, ok := key["key"].(string); ok {
			formatter.PrintKeyValue(2, "Key", keyVal)
		}
		if createdAt, ok := key["created_at"].(string); ok {
			formatter.PrintKeyValue(2, "Created", createdAt)
		}
		if expiresAt, ok := key["expires_at"].(string); ok {
			formatter.PrintKeyValue(2, "Expires", expiresAt)
		}
		if lastUsed, ok := key["last_used_at"].(string); ok {
			formatter.PrintKeyValue(2, "Last Used", lastUsed)
		}
	}

	return nil
}

func (c *Client) userGenerateKeyCommand(args []string) error {
	fs := flag.NewFlagSet("user-generate-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username to generate key for")
	name := fs.String("name", "", "Name for the API key")
	expiryDays := fs.Int("expiry-days", 90, "Number of days until expiry")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" || *name == "" {
		return fmt.Errorf("username and name are required")
	}

	result, err := c.AdminGenerateAPIKey(*username, *name, *expiryDays)
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("API key generated for user '%s'", *username))
	formatter.PrintEmpty()
	if key, ok := result["key"].(string); ok {
		formatter.PrintWarning("IMPORTANT: Save this key now - it won't be shown again!")
		formatter.PrintKeyValue(1, "API Key", key)
	}
	if name, ok := result["name"].(string); ok {
		formatter.PrintKeyValue(1, "Name", name)
	}
	if createdAt, ok := result["created_at"].(string); ok {
		formatter.PrintKeyValue(1, "Created", createdAt)
	}
	if expiresAt, ok := result["expires_at"].(string); ok {
		formatter.PrintKeyValue(1, "Expires", expiresAt)
	}

	return nil
}

func (c *Client) userRevokeKeyCommand(args []string) error {
	fs := flag.NewFlagSet("user-revoke-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username")
	keyName := fs.String("key-name", "", "Name of the API key to revoke")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" || *keyName == "" {
		return fmt.Errorf("username and key-name are required")
	}

	err := c.AdminRevokeAPIKey(*username, *keyName)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("API key '%s' revoked for user '%s'", *keyName, *username))
	return nil
}

// TeamCommand handles team management subcommands
func (c *Client) TeamCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("team command requires a subcommand (list|get|create|delete)")
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		return c.listTeamsCommand()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("team get requires a team ID")
		}
		return c.getTeamCommand(args[1])
	case "create":
		return c.createTeamCommand(args[1:])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("team delete requires a team ID")
		}
		return c.deleteTeamCommand(args[1])
	default:
		return fmt.Errorf("unknown team subcommand: %s", subcommand)
	}
}

// listTeamsCommand lists all teams
func (c *Client) listTeamsCommand() error {
	teams, err := c.ListTeams()
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	formatter := NewOutputFormatter()
	if len(teams) == 0 {
		formatter.PrintEmptyState("No teams found")
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("Teams (%d):", len(teams)))
	for _, team := range teams {
		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolTeam, fmt.Sprintf("Team: %s", team.Name))
		formatter.PrintKeyValue(1, "ID", team.ID)
		if team.Description != "" {
			formatter.PrintKeyValue(1, "Description", team.Description)
		}
		if len(team.Members) > 0 {
			formatter.PrintKeyValue(1, "Members", fmt.Sprintf("%d", len(team.Members)))
		}
	}

	return nil
}

// getTeamCommand gets detailed team information
func (c *Client) getTeamCommand(teamID string) error {
	team, err := c.GetTeam(teamID)
	if err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintHeader(fmt.Sprintf("Team: %s", team.Name))
	formatter.PrintKeyValue(0, "ID", team.ID)
	if team.Description != "" {
		formatter.PrintKeyValue(0, "Description", team.Description)
	}

	if len(team.Members) > 0 {
		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolUser, fmt.Sprintf("Members (%d):", len(team.Members)))
		for _, member := range team.Members {
			formatter.PrintItem(1, SymbolBullet, member)
		}
	}

	return nil
}

// createTeamCommand creates a new team
func (c *Client) createTeamCommand(args []string) error {
	createFlags := flag.NewFlagSet("team create", flag.ContinueOnError)
	name := createFlags.String("name", "", "Team name")
	description := createFlags.String("description", "", "Team description")

	if err := createFlags.Parse(args); err != nil {
		return err
	}

	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	if err := c.CreateTeam(*name, *description); err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("Team '%s' created successfully", *name))
	return nil
}

// deleteTeamCommand deletes a team
func (c *Client) deleteTeamCommand(teamID string) error {
	if err := c.DeleteTeam(teamID); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	formatter := NewOutputFormatter()
	formatter.PrintSuccess(fmt.Sprintf("Team '%s' deleted successfully", teamID))
	return nil
}

// ListGoldenPathsCommand lists all available golden paths with metadata from the server
func (c *Client) ListGoldenPathsCommand() error {
	formatter := NewOutputFormatter()

	// Fetch golden paths from the server API
	paths, err := c.GetGoldenPaths()
	if err != nil {
		return fmt.Errorf("failed to load golden paths: %w", err)
	}

	if len(paths) == 0 {
		formatter.PrintEmptyState("No golden paths configured")
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("Available Golden Paths (%d):", len(paths)))

	// Sort path names for consistent output
	pathNames := make([]string, 0, len(paths))
	for pathName := range paths {
		pathNames = append(pathNames, pathName)
	}
	sort.Strings(pathNames)

	for _, pathName := range pathNames {
		metadata := paths[pathName]

		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolWorkflow, pathName)

		// Description
		if metadata.Description != "" {
			formatter.PrintKeyValue(1, "Description", metadata.Description)
		}

		// Workflow file
		if metadata.WorkflowFile != "" {
			formatter.PrintKeyValue(1, "Workflow", metadata.WorkflowFile)
		}

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

		// Required parameters (backward compatibility)
		if len(metadata.RequiredParams) > 0 {
			formatter.PrintSection(1, "", "Required Parameters:")
			for _, param := range metadata.RequiredParams {
				formatter.PrintItem(2, SymbolBullet, param)
			}
		}

		// Optional parameters with defaults (backward compatibility)
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
		// Check if it's a parameter validation error for better messaging
		if paramErr, ok := err.(*goldenpaths.ParameterValidationError); ok {
			formatter.PrintError(fmt.Sprintf("Parameter validation failed for '%s'", pathName))
			formatter.PrintKeyValue(1, "Parameter", paramErr.ParameterName)
			if paramErr.ProvidedValue != "" {
				formatter.PrintKeyValue(1, "Provided Value", paramErr.ProvidedValue)
			}
			if paramErr.ExpectedType != "" {
				formatter.PrintKeyValue(1, "Expected Type", paramErr.ExpectedType)
			}
			if paramErr.Constraint != "" {
				formatter.PrintKeyValue(1, "Constraint", paramErr.Constraint)
			}
			if paramErr.Suggestion != "" {
				formatter.PrintKeyValue(1, "Suggestion", paramErr.Suggestion)
			}
			return fmt.Errorf("parameter validation failed")
		}
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
		// Validate file path
		cleanPath, err := filepath.Abs(scoreFile)
		if err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}
		if err := security.ValidateFilePath(cleanPath); err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}

		scoreData, err := os.ReadFile(cleanPath) // #nosec G304 - path validated above
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

	// Execute the workflow using the existing RunWorkflow function with golden path parameters
	err = c.runWorkflow(metadata.WorkflowFile, scoreFile, finalParams)
	if err != nil {
		return fmt.Errorf("failed to execute golden path workflow: %w", err)
	}

	formatter.PrintSuccess(fmt.Sprintf("Golden path '%s' completed successfully", pathName))
	return nil
}

// runWorkflow executes a workflow via the server API with real resource provisioning
func (c *Client) runWorkflow(workflowFile string, scoreFile string, parameters map[string]string) error {
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
		// Validate file path
		cleanPath, err := filepath.Abs(scoreFile)
		if err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}
		if err := security.ValidateFilePath(cleanPath); err != nil {
			return fmt.Errorf("invalid file path: %w", err)
		}

		scoreData, err = os.ReadFile(cleanPath) // #nosec G304 - path validated above
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

	// Add golden path parameters as query parameters
	if len(parameters) > 0 {
		queryParams := make([]string, 0, len(parameters))
		for key, value := range parameters {
			queryParams = append(queryParams, fmt.Sprintf("param.%s=%s", key, value))
		}
		url = url + "?" + strings.Join(queryParams, "&")
	}

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

	// Retry logic with exponential backoff for transient failures
	maxRetries := 3
	var resp *http.Response
	var body []byte
	client := &http.Client{Timeout: 30 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * time.Second
			formatter.PrintInfo(fmt.Sprintf("Retrying request (attempt %d/%d) after %v...", attempt+1, maxRetries+1, backoff))
			time.Sleep(backoff)

			// Recreate URL with parameters for retry
			retryURL := fmt.Sprintf("%s/api/workflows/golden-paths/%s/execute", c.baseURL, workflowName)
			if len(parameters) > 0 {
				queryParams := make([]string, 0, len(parameters))
				for key, value := range parameters {
					queryParams = append(queryParams, fmt.Sprintf("param.%s=%s", key, value))
				}
				retryURL = retryURL + "?" + strings.Join(queryParams, "&")
			}

			// Recreate request body for retry
			if scoreData != nil {
				req, err = http.NewRequest("POST", retryURL, bytes.NewBuffer(scoreData))
				if err != nil {
					return fmt.Errorf("failed to create retry request: %w", err)
				}
				req.Header.Set("Content-Type", "application/yaml")
			} else {
				req, err = http.NewRequest("POST", retryURL, nil)
				if err != nil {
					return fmt.Errorf("failed to create retry request: %w", err)
				}
			}
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err = client.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to execute workflow after %d retries: %w", maxRetries+1, err)
			}
			formatter.PrintWarning(fmt.Sprintf("Request failed: %v", err))
			continue
		}

		body, err = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			if attempt == maxRetries {
				return fmt.Errorf("failed to read response after %d retries: %w", maxRetries+1, err)
			}
			formatter.PrintWarning(fmt.Sprintf("Failed to read response: %v", err))
			continue
		}

		// Check for transient errors (5xx) or JSON parsing issues
		if resp.StatusCode >= 500 {
			if attempt == maxRetries {
				return fmt.Errorf("workflow execution failed (status %d) after %d retries: %s", resp.StatusCode, maxRetries+1, string(body))
			}
			formatter.PrintWarning(fmt.Sprintf("Server error (status %d), will retry", resp.StatusCode))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("workflow execution failed (status %d): %s", resp.StatusCode, string(body))
		}

		// Success - break out of retry loop
		break
	}

	// Parse response
	var response map[string]interface{}

	// Check if response is HTML (likely an error page)
	if len(body) > 0 && body[0] == '<' {
		truncated := string(body)
		if len(truncated) > 500 {
			truncated = truncated[:500] + "..."
		}
		return fmt.Errorf("server returned HTML instead of JSON (possible gateway/server error):\n%s", truncated)
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		preview := string(body)
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		return fmt.Errorf("failed to parse response: %w\nReceived: %s", err, preview)
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
func (c *Client) DemoTimeCommand(componentFilter string) error {
	// Parse component filter
	var filter []string
	if componentFilter != "" {
		// Split by comma and trim whitespace
		parts := strings.Split(componentFilter, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				filter = append(filter, trimmed)
			}
		}
	}

	// Create demo environment configuration
	env := demo.NewDemoEnvironment()

	// Create installer
	installer := demo.NewInstaller(env.KubeContext, false)

	// Create health checker
	healthChecker := demo.NewHealthChecker(30 * time.Second)

	// Create git manager
	gitManager := demo.NewGitManager("gitea.localtest.me", "giteaadmin", "admin123", "platform-config")

	// Create Grafana manager
	grafanaManager := demo.NewGrafanaManager("http://grafana.localtest.me", "admin", "admin")

	// Create cheat sheet
	cheatSheet := demo.NewCheatSheet(env)

	// Print welcome message
	cheatSheet.PrintWelcome()

	// Print filter information if components are filtered
	if len(filter) > 0 {
		fmt.Printf("\n🎯 Component filter active: %s\n", strings.Join(filter, ", "))
		fmt.Printf("   (Dependencies will be automatically included)\n\n")
	}

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

	// Install components (filtered or all)
	componentsToInstall := env.GetFilteredComponents(filter)
	cheatSheet.PrintProgress(fmt.Sprintf("Installing %d demo environment components...", len(componentsToInstall)))

	for _, component := range componentsToInstall {
		cheatSheet.PrintProgress(fmt.Sprintf("Installing %s...", component.Name))
		if err := installer.InstallComponent(component); err != nil {
			cheatSheet.PrintError(fmt.Sprintf("Installing %s", component.Name), err)
			return err
		}
	}

	// Build list of all installed components for health checking
	allInstalledComponents := componentsToInstall

	// Install Kubernetes Dashboard (if requested)
	if env.IsComponentRequested("kubernetes-dashboard", filter) {
		cheatSheet.PrintProgress("Installing Kubernetes Dashboard...")
		if err := installer.InstallKubernetesDashboard(); err != nil {
			cheatSheet.PrintError("Installing Kubernetes Dashboard", err)
			return err
		}
		// Add to health check list
		for _, comp := range env.Components {
			if comp.Name == "kubernetes-dashboard" {
				allInstalledComponents = append(allInstalledComponents, comp)
				break
			}
		}
	}

	// Install Demo App (if requested)
	if env.IsComponentRequested("demo-app", filter) {
		cheatSheet.PrintProgress("Installing Demo Application...")
		if err := installer.InstallDemoApp(); err != nil {
			cheatSheet.PrintError("Installing Demo Application", err)
			return err
		}
		// Add to health check list
		for _, comp := range env.Components {
			if comp.Name == "demo-app" {
				allInstalledComponents = append(allInstalledComponents, comp)
				break
			}
		}
	}

	// Install PostgreSQL Operator (Zalando)
	cheatSheet.PrintProgress("Installing PostgreSQL Operator (Zalando)...")
	if err := demo.InstallPostgresOperator(env.KubeContext); err != nil {
		cheatSheet.PrintError("Installing PostgreSQL Operator", err)
		return err
	}

	// Wait for services to be healthy (only check installed components)
	cheatSheet.PrintProgress("Waiting for services to become healthy...")
	if err := healthChecker.WaitForComponentsHealthy(allInstalledComponents, 30, 10*time.Second); err != nil {
		cheatSheet.PrintError("Health Check", err)
		return err
	}

	// Configure Keycloak realm and ArgoCD OIDC (if Keycloak and ArgoCD are installed)
	if env.IsComponentRequested("keycloak", filter) && env.IsComponentRequested("argocd", filter) {
		cheatSheet.PrintProgress("Configuring Keycloak realm and ArgoCD OIDC...")
		if err := installer.ApplyKeycloakConfig(); err != nil {
			cheatSheet.PrintError("Keycloak Configuration", err)
			return err
		}

		// Restart ArgoCD server to apply OIDC configuration
		cheatSheet.PrintProgress("Restarting ArgoCD server...")
		if err := installer.RestartArgoCDServer(); err != nil {
			cheatSheet.PrintError("ArgoCD Restart", err)
			return err
		}
	}

	// Seed Git repository (if Gitea is installed)
	if env.IsComponentRequested("gitea", filter) {
		cheatSheet.PrintProgress("Seeding Git repository...")
		if err := gitManager.SeedRepository(); err != nil {
			cheatSheet.PrintError("Git Repository Seeding", err)
			return err
		}
	}

	// Create admin configuration file with provider settings
	cheatSheet.PrintProgress("Creating admin configuration...")
	if err := demo.CreateAdminConfig("admin-config.yaml"); err != nil {
		cheatSheet.PrintError("Admin Config Creation", err)
		return err
	}

	// Install Grafana dashboards (if Grafana is installed)
	if env.IsComponentRequested("grafana", filter) {
		cheatSheet.PrintProgress("Installing Grafana dashboards...")
		if err := grafanaManager.InstallClusterHealthDashboard(); err != nil {
			cheatSheet.PrintError("Grafana Dashboard Installation", err)
			return err
		}

		// Install Innominatus Platform Metrics Dashboard
		if err := grafanaManager.InstallInnominatusDashboard(); err != nil {
			cheatSheet.PrintError("Innominatus Dashboard Installation", err)
			return err
		}
	}

	// Print installation complete
	cheatSheet.PrintInstallationComplete()

	// Print status and credentials (only for installed components)
	healthResults := healthChecker.CheckComponents(allInstalledComponents)
	cheatSheet.PrintStatus(healthResults)
	cheatSheet.PrintCredentials()
	cheatSheet.PrintQuickStart()
	cheatSheet.PrintProviderInfo()
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

	// Uninstall PostgreSQL Operator
	cheatSheet.PrintProgress("Uninstalling PostgreSQL Operator...")
	if err := demo.UninstallPostgresOperator(env.KubeContext); err != nil {
		fmt.Printf("Warning: Failed to uninstall PostgreSQL Operator: %v\n", err)
	}

	// Remove ArgoCD OIDC configuration before deleting namespaces
	cheatSheet.PrintProgress("Removing ArgoCD OIDC configuration...")
	// #nosec G204 - kubectl context from controlled demo environment
	removeOIDCCmd := exec.Command("kubectl", "--context", env.KubeContext, "patch", "configmap", "argocd-cm",
		"-n", "argocd",
		"--type", "json",
		"-p", `[{"op": "remove", "path": "/data/oidc.config"}]`)
	if err := removeOIDCCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to remove OIDC config: %v\n", err)
	}

	// #nosec G204 - kubectl context from controlled demo environment
	removeSecretCmd := exec.Command("kubectl", "--context", env.KubeContext, "delete", "secret", "argocd-oidc-secret",
		"-n", "argocd",
		"--ignore-not-found=true")
	if err := removeSecretCmd.Run(); err != nil {
		fmt.Printf("Warning: Failed to remove OIDC secret: %v\n", err)
	}

	// Delete namespaces
	namespaces := []string{"demo", "monitoring", "vault", "argocd", "gitea", "minio-system", "keycloak", "ingress-nginx", "kubernetes-dashboard"}
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
		defer func() {
			if err := db.Close(); err != nil {
				fmt.Printf("⚠️  Warning: Failed to close database: %v\n", err)
			}
		}()
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

	// Check PostgreSQL Operator status
	fmt.Println("\n🐘 PostgreSQL Operator:")
	if err := demo.CheckPostgresOperatorStatus(env.KubeContext); err != nil {
		fmt.Printf("❌ PostgreSQL Operator: %v\n", err)
	} else {
		fmt.Println("✅ PostgreSQL Operator: Running")
	}

	// Print credentials
	cheatSheet.PrintCredentials()

	// Print quick start guide
	cheatSheet.PrintQuickStart()

	// Print useful commands
	cheatSheet.PrintCommands()

	return nil
}

// DemoResetCommand resets the database to a clean state
func (c *Client) DemoResetCommand(noCheck bool) error {
	// Create demo environment and reset handler
	env := demo.NewDemoEnvironment()
	resetHandler := demo.NewDemoReset(env.KubeContext)

	fmt.Println("🔄 Demo Database Reset")
	fmt.Println("")

	// Check if demo-time has been run (unless --no-check is specified)
	if !noCheck {
		fmt.Println("Checking if demo environment is installed...")
		installed, err := resetHandler.CheckDemoInstalled()
		if err != nil {
			return fmt.Errorf("failed to check demo installation: %w", err)
		}

		if !installed {
			fmt.Println("❌ Demo environment not detected.")
			fmt.Println("   Run 'demo-time' first to install the demo environment.")
			fmt.Println("   Or use --no-check to skip this verification.")
			return fmt.Errorf("demo environment not installed")
		}

		fmt.Println("✅ Demo environment detected")
		fmt.Println("")
	} else {
		fmt.Println("⚠️  Skipping demo environment check (--no-check)")
		fmt.Println("")
	}

	// Display warning
	fmt.Println("⚠️  WARNING: This will DELETE ALL DATA from the database!")
	fmt.Println("   This includes:")
	fmt.Println("   - All workflow executions and step logs")
	fmt.Println("   - All applications and resources")
	fmt.Println("   - All graph data and annotations")
	fmt.Println("   - All sessions and API keys")
	fmt.Println("   - All queue tasks")
	fmt.Println("")
	fmt.Println("   The database will be completely empty (except schema).")
	fmt.Println("")

	// Require explicit confirmation
	fmt.Print("Type 'yes' to confirm: ")
	var confirmation string
	_, _ = fmt.Scanln(&confirmation) // nolint:errcheck

	if confirmation != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	fmt.Println("")
	fmt.Println("🔄 Resetting database...")

	// Connect to database
	db, err := database.NewDatabase()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() { _ = db.Close() }() // nolint:errcheck

	// Truncate all tables
	tableCount, err := db.TruncateAllTables()
	if err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}

	fmt.Println("")
	fmt.Println("✅ Database reset complete!")
	fmt.Println("")
	fmt.Printf("   📊 Statistics:\n")
	fmt.Printf("      • Tables truncated: %d\n", tableCount)
	fmt.Println("")
	fmt.Println("   The database is now empty and ready for fresh data.")
	fmt.Println("   Visit http://localhost:8081 to verify.")
	fmt.Println("")

	return nil
}

// FixGiteaOAuthCommand fixes Gitea OAuth2 configuration to enable auto-registration
func (c *Client) FixGiteaOAuthCommand() error {
	fmt.Println("🔧 Fixing Gitea OAuth2 configuration for auto-registration...")
	fmt.Println("")

	namespace := "gitea"
	keycloakURL := "http://keycloak.localtest.me"
	keycloakRealm := "demo-realm"
	oauthName := "Keycloak"

	// Get Gitea pod name
	fmt.Println("📦 Finding Gitea pod...")
	getPodCmd := exec.Command("kubectl", "get", "pods", "-n", namespace, "-l", "app.kubernetes.io/name=gitea", "-o", "jsonpath={.items[0].metadata.name}") // #nosec G204 - kubectl command with controlled namespace
	podNameBytes, err := getPodCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Gitea pod: %v. Make sure demo environment is running (./innominatus-ctl demo-time)", err)
	}

	podName := strings.TrimSpace(string(podNameBytes))
	if podName == "" {
		return fmt.Errorf("no Gitea pod found in namespace %s. Make sure demo environment is running", namespace)
	}

	fmt.Printf("   Found pod: %s\n\n", podName)

	// List existing OAuth2 sources
	fmt.Println("🔍 Checking existing OAuth2 sources...")
	listCmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "gitea", "admin", "auth", "list") // #nosec G204 - kubectl exec with controlled pod name
	listCmd.Stdout = os.Stdout
	listCmd.Stderr = os.Stderr
	_ = listCmd.Run()
	fmt.Println("")

	// Try to delete existing OAuth2 source (ignore errors if it doesn't exist)
	fmt.Println("🗑️  Removing existing OAuth2 source (if any)...")
	deleteCmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "gitea", "admin", "auth", "delete", "--id", "1") // #nosec G204 - kubectl exec with controlled pod name
	_ = deleteCmd.Run()
	fmt.Println("")

	// Add OAuth2 source
	// Note: Auto-registration is controlled by app.ini [oauth2] ENABLE_AUTO_REGISTRATION = true, not by CLI flag
	fmt.Println("➕ Adding OAuth2 source...")
	addCmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", // #nosec G204 - kubectl exec with controlled parameters
		"gitea", "admin", "auth", "add-oauth",
		"--name", oauthName,
		"--provider", "openidConnect",
		"--key", "gitea",
		"--secret", "gitea-client-secret",
		"--auto-discover-url", fmt.Sprintf("%s/realms/%s/.well-known/openid-configuration", keycloakURL, keycloakRealm),
		"--skip-local-2fa",
		"--scopes", "openid", "email", "profile")

	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	err = addCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to add OAuth2 source: %v", err)
	}

	fmt.Println("   ✅ OAuth2 source added successfully with auto-registration!")
	fmt.Println("")

	// Verify the OAuth2 source was added
	fmt.Println("✅ Verifying OAuth2 sources...")
	verifyCmd := exec.Command("kubectl", "exec", "-n", namespace, podName, "--", "gitea", "admin", "auth", "list") // #nosec G204 - kubectl exec with controlled pod name
	verifyCmd.Stdout = os.Stdout
	verifyCmd.Stderr = os.Stderr
	_ = verifyCmd.Run()
	fmt.Println("")

	fmt.Println("🎉 OAuth2 configuration fix completed!")
	fmt.Println("")
	fmt.Println("📝 Next steps:")
	fmt.Println("   1. Go to http://gitea.localtest.me")
	fmt.Println("   2. Click 'Sign In'")
	fmt.Println("   3. Click 'Sign in with OAuth' and select 'Keycloak'")
	fmt.Println("   4. Login with Keycloak credentials:")
	fmt.Println("      - Username: demo-user")
	fmt.Println("      - Password: password123")
	fmt.Println("   5. Your account should be automatically created in Gitea!")
	fmt.Println("")

	return nil
}

// killProcessesOnPort kills any processes listening on the specified port
func killProcessesOnPort(port int) error {
	// Use lsof to find processes using the port
	cmd := exec.Command("lsof", "-ti:"+strconv.Itoa(port)) // #nosec G204 - port is an integer, safe from injection
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
		killCmd := exec.Command("kill", "-9", pid) // #nosec G204 - PID from lsof output, validated by strings.Fields
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
func (c *Client) generateAPIKeyCommand(args []string) error {
	fs := flag.NewFlagSet("generate-api-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username to generate API key for (required)")
	keyName := fs.String("name", "", "Name for the API key")
	expiryDays := fs.Int("expiry-days", 0, "Number of days until expiry (required, must be > 0)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" {
		return fmt.Errorf("--username is required")
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
func (c *Client) listAPIKeysCommand(args []string) error {
	fs := flag.NewFlagSet("list-api-keys", flag.ContinueOnError)
	username := fs.String("username", "", "Username to list API keys for (required)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" {
		return fmt.Errorf("--username is required")
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
func (c *Client) revokeAPIKeyCommand(args []string) error {
	fs := flag.NewFlagSet("revoke-api-key", flag.ContinueOnError)
	username := fs.String("username", "", "Username to revoke API key for (required)")
	keyName := fs.String("name", "", "Name of the API key to revoke")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *username == "" {
		return fmt.Errorf("--username is required")
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

// LoginCommand authenticates the user, generates an API key, and stores it locally
func (c *Client) LoginCommand(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	keyName := fs.String("name", "", "Name for the API key (default: cli-<hostname>-<timestamp>)")
	expiryDays := fs.Int("expiry-days", 90, "Number of days until API key expiry")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Prompt for username and password
	user, err := users.PromptLogin()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Authenticate with server to get session token
	err = c.Login(user.Username, user.Password)
	if err != nil {
		return fmt.Errorf("server authentication failed: %w", err)
	}

	fmt.Printf("✓ Authenticated as %s (%s, %s)\n", user.Username, user.Team, user.Role)

	// Generate default key name if not provided
	if *keyName == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		*keyName = fmt.Sprintf("cli-%s-%d", hostname, time.Now().Unix())
	}

	// Generate API key via the API
	req := map[string]interface{}{
		"name":        *keyName,
		"expiry_days": *expiryDays,
	}

	var resp map[string]interface{}
	err = c.http.POST("/api/profile/api-keys", req, &resp)
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	// Extract API key from response
	apiKey, ok := resp["key"].(string)
	if !ok || apiKey == "" {
		return fmt.Errorf("server did not return API key")
	}

	// Parse timestamps
	createdAtStr, _ := resp["created_at"].(string)
	expiresAtStr, _ := resp["expires_at"].(string)

	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)

	// Save credentials to file
	creds := &Credentials{
		ServerURL: c.baseURL,
		Username:  user.Username,
		APIKey:    apiKey,
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
		KeyName:   *keyName,
	}

	err = SaveCredentials(creds)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	credPath, _ := GetCredentialsPath()
	fmt.Printf("✓ Generated API key '%s'\n", *keyName)
	fmt.Printf("✓ Expires: %s\n", expiresAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("✓ Credentials saved to: %s\n", credPath)
	fmt.Printf("\nYou can now use the CLI without authentication prompts.\n")
	fmt.Printf("To logout, run: %s logout\n", os.Args[0])

	return nil
}

// LoginSSOCommand authenticates using OIDC/SSO and saves API key locally
func (c *Client) LoginSSOCommand(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	keyName := fs.String("name", "", "Name for the API key (default: cli-<hostname>-<timestamp>)")
	expiryDays := fs.Int("expiry-days", 90, "Number of days until API key expiry")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("🔐 Starting SSO authentication...")

	// 1. Generate PKCE code verifier and challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeChallenge := generateCodeChallenge(codeVerifier)

	// 2. Generate state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// 3. Start local callback server
	port, callbackURL, resultChan := startCallbackServer(state)
	fmt.Printf("📡 Started local callback server on port %d\n", port)

	// 4. Build authorization URL
	authURL, err := buildOIDCAuthURL(c.baseURL, callbackURL, codeChallenge, state)
	if err != nil {
		return fmt.Errorf("failed to build authorization URL: %w", err)
	}

	// 5. Open browser
	fmt.Println("🌐 Opening browser for authentication...")
	fmt.Printf("If browser doesn't open automatically, visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Printf("⚠️  Failed to open browser automatically: %v\n", err)
		fmt.Printf("Please open the URL manually in your browser.\n\n")
	}

	// 6. Wait for callback with timeout
	var result callbackServerResult
	select {
	case result = <-resultChan:
		// Shutdown callback server
		if result.shutdownFn != nil {
			defer result.shutdownFn()
		}

		if result.err != nil {
			return result.err
		}

	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authentication timed out after 5 minutes")
	}

	fmt.Println("✓ Authorization code received")

	// 6. Exchange code for token
	accessToken, username, err := exchangeCodeForToken(c.baseURL, result.code, codeVerifier, callbackURL)
	if err != nil {
		return fmt.Errorf("token exchange failed: %w", err)
	}

	fmt.Printf("✓ Authenticated as %s\n", username)

	// 7. Generate default key name if not provided
	if *keyName == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		*keyName = fmt.Sprintf("cli-%s-%d", hostname, time.Now().Unix())
	}

	// 8. Generate API key using the access token
	apiKey, apiKeyName, expiresAt, err := generateAPIKeyWithToken(c.baseURL, accessToken, *keyName, *expiryDays)
	if err != nil {
		return fmt.Errorf("failed to generate API key: %w", err)
	}

	// 9. Save credentials to file
	creds := &Credentials{
		ServerURL: c.baseURL,
		Username:  username,
		APIKey:    apiKey,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		KeyName:   apiKeyName,
	}

	err = SaveCredentials(creds)
	if err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	credPath, _ := GetCredentialsPath()
	fmt.Printf("✓ Generated API key '%s'\n", apiKeyName)
	fmt.Printf("✓ Expires: %s\n", expiresAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("✓ Credentials saved to: %s\n", credPath)
	fmt.Printf("\nYou can now use the CLI without authentication prompts.\n")
	fmt.Printf("To logout, run: %s logout\n", os.Args[0])

	return nil
}

// LogoutCommand removes the locally stored credentials
func (c *Client) LogoutCommand() error {
	// Check if credentials exist
	credPath, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(credPath); os.IsNotExist(err) {
		fmt.Println("No credentials found. You are not logged in.")
		return nil
	}

	// Remove credentials file
	err = ClearCredentials()
	if err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	fmt.Println("✓ Logged out successfully")
	fmt.Printf("✓ Removed credentials from: %s\n", credPath)
	fmt.Printf("\nTo login again, run: %s login\n", os.Args[0])

	return nil
}

// filterResources applies client-side filtering to resource instances
func (c *Client) filterResources(resources map[string][]*ResourceInstance, resourceType, state string) map[string][]*ResourceInstance {
	filtered := make(map[string][]*ResourceInstance)

	for appName, resourceList := range resources {
		filteredList := []*ResourceInstance{}

		for _, resource := range resourceList {
			// Check type filter
			if resourceType != "" && !strings.EqualFold(resource.ResourceType, resourceType) {
				continue
			}

			// Check state filter
			if state != "" && !strings.EqualFold(resource.State, state) {
				continue
			}

			// Resource matches all filters
			filteredList = append(filteredList, resource)
		}

		// Only include app if it has matching resources
		if len(filteredList) > 0 {
			filtered[appName] = filteredList
		}
	}

	return filtered
}

// ListResourcesCommand lists all resource instances with optional filtering by application, type, and state
func (c *Client) ListResourcesCommand(appName, resourceType, state string) error {
	resources, err := c.ListResources(appName)
	if err != nil {
		return err
	}

	// Apply client-side filtering
	if resourceType != "" || state != "" {
		resources = c.filterResources(resources, resourceType, state)
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

	// Add filter info to title
	if resourceType != "" || state != "" {
		filterParts := []string{}
		if resourceType != "" {
			filterParts = append(filterParts, fmt.Sprintf("type=%s", resourceType))
		}
		if state != "" {
			filterParts = append(filterParts, fmt.Sprintf("state=%s", state))
		}
		title += fmt.Sprintf(" [filtered: %s]", strings.Join(filterParts, ", "))
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
			//nolint:staticcheck // Simple if statement is more readable than tagged switch here
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

// ResourceCommand handles resource management subcommands
func (c *Client) ResourceCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("resource command requires a subcommand (get, delete, update, transition, health)")
	}

	subcommand := args[0]
	formatter := NewOutputFormatter()

	switch subcommand {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("get subcommand requires a resource ID")
		}
		resourceID := args[1]

		resource, err := c.GetResource(resourceID)
		if err != nil {
			return fmt.Errorf("failed to get resource: %w", err)
		}

		formatter.PrintHeader(fmt.Sprintf("Resource Details: %s", resource.ResourceName))
		formatter.PrintKeyValue(0, "ID", fmt.Sprintf("%d", resource.ID))
		formatter.PrintKeyValue(0, "Application", resource.ApplicationName)
		formatter.PrintKeyValue(0, "Name", resource.ResourceName)
		formatter.PrintKeyValue(0, "Type", resource.ResourceType)
		formatter.PrintKeyValue(0, "State", resource.State)
		formatter.PrintKeyValue(0, "Health Status", resource.HealthStatus)

		if resource.ProviderID != nil && *resource.ProviderID != "" {
			formatter.PrintKeyValue(0, "Provider ID", *resource.ProviderID)
		}

		if len(resource.Configuration) > 0 {
			formatter.PrintSection(0, SymbolResource, "Configuration:")
			for key, value := range resource.Configuration {
				formatter.PrintKeyValue(1, key, value)
			}
		}

	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("delete subcommand requires a resource ID")
		}
		resourceID := args[1]

		if err := c.DeleteResource(resourceID); err != nil {
			return fmt.Errorf("failed to delete resource: %w", err)
		}

		formatter.PrintSuccess(fmt.Sprintf("Resource %s deleted successfully", resourceID))

	case "update":
		if len(args) < 3 {
			return fmt.Errorf("update subcommand requires a resource ID and config JSON")
		}
		resourceID := args[1]
		configJSON := args[2]

		var config map[string]interface{}
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return fmt.Errorf("invalid config JSON: %w", err)
		}

		if err := c.UpdateResource(resourceID, config); err != nil {
			return fmt.Errorf("failed to update resource: %w", err)
		}

		formatter.PrintSuccess(fmt.Sprintf("Resource %s updated successfully", resourceID))

	case "transition":
		if len(args) < 3 {
			return fmt.Errorf("transition subcommand requires a resource ID and target state")
		}
		resourceID := args[1]
		targetState := args[2]

		if err := c.TransitionResource(resourceID, targetState); err != nil {
			return fmt.Errorf("failed to transition resource: %w", err)
		}

		formatter.PrintSuccess(fmt.Sprintf("Resource %s transitioned to %s", resourceID, targetState))

	case "health":
		if len(args) < 2 {
			return fmt.Errorf("health subcommand requires a resource ID")
		}
		resourceID := args[1]

		// Check if --check flag is present to trigger new health check
		checkNew := false
		for _, arg := range args {
			if arg == "--check" {
				checkNew = true
				break
			}
		}

		var health map[string]interface{}
		var err error

		if checkNew {
			health, err = c.CheckResourceHealth(resourceID)
			if err != nil {
				return fmt.Errorf("failed to check resource health: %w", err)
			}
		} else {
			health, err = c.GetResourceHealth(resourceID)
			if err != nil {
				return fmt.Errorf("failed to get resource health: %w", err)
			}
		}

		formatter.PrintHeader(fmt.Sprintf("Resource Health: %s", resourceID))
		for key, value := range health {
			formatter.PrintKeyValue(0, key, value)
		}

	default:
		return fmt.Errorf("unknown resource subcommand: %s (valid: get, delete, update, transition, health)", subcommand)
	}

	return nil
}

// AnalyzeCommand analyzes a Score specification for workflow dependencies and execution plan
func (c *Client) AnalyzeCommand(filename string) error {
	// Validate file path to prevent path traversal
	cleanPath, err := filepath.Abs(filename)
	if err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}
	if err := security.ValidateFilePath(cleanPath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	// Read the Score specification file
	data, err := os.ReadFile(cleanPath) // #nosec G304 - path validated above
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
		return fmt.Errorf("score specification must have metadata.name")
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

// RetryWorkflowCommand retries a failed workflow execution from the first failed step
func (c *Client) RetryWorkflowCommand(workflowID, workflowSpecFile string) error {
	formatter := NewOutputFormatter()

	// Read and parse the workflow specification file
	// #nosec G304 -- workflowSpecFile is user-provided CLI argument
	workflowData, err := os.ReadFile(workflowSpecFile)
	if err != nil {
		return fmt.Errorf("failed to read workflow spec file: %w", err)
	}

	var workflow types.Workflow
	if err := yaml.Unmarshal(workflowData, &workflow); err != nil {
		return fmt.Errorf("failed to parse workflow spec: %w", err)
	}

	// Convert workflow to JSON for API request
	workflowJSON, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	formatter.PrintInfo(fmt.Sprintf("%s Retrying workflow execution %s...", SymbolWorkflow, workflowID))

	// Make API request to retry endpoint
	url := fmt.Sprintf("%s/api/workflows/%s/retry", c.baseURL, workflowID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(workflowJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send retry request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("retry failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Display success message
	formatter.PrintSuccess("Workflow retry completed successfully!")
	formatter.PrintKeyValue(1, "Parent Execution ID", result["parent_execution_id"])
	formatter.PrintKeyValue(1, "Application", result["app_name"])
	formatter.PrintKeyValue(1, "Workflow", result["workflow_name"])

	return nil
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

	// ALWAYS show error message for failed steps (not just in verbose mode)
	if step.Status == "failed" && step.ErrorMessage != nil && *step.ErrorMessage != "" {
		fmt.Printf("   ❌ ERROR: %s\n", *step.ErrorMessage)
	}

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
	}

	// Display logs with better messaging for different scenarios
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
		// Different messages based on step status
		if step.Status == "failed" {
			fmt.Printf("   Logs: ⚠️  No logs available. Step may have failed before producing output.\n")
			fmt.Printf("         Check error message above for details.\n")
		} else if step.Status == "completed" {
			fmt.Printf("   Logs: (No output - step completed successfully without producing logs)\n")
		} else {
			fmt.Printf("   Logs: No output logs available\n")
		}
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

// ProviderCommand handles provider-related subcommands
func (c *Client) ProviderCommand(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("provider command requires a subcommand (list, stats, reload)")
	}

	subcommand := args[0]

	switch subcommand {
	case "list":
		return c.ListProvidersCommand()
	case "stats":
		return c.ProviderStatsCommand()
	case "reload":
		return c.ProviderReloadCommand()
	default:
		return fmt.Errorf("unknown provider subcommand: %s (available: list, stats, reload)", subcommand)
	}
}

// ListProvidersCommand lists all loaded providers
func (c *Client) ListProvidersCommand() error {
	formatter := NewOutputFormatter()

	providers, err := c.ListProviders()
	if err != nil {
		return fmt.Errorf("failed to list providers: %w", err)
	}

	if len(providers) == 0 {
		formatter.PrintEmptyState("No providers loaded")
		return nil
	}

	formatter.PrintHeader(fmt.Sprintf("Loaded Providers (%d):", len(providers)))

	for _, provider := range providers {
		formatter.PrintEmpty()
		formatter.PrintSection(0, SymbolApp, fmt.Sprintf("%s v%s", provider.Name, provider.Version))

		if provider.Category != "" {
			formatter.PrintKeyValue(1, "Category", provider.Category)
		}
		if provider.Description != "" {
			formatter.PrintKeyValue(1, "Description", provider.Description)
		}
		formatter.PrintKeyValue(1, "Provisioners", fmt.Sprintf("%d", provider.Provisioners))
		formatter.PrintKeyValue(1, "Golden Paths", fmt.Sprintf("%d", provider.GoldenPaths))
	}

	formatter.PrintEmpty()
	return nil
}

// ProviderStatsCommand shows provider statistics
func (c *Client) ProviderStatsCommand() error {
	formatter := NewOutputFormatter()

	stats, err := c.GetProviderStats()
	if err != nil {
		return fmt.Errorf("failed to get provider stats: %w", err)
	}

	formatter.PrintHeader("Provider Statistics")
	formatter.PrintEmpty()
	formatter.PrintKeyValue(0, "Total Providers", fmt.Sprintf("%d", stats.Providers))
	formatter.PrintKeyValue(0, "Total Provisioners", fmt.Sprintf("%d", stats.Provisioners))
	formatter.PrintEmpty()

	return nil
}

// ProviderReloadCommand reloads providers from admin-config.yaml
func (c *Client) ProviderReloadCommand() error {
	formatter := NewOutputFormatter()

	formatter.PrintHeader("Reloading Providers")
	formatter.PrintEmpty()

	response, err := c.ReloadProviders()
	if err != nil {
		return fmt.Errorf("failed to reload providers: %w", err)
	}

	if success, ok := response["success"].(bool); ok && success {
		formatter.PrintSuccess(fmt.Sprintf("%v", response["message"]))
		formatter.PrintEmpty()

		if providers, ok := response["providers"].(float64); ok {
			formatter.PrintKeyValue(0, "Providers Loaded", fmt.Sprintf("%d", int(providers)))
		}
		if provisioners, ok := response["provisioners"].(float64); ok {
			formatter.PrintKeyValue(0, "Provisioners Loaded", fmt.Sprintf("%d", int(provisioners)))
		}
		formatter.PrintEmpty()
	} else {
		return fmt.Errorf("reload failed: %v", response)
	}

	return nil
}

// StatsCommand displays platform statistics (applications, workflows, resources, users)
func (c *Client) StatsCommand() error {
	formatter := NewOutputFormatter()

	stats, err := c.GetStats()
	if err != nil {
		return fmt.Errorf("failed to get statistics: %w", err)
	}

	formatter.PrintHeader("📊 Platform Statistics")
	formatter.PrintEmpty()
	formatter.PrintSection(0, "📦", fmt.Sprintf("Applications: %d", stats.Applications))
	formatter.PrintSection(0, "⚙️", fmt.Sprintf("Active Workflows: %d", stats.Workflows))
	formatter.PrintSection(0, "🔧", fmt.Sprintf("Resources: %d", stats.Resources))
	formatter.PrintSection(0, "👤", fmt.Sprintf("Users: %d", stats.Users))
	formatter.PrintEmpty()

	return nil
}

// WorkflowDetailCommand displays comprehensive metadata about a workflow execution
func (c *Client) WorkflowDetailCommand(workflowID string) error {
	formatter := NewOutputFormatter()

	// Get workflow details
	workflow, err := c.GetWorkflowDetail(workflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow details: %w", err)
	}

	// Header
	formatter.PrintHeader(fmt.Sprintf("Workflow Details: %s", workflow.WorkflowName))
	formatter.PrintEmpty()

	// Basic information
	formatter.PrintKeyValue(0, "Workflow ID", workflow.ID)
	formatter.PrintKeyValue(0, "Workflow Name", workflow.WorkflowName)
	formatter.PrintKeyValue(0, "Application", workflow.ApplicationName)

	// Status with visual indicator
	statusDisplay := formatter.PrintStatusBadge(workflow.Status)
	formatter.PrintKeyValue(0, "Status", statusDisplay)

	// Timestamps
	formatter.PrintKeyValue(0, "Started At", workflow.StartedAt.Format(time.RFC3339))
	if workflow.CompletedAt != nil {
		formatter.PrintKeyValue(0, "Completed At", workflow.CompletedAt.Format(time.RFC3339))

		// Calculate duration from timestamps
		duration := workflow.CompletedAt.Sub(workflow.StartedAt)
		formatter.PrintKeyValue(0, "Duration", formatter.FormatDuration(duration))
	}

	// Error message if failed
	if workflow.ErrorMessage != nil && *workflow.ErrorMessage != "" {
		formatter.PrintEmpty()
		formatter.PrintError(fmt.Sprintf("Error: %s", *workflow.ErrorMessage))
	}

	formatter.PrintEmpty()

	// Step summary
	completedSteps := 0
	for _, step := range workflow.Steps {
		if step.Status == "completed" || step.Status == "succeeded" {
			completedSteps++
		}
	}
	formatter.PrintSection(0, "⚙️", fmt.Sprintf("Steps: %d/%d completed", completedSteps, workflow.TotalSteps))
	formatter.PrintEmpty()

	// Step breakdown table
	if len(workflow.Steps) > 0 {
		columns := []TableColumn{
			{Header: "#", Width: 3},
			{Header: "Step Name", Width: 30},
			{Header: "Type", Width: 15},
			{Header: "Status", Width: 12},
			{Header: "Duration", Width: 10},
		}

		formatter.PrintTableHeader(columns)

		for _, step := range workflow.Steps {
			statusIcon := ""
			switch step.Status {
			case "completed", "succeeded":
				statusIcon = "✓"
			case "running":
				statusIcon = "⏳"
			case "failed":
				statusIcon = "✗"
			default:
				statusIcon = "○"
			}

			stepNum := fmt.Sprintf("%d", step.StepNumber)
			stepName := step.StepName
			if len(stepName) > 28 {
				stepName = stepName[:25] + "..."
			}
			stepType := step.StepType

			durationStr := "-"
			if step.DurationMs != nil && *step.DurationMs > 0 {
				duration := time.Duration(*step.DurationMs) * time.Millisecond
				durationStr = formatter.FormatDuration(duration)
			}

			formatter.PrintTableRow(columns, []string{
				stepNum,
				stepName,
				stepType,
				fmt.Sprintf("%s %s", statusIcon, step.Status),
				durationStr,
			})
		}

		formatter.PrintEmpty()
	}

	return nil
}

// WhoamiCommand displays current user information and authentication status
func (c *Client) WhoamiCommand() error {
	formatter := NewOutputFormatter()

	// Get user profile from server
	profile, err := c.GetProfile()
	if err != nil {
		return fmt.Errorf("failed to get user profile: %w", err)
	}

	formatter.PrintHeader("👤 Current User")
	formatter.PrintEmpty()
	formatter.PrintKeyValue(0, "Username", profile.Username)
	formatter.PrintKeyValue(0, "Team", profile.Team)
	formatter.PrintKeyValue(0, "Role", profile.Role)

	// Display authentication source
	formatter.PrintEmpty()
	formatter.PrintSection(0, SymbolInfo, "Authentication:")

	if apiKey := os.Getenv("IDP_API_KEY"); apiKey != "" {
		formatter.PrintKeyValue(1, "Source", "Environment variable (IDP_API_KEY)")
		formatter.PrintKeyValue(1, "Key", maskAPIKey(apiKey))
	} else {
		creds, _ := LoadCredentials()
		if creds != nil {
			formatter.PrintKeyValue(1, "Source", "Credentials file")
			formatter.PrintKeyValue(1, "Key Name", creds.KeyName)
			formatter.PrintKeyValue(1, "Key", maskAPIKey(creds.APIKey))
			formatter.PrintKeyValue(1, "Expires", creds.ExpiresAt.Format("2006-01-02 15:04:05"))
		}
	}

	formatter.PrintEmpty()

	return nil
}

// maskAPIKey masks all but first and last 4 characters of API key
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
