# TFE and Git-based Workflow Extension

This extension adds support for external provisioning via Terraform Enterprise (TFE) and Git-based workflows to the Score Orchestrator.

## New Workflow Step Types

### 1. `terraform-generate`
Generates Terraform files locally based on resource information.

**Required fields:**
- `resource`: Name of the resource to generate Terraform for
- `outputDir`: Directory to write the generated files

**Example:**
```yaml
- name: generate-tf
  type: terraform-generate
  resource: postgres
  outputDir: ./tmp-tf
```

### 2. `git-pr`
Clones a repository, creates a branch, commits files, pushes, and opens a pull request.

**Required fields:**
- `repo`: Git repository URL
- `branch`: Branch name to create
- `commitMessage`: Commit message for the changes

**Example:**
```yaml
- name: push-to-git
  type: git-pr
  repo: git@github.com:org/infrastructure.git
  branch: feature/provision-db
  commitMessage: "Provision new Postgres DB via Orchestrator"
```

### 3. `git-check-pr`
Polls GitHub API until the pull request is merged.

**Required fields:**
- `repo`: Git repository URL
- `branch`: Branch name to monitor

**Example:**
```yaml
- name: wait-for-merge
  type: git-check-pr
  repo: git@github.com:org/infrastructure.git
  branch: feature/provision-db
```

### 4. `tfe-status`
Polls Terraform Enterprise API for workspace run status until completion, then displays terraform outputs.

**Required fields:**
- `workspace`: TFE workspace name

**Required environment variables:**
- `TFE_TOKEN`: Terraform Cloud API token
- `TFE_ORGANIZATION`: TFE organization name

**Features:**
- Polls TFE workspace every 5 seconds until run completes
- On successful completion, fetches and displays all terraform outputs
- Sensitive outputs are masked as `<sensitive>`
- Non-sensitive outputs show their actual values

**Example:**
```yaml
- name: wait-for-tfe
  type: tfe-status
  workspace: db-workspace
```

**Example output when TFE run completes:**
```
TFE workspace 'db-workspace' run completed successfully

Terraform outputs:
  postgres_connection_string: <sensitive>
  postgres_endpoint: postgres.example.com:5432
  postgres_id: rds-postgres-12345
  postgres_status: available
```

## Prerequisites

Before using TFE workflows, ensure you have:

1. **GitHub CLI (`gh`)** installed and authenticated
2. **Git** configured with access to target repositories
3. **TFE_TOKEN** environment variable set with your Terraform Cloud API token
4. **TFE_ORGANIZATION** environment variable set with your organization name

## Example Usage

### 1. Complete TFE Workflow Example

See `example-tfe-workflow.yaml` for a complete workflow that:
1. Generates Terraform files for a Postgres resource
2. Creates a pull request with the files
3. Waits for the PR to be merged
4. Monitors the TFE workspace for completion

### 2. Running the Example

```bash
# Build the demo
go build -o tfe-workflow-demo example-tfe-main.go types.go workflow.go

# Set up environment (replace with your values)
export TFE_TOKEN=your-terraform-cloud-token
export TFE_ORGANIZATION=your-org-name

# Run the demo
./tfe-workflow-demo example-tfe-workflow.yaml
```

### 3. Integration with Existing Server

The new workflow types are automatically available in the existing server. Just include them in your Score specs:

```bash
# Deploy a spec with TFE workflow
./innominatus-ctl deploy example-tfe-workflow.yaml
```

## Implementation Details

- **MVP Implementation**: Basic error handling with immediate failure on errors
- **Polling Strategy**: 5-second intervals with 10-minute timeouts
- **Workspace Isolation**: Each app-environment combination gets its own workspace
- **Spinner Integration**: Progress feedback for long-running operations
- **Authentication**: Assumes pre-configured GitHub and TFE credentials

## Architecture

The extension follows the existing workflow pattern:
1. New step types added to `Step` struct in `types.go`
2. Handler functions implemented in `workflow.go`
3. Switch statement updated in `runStepWithSpinner()`
4. Maintains compatibility with existing workflow types

## Error Handling

Workflows stop on first failure with clear error messages:
- Missing required fields
- Authentication failures
- Network timeouts
- Git/GitHub operations
- TFE API errors

## Future Enhancements

Potential improvements for production use:
- Retry logic with exponential backoff
- Configurable timeouts and polling intervals
- Support for multiple TFE organizations
- Enhanced Terraform template generation
- Webhook-based status updates instead of polling