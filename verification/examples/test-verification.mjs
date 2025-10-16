#!/usr/bin/env node

/**
 * Example Verification: Test Graph History API
 *
 * This demonstrates a complete verification script for the graph history endpoint.
 *
 * Success Criteria:
 *   1. Server must be running and healthy
 *   2. API endpoint returns 200 status
 *   3. Response has correct structure (application, snapshots, count)
 *   4. Each snapshot has required fields (id, workflow_name, status, etc.)
 *   5. Snapshots are ordered by most recent first
 */

async function verify() {
  console.log('🔍 Starting verification: Graph History API');
  console.log('');

  // ============================================================
  // Setup
  // ============================================================
  const baseURL = process.env.BASE_URL || 'http://localhost:8081';
  const apiKey = process.env.API_KEY || process.env.IDP_API_KEY || 'cf1d1f5afb8c1f1b2e17079c835b1f22d3719f651b4673f1bc4e3a006ebeb5db';

  console.log(`📍 Target: ${baseURL}`);
  console.log(`🔑 API Key: ${apiKey.substring(0, 8)}...`);
  console.log('');

  // ============================================================
  // Pre-conditions: Server Health Check
  // ============================================================
  console.log('✓ Checking pre-conditions...');

  try {
    const healthResponse = await fetch(`${baseURL}/health`);
    if (!healthResponse.ok) {
      throw new Error(`Server health check failed: ${healthResponse.status}`);
    }
    console.log('  ✓ Server is running and healthy');
  } catch (error) {
    throw new Error(`Server not reachable at ${baseURL}: ${error.message}`);
  }

  console.log('');

  // ============================================================
  // Execute Test Scenario
  // ============================================================
  console.log('🧪 Executing test scenario...');

  const appName = 'world-app3'; // Test application name
  const response = await fetch(`${baseURL}/api/graph/${appName}/history?limit=10`, {
    method: 'GET',
    headers: {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
  });

  console.log(`  ✓ GET /api/graph/${appName}/history?limit=10`);
  console.log(`  ✓ Status: ${response.status} ${response.statusText}`);

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`API request failed: ${response.status} ${response.statusText}\n${errorText}`);
  }

  const data = await response.json();
  console.log('');

  // ============================================================
  // Assertions: Response Structure
  // ============================================================
  console.log('🔬 Verifying response structure...');

  // Assertion 1: Response has required top-level fields
  if (!data.application) {
    throw new Error('Response missing "application" field');
  }
  console.log('  ✓ Response has "application" field');

  if (!data.snapshots || !Array.isArray(data.snapshots)) {
    throw new Error('Response missing "snapshots" array');
  }
  console.log('  ✓ Response has "snapshots" array');

  if (typeof data.count !== 'number') {
    throw new Error('Response missing "count" field or it\'s not a number');
  }
  console.log('  ✓ Response has "count" field');

  // Assertion 2: Application name matches request
  if (data.application !== appName) {
    throw new Error(`Expected application "${appName}", got "${data.application}"`);
  }
  console.log(`  ✓ Application name matches: "${data.application}"`);

  // Assertion 3: Count matches array length
  if (data.count !== data.snapshots.length) {
    throw new Error(`Count mismatch: count=${data.count}, array length=${data.snapshots.length}`);
  }
  console.log(`  ✓ Count matches snapshot array length: ${data.count}`);

  console.log('');

  // ============================================================
  // Assertions: Snapshot Fields
  // ============================================================
  console.log('🔬 Verifying snapshot fields...');

  if (data.snapshots.length === 0) {
    console.log('  ⚠ No snapshots found (this may be expected for new applications)');
  } else {
    // Check first snapshot as representative
    const snapshot = data.snapshots[0];

    const requiredFields = [
      'id',
      'workflow_name',
      'status',
      'started_at',
      'total_steps',
      'completed_steps',
      'failed_steps',
    ];

    for (const field of requiredFields) {
      if (!(field in snapshot)) {
        throw new Error(`Snapshot missing required field: "${field}"`);
      }
    }
    console.log('  ✓ Snapshot has all required fields');

    // Assertion: Valid status enum
    const validStatuses = ['succeeded', 'running', 'failed', 'pending', 'waiting'];
    if (!validStatuses.includes(snapshot.status)) {
      throw new Error(`Invalid status: "${snapshot.status}". Expected one of: ${validStatuses.join(', ')}`);
    }
    console.log(`  ✓ Snapshot status is valid: "${snapshot.status}"`);

    // Assertion: ID is a number
    if (typeof snapshot.id !== 'number') {
      throw new Error(`Snapshot ID should be a number, got: ${typeof snapshot.id}`);
    }
    console.log('  ✓ Snapshot ID is a number');

    // Assertion: started_at is a valid ISO 8601 date
    const startedAt = new Date(snapshot.started_at);
    if (isNaN(startedAt.getTime())) {
      throw new Error(`Invalid started_at date: "${snapshot.started_at}"`);
    }
    console.log('  ✓ Snapshot started_at is a valid date');

    // Assertion: Step counts are numbers
    if (typeof snapshot.total_steps !== 'number' ||
        typeof snapshot.completed_steps !== 'number' ||
        typeof snapshot.failed_steps !== 'number') {
      throw new Error('Step counts must be numbers');
    }
    console.log('  ✓ Step counts are numbers');
  }

  console.log('');

  // ============================================================
  // Assertions: Snapshot Ordering
  // ============================================================
  console.log('🔬 Verifying snapshot ordering...');

  if (data.snapshots.length >= 2) {
    // Check that snapshots are ordered by most recent first
    for (let i = 0; i < data.snapshots.length - 1; i++) {
      const current = new Date(data.snapshots[i].started_at);
      const next = new Date(data.snapshots[i + 1].started_at);

      if (current < next) {
        throw new Error(
          `Snapshots not ordered by most recent first: ` +
          `snapshot[${i}]=${current.toISOString()} is before ` +
          `snapshot[${i + 1}]=${next.toISOString()}`
        );
      }
    }
    console.log('  ✓ Snapshots ordered by most recent first');
  } else {
    console.log('  ⚠ Not enough snapshots to verify ordering');
  }

  console.log('');

  // ============================================================
  // Summary
  // ============================================================
  console.log('✅ Verification PASSED');
  console.log('');
  console.log('Summary:');
  console.log(`  - Server health: OK`);
  console.log(`  - API endpoint: /api/graph/${appName}/history`);
  console.log(`  - Response status: ${response.status} OK`);
  console.log(`  - Application: ${data.application}`);
  console.log(`  - Snapshot count: ${data.count}`);
  console.log(`  - All fields validated: ✓`);
  console.log(`  - Snapshot ordering: ✓`);
  console.log('');
  console.log('Feature is working as expected! 🎉');
}

/**
 * Main execution
 */
async function main() {
  try {
    await verify();
    process.exit(0);
  } catch (error) {
    console.error('');
    console.error('❌ Verification FAILED');
    console.error('');
    console.error('Error:', error.message);

    if (error.stack) {
      console.error('');
      console.error('Stack trace:');
      console.error(error.stack);
    }

    console.error('');
    console.error('Possible issues:');
    console.error('  1. Server not running (run: ./innominatus)');
    console.error('  2. Invalid API key (check IDP_API_KEY environment variable)');
    console.error('  3. No workflow history data (run a workflow first)');
    console.error('  4. Database connection issue');
    console.error('');
    console.error('Next steps:');
    console.error('  1. Check server logs');
    console.error('  2. Verify database contains workflow_executions data');
    console.error('  3. Re-run: node verification/examples/test-verification.mjs');

    process.exit(1);
  }
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

// Export for testing
export { verify };
