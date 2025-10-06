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
		"list-goldenpaths": true,
		"run":              true,
		"validate":         true,
		"analyze":          true,
		"demo-time":        true,
		"demo-nuke":        true,
		"demo-status":      true,
		"login":            true,
		"logout":           true,
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
		// Check if API key is already set (from environment variable)
		if os.Getenv("IDP_API_KEY") != "" {
			// API key authentication - no need to prompt for login
			fmt.Printf("✓ Using API key authentication\n")
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
			fmt.Fprintf(os.Stderr, "Usage: %s admin <show|add-user|list-users|delete-user>\n", os.Args[0])
			os.Exit(1)
		}
		if user == nil {
			fmt.Fprintf(os.Stderr, "Error: admin command requires authentication\n")
			os.Exit(1)
		}
		err = client.AdminCommand(user, flag.Args()[1:])

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

	case "list-workflows":
		appName := ""
		if len(flag.Args()) >= 2 {
			appName = flag.Args()[1]
		}
		err = client.ListWorkflowsCommand(appName)

	case "list-resources":
		appName := ""
		if len(flag.Args()) >= 2 {
			appName = flag.Args()[1]
		}
		err = client.ListResourcesCommand(appName)

	case "logs":
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

	case "login":
		// Login command - authenticate and store credentials
		err = client.LoginCommand(flag.Args()[1:])

	case "logout":
		// Logout command - remove stored credentials
		err = client.LogoutCommand()

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
	fmt.Printf("  environments          List active environments\n")
	fmt.Printf("  delete <name>         Delete application and all resources completely\n")
	fmt.Printf("  deprovision <name>    Deprovision infrastructure (keep audit trail)\n")
	fmt.Printf("  list-workflows [app]  List workflow executions (optionally filtered by app)\n")
	fmt.Printf("  list-resources [app]  List resource instances (optionally filtered by app)\n")
	fmt.Printf("  logs <workflow-id>    Show workflow execution logs with step details\n")
	fmt.Printf("  graph-export <app>    Export workflow graph visualization\n")
	fmt.Printf("  graph-status <app>    Show workflow graph status and statistics\n")
	fmt.Printf("  list-goldenpaths      List available golden paths\n")
	fmt.Printf("  run <path> [spec]     Run a golden path workflow\n")
	fmt.Printf("  login [options]       Authenticate and store API key locally\n")
	fmt.Printf("  logout                Remove stored credentials\n")
	fmt.Printf("  admin <command>       Admin commands (requires admin role)\n")
	fmt.Printf("    show                Show admin configuration\n")
	fmt.Printf("    add-user            Add new user\n")
	fmt.Printf("    list-users          List all users\n")
	fmt.Printf("    delete-user         Delete user\n")
	fmt.Printf("    generate-api-key    Generate API key for user\n")
	fmt.Printf("    list-api-keys       List API keys for user\n")
	fmt.Printf("    revoke-api-key      Revoke API key for user\n")
	fmt.Printf("  demo-time [options]   Install/reconcile demo environment\n")
	fmt.Printf("    -component <names>  Comma-separated list of components to install\n")
	fmt.Printf("                        (e.g., grafana, gitea,argocd). Dependencies are\n")
	fmt.Printf("                        automatically included. Omit to install all.\n")
	fmt.Printf("  demo-nuke             Uninstall and clean demo environment\n")
	fmt.Printf("  demo-status           Check demo environment health and status\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  --server <url>        Orchestrator server URL (default: http://localhost:8081)\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s list\n", os.Args[0])
	fmt.Printf("  %s status product-service\n", os.Args[0])
	fmt.Printf("  %s validate score-spec.yaml\n", os.Args[0])
	fmt.Printf("  %s analyze score-spec.yaml\n", os.Args[0])
	fmt.Printf("  %s list-workflows\n", os.Args[0])
	fmt.Printf("  %s list-workflows my-app\n", os.Args[0])
	fmt.Printf("  %s list-resources\n", os.Args[0])
	fmt.Printf("  %s list-resources my-app\n", os.Args[0])
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
	fmt.Printf("  %s demo-nuke\n", os.Args[0])
	fmt.Printf("  %s login\n", os.Args[0])
	fmt.Printf("  %s login --name my-laptop --expiry-days 30\n", os.Args[0])
	fmt.Printf("  %s logout\n", os.Args[0])
	fmt.Printf("  %s admin show\n", os.Args[0])
	fmt.Printf("  %s admin add-user --username bob --password secret --team dev --role user\n", os.Args[0])
	fmt.Printf("  %s admin list-users\n", os.Args[0])
	fmt.Printf("  %s admin delete-user bob\n", os.Args[0])
	fmt.Printf("  %s admin generate-api-key --name cli-key --expiry-days 90\n", os.Args[0])
	fmt.Printf("  %s admin list-api-keys\n", os.Args[0])
	fmt.Printf("  %s admin revoke-api-key --name cli-key\n", os.Args[0])
	fmt.Printf("  export IDP_API_KEY=your_api_key_here\n")
	fmt.Printf("  %s list  # Uses API key from environment\n", os.Args[0])
	fmt.Printf("  %s --server http://prod-orchestrator:8081 list\n", os.Args[0])
}
