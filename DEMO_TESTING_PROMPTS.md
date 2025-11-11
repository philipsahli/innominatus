# Innominatus Demo Testing Prompts

This document contains comprehensive testing prompts for Claude to verify all critical user stories and identify bugs/flaws before the demo. Each prompt is designed to be executed independently to validate specific functionality.

---

## How to Use This Document

For each test prompt:
1. **Execute the steps exactly as written**
2. **Verify the expected outcomes**
3. **Document any deviations, errors, or bugs found**
4. **Check for common failure scenarios listed**
5. **Report findings with error messages and logs**

---

# Part 1: Application Developer Testing

## AD-1: Deploy Application with Automatic Resource Provisioning

**User Story:** As an application developer, I want to deploy my application by submitting a Score specification, so that my app and all required infrastructure are provisioned automatically.

### Test Prompt for Claude:

```
Please test the end-to-end application deployment workflow:

STEP 1: Start the innominatus server
- Run: ./innominatus
- Verify: Server starts without errors
- Check: Logs show "Server listening on :8081"
- Check: Database migrations complete successfully
- Check: Providers loaded (database-team, container-team, storage-team, vault-team, identity-team, observability-team)

STEP 2: Create a Score specification with resources
- Create file: test-app-deploy.yaml
- Content:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-ecommerce-app
    labels:
      environment: development
      team: qa-testing
  containers:
    web:
      image: nginx:latest
      variables:
        APP_ENV: production
        LOG_LEVEL: info
  service:
    ports:
      http:
        port: 8080
        targetPort: 8080
  resources:
    database:
      type: postgres
      params:
        version: "15"
        size: small
    storage:
      type: s3-bucket
      params:
        region: us-east-1
  ```

STEP 3: Deploy via CLI
- Run: ./innominatus-ctl deploy test-app-deploy.yaml
- Verify: Command succeeds with success message
- Check: Application appears in list: ./innominatus-ctl list
- Check: Application status shows "running" or "provisioning"

STEP 4: Verify resource provisioning
- Run: ./innominatus-ctl list-resources test-ecommerce-app
- Expected: Shows 2 resources (database, storage)
- Verify: Resource states are "requested", "provisioning", or "active"
- Check: Each resource has a workflow_execution_id assigned
- Check: Provider field shows "database-team" for postgres, "storage-team" for s3-bucket

STEP 5: Check workflow execution
- Run: ./innominatus-ctl list-workflows test-ecommerce-app
- Expected: Shows workflow executions for resource provisioning
- Verify: Workflows are in "running", "completed", or "failed" state
- For each workflow, run: ./innominatus-ctl workflow detail <workflow-id>
- Check: Workflow steps executed without errors
- If failed, run: ./innominatus-ctl workflow logs <workflow-id>

STEP 6: Verify via Web UI
- Open: http://localhost:8081/apps
- Verify: test-ecommerce-app appears in the list
- Click: "View" or application name
- Check: Application detail page loads
- Check: Resources section shows postgres and s3-bucket
- Check: Status badges are correct

STEP 7: Verify via API
- Run: curl -H "Authorization: Bearer $API_TOKEN" http://localhost:8081/api/applications
- Expected: JSON array includes test-ecommerce-app
- Run: curl -H "Authorization: Bearer $API_TOKEN" http://localhost:8081/api/resources?app=test-ecommerce-app
- Expected: JSON shows both resources with details

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Provider not found for resource type → Check provider capabilities match resource types
2. ❌ Workflow stuck in "provisioning" → Check orchestration engine is running
3. ❌ "metadata.name is required" error → Verify validation added in handlers.go:552
4. ❌ YAML parsing errors → Verify Content-Type: application/yaml is used
5. ❌ Resources created but workflow_execution_id is NULL → Check orchestration engine polling
6. ❌ Graph edges without nodes → Verify nodes created before edges

BUGS TO REPORT:
- Any errors in server logs
- Failed workflow steps with details
- Resources stuck in non-terminal states
- Missing or incorrect status displays
- UI/UX issues in web interface
```

---

## AD-2: Monitor Application Status

**User Story:** As an application developer, I want to check the status of my deployed applications, so that I can troubleshoot issues and verify deployment success.

### Test Prompt for Claude:

```
Please test the application monitoring capabilities:

STEP 1: List all applications via CLI
- Run: ./innominatus-ctl list
- Verify: Shows list of deployed applications
- Check: Columns include Name, Status, Environment, Resources, Updated
- Verify: test-ecommerce-app from AD-1 appears

STEP 2: Get detailed application status
- Run: ./innominatus-ctl status test-ecommerce-app
- Expected: Shows detailed application information
- Verify: Status field is one of: running, provisioning, failed, pending
- Check: Environment matches Score spec (development)
- Check: Team matches Score spec (qa-testing)
- Check: Resource count is accurate

STEP 3: View application via Web UI Dashboard
- Open: http://localhost:8081/dashboard
- Verify: Dashboard loads without errors
- Check: Application statistics show correct counts
- Check: Recent activity section shows deployments
- Check: Resource status breakdown is accurate

STEP 4: View application details via Web UI
- Navigate to: http://localhost:8081/dev/applications
- Verify: Application list loads
- Click: "View" on test-ecommerce-app
- Expected: Application detail page loads at /dev/applications/test-ecommerce-app
- Check: Application name displayed correctly
- Check: Status badge shows correct state
- Check: Environment card shows "development"
- Check: Resources card shows "2 resources"
- Check: Resources table lists postgres and s3-bucket
- Check: Each resource shows name, type, state, provider, created date

STEP 5: Check workflow execution history
- Run: ./innominatus-ctl list-workflows test-ecommerce-app
- Verify: Shows all workflows executed for this app
- Check: Timestamps are recent and correct
- For a completed workflow:
  - Run: ./innominatus-ctl workflow detail <workflow-id>
  - Verify: Shows workflow metadata, steps, and execution status
  - Check: Duration is calculated correctly
  - Check: All steps have completion timestamps

STEP 6: Monitor resource health
- Run: ./innominatus-ctl list-resources test-ecommerce-app
- For each resource:
  - Check: State is a valid lifecycle state
  - Check: health_status field exists
  - If active: Verify health_status is "healthy" or has error details
- Via API: curl http://localhost:8081/api/resources?app=test-ecommerce-app
- Verify: Response includes health_status for each resource

STEP 7: View graph visualization
- Open: http://localhost:8081/graph
- Verify: Graph loads without errors
- Search for: test-ecommerce-app
- Check: Spec node exists
- Check: Resource nodes connected to spec
- Check: Provider nodes connected to resources
- Check: Workflow nodes connected to resources
- Check: Edge types are correct (contains, requires)

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Application shows 0 resources but resources exist → Check resource query logic
2. ❌ Status shows "unknown" → Verify status mapping in Application interface
3. ❌ Workflow logs empty → Check if log storage is working
4. ❌ Graph visualization shows no connections → Verify graph edges created
5. ❌ "Application not found" for existing app → Check API authentication
6. ❌ Timestamps show "Invalid Date" → Check date parsing in frontend

BUGS TO REPORT:
- Incorrect status displays
- Missing or stale data
- UI components failing to load
- Graph visualization errors
- Performance issues (slow queries)
```

---

## AD-3: Access Application Resources

**User Story:** As an application developer, I want to view connection details for provisioned resources, so that I can configure my application to use them.

### Test Prompt for Claude:

```
Please test resource connection details and hints:

STEP 1: List resources with details
- Run: ./innominatus-ctl list-resources test-ecommerce-app
- Verify: Shows all resources for the application
- Check: Each resource has configuration field
- Check: Provider metadata exists (if applicable)

STEP 2: Get specific resource details
- Identify a resource ID from step 1
- Run: ./innominatus-ctl get-resource <resource-id>
- Expected: Detailed JSON output
- Verify: Fields include:
  - id, application_name, resource_name, resource_type
  - state, health_status
  - configuration (Score spec params)
  - provider_id, provider_metadata
  - created_at, updated_at
- Check: Resource hints array exists

STEP 3: Verify resource hints
- For postgres resource:
  - Check: Hints include connection string format
  - Check: Hints include database credentials location (Vault path)
  - Example hint: "Connection string: postgresql://user:pass@host:5432/dbname"
  - Example hint: "Credentials in Vault: secret/data/test-ecommerce-app/database"

- For s3-bucket resource:
  - Check: Hints include bucket endpoint URL
  - Check: Hints include bucket name
  - Check: Hints may include MinIO console link
  - Example hint: "Bucket: test-ecommerce-app-storage"
  - Example hint: "Endpoint: http://minio.localtest.me:9000"

STEP 4: View resources in Web UI
- Navigate to: http://localhost:8081/resources
- Verify: Page loads resource list
- Filter by: test-ecommerce-app
- Click on: postgres resource
- Check: Resource details pane opens
- Verify: Shows configuration parameters
- Verify: Shows provider information
- Check: Hints/connection details displayed
- Check: Links to external services (if any) are clickable

STEP 5: Test API for resource details
- Run: curl http://localhost:8081/api/resources?app=test-ecommerce-app
- Verify: Returns resources with full details
- Check: hints field is populated with useful information
- For each hint:
  - Check: text field has descriptive message
  - Check: url field exists (if applicable)
  - Check: type field categorizes the hint (connection, url, command)

STEP 6: Verify provider metadata
- Check postgres resource provider_metadata:
  - Should include operator-specific details
  - May include namespace, service name, port
- Check s3-bucket resource provider_metadata:
  - Should include bucket configuration
  - May include region, encryption settings

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Hints array is empty → Check if providers populate hints
2. ❌ Connection strings have placeholder values → Verify actual credentials generated
3. ❌ URLs return 404 → Check if target services are running
4. ❌ Vault paths don't exist → Verify vault integration
5. ❌ Provider metadata is null → Check workflow populates metadata
6. ❌ Resource detail page shows "Resource not found" → Verify resource ID in URL

BUGS TO REPORT:
- Missing or incomplete connection details
- Broken links to external services
- Invalid connection string formats
- Missing credentials or access keys
- UI display issues for hints
```

---

## AD-4: Troubleshoot Failed Deployments

**User Story:** As an application developer, I want to view detailed logs when a deployment fails, so that I can understand and fix the issue.

### Test Prompt for Claude:

```
Please test failure scenarios and troubleshooting capabilities:

STEP 1: Create a Score spec with invalid resource type
- Create file: test-invalid-resource.yaml
- Content:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-invalid-app
  containers:
    web:
      image: nginx:latest
  resources:
    cache:
      type: redis-cluster  # This type doesn't exist
  ```
- Run: ./innominatus-ctl deploy test-invalid-resource.yaml
- Expected: Deployment should fail with clear error message
- Verify: Error mentions "no provider found for resource type: redis-cluster"
- Check: Error message is helpful and actionable

STEP 2: Create a Score spec with missing required field
- Create file: test-missing-name.yaml
- Content:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    labels:
      team: testing
  # Missing metadata.name!
  containers:
    web:
      image: nginx:latest
  ```
- Run: ./innominatus-ctl deploy test-missing-name.yaml
- Expected: Deployment fails with validation error
- Verify: Error message: "metadata.name is required in Score specification"
- Check: HTTP status is 400 Bad Request (if via API)

STEP 3: Validate Score spec locally before deployment
- Run: ./innominatus-ctl validate test-invalid-resource.yaml
- Expected: Validation catches errors before deployment
- Verify: Shows specific validation errors
- Run: ./innominatus-ctl validate test-missing-name.yaml
- Expected: Shows "metadata.name is required"

STEP 4: Deploy app and cause workflow failure
- Create file: test-workflow-failure.yaml
- Content:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-workflow-fail
  containers:
    web:
      image: nginx:latest
  resources:
    db:
      type: postgres
      params:
        version: "invalid-version"  # Should cause failure
  ```
- Run: ./innominatus-ctl deploy test-workflow-failure.yaml
- Wait for workflow to execute and fail
- Run: ./innominatus-ctl list-workflows test-workflow-fail
- Expected: Shows workflow with "failed" status

STEP 5: View workflow logs for failed execution
- Get workflow ID from step 4
- Run: ./innominatus-ctl workflow logs <workflow-id>
- Expected: Shows detailed logs from workflow execution
- Check: Error messages are clear and specific
- Check: Shows which step failed
- Check: Includes stderr output from failed commands
- Verify: Timestamps for each log entry

STEP 6: View workflow logs by step
- Run: ./innominatus-ctl workflow logs <workflow-id> --step init
- Verify: Shows only logs from "init" step
- Run: ./innominatus-ctl workflow logs <workflow-id> --step provision-postgres
- Expected: Shows logs from postgres provisioning step
- Check: Error details are visible

STEP 7: Check failure via Web UI
- Open: http://localhost:8081/workflows
- Verify: Failed workflow appears in list
- Check: Status badge shows "failed" in red
- Click: Failed workflow
- Expected: Workflow detail page shows error information
- Check: Error message is displayed
- Check: Failed step is highlighted
- Verify: Can view logs for each step

STEP 8: Verify resource state on failure
- Run: ./innominatus-ctl list-resources test-workflow-fail
- Expected: Resource state should be "failed"
- Check: error_message field populated with failure reason
- Via API: curl http://localhost:8081/api/resources?app=test-workflow-fail
- Verify: Resource shows failed state and error details

STEP 9: Test retry mechanism (if implemented)
- Run: ./innominatus-ctl retry <workflow-id> test-workflow-failure.yaml
- Expected: Creates new workflow execution
- Verify: New workflow ID generated
- Check: Can view logs of retry attempt

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ No error message shown on failure → Check error handling in handlers
2. ❌ Logs show "no logs available" → Verify log storage works
3. ❌ Workflow shows "running" but never completes → Check for infinite loops
4. ❌ Error messages are cryptic → Improve error messaging
5. ❌ Failed resources don't show error_message → Check resource update logic
6. ❌ Can't distinguish which step failed → Add step status tracking

BUGS TO REPORT:
- Missing or unclear error messages
- Logs not captured properly
- Resource stuck in wrong state after failure
- UI not showing failure details
- Stack traces exposed to users
```

---

## AD-5: Manage API Keys

**User Story:** As an application developer, I want to generate API keys for CLI/API access, so that I can automate deployments in CI/CD pipelines.

### Test Prompt for Claude:

```
Please test API key generation and usage:

STEP 1: Access profile page
- Open: http://localhost:8081/profile
- Verify: Page loads without errors
- Check: Shows user information (username, team, role)
- Check: API Keys section is visible

STEP 2: Generate a new API key
- Click: "Generate New API Key" button
- Fill in: Name = "CI/CD Pipeline"
- Select: Expiry = 30 days
- Click: Generate
- Expected: API key is generated and displayed
- Check: Key starts with "inn_"
- Check: Key is displayed only once with warning to copy it
- Verify: Key appears in API keys list
- Check: Shows name, created date, expiry date, last used (never)

STEP 3: Copy and test API key
- Copy the generated API key
- Export: export INNOMINATUS_API_KEY="<copied-key>"
- Run: ./innominatus-ctl list
- Expected: Command succeeds without prompting for login
- Verify: Applications list is returned
- Run: ./innominatus-ctl status test-ecommerce-app
- Expected: Command uses API key authentication

STEP 4: Test API key via HTTP API
- Run: curl -H "Authorization: Bearer <api-key>" http://localhost:8081/api/applications
- Expected: Returns JSON array of applications
- Check: HTTP status 200
- Run: curl -H "Authorization: Bearer <api-key>" http://localhost:8081/api/resources
- Expected: Returns resources
- Verify: No 401 Unauthorized errors

STEP 5: Verify last used timestamp
- Wait a few seconds
- Refresh: http://localhost:8081/profile
- Check: API key "CI/CD Pipeline" shows recent "Last Used" timestamp
- Verify: Timestamp updates after each API call

STEP 6: Test invalid API key
- Run: curl -H "Authorization: Bearer invalid_key_123" http://localhost:8081/api/applications
- Expected: HTTP 401 Unauthorized
- Check: Error message: "Invalid or expired API key"

STEP 7: Test expired API key
- Generate API key with very short expiry (if possible)
- OR manually update expiry in database:
  - Connect to DB: psql -h localhost -U orchestrator -d idp_orchestrator
  - Update: UPDATE api_keys SET expires_at = NOW() - INTERVAL '1 day' WHERE name = 'CI/CD Pipeline';
- Run: curl -H "Authorization: Bearer <expired-key>" http://localhost:8081/api/applications
- Expected: HTTP 401 Unauthorized
- Check: Error message indicates key is expired

STEP 8: Revoke API key
- Open: http://localhost:8081/profile
- Find: "CI/CD Pipeline" key in list
- Click: "Revoke" or "Delete" button
- Confirm: Deletion
- Expected: Key removed from list
- Run: curl -H "Authorization: Bearer <revoked-key>" http://localhost:8081/api/applications
- Expected: HTTP 401 Unauthorized
- Check: Error indicates key was revoked

STEP 9: Test API key in CI/CD simulation
- Create script: test-ci-cd.sh
- Content:
  ```bash
  #!/bin/bash
  export INNOMINATUS_API_KEY="<api-key>"

  # Deploy application
  ./innominatus-ctl deploy test-app.yaml

  # Wait for completion
  sleep 10

  # Check status
  ./innominatus-ctl status test-app

  # List resources
  ./innominatus-ctl list-resources test-app
  ```
- Run: chmod +x test-ci-cd.sh && ./test-ci-cd.sh
- Expected: All commands succeed with API key auth
- Verify: No interactive prompts for credentials

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ API key not generated → Check /api/profile/api-keys endpoint
2. ❌ Key doesn't work immediately → Check key is committed to DB
3. ❌ Last used timestamp doesn't update → Check update logic in auth middleware
4. ❌ Expired keys still work → Verify expiry check in authentication
5. ❌ Can't revoke keys → Check DELETE endpoint exists
6. ❌ API key visible in logs → Check sensitive data is redacted

BUGS TO REPORT:
- API key generation failures
- Authentication not working with valid keys
- Expiry logic not enforced
- UI issues on profile page
- Security concerns (keys in logs, etc.)
```

---

# Part 2: Product/Service Team Testing

## PST-1: Create Provider Workflows

**User Story:** As a product team engineer, I want to create workflows that provision my team's services, so that developers can automatically consume our infrastructure.

### Test Prompt for Claude:

```
Please test provider creation and registration:

STEP 1: Create a test provider directory
- Create: providers/test-team/
- Create: providers/test-team/provider.yaml
- Content:
  ```yaml
  apiVersion: v1
  kind: Provider
  metadata:
    name: test-team
    version: 1.0.0
    category: service
    description: Test provider for validation

  capabilities:
    resourceTypes:
      - test-resource
      - test-service

  workflows:
    - name: provision-test-resource
      file: ./workflows/provision-test-resource.yaml
      description: Provision a test resource
      category: provisioner
      operation: create
      tags: [test, resource]
  ```

STEP 2: Create provisioner workflow
- Create: providers/test-team/workflows/
- Create: providers/test-team/workflows/provision-test-resource.yaml
- Content:
  ```yaml
  name: provision-test-resource
  description: Provisions a test resource

  inputs:
    - name: resource_name
      type: string
      required: true
    - name: application_name
      type: string
      required: true

  steps:
    - name: init
      type: shell
      config:
        command: |
          echo "Provisioning test resource: ${resource_name}"
          echo "For application: ${application_name}"

    - name: create-resource
      type: shell
      config:
        command: |
          echo "Resource created successfully"
          echo "Resource ID: test-${resource_name}-123"

  outputs:
    resource_id: "test-${resource_name}-123"
    status: "active"
  ```

STEP 3: Restart server to load new provider
- Stop: innominatus server (Ctrl+C)
- Start: ./innominatus
- Check logs: Look for "Loading provider: test-team"
- Verify: No errors loading provider
- Check: "Registered capabilities for test-team: [test-resource, test-service]"

STEP 4: Verify provider loaded
- Run: ./innominatus-ctl list-providers
- Expected: test-team appears in list
- Check: Category shows "service"
- Check: Version is "1.0.0"
- Via API: curl http://localhost:8081/api/providers
- Verify: test-team in JSON response
- Check: capabilities field includes test-resource and test-service

STEP 5: Test provider capability resolution
- Create Score spec using test-resource:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-provider-app
  containers:
    web:
      image: nginx:latest
  resources:
    my-test:
      type: test-resource
  ```
- Run: ./innominatus-ctl deploy test-provider-app.yaml
- Expected: Deployment succeeds
- Verify: Resource created with type "test-resource"
- Check: Provider field = "test-team"

STEP 6: Verify workflow execution
- Run: ./innominatus-ctl list-workflows test-provider-app
- Expected: Shows workflow execution for provision-test-resource
- Get workflow ID
- Run: ./innominatus-ctl workflow logs <workflow-id>
- Check: Logs show "Provisioning test resource: my-test"
- Check: Logs show "Resource created successfully"
- Verify: Workflow completed successfully

STEP 7: Test provider conflict detection
- Create another provider claiming same resource type
- Create: providers/conflict-team/provider.yaml
- Content:
  ```yaml
  apiVersion: v1
  kind: Provider
  metadata:
    name: conflict-team
    version: 1.0.0
  capabilities:
    resourceTypes:
      - test-resource  # Same as test-team!
  workflows: []
  ```
- Restart server: ./innominatus
- Expected: Server detects conflict
- Check logs: "Capability conflict detected: resource type 'test-resource' claimed by multiple providers"
- Verify: Server fails to start OR logs warning
- Clean up: Remove providers/conflict-team/

STEP 8: Test provider with CRUD operations
- Update: providers/test-team/provider.yaml
- Add UPDATE and DELETE workflows:
  ```yaml
  workflows:
    - name: provision-test-resource
      file: ./workflows/provision-test-resource.yaml
      category: provisioner
      operation: create

    - name: update-test-resource
      file: ./workflows/update-test-resource.yaml
      category: provisioner
      operation: update

    - name: delete-test-resource
      file: ./workflows/delete-test-resource.yaml
      category: provisioner
      operation: delete
  ```
- Create the update and delete workflow files
- Restart server
- Verify: No errors loading workflows
- Check: All three operations registered

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Provider not loaded → Check YAML syntax errors
2. ❌ Capabilities not registered → Verify resourceTypes array format
3. ❌ Workflow file not found → Check file path is correct relative to provider.yaml
4. ❌ Duplicate resource type claims → Check conflict detection works
5. ❌ Workflow doesn't execute → Verify category is "provisioner" not "goldenpath"
6. ❌ Variables not interpolated → Check ${variable} syntax in workflow

BUGS TO REPORT:
- Provider loading errors
- Capability registration failures
- Workflow file loading issues
- Conflict detection not working
- Variable interpolation problems
```

---

## PST-2: Test Workflows in Demo Environment

**User Story:** As a product team engineer, I want to test my workflows in a demo environment, so that I can verify they work before releasing to developers.

### Test Prompt for Claude:

```
Please test the demo environment setup and workflow testing:

STEP 1: Install demo environment
- Run: ./innominatus-ctl demo-time
- Expected: Installs demo services (Gitea, ArgoCD, Vault, MinIO, Keycloak)
- Wait: Installation may take 5-10 minutes
- Check logs: Should show progress for each service
- Verify: All services start successfully

STEP 2: Check demo status
- Run: ./innominatus-ctl demo-status
- Expected: Shows health status of all demo services
- Check: Gitea - http://gitea.localtest.me (should be accessible)
- Check: ArgoCD - http://argocd.localtest.me (should be accessible)
- Check: Vault - http://vault.localtest.me (should be accessible)
- Check: MinIO - http://minio.localtest.me (should be accessible)
- Check: Keycloak - http://keycloak.localtest.me (should be accessible)
- Verify: All show "healthy" or "running"

STEP 3: Test Gitea integration
- Open: http://gitea.localtest.me
- Login: admin/admin
- Verify: Can access Gitea UI
- Via API: curl http://gitea.localtest.me/api/v1/repos/search
- Check: Returns JSON response

STEP 4: Deploy app that uses Gitea
- Create Score spec:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-gitops-app
  containers:
    web:
      image: nginx:latest
  resources:
    repo:
      type: gitea-repo
      params:
        description: "Test repository"
        private: false
  ```
- Run: ./innominatus-ctl deploy test-gitops-app.yaml
- Wait for provisioning
- Check workflow logs
- Expected: Repository created in Gitea
- Verify in Gitea UI: Repository "test-gitops-app" exists

STEP 5: Test ArgoCD integration
- Deploy app with ArgoCD application:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-argocd-app
  containers:
    web:
      image: nginx:latest
  resources:
    gitops:
      type: argocd-app
      params:
        repoURL: http://gitea.localtest.me/admin/test-repo
        path: manifests
  ```
- Run: ./innominatus-ctl deploy test-argocd-app.yaml
- Wait for provisioning
- Open: http://argocd.localtest.me
- Login: admin/argocd123
- Verify: Application "test-argocd-app" appears in ArgoCD UI
- Check: Application status and health

STEP 6: Test Vault integration
- Deploy app that uses Vault:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-vault-app
  containers:
    web:
      image: nginx:latest
  resources:
    secrets:
      type: vault-space
      params:
        namespace: app-secrets
  ```
- Run: ./innominatus-ctl deploy test-vault-app.yaml
- Wait for provisioning
- Open: http://vault.localtest.me
- Login: root token
- Navigate to: secret/data/test-vault-app/
- Verify: Secrets space created

STEP 7: Test MinIO integration
- Deploy app with S3 bucket:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: test-storage-app
  containers:
    web:
      image: nginx:latest
  resources:
    storage:
      type: s3-bucket
      params:
        public: false
  ```
- Run: ./innominatus-ctl deploy test-storage-app.yaml
- Wait for provisioning
- Open: http://minio.localtest.me
- Login: minioadmin/minioadmin
- Verify: Bucket created for test-storage-app
- Check: Bucket policies and access

STEP 8: Test Keycloak OIDC integration
- Verify: Keycloak realm "demo-realm" exists
- Check: Client "innominatus" registered
- Verify: Redirect URIs include:
  - http://localhost:8081/auth/callback
  - http://innominatus.localtest.me/auth/callback
  - http://127.0.0.1:8082/callback (for CLI SSO)
- Test SSO login flow (see AD-5 tests)

STEP 9: Clean up demo environment
- Run: ./innominatus-ctl demo-nuke
- Expected: Removes all demo services
- Verify: Services are stopped and removed
- Check: No orphaned containers or volumes
- Run: docker ps | grep -E "gitea|argocd|vault|minio|keycloak"
- Expected: No results

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Demo services fail to start → Check Docker Desktop running with Kubernetes enabled
2. ❌ Services not accessible via .localtest.me → Check DNS resolution
3. ❌ Workflows can't connect to services → Check service discovery
4. ❌ OIDC callback fails → Verify redirect URIs registered (recent fix)
5. ❌ Repository creation fails in Gitea → Check Gitea API credentials
6. ❌ ArgoCD app not syncing → Check repository access

BUGS TO REPORT:
- Demo installation failures
- Service health check issues
- Integration authentication problems
- Workflow execution failures in demo environment
- DNS or networking issues
```

---

## PST-3: Create Golden Path Workflows

**User Story:** As a product team engineer, I want to create end-to-end golden path workflows, so that developers have opinionated happy paths for common scenarios.

### Test Prompt for Claude:

```
Please test golden path workflow creation and execution:

STEP 1: Create a golden path workflow
- Create: workflows/test-golden-path.yaml (Note: May need to use provider-based location)
- Content:
  ```yaml
  name: test-golden-path
  description: End-to-end test golden path
  category: goldenpath

  inputs:
    - name: team_name
      type: string
      required: true
      description: Name of the team to onboard

    - name: environment
      type: string
      required: false
      default: development

  steps:
    - name: create-namespace
      type: kubernetes
      config:
        operation: create-namespace
        namespace: "${team_name}-${environment}"

    - name: create-git-repo
      type: gitea-repo
      config:
        org: ${team_name}
        repo: ${team_name}-app
        description: "Repository for ${team_name}"

    - name: provision-database
      type: kubernetes
      config:
        operation: apply
        manifest: |
          apiVersion: postgres-operator.crunchydata.com/v1beta1
          kind: PostgresCluster
          metadata:
            name: ${team_name}-db
            namespace: ${team_name}-${environment}

    - name: create-argocd-app
      type: argocd-app
      config:
        name: ${team_name}-app
        repoURL: http://gitea.localtest.me/${team_name}/${team_name}-app
        path: manifests
        namespace: ${team_name}-${environment}

  outputs:
    namespace: "${team_name}-${environment}"
    repository: "${team_name}-app"
    database: "${team_name}-db"
  ```

STEP 2: Register golden path (if needed)
- Add to provider or admin config
- Restart server if necessary
- Verify golden path appears in registry

STEP 3: List golden paths
- Run: ./innominatus-ctl list-goldenpaths
- Expected: Shows list of available golden paths
- Check: test-golden-path appears in list
- Verify: Description is shown
- Check: Required inputs are listed

STEP 4: Execute golden path workflow
- Run: ./innominatus-ctl run test-golden-path --param team_name=qa-team
- Expected: Workflow execution starts
- Verify: Workflow ID returned
- Check: All steps execute in sequence

STEP 5: Monitor golden path execution
- Get workflow ID from step 4
- Run: ./innominatus-ctl workflow detail <workflow-id>
- Check: Shows all 4 steps
- Verify: Steps execute in order
- Check: Each step shows start/end time
- Run: ./innominatus-ctl workflow logs <workflow-id>
- Verify: Logs from all steps are visible

STEP 6: Verify golden path outcomes
- Check namespace created:
  - Run: kubectl get namespace qa-team-development
  - Expected: Namespace exists
- Check Git repository:
  - Open: http://gitea.localtest.me/qa-team
  - Verify: Organization exists
  - Check: Repository "qa-team-app" exists
- Check PostgreSQL cluster:
  - Run: kubectl get postgrescluster -n qa-team-development
  - Expected: qa-team-db cluster exists
- Check ArgoCD application:
  - Open: http://argocd.localtest.me
  - Verify: Application "qa-team-app" exists

STEP 7: Test golden path via Web UI
- Open: http://localhost:8081/goldenpaths
- Verify: test-golden-path appears in list
- Click: test-golden-path
- Expected: Golden path detail page loads
- Check: Shows description and inputs
- Fill: team_name = "ui-test-team"
- Click: "Execute" or "Run"
- Verify: Execution starts
- Check: Redirects to workflow detail page

STEP 8: Test golden path with invalid inputs
- Run: ./innominatus-ctl run test-golden-path
- Expected: Error - missing required input "team_name"
- Verify: Clear error message shown
- Run: ./innominatus-ctl run test-golden-path --param invalid_param=value
- Expected: Warning or error about unknown parameter

STEP 9: Test dependent step failure handling
- Modify workflow to include a step that will fail
- Add step:
  ```yaml
  - name: fail-on-purpose
    type: shell
    config:
      command: exit 1
  ```
- Run workflow
- Expected: Workflow fails at that step
- Verify: Subsequent steps are not executed
- Check: Workflow status shows "failed"
- Verify: Error details available in logs

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Golden path not listed → Check category is "goldenpath" not "provisioner"
2. ❌ Steps execute out of order → Verify sequential execution
3. ❌ Variables not interpolated → Check ${variable} syntax
4. ❌ Step failure doesn't stop workflow → Verify error handling
5. ❌ Can't find workflow file → Check file path in registration
6. ❌ Inputs not validated → Verify required field checking

BUGS TO REPORT:
- Golden path registration issues
- Execution ordering problems
- Variable interpolation failures
- Error handling gaps
- Missing validation
```

---

## PST-4: Implement CRUD Operations for Resources

**User Story:** As a product team engineer, I want to define separate workflows for CREATE, UPDATE, DELETE operations, so that resources can be managed throughout their full lifecycle.

### Test Prompt for Claude:

```
Please test full CRUD lifecycle for resources:

STEP 1: Verify provider has CRUD workflows defined
- Check: providers/database-team/provider.yaml
- Verify: Contains workflows for all operations:
  ```yaml
  capabilities:
    resourceTypeCapabilities:
      - type: postgres
        operations:
          create:
            workflow: provision-postgres
          update:
            workflow: update-postgres
          delete:
            workflow: delete-postgres
  ```
- If missing: Provider may need CRUD workflows added

STEP 2: CREATE - Provision a resource
- Create Score spec:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: crud-test-app
  containers:
    web:
      image: nginx:latest
  resources:
    database:
      type: postgres
      params:
        version: "15"
        replicas: 2
  ```
- Run: ./innominatus-ctl deploy crud-test-app.yaml
- Wait for provisioning
- Run: ./innominatus-ctl list-resources crud-test-app
- Expected: Resource state = "active"
- Check: replicas = 2 in configuration
- Note: Resource ID for next steps

STEP 3: READ - Get resource details
- Run: ./innominatus-ctl get-resource <resource-id>
- Verify: Shows full resource details
- Check: state = "active"
- Check: configuration.replicas = 2
- Check: provider_id populated
- Via API: curl http://localhost:8081/api/resources/<resource-id>
- Verify: Same data returned

STEP 4: UPDATE - Scale resource up
- Update Score spec to increase replicas:
  ```yaml
  resources:
    database:
      type: postgres
      params:
        version: "15"
        replicas: 5  # Increased from 2
  ```
- OR use direct API:
  ```bash
  curl -X PUT http://localhost:8081/api/resources/<resource-id> \
    -H "Content-Type: application/json" \
    -d '{
      "configuration": {
        "version": "15",
        "replicas": 5
      },
      "desired_operation": "update"
    }'
  ```
- Expected: Resource state transitions to "updating"
- Check workflow: Run ./innominatus-ctl list-workflows crud-test-app
- Verify: New workflow with update-postgres
- Wait for completion
- Check: Resource state returns to "active"
- Verify: configuration.replicas = 5

STEP 5: Verify UPDATE workflow execution
- Get update workflow ID
- Run: ./innominatus-ctl workflow detail <update-workflow-id>
- Check: Workflow name = "update-postgres"
- Verify: Operation type = "update"
- Run: ./innominatus-ctl workflow logs <update-workflow-id>
- Check: Logs show scaling operation
- Verify: No errors

STEP 6: DELETE - Remove resource
- Via API:
  ```bash
  curl -X DELETE http://localhost:8081/api/resources/<resource-id>
  ```
- OR update Score spec to remove resource and redeploy
- Expected: Resource state transitions to "terminating"
- Check workflow: Should create delete-postgres workflow
- Wait for completion
- Verify: Resource state = "terminated"
- Check: Resource still exists in DB (soft delete)

STEP 7: Verify DELETE workflow execution
- Get delete workflow ID
- Run: ./innominatus-ctl workflow detail <delete-workflow-id>
- Check: Workflow name = "delete-postgres"
- Verify: Operation type = "delete"
- Run: ./innominatus-ctl workflow logs <delete-workflow-id>
- Check: Logs show cleanup operations
- Verify: No errors

STEP 8: Test resource state transitions
- Verify state machine:
  - CREATE: requested → provisioning → active
  - UPDATE: active → updating → active
  - DELETE: active → terminating → terminated
  - FAILED: any state → failed (on error)
- Check: No invalid state transitions occurred
- Verify: State history is tracked

STEP 9: Test operation-specific workflow selection
- Create resource with tags:
  ```yaml
  resources:
    database:
      type: postgres
      params:
        version: "15"
        operation_tags: [scaling]
  ```
- Verify: If multiple UPDATE workflows exist, tags select correct one
- Check: Workflow resolver uses tags for disambiguation

STEP 10: Test concurrent operations prevention
- Trigger UPDATE on a resource
- While updating, try to DELETE
- Expected: Error or queuing - can't have concurrent operations
- Verify: Second operation waits or fails gracefully

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ UPDATE workflow not found → Verify operation mapping in provider.yaml
2. ❌ Resource stuck in "updating" → Check workflow completion updates state
3. ❌ DELETE doesn't trigger workflow → Verify DELETE operation registered
4. ❌ State transitions invalid → Check state machine logic
5. ❌ Concurrent operations cause conflicts → Verify locking mechanism
6. ❌ Configuration not updated after UPDATE → Check resource update logic

BUGS TO REPORT:
- CRUD workflow registration issues
- State transition problems
- Operation selection failures
- Concurrent operation conflicts
- Configuration update bugs
```

---

# Part 3: Platform Engineer Testing

## PE-1: Deploy and Configure innominatus Platform

**User Story:** As a platform engineer, I want to deploy innominatus to Kubernetes or standalone, so that my organization has a running platform orchestration system.

### Test Prompt for Claude:

```
Please test platform deployment and configuration:

STEP 1: Test standalone deployment
- Build: make build OR go build -o innominatus cmd/server/main.go
- Verify: Binary created successfully
- Run: ./innominatus --help
- Check: Shows usage and options
- Set environment variables:
  ```bash
  export DB_HOST=localhost
  export DB_USER=orchestrator
  export DB_PASSWORD=test123
  export DB_NAME=idp_orchestrator
  export AUTH_TYPE=file
  export USERS_FILE=./users.yaml
  ```
- Run: ./innominatus
- Expected: Server starts on port 8081
- Check logs: "Server listening on :8081"
- Verify: Database migrations run automatically
- Check: Providers loaded successfully

STEP 2: Test health and readiness endpoints
- Run: curl http://localhost:8081/health
- Expected: {"status": "healthy", "timestamp": "..."}
- Check: HTTP 200 status
- Run: curl http://localhost:8081/ready
- Expected: {"status": "ready", "database": "connected", ...}
- Verify: All components show healthy

STEP 3: Test metrics endpoint
- Run: curl http://localhost:8081/metrics
- Expected: Prometheus-formatted metrics
- Check: Includes metrics like:
  - workflow_executions_total
  - resource_provisioning_duration
  - http_requests_total
  - database_query_duration
- Verify: Metrics update on activity

STEP 4: Test file-based authentication
- Check: users.yaml exists
- Verify: Contains test users
- Open: http://localhost:8081/login
- Login: admin/admin123
- Expected: Successful login
- Check: Redirects to dashboard
- Verify: Session cookie set

STEP 5: Configure OIDC authentication
- Set environment variables:
  ```bash
  export OIDC_ENABLED=true
  export OIDC_ISSUER=https://keycloak.localtest.me/realms/demo-realm
  export OIDC_CLIENT_ID=innominatus
  export OIDC_CLIENT_SECRET=innominatus-client-secret
  ```
- Restart: ./innominatus
- Check logs: "OIDC authentication enabled"
- Verify: Issuer validation succeeds
- Check: Discovery document fetched
- Open: http://localhost:8081/login
- Verify: Shows "Login with SSO" option

STEP 6: Test OIDC login flow
- Click: "Login with SSO"
- Expected: Redirects to Keycloak
- Login: admin/admin (in Keycloak)
- Expected: Redirects back to innominatus at /auth/callback
- Verify: Callback succeeds (no "invalid redirect URI" error - recent fix)
- Check: User logged in and session created
- Verify: Username and roles extracted from OIDC token

STEP 7: Test CLI SSO authentication (recent fix)
- Run: ./innominatus-ctl login --sso
- Expected: Opens browser to OIDC provider
- Login: in browser
- Expected: CLI receives callback on http://127.0.0.1:8082/callback
- Verify: Callback succeeds (recent fix for fixed port)
- Check: CLI displays "Authentication successful"
- Verify: API key or session token saved
- Run: ./innominatus-ctl list
- Expected: Uses SSO authentication, no login prompt

STEP 8: Test database configuration
- Stop server
- Test PostgreSQL connection:
  ```bash
  export DB_HOST=postgres.example.com
  export DB_PORT=5432
  export DB_USER=orchestrator
  export DB_PASSWORD=secure_password
  export DB_NAME=idp_orchestrator
  export DB_SSLMODE=require
  ```
- Start: ./innominatus
- Verify: Connects to external PostgreSQL
- Check logs: "Database connection established"
- Verify: Migrations run successfully

STEP 9: Test configuration file
- Create: innominatus-config.yaml
- Content:
  ```yaml
  server:
    port: 8081
    host: 0.0.0.0

  database:
    host: localhost
    port: 5432
    user: orchestrator
    database: idp_orchestrator

  auth:
    type: oidc
    oidc:
      enabled: true
      issuer: https://keycloak.localtest.me/realms/demo-realm
      client_id: innominatus
      client_secret: secret

  providers:
    discovery:
      enabled: true
      paths:
        - ./providers
  ```
- Run: ./innominatus --config innominatus-config.yaml
- Verify: Configuration loaded from file
- Check: Settings override environment variables

STEP 10: Test Docker deployment
- Build image: docker build -t innominatus:test .
- Run container:
  ```bash
  docker run -p 8081:8081 \
    -e DB_HOST=host.docker.internal \
    -e DB_USER=orchestrator \
    -e DB_PASSWORD=test123 \
    -e DB_NAME=idp_orchestrator \
    -e AUTH_TYPE=file \
    innominatus:test
  ```
- Verify: Container starts successfully
- Check: Accessible at http://localhost:8081
- Test health: curl http://localhost:8081/health

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Server fails to start → Check database connection
2. ❌ Migrations fail → Verify database permissions
3. ❌ OIDC discovery fails → Check issuer URL is accessible
4. ❌ Providers don't load → Check filesystem permissions
5. ❌ Port already in use → Change PORT environment variable
6. ❌ SSL/TLS errors → Check DB_SSLMODE setting

BUGS TO REPORT:
- Deployment failures
- Configuration parsing errors
- Authentication setup issues
- Database connection problems
- Docker container issues
```

---

## PE-2: Configure Providers

**User Story:** As a platform engineer, I want to register and configure product team providers, so that their services are available to developers.

### Test Prompt for Claude:

```
Please test provider configuration and management:

STEP 1: Test filesystem provider loading
- Check: providers/ directory exists
- Verify: Contains builtin providers:
  - database-team
  - storage-team
  - container-team
  - vault-team
  - identity-team
  - observability-team
- Start server: ./innominatus
- Check logs: Each provider should be loaded
- Verify: No errors like "failed to load provider"

STEP 2: List loaded providers via CLI
- Run: ./innominatus-ctl list-providers
- Expected: Shows all providers
- Check: Each provider shows:
  - Name
  - Version
  - Category (infrastructure or service)
  - Capabilities (resource types)
  - Workflow count
- Verify: All expected providers present

STEP 3: View provider details
- Run: ./innominatus-ctl get-provider database-team
- Expected: Detailed provider information
- Check: Capabilities list includes postgres, postgresql
- Verify: Workflows list shows:
  - provision-postgres (create)
  - update-postgres (update)
  - delete-postgres (delete)
- Check: Each workflow shows category and operation

STEP 4: Test Git-based provider loading
- Create: admin-config.yaml (if not exists)
- Add Git provider:
  ```yaml
  providers:
    - name: external-team
      type: git
      url: https://github.com/your-org/external-provider
      ref: v1.0.0
      enabled: true
      auth:
        type: none  # or token/ssh
  ```
- Restart: ./innominatus
- Check logs: "Loading Git provider: external-team"
- Verify: Provider cloned to temp directory
- Check: Provider appears in list-providers
- NOTE: Use actual Git URL for real testing

STEP 5: Test provider hot-reload
- Via API:
  ```bash
  curl -X POST http://localhost:8081/api/admin/providers/reload \
    -H "Authorization: Bearer <admin-token>"
  ```
- Expected: Providers reloaded without restart
- Check logs: "Reloading providers..."
- Verify: Provider list updates
- Check: New providers appear
- Verify: Removed providers disappear

STEP 6: Test provider capability conflict detection
- Create conflicting provider:
  - Create: providers/conflict-test/provider.yaml
  - Claim same resource type as existing provider
  - Content:
    ```yaml
    capabilities:
      resourceTypes:
        - postgres  # Conflicts with database-team
    ```
- Restart: ./innominatus
- Expected: Server detects conflict
- Check logs: "Capability conflict detected: resource type 'postgres' claimed by multiple providers"
- Verify: Server logs error OR refuses to start
- Clean up: Remove conflicting provider

STEP 7: Test provider validation
- Create invalid provider:
  - Create: providers/invalid-test/provider.yaml
  - Content: Invalid YAML or missing required fields
- Restart: ./innominatus
- Expected: Validation error logged
- Check: Invalid provider not loaded
- Verify: Other providers still load successfully

STEP 8: Configure provider via admin API
- Create new provider registration:
  ```bash
  curl -X POST http://localhost:8081/api/admin/providers \
    -H "Authorization: Bearer <admin-token>" \
    -H "Content-Type: application/json" \
    -d '{
      "name": "api-test-provider",
      "type": "git",
      "url": "https://github.com/your-org/test-provider",
      "ref": "main",
      "enabled": true
    }'
  ```
- Expected: Provider registered
- Verify: Appears in provider list
- Check: Auto-loaded on next startup

STEP 9: Test provider versioning
- Check provider versions:
  - Run: ./innominatus-ctl list-providers
  - Verify: Each shows version number
- Update provider version in provider.yaml
- Reload providers
- Verify: New version reflected

STEP 10: View providers in Web UI
- Open: http://localhost:8081/providers
- Verify: Provider list page loads
- Check: All providers displayed with cards/tiles
- Click: database-team
- Expected: Provider detail page
- Verify: Shows:
  - Metadata (name, version, category)
  - Capabilities (resource types)
  - Workflows (with descriptions)
  - Statistics (resources provisioned)

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Provider YAML syntax error → Check validation and error messages
2. ❌ Git clone fails → Verify network access and credentials
3. ❌ Workflow files not found → Check relative paths in provider.yaml
4. ❌ Capability not registered → Verify resourceTypes array format
5. ❌ Hot-reload fails → Check for in-flight workflow executions
6. ❌ Version conflicts → Ensure compatibility checks

BUGS TO REPORT:
- Provider loading failures
- Validation error messages unclear
- Hot-reload issues
- Git authentication problems
- UI display errors
```

---

## PE-3: Monitor Platform Health

**User Story:** As a platform engineer, I want to monitor the health and performance of innominatus, so that I can ensure high availability and troubleshoot issues.

### Test Prompt for Claude:

```
Please test monitoring and observability features:

STEP 1: Check health endpoint
- Run: curl http://localhost:8081/health
- Expected: {"status": "healthy", "timestamp": "2025-..."}
- Check: HTTP 200 status
- Verify: Response time < 100ms
- Test during load: Deploy multiple apps and check health
- Verify: Still returns healthy

STEP 2: Check readiness endpoint
- Run: curl http://localhost:8081/ready
- Expected: Detailed readiness status
- Check: Fields include:
  - status: "ready"
  - database: "connected"
  - providers: "loaded"
  - orchestration_engine: "running"
- Verify: All components ready
- Test during startup: Should return "not ready" until fully initialized

STEP 3: Collect Prometheus metrics
- Run: curl http://localhost:8081/metrics
- Verify: Prometheus format:
  - Lines starting with # are comments
  - Metric lines: name{labels} value timestamp
- Check for key metrics:
  - workflow_executions_total
  - workflow_execution_duration_seconds
  - resource_provisioning_total
  - resource_provisioning_duration_seconds
  - http_requests_total
  - http_request_duration_seconds
  - database_query_duration_seconds
  - active_sessions_count

STEP 4: Test metric updates
- Deploy an application
- Run: curl http://localhost:8081/metrics | grep workflow_executions_total
- Note: Current value
- Deploy another application
- Run again: curl http://localhost:8081/metrics | grep workflow_executions_total
- Verify: Counter incremented
- Check: Labels include status (success/failure)

STEP 5: Monitor server logs
- Check: Server logs are structured (JSON or key-value)
- Verify: Log levels (INFO, WARN, ERROR)
- Look for:
  - Request logging: Method, Path, Status, Duration
  - Workflow execution: Start, Progress, Completion
  - Resource provisioning: State changes
  - Errors: Stack traces, context
- Check: Sensitive data not logged (passwords, tokens)

STEP 6: Test error logging
- Trigger an error (deploy invalid spec)
- Check logs: Error logged with context
- Verify: Includes:
  - Timestamp
  - User/session
  - Request details
  - Error message and stack trace
  - Correlation ID (if implemented)

STEP 7: Monitor database performance
- Check metrics: database_query_duration_seconds
- Run: curl http://localhost:8081/metrics | grep database_query
- Verify: Percentiles shown (p50, p90, p99)
- Look for slow queries (>1s)
- Check: Connection pool metrics (active, idle, max)

STEP 8: Monitor workflow execution performance
- Check metrics: workflow_execution_duration_seconds
- Group by: workflow name
- Identify: Slow workflows
- Check: Success/failure rates
- Verify: Metrics match actual execution times

STEP 9: Test dashboard integration (if exists)
- Open: http://localhost:8081/dashboard
- Check: Health status indicators
- Verify: Shows:
  - Active applications count
  - Resource provisioning status
  - Recent workflow executions
  - Error rate
  - System uptime
- Check: Metrics update in real-time

STEP 10: Configure Prometheus scraping
- Create: prometheus.yml
- Content:
  ```yaml
  scrape_configs:
    - job_name: 'innominatus'
      static_configs:
        - targets: ['localhost:8081']
      metrics_path: '/metrics'
      scrape_interval: 15s
  ```
- Run: Prometheus (if installed)
- Verify: Metrics scraped successfully
- Check: Targets show "UP"
- Query: workflow_executions_total
- Verify: Data appears in Prometheus

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Health endpoint returns 500 → Check database connection
2. ❌ Metrics missing → Verify instrumentation code
3. ❌ High error rate → Investigate failing workflows
4. ❌ Slow query warnings → Optimize database queries
5. ❌ Memory leak → Check for goroutine leaks
6. ❌ Connection pool exhausted → Increase max connections

BUGS TO REPORT:
- Health check failures
- Missing or incorrect metrics
- Performance degradation
- Memory leaks
- Database connection issues
```

---

## PE-4: Manage Users and Authentication

**User Story:** As a platform engineer, I want to manage user access and authentication, so that only authorized users can deploy applications.

### Test Prompt for Claude:

```
Please test user management and authentication:

STEP 1: Test file-based authentication (development)
- Check: users.yaml exists
- Verify: Format:
  ```yaml
  users:
    - username: admin
      password: admin123
      team: platform
      role: admin
    - username: alice
      password: alice123
      team: frontend
      role: user
  ```
- Set: export AUTH_TYPE=file
- Start: ./innominatus
- Test login: curl -X POST http://localhost:8081/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username": "admin", "password": "admin123"}'
- Expected: Returns session token or cookie
- Verify: Can access protected endpoints

STEP 2: Test role-based access control (RBAC)
- Login as admin:
  - Try: Access /api/admin/providers
  - Expected: Success (200)
- Login as regular user (alice):
  - Try: Access /api/admin/providers
  - Expected: Forbidden (403)
  - Try: Access /api/applications
  - Expected: Success (200)
- Verify: Role enforcement working

STEP 3: Test team isolation
- Login as alice (team: frontend)
- Deploy app: test-app (team will be set to frontend)
- Login as another user (team: backend)
- Try: Access alice's app
- Expected: Should only see own team's apps OR all apps depending on design
- Verify: Team-based filtering works correctly

STEP 4: Configure OIDC authentication (production)
- Set environment variables:
  ```bash
  export OIDC_ENABLED=true
  export OIDC_ISSUER=https://keycloak.localtest.me/realms/demo-realm
  export OIDC_CLIENT_ID=innominatus
  export OIDC_CLIENT_SECRET=innominatus-client-secret
  export OIDC_REDIRECT_URI=http://localhost:8081/auth/callback
  ```
- Restart server
- Check logs: "OIDC authentication enabled"
- Verify: Discovery document fetched successfully

STEP 5: Test OIDC login via Web UI
- Open: http://localhost:8081/login
- Click: "Login with SSO"
- Expected: Redirects to Keycloak
- Login: admin/admin
- Expected: Redirects to /auth/callback
- Verify: Callback succeeds (no redirect URI error - recent fix)
- Check: Session created
- Verify: User info extracted from OIDC token
- Check: Username, email, roles populated

STEP 6: Test OIDC login via CLI (recent fix)
- Run: ./innominatus-ctl login --sso
- Expected: Opens browser
- Verify: Redirects to OIDC provider
- Login: in browser
- Expected: Callback to http://127.0.0.1:8082/callback
- Verify: Callback succeeds (fixed port - recent fix)
- Check: CLI saves token
- Run: ./innominatus-ctl list
- Expected: Authenticated request succeeds

STEP 7: Test session management
- Login via Web UI
- Check: Session cookie set
- Verify: Cookie properties:
  - HttpOnly: true
  - Secure: true (if HTTPS)
  - SameSite: Lax or Strict
  - Max-Age: Appropriate (1 hour - 1 day)
- Close browser and reopen
- Navigate to: http://localhost:8081
- Verify: Still logged in (session persisted)

STEP 8: Test session expiry
- Login
- Wait for session timeout (or manually expire in DB)
- Try: Access protected endpoint
- Expected: 401 Unauthorized
- Verify: Redirects to login page
- Check: Clear error message

STEP 9: Test API key authentication
- Generate API key via Web UI (/profile)
- Copy key
- Test: curl -H "Authorization: Bearer <api-key>" http://localhost:8081/api/applications
- Expected: Success
- Verify: API key works for all endpoints
- Check: Last used timestamp updates
- Revoke key
- Test again
- Expected: 401 Unauthorized

STEP 10: Test user administration (if implemented)
- Login as admin
- Navigate to: http://localhost:8081/admin/users
- Verify: User list page loads
- Check: Shows all users
- Try: Create new user (if supported)
- Try: Update user role
- Try: Disable/delete user
- Verify: Changes persist

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ OIDC discovery fails → Check issuer URL accessibility
2. ❌ Invalid redirect URI → Verify registered in OIDC provider (recent fix)
3. ❌ Session not created → Check cookie settings
4. ❌ RBAC not enforced → Verify middleware applied
5. ❌ CLI SSO callback fails → Check port 8082 available (recent fix)
6. ❌ Token expired but still accepted → Verify expiry checks

BUGS TO REPORT:
- Authentication failures
- OIDC configuration issues
- Session management bugs
- RBAC enforcement gaps
- API key issues
```

---

## PE-5: Configure Platform Policies

**User Story:** As a platform engineer, I want to configure platform policies and limits, so that I can enforce governance and security standards.

### Test Prompt for Claude:

```
Please test platform policy configuration and enforcement:

STEP 1: Check admin configuration file
- Check: admin-config.yaml exists
- Verify: Contains sections for:
  - policies
  - workflowPolicies
  - providers
  - integrations
- Review: Default policies
- Check: Documentation for each policy

STEP 2: Test workflow execution policies
- Check: admin-config.yaml for:
  ```yaml
  workflowPolicies:
    maxWorkflowDuration: 30m
    maxConcurrentWorkflows: 10
    allowedStepTypes:
      - terraform
      - kubernetes
      - ansible
      - shell
  ```
- Create workflow with disallowed step type
- Try: Execute workflow
- Expected: Validation error
- Verify: Clear error message

STEP 3: Test workflow timeout enforcement
- Create workflow with long-running step:
  ```yaml
  steps:
    - name: long-task
      type: shell
      config:
        command: sleep 3600  # 1 hour
  ```
- Set: maxWorkflowDuration: 5m
- Execute workflow
- Expected: Workflow times out after 5 minutes
- Verify: Status = "failed"
- Check: Error message indicates timeout

STEP 4: Test concurrent workflow limits
- Set: maxConcurrentWorkflows: 2
- Start 3 workflows simultaneously
- Expected: First 2 execute, 3rd queued or rejected
- Verify: Queue mechanism works
- Check: Queued workflow starts after one completes

STEP 5: Test environment policies
- Check: admin-config.yaml for:
  ```yaml
  policies:
    allowedEnvironments:
      - development
      - staging
      - production
  ```
- Deploy app with invalid environment:
  ```yaml
  metadata:
    labels:
      environment: testing  # Not in allowed list
  ```
- Expected: Deployment rejected
- Verify: Error mentions allowed environments

STEP 6: Test resource quota policies (if implemented)
- Check for resource limits:
  ```yaml
  resourcePolicies:
    maxResourcesPerApp: 10
    maxPostgresInstances: 20
  ```
- Deploy app with 11 resources
- Expected: Deployment fails
- Verify: Error message clear

STEP 7: Test approval gates (if implemented)
- Check: admin-config.yaml for:
  ```yaml
  approvalPolicies:
    - environment: production
      requiresApproval: true
      approvers:
        - admin
        - platform-lead
  ```
- Deploy to production environment
- Expected: Deployment enters "pending approval" state
- Verify: Workflow doesn't execute
- Check: Approval notification sent
- Approve deployment (as admin)
- Verify: Workflow proceeds

STEP 8: Test security policies
- Check for security settings:
  ```yaml
  security:
    allowedImageRegistries:
      - docker.io
      - ghcr.io
    requireSignedImages: false
  ```
- Deploy app with image from disallowed registry:
  ```yaml
  containers:
    web:
      image: quay.io/someimage:latest
  ```
- Expected: Deployment rejected
- Verify: Error mentions allowed registries

STEP 9: Test policy updates and hot-reload
- Modify: admin-config.yaml
- Change: maxWorkflowDuration from 30m to 15m
- Reload: curl -X POST http://localhost:8081/api/admin/config/reload
- Verify: New policy active immediately
- Test: Create long workflow
- Expected: Times out at 15m, not 30m

STEP 10: View policies in Web UI (if implemented)
- Open: http://localhost:8081/admin/policies
- Verify: Policy dashboard loads
- Check: Shows current policies
- Verify: Can edit policies (admin only)
- Check: Changes require confirmation
- Test: Update policy via UI
- Verify: Change reflected immediately

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Policies not enforced → Check middleware implementation
2. ❌ Invalid policy syntax → Verify YAML validation
3. ❌ Timeout not working → Check workflow executor timeout logic
4. ❌ Concurrent limit ignored → Verify orchestration engine queue
5. ❌ Policy updates require restart → Implement hot-reload
6. ❌ No audit log of policy changes → Add audit trail

BUGS TO REPORT:
- Policy enforcement failures
- Configuration validation issues
- Hot-reload not working
- Missing error messages
- Security policy bypasses
```

---

## PE-6: Verify Database Migrations

**User Story:** As a platform engineer, I want to safely upgrade the database schema, so that new features are available without data loss.

### Test Prompt for Claude:

```
Please test database migration safety and integrity:

STEP 1: Check migration files
- Location: internal/database/migrations/
- List files: ls -la internal/database/migrations/
- Expected: Sequential numbering:
  - 001_create_graph_tables.sql
  - 002_create_application_tables.sql
  - 003_create_sessions_table.sql
  - ... up to 010_add_application_labels.sql
- Verify: No gaps in numbering
- Check: Each file has .sql extension

STEP 2: Review migration structure
- Read: 001_create_graph_tables.sql
- Check: Contains CREATE TABLE statements
- Verify: Foreign key constraints defined
- Check: Indexes created for performance
- Verify: No DROP statements without checks

STEP 3: Test fresh database initialization
- Drop database: DROP DATABASE IF EXISTS idp_orchestrator_test;
- Create database: CREATE DATABASE idp_orchestrator_test;
- Set: export DB_NAME=idp_orchestrator_test
- Start: ./innominatus
- Check logs: "Running database migrations..."
- Verify: All migrations run in order
- Check: "Migration completed: 001_create_graph_tables.sql"
- Verify: All 10 migrations complete successfully

STEP 4: Verify schema after migrations
- Connect: psql -d idp_orchestrator_test
- List tables: \dt
- Expected tables:
  - graph_nodes
  - graph_edges
  - applications
  - workflow_executions
  - workflow_steps
  - workflow_logs
  - resources
  - resource_state_transitions
  - sessions
  - api_keys
  - queue_tasks (if applicable)
- Verify: All expected tables exist

STEP 5: Check foreign key integrity
- Query: \d graph_edges
- Verify: Foreign keys to graph_nodes
- Query: \d resources
- Check: Foreign keys to applications
- Verify: ON DELETE CASCADE where appropriate
- Check: Constraints prevent orphaned records

STEP 6: Test migration idempotency
- Stop server
- Start again: ./innominatus
- Check logs: "Database schema is up to date"
- Verify: No duplicate migrations run
- Check: No errors about existing tables

STEP 7: Test migration rollback (if supported)
- Note: Current migration version
- Check: Does system support DOWN migrations?
- If yes:
  - Run: ./innominatus migrate down
  - Verify: Last migration reversed
  - Check: Tables dropped or altered back
  - Run: ./innominatus migrate up
  - Verify: Migration reapplied

STEP 8: Test partial migration failure
- Create invalid migration:
  - Create: 011_test_invalid.sql
  - Content: CREATE TABL invalid_syntax;  # Typo
- Start: ./innominatus
- Expected: Migration fails
- Check logs: Error message from database
- Verify: Server doesn't start OR rolls back
- Check: Database in consistent state
- Remove: 011_test_invalid.sql

STEP 9: Test migration with data preservation
- Deploy some applications
- Create resources
- Note: Record counts in each table
- Add new migration:
  - Create: 011_add_resource_tags.sql
  - Content: ALTER TABLE resources ADD COLUMN tags JSONB;
- Restart: ./innominatus
- Verify: Migration runs successfully
- Check: Existing data still present
- Verify: New column exists
- Check: SELECT tags FROM resources; (should be NULL for existing records)

STEP 10: Check migration tracking table
- Query: SELECT * FROM schema_migrations;
- Verify: Shows version numbers
- Check: dirty flag (indicates failed migration)
- Verify: Timestamp for each migration

COMMON FAILURE SCENARIOS TO CHECK:
1. ❌ Migration numbering gap → Ensure sequential
2. ❌ Foreign key violation → Check constraint definitions
3. ❌ Migration runs twice → Verify tracking mechanism
4. ❌ Data loss during migration → Test with backup data
5. ❌ Migration deadlock → Check for concurrent access
6. ❌ Schema version mismatch → Verify version tracking

BUGS TO REPORT:
- Migration failures
- Schema inconsistencies
- Data loss issues
- Foreign key problems
- Performance degradation after migration
```

---

# Part 4: Cross-Cutting Integration Tests

## INT-1: CLI Functionality

**User Story:** Verify all CLI commands work correctly across different scenarios.

### Test Prompt for Claude:

```
Please test complete CLI functionality:

STEP 1: Test CLI help and documentation
- Run: ./innominatus-ctl --help
- Verify: Shows command list
- Check: 31 commands visible
- For each major command:
  - Run: ./innominatus-ctl <command> --help
  - Verify: Shows usage, flags, examples
  - Check: Help text is clear and accurate

STEP 2: Test shell completion
- Generate: ./innominatus-ctl completion bash > /tmp/innominatus-completion
- Source: source /tmp/innominatus-completion
- Test: Type "innominatus-ctl " and press TAB
- Expected: Shows command completions
- Type: "innominatus-ctl list-" and press TAB
- Expected: Shows list-* commands

STEP 3: Test global flags
- Run: ./innominatus-ctl --server http://localhost:8081 list
- Verify: Connects to specified server
- Run: ./innominatus-ctl --details list
- Check: Shows more detailed output
- Run: ./innominatus-ctl --output json list
- Verify: Returns JSON format
- Run: ./innominatus-ctl --skip-validation deploy invalid.yaml
- Check: Skips validation (use with caution)

STEP 4: Test application commands
- deploy: ./innominatus-ctl deploy app.yaml
- list: ./innominatus-ctl list
- status: ./innominatus-ctl status my-app
- delete: ./innominatus-ctl delete my-app
- deprovision: ./innominatus-ctl deprovision my-app
- Verify: Each command works correctly

STEP 5: Test workflow commands
- list-workflows: ./innominatus-ctl list-workflows my-app
- workflow detail: ./innominatus-ctl workflow detail <id>
- workflow logs: ./innominatus-ctl workflow logs <id>
- workflow logs with filter: ./innominatus-ctl workflow logs <id> --step init
- retry: ./innominatus-ctl retry <id> app.yaml (if implemented)
- Verify: All commands produce expected output

STEP 6: Test resource commands
- list-resources: ./innominatus-ctl list-resources my-app
- get-resource: ./innominatus-ctl get-resource <id>
- list-resources by type: ./innominatus-ctl list-resources --type postgres
- Verify: Filtering and display work correctly

STEP 7: Test provider commands
- list-providers: ./innominatus-ctl list-providers
- get-provider: ./innominatus-ctl get-provider database-team
- list-goldenpaths: ./innominatus-ctl list-goldenpaths
- run golden path: ./innominatus-ctl run onboard-team --param team=qa
- Verify: Provider operations work

STEP 8: Test validation commands
- validate: ./innominatus-ctl validate app.yaml
- analyze: ./innominatus-ctl analyze app.yaml (if implemented)
- Verify: Catches errors in Score specs

STEP 9: Test demo commands
- demo-time: ./innominatus-ctl demo-time
- demo-status: ./innominatus-ctl demo-status
- demo-nuke: ./innominatus-ctl demo-nuke
- Verify: Demo environment management works

STEP 10: Test error handling
- Run: ./innominatus-ctl status nonexistent-app
- Expected: Clear error message
- Run: ./innominatus-ctl deploy missing-file.yaml
- Expected: "File not found" error
- Run: ./innominatus-ctl --server http://invalid-host list
- Expected: Connection error with helpful message

BUGS TO REPORT:
- Commands that fail unexpectedly
- Unclear error messages
- Help text inaccuracies
- Flag conflicts or bugs
- Output formatting issues
```

---

## INT-2: Web UI Navigation and Forms

**User Story:** Verify web UI provides intuitive navigation and working forms.

### Test Prompt for Claude:

```
Please test complete web UI functionality:

STEP 1: Test navigation
- Open: http://localhost:8081
- Check: Main navigation visible
- Click each nav item:
  - Dashboard
  - Applications
  - Resources
  - Workflows
  - Providers
  - Golden Paths
  - Admin (if admin user)
  - Profile
- Verify: All pages load without errors

STEP 2: Test dashboard
- Navigate to: /dashboard
- Check: Application statistics card
- Check: Resource status breakdown
- Check: Recent activity timeline
- Verify: Charts/graphs render correctly
- Check: Links to detail pages work

STEP 3: Test applications page
- Navigate to: /apps
- Verify: Application list loads
- Check: Search/filter functionality
- Click: Application name
- Expected: Application details pane opens
- Check: Tabs (Overview, Score Spec, Workflows, Resources)
- Verify: All tabs load content

STEP 4: Test deployment wizard (recent fix)
- Navigate to: /apps
- Click: "Deploy New" button
- Verify: Wizard dialog opens
- Step 1 - Basic Info:
  - Fill: App name, environment, TTL
  - Check: Validation (name must be DNS-compliant)
  - Click: Next
- Step 2 - Container:
  - Fill: Image, port, env vars
  - Check: Validation (image required)
  - Click: Next
- Step 3 - Resources:
  - Add: postgres resource
  - Add: s3-bucket resource
  - Click: Next
- Step 4 - Review:
  - Check: Generated YAML preview
  - Verify: YAML is valid
  - Click: Deploy
- Expected: Deployment starts
- Verify: No "Error parsing YAML" (recent fix - sends raw YAML, not JSON)
- Check: Redirects to workflow page or app detail

STEP 5: Test application detail page (recent addition)
- Navigate to: /dev/applications
- Click: "View" on an application
- Expected: Detail page loads at /dev/applications/{name}
- Verify: Shows:
  - Application name and status badge
  - Environment card
  - Resources count
  - Resources table with name, type, state, provider
- Check: Back button returns to list

STEP 6: Test adding resources (recent addition)
- On application detail page:
  - Click: "Add Resource" button
  - Fill: Resource name (e.g., "cache")
  - Select: Resource type (e.g., "PostgreSQL Database")
  - Click: "Add"
- Expected: Resource created
- Verify: Resource appears in table
- Check: State shows "requested" or "provisioning"

STEP 7: Test resources page
- Navigate to: /resources
- Verify: Resource list loads
- Check: Filter by application
- Check: Filter by type
- Check: Filter by state
- Click: Resource
- Expected: Resource details pane opens
- Verify: Shows configuration, hints, state history

STEP 8: Test workflows page
- Navigate to: /workflows
- Verify: Workflow execution list
- Check: Filter by application
- Check: Filter by status
- Click: Workflow
- Expected: Workflow detail page
- Verify: Shows:
  - Metadata (name, status, duration)
  - Steps with status badges
  - Logs viewer
- Click: Step
- Expected: Shows step details and logs

STEP 9: Test providers page
- Navigate to: /providers
- Verify: Provider cards/list
- Check: Shows category, capabilities
- Click: Provider
- Expected: Provider detail page
- Verify: Shows:
  - Workflows list
  - Resource types handled
  - Statistics (if available)

STEP 10: Test profile page
- Navigate to: /profile
- Verify: User info displayed
- Check: API Keys section
- Click: "Generate New API Key"
- Fill: Name and expiry
- Click: Generate
- Verify: Key displayed with warning to copy
- Check: Key appears in list
- Click: Revoke
- Verify: Key removed

STEP 11: Test admin pages (admin only)
- Navigate to: /admin/users
- Verify: User list (if implemented)
- Check: Can view user details
- Navigate to: /admin/settings
- Check: Platform configuration options
- Navigate to: /admin/integrations
- Verify: Integration status (Gitea, Vault, etc.)

STEP 12: Test dark mode (if implemented)
- Find: Theme toggle
- Switch: To dark mode
- Verify: Colors update correctly
- Check: All pages readable in dark mode
- Switch back: To light mode

STEP 13: Test responsive design
- Resize: Browser to mobile width
- Check: Navigation becomes hamburger menu
- Verify: Tables become scrollable or stack
- Check: All functionality still accessible

STEP 14: Test error states
- Navigate to: /apps/nonexistent-app
- Expected: "Not found" or error message
- Navigate to: /workflows/invalid-id
- Expected: Clear error message
- Disconnect: Network
- Try: Load page
- Expected: Connection error message

BUGS TO REPORT:
- Pages that fail to load
- Broken links
- Form validation issues
- UI components not rendering
- Error messages unclear
- Responsive design problems
- Deployment wizard sending invalid data (should be fixed)
```

---

## INT-3: API Endpoint Validation

**User Story:** Verify all REST API endpoints work correctly and return proper status codes.

### Test Prompt for Claude:

```
Please test all API endpoints systematically:

SETUP: Get authentication token
- Login: curl -X POST http://localhost:8081/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}'
- Extract: Token from response
- Set: export TOKEN="<token>"

STEP 1: Test health and status endpoints
- GET /health
  - Expected: 200, {"status": "healthy"}
- GET /ready
  - Expected: 200, detailed readiness status
- GET /metrics
  - Expected: 200, Prometheus metrics format

STEP 2: Test application endpoints
- POST /api/applications (or /api/specs)
  - Headers: Authorization: Bearer $TOKEN, Content-Type: application/yaml
  - Body: Valid Score YAML (raw, not JSON-wrapped - recent fix)
  - Expected: 201 Created
  - Verify: Returns application details
- GET /api/applications
  - Expected: 200, array of applications
- GET /api/applications/{name}
  - Expected: 200, application details
- DELETE /api/applications/{name}
  - Expected: 204 No Content or 200
- POST /api/applications/{name}/deprovision
  - Expected: 200, deprovision started

STEP 3: Test resource endpoints (recent addition)
- GET /api/resources
  - Expected: 200, map of resources by app
- GET /api/resources?app={name}
  - Expected: 200, filtered resources
- GET /api/resources/{id}
  - Expected: 200, resource details
- POST /api/resources (recent addition)
  - Headers: Authorization: Bearer $TOKEN, Content-Type: application/json
  - Body: {
      "application_name": "test-app",
      "resource_name": "test-resource",
      "resource_type": "postgres",
      "configuration": {}
    }
  - Expected: 201 Created
  - Verify: Returns created resource
- PUT /api/resources/{id}
  - Expected: 200, updated resource
- DELETE /api/resources/{id}
  - Expected: 204 No Content

STEP 4: Test workflow endpoints
- GET /api/workflows
  - Expected: 200, workflow executions list
- GET /api/workflows?app={name}
  - Expected: 200, filtered by application
- GET /api/workflows/{id}
  - Expected: 200, workflow details with steps
- GET /api/workflows/{id}/logs
  - Expected: 200, workflow logs
- GET /api/workflows/{id}/logs?step={step-name}
  - Expected: 200, filtered logs

STEP 5: Test golden path endpoints
- GET /api/goldenpaths
  - Expected: 200, list of golden paths
- POST /api/goldenpaths/{name}/execute
  - Body: {"inputs": {"param1": "value1"}}
  - Expected: 200, workflow execution started

STEP 6: Test provider endpoints
- GET /api/providers
  - Expected: 200, list of providers
- GET /api/providers/{name}
  - Expected: 200, provider details
- GET /api/providers/stats
  - Expected: 200, provider statistics

STEP 7: Test admin endpoints (admin token required)
- POST /api/admin/providers
  - Body: Provider registration
  - Expected: 201 Created
- POST /api/admin/providers/reload
  - Expected: 200, providers reloaded
- GET /api/admin/users (if implemented)
  - Expected: 200, user list

STEP 8: Test profile endpoints
- GET /api/profile
  - Expected: 200, user profile
- GET /api/profile/api-keys
  - Expected: 200, list of API keys
- POST /api/profile/api-keys
  - Body: {"name": "test-key", "expiry_days": 30}
  - Expected: 201, new API key
- DELETE /api/profile/api-keys/{id}
  - Expected: 204 No Content

STEP 9: Test OIDC endpoints
- GET /api/oidc/config
  - Expected: 200, OIDC configuration (if enabled)
- POST /api/oidc/token
  - Body: {"code": "...", "code_verifier": "..."}
  - Expected: 200, access token

STEP 10: Test error responses
- GET /api/applications/nonexistent
  - Expected: 404 Not Found
- POST /api/applications (missing metadata.name - recent validation)
  - Expected: 400 Bad Request, "metadata.name is required"
- POST /api/applications (invalid YAML)
  - Expected: 400 Bad Request, "Error parsing YAML"
- POST /api/resources (no auth)
  - Expected: 401 Unauthorized
- POST /api/admin/providers (non-admin user)
  - Expected: 403 Forbidden

STEP 11: Test CORS headers
- OPTIONS /api/applications
  - Expected: Appropriate CORS headers
- Check: Access-Control-Allow-Origin
- Check: Access-Control-Allow-Methods
- Check: Access-Control-Allow-Headers

STEP 12: Test rate limiting (if implemented)
- Make: 100 rapid requests to /api/applications
- Check: Rate limit headers (X-RateLimit-*)
- Verify: 429 Too Many Requests after threshold

BUGS TO REPORT:
- Incorrect status codes
- Missing or wrong headers
- JSON validation errors (wrong structure - e.g., wrapped YAML - recent fix)
- Authentication/authorization failures
- Missing endpoints
- Inconsistent error response formats
```

---

## INT-4: SSO Login Flow (Recent Fix)

**User Story:** Verify SSO login works correctly after recent fixes.

### Test Prompt for Claude:

```
Please test the complete SSO/OIDC login flow:

PREREQUISITES:
- Demo environment running (./innominatus-ctl demo-time)
- Keycloak accessible at http://keycloak.localtest.me
- innominatus server running with OIDC enabled

STEP 1: Verify Keycloak client configuration
- Open: http://keycloak.localtest.me
- Login: admin/admin
- Navigate: Clients → innominatus
- Check: Redirect URIs include:
  - http://localhost:8081/auth/callback (Web UI)
  - http://innominatus.localtest.me/auth/callback (Web UI alternate)
  - http://127.0.0.1:8082/callback (CLI SSO - RECENT FIX)
- Verify: Client is enabled
- Check: Access Type = confidential or public
- Verify: Client secret matches server config

STEP 2: Test Web UI SSO login
- Open: http://localhost:8081/login
- Verify: "Login with SSO" button visible
- Click: "Login with SSO"
- Expected: Redirects to Keycloak at:
  http://keycloak.localtest.me/realms/demo-realm/protocol/openid-connect/auth?...
- Check: URL parameters include:
  - client_id=innominatus
  - redirect_uri=http://localhost:8081/auth/callback
  - response_type=code
  - scope=openid profile email roles
- Login: admin/admin (in Keycloak)
- Expected: Redirects back to http://localhost:8081/auth/callback
- Verify: Callback succeeds (NO "invalid redirect URI" error - RECENT FIX)
- Check: User logged into innominatus
- Verify: Session created
- Check: Username displayed in UI

STEP 3: Test CLI SSO login (RECENT FIX)
- Run: ./innominatus-ctl login --sso
- Expected: CLI output: "Opening browser for authentication..."
- Verify: Browser opens to Keycloak login page
- Check: Redirect URI in URL includes redirect_uri=http://127.0.0.1:8082/callback
- Login: admin/admin (in Keycloak)
- Expected: Browser redirects to http://127.0.0.1:8082/callback
- Verify: Callback succeeds (RECENT FIX - fixed port 8082)
- Check: CLI displays "✓ Authentication Successful"
- Verify: CLI receives and stores token
- Test: ./innominatus-ctl list
- Expected: Command succeeds without additional auth

STEP 4: Verify CLI callback server (RECENT FIX)
- During CLI SSO login:
  - Check: CLI starts local HTTP server on port 8082
  - Verify: Port is fixed (not random - RECENT FIX)
  - Check: Server listens on http://127.0.0.1:8082
  - Verify: Accepts callback at /callback endpoint
  - Check: Server shuts down after receiving callback

STEP 5: Test PKCE flow (if implemented)
- During login:
  - Check: code_challenge parameter in auth URL
  - Verify: code_challenge_method=S256
- During token exchange:
  - Check: code_verifier sent to token endpoint
  - Verify: Token exchange succeeds

STEP 6: Test token storage and reuse
- After CLI SSO login:
  - Check: Token stored (e.g., ~/.innominatus/token)
  - Verify: Token includes expiry
  - Run: ./innominatus-ctl list (multiple times)
  - Check: Token reused, no repeated login

STEP 7: Test token expiry handling
- Wait for token to expire OR manually expire
- Run: ./innominatus-ctl list
- Expected: Error indicating expired token
- OR: Auto-refresh if refresh token available
- Run: ./innominatus-ctl login --sso
- Expected: Can re-authenticate

STEP 8: Test Web UI session expiry
- Login via Web UI
- Wait for session timeout
- OR expire session in database
- Refresh: page
- Expected: Redirects to /login
- Check: Clear message about session expiration

STEP 9: Test logout
- Web UI: Click logout button
- Expected: Session cleared
- Verify: Redirected to login page
- Try: Access protected page
- Expected: Redirected to login

STEP 10: Test error scenarios
- Scenario 1: Wrong credentials in Keycloak
  - Login with invalid password
  - Expected: Keycloak shows error
  - Verify: innominatus shows error message

- Scenario 2: User cancels login
  - Start SSO login
  - Cancel in Keycloak
  - Expected: Callback receives error
  - Verify: Clear error message to user

- Scenario 3: Keycloak down
  - Stop Keycloak (docker stop keycloak)
  - Try: SSO login
  - Expected: Error about unable to connect
  - Start Keycloak back

- Scenario 4: Invalid callback (wrong port - old bug)
  - Temporarily change CLI callback port (simulate old behavior)
  - Try: CLI SSO login
  - Expected: Would fail with "invalid redirect URI"
  - Verify: FIXED - port 8082 is registered

BUGS TO REPORT:
- Callback URL errors (should be fixed)
- Token storage issues
- Session management problems
- Error message clarity
- PKCE flow issues
- Browser opening failures
```

---

## INT-5: End-to-End Application Lifecycle

**User Story:** Verify complete application lifecycle from deployment to deletion.

### Test Prompt for Claude:

```
Please test complete end-to-end application lifecycle:

SCENARIO: Deploy e-commerce application with multiple resources

STEP 1: Create comprehensive Score specification
- Create: ecommerce-app.yaml
- Content:
  ```yaml
  apiVersion: score.dev/v1b1
  metadata:
    name: ecommerce-backend
    labels:
      environment: production
      team: ecommerce
      cost-center: engineering

  containers:
    api:
      image: mycompany/ecommerce-api:v2.1.0
      variables:
        NODE_ENV: production
        LOG_LEVEL: info
        DB_POOL_SIZE: "20"
      resources:
        limits:
          memory: 2Gi
          cpu: 1000m
        requests:
          memory: 1Gi
          cpu: 500m

    worker:
      image: mycompany/ecommerce-worker:v2.1.0
      variables:
        QUEUE_CONCURRENCY: "5"

  service:
    ports:
      http:
        port: 8080
        targetPort: 8080
      metrics:
        port: 9090
        targetPort: 9090

  resources:
    database:
      type: postgres
      params:
        version: "15"
        size: medium
        replicas: 3
        backup_enabled: true

    cache:
      type: redis
      params:
        version: "7"
        memory: 4Gi

    storage:
      type: s3-bucket
      params:
        region: us-east-1
        versioning: true
        encryption: true

    secrets:
      type: vault-space
      params:
        namespace: ecommerce-backend

    repository:
      type: gitea-repo
      params:
        org: ecommerce-team
        private: true

    deployment:
      type: argocd-app
      params:
        repoURL: ${resources.repository.outputs.clone_url}
        path: k8s/manifests
        syncPolicy: automated
  ```

STEP 2: Validate Score specification
- Run: ./innominatus-ctl validate ecommerce-app.yaml
- Expected: Validation passes
- Check: No errors about:
  - Missing required fields
  - Invalid resource types
  - Syntax errors
- Verify: All resource types have registered providers

STEP 3: Deploy application
- Run: ./innominatus-ctl deploy ecommerce-app.yaml
- Expected: Deployment accepted
- Verify: Application created
- Check: Returns application ID or confirmation

STEP 4: Monitor initial deployment
- Run: ./innominatus-ctl status ecommerce-backend
- Expected: Status = "provisioning" or "pending"
- Check: Environment = production
- Check: Team = ecommerce
- Wait: 30 seconds
- Run again: ./innominatus-ctl status ecommerce-backend
- Verify: Status progressing

STEP 5: Track resource provisioning
- Run: ./innominatus-ctl list-resources ecommerce-backend
- Expected: Shows 6 resources:
  - database (postgres)
  - cache (redis)
  - storage (s3-bucket)
  - secrets (vault-space)
  - repository (gitea-repo)
  - deployment (argocd-app)
- For each resource:
  - Check: State is "requested", "provisioning", or "active"
  - Verify: Provider assigned
  - Check: Workflow execution ID present

STEP 6: Monitor workflow executions
- Run: ./innominatus-ctl list-workflows ecommerce-backend
- Expected: Shows 6 workflow executions (one per resource)
- For each workflow:
  - Check: Status (running, completed, or failed)
  - Note: Workflow ID
  - Run: ./innominatus-ctl workflow detail <workflow-id>
  - Verify: Shows steps and progress

STEP 7: View detailed logs
- For database provisioning:
  - Run: ./innominatus-ctl workflow logs <db-workflow-id>
  - Check: Shows PostgreSQL cluster creation
  - Verify: Operator CRD applied
  - Check: Replicas = 3
- For storage provisioning:
  - Run: ./innominatus-ctl workflow logs <storage-workflow-id>
  - Check: Shows MinIO bucket creation
  - Verify: Encryption enabled
- Continue for all resources

STEP 8: Verify resources become active
- Wait: For all workflows to complete (may take 2-10 minutes)
- Run: ./innominatus-ctl list-resources ecommerce-backend
- Expected: All resources state = "active"
- For each resource:
  - Run: ./innominatus-ctl get-resource <resource-id>
  - Check: health_status = "healthy"
  - Verify: Configuration matches Score spec
  - Check: Hints/connection details populated

STEP 9: Access provisioned resources
- Database:
  - Check hints for connection string
  - Verify: PostgreSQL cluster exists
  - Run: kubectl get postgrescluster -n ecommerce-backend
  - Check: 3 replicas running

- Storage:
  - Open: http://minio.localtest.me
  - Verify: Bucket exists for ecommerce-backend
  - Check: Encryption and versioning enabled

- Repository:
  - Open: http://gitea.localtest.me/ecommerce-team
  - Verify: Repository created
  - Check: Is private

- ArgoCD:
  - Open: http://argocd.localtest.me
  - Verify: Application deployed
  - Check: Sync status

STEP 10: Update application (scale database)
- Modify: ecommerce-app.yaml
- Change: database replicas from 3 to 5
- Run: ./innominatus-ctl deploy ecommerce-app.yaml
- OR: ./innominatus-ctl update ecommerce-backend ecommerce-app.yaml
- Expected: Update workflow starts
- Run: ./innominatus-ctl list-workflows ecommerce-backend
- Verify: New workflow with operation="update"
- Wait: For completion
- Check: Database replicas scaled to 5

STEP 11: View application graph
- Open: http://localhost:8081/graph
- Search: ecommerce-backend
- Verify: Graph shows:
  - Spec node (ecommerce-backend)
  - 6 resource nodes
  - Provider nodes (database-team, storage-team, etc.)
  - Workflow nodes
  - Edges connecting them
- Check: Node colors indicate status
- Click: Nodes to see details

STEP 12: Test application deletion (deprovision)
- Run: ./innominatus-ctl deprovision ecommerce-backend
- Expected: Deprovision workflows start
- Verify: Delete workflows created for each resource
- Wait: For completion
- Run: ./innominatus-ctl list-resources ecommerce-backend
- Expected: Resources state = "terminating" or "terminated"
- Check: Resources cleaned up:
  - PostgreSQL cluster deleted
  - MinIO bucket deleted (or marked for deletion)
  - Gitea repository archived/deleted
  - ArgoCD app removed

STEP 13: Remove application
- Run: ./innominatus-ctl delete ecommerce-backend
- Expected: Application removed from system
- Run: ./innominatus-ctl list
- Verify: ecommerce-backend not in list
- Run: ./innominatus-ctl status ecommerce-backend
- Expected: "Application not found"

STEP 14: Verify cleanup
- Check: Kubernetes namespace deleted
- Check: Database cleaned up
- Check: Storage bucket removed
- Check: No orphaned resources
- Run: kubectl get all -n ecommerce-backend
- Expected: Namespace not found or empty

TOTAL TIME: Record total time from deploy to fully active
- Expected: 5-15 minutes depending on resources

BUGS TO REPORT:
- Resources that fail to provision
- Workflows that hang or timeout
- Resource state inconsistencies
- Cleanup failures
- Graph visualization errors
- Any step that deviates from expected behavior
```

---

# Summary

This comprehensive testing document provides systematic prompts for Claude to verify:

- **15 Critical User Stories** across 3 personas
- **5 Integration Tests** covering CLI, Web UI, API, SSO, and E2E workflows
- **Recent fixes verified**: SSO callback URL (fixed port 8082), Web UI YAML payload (raw YAML, not JSON), Application detail view, Resource creation endpoint

**How to Use:**
1. Run tests in order (AD-1 through INT-5)
2. Document all findings
3. Create bug reports for failures
4. Retest after fixes
5. Achieve green status before demo

**Success Criteria for Demo:**
- All "Application Developer" tests pass ✅
- Core workflows complete without errors ✅
- Web UI deployment wizard works ✅
- SSO login successful ✅
- E2E application lifecycle completes ✅
