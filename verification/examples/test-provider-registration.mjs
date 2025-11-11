#!/usr/bin/env node

/**
 * Verification Script: Provider Registration and Capability Resolution
 *
 * Purpose: Verify that providers are correctly loaded, capabilities are registered,
 *          and the resolver can match resource types to providers.
 *
 * What it tests:
 * - Provider loading from filesystem and Git sources
 * - Capability registration without conflicts
 * - Provider resolution algorithm
 * - Workflow retrieval for resource types
 *
 * Usage:
 *   node verification/examples/test-provider-registration.mjs
 *
 * Expected outcome:
 * - All providers loaded successfully
 * - No capability conflicts detected
 * - Resource type → Provider mapping correct
 * - Screenshots and outputs saved to docs/verification/
 */

import fs from 'fs/promises';
import path from 'path';

const API_BASE = process.env.API_BASE || 'http://localhost:8081';
const API_TOKEN = process.env.INNOMINATUS_API_KEY || process.env.API_TOKEN;
const OUTPUT_DIR = 'docs/verification/provider-registration';

// ANSI color codes
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
};

function log(message, color = 'reset') {
  console.log(`${colors[color]}${message}${colors.reset}`);
}

function logSection(title) {
  log('\n' + '='.repeat(60), 'cyan');
  log(title, 'cyan');
  log('='.repeat(60), 'cyan');
}

async function makeRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;
  const headers = {
    'Content-Type': 'application/json',
    ...(API_TOKEN && { 'Authorization': `Bearer ${API_TOKEN}` }),
    ...options.headers,
  };

  try {
    const response = await fetch(url, { ...options, headers });
    const data = await response.json();
    return { ok: response.ok, status: response.status, data };
  } catch (error) {
    return { ok: false, error: error.message };
  }
}

async function ensureOutputDir() {
  await fs.mkdir(OUTPUT_DIR, { recursive: true });
}

async function saveOutput(filename, content) {
  const filepath = path.join(OUTPUT_DIR, filename);
  await fs.writeFile(filepath, typeof content === 'string' ? content : JSON.stringify(content, null, 2));
  log(`  💾 Saved: ${filepath}`, 'blue');
}

// Test 1: List all providers
async function testListProviders() {
  logSection('Test 1: List All Providers');

  const { ok, data, error } = await makeRequest('/api/providers');

  if (!ok) {
    log(`  ❌ FAIL: Failed to list providers: ${error}`, 'red');
    return false;
  }

  log(`  ✅ SUCCESS: Found ${data.length} providers`, 'green');

  // Save provider list
  await saveOutput('providers-list.json', data);

  // Print summary
  log('\n  Providers:', 'cyan');
  data.forEach(provider => {
    log(`    • ${provider.name} (${provider.category}) - ${provider.workflows?.length || 0} workflows`, 'blue');
  });

  return true;
}

// Test 2: Verify provider capabilities
async function testProviderCapabilities() {
  logSection('Test 2: Verify Provider Capabilities');

  // Get all providers
  const { ok, data: providers } = await makeRequest('/api/providers');

  if (!ok) {
    log(`  ❌ FAIL: Cannot fetch providers`, 'red');
    return false;
  }

  const capabilityMap = {};
  let conflicts = [];

  log('\n  Analyzing capabilities...', 'cyan');

  for (const provider of providers) {
    const { data: details } = await makeRequest(`/api/providers/${provider.name}`);

    if (details.capabilities?.resourceTypes) {
      log(`\n  ${provider.name}:`, 'yellow');

      details.capabilities.resourceTypes.forEach(resType => {
        log(`    → ${resType}`, 'blue');

        if (!capabilityMap[resType]) {
          capabilityMap[resType] = [];
        }
        capabilityMap[resType].push(provider.name);

        // Check for conflicts
        if (capabilityMap[resType].length > 1) {
          conflicts.push({ resourceType: resType, providers: capabilityMap[resType] });
        }
      });
    }
  }

  await saveOutput('capability-map.json', capabilityMap);

  // Check for conflicts
  if (conflicts.length > 0) {
    log(`\n  ⚠️  WARNING: ${conflicts.length} capability conflicts detected!`, 'yellow');
    conflicts.forEach(conflict => {
      log(`    • ${conflict.resourceType}: ${conflict.providers.join(', ')}`, 'red');
    });
    await saveOutput('capability-conflicts.json', conflicts);
    return false;
  }

  log(`\n  ✅ SUCCESS: No capability conflicts detected`, 'green');
  log(`  📊 Total resource types covered: ${Object.keys(capabilityMap).length}`, 'blue');

  return true;
}

// Test 3: Test provider resolution for common resource types
async function testProviderResolution() {
  logSection('Test 3: Provider Resolution for Common Resource Types');

  const testCases = [
    { resourceType: 'postgres', expectedProvider: 'database-team' },
    { resourceType: 'postgresql', expectedProvider: 'database-team' },
    { resourceType: 's3', expectedProvider: 'storage-team' },
    { resourceType: 's3-bucket', expectedProvider: 'storage-team' },
    { resourceType: 'namespace', expectedProvider: 'container-team' },
    { resourceType: 'gitea-repo', expectedProvider: 'container-team' },
    { resourceType: 'argocd-app', expectedProvider: 'container-team' },
    { resourceType: 'vault-space', expectedProvider: 'vault-team' },
    { resourceType: 'keycloak-group', expectedProvider: 'identity-team' },
    { resourceType: 'gitea-org', expectedProvider: 'identity-team' },
  ];

  const results = [];
  let passed = 0;
  let failed = 0;

  log('\n  Testing resolution...', 'cyan');

  for (const testCase of testCases) {
    // Get capability map
    const { data: providers } = await makeRequest('/api/providers');

    let resolvedProvider = null;

    for (const provider of providers) {
      const { data: details } = await makeRequest(`/api/providers/${provider.name}`);

      if (details.capabilities?.resourceTypes?.includes(testCase.resourceType)) {
        resolvedProvider = provider.name;
        break;
      }
    }

    const success = resolvedProvider === testCase.expectedProvider;

    if (success) {
      log(`  ✅ ${testCase.resourceType} → ${resolvedProvider}`, 'green');
      passed++;
    } else {
      log(`  ❌ ${testCase.resourceType} → Expected: ${testCase.expectedProvider}, Got: ${resolvedProvider || 'none'}`, 'red');
      failed++;
    }

    results.push({
      resourceType: testCase.resourceType,
      expectedProvider: testCase.expectedProvider,
      resolvedProvider,
      success,
    });
  }

  await saveOutput('resolution-results.json', results);

  log(`\n  📊 Results: ${passed} passed, ${failed} failed`, passed === testCases.length ? 'green' : 'red');

  return failed === 0;
}

// Test 4: Verify provisioner workflows exist
async function testProvisionerWorkflows() {
  logSection('Test 4: Verify Provisioner Workflows');

  const { data: providers } = await makeRequest('/api/providers');
  const results = [];
  let passed = 0;
  let failed = 0;

  log('\n  Checking provisioner workflows...', 'cyan');

  for (const provider of providers) {
    const { data: details } = await makeRequest(`/api/providers/${provider.name}`);

    if (details.capabilities?.resourceTypes) {
      const provisionerWorkflows = details.workflows?.filter(w => w.category === 'provisioner') || [];

      if (provisionerWorkflows.length === 0) {
        log(`  ⚠️  ${provider.name}: No provisioner workflows (but has ${details.capabilities.resourceTypes.length} capabilities)`, 'yellow');
        failed++;
        results.push({ provider: provider.name, hasProvisioner: false, capabilities: details.capabilities.resourceTypes });
      } else {
        log(`  ✅ ${provider.name}: ${provisionerWorkflows.length} provisioner workflow(s)`, 'green');
        passed++;
        results.push({ provider: provider.name, hasProvisioner: true, workflows: provisionerWorkflows.map(w => w.name) });
      }
    }
  }

  await saveOutput('provisioner-workflows.json', results);

  log(`\n  📊 Results: ${passed} providers with provisioners, ${failed} without`, failed === 0 ? 'green' : 'yellow');

  return failed === 0;
}

// Main verification flow
async function main() {
  log('╔═══════════════════════════════════════════════════════════╗', 'cyan');
  log('║   Provider Registration & Capability Verification        ║', 'cyan');
  log('╚═══════════════════════════════════════════════════════════╝', 'cyan');

  await ensureOutputDir();

  const tests = [
    { name: 'List Providers', fn: testListProviders },
    { name: 'Provider Capabilities', fn: testProviderCapabilities },
    { name: 'Provider Resolution', fn: testProviderResolution },
    { name: 'Provisioner Workflows', fn: testProvisionerWorkflows },
  ];

  const results = [];

  for (const test of tests) {
    try {
      const passed = await test.fn();
      results.push({ test: test.name, passed });
    } catch (error) {
      log(`\n  💥 ERROR in ${test.name}: ${error.message}`, 'red');
      results.push({ test: test.name, passed: false, error: error.message });
    }
  }

  // Summary
  logSection('Verification Summary');

  const passedCount = results.filter(r => r.passed).length;
  const totalCount = results.length;

  results.forEach(result => {
    const icon = result.passed ? '✅' : '❌';
    const color = result.passed ? 'green' : 'red';
    log(`  ${icon} ${result.test}`, color);
  });

  log(`\n  Overall: ${passedCount}/${totalCount} tests passed`, passedCount === totalCount ? 'green' : 'red');
  log(`  Output directory: ${OUTPUT_DIR}`, 'blue');

  // Save summary
  const summary = {
    timestamp: new Date().toISOString(),
    results,
    passed: passedCount,
    total: totalCount,
    success: passedCount === totalCount,
  };

  await saveOutput('verification-summary.json', summary);

  // Exit code
  process.exit(passedCount === totalCount ? 0 : 1);
}

main().catch(error => {
  log(`\n💥 FATAL ERROR: ${error.message}`, 'red');
  console.error(error);
  process.exit(1);
});
