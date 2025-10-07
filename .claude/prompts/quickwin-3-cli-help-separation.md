# Quick Win 3: CLI Help Separation (User vs Admin)

**Effort:** 1 Tag
**Impact:** MEDIUM - Reduziert Cognitive Load massiv
**ROI:** Hoch

---

## Aufgabe

Trenne CLI Help in zwei Modi:
1. **User Mode** (default) - Nur Commands die Developer brauchen
2. **Admin Mode** (opt-in) - Platform Team Commands

## Kontext

**Problem:**
```bash
./innominatus-ctl --help

Commands:
  list              List applications       ← USER
  status            Show status            ← USER
  validate          Validate spec          ← USER/DEV
  delete            Delete app             ← USER

  admin             Admin operations       ← ADMIN!!
  demo-time         Install demo           ← DEVELOPMENT!!
  demo-nuke         Remove demo            ← DEVELOPMENT!!
  graph-export      Export graph           ← ADVANCED!!
  analyze           Analyze workflow       ← ADVANCED!!
```

**User Reaktion:**
- 😰 "25 Commands?! Was brauche ich?"
- 😰 "Was ist `admin`? `demo-time`? `graph-export`?"
- 😰 Information Overload!

**Ziel:**
User sieht nur 5-8 Commands die er wirklich braucht.

## Lösung: Progressive Disclosure

### Default Help (User Mode)

```bash
./innominatus-ctl --help

innominatus CLI - Deploy applications to your platform

COMMON COMMANDS
  deploy <file>        Deploy an application
  status <app>         Check application status
  logs <app>           View application logs
  delete <app>         Remove an application

GETTING STARTED
  innominatus-ctl tutorial         Interactive tutorial
  innominatus-ctl examples         Show example deployments

INFORMATION
  innominatus-ctl list             List your deployed applications
  innominatus-ctl list-goldenpaths Show available deployment patterns

HELP & SUPPORT
  innominatus-ctl docs             Open documentation
  innominatus-ctl support          Get help from Platform Team

Use 'innominatus-ctl <command> --help' for more info about a command

Advanced users: innominatus-ctl --advanced --help
Platform admins: innominatus-ctl --admin --help
```

**Total: 9 Commands** - Übersichtlich!

### Advanced Help

```bash
./innominatus-ctl --advanced --help

innominatus CLI - Advanced Commands

VALIDATION & ANALYSIS
  validate <file>      Validate Score specification
  analyze <file>       Analyze workflow dependencies
  graph-status <app>   Show workflow graph status
  graph-export <app>   Export workflow visualization

WORKFLOW MANAGEMENT
  list-workflows       List all workflow executions
  list-resources       List resource instances
  logs <workflow-id>   View workflow execution logs

GOLDEN PATHS
  run <path> <file>    Run a golden path workflow

Use 'innominatus-ctl <command> --help' for detailed info
```

### Admin Help

```bash
./innominatus-ctl --admin --help

innominatus CLI - Platform Administration

USER MANAGEMENT
  admin list-users         List all users
  admin add-user           Create new user
  admin delete-user        Remove user
  admin generate-api-key   Generate API key
  admin list-api-keys      List API keys
  admin revoke-api-key     Revoke API key

CONFIGURATION
  admin show               Show platform configuration
  environments             List environments

DEMO ENVIRONMENT (Development only)
  demo-time                Install demo environment
  demo-status              Check demo environment
  demo-nuke                Remove demo environment

Use 'innominatus-ctl admin <command> --help' for detailed info
```

## Implementation

### File: cmd/cli/main.go

Ändere `printUsage()` Funktion:

```go
func printUsage() {
	// Check for advanced/admin flags
	showAdvanced := hasFlag("--advanced")
	showAdmin := hasFlag("--admin")

	if showAdmin {
		printAdminHelp()
		return
	}

	if showAdvanced {
		printAdvancedHelp()
		return
	}

	// Default: User-friendly help
	printUserHelp()
}

func printUserHelp() {
	fmt.Printf("innominatus CLI - Deploy applications to your platform\n\n")

	fmt.Printf("COMMON COMMANDS\n")
	fmt.Printf("  deploy <file>        Deploy an application\n")
	fmt.Printf("  status <app>         Check application status\n")
	fmt.Printf("  logs <app>           View application logs\n")
	fmt.Printf("  delete <app>         Remove an application\n\n")

	fmt.Printf("GETTING STARTED\n")
	fmt.Printf("  innominatus-ctl tutorial         Interactive tutorial\n")
	fmt.Printf("  innominatus-ctl examples         Show example deployments\n\n")

	fmt.Printf("INFORMATION\n")
	fmt.Printf("  list                 List your deployed applications\n")
	fmt.Printf("  list-goldenpaths     Show available deployment patterns\n\n")

	fmt.Printf("HELP & SUPPORT\n")
	fmt.Printf("  docs                 Open documentation\n")
	fmt.Printf("  support              Get help from Platform Team\n\n")

	fmt.Printf("Use 'innominatus-ctl <command> --help' for more info\n\n")
	fmt.Printf("Advanced users: innominatus-ctl --advanced --help\n")
	fmt.Printf("Platform admins: innominatus-ctl --admin --help\n")
}

func printAdvancedHelp() {
	fmt.Printf("innominatus CLI - Advanced Commands\n\n")

	fmt.Printf("VALIDATION & ANALYSIS\n")
	fmt.Printf("  validate <file>      Validate Score specification\n")
	fmt.Printf("  analyze <file>       Analyze workflow dependencies\n")
	fmt.Printf("  graph-status <app>   Show workflow graph status\n")
	fmt.Printf("  graph-export <app>   Export workflow visualization\n\n")

	fmt.Printf("WORKFLOW MANAGEMENT\n")
	fmt.Printf("  list-workflows       List all workflow executions\n")
	fmt.Printf("  list-resources       List resource instances\n")
	fmt.Printf("  logs <workflow-id>   View workflow execution logs\n\n")

	fmt.Printf("GOLDEN PATHS\n")
	fmt.Printf("  run <path> <file>    Run a golden path workflow\n\n")

	fmt.Printf("Use 'innominatus-ctl <command> --help' for detailed info\n")
}

func printAdminHelp() {
	fmt.Printf("innominatus CLI - Platform Administration\n\n")

	fmt.Printf("USER MANAGEMENT\n")
	fmt.Printf("  admin list-users         List all users\n")
	fmt.Printf("  admin add-user           Create new user\n")
	fmt.Printf("  admin delete-user        Remove user\n")
	fmt.Printf("  admin generate-api-key   Generate API key\n")
	fmt.Printf("  admin list-api-keys      List API keys\n")
	fmt.Printf("  admin revoke-api-key     Revoke API key\n\n")

	fmt.Printf("CONFIGURATION\n")
	fmt.Printf("  admin show               Show platform configuration\n")
	fmt.Printf("  environments             List environments\n\n")

	fmt.Printf("DEMO ENVIRONMENT (Development only)\n")
	fmt.Printf("  demo-time                Install demo environment\n")
	fmt.Printf("  demo-status              Check demo environment\n")
	fmt.Printf("  demo-nuke                Remove demo environment\n\n")

	fmt.Printf("Use 'innominatus-ctl admin <command> --help' for detailed info\n")
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}
```

## Neue Commands hinzufügen

### 1. `tutorial` Command

```go
case "tutorial":
	err = client.TutorialCommand()
```

In `internal/cli/client.go`:

```go
func (c *Client) TutorialCommand() error {
	fmt.Println("🎓 innominatus Interactive Tutorial")
	fmt.Println("══════════════════════════════════")
	fmt.Println()
	fmt.Println("This tutorial will guide you through deploying your first app.")
	fmt.Println()
	fmt.Println("Prerequisites:")
	fmt.Println("  ✓ You have access to your company's innominatus platform")
	fmt.Println("  ✓ You have an API key (IDP_API_KEY environment variable)")
	fmt.Println()
	fmt.Println("📚 For the full tutorial, visit:")
	fmt.Println("   https://docs.innominatus.dev/user-guide/getting-started")
	fmt.Println()
	fmt.Println("Or run: innominatus-ctl docs")
	return nil
}
```

### 2. `examples` Command

```go
func (c *Client) ExamplesCommand() error {
	fmt.Println("📋 Example Deployments")
	fmt.Println("═══════════════════════")
	fmt.Println()
	fmt.Println("1. Simple Web App")
	fmt.Println("   innominatus-ctl deploy https://examples.innominatus.dev/simple-web.yaml")
	fmt.Println()
	fmt.Println("2. App with Database")
	fmt.Println("   innominatus-ctl deploy https://examples.innominatus.dev/app-with-db.yaml")
	fmt.Println()
	fmt.Println("3. Microservices")
	fmt.Println("   innominatus-ctl deploy https://examples.innominatus.dev/microservices.yaml")
	fmt.Println()
	fmt.Println("📚 More examples:")
	fmt.Println("   https://docs.innominatus.dev/user-guide/recipes")
	return nil
}
```

### 3. `docs` Command

```go
func (c *Client) DocsCommand() error {
	docsURL := "https://docs.innominatus.dev/user-guide"

	fmt.Printf("📚 Opening documentation: %s\n", docsURL)

	// Try to open in browser
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", docsURL)
	case "linux":
		cmd = exec.Command("xdg-open", docsURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", docsURL)
	default:
		fmt.Printf("\nPlease visit: %s\n", docsURL)
		return nil
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("\nCouldn't open browser. Please visit: %s\n", docsURL)
	}

	return nil
}
```

### 4. `support` Command

```go
func (c *Client) SupportCommand() error {
	fmt.Println("🆘 Getting Help")
	fmt.Println("═══════════════")
	fmt.Println()
	fmt.Println("Platform Support:")
	fmt.Println("  💬 Slack: #platform-support")
	fmt.Println("  📧 Email: platform-team@company.com")
	fmt.Println()
	fmt.Println("When asking for help, include:")
	fmt.Println("  • Your application name")
	fmt.Println("  • The command you ran")
	fmt.Println("  • The error message")
	fmt.Println("  • Output of: innominatus-ctl status <app>")
	fmt.Println()
	fmt.Println("Documentation:")
	fmt.Println("  📚 User Guide: https://docs.innominatus.dev/user-guide")
	fmt.Println("  ❓ Troubleshooting: https://docs.innominatus.dev/user-guide/troubleshooting")
	fmt.Println()
	fmt.Println("Or run: innominatus-ctl docs")
	return nil
}
```

## Command Aliases

Mache häufige Commands einfacher:

```go
// In main() switch statement

case "deploy":
	// Alias for: run deploy-app
	if len(flag.Args()) < 2 {
		fmt.Fprintf(os.Stderr, "Error: deploy command requires a file\n")
		fmt.Fprintf(os.Stderr, "Usage: innominatus-ctl deploy <score-spec.yaml>\n")
		os.Exit(1)
	}
	err = client.RunGoldenPathCommand("deploy-app", flag.Args()[1], nil)
```

Jetzt kann User machen:
```bash
innominatus-ctl deploy my-app.yaml
# Statt:
innominatus-ctl run deploy-app my-app.yaml
```

Viel intuitiver!

## Acceptance Criteria

✅ Default `--help` zeigt nur 9 User-relevante Commands
✅ `--advanced --help` zeigt erweiterte Commands
✅ `--admin --help` zeigt Admin Commands
✅ Neue Commands funktionieren:
   - `tutorial` - Zeigt Quick Start Link
   - `examples` - Zeigt Beispiel-Deployments
   - `docs` - Öffnet Dokumentation im Browser
   - `support` - Zeigt Support-Kontakte
✅ `deploy` Command ist Alias für `run deploy-app`
✅ Help ist gruppiert (COMMON, GETTING STARTED, etc.)
✅ Visually ansprechend mit Emojis und Separatoren
✅ Links zu weiterführender Doku

## Testing

### Test 1: New User Experience

```bash
# New user runs help
./innominatus-ctl --help

Expected:
- Sieht nur 9 Commands
- "deploy" is first command
- "tutorial" und "examples" sind prominent
- Hint auf --advanced für mehr
```

### Test 2: Advanced User

```bash
./innominatus-ctl --advanced --help

Expected:
- Sieht validate, analyze, graph-export
- Keine Admin Commands
```

### Test 3: Platform Admin

```bash
./innominatus-ctl --admin --help

Expected:
- Sieht admin commands
- Sieht demo-time, demo-nuke
```

### Test 4: New Helper Commands

```bash
./innominatus-ctl tutorial
# → Shows tutorial link

./innominatus-ctl examples
# → Shows example deployments

./innominatus-ctl docs
# → Opens browser with docs

./innominatus-ctl support
# → Shows support contacts
```

### Test 5: Deploy Alias

```bash
./innominatus-ctl deploy my-app.yaml
# → Same as: run deploy-app my-app.yaml
```

## Success Metrics

**Vorher:**
- ❌ 25+ Commands in Help
- ❌ User overwhelmed
- ❌ Keine Guidance wo zu starten

**Nachher:**
- ✅ 9 Commands in Default Help
- ✅ Klare Gruppierung
- ✅ `tutorial` und `examples` helfen beim Start
- ✅ Progressive Disclosure (--advanced, --admin)

**Impact:**
- Cognitive Load: -70%
- Time to find right command: -80%
- "I don't know what to do" → "Oh, I'll run tutorial!"
