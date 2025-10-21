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

## Examples

See `verification/examples/` for complete examples:
- **test-verification.mjs** - Example API endpoint verification
- More examples coming soon...

## Tips

- **Start Simple**: Begin with basic API response verification
- **Add Edge Cases**: Once basic verification passes, add edge case tests
- **Keep It Fast**: Verifications should run in seconds, not minutes
- **Make It Repeatable**: Verification should pass every time if feature works
- **Use Real Environment**: Verify against actual server, not mocks

## References

- **CLAUDE.md** - Verification-First Development Protocol section
- **verification/template.mjs** - Copy this to start new verification
- **verification/examples/** - See complete examples
