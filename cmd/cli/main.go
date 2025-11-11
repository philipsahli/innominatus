package main

import (
	"fmt"
	"innominatus/internal/cli"
	"innominatus/internal/users"
	"innominatus/internal/validation"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// NOTE: This file contains numerous fmt.Println/Printf statements that are INTENTIONAL
// user-facing CLI output. These are NOT debug logging and should NOT be converted to
// structured logging. They provide the interactive UX for the CLI commands and are
// designed for human-readable terminal output.

var (
	serverURL      string
	details        bool
	skipValidation bool
	client         *cli.Client
)

// Commands that don't require server authentication
var localCommands = map[string]bool{
	"run":             true,
	"validate":        true,
	"analyze":         true,
	"demo-time":       true,
	"demo-nuke":       true,
	"demo-status":     true,
	"demo-reset":      true,
	"fix-gitea-oauth": true,
	"login":           true,
	"logout":          true,
	"chat":            true,
	"help":            true, // Cobra built-in help command
	"completion":      true, // Cobra built-in completion command
	"bash":            true, // completion subcommands
	"zsh":             true,
	"fish":            true,
	"powershell":      true,
}

var rootCmd = &cobra.Command{
	Use:   "innominatus-ctl",
	Short: "Open Alps CLI",
	Long:  `Command-line interface for the Open Alps Score-based Platform Orchestration system.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize client with server URL
		client = cli.NewClient(serverURL)

		// Skip authentication for built-in Cobra commands (help, completion)
		// These commands have no RunE or Run function
		if cmd.RunE == nil && cmd.Run == nil {
			return nil
		}

		// Skip authentication for local commands
		cmdName := cmd.Name()
		if localCommands[cmdName] {
			return nil
		}

		// Run fast configuration validation for server commands
		if !skipValidation {
			summary := validation.ValidateWithMode(validation.ValidationModeFast)
			if !summary.Valid {
				fmt.Printf("❌ Configuration validation failed. Run with --skip-validation to bypass.\n")
				summary.PrintSummary()
				os.Exit(1)
			}
			if summary.WarningCount > 0 {
				fmt.Printf("⚠️  Configuration warnings detected (%d warnings)\n", summary.WarningCount)
			}
		}

		// Check if API key is already set
		if client.HasToken() {
			if os.Getenv("IDP_API_KEY") != "" {
				fmt.Printf("✓ Using API key from environment variable\n")
			} else {
				fmt.Printf("✓ Using API key from credentials file\n")
			}
			return nil
		}

		// Prompt for login for server commands
		user, err := users.PromptLogin()
		if err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}

		// Login to server
		err = client.Login(user.Username, user.Password)
		if err != nil {
			return fmt.Errorf("server authentication failed: %w", err)
		}

		return nil
	},
}

func init() {
	// Persistent flags available to all commands
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:8081", "Score orchestrator server URL")
	rootCmd.PersistentFlags().BoolVar(&details, "details", false, "Show detailed information including URLs and workflow links")
	rootCmd.PersistentFlags().BoolVar(&skipValidation, "skip-validation", false, "Skip configuration validation")
}

// Basic commands
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployed applications",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ListCommand(details)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status <app-name>",
	Short: "Show application status and resources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.StatusCommand(args[0])
	},
}

var (
	validateExplain bool
	validateFormat  string
)

var validateCmd = &cobra.Command{
	Use:   "validate <score-spec.yaml>",
	Short: "Validate Score spec locally",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ValidateCommand(args[0], validateExplain, validateFormat)
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze <score-spec.yaml>",
	Short: "Analyze Score spec workflow dependencies",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.AnalyzeCommand(args[0])
	},
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show platform statistics (apps, workflows, resources, users)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.StatsCommand()
	},
}

var environmentsCmd = &cobra.Command{
	Use:   "environments",
	Short: "List active environments",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.EnvironmentsCommand()
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <app-name>",
	Short: "Delete application and all resources completely",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DeleteCommand(args[0])
	},
}

var deprovisionCmd = &cobra.Command{
	Use:   "deprovision <app-name>",
	Short: "Deprovision infrastructure (keep audit trail)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DeprovisionCommand(args[0])
	},
}

// Workflow commands
var listWorkflowsCmd = &cobra.Command{
	Use:   "list-workflows [app-name]",
	Short: "List workflow executions (optionally filtered by app)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := ""
		if len(args) > 0 {
			appName = args[0]
		}
		return client.ListWorkflowsCommand(appName)
	},
}

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Workflow operations",
}

var workflowDetailCmd = &cobra.Command{
	Use:   "detail <workflow-id>",
	Short: "Show detailed workflow metadata and step breakdown",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.WorkflowDetailCommand(args[0])
	},
}

var (
	logsStep     string
	logsStepOnly bool
	logsTail     int
	logsVerbose  bool
)

var workflowLogsCmd = &cobra.Command{
	Use:   "logs <workflow-id>",
	Short: "Show workflow execution logs with step details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := cli.LogsOptions{
			Step:     logsStep,
			StepOnly: logsStepOnly,
			Tail:     logsTail,
			Verbose:  logsVerbose,
		}
		return client.LogsCommand(args[0], options)
	},
}

// Backward compatibility: logs command
var logsCmd = &cobra.Command{
	Use:   "logs <workflow-id>",
	Short: "Show workflow execution logs (shortcut for workflow logs)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		options := cli.LogsOptions{
			Step:     logsStep,
			StepOnly: logsStepOnly,
			Tail:     logsTail,
			Verbose:  logsVerbose,
		}
		return client.LogsCommand(args[0], options)
	},
}

var retryCmd = &cobra.Command{
	Use:   "retry <workflow-id> <workflow-spec.yaml>",
	Short: "Retry failed workflow from first failed step",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.RetryWorkflowCommand(args[0], args[1])
	},
}

// Resource commands
var (
	resourceType  string
	resourceState string
)

var listResourcesCmd = &cobra.Command{
	Use:   "list-resources [app-name]",
	Short: "List resource instances with optional filters",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := ""
		if len(args) > 0 {
			appName = args[0]
		}
		return client.ListResourcesCommand(appName, resourceType, resourceState)
	},
}

var resourceCmd = &cobra.Command{
	Use:   "resource",
	Short: "Manage resource instances",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ResourceCommand(args)
	},
}

// Graph commands
var (
	graphFormat string
	graphOutput string
)

var graphExportCmd = &cobra.Command{
	Use:   "graph-export <app-name>",
	Short: "Export workflow graph visualization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.GraphExportCommand(args[0], graphFormat, graphOutput)
	},
}

var graphStatusCmd = &cobra.Command{
	Use:   "graph-status <app-name>",
	Short: "Show workflow graph status and statistics",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.GraphStatusCommand(args[0])
	},
}

// Golden path commands
var listGoldenPathsCmd = &cobra.Command{
	Use:   "list-goldenpaths",
	Short: "List available golden paths",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ListGoldenPathsCommand()
	},
}

var runParams []string

var runCmd = &cobra.Command{
	Use:   "run <golden-path-name> [score-spec.yaml]",
	Short: "Run a golden path workflow",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		goldenPath := args[0]
		scoreFile := ""
		if len(args) > 1 {
			scoreFile = args[1]
		}

		// Parse parameters into map
		paramMap := make(map[string]string)
		for _, param := range runParams {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid parameter format '%s'. Use key=value", param)
			}
			paramMap[parts[0]] = parts[1]
		}

		return client.RunGoldenPathCommand(goldenPath, scoreFile, paramMap)
	},
}

// Demo commands
var demoComponent string

var demoTimeCmd = &cobra.Command{
	Use:   "demo-time",
	Short: "Install/reconcile demo environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DemoTimeCommand(demoComponent)
	},
}

var demoNukeCmd = &cobra.Command{
	Use:   "demo-nuke",
	Short: "Uninstall and clean demo environment",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DemoNukeCommand()
	},
}

var demoStatusCmd = &cobra.Command{
	Use:   "demo-status",
	Short: "Check demo environment health and status",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DemoStatusCommand()
	},
}

var noCheck bool

var demoResetCmd = &cobra.Command{
	Use:   "demo-reset",
	Short: "Reset database to clean state (deletes all data)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DemoResetCommand(noCheck)
	},
}

var fixGiteaOAuthCmd = &cobra.Command{
	Use:   "fix-gitea-oauth",
	Short: "Fix Gitea OAuth2 auto-registration with Keycloak",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.FixGiteaOAuthCommand()
	},
}

// Auth commands
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate and store API key locally",
	Long: `Authenticate with the innominatus server and store credentials locally.

By default, uses username/password authentication. Use the --sso flag for
browser-based OIDC/Keycloak authentication.

Examples:
  # Password-based login
  innominatus-ctl login

  # SSO login (opens browser)
  innominatus-ctl login --sso

  # Specify API key name and expiry
  innominatus-ctl login --sso --name my-laptop --expiry-days 30`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sso, _ := cmd.Flags().GetBool("sso")
		if sso {
			return client.LoginSSOCommand(args)
		}
		return client.LoginCommand(args)
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.LogoutCommand()
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user information",
	Long: `Display current authenticated user information and authentication status.

This command verifies authentication by querying the server and shows:
  - Username, team, and role
  - Authentication source (environment variable or credentials file)
  - Masked API key

Examples:
  innominatus-ctl whoami`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.WhoamiCommand()
	},
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Interactive AI assistant chat",
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("chat command not yet implemented")
	},
}

// Admin commands
var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin commands (requires admin role)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.AdminCommand(args)
	},
}

// Team commands
var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Team management commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.TeamCommand(args)
	},
}

// Provider commands
var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Provider management commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ProviderCommand(args)
	},
}

func init() {
	// Add flags to specific commands

	// Login command flags
	loginCmd.Flags().BoolP("sso", "s", false, "Use SSO (OIDC) authentication instead of password")
	loginCmd.Flags().String("name", "", "Name for API key (default: cli-<hostname>-<timestamp>)")
	loginCmd.Flags().Int("expiry-days", 90, "Days until API key expires")

	validateCmd.Flags().BoolVar(&validateExplain, "explain", false, "Show detailed validation explanations")
	validateCmd.Flags().StringVar(&validateFormat, "format", "text", "Output format (text, json, simple)")

	workflowLogsCmd.Flags().StringVar(&logsStep, "step", "", "Show logs for specific step name")
	workflowLogsCmd.Flags().BoolVar(&logsStepOnly, "step-only", false, "Only show step logs, skip workflow header")
	workflowLogsCmd.Flags().IntVar(&logsTail, "tail", 0, "Number of lines to show from end of logs (0 = all)")
	workflowLogsCmd.Flags().BoolVar(&logsVerbose, "verbose", false, "Show additional metadata")

	logsCmd.Flags().StringVar(&logsStep, "step", "", "Show logs for specific step name")
	logsCmd.Flags().BoolVar(&logsStepOnly, "step-only", false, "Only show step logs, skip workflow header")
	logsCmd.Flags().IntVar(&logsTail, "tail", 0, "Number of lines to show from end of logs (0 = all)")
	logsCmd.Flags().BoolVar(&logsVerbose, "verbose", false, "Show additional metadata")

	listResourcesCmd.Flags().StringVar(&resourceType, "type", "", "Filter by resource type (e.g., postgres, redis)")
	listResourcesCmd.Flags().StringVar(&resourceState, "state", "", "Filter by state (e.g., active, provisioning, failed)")

	graphExportCmd.Flags().StringVar(&graphFormat, "format", "svg", "Output format (svg, png, dot)")
	graphExportCmd.Flags().StringVar(&graphOutput, "output", "", "Output file path (default: stdout)")

	runCmd.Flags().StringArrayVar(&runParams, "param", []string{}, "Parameter override (key=value)")

	demoTimeCmd.Flags().StringVar(&demoComponent, "component", "", "Comma-separated list of components to install")

	demoResetCmd.Flags().BoolVar(&noCheck, "no-check", false, "Skip demo environment check")

	// Add workflow subcommands
	workflowCmd.AddCommand(workflowDetailCmd, workflowLogsCmd)

	// Add all commands to root
	rootCmd.AddCommand(
		listCmd,
		statusCmd,
		validateCmd,
		analyzeCmd,
		statsCmd,
		environmentsCmd,
		deleteCmd,
		deprovisionCmd,
		listWorkflowsCmd,
		workflowCmd,
		logsCmd,
		retryCmd,
		listResourcesCmd,
		resourceCmd,
		graphExportCmd,
		graphStatusCmd,
		listGoldenPathsCmd,
		runCmd,
		demoTimeCmd,
		demoNukeCmd,
		demoStatusCmd,
		demoResetCmd,
		fixGiteaOAuthCmd,
		loginCmd,
		logoutCmd,
		whoamiCmd,
		chatCmd,
		adminCmd,
		teamCmd,
		providerCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
