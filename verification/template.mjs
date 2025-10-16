#!/usr/bin/env node

/**
 * Verification Script Template
 *
 * Purpose: Define success criteria before implementation
 *
 * Usage:
 *   1. Copy this template: cp verification/template.mjs verification/my-feature.mjs
 *   2. Define success criteria in verify() function
 *   3. Run: node verification/my-feature.mjs
 *   4. Implement feature
 *   5. Re-run verification until it passes
 *
 * Verification-First Development:
 *   - Write this script BEFORE implementing the feature
 *   - Define what "success" looks like
 *   - Iterate on implementation until verification passes
 */

/**
 * Verification function - Define success criteria here
 */
async function verify() {
  console.log('🔍 Starting verification: [FEATURE NAME]');
  console.log('');

  // ============================================================
  // STEP 1: Setup
  // ============================================================
  const baseURL = process.env.BASE_URL || 'http://localhost:8081';
  const apiKey = process.env.API_KEY || process.env.IDP_API_KEY;

  if (!apiKey) {
    throw new Error('API_KEY or IDP_API_KEY environment variable required');
  }

  console.log(`📍 Target: ${baseURL}`);
  console.log('');

  // ============================================================
  // STEP 2: Pre-conditions
  // ============================================================
  console.log('✓ Checking pre-conditions...');

  // Example: Check server is running
  try {
    const healthResponse = await fetch(`${baseURL}/health`);
    if (!healthResponse.ok) {
      throw new Error(`Server health check failed: ${healthResponse.status}`);
    }
    console.log('  ✓ Server is running');
  } catch (error) {
    throw new Error(`Server not reachable: ${error.message}`);
  }

  console.log('');

  // ============================================================
  // STEP 3: Execute Test Scenario
  // ============================================================
  console.log('🧪 Executing test scenario...');

  // TODO: Replace with your actual verification logic
  // Example test scenarios:

  // API Endpoint Test
  // const response = await fetch(`${baseURL}/api/endpoint`, {
  //   method: 'GET',
  //   headers: {
  //     'Authorization': `Bearer ${apiKey}`,
  //     'Content-Type': 'application/json',
  //   },
  // });
  //
  // if (!response.ok) {
  //   throw new Error(`API request failed: ${response.status} ${response.statusText}`);
  // }
  //
  // const data = await response.json();
  // console.log(`  ✓ API returned status ${response.status}`);

  // Database State Test
  // TODO: Query database and verify expected state

  // File System Test
  // TODO: Check if expected files exist

  // Process Test
  // TODO: Verify expected processes are running

  console.log('');

  // ============================================================
  // STEP 4: Assertions
  // ============================================================
  console.log('🔬 Verifying assertions...');

  // TODO: Add your assertions here
  // Example assertions:

  // Assert response structure
  // if (!data || !data.property) {
  //   throw new Error('Response missing required property');
  // }
  // console.log('  ✓ Response has correct structure');

  // Assert values
  // if (data.count !== expectedCount) {
  //   throw new Error(`Expected count ${expectedCount}, got ${data.count}`);
  // }
  // console.log('  ✓ Values match expected');

  // Assert behavior
  // if (data.status !== 'success') {
  //   throw new Error(`Expected status 'success', got '${data.status}'`);
  // }
  // console.log('  ✓ Behavior is correct');

  console.log('');

  // ============================================================
  // STEP 5: Post-conditions
  // ============================================================
  console.log('🧹 Checking post-conditions...');

  // Example: Verify no side effects
  // TODO: Check that system state is as expected

  console.log('  ✓ No unexpected side effects');
  console.log('');

  // ============================================================
  // STEP 6: Cleanup
  // ============================================================
  console.log('🧼 Cleaning up test data...');

  // TODO: Clean up any test data created during verification

  console.log('  ✓ Cleanup complete');
  console.log('');

  // ============================================================
  // Success!
  // ============================================================
  console.log('✅ Verification PASSED');
  console.log('');
  console.log('Summary:');
  console.log('  - All pre-conditions met');
  console.log('  - Test scenario executed successfully');
  console.log('  - All assertions passed');
  console.log('  - Post-conditions verified');
  console.log('  - Cleanup completed');
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
    console.error('Next steps:');
    console.error('  1. Review the error message above');
    console.error('  2. Fix the implementation');
    console.error('  3. Re-run this verification script');

    process.exit(1);
  }
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main();
}

// Export for testing
export { verify };
