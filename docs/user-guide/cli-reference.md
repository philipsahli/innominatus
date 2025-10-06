# CLI Reference

Complete reference for the `innominatus-ctl` CLI tool.

---

## Global Flags

```bash
--url string       Platform URL (or set INNOMINATUS_URL)
--api-key string   API key (or set INNOMINATUS_API_KEY)
--config string    Config file path (default: ~/.innominatus/config.yaml)
--output string    Output format: table, json, yaml (default: table)
```

---

## Commands

### `deploy`

Deploy an application from Score specification.

```bash
innominatus-ctl deploy <score-file>
```

**Example:**
```bash
innominatus-ctl deploy my-app.yaml
```

---

### `status`

Get application status and deployment information.

```bash
innominatus-ctl status <app-name>
```

**Example:**
```bash
innominatus-ctl status my-app
```

---

### `list`

List all deployed applications.

```bash
innominatus-ctl list
```

**Flags:**
- `--output json` - JSON output
- `--filter team=platform` - Filter by label

---

### `delete`

Delete an application and clean up resources.

```bash
innominatus-ctl delete <app-name>
```

**Example:**
```bash
innominatus-ctl delete my-app
```

---

### `workflows`

List workflow executions for an application.

```bash
innominatus-ctl workflows <app-name>
```

**Example:**
```bash
innominatus-ctl workflows my-app --limit 10
```

---

### `logs`

View application logs.

```bash
innominatus-ctl logs <app-name>
```

**Flags:**
- `--follow` - Stream logs
- `--since 1h` - Show logs from last hour
- `--tail 100` - Show last 100 lines

---

### `list-goldenpaths`

List available golden path workflows.

```bash
innominatus-ctl list-goldenpaths
```

---

### `run`

Run a specific golden path workflow.

```bash
innominatus-ctl run <golden-path> <score-file> [--param key=value]
```

**Examples:**
```bash
# Deploy with standard workflow
innominatus-ctl run deploy-app my-app.yaml

# Create ephemeral environment with 2-hour TTL
innominatus-ctl run ephemeral-env my-app.yaml --param ttl=2h
```

---

### `validate`

Validate a Score specification without deploying.

```bash
innominatus-ctl validate <score-file>
```

**Example:**
```bash
innominatus-ctl validate my-app.yaml
```

---

## Configuration File

Create `~/.innominatus/config.yaml`:

```yaml
url: https://innominatus.yourcompany.com
api_key: your-api-key-here
output: table
```

---

## Environment Variables

```bash
# Platform URL
export INNOMINATUS_URL="https://innominatus.yourcompany.com"

# API Key
export INNOMINATUS_API_KEY="your-api-key"

# Output format
export INNOMINATUS_OUTPUT="json"
```

---

## Examples

### Deploy with custom parameters

```bash
innominatus-ctl run deploy-app my-app.yaml \
  --param environment=production \
  --param replicas=3
```

### Get deployment status in JSON

```bash
innominatus-ctl status my-app --output json | jq .
```

### Stream application logs

```bash
innominatus-ctl logs my-app --follow
```

### List workflows with filter

```bash
innominatus-ctl workflows my-app --status completed
```

---

## Getting Help

```bash
# Global help
innominatus-ctl --help

# Command-specific help
innominatus-ctl deploy --help
```

---

**Next:** [Troubleshooting Guide](troubleshooting.md)
