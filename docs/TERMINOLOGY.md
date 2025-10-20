# innominatus Terminology Guide

**Last Updated:** 2025-10-19

This guide defines the standard terminology used across the innominatus platform orchestrator (API, CLI, and Web UI). Understanding these terms ensures consistent communication and reduces confusion when working with the platform.

---

## Core Concepts

### Score Specification (Score Spec)

**Definition:** A YAML file following the [Score specification](https://score.dev) format that describes a workload's containers, resources, and configuration.

**When to use:**
- Referring to YAML files
- Documentation about file format
- Validation and analysis operations
- File upload contexts

**Examples:**
- "Upload your `score-spec.yaml` file"
- "Validate Score specification syntax"
- "The Score spec defines containers and resources"

**Related Terms:**
- Score file, spec file, workload specification

**API Usage:** `/api/specs` (historical naming)

**CLI Usage:** `validate <score-spec.yaml>`

**UI Usage:** "Upload Spec" button, "Spec Analysis"

---

### Application

**Definition:** A deployed instance created from a Score specification. An application represents the running workload with associated infrastructure, state, and lifecycle.

**When to use:**
- Referring to deployed instances
- Runtime operations (start, stop, delete)
- Operational status and monitoring
- Application lifecycle management

**Examples:**
- "Application 'my-api' is running in staging"
- "Delete application 'frontend-app'"
- "Application has 3 active resources"

**Related Terms:**
- Deployed app, workload instance

**API Usage:** `/api/applications/{name}` (preferred), `/api/specs` (legacy)

**CLI Usage:** `innominatus-ctl list`, `innominatus-ctl status <name>`, `innominatus-ctl delete <name>`

**UI Usage:** "Applications" page, application list

**State:**
- Applications have lifecycle state (running, failed, pending)
- Applications are created from Score specs
- Applications own resources and workflows

---

### App

**Definition:** Shortened form of "Application" used primarily in UI contexts where space is limited.

**When to use:**
- UI labels and headings
- Compact displays (mobile, tables)
- Navigation menus
- URL parameters

**Examples:**
- "Apps" navigation link
- "View app details"
- `/apps` page route

**Related Terms:**
- Application (formal)

**API Usage:** Query parameters: `?app=my-api`

**CLI Usage:** Occasionally in help text

**UI Usage:** Throughout UI for brevity

**Note:** "App" and "Application" are interchangeable in user-facing contexts. Use "Application" in documentation and formal contexts.

---

### Workflow

**Definition:** An ordered sequence of steps that provision, configure, or manage infrastructure and applications. Workflows are the execution engine of innominatus.

**When to use:**
- Describing provisioning processes
- Deployment sequences
- Infrastructure orchestration
- Execution tracking

**Examples:**
- "The deploy workflow has 5 steps"
- "Workflow execution failed at step 3"
- "View workflow logs"

**Components:**
- **Workflow Execution:** A specific run of a workflow (has ID, status, timestamps)
- **Workflow Definition:** The template defining steps (YAML file)
- **Workflow Step:** Individual operation within a workflow

**Types:**
- Platform workflows (in `workflows/platform/`)
- Product workflows (in `workflows/product/`)
- Golden path workflows (curated templates)

**API Usage:** `/api/workflows`, `/api/workflows/{id}`

**CLI Usage:** `innominatus-ctl list-workflows`, `innominatus-ctl logs <workflow-id>`

**UI Usage:** "Workflows" page, workflow execution list

---

### Golden Path

**Definition:** A pre-defined, tested, and curated workflow template for common scenarios. Golden paths represent best practices and are the recommended way to perform operations.

**When to use:**
- Referring to curated workflow templates
- Recommended deployment approaches
- Best-practice workflows

**Examples:**
- "Use the `deploy-app` golden path"
- "Available golden paths: deploy-app, ephemeral-env, database-provision"
- "Golden paths are maintained by the platform team"

**Characteristics:**
- Tested and validated
- Follow organizational best practices
- Include all necessary steps
- Have parameter definitions
- Maintained centrally

**API Usage:** `/api/workflows/golden-paths/{name}/execute`

**CLI Usage:** `innominatus-ctl list-goldenpaths`, `innominatus-ctl run <path-name>`

**UI Usage:** "Golden Paths" page, golden path catalog

**Location:** Workflow YAML files in `workflows/golden-paths/`

---

### Resource

**Definition:** A provisioned infrastructure component such as a database, storage bucket, message queue, or any managed service. Resources are defined in Score specs and provisioned by workflows.

**When to use:**
- Infrastructure components
- Managed services (databases, storage)
- Dependencies of applications

**Examples:**
- "PostgreSQL resource is healthy"
- "Application requires 3 resources"
- "Resource provisioning failed"

**Types:**
- **Native Resources:** Built-in (postgres, redis, s3, volume)
- **Delegated Resources:** Provisioned by external systems
- **Custom Resources:** Defined by platform team

**Properties:**
- Resource Name: Logical name (e.g., `main-db`)
- Resource Type: Type identifier (e.g., `postgres`)
- Resource State: Lifecycle state (active, provisioning, failed)
- Resource Health: Health status (healthy, unhealthy, unknown)

**API Usage:** `/api/resources`, `/api/resources/{id}`

**CLI Usage:** `innominatus-ctl list-resources`, `innominatus-ctl environments`

**UI Usage:** "Resources" page, resource list and details

---

### Environment

**Definition:** A deployment target representing a stage in the software lifecycle (development, staging, production). Environments have different configurations and policies.

**When to use:**
- Deployment targeting
- Environment-specific configuration
- Promotion workflows

**Examples:**
- "Deploy to staging environment"
- "Production environment requires approval"
- "List available environments"

**Common Environments:**
- `development` / `dev` - Local or development cluster
- `staging` - Pre-production testing
- `production` / `prod` - Production workloads
- `ephemeral` - Temporary test environments

**Properties:**
- Environment Type: Name/identifier
- Configuration: Environment-specific settings
- Policies: Deployment rules and constraints

**API Usage:** `/api/environments`

**CLI Usage:** `innominatus-ctl environments`

**UI Usage:** Environment selector, application environment field

---

### Provisioner

**Definition:** The infrastructure automation tool used to provision and manage resources. Provisioners execute workflow steps.

**Supported Provisioners:**
- **Terraform:** Infrastructure as Code
- **Ansible:** Configuration management
- **Kubernetes:** Container orchestration
- **Helm:** Kubernetes package management
- **Script:** Custom shell scripts

**When to use:**
- Workflow step definitions
- Infrastructure automation contexts
- Technical documentation

**Examples:**
- "Terraform provisioner manages cloud resources"
- "Kubernetes provisioner deploys containers"
- "Step type: terraform"

**API Usage:** Workflow step configurations

**CLI Usage:** Workflow step output in logs

**UI Usage:** Workflow step type display

---

## Workflow-Related Terms

### Workflow Execution

**Definition:** A specific run of a workflow with a unique ID, status, timestamps, and logs.

**Properties:**
- Execution ID (integer)
- Application Name
- Workflow Name
- Status (pending, running, completed, failed)
- Start/End timestamps
- Step execution details

---

### Workflow Step

**Definition:** An individual operation within a workflow. Steps execute sequentially or in parallel.

**Properties:**
- Step Number (execution order)
- Step Name (identifier)
- Step Type (terraform, ansible, kubernetes, script)
- Status (pending, running, completed, failed)
- Logs (stdout/stderr output)
- Configuration (step-specific parameters)

**Execution Modes:**
- Sequential (default)
- Parallel (when `parallel: true`)
- Conditional (when `condition` specified)

---

### Step Type

**Definition:** The category of operation a workflow step performs.

**Available Types:**
- `terraform` - Terraform operations (init, plan, apply, destroy)
- `ansible` - Ansible playbook execution
- `kubernetes` - Kubernetes manifest application
- `helm` - Helm chart operations
- `script` - Shell script execution

---

## API-Specific Terms

### Endpoint

**Definition:** An API URL path that provides a specific function.

**Format:** `[HTTP METHOD] /api/[resource]/[identifier]`

**Examples:**
- `GET /api/specs` - List applications
- `POST /api/workflows/golden-paths/deploy-app/execute` - Execute golden path
- `DELETE /api/applications/{name}` - Delete application

---

### Authentication Token (API Key)

**Definition:** A secret credential used to authenticate API requests.

**Types:**
- **Session Token:** Temporary, issued on login
- **API Key:** Long-lived, generated in profile

**Usage:**
```
Authorization: Bearer <token>
```

**Management:**
- Generate in UI profile page or via `login` command
- Revoke in UI security tab or via admin commands
- Expires after configured duration (default: 90 days)

---

## CLI-Specific Terms

### Command

**Definition:** A CLI operation invoked via `innominatus-ctl <command>`.

**Examples:**
- `innominatus-ctl list` - List applications
- `innominatus-ctl logs <workflow-id>` - View workflow logs
- `innominatus-ctl admin show` - Show admin configuration

---

### Flag

**Definition:** A command-line option that modifies command behavior.

**Format:** `--flag-name` or `--flag-name value`

**Examples:**
- `--details` - Show detailed information
- `--format json` - Output in JSON format
- `--param key=value` - Pass parameter

**Global Flags:**
- `--server <url>` - Server URL (default: http://localhost:8081)
- `--details` - Show detailed information
- `--skip-validation` - Skip configuration validation

---

## UI-Specific Terms

### Page

**Definition:** A route/view in the web UI.

**Examples:**
- Dashboard (`/dashboard`)
- Applications (`/apps`)
- Workflows (`/workflows`)
- Profile (`/profile`)

---

### Details Pane

**Definition:** A slide-out panel showing detailed information about a selected item.

**Used For:**
- Application details
- Resource details
- Workflow execution details

---

## Admin-Specific Terms

### Team

**Definition:** A group of users with shared access to applications and resources.

**Properties:**
- Team Name
- Team Members
- Access Policies

**Usage:**
- Applications belong to teams
- Users belong to teams
- Team-based access control

---

### Role

**Definition:** User permission level.

**Available Roles:**
- `user` - Standard user (can deploy, view own team's resources)
- `admin` - Administrator (full platform access)

---

### Impersonation

**Definition:** Admin feature to view the platform as another user for debugging.

**Usage:**
- Admin starts impersonation
- UI shows "Impersonating [username]" banner
- Admin sees what the user sees
- Admin stops impersonation to return to normal

---

## Demo-Specific Terms

### Demo Environment

**Definition:** A complete local deployment of all integrated services for testing and demonstration.

**Components:**
- Gitea (Git server)
- ArgoCD (GitOps deployment)
- Vault (Secrets management)
- Minio (Object storage)
- Keycloak (Identity provider)
- Prometheus (Metrics)
- Grafana (Dashboards)
- Backstage (Developer portal)

**Commands:**
- `demo-time` - Install demo
- `demo-status` - Check health
- `demo-nuke` - Remove demo

---

## State and Status Terms

### Application State

Applications don't have explicit state in the current implementation. State is inferred from:
- Latest workflow execution status
- Resource health status
- Deployment existence

### Workflow Status

- **pending** - Queued, not started
- **running** - Currently executing
- **completed** - Successfully finished
- **failed** - Encountered error

### Resource State

- **active** - Successfully provisioned and available
- **provisioning** - Being created
- **failed** - Provisioning failed
- **deprovisioning** - Being removed
- **deprovisioned** - Removed, audit trail kept

### Resource Health

- **healthy** - Passing health checks
- **unhealthy** - Failing health checks
- **unknown** - Health status not available

---

## Cross-Reference Table

| Concept | API Term | CLI Term | UI Term | Storage |
|---------|----------|----------|---------|---------|
| Deployed instance | specs (legacy), applications | application | app | specs table |
| Execution sequence | workflow | workflow | workflow | workflow_executions |
| Infrastructure component | resource | resource | resource | resource_instances |
| Target environment | environment | environment | environment | environments table |
| User group | team | team | team | users.yaml / teams table |
| Access level | role | role | role | users.yaml |
| Curated template | golden path | golden path | golden path | workflows/golden-paths/ |
| Auth credential | API key / token | API key / token | API key | api_keys table |

---

## Common Confusion Points

### ❓ "Spec" vs "Application" vs "App"

**Answer:**
- **Score Spec** = The YAML file definition
- **Application** = The deployed instance from that spec
- **App** = Short form of application (UI only)

**Rule of Thumb:**
- File context → "Score spec"
- Runtime context → "Application"
- UI label → "App"

---

### ❓ "Workflow" vs "Golden Path"

**Answer:**
- **Workflow** = Any sequence of provisioning steps
- **Golden Path** = A curated, tested workflow template maintained by platform team

**All golden paths are workflows, but not all workflows are golden paths.**

---

### ❓ "Resource" vs "Container"

**Answer:**
- **Resource** = Infrastructure dependency (database, storage, etc.)
- **Container** = Application runtime (Docker container)

**Resources are provisioned FOR containers. Containers use resources.**

---

### ❓ "Environment" vs "Namespace"

**Answer:**
- **Environment** = Logical deployment stage (dev, staging, prod)
- **Namespace** = Kubernetes isolation boundary

**Environments may map to Kubernetes namespaces, but they're conceptually different.**

---

### ❓ "/api/specs" vs "/api/applications"

**Answer:**
This is a known inconsistency (see Gap Analysis P0-1).

**Current State:**
- `GET /api/specs` - List applications (legacy)
- `GET /api/specs/{name}` - Get application (legacy)
- `DELETE /api/applications/{name}` - Delete application (correct)

**Future State:** All endpoints will use `/api/applications` consistently.

**For now:** Use `/api/specs` for GET operations, `/api/applications` for DELETE/POST operations.

---

## Best Practices

### Documentation Writing

✅ **DO:**
- Use "Score specification" or "Score spec" for YAML files
- Use "application" for deployed instances
- Use "workflow execution" when referring to specific runs
- Use "golden path" for curated templates
- Be consistent within a single document

❌ **DON'T:**
- Mix "spec" and "application" in the same context
- Use "app" in formal documentation
- Use "workflow" and "golden path" interchangeably
- Assume readers know internal terminology

### Code Comments

✅ **DO:**
```go
// ListApplications returns all deployed applications (created from Score specs)
func (s *Server) ListApplications() ([]Application, error)
```

❌ **DON'T:**
```go
// ListSpecs returns specs
func (s *Server) ListSpecs() ([]Spec, error)  // Vague, mixes terminology
```

### User-Facing Messages

✅ **DO:**
- "Application 'my-api' deployed successfully"
- "Workflow execution completed in 45 seconds"
- "Resource 'postgres-db' is healthy"

❌ **DON'T:**
- "Spec created" (deployed what?)
- "WF done" (abbreviations in user messages)
- "DB OK" (technical jargon without context)

---

## Glossary Quick Reference

| Term | Short Definition |
|------|------------------|
| **Score Spec** | YAML file defining workload |
| **Application** | Deployed instance from Score spec |
| **App** | Short for application (UI only) |
| **Workflow** | Sequence of provisioning steps |
| **Golden Path** | Curated workflow template |
| **Resource** | Infrastructure component (DB, storage) |
| **Environment** | Deployment target (dev, staging, prod) |
| **Provisioner** | Tool executing steps (Terraform, Ansible) |
| **Execution** | Specific workflow run |
| **Step** | Individual operation in workflow |
| **Endpoint** | API URL providing functionality |
| **API Key** | Authentication credential |
| **Team** | User group with shared access |
| **Role** | User permission level (user, admin) |
| **State** | Lifecycle status of resource |
| **Health** | Operational status of resource |

---

## Additional Resources

- [Score Specification](https://score.dev/docs/score-specification)
- [API Documentation](/swagger-user)
- [CLI Reference](./user-guide/cli-reference.md)
- [Architecture Overview](./development/architecture.md)
- [Gap Analysis](./API_CLI_UI_GAP_ANALYSIS.md)

---

**Document Version:** 1.0
**Last Updated:** 2025-10-19
