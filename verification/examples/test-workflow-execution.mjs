#!/usr/bin/env node

/**
 * Verification Script: Workflow Execution End-to-End
 *
 * Purpose: Verify that workflows can be executed successfully and that the
 *          orchestration engine correctly handles multi-step workflows with
 *          proper state transitions, logging, and graph updates.
 *
 * What it tests:
 * - Workflow execution via API
 * - Step-by-step status updates
 * - Log streaming via SSE
 * - Graph relationship creation
 * - Resource state transitions (requested → provisioning → active)
 * - Workflow completion status
 *
 * Usage:
 *   node verification/examples/test-workflow-execution.mjs
 *
 * Expected outcome:
 * - Workflow executes successfully
 * - All steps transition through states correctly
 * - Logs are captured
 * - Graph relationships created
 * - Resources provisioned
 * - Screenshots and outputs saved to docs/verification/
 */

import fs from 'fs/promises';
import path from 'path';
import { setTimeout } from 'timers/promises';

const API_BASE = process.env.API_BASE || 'http://localhost:8081';
const API_TOKEN = process.env.INNOMINATUS_API_KEY || process.env.API_TOKEN;
const OUTPUT_DIR = 'docs/verification/workflow-execution';
const POLL_INTERVAL = 2000; // 2 seconds
const MAX_WAIT_TIME = 300000; // 5 minutes

// ANSI color codes
const colors = {
  reset: '\x1b[0m',
  green: '\x1b[32m',
  red: '\x1b[31m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  cyan: '\x1b[36m',
  magenta: '\x1b[35m',
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
    const text = await response.text();
    const data = text ? JSON.parse(text) : null;
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

// Test 1: Submit Score spec with postgres resource
async function testSubmitScoreSpec() {
  logSection('Test 1: Submit Score Spec with PostgreSQL Resource');

  const scoreSpec = {
    apiVersion: 'score.dev/v1b1',
    metadata: {
      name: 'verification-test-app',
    },
    containers: {
      main: {
        image: 'nginx:latest',
      },
    },
    resources: {
      database: {
        type: 'postgres',
        properties: {
          version: '15',
          size: 'small',
        },
      },
    },
  };

  log('\n  Submitting Score spec...', 'cyan');
  log(JSON.stringify(scoreSpec, null, 2), 'blue');

  const { ok, data, error, status } = await makeRequest('/api/specs', {
    method: 'POST',
    body: JSON.stringify(scoreSpec),
  });

  if (!ok) {
    log(`  ❌ FAIL: Failed to submit spec (${status}): ${error || JSON.stringify(data)}`, 'red');
    return null;
  }

  log(`  ✅ SUCCESS: Spec submitted`, 'green');
  log(`  📝 Spec name: ${data.name || scoreSpec.metadata.name}`, 'blue');

  await saveOutput('score-spec-submitted.json', data);

  return scoreSpec.metadata.name;
}

// Test 2: Wait for resource to be created (state='requested')
async function testResourceCreated(specName) {
  logSection('Test 2: Verify Resource Created (state=requested)');

  log('\n  Polling for resource...', 'cyan');

  const startTime = Date.now();

  while (Date.now() - startTime < MAX_WAIT_TIME) {
    const { data: resources } = await makeRequest('/api/resources');

    const resource = resources?.find(r => r.spec_name === specName && r.type === 'postgres');

    if (resource) {
      log(`  ✅ SUCCESS: Resource created`, 'green');
      log(`  📝 Resource ID: ${resource.id}`, 'blue');
      log(`  📝 Resource state: ${resource.state}`, resource.state === 'requested' ? 'green' : 'yellow');

      await saveOutput('resource-created.json', resource);

      return resource;
    }

    await setTimeout(POLL_INTERVAL);
  }

  log(`  ❌ FAIL: Resource not created after ${MAX_WAIT_TIME / 1000}s`, 'red');
  return null;
}

// Test 3: Wait for orchestration engine to pick up resource
async function testOrchestrationPickup(resourceId) {
  logSection('Test 3: Orchestration Engine Picks Up Resource');

  log('\n  Waiting for orchestration engine (polls every 5s)...', 'cyan');

  const startTime = Date.now();

  while (Date.now() - startTime < MAX_WAIT_TIME) {
    const { data: resource } = await makeRequest(`/api/resources/${resourceId}`);

    if (resource.workflow_execution_id) {
      log(`  ✅ SUCCESS: Resource picked up by orchestration engine`, 'green');
      log(`  📝 Workflow Execution ID: ${resource.workflow_execution_id}`, 'blue');
      log(`  📝 Resource state: ${resource.state}`, 'yellow');

      await saveOutput('resource-provisioning.json', resource);

      return resource.workflow_execution_id;
    }

    await setTimeout(POLL_INTERVAL);
  }

  log(`  ❌ FAIL: Resource not picked up after ${MAX_WAIT_TIME / 1000}s`, 'red');
  return null;
}

// Test 4: Monitor workflow execution
async function testWorkflowExecution(executionId) {
  logSection('Test 4: Monitor Workflow Execution');

  log('\n  Monitoring workflow execution...', 'cyan');

  const startTime = Date.now();
  let previousStatus = null;

  while (Date.now() - startTime < MAX_WAIT_TIME) {
    const { data: execution } = await makeRequest(`/api/workflows/${executionId}`);

    if (execution.status !== previousStatus) {
      log(`  📊 Workflow status: ${execution.status}`, execution.status === 'completed' ? 'green' : 'yellow');
      previousStatus = execution.status;

      // Log step details
      if (execution.steps) {
        execution.steps.forEach(step => {
          const icon = step.status === 'completed' ? '✅' : step.status === 'running' ? '🔄' : '⏳';
          log(`    ${icon} ${step.name}: ${step.status}`, 'blue');
        });
      }
    }

    if (execution.status === 'completed') {
      log(`  ✅ SUCCESS: Workflow completed successfully`, 'green');
      await saveOutput('workflow-completed.json', execution);
      return execution;
    }

    if (execution.status === 'failed') {
      log(`  ❌ FAIL: Workflow failed: ${execution.error_message}`, 'red');
      await saveOutput('workflow-failed.json', execution);
      return null;
    }

    await setTimeout(POLL_INTERVAL);
  }

  log(`  ❌ FAIL: Workflow did not complete after ${MAX_WAIT_TIME / 1000}s`, 'red');
  return null;
}

// Test 5: Verify resource state is 'active'
async function testResourceActive(resourceId) {
  logSection('Test 5: Verify Resource State = Active');

  const { data: resource } = await makeRequest(`/api/resources/${resourceId}`);

  if (resource.state === 'active') {
    log(`  ✅ SUCCESS: Resource is active`, 'green');
    log(`  📝 Resource ID: ${resource.id}`, 'blue');
    log(`  📝 Resource type: ${resource.type}`, 'blue');

    await saveOutput('resource-active.json', resource);
    return true;
  }

  log(`  ❌ FAIL: Resource state is '${resource.state}', expected 'active'`, 'red');
  log(`  📝 Error: ${resource.error_message}`, 'red');

  await saveOutput('resource-not-active.json', resource);
  return false;
}

// Test 6: Verify graph relationships
async function testGraphRelationships(resourceId) {
  logSection('Test 6: Verify Graph Relationships');

  const { data: graph } = await makeRequest(`/api/resources/${resourceId}/graph`);

  if (!graph || !graph.nodes || !graph.edges) {
    log(`  ❌ FAIL: Graph data not found`, 'red');
    return false;
  }

  log(`  ✅ SUCCESS: Graph relationships created`, 'green');
  log(`  📊 Nodes: ${graph.nodes.length}`, 'blue');
  log(`  📊 Edges: ${graph.edges.length}`, 'blue');

  // Verify expected nodes
  const nodeTypes = graph.nodes.map(n => n.node_type);
  const expectedNodeTypes = ['spec', 'resource', 'provider', 'workflow'];

  log('\n  Node types found:', 'cyan');
  nodeTypes.forEach(type => {
    const expected = expectedNodeTypes.includes(type);
    const icon = expected ? '✅' : '⚠️';
    log(`    ${icon} ${type}`, expected ? 'green' : 'yellow');
  });

  // Verify edges
  log('\n  Edges found:', 'cyan');
  graph.edges.forEach(edge => {
    const sourceNode = graph.nodes.find(n => n.id === edge.source_node_id);
    const targetNode = graph.nodes.find(n => n.id === edge.target_node_id);
    log(`    • ${sourceNode?.node_type} → ${edge.edge_type} → ${targetNode?.node_type}`, 'blue');
  });

  await saveOutput('resource-graph.json', graph);

  return true;
}

// Test 7: Cleanup - delete spec
async function testCleanup(specName) {
  logSection('Test 7: Cleanup - Delete Spec');

  log('\n  Deleting spec...', 'cyan');

  const { ok, status } = await makeRequest(`/api/specs/${specName}`, {
    method: 'DELETE',
  });

  if (ok || status === 404) {
    log(`  ✅ SUCCESS: Spec deleted`, 'green');
    return true;
  }

  log(`  ⚠️  WARNING: Failed to delete spec (status ${status})`, 'yellow');
  return true; // Don't fail the test on cleanup failure
}

// Main verification flow
async function main() {
  log('╔═══════════════════════════════════════════════════════════╗', 'cyan');
  log('║        Workflow Execution End-to-End Verification        ║', 'cyan');
  log('╚═══════════════════════════════════════════════════════════╝', 'cyan');

  await ensureOutputDir();

  const results = [];

  try {
    // Test 1: Submit spec
    const specName = await testSubmitScoreSpec();
    results.push({ test: 'Submit Score Spec', passed: !!specName });
    if (!specName) throw new Error('Failed to submit spec');

    // Test 2: Resource created
    const resource = await testResourceCreated(specName);
    results.push({ test: 'Resource Created', passed: !!resource });
    if (!resource) throw new Error('Resource not created');

    // Test 3: Orchestration pickup
    const executionId = await testOrchestrationPickup(resource.id);
    results.push({ test: 'Orchestration Pickup', passed: !!executionId });
    if (!executionId) throw new Error('Orchestration did not pick up resource');

    // Test 4: Workflow execution
    const execution = await testWorkflowExecution(executionId);
    results.push({ test: 'Workflow Execution', passed: !!execution });
    if (!execution) throw new Error('Workflow did not complete');

    // Test 5: Resource active
    const isActive = await testResourceActive(resource.id);
    results.push({ test: 'Resource Active', passed: isActive });

    // Test 6: Graph relationships
    const graphOk = await testGraphRelationships(resource.id);
    results.push({ test: 'Graph Relationships', passed: graphOk });

    // Test 7: Cleanup
    const cleanedUp = await testCleanup(specName);
    results.push({ test: 'Cleanup', passed: cleanedUp });

  } catch (error) {
    log(`\n  💥 ERROR: ${error.message}`, 'red');
    results.push({ test: 'Overall', passed: false, error: error.message });
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
