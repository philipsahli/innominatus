# CLI Reference

Complete reference for the `innominatus-ctl` CLI tool powered by Cobra framework.

---

## Global Flags

Available for all commands:

```bash
--server string          Score orchestrator server URL (default: http://localhost:8081)
--details               Show detailed information including URLs and workflow links
--skip-validation       Skip configuration validation
```

**Examples:**
```bash
innominatus-ctl list --server https://innominatus.company.com
innominatus-ctl status my-app --details
innominatus-ctl delete my-app --skip-validation
```

---

## Built-in Cobra Commands

### `help`

Get help for any command:

```bash
innominatus-ctl --help              # Global help
innominatus-ctl list --help         # Command-specific help
innominatus-ctl workflow --help     # Subcommand help
```

### `completion`

Generate shell completion scripts for bash, zsh, fish, or powershell:

```bash
# Bash
innominatus-ctl completion bash > /etc/bash_completion.d/innominatus-ctl

# Zsh
innominatus-ctl completion zsh > "${fpath[1]}/_innominatus-ctl"

# Fish
innominatus-ctl completion fish > ~/.config/fish/completions/innominatus-ctl.fish

# PowerShell
innominatus-ctl completion powershell > innominatus-ctl.ps1
```

After installation, restart your shell or source the completion file:

```bash
# Bash
source /etc/bash_completion.d/innominatus-ctl

# Zsh
source ~/.zshrc

# Fish
source ~/.config/fish/config.fish
```

---

## Application Management

### `deploy`

Deploy a Score specification directly to the platform.

```bash
innominatus-ctl deploy <score-file.yaml> [flags]
```

**Flags:**
- `-w, --watch` - Stream real-time deployment events and workflow progress
- `--verbose` - Show detailed watch output with timestamps
- `--timeout duration` - Custom timeout (default: 5m0s)

**Examples:**
```bash
# Basic deployment
innominatus-ctl deploy myapp.yaml

# Deploy with real-time watch (recommended)
innominatus-ctl deploy myapp.yaml -w

# Deploy with verbose output and custom timeout
innominatus-ctl deploy myapp.yaml -w --verbose --timeout 10m

# Incremental deployment (add S3 to existing app)
innominatus-ctl deploy myapp-v2.yaml -w
```

**Watch Mode Output:**
```
✅ Score specification deployed: my-app
   📦 Resource detected: db (postgres)
   🔄 Workflow executing: provision-postgres (workflow ID: 123)

🔄 Workflow Executing: provision-postgres
✅ Step 1: create-namespace (completed in 2.3s)
✅ Step 2: create-postgres-cluster (completed in 15.7s)
✅ Step 3: wait-for-database (completed in 23.4s)
✅ Workflow completed successfully

   Resource db: requested → provisioning → active
   📊 View details: http://localhost:8081/resources/5
```

**Notes:**
- Resources are declared in Score specs, not created via CLI commands
- Idempotent: deploying same spec multiple times is safe
- Existing resources are detected and skipped automatically
- Use `-w` flag for real-time feedback (recommended for demos)

---

### `list`

List all deployed applications.

```bash
innominatus-ctl list [--details]
```

**Examples:**
```bash
innominatus-ctl list
innominatus-ctl list --details  # Show URLs and workflow links
```

---

### `status`

Show application status and resources.

```bash
innominatus-ctl status <app-name>
```

**Examples:**
```bash
innominatus-ctl status my-app
innominatus-ctl status web-frontend
```

---

### `delete`

Delete application and all resources completely (removes from database).

```bash
innominatus-ctl delete <app-name>
```

**Examples:**
```bash
innominatus-ctl delete my-app
```

**Note:** This permanently removes the application and all audit trail.

---

### `deprovision`

Deprovision infrastructure but keep audit trail (soft delete).

```bash
innominatus-ctl deprovision <app-name>
```

**Examples:**
```bash
innominatus-ctl deprovision my-app
```

**Note:** This tears down infrastructure but keeps application records for auditing.

---

### `stats`

Show platform statistics (apps, workflows, resources, users).

```bash
innominatus-ctl stats
```

---

### `environments`

List active environments.

```bash
innominatus-ctl environments
```

---

## Workflow Management

### `list-workflows`

List workflow executions (optionally filtered by app).

```bash
innominatus-ctl list-workflows [app-name]
```

**Examples:**
```bash
innominatus-ctl list-workflows                # All workflows
innominatus-ctl list-workflows my-app         # Workflows for my-app
```

---

### `workflow`

Hierarchical parent command for workflow operations.

#### `workflow detail`

Show detailed workflow metadata and step breakdown.

```bash
innominatus-ctl workflow detail <workflow-id>
```

**Examples:**
```bash
innominatus-ctl workflow detail wf_abc123
```

---

#### `workflow logs`

Show workflow execution logs with step details.

```bash
innominatus-ctl workflow logs <workflow-id> [flags]
```

**Flags:**
- `--step <name>` - Show logs for specific step name
- `--step-only` - Only show step logs, skip workflow header
- `--tail <n>` - Number of lines to show from end of logs (0 = all)
- `--verbose` - Show additional metadata

**Examples:**
```bash
innominatus-ctl workflow logs wf_abc123
innominatus-ctl workflow logs wf_abc123 --step "provision-storage"
innominatus-ctl workflow logs wf_abc123 --tail 100
innominatus-ctl workflow logs wf_abc123 --step "deploy-app" --step-only
```

---

### `logs`

Shortcut for `workflow logs` (backward compatibility).

```bash
innominatus-ctl logs <workflow-id> [flags]
```

Same flags as `workflow logs`.

---

### `retry`

Retry failed workflow from first failed step.

```bash
innominatus-ctl retry <workflow-id> <workflow-spec.yaml>
```

**Examples:**
```bash
innominatus-ctl retry wf_abc123 deployment-workflow.yaml
```

---

## Resource Management

### `list-resources`

List resource instances with optional filters.

```bash
innominatus-ctl list-resources [app-name] [flags]
```

**Flags:**
- `--type <type>` - Filter by resource type (e.g., postgres, redis, s3)
- `--state <state>` - Filter by state (e.g., active, provisioning, failed)

**Examples:**
```bash
innominatus-ctl list-resources                           # All resources
innominatus-ctl list-resources my-app                    # Resources for my-app
innominatus-ctl list-resources --type postgres           # All postgres resources
innominatus-ctl list-resources my-app --state active     # Active resources for my-app
innominatus-ctl list-resources --type redis --state provisioning
```

---

### `resource`

Manage resource instances (parent command for resource operations).

```bash
innominatus-ctl resource [subcommands]
```

---

## Graph Visualization

### `graph-export`

Export workflow graph visualization in multiple formats.

```bash
innominatus-ctl graph-export <app-name> [flags]
```

**Flags:**
- `--format <format>` - Output format (default: svg)
  - `svg` - Scalable Vector Graphics
  - `png` - PNG image
  - `dot` - GraphViz DOT format
  - `json` - Enhanced JSON with timing metadata (NEW)
  - `mermaid` - Mermaid flowchart diagram (NEW)
  - `mermaid-state` - Mermaid state diagram (NEW)
  - `mermaid-gantt` - Mermaid Gantt timeline (NEW)
- `--output <path>` - Output file path (default: stdout)

**Examples:**
```bash
# Traditional formats
innominatus-ctl graph-export my-app                          # SVG to stdout
innominatus-ctl graph-export my-app --format png             # PNG to stdout
innominatus-ctl graph-export my-app --output graph.svg       # SVG to file
innominatus-ctl graph-export my-app --format dot --output graph.dot

# NEW: Enhanced formats with timing and visualization
innominatus-ctl graph-export my-app --format json --output graph.json
innominatus-ctl graph-export my-app --format mermaid --output flowchart.mmd
innominatus-ctl graph-export my-app --format mermaid-state --output state.mmd
innominatus-ctl graph-export my-app --format mermaid-gantt --output timeline.mmd
```

**Format Details:**
- **json**: Includes node timing (started_at, completed_at, duration), metadata, and complete graph structure
- **mermaid**: Flowchart with color-coded nodes by state (running/succeeded/failed)
- **mermaid-state**: State transition diagram showing workflow progression
- **mermaid-gantt**: Timeline chart with execution durations and dependencies

---

### `graph-status`

Show workflow graph status and statistics.

```bash
innominatus-ctl graph-status <app-name>
```

**Output Includes:**
- Total nodes and breakdown by type (workflow, step, resource)
- Node states (running, succeeded, failed, waiting)
- Total edges and relationship types
- Execution timing statistics (if available)

**Examples:**
```bash
innominatus-ctl graph-status my-app

# Expected output:
# Graph Status for Application: my-app
#
# Total Nodes: 12
#
# Node Counts by Type:
#   workflow: 1
#   step: 8
#   resource: 3
#
# Node Counts by State:
#   succeeded: 10
#   running: 1
#   waiting: 1
#
# Total Edges: 11
```

---

## Golden Paths

### `list-goldenpaths`

List available golden path workflows.

```bash
innominatus-ctl list-goldenpaths
```

---

### `run`

Run a golden path workflow.

```bash
innominatus-ctl run <golden-path-name> [score-spec.yaml] [flags]
```

**Flags:**
- `--param <key=value>` - Parameter override (can be used multiple times)

**Examples:**
```bash
# Run golden path with Score spec
innominatus-ctl run deploy-app my-app.yaml

# Run with custom parameters
innominatus-ctl run deploy-app my-app.yaml \
  --param environment=production \
  --param replicas=3

# Run without Score spec (golden path provides defaults)
innominatus-ctl run ephemeral-env \
  --param ttl=2h \
  --param namespace=test-env-123
```

---

## Validation & Analysis

### `validate`

Validate Score spec locally (no server required).

```bash
innominatus-ctl validate <score-spec.yaml> [flags]
```

**Flags:**
- `--explain` - Show detailed validation explanations
- `--format <format>` - Output format: text, json, simple (default: text)

**Examples:**
```bash
innominatus-ctl validate my-app.yaml
innominatus-ctl validate my-app.yaml --explain
innominatus-ctl validate my-app.yaml --format json
```

---

### `analyze`

Analyze Score spec workflow dependencies (no server required).

```bash
innominatus-ctl analyze <score-spec.yaml>
```

**Examples:**
```bash
innominatus-ctl analyze my-app.yaml
```

Shows dependency graph and potential issues.

---

## Authentication

### `login`

Authenticate and store API key locally.

```bash
innominatus-ctl login
```

Prompts for username and password, retrieves API key from server, and stores it in credentials file.

**Note:** You can also set `IDP_API_KEY` environment variable to bypass login.

---

### `logout`

Remove stored credentials.

```bash
innominatus-ctl logout
```

Removes API key from credentials file.

---

## Demo Environment

**Note:** These commands are for local development/demo only. They install demo services (Gitea, ArgoCD, Vault, Minio) to Docker Desktop Kubernetes.

### `demo-time`

Install/reconcile demo environment.

```bash
innominatus-ctl demo-time [flags]
```

**Flags:**
- `--component <components>` - Comma-separated list of components to install

**Examples:**
```bash
innominatus-ctl demo-time                           # Install all demo components
innominatus-ctl demo-time --component gitea,vault   # Install only Gitea and Vault
```

---

### `demo-status`

Check demo environment health and status.

```bash
innominatus-ctl demo-status
```

Shows status of Gitea, ArgoCD, Vault, Minio, and other demo services.

---

### `demo-reset`

Reset database to clean state (deletes all data).

```bash
innominatus-ctl demo-reset
```

**Warning:** This permanently deletes all applications, workflows, and resources from the database.

---

### `demo-nuke`

Uninstall and clean demo environment.

```bash
innominatus-ctl demo-nuke
```

Removes all demo services from Kubernetes cluster.

---

### `fix-gitea-oauth`

Fix Gitea OAuth2 auto-registration with Keycloak.

```bash
innominatus-ctl fix-gitea-oauth
```

---

## Administration

### `admin`

Admin commands (requires admin role).

```bash
innominatus-ctl admin [subcommands]
```

---

### `team`

Team management commands.

```bash
innominatus-ctl team [subcommands]
```

---

### `provider`

Provider management commands.

```bash
innominatus-ctl provider [subcommands]
```

---

## Chat (Experimental)

### `chat`

Interactive AI assistant chat (not yet implemented).

```bash
innominatus-ctl chat
```

---

## Authentication Flow

The CLI uses automatic authentication for server commands:

1. **Check for API key**: First checks `IDP_API_KEY` environment variable
2. **Check credentials file**: If no env var, checks stored credentials
3. **Prompt for login**: If no credentials found, prompts for username/password
4. **Store credentials**: After successful login, stores API key in credentials file

**Commands that skip authentication** (local-only):
- `run`, `validate`, `analyze`
- `demo-time`, `demo-nuke`, `demo-status`, `demo-reset`, `fix-gitea-oauth`
- `login`, `logout`, `chat`
- `help`, `completion`

**Environment Variables:**
```bash
export IDP_API_KEY="your-api-key-here"  # Skip login prompt
```

---

## Configuration Validation

By default, the CLI runs fast configuration validation before server commands. To skip:

```bash
innominatus-ctl list --skip-validation
```

---

## Shell Completion Features

After installing shell completion, you get:

- **Command completion**: Tab to see available commands
- **Flag completion**: Tab after `--` to see available flags
- **Subcommand completion**: Tab after parent commands (e.g., `workflow <tab>`)
- **Help text**: Completion shows command descriptions

**Example (zsh):**
```bash
$ innominatus-ctl <tab>
admin           -- Admin commands (requires admin role)
analyze         -- Analyze Score spec workflow dependencies
chat            -- Interactive AI assistant chat
completion      -- Generate the autocompletion script for the specified shell
delete          -- Delete application and all resources completely
...
```

---

## Examples

### Deploy application with golden path
```bash
innominatus-ctl run deploy-app my-app.yaml
```

### Check workflow execution logs
```bash
innominatus-ctl workflow logs wf_abc123 --verbose
```

### View specific step logs
```bash
innominatus-ctl workflow logs wf_abc123 --step "provision-storage" --step-only
```

### List resources with filters
```bash
innominatus-ctl list-resources my-app --type postgres --state active
```

### Export workflow graph
```bash
innominatus-ctl graph-export my-app --format png --output workflow-graph.png
```

### Validate Score spec before deploying
```bash
innominatus-ctl validate my-app.yaml --explain
```

### Get detailed application status
```bash
innominatus-ctl status my-app --details
```

### Retry failed workflow
```bash
innominatus-ctl retry wf_abc123 deployment-workflow.yaml
```

---

## Troubleshooting

### Authentication issues

```bash
# Clear stored credentials
innominatus-ctl logout

# Set API key manually
export IDP_API_KEY="your-api-key"

# Re-login
innominatus-ctl login
```

### Connection issues

```bash
# Specify custom server URL
innominatus-ctl list --server https://innominatus.company.com

# Skip validation if configuration is incomplete
innominatus-ctl list --skip-validation
```

### Command not found

Make sure `innominatus-ctl` is in your PATH or use the full path:

```bash
./innominatus-ctl list
```

---

**Next:** [Troubleshooting Guide](troubleshooting.md)
