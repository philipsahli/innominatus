# E2E Test Implementation: Demo Environment Workflows

## Objective
Implement comprehensive UI and End-to-End tests covering the full lifecycle of the demo environment and typical user workflows. Tests should verify both CLI/API (Go tests) and Web UI (Node/TypeScript tests) functionality.

## Testing Approach

Follow the project's **verification-first** principle:
1. Write verification test first
2. Implement/verify feature
3. Run verification
4. Iterate until pass

Apply **KISS** (Keep It Simple):
- Use standard testing tools (Go testing, Playwright/Cypress)
- Clear test names describing what is being verified
- Minimal mocking - test against real demo environment where possible

## Test Scenarios

### 1. Setup Demo Environment
**Scope**: Verify `demo-time` command provisions all required services

**Go Tests** (`cmd/cli/commands/demo_test.go`):
```go
func TestDemoTime_ProvisionsAllServices(t *testing.T) {
  // Verify: Gitea, ArgoCD, Vault, Minio, Grafana are deployed
  // Verify: All services are healthy
  // Verify: Ingress routes are configured
}

func TestDemoTime_HandlesExistingInstallation(t *testing.T) {
  // Verify: Idempotent - can run multiple times
  // Verify: Detects and skips existing services
}
```

**Web UI Tests** (`web-ui/tests/e2e/demo-setup.spec.ts`):
```typescript
test('demo environment health visible in UI', async ({ page }) => {
  // Navigate to dashboard
  // Verify demo services status shown
  // Verify service URLs accessible
});
```

**Acceptance Criteria**:
- ✅ All demo services deployed successfully
- ✅ `demo-status` shows all services healthy
- ✅ Web UI displays demo environment status
- ✅ All service URLs respond (Gitea, ArgoCD, Vault, Minio, Grafana)

---

### 2. Team Setup
**Scope**: Verify team creation and user management

**Go Tests** (`internal/server/handlers_test.go`):
```go
func TestCreateTeam_Success(t *testing.T) {
  // Create team via API
  // Verify team exists in database
  // Verify team appears in list
}

func TestAddUserToTeam_Success(t *testing.T) {
  // Create team and user
  // Add user to team
  // Verify user has team access
}
```

**Web UI Tests** (`web-ui/tests/e2e/teams.spec.ts`):
```typescript
test('create team and add members', async ({ page }) => {
  // Login as admin
  // Navigate to teams page
  // Create new team
  // Add team members
  // Verify team appears in list
  // Verify members have access
});
```

**Acceptance Criteria**:
- ✅ Team created via API and CLI
- ✅ Users can be added/removed from teams
- ✅ Web UI shows team management interface
- ✅ Team permissions enforced in API

---

### 3. Provider Add
**Scope**: Verify adding external providers (Kubernetes, cloud platforms)

**Go Tests** (`internal/database/database_test.go`):
```go
func TestAddProvider_Kubernetes(t *testing.T) {
  // Add Kubernetes provider with kubeconfig
  // Verify provider stored in database
  // Verify provider validation succeeds
}

func TestAddProvider_InvalidCredentials(t *testing.T) {
  // Attempt to add provider with invalid credentials
  // Verify error returned
  // Verify provider not stored
}
```

**Web UI Tests** (`web-ui/tests/e2e/providers.spec.ts`):
```typescript
test('add and configure provider', async ({ page }) => {
  // Navigate to providers page
  // Click "Add Provider"
  // Fill provider details (name, type, credentials)
  // Submit form
  // Verify provider appears in list
  // Verify connection test passes
});
```

**Acceptance Criteria**:
- ✅ Providers can be added via API/CLI/UI
- ✅ Provider credentials validated before storage
- ✅ Provider list shows all configured providers
- ✅ Provider connection test verifies connectivity

---

### 4. Deploy Resource
**Scope**: Verify deploying applications via golden paths

**Go Tests** (`internal/workflow/executor_test.go`):
```go
func TestDeployApp_ViaGoldenPath(t *testing.T) {
  // Load score spec
  // Execute deploy-app golden path
  // Verify workflow execution succeeds
  // Verify resources created (namespace, deployment, service)
}

func TestDeployApp_WithTerraform(t *testing.T) {
  // Execute golden path with Terraform step
  // Verify terraform init/plan/apply executed
  // Verify resources created
}
```

**Web UI Tests** (`web-ui/tests/e2e/deploy.spec.ts`):
```typescript
test('deploy application via golden path', async ({ page }) => {
  // Navigate to golden paths
  // Select "deploy-app"
  // Upload score-spec.yaml
  // Start deployment
  // Monitor workflow execution
  // Verify deployment succeeds
  // Verify application appears in list
});

test('view deployment logs and status', async ({ page }) => {
  // Navigate to application details
  // View workflow execution logs
  // Verify all steps show success
  // Verify resource status visible
});
```

**Acceptance Criteria**:
- ✅ Application deployed via CLI golden path
- ✅ Application deployed via Web UI
- ✅ Workflow execution tracked in database
- ✅ Step logs captured and visible
- ✅ Resources created in target environment

---

### 5. Lifecycle
**Scope**: Verify full application lifecycle (update, scale, configure)

**Go Tests** (`internal/workflow/lifecycle_test.go`):
```go
func TestUpdateApp_ViaGoldenPath(t *testing.T) {
  // Deploy initial version
  // Update score spec (e.g., change replicas)
  // Execute update golden path
  // Verify changes applied
}

func TestScaleApp_Success(t *testing.T) {
  // Deploy app with 1 replica
  // Scale to 3 replicas
  // Verify scaling applied
}
```

**Web UI Tests** (`web-ui/tests/e2e/lifecycle.spec.ts`):
```typescript
test('update application configuration', async ({ page }) => {
  // Navigate to existing application
  // Click "Update"
  // Modify configuration (replicas, resources)
  // Apply changes
  // Verify workflow executes
  // Verify changes reflected in UI
});

test('view application history', async ({ page }) => {
  // Navigate to application details
  // View deployment history
  // Verify all past deployments shown
  // Compare configurations between versions
});
```

**Acceptance Criteria**:
- ✅ Application updates tracked as new workflow executions
- ✅ Configuration changes applied correctly
- ✅ History shows all past deployments
- ✅ Rollback capability (if implemented)

---

### 6. Destroy
**Scope**: Verify resource cleanup and deletion

**Go Tests** (`internal/workflow/destroy_test.go`):
```go
func TestDestroyApp_RemovesAllResources(t *testing.T) {
  // Deploy application
  // Execute destroy workflow
  // Verify all resources removed (namespace, deployments, services)
  // Verify database records marked as deleted
}

func TestDestroyApp_HandlesMissingResources(t *testing.T) {
  // Deploy application
  // Manually delete some resources
  // Execute destroy workflow
  // Verify graceful handling of missing resources
}
```

**Web UI Tests** (`web-ui/tests/e2e/destroy.spec.ts`):
```typescript
test('destroy application via UI', async ({ page }) => {
  // Navigate to application details
  // Click "Destroy"
  // Confirm deletion
  // Monitor workflow execution
  // Verify application removed from list
  // Verify resources cleaned up
});

test('destroy shows confirmation dialog', async ({ page }) => {
  // Navigate to application details
  // Click "Destroy"
  // Verify confirmation dialog appears
  // Verify dialog shows resource count
  // Cancel and verify nothing destroyed
});
```

**Acceptance Criteria**:
- ✅ Destroy workflow removes all created resources
- ✅ Database records marked as deleted (soft delete)
- ✅ UI shows confirmation before destruction
- ✅ Graceful handling of partially destroyed resources

---

### 7. Demo Nuke
**Scope**: Verify complete demo environment cleanup

**Go Tests** (`cmd/cli/commands/demo_test.go`):
```go
func TestDemoNuke_RemovesAllServices(t *testing.T) {
  // Setup demo environment
  // Execute demo-nuke
  // Verify all demo services removed
  // Verify namespaces deleted
  // Verify PVCs cleaned up
}

func TestDemoNuke_HandlesMissingDemo(t *testing.T) {
  // Run demo-nuke without demo installed
  // Verify graceful handling (no-op or warning)
}
```

**Web UI Tests** (`web-ui/tests/e2e/demo-nuke.spec.ts`):
```typescript
test('demo environment cleanup shown in UI', async ({ page }) => {
  // Navigate to admin/settings
  // Verify demo environment section
  // Verify cleanup warnings/status
  // (Note: May not trigger nuke from UI - verify status only)
});
```

**Acceptance Criteria**:
- ✅ All demo services removed
- ✅ All demo namespaces deleted
- ✅ No orphaned resources (PVCs, ConfigMaps, Secrets)
- ✅ Clean environment ready for re-installation

---

## Implementation Guidelines

### Test Structure

**Go Tests**:
```
internal/
  database/
    database_test.go         # CRUD operations
  server/
    handlers_test.go         # API endpoints
  workflow/
    executor_test.go         # Workflow execution
    lifecycle_test.go        # Lifecycle operations
    destroy_test.go          # Resource cleanup
cmd/
  cli/
    commands/
      demo_test.go           # Demo environment commands
      deploy_test.go         # Deployment commands
```

**Node/TypeScript Tests**:
```
web-ui/
  tests/
    e2e/
      demo-setup.spec.ts     # Demo environment
      teams.spec.ts          # Team management
      providers.spec.ts      # Provider configuration
      deploy.spec.ts         # Application deployment
      lifecycle.spec.ts      # Lifecycle operations
      destroy.spec.ts        # Resource deletion
      demo-nuke.spec.ts      # Demo cleanup
```

### Test Tooling

**Go**:
- Standard `testing` package
- Table-driven tests for multiple scenarios
- `testify/assert` for assertions
- `testcontainers-go` for PostgreSQL (if needed)

**TypeScript**:
- Playwright (recommended) or Cypress
- Page Object Model for UI interactions
- Fixtures for test data
- Screenshots/videos on failure

### Test Data

**Minimal fixtures**:
```yaml
# testdata/score-spec.yaml
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  web:
    image: nginx:latest
    variables:
      PORT: "8080"
```

### CI Integration

**GitHub Actions** (`.github/workflows/e2e-tests.yml`):
```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  test-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go test ./... -v -race -coverprofile=coverage.out

  test-ui:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - run: cd web-ui && npm ci && npm run test:e2e
```

---

## Success Criteria

**Coverage Targets**:
- Go: 80%+ coverage for handlers, database, workflow packages
- TypeScript: Critical user paths covered (deploy, destroy, team setup)

**Performance**:
- Go tests: Complete in < 2 minutes
- UI tests: Complete in < 5 minutes
- Demo setup/teardown: < 2 minutes (or skip for fast tests)

**Reliability**:
- Tests pass consistently (no flaky tests)
- Failures clearly indicate root cause
- Tests clean up after themselves (no state leakage)

---

## Verification Checklist

Before marking this feature complete:

- [ ] All 7 test scenarios implemented
- [ ] Go tests pass: `go test ./... -v`
- [ ] UI tests pass: `cd web-ui && npm run test:e2e`
- [ ] CI pipeline configured and passing
- [ ] Coverage reports generated
- [ ] Test documentation added to `docs/development/testing.md`
- [ ] Known limitations documented (if any)

---

## References

- Project docs: `docs/development/testing.md`
- Verification-first principle: `CLAUDE.md`
- Demo environment: `cmd/cli/commands/demo.go`
- Golden paths: `cmd/cli/commands/goldenpath.go`
- Web UI: `web-ui/src/`
