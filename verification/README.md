# Verification-First Development

**Philosophy**: Write tests before code. Define success criteria first, then implement until verification passes.

## Overview

Verification-first development ensures:
- **Clear Success Criteria**: You know what "done" looks like before starting
- **Faster Iteration**: Quickly verify if implementation meets requirements
- **Better Design**: Thinking about verification reveals edge cases early
- **Confidence**: Green verification = feature works as intended

## Verification Workflow

```bash
# 1. Create verification script from template
cp verification/template.mjs verification/my-feature.mjs

# 2. Define success criteria (edit my-feature.mjs)
# - What should the API return?
# - What database state should exist?
# - What files should be created?

# 3. Run verification (should FAIL initially)
node verification/my-feature.mjs

# 4. Implement the feature
# ... code changes ...

# 5. Re-run verification
node verification/my-feature.mjs

# 6. Iterate until verification PASSES
# Repeat steps 4-5 until green
```

## Example: API Endpoint Verification

**Scenario**: Add new `/api/workflows/{id}/status` endpoint

### Step 1: Create Verification Script

```bash
cp verification/template.mjs verification/workflow-status-endpoint.mjs
```

### Step 2: Define Success Criteria

```javascript
async function verify() {
  const baseURL = 'http://localhost:8081';
  const apiKey = process.env.IDP_API_KEY;

  // Test: GET /api/workflows/123/status
  const response = await fetch(`${baseURL}/api/workflows/123/status`, {
    headers: { 'Authorization': `Bearer ${apiKey}` }
  });

  // Assertion 1: Status code should be 200
  if (response.status !== 200) {
    throw new Error(`Expected 200, got ${response.status}`);
  }

  // Assertion 2: Response should have correct structure
  const data = await response.json();
  if (!data.id || !data.status || !data.started_at) {
    throw new Error('Missing required fields in response');
  }

  // Assertion 3: Status should be valid enum value
  const validStatuses = ['succeeded', 'running', 'failed', 'pending'];
  if (!validStatuses.includes(data.status)) {
    throw new Error(`Invalid status: ${data.status}`);
  }

  console.log('✅ Verification PASSED');
}
```

### Step 3: Run (Should Fail)

```bash
$ node verification/workflow-status-endpoint.mjs
❌ Verification FAILED
Error: Expected 200, got 404
```

### Step 4: Implement Feature

```go
// internal/server/handlers.go
func handleWorkflowStatus(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    workflow, err := db.GetWorkflowByID(id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    response := map[string]interface{}{
        "id":         workflow.ID,
        "status":     workflow.Status,
        "started_at": workflow.StartedAt,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### Step 5: Run Again (Should Pass)

```bash
$ node verification/workflow-status-endpoint.mjs
✅ Verification PASSED
```

## Verification Types

### 1. API Endpoint Verification

Test HTTP endpoints:
```javascript
const response = await fetch(`${baseURL}/api/endpoint`, {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${apiKey}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ key: 'value' }),
});

if (!response.ok) {
  throw new Error(`API failed: ${response.status}`);
}

const data = await response.json();
// Assert on response data
```

### 2. Database State Verification

Test database state (requires database access):
```javascript
import pg from 'pg';

const client = new pg.Client({
  host: 'localhost',
  port: 5432,
  database: 'idp_orchestrator',
  user: 'orchestrator_user',
  password: process.env.DB_PASSWORD,
});

await client.connect();

const result = await client.query(
  'SELECT * FROM workflows WHERE application_name = $1',
  ['test-app']
);

if (result.rows.length === 0) {
  throw new Error('Expected workflow not found in database');
}

await client.end();
```

### 3. File System Verification

Test file creation:
```javascript
import fs from 'fs/promises';

const filePath = './workspace/test-app/terraform.tfstate';

try {
  const content = await fs.readFile(filePath, 'utf-8');
  const state = JSON.parse(content);

  if (!state.version || !state.resources) {
    throw new Error('Terraform state file has invalid structure');
  }
} catch (error) {
  throw new Error(`Expected file not found: ${filePath}`);
}
```

### 4. Process Verification

Test running processes:
```javascript
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

const { stdout } = await execAsync('ps aux | grep innominatus');

if (!stdout.includes('cmd/server/main.go')) {
  throw new Error('innominatus server not running');
}
```

### 5. WebSocket Verification

Test real-time connections:
```javascript
import WebSocket from 'ws';

const ws = new WebSocket(`ws://localhost:8081/api/graph/test-app/ws?token=${apiKey}`);

await new Promise((resolve, reject) => {
  ws.on('open', resolve);
  ws.on('error', reject);
});

ws.on('message', (data) => {
  const update = JSON.parse(data);
  if (!update.nodes || !update.edges) {
    throw new Error('Invalid WebSocket message structure');
  }
});

ws.close();
```

## Best Practices

### 1. One Verification Per Feature

Don't create monolithic verification scripts. Each feature should have its own verification:

```
verification/
├── api-graph-history.mjs
├── api-critical-path.mjs
├── websocket-updates.mjs
└── golden-path-deploy.mjs
```

### 2. Clear Assertions

Make assertions explicit and descriptive:

```javascript
// ❌ BAD: Unclear assertion
if (data.count < 1) throw new Error('bad count');

// ✅ GOOD: Clear assertion
if (data.count !== 3) {
  throw new Error(`Expected 3 workflows, got ${data.count}`);
}
```

### 3. Cleanup After Tests

Always clean up test data:

```javascript
// Create test data
await createTestWorkflow('test-app');

try {
  // Run verification
  await verify();
} finally {
  // Always cleanup, even if verification fails
  await deleteTestWorkflow('test-app');
}
```

### 4. Environment Configuration

Use environment variables for configuration:

```javascript
const baseURL = process.env.BASE_URL || 'http://localhost:8081';
const apiKey = process.env.IDP_API_KEY;
const dbHost = process.env.DB_HOST || 'localhost';
```

### 5. Descriptive Output

Provide clear feedback during verification:

```javascript
console.log('🔍 Verifying workflow execution...');
console.log('  ✓ Workflow created successfully');
console.log('  ✓ Steps executed in correct order');
console.log('  ✓ Final status is "succeeded"');
console.log('✅ Verification PASSED');
```

## Integration with Development Workflow

### During Feature Development

```bash
# 1. Create verification script
cp verification/template.mjs verification/my-feature.mjs

# 2. Define success criteria
vim verification/my-feature.mjs

# 3. Start development loop
while true; do
  # Implement feature
  vim internal/server/handlers.go

  # Rebuild server
  go build -o innominatus cmd/server/main.go

  # Restart server (in another terminal)
  ./innominatus

  # Run verification
  node verification/my-feature.mjs

  # If passes, break; otherwise, iterate
done
```

### In CI/CD Pipeline

Add verification scripts to GitHub Actions:

```yaml
# .github/workflows/verify.yml
name: Verify

on: [push, pull_request]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - uses: actions/setup-node@v3

      - name: Start server
        run: |
          go build -o innominatus cmd/server/main.go
          ./innominatus &

      - name: Run verifications
        env:
          IDP_API_KEY: ${{ secrets.TEST_API_KEY }}
        run: |
          for script in verification/*.mjs; do
            echo "Running $script"
            node "$script"
          done
```

## Real-World Verification Examples

### Example 1: Provider Registration Verification

**File:** `verification/examples/test-provider-registration.mjs`

**What it tests:**
- Provider loading from filesystem and Git sources
- Capability registration without conflicts
- Provider resolution algorithm (resource type → provider)
- Workflow retrieval for resource types

**Usage:**
```bash
node verification/examples/test-provider-registration.mjs
```

**Expected outputs:**
```
✅ List Providers: 6 providers found (database-team, container-team, storage-team, etc.)
✅ Provider Capabilities: No conflicts detected, 30+ resource types covered
✅ Provider Resolution: postgres → database-team, s3 → storage-team, etc.
✅ Provisioner Workflows: All providers have provisioner workflows
```

**Verification criteria:**
- All providers load successfully
- No capability conflicts (one resource type = one provider)
- Resource types resolve to correct providers
- Each provider with capabilities has at least one provisioner workflow

**Output directory:** `docs/verification/provider-registration/`

---

### Example 2: Workflow Execution End-to-End

**File:** `verification/examples/test-workflow-execution.mjs`

**What it tests:**
- Complete orchestration flow: Score spec → Resource → Provider → Workflow
- Resource state transitions: requested → provisioning → active
- Workflow execution: steps, logs, status updates
- Graph relationship creation: spec → resource → provider → workflow

**Usage:**
```bash
node verification/examples/test-workflow-execution.mjs
```

**Flow:**
```
1. Submit Score spec with postgres resource
2. Verify resource created (state='requested')
3. Wait for orchestration engine to pick up resource (polls every 5s)
4. Monitor workflow execution (step-by-step status)
5. Verify resource becomes 'active'
6. Verify graph relationships created
7. Cleanup (delete spec)
```

**Expected outputs:**
```
✅ Submit Score Spec: verification-test-app created
✅ Resource Created: postgres resource (state='requested')
✅ Orchestration Pickup: Engine assigned workflow execution ID
✅ Workflow Execution: provision-postgres completed
✅ Resource Active: Resource state = 'active'
✅ Graph Relationships: 4 nodes, 3 edges created
✅ Cleanup: Spec deleted
```

**Verification criteria:**
- Spec submission successful
- Resource created with state='requested'
- Orchestration engine picks up resource within 10s
- Workflow executes all steps successfully
- Resource state becomes 'active'
- Graph has complete dependency chain

**Output directory:** `docs/verification/workflow-execution/`

---

### Example 3: API Endpoints Health Check

**File:** `verification/examples/test-api-endpoints.mjs`

**What it tests:**
- All API endpoints respond correctly
- Authentication works (file-based, API key, OIDC)
- Data structures match expected schema
- Swagger documentation accessible
- WebSocket connectivity

**Usage:**
```bash
export INNOMINATUS_API_KEY=your-api-key
node verification/examples/test-api-endpoints.mjs
```

**Endpoints tested:**
```
✅ /health                          (200 OK)
✅ /ready                           (200 OK)
✅ /metrics                         (200 OK, Prometheus format)
✅ /swagger-user                    (200 OK, API docs)
✅ /swagger-admin                   (200 OK, Admin API docs)
✅ GET /api/specs                   (200 OK, array)
✅ GET /api/workflows               (200 OK, array)
✅ GET /api/resources               (200 OK, array)
✅ GET /api/providers               (200 OK, array)
✅ GET /api/providers/{name}        (200 OK, provider details)
✅ GET /api/workflows/{id}          (200 OK, execution details)
✅ GET /api/workflows/{id}/graph    (200 OK, graph nodes/edges)
✅ GET /api/resources/{id}/graph    (200 OK, resource graph)
```

**Verification criteria:**
- All endpoints return expected status codes
- Response structures match API specification
- Authentication required for protected endpoints
- Swagger documentation accessible

**Output directory:** `docs/verification/api-endpoints/`

---

### Example 4: Provider Resolution Algorithm

**Scenario:** Verify that requesting a 'postgres' resource automatically triggers the correct provider

**Verification script:**
```javascript
// 1. Submit Score spec with postgres resource
const scoreSpec = {
  apiVersion: 'score.dev/v1b1',
  metadata: { name: 'test-db-app' },
  resources: {
    database: {
      type: 'postgres',  // Key: resource type
      properties: { version: '15' }
    }
  }
};

await submitSpec(scoreSpec);

// 2. Wait for orchestration engine
await sleep(6000);  // Wait > 5s (poll interval)

// 3. Check resource was assigned to correct provider
const resource = await getResource('test-db-app', 'postgres');
const execution = await getWorkflowExecution(resource.workflow_execution_id);

// Assertions:
assert(execution.workflow_name === 'provision-postgres');
assert(execution.provider === 'database-team');
assert(resource.state === 'provisioning' || resource.state === 'active');
```

**What it verifies:**
- Provider resolver correctly maps 'postgres' → 'database-team'
- Orchestration engine triggers 'provision-postgres' workflow
- Resource state transitions from 'requested' → 'provisioning' → 'active'

---

### Example 5: Graph Integrity Verification

**Scenario:** Verify that graph relationships are created correctly for multi-resource specs

**Verification script:**
```javascript
// Submit spec with multiple resources
const scoreSpec = {
  metadata: { name: 'full-stack-app' },
  resources: {
    database: { type: 'postgres' },
    storage: { type: 's3-bucket' },
    namespace: { type: 'kubernetes-namespace' }
  }
};

await submitSpec(scoreSpec);

// Wait for all resources to be provisioned
await waitForAllResourcesActive('full-stack-app');

// Get graph
const graph = await getSpecGraph('full-stack-app');

// Expected structure:
// spec: full-stack-app
//   ↓ contains
// resource: postgres
//   ↓ requires
// provider: database-team
//   ↓ executes
// workflow: provision-postgres

// Assertions:
const specNode = graph.nodes.find(n => n.node_type === 'spec');
const resourceNodes = graph.nodes.filter(n => n.node_type === 'resource');
const providerNodes = graph.nodes.filter(n => n.node_type === 'provider');
const workflowNodes = graph.nodes.filter(n => n.node_type === 'workflow');

assert(resourceNodes.length === 3, 'Should have 3 resources');
assert(providerNodes.length === 3, 'Should have 3 providers');
assert(workflowNodes.length === 3, 'Should have 3 workflows');

// Verify edges
const containsEdges = graph.edges.filter(e => e.edge_type === 'contains');
const requiresEdges = graph.edges.filter(e => e.edge_type === 'requires');

assert(containsEdges.length === 3, 'Spec should contain 3 resources');
assert(requiresEdges.length === 6, '3 resources → 3 providers → 3 workflows');
```

**What it verifies:**
- Graph nodes created for spec, resources, providers, workflows
- Edge types correct ('contains', 'requires')
- Complete dependency chain for each resource
- No orphaned edges (foreign key constraints)

---

### Example 6: OIDC Authentication Flow

**Scenario:** Verify OIDC authentication and token validation

**Verification script:**
```javascript
// 1. Initiate OIDC login
const authURL = `${API_BASE}/auth/login`;
const response = await fetch(authURL);

// Should redirect to OIDC provider
assert(response.status === 302);
const location = response.headers.get('location');
assert(location.includes('keycloak.example.com'));

// 2. Simulate OIDC callback (in real test, use Playwright)
const callbackURL = `${API_BASE}/auth/callback?code=test-code&state=test-state`;
const callbackResponse = await fetch(callbackURL);

// Should set session cookie
const cookies = callbackResponse.headers.get('set-cookie');
assert(cookies.includes('session='));

// 3. Verify authenticated request works
const apiResponse = await fetch(`${API_BASE}/api/specs`, {
  headers: { 'Cookie': cookies }
});

assert(apiResponse.status === 200);

// 4. Verify unauthenticated request fails
const unauthResponse = await fetch(`${API_BASE}/api/specs`);
assert(unauthResponse.status === 401 || unauthResponse.status === 403);
```

**What it verifies:**
- OIDC redirect flow works
- Session cookie created after authentication
- Authenticated requests succeed
- Unauthenticated requests blocked

---

## Complete Verification Examples

All examples are in `verification/examples/`:

| File | What It Tests | Duration |
|------|--------------|----------|
| `test-provider-registration.mjs` | Provider loading, capability resolution | 5-10s |
| `test-workflow-execution.mjs` | End-to-end orchestration flow | 30-60s |
| `test-api-endpoints.mjs` | All API endpoints health check | 10-15s |

**Run all verifications:**
```bash
for script in verification/examples/*.mjs; do
  echo "Running $(basename $script)..."
  node "$script" || echo "❌ FAILED"
done
```

## Tips

- **Start Simple**: Begin with basic API response verification
- **Add Edge Cases**: Once basic verification passes, add edge case tests
- **Keep It Fast**: Verifications should run in seconds, not minutes
- **Make It Repeatable**: Verification should pass every time if feature works
- **Use Real Environment**: Verify against actual server, not mocks
- **Save Outputs**: Store verification results in `docs/verification/` for AI analysis
- **Clean Up**: Always delete test data to avoid polluting database

## References

- **CLAUDE.md** - Verification-First Development Protocol section
- **verification/template.mjs** - Copy this to start new verification
- **verification/examples/** - See complete examples
- **docs/verification/** - Verification output directory
