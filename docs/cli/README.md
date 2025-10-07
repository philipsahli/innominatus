# CLI Reference

Command-line interface documentation for innominatus-ctl.

## Overview

**innominatus-ctl** is the command-line interface for deploying applications and managing workloads on your innominatus platform.

### Quick Start

```bash
# Deploy an application
innominatus-ctl deploy my-app.yaml

# Check application status
innominatus-ctl status my-app

# List your applications
innominatus-ctl list

# View logs
innominatus-ctl logs my-app --follow
```

---

## Documentation

| Guide | Description |
|-------|-------------|
| **[Output Formatting](output-formatting.md)** | CLI output formats and styling |
| **[Golden Paths](golden-paths.md)** | Pre-defined workflows and parameters |

---

## Common Commands

### Application Management

```bash
# Deploy application from Score spec
innominatus-ctl deploy score.yaml

# Get application status
innominatus-ctl status <app-name>

# List all deployed applications
innominatus-ctl list

# Delete an application
innominatus-ctl delete <app-name>

# View application logs
innominatus-ctl logs <app-name> [--follow] [--tail N]
```

### Golden Paths

```bash
# List available golden paths
innominatus-ctl list-goldenpaths

# Run a golden path workflow
innominatus-ctl run <golden-path> <score-spec.yaml>

# Examples
innominatus-ctl run deploy-app my-app.yaml
innominatus-ctl run ephemeral-env test-env.yaml
innominatus-ctl run db-lifecycle db-spec.yaml
```

### Environment Management

```bash
# List environments
innominatus-ctl environments

# Validate Score specification
innominatus-ctl validate score.yaml
```

---

## CLI Modes

### User Mode (Default)

For developers deploying applications:

```bash
innominatus-ctl --help
```

Shows common commands:
- deploy
- status
- logs
- list
- delete

### Advanced Mode

For advanced users who need validation and analysis:

```bash
innominatus-ctl --advanced --help
```

Shows advanced commands:
- validate
- analyze
- graph-status
- graph-export
- list-workflows

### Admin Mode

For platform administrators:

```bash
innominatus-ctl --admin --help
```

Shows admin commands:
- admin (user management, API keys)
- environments
- demo-time / demo-status / demo-nuke

---

## Configuration

### Server URL

```bash
# Set platform server URL
innominatus-ctl config set server https://platform.company.com

# Or use environment variable
export INNOMINATUS_SERVER=https://platform.company.com
```

### Authentication

```bash
# Set API key via environment variable
export IDP_API_KEY="your-api-key-here"

# Or via config file
innominatus-ctl config set api-key "your-api-key"
```

---

## Output Formats

innominatus-ctl supports multiple output formats:

```bash
# Default: Pretty-printed table
innominatus-ctl list

# JSON output
innominatus-ctl list --output json

# YAML output
innominatus-ctl list --output yaml

# Silent mode (no output)
innominatus-ctl deploy app.yaml --silent
```

See [Output Formatting](output-formatting.md) for details.

---

## Help & Support

### Built-in Help

```bash
# General help
innominatus-ctl --help

# Command-specific help
innominatus-ctl deploy --help
innominatus-ctl run --help
innominatus-ctl logs --help
```

### Documentation

- **User Guide**: [Getting Started](../user-guide/getting-started.md)
- **Recipes**: [Real-world Examples](../user-guide/recipes/README.md)
- **CLI Reference**: [Full Command Reference](../user-guide/cli-reference.md)

### Platform Support

- Contact your Platform Team
- Check internal documentation portal
- Slack: `#platform-support`

---

## Examples

### Deploy Node.js App with Database

```bash
# Create Score specification
cat > my-app.yaml <<EOF
apiVersion: score.dev/v1b1
metadata:
  name: my-app

containers:
  web:
    image: registry.company.com/my-app:latest
    variables:
      DATABASE_URL: "postgresql://\${resources.db.username}:\${resources.db.password}@\${resources.db.host}:\${resources.db.port}/\${resources.db.name}"

resources:
  db:
    type: postgres
    params:
      version: "15"
      size: small

  route:
    type: route
    params:
      host: my-app.company.com
      port: 3000
EOF

# Deploy
innominatus-ctl deploy my-app.yaml

# Monitor deployment
innominatus-ctl status my-app
innominatus-ctl logs my-app --follow
```

### Run Golden Path Workflow

```bash
# List available golden paths
innominatus-ctl list-goldenpaths

# Deploy application using golden path
innominatus-ctl run deploy-app my-app.yaml

# Create ephemeral environment
innominatus-ctl run ephemeral-env test-env.yaml --param ttl=4h

# Database lifecycle operations
innominatus-ctl run db-lifecycle db-spec.yaml --param operation=backup
```

---

## Advanced Usage

### Custom Workflows

```bash
# Analyze workflow dependencies
innominatus-ctl analyze workflow.yaml

# Export workflow graph
innominatus-ctl graph-export my-app --format dot

# List all workflow executions
innominatus-ctl list-workflows
```

### Debugging

```bash
# Verbose output
innominatus-ctl deploy app.yaml --verbose

# Debug mode
innominatus-ctl deploy app.yaml --debug

# Dry run (validate without deploying)
innominatus-ctl deploy app.yaml --dry-run
```

---

## Next Steps

- **[Golden Paths Guide](golden-paths.md)** - Learn about pre-defined workflows
- **[Output Formatting](output-formatting.md)** - Customize CLI output
- **[User Guide](../user-guide/README.md)** - Complete user documentation
- **[Recipes](../user-guide/recipes/README.md)** - Real-world deployment examples

---

**Questions?** Contact your Platform Team or check the [User Guide](../user-guide/README.md).
