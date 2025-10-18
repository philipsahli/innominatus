#!/usr/bin/env node

/**
 * Manual Interactive Test for Graph Visualization
 *
 * This test opens a real browser (non-headless) and waits for manual verification.
 * Use this to see exactly what the user sees and debug issues.
 */

import puppeteer from 'puppeteer';

const BASE_URL = 'http://localhost:8081';
const APP_NAME = 'app-with-storage';

// Get OIDC session from localStorage (you need to login first manually)
const SESSION_ID = 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46';

async function testGraphManually() {
  console.log('🧪 Opening browser for manual testing...');
  console.log(`App: ${APP_NAME}`);
  console.log('');

  const browser = await puppeteer.launch({
    headless: false, // Show the browser
    devtools: true,  // Open DevTools automatically
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--window-size=1920,1080'
    ]
  });

  const page = await browser.newPage();

  // Set viewport
  await page.setViewport({ width: 1920, height: 1080 });

  // Track all console messages
  page.on('console', msg => {
    const type = msg.type();
    const text = msg.text();

    if (type === 'error') {
      console.log(`🔴 BROWSER ERROR: ${text}`);
    } else if (type === 'warning') {
      console.log(`⚠️  BROWSER WARNING: ${text}`);
    } else if (text.includes('WebSocket') || text.includes('ws://')) {
      console.log(`🔌 WEBSOCKET: ${text}`);
    }
  });

  // Track page errors
  page.on('pageerror', error => {
    console.log(`💥 PAGE ERROR: ${error.message}`);
  });

  // Track request failures
  page.on('requestfailed', request => {
    console.log(`❌ REQUEST FAILED: ${request.url()}`);
    console.log(`   Failure: ${request.failure().errorText}`);
  });

  // Track WebSocket connections
  page.on('websocket', ws => {
    const url = ws.url();
    console.log(`🔌 WebSocket created: ${url}`);

    ws.on('open', () => {
      console.log(`  ✅ WebSocket opened: ${url}`);
    });

    ws.on('close', () => {
      console.log(`  🔌 WebSocket closed: ${url}`);
    });

    ws.on('framereceived', (frame) => {
      console.log(`  📩 WebSocket received: ${frame.payload.length} bytes`);
    });

    ws.on('framesent', (frame) => {
      console.log(`  📤 WebSocket sent: ${frame.payload.length} bytes`);
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

    // Navigate to login page first to set localStorage token
    console.log('Step 1: Loading login page to set auth token...');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle2' });

    // Inject auth token into localStorage (simulate successful login)
    await page.evaluate(() => {
      localStorage.setItem('auth-token', 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46');
    });

    console.log('✅ Auth token set in localStorage');
    console.log('');

    // Now navigate to graph page
    const graphUrl = `${BASE_URL}/graph/${APP_NAME}`;
    console.log(`Step 2: Navigating to graph page...`);
    console.log(`URL: ${graphUrl}`);
    console.log('');

    await page.goto(graphUrl, {
      waitUntil: 'networkidle2',
      timeout: 30000
    });

    // Wait a bit for React to render
    await new Promise(resolve => setTimeout(resolve, 3000));

    // Check what's on the page
    const pageInfo = await page.evaluate(() => {
      return {
        title: document.title,
        hasReactFlow: document.querySelectorAll('.react-flow').length > 0,
        hasReactFlowNodes: document.querySelectorAll('.react-flow__node').length,
        hasReactFlowEdges: document.querySelectorAll('.react-flow__edge').length,
        hasErrorMessage: document.body.innerHTML.includes('Application error') ||
                        document.body.innerHTML.includes('client-side exception'),
        bodyText: document.body.innerText.substring(0, 500)
      };
    });

    console.log('📊 Page Information:');
    console.log(`  Title: ${pageInfo.title}`);
    console.log(`  Has React Flow: ${pageInfo.hasReactFlow ? '✅' : '❌'}`);
    console.log(`  React Flow Nodes: ${pageInfo.hasReactFlowNodes}`);
    console.log(`  React Flow Edges: ${pageInfo.hasReactFlowEdges}`);
    console.log(`  Has Error: ${pageInfo.hasErrorMessage ? '❌ YES' : '✅ NO'}`);
    console.log('');

    if (pageInfo.hasErrorMessage) {
      console.log('🔴 ERROR DETECTED ON PAGE:');
      console.log(pageInfo.bodyText);
      console.log('');
    }

    // Take screenshot
    await page.screenshot({
      path: '/tmp/graph-manual-test.png',
      fullPage: true
    });
    console.log('📸 Screenshot saved: /tmp/graph-manual-test.png');
    console.log('');

    console.log('════════════════════════════════════════════════════════');
    console.log('🎯 Browser is open for manual inspection');
    console.log('   - Check the DevTools Console tab for errors');
    console.log('   - Check the Network tab for failed requests');
    console.log('   - Check if the graph is rendering');
    console.log('');
    console.log('Press Ctrl+C when done inspecting');
    console.log('════════════════════════════════════════════════════════');

    // Keep browser open for manual inspection
    await new Promise(resolve => {
      process.on('SIGINT', () => {
        console.log('\n\nClosing browser...');
        resolve();
      });
    });

  } catch (error) {
    console.error('❌ Test failed with error:', error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

testGraphManually().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
