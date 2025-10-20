package main

import (
	"flag"
	"fmt"
	"innominatus/internal/cli"
	"innominatus/internal/users"
	"innominatus/internal/validation"
	"os"
	"strings"
)

// NOTE: This file contains numerous fmt.Println/Printf statements that are INTENTIONAL
// user-facing CLI output. These are NOT debug logging and should NOT be converted to
// structured logging. They provide the interactive UX for the CLI commands and are
// designed for human-readable terminal output.

func main() {
	var serverURL = flag.String("server", "http://localhost:8081", "Score orchestrator server URL")
	var details = flag.Bool("details", false, "Show detailed information including URLs and workflow links")
	var skipValidation = flag.Bool("skip-validation", false, "Skip configuration validation")
	flag.Parse()

	if len(flag.Args()) < 1 {
		printUsage()
		os.Exit(1)
	}

	client := cli.NewClient(*serverURL)
	command := flag.Args()[0]

	// Commands that don't require server authentication
	localCommands := map[string]bool{
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
		"chat":            true, // AI assistant chat
	}

	// Run fast configuration validation for server commands (skip local commands)
	if !localCommands[command] && !*skipValidation {
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

	var user *users.User
	var err error
	if !localCommands[command] {
		// Check if API key is already set (from environment variable or credentials file)
		if client.HasToken() {
			// API key authentication - no need to prompt for login
			if os.Getenv("IDP_API_KEY") != "" {
				fmt.Printf("✓ Using API key from environment variable\n")
			} else {
				fmt.Printf("✓ Using API key from credentials file\n")
			}
		} else {
			// Authenticate user with server for server commands
			user, err = users.PromptLogin()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
				os.Exit(1)
			}

			// Login to server to get authentication token
			err = client.Login(user.Username, user.Password)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Server authentication failed: %v\n", err)
				os.Exit(1)
			}

			//fmt.Printf("✓ Authenticated as %s (%s, %s)\n", user.Username, user.Team, user.Role)
		}
	}
	switch command {
	case "list":
		err = client.ListCommand(*details)

	case "status":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: status command requires an application name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s status <app-name>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.StatusCommand(flag.Args()[1])

	case "validate":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: validate command requires a file path\n")
			fmt.Fprintf(os.Stderr, "Usage: %s validate <score-spec.yaml> [--explain] [--format=<text|json|simple>]\n", os.Args[0])
			os.Exit(1)
		}

		// Parse validate-specific flags
		validateFlags := flag.NewFlagSet("validate", flag.ExitOnError)
		explainFlag := validateFlags.Bool("explain", false, "Show detailed validation explanations")
		formatFlag := validateFlags.String("format", "text", "Output format (text, json, simple)")

		// Parse remaining arguments for validate command
		validateArgs := flag.Args()[2:]
		if len(validateArgs) > 0 {
			if err := validateFlags.Parse(validateArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing validate flags: %v\n", err)
				os.Exit(1)
			}
		}

		err = client.ValidateCommand(flag.Args()[1], *explainFlag, *formatFlag)

	case "analyze":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: analyze command requires a file path\n")
			fmt.Fprintf(os.Stderr, "Usage: %s analyze <score-spec.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.AnalyzeCommand(flag.Args()[1])

	case "stats":
		err = client.StatsCommand()

	case "environments":
		err = client.EnvironmentsCommand()

	case "delete":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: delete command requires an application name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s delete <app-name>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.DeleteCommand(flag.Args()[1])

	case "deprovision":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: deprovision command requires an application name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s deprovision <app-name>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.DeprovisionCommand(flag.Args()[1])

	case "admin":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: admin command requires a subcommand\n")
			fmt.Fprintf(os.Stderr, "Usage: %s admin <show|add-user|list-users|delete-user|user-api-keys|user-generate-key|user-revoke-key>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.AdminCommand(flag.Args()[1:])

	case "team":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: team command requires a subcommand\n")
			fmt.Fprintf(os.Stderr, "Usage: %s team <list|get|create|delete>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.TeamCommand(flag.Args()[1:])

	case "provider":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: provider command requires a subcommand\n")
			fmt.Fprintf(os.Stderr, "Usage: %s provider <list|stats>\n", os.Args[0])
			os.Exit(1)
		}
		err = client.ProviderCommand(flag.Args()[1:])

	case "list-goldenpaths":
		err = client.ListGoldenPathsCommand()

	case "run":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: run command requires a golden path name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s run <golden-path-name> [score-spec.yaml] [--param key=value ...]\n", os.Args[0])
			os.Exit(1)
		}

		// Parse run-specific flags
		runFlags := flag.NewFlagSet("run", flag.ContinueOnError)
		runFlags.SetOutput(os.Stderr)

		// Define --param flag for parameter overrides
		var params []string
		runFlags.Func("param", "Parameter override (key=value)", func(s string) error {
			params = append(params, s)
			return nil
		})

		// Parse flags after the golden path name
		goldenPath := flag.Args()[1]
		scoreFile := ""
		remainingArgs := flag.Args()[2:]

		// Parse flags
		if err := runFlags.Parse(remainingArgs); err != nil {
			os.Exit(1)
		}

		// Check for score file (first non-flag argument)
		if runFlags.NArg() > 0 {
			scoreFile = runFlags.Arg(0)
		}

		// Parse parameters into map
		paramMap := make(map[string]string)
		for _, param := range params {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) != 2 {
				fmt.Fprintf(os.Stderr, "Error: invalid parameter format '%s'. Use key=value\n", param)
				os.Exit(1)
			}
			paramMap[parts[0]] = parts[1]
		}

		err = client.RunGoldenPathCommand(goldenPath, scoreFile, paramMap)

	case "demo-time":
		// Parse demo-time specific flags
		demoFlags := flag.NewFlagSet("demo-time", flag.ExitOnError)
		componentFilter := demoFlags.String("component", "", "Comma-separated list of components to install (e.g., grafana, gitea,argocd)")

		// Parse remaining arguments for demo-time command
		if len(flag.Args()) > 1 {
			_ = demoFlags.Parse(flag.Args()[1:])
		}

		err = client.DemoTimeCommand(*componentFilter)

	case "demo-nuke":
		err = client.DemoNukeCommand()

	case "demo-status":
		err = client.DemoStatusCommand()

	case "demo-reset":
		err = client.DemoResetCommand()

	case "fix-gitea-oauth":
		err = client.FixGiteaOAuthCommand()

	case "list-workflows":
		appName := ""
		if len(flag.Args()) >= 2 {
			appName = flag.Args()[1]
		}
		err = client.ListWorkflowsCommand(appName)

	case "list-resources":
		// Parse list-resources-specific flags
		resourcesFlags := flag.NewFlagSet("list-resources", flag.ExitOnError)
		resourceType := resourcesFlags.String("type", "", "Filter by resource type (e.g., postgres, redis)")
		state := resourcesFlags.String("state", "", "Filter by state (e.g., active, provisioning, failed)")

		// Parse remaining arguments
		appName := ""
		resourcesArgs := flag.Args()[1:]

		// First non-flag argument is app name
		if len(resourcesArgs) > 0 && !strings.HasPrefix(resourcesArgs[0], "-") {
			appName = resourcesArgs[0]
			resourcesArgs = resourcesArgs[1:]
		}

		// Parse flags
		if len(resourcesArgs) > 0 {
			if err := resourcesFlags.Parse(resourcesArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing list-resources flags: %v\n", err)
				os.Exit(1)
			}
		}

		err = client.ListResourcesCommand(appName, *resourceType, *state)

	case "resource":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: resource command requires a subcommand\n")
			fmt.Fprintf(os.Stderr, "Usage: %s resource <get|delete|update|transition|health> <resource-id> [options]\n", os.Args[0])
			os.Exit(1)
		}
		err = client.ResourceCommand(flag.Args()[1:])

	case "workflow":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: workflow command requires a subcommand\n")
			fmt.Fprintf(os.Stderr, "Usage: %s workflow <detail|logs> <workflow-id> [options]\n", os.Args[0])
			os.Exit(1)
		}

		subcommand := flag.Args()[1]
		switch subcommand {
		case "detail":
			if len(flag.Args()) < 3 {
				fmt.Fprintf(os.Stderr, "Error: workflow detail requires a workflow ID\n")
				fmt.Fprintf(os.Stderr, "Usage: %s workflow detail <workflow-id>\n", os.Args[0])
				os.Exit(1)
			}
			err = client.WorkflowDetailCommand(flag.Args()[2])

		case "logs":
			if len(flag.Args()) < 3 {
				fmt.Fprintf(os.Stderr, "Error: workflow logs requires a workflow ID\n")
				fmt.Fprintf(os.Stderr, "Usage: %s workflow logs <workflow-id> [options]\n", os.Args[0])
				os.Exit(1)
			}
			workflowID := flag.Args()[2]

			// Parse logs-specific flags
			logsFlags := flag.NewFlagSet("logs", flag.ExitOnError)
			stepFlag := logsFlags.String("step", "", "Show logs for specific step name")
			stepOnlyFlag := logsFlags.Bool("step-only", false, "Only show step logs, skip workflow header")
			tailFlag := logsFlags.Int("tail", 0, "Number of lines to show from end of logs (0 = all)")
			verboseFlag := logsFlags.Bool("verbose", false, "Show additional metadata")

			// Parse remaining arguments for logs command
			logsArgs := flag.Args()[3:]
			if len(logsArgs) > 0 {
				if err := logsFlags.Parse(logsArgs); err != nil {
					fmt.Fprintf(os.Stderr, "Error parsing logs flags: %v\n", err)
					os.Exit(1)
				}
			}

			options := cli.LogsOptions{
				Step:     *stepFlag,
				StepOnly: *stepOnlyFlag,
				Tail:     *tailFlag,
				Verbose:  *verboseFlag,
			}

			err = client.LogsCommand(workflowID, options)

		default:
			fmt.Fprintf(os.Stderr, "Error: unknown workflow subcommand '%s'\n", subcommand)
			fmt.Fprintf(os.Stderr, "Usage: %s workflow <detail|logs> <workflow-id> [options]\n", os.Args[0])
			os.Exit(1)
		}

	case "logs":
		// Backward compatibility: logs <id> redirects to workflow logs <id>
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: logs command requires a workflow ID\n")
			fmt.Fprintf(os.Stderr, "Usage: %s logs <workflow-id> [options]\n", os.Args[0])
			os.Exit(1)
		}
		workflowID := flag.Args()[1]

		// Parse logs-specific flags
		logsFlags := flag.NewFlagSet("logs", flag.ExitOnError)
		stepFlag := logsFlags.String("step", "", "Show logs for specific step name")
		stepOnlyFlag := logsFlags.Bool("step-only", false, "Only show step logs, skip workflow header")
		tailFlag := logsFlags.Int("tail", 0, "Number of lines to show from end of logs (0 = all)")
		verboseFlag := logsFlags.Bool("verbose", false, "Show additional metadata")

		// Parse remaining arguments for logs command
		logsArgs := flag.Args()[2:]
		if len(logsArgs) > 0 {
			if err := logsFlags.Parse(logsArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing logs flags: %v\n", err)
				os.Exit(1)
			}
		}

		options := cli.LogsOptions{
			Step:     *stepFlag,
			StepOnly: *stepOnlyFlag,
			Tail:     *tailFlag,
			Verbose:  *verboseFlag,
		}

		err = client.LogsCommand(workflowID, options)

	case "graph-export":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: graph-export command requires an application name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s graph-export <app-name> [--format svg|png|dot] [--output file]\n", os.Args[0])
			os.Exit(1)
		}
		appName := flag.Args()[1]

		// Parse graph-export-specific flags
		graphFlags := flag.NewFlagSet("graph-export", flag.ExitOnError)
		formatFlag := graphFlags.String("format", "svg", "Output format (svg, png, dot)")
		outputFlag := graphFlags.String("output", "", "Output file path (default: stdout)")

		// Parse remaining arguments
		graphArgs := flag.Args()[2:]
		if len(graphArgs) > 0 {
			if err := graphFlags.Parse(graphArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing graph-export flags: %v\n", err)
				os.Exit(1)
			}
		}

		err = client.GraphExportCommand(appName, *formatFlag, *outputFlag)

	case "graph-status":
		if len(flag.Args()) < 2 {
			fmt.Fprintf(os.Stderr, "Error: graph-status command requires an application name\n")
			fmt.Fprintf(os.Stderr, "Usage: %s graph-status <app-name>\n", os.Args[0])
			os.Exit(1)
		}
		appName := flag.Args()[1]
		err = client.GraphStatusCommand(appName)

	case "retry":
		if len(flag.Args()) < 3 {
			fmt.Fprintf(os.Stderr, "Error: retry command requires workflow ID and workflow spec file\n")
			fmt.Fprintf(os.Stderr, "Usage: %s retry <workflow-id> <workflow-spec.yaml>\n", os.Args[0])
			os.Exit(1)
		}
		workflowID := flag.Args()[1]
		workflowSpec := flag.Args()[2]
		err = client.RetryWorkflowCommand(workflowID, workflowSpec)

	case "login":
		// Login command - authenticate and store credentials
		err = client.LoginCommand(flag.Args()[1:])

	case "logout":
		// Logout command - remove stored credentials
		err = client.LogoutCommand()

	case "chat":
		// AI chat command - TODO: implement AI chat functionality
		fmt.Fprintf(os.Stderr, "Chat command not yet implemented\n")
		os.Exit(1)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command '%s'\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("Open Alps CLI\n\n")
	fmt.Printf("Usage: %s [--server <url>] <command> [arguments]\n\n", os.Args[0])
	fmt.Printf("Commands:\n")
	fmt.Printf("  list                  List all deployed applications\n")
	fmt.Printf("  status <name>         Show application status and resources\n")
	fmt.Printf("  validate <file>       Validate Score spec locally\n")
	fmt.Printf("  analyze <file>        Analyze Score spec workflow dependencies\n")
	fmt.Printf("  stats                 Show platform statistics (apps, workflows, resources, users)\n")
	fmt.Printf("  environments          List active environments\n")
	fmt.Printf("  delete <name>         Delete application and all resources completely\n")
	fmt.Printf("  deprovision <name>    Deprovision infrastructure (keep audit trail)\n")
	fmt.Printf("  list-workflows [app]  List workflow executions (optionally filtered by app)\n")
	fmt.Printf("  list-resources [app] [--type TYPE] [--state STATE]\n")
	fmt.Printf("                        List resource instances with optional filters\n")
	fmt.Printf("  resource <command>    Manage resource instances\n")
	fmt.Printf("    get <id>            Get resource details\n")
	fmt.Printf("    delete <id>         Delete resource\n")
	fmt.Printf("    update <id> <json>  Update resource configuration\n")
	fmt.Printf("    transition <id> <state>  Transition resource state\n")
	fmt.Printf("    health <id> [--check]    Get/check resource health\n")
	fmt.Printf("  workflow <command>    Workflow operations\n")
	fmt.Printf("    detail <id>         Show detailed workflow metadata and step breakdown\n")
	fmt.Printf("    logs <id>           Show workflow execution logs with step details\n")
	fmt.Printf("  logs <workflow-id>    Show workflow execution logs (shortcut for workflow logs)\n")
	fmt.Printf("  retry <id> <spec>     Retry failed workflow from first failed step\n")
	fmt.Printf("  graph-export <app>    Export workflow graph visualization\n")
	fmt.Printf("  graph-status <app>    Show workflow graph status and statistics\n")
	fmt.Printf("  list-goldenpaths      List available golden paths\n")
	fmt.Printf("  run <path> [spec]     Run a golden path workflow\n")
	fmt.Printf("  login [options]       Authenticate and store API key locally\n")
	fmt.Printf("  logout                Remove stored credentials\n")
	fmt.Printf("  chat                  Interactive AI assistant chat\n")
	fmt.Printf("    --one-shot <q>      Ask a single question and exit\n")
	fmt.Printf("    --generate-spec <d> Generate Score spec from description\n")
	fmt.Printf("    -o <file>           Output file for generated spec (default: spec.yaml)\n")
	fmt.Printf("  admin <command>       Admin commands (requires admin role)\n")
	fmt.Printf("    show                Show admin configuration\n")
	fmt.Printf("    add-user            Add new user\n")
	fmt.Printf("    list-users          List all users\n")
	fmt.Printf("    delete-user         Delete user\n")
	fmt.Printf("    generate-api-key    Generate API key for user\n")
	fmt.Printf("    list-api-keys       List API keys for user\n")
	fmt.Printf("    revoke-api-key      Revoke API key for user\n")
	fmt.Printf("  team <command>        Team management commands\n")
	fmt.Printf("    list                List all teams\n")
	fmt.Printf("    get <id>            Get team details\n")
	fmt.Printf("    create              Create new team\n")
	fmt.Printf("    delete <id>         Delete team\n")
	fmt.Printf("  provider <command>    Provider management commands\n")
	fmt.Printf("    list                List all loaded providers\n")
	fmt.Printf("    stats               Show provider statistics\n")
	fmt.Printf("  demo-time [options]   Install/reconcile demo environment\n")
	fmt.Printf("    -component <names>  Comma-separated list of components to install\n")
	fmt.Printf("                        (e.g., grafana, gitea,argocd). Dependencies are\n")
	fmt.Printf("                        automatically included. Omit to install all.\n")
	fmt.Printf("  demo-nuke             Uninstall and clean demo environment\n")
	fmt.Printf("  demo-status           Check demo environment health and status\n")
	fmt.Printf("  demo-reset            Reset database to clean state (deletes all data)\n")
	fmt.Printf("  fix-gitea-oauth       Fix Gitea OAuth2 auto-registration with Keycloak\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  --server <url>        Orchestrator server URL (default: http://localhost:8081)\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s list\n", os.Args[0])
	fmt.Printf("  %s status product-service\n", os.Args[0])
	fmt.Printf("  %s validate score-spec.yaml\n", os.Args[0])
	fmt.Printf("  %s analyze score-spec.yaml\n", os.Args[0])
	fmt.Printf("  %s stats\n", os.Args[0])
	fmt.Printf("  %s list-workflows\n", os.Args[0])
	fmt.Printf("  %s list-workflows my-app\n", os.Args[0])
	fmt.Printf("  %s list-resources\n", os.Args[0])
	fmt.Printf("  %s list-resources my-app\n", os.Args[0])
	fmt.Printf("  %s list-resources --type postgres\n", os.Args[0])
	fmt.Printf("  %s list-resources --state active\n", os.Args[0])
	fmt.Printf("  %s list-resources my-app --type redis --state failed\n", os.Args[0])
	fmt.Printf("  %s resource get 42\n", os.Args[0])
	fmt.Printf("  %s resource health 42 --check\n", os.Args[0])
	fmt.Printf("  %s resource transition 42 deprovisioning\n", os.Args[0])
	fmt.Printf("  %s workflow detail 1234\n", os.Args[0])
	fmt.Printf("  %s workflow logs 1234 --step deploy-application\n", os.Args[0])
	fmt.Printf("  %s logs 1234\n", os.Args[0])
	fmt.Printf("  %s logs 1234 --step deploy-application --verbose\n", os.Args[0])
	fmt.Printf("  %s logs 1234 --tail 50 --step-only\n", os.Args[0])
	fmt.Printf("  %s list-goldenpaths\n", os.Args[0])
	fmt.Printf("  %s run deploy-app score-spec.yaml\n", os.Args[0])
	fmt.Printf("  %s run ephemeral-env\n", os.Args[0])
	fmt.Printf("  %s demo-time\n", os.Args[0])
	fmt.Printf("  %s demo-time -component grafana\n", os.Args[0])
	fmt.Printf("  %s demo-time -component gitea,argocd\n", os.Args[0])
	fmt.Printf("  %s demo-status\n", os.Args[0])
	fmt.Printf("  %s demo-reset\n", os.Args[0])
	fmt.Printf("  %s demo-nuke\n", os.Args[0])
	fmt.Printf("  %s fix-gitea-oauth\n", os.Args[0])
	fmt.Printf("  %s login\n", os.Args[0])
	fmt.Printf("  %s login --name my-laptop --expiry-days 30\n", os.Args[0])
	fmt.Printf("  %s logout\n", os.Args[0])
	fmt.Printf("  %s chat\n", os.Args[0])
	fmt.Printf("  %s chat --one-shot \"How do I deploy a Node.js app?\"\n", os.Args[0])
	fmt.Printf("  %s chat --generate-spec \"Python FastAPI app with Redis\" -o my-app.yaml\n", os.Args[0])
	fmt.Printf("  %s admin show\n", os.Args[0])
	fmt.Printf("  %s admin add-user --username bob --password secret --team dev --role user\n", os.Args[0])
	fmt.Printf("  %s admin list-users\n", os.Args[0])
	fmt.Printf("  %s admin delete-user bob\n", os.Args[0])
	fmt.Printf("  %s admin generate-api-key --name cli-key --expiry-days 90\n", os.Args[0])
	fmt.Printf("  %s admin list-api-keys\n", os.Args[0])
	fmt.Printf("  %s admin revoke-api-key --name cli-key\n", os.Args[0])
	fmt.Printf("  %s admin user-api-keys alice  # List API keys for user alice\n", os.Args[0])
	fmt.Printf("  %s admin user-generate-key --username alice --name alice-cli --expiry-days 30\n", os.Args[0])
	fmt.Printf("  %s admin user-revoke-key --username alice --key-name alice-cli\n", os.Args[0])
	fmt.Printf("  %s team list\n", os.Args[0])
	fmt.Printf("  %s team get platform-team\n", os.Args[0])
	fmt.Printf("  %s team create --name dev-team --description \"Development Team\"\n", os.Args[0])
	fmt.Printf("  %s team delete dev-team\n", os.Args[0])
	fmt.Printf("  export IDP_API_KEY=your_api_key_here\n")
	fmt.Printf("  %s list  # Uses API key from environment\n", os.Args[0])
	fmt.Printf("  %s --server http://prod-orchestrator:8081 list\n", os.Args[0])
}
