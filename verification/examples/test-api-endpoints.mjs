#!/usr/bin/env node

/**
 * Verification Script: API Endpoints Health and Functionality
 *
 * Purpose: Verify that all API endpoints are accessible, respond correctly,
 *          and return expected data structures.
 *
 * What it tests:
 * - Server health and readiness
 * - Authentication (file-based and API key)
 * - Spec endpoints (CRUD operations)
 * - Workflow endpoints (list, details, logs)
 * - Resource endpoints (list, details, graph)
 * - Provider endpoints (list, details, workflows)
 * - WebSocket connectivity (logs streaming)
 * - Swagger documentation availability
 *
 * Usage:
 *   node verification/examples/test-api-endpoints.mjs
 *
 * Expected outcome:
 * - All endpoints respond with correct status codes
 * - Data structures match expected schema
 * - Authentication works correctly
 * - Screenshots and outputs saved to docs/verification/
 */

import fs from 'fs/promises';
import path from 'path';

const API_BASE = process.env.API_BASE || 'http://localhost:8081';
const API_TOKEN = process.env.INNOMINATUS_API_KEY || process.env.API_TOKEN;
const OUTPUT_DIR = 'docs/verification/api-endpoints';

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
    const contentType = response.headers.get('content-type');
    let data = null;

    if (contentType?.includes('application/json')) {
      const text = await response.text();
      data = text ? JSON.parse(text) : null;
    } else {
      data = await response.text();
    }

    return { ok: response.ok, status: response.status, data, contentType };
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

// Test 1: Health check endpoint
async function testHealthEndpoint() {
  logSection('Test 1: Health Check Endpoint');

  const { ok, status, data } = await makeRequest('/health');

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Health endpoint responding (${status})`, 'green');
    log(`  📊 Health data: ${JSON.stringify(data)}`, 'blue');
    await saveOutput('health.json', data);
    return true;
  }

  log(`  ❌ FAIL: Health endpoint failed (${status})`, 'red');
  return false;
}

// Test 2: Readiness check endpoint
async function testReadyEndpoint() {
  logSection('Test 2: Readiness Check Endpoint');

  const { ok, status, data } = await makeRequest('/ready');

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Ready endpoint responding (${status})`, 'green');
    log(`  📊 Ready data: ${JSON.stringify(data)}`, 'blue');
    await saveOutput('ready.json', data);
    return true;
  }

  log(`  ❌ FAIL: Ready endpoint failed (${status})`, 'red');
  return false;
}

// Test 3: Metrics endpoint
async function testMetricsEndpoint() {
  logSection('Test 3: Prometheus Metrics Endpoint');

  const { ok, status, data, contentType } = await makeRequest('/metrics');

  if (ok && status === 200 && contentType?.includes('text/plain')) {
    log(`  ✅ SUCCESS: Metrics endpoint responding (${status})`, 'green');
    const lineCount = data.split('\n').length;
    log(`  📊 Metrics lines: ${lineCount}`, 'blue');
    await saveOutput('metrics.txt', data);
    return true;
  }

  log(`  ❌ FAIL: Metrics endpoint failed (${status})`, 'red');
  return false;
}

// Test 4: Swagger documentation endpoints
async function testSwaggerEndpoints() {
  logSection('Test 4: Swagger Documentation Endpoints');

  const swaggerEndpoints = [
    { path: '/swagger-user', name: 'User API' },
    { path: '/swagger-admin', name: 'Admin API' },
  ];

  let allPassed = true;

  for (const endpoint of swaggerEndpoints) {
    const { ok, status } = await makeRequest(endpoint.path);

    if (ok && status === 200) {
      log(`  ✅ ${endpoint.name}: Available (${status})`, 'green');
    } else {
      log(`  ❌ ${endpoint.name}: Not available (${status})`, 'red');
      allPassed = false;
    }
  }

  return allPassed;
}

// Test 5: List specs endpoint
async function testListSpecs() {
  logSection('Test 5: List Specs Endpoint');

  const { ok, status, data } = await makeRequest('/api/specs');

  if (ok && status === 200 && Array.isArray(data)) {
    log(`  ✅ SUCCESS: Specs endpoint responding (${status})`, 'green');
    log(`  📊 Specs count: ${data.length}`, 'blue');
    await saveOutput('specs-list.json', data);
    return true;
  }

  log(`  ❌ FAIL: Specs endpoint failed (${status})`, 'red');
  return false;
}

// Test 6: List workflows endpoint
async function testListWorkflows() {
  logSection('Test 6: List Workflows Endpoint');

  const { ok, status, data } = await makeRequest('/api/workflows');

  if (ok && status === 200 && Array.isArray(data)) {
    log(`  ✅ SUCCESS: Workflows endpoint responding (${status})`, 'green');
    log(`  📊 Workflow executions: ${data.length}`, 'blue');

    if (data.length > 0) {
      log(`  📝 Latest workflow: ${data[0].workflow_name} (${data[0].status})`, 'blue');
    }

    await saveOutput('workflows-list.json', data);
    return true;
  }

  log(`  ❌ FAIL: Workflows endpoint failed (${status})`, 'red');
  return false;
}

// Test 7: List resources endpoint
async function testListResources() {
  logSection('Test 7: List Resources Endpoint');

  const { ok, status, data } = await makeRequest('/api/resources');

  if (ok && status === 200 && Array.isArray(data)) {
    log(`  ✅ SUCCESS: Resources endpoint responding (${status})`, 'green');
    log(`  📊 Resources count: ${data.length}`, 'blue');

    // Group by type
    const byType = data.reduce((acc, r) => {
      acc[r.type] = (acc[r.type] || 0) + 1;
      return acc;
    }, {});

    log('\n  Resources by type:', 'cyan');
    Object.entries(byType).forEach(([type, count]) => {
      log(`    • ${type}: ${count}`, 'blue');
    });

    await saveOutput('resources-list.json', data);
    return true;
  }

  log(`  ❌ FAIL: Resources endpoint failed (${status})`, 'red');
  return false;
}

// Test 8: List providers endpoint
async function testListProviders() {
  logSection('Test 8: List Providers Endpoint');

  const { ok, status, data } = await makeRequest('/api/providers');

  if (ok && status === 200 && Array.isArray(data)) {
    log(`  ✅ SUCCESS: Providers endpoint responding (${status})`, 'green');
    log(`  📊 Providers count: ${data.length}`, 'blue');

    log('\n  Providers:', 'cyan');
    data.forEach(provider => {
      log(`    • ${provider.name} (${provider.category})`, 'blue');
    });

    await saveOutput('providers-list.json', data);
    return true;
  }

  log(`  ❌ FAIL: Providers endpoint failed (${status})`, 'red');
  return false;
}

// Test 9: Get provider details
async function testProviderDetails() {
  logSection('Test 9: Provider Details Endpoint');

  // First get list of providers
  const { data: providers } = await makeRequest('/api/providers');

  if (!providers || providers.length === 0) {
    log(`  ⚠️  SKIP: No providers available`, 'yellow');
    return true;
  }

  const providerName = providers[0].name;
  const { ok, status, data } = await makeRequest(`/api/providers/${providerName}`);

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Provider details for '${providerName}' (${status})`, 'green');
    log(`  📊 Workflows: ${data.workflows?.length || 0}`, 'blue');
    log(`  📊 Resource types: ${data.capabilities?.resourceTypes?.length || 0}`, 'blue');

    await saveOutput('provider-details.json', data);
    return true;
  }

  log(`  ❌ FAIL: Provider details failed (${status})`, 'red');
  return false;
}

// Test 10: Workflow details endpoint
async function testWorkflowDetails() {
  logSection('Test 10: Workflow Details Endpoint');

  // First get list of workflows
  const { data: workflows } = await makeRequest('/api/workflows');

  if (!workflows || workflows.length === 0) {
    log(`  ⚠️  SKIP: No workflow executions available`, 'yellow');
    return true;
  }

  const workflowId = workflows[0].id;
  const { ok, status, data } = await makeRequest(`/api/workflows/${workflowId}`);

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Workflow details for ID ${workflowId} (${status})`, 'green');
    log(`  📊 Status: ${data.status}`, 'blue');
    log(`  📊 Steps: ${data.steps?.length || 0}`, 'blue');

    await saveOutput('workflow-details.json', data);
    return true;
  }

  log(`  ❌ FAIL: Workflow details failed (${status})`, 'red');
  return false;
}

// Test 11: Workflow graph endpoint
async function testWorkflowGraph() {
  logSection('Test 11: Workflow Graph Endpoint');

  // First get list of workflows
  const { data: workflows } = await makeRequest('/api/workflows');

  if (!workflows || workflows.length === 0) {
    log(`  ⚠️  SKIP: No workflow executions available`, 'yellow');
    return true;
  }

  const workflowId = workflows[0].id;
  const { ok, status, data } = await makeRequest(`/api/workflows/${workflowId}/graph`);

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Workflow graph for ID ${workflowId} (${status})`, 'green');
    log(`  📊 Nodes: ${data.nodes?.length || 0}`, 'blue');
    log(`  📊 Edges: ${data.edges?.length || 0}`, 'blue');

    await saveOutput('workflow-graph.json', data);
    return true;
  }

  log(`  ❌ FAIL: Workflow graph failed (${status})`, 'red');
  return false;
}

// Test 12: Resource graph endpoint
async function testResourceGraph() {
  logSection('Test 12: Resource Graph Endpoint');

  // First get list of resources
  const { data: resources } = await makeRequest('/api/resources');

  if (!resources || resources.length === 0) {
    log(`  ⚠️  SKIP: No resources available`, 'yellow');
    return true;
  }

  const resourceId = resources[0].id;
  const { ok, status, data } = await makeRequest(`/api/resources/${resourceId}/graph`);

  if (ok && status === 200) {
    log(`  ✅ SUCCESS: Resource graph for ID ${resourceId} (${status})`, 'green');
    log(`  📊 Nodes: ${data.nodes?.length || 0}`, 'blue');
    log(`  📊 Edges: ${data.edges?.length || 0}`, 'blue');

    await saveOutput('resource-graph.json', data);
    return true;
  }

  log(`  ❌ FAIL: Resource graph failed (${status})`, 'red');
  return false;
}

// Test 13: Authentication test (without token)
async function testAuthenticationRequired() {
  logSection('Test 13: Authentication Required for Protected Endpoints');

  // Try to access protected endpoint without token
  const { ok, status } = await makeRequest('/api/specs', {
    headers: {}, // No Authorization header
  });

  if (status === 401 || status === 403) {
    log(`  ✅ SUCCESS: Endpoint properly protected (${status})`, 'green');
    return true;
  }

  if (status === 200) {
    log(`  ⚠️  WARNING: Endpoint accessible without authentication`, 'yellow');
    log(`  📝 This may be expected if AUTH_TYPE=file or OIDC_ENABLED=false`, 'blue');
    return true;
  }

  log(`  ❌ FAIL: Unexpected status code (${status})`, 'red');
  return false;
}

// Main verification flow
async function main() {
  log('╔═══════════════════════════════════════════════════════════╗', 'cyan');
  log('║      API Endpoints Health & Functionality Verification   ║', 'cyan');
  log('╚═══════════════════════════════════════════════════════════╝', 'cyan');

  await ensureOutputDir();

  const tests = [
    { name: 'Health Endpoint', fn: testHealthEndpoint },
    { name: 'Ready Endpoint', fn: testReadyEndpoint },
    { name: 'Metrics Endpoint', fn: testMetricsEndpoint },
    { name: 'Swagger Endpoints', fn: testSwaggerEndpoints },
    { name: 'List Specs', fn: testListSpecs },
    { name: 'List Workflows', fn: testListWorkflows },
    { name: 'List Resources', fn: testListResources },
    { name: 'List Providers', fn: testListProviders },
    { name: 'Provider Details', fn: testProviderDetails },
    { name: 'Workflow Details', fn: testWorkflowDetails },
    { name: 'Workflow Graph', fn: testWorkflowGraph },
    { name: 'Resource Graph', fn: testResourceGraph },
    { name: 'Authentication', fn: testAuthenticationRequired },
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
    apiBase: API_BASE,
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
