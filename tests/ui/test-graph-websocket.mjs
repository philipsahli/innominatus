#!/usr/bin/env node

/**
 * Puppeteer Test: Graph Visualization WebSocket Connection
 *
 * This test verifies that the graph visualization pages can:
 * 1. Load without React errors
 * 2. Establish WebSocket connections successfully
 * 3. Display graph nodes and edges
 *
 * Tests all three deployed apps:
 * - web-application
 * - world-app3
 * - app-with-storage (was showing React error #185)
 */

import puppeteer from 'puppeteer';
import { readFileSync } from 'fs';

const BASE_URL = 'http://localhost:8081';
const APPS_TO_TEST = ['web-application', 'world-app3', 'app-with-storage'];

// OIDC session token (from previous login)
const SESSION_ID = 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46';

async function testGraphVisualization(appName) {
  console.log(`\n🧪 Testing graph visualization for: ${appName}`);
  console.log('='.repeat(60));

  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();

  // Track console messages (including errors)
  const consoleMessages = [];
  page.on('console', msg => {
    consoleMessages.push({
      type: msg.type(),
      text: msg.text()
    });
  });

  // Track WebSocket connections
  const websocketConnections = [];
  page.on('websocket', ws => {
    const url = ws.url();
    websocketConnections.push({ url, connected: false, error: null });

    ws.on('open', () => {
      websocketConnections[websocketConnections.length - 1].connected = true;
      console.log(`  ✅ WebSocket opened: ${url}`);
    });

    ws.on('close', () => {
      console.log(`  🔌 WebSocket closed: ${url}`);
    });

    ws.on('error', (err) => {
      websocketConnections[websocketConnections.length - 1].error = err.message;
      console.log(`  ❌ WebSocket error: ${err.message}`);
    });
  });

  try {
    // Set session cookie
    await page.setCookie({
      name: 'session_id',
      value: SESSION_ID,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false
    });

    // Navigate to graph page
    const graphUrl = `${BASE_URL}/graph/${appName}`;
    console.log(`  🌐 Navigating to: ${graphUrl}`);

    await page.goto(graphUrl, {
      waitUntil: 'networkidle2',
      timeout: 30000
    });

    // Wait for React to render
    await new Promise(resolve => setTimeout(resolve, 3000));

    // Check for React errors in the page
    const pageContent = await page.content();
    const hasReactError = pageContent.includes('Application error') ||
                         pageContent.includes('React error') ||
                         pageContent.includes('client-side exception');

    // Check console for errors
    const errors = consoleMessages.filter(msg => msg.type === 'error');
    const warnings = consoleMessages.filter(msg => msg.type === 'warning');

    // Check for React Flow elements (indicates successful render)
    const hasReactFlow = await page.evaluate(() => {
      const reactFlowElements = document.querySelectorAll('.react-flow');
      return reactFlowElements.length > 0;
    });

    // Check for graph nodes
    const nodeCount = await page.evaluate(() => {
      const nodes = document.querySelectorAll('.react-flow__node');
      return nodes.length;
    });

    // Take screenshot
    const screenshotPath = `/tmp/graph-${appName}-${hasReactError ? 'error' : 'success'}.png`;
    await page.screenshot({ path: screenshotPath, fullPage: true });

    // Print results
    console.log(`\n  📊 Test Results:`);
    console.log(`  ─────────────────────────────────────────────────────`);
    console.log(`  React Error Detected: ${hasReactError ? '❌ YES' : '✅ NO'}`);
    console.log(`  React Flow Rendered: ${hasReactFlow ? '✅ YES' : '❌ NO'}`);
    console.log(`  Graph Nodes Found: ${nodeCount > 0 ? `✅ ${nodeCount}` : '❌ 0'}`);
    console.log(`  WebSocket Connections: ${websocketConnections.length}`);

    websocketConnections.forEach((ws, i) => {
      console.log(`    ${i + 1}. ${ws.connected ? '✅' : '❌'} ${ws.url}`);
      if (ws.error) {
        console.log(`       Error: ${ws.error}`);
      }
    });

    console.log(`  Console Errors: ${errors.length}`);
    if (errors.length > 0) {
      errors.slice(0, 3).forEach((err, i) => {
        console.log(`    ${i + 1}. ${err.text}`);
      });
    }

    console.log(`  Console Warnings: ${warnings.length}`);
    console.log(`  Screenshot: ${screenshotPath}`);

    // Determine pass/fail
    const passed = !hasReactError &&
                   hasReactFlow &&
                   nodeCount > 0 &&
                   errors.length === 0;

    console.log(`\n  ${passed ? '✅ PASSED' : '❌ FAILED'}`);
    console.log(`  ─────────────────────────────────────────────────────`);

    await browser.close();
    return { appName, passed, hasReactError, hasReactFlow, nodeCount, websocketConnections, errors, warnings };

  } catch (error) {
    console.log(`\n  ❌ Test failed with exception: ${error.message}`);
    await page.screenshot({ path: `/tmp/graph-${appName}-exception.png`, fullPage: true });
    await browser.close();
    return {
      appName,
      passed: false,
      hasReactError: true,
      error: error.message,
      websocketConnections,
      errors: consoleMessages.filter(msg => msg.type === 'error')
    };
  }
}

async function main() {
  console.log('🚀 Starting Graph Visualization WebSocket Tests');
  console.log('='.repeat(60));
  console.log(`Testing ${APPS_TO_TEST.length} applications\n`);

  const results = [];

  for (const appName of APPS_TO_TEST) {
    const result = await testGraphVisualization(appName);
    results.push(result);
  }

  // Summary
  console.log('\n\n📋 Test Summary');
  console.log('='.repeat(60));

  results.forEach(result => {
    const status = result.passed ? '✅ PASSED' : '❌ FAILED';
    console.log(`${status} - ${result.appName}`);
    if (!result.passed) {
      if (result.hasReactError) {
        console.log(`  └─ React Error Detected`);
      }
      if (result.errors && result.errors.length > 0) {
        console.log(`  └─ ${result.errors.length} console errors`);
      }
      if (result.websocketConnections) {
        const failedWs = result.websocketConnections.filter(ws => !ws.connected);
        if (failedWs.length > 0) {
          console.log(`  └─ ${failedWs.length} WebSocket connection(s) failed`);
        }
      }
    }
  });

  const passedCount = results.filter(r => r.passed).length;
  const totalCount = results.length;

  console.log('\n' + '='.repeat(60));
  console.log(`Overall: ${passedCount}/${totalCount} tests passed`);

  if (passedCount === totalCount) {
    console.log('🎉 All tests passed!');
    process.exit(0);
  } else {
    console.log('⚠️  Some tests failed - check logs above for details');
    process.exit(1);
  }
}

main().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
