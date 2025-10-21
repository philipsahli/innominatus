#!/usr/bin/env node

/**
 * Comprehensive Authentication & Providers Page Test
 *
 * Tests both authentication methods:
 * 1. Local file-based auth (admin/admin123)
 * 2. OIDC Keycloak auth (demo-user)
 *
 * Verifies:
 * - Login works correctly
 * - Providers page loads without errors
 * - No JSON parse errors (double /api prefix bug)
 * - Sessions persist across page refreshes
 * - No random logouts during activity
 */

import puppeteer from 'puppeteer';

const BASE_URL = 'http://localhost:8081';
const KEYCLOAK_URL = 'http://keycloak.localtest.me';

// Local admin credentials
const LOCAL_USERNAME = 'admin';
const LOCAL_PASSWORD = 'admin123';

// OIDC demo user credentials
const OIDC_USERNAME = 'demo-user';
const OIDC_PASSWORD = 'demo123';

// Test configuration
const TEST_CONFIG = {
  headless: false,  // Show browser for debugging
  slowMo: 100,     // Slow down operations for visibility
  devtools: true,  // Open DevTools
};

/**
 * Setup browser with logging
 */
async function setupBrowser() {
  const browser = await puppeteer.launch({
    headless: TEST_CONFIG.headless,
    devtools: TEST_CONFIG.devtools,
    slowMo: TEST_CONFIG.slowMo,
    args: [
      '--no-sandbox',
      '--disable-setuid-sandbox',
      '--window-size=1920,1080'
    ]
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1920, height: 1080 });

  // Track console messages
  page.on('console', msg => {
    const type = msg.type();
    const text = msg.text();

    if (type === 'error') {
      console.log(`  🔴 BROWSER ERROR: ${text}`);
    } else if (type === 'warning' && !text.includes('DevTools')) {
      console.log(`  ⚠️  BROWSER WARNING: ${text}`);
    }
  });

  // Track page errors
  page.on('pageerror', error => {
    console.log(`  💥 PAGE ERROR: ${error.message}`);
  });

  // Track request failures
  page.on('requestfailed', request => {
    console.log(`  ❌ REQUEST FAILED: ${request.url()}`);
    console.log(`     Failure: ${request.failure().errorText}`);
  });

  return { browser, page };
}

/**
 * Test local authentication (admin/admin123)
 */
async function testLocalAuth() {
  console.log('\n╔═══════════════════════════════════════════════════════════╗');
  console.log('║  TEST 1: Local File-Based Authentication                 ║');
  console.log('╚═══════════════════════════════════════════════════════════╝\n');

  const { browser, page } = await setupBrowser();

  try {
    // Step 1: Navigate to login page
    console.log('Step 1: Navigating to login page...');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle2' });
    console.log('  ✅ Login page loaded');

    // Step 2: Fill in credentials
    console.log('\nStep 2: Entering credentials...');
    await page.type('input[name="username"]', LOCAL_USERNAME);
    await page.type('input[name="password"]', LOCAL_PASSWORD);
    console.log(`  ✅ Username: ${LOCAL_USERNAME}`);
    console.log(`  ✅ Password: ${'*'.repeat(LOCAL_PASSWORD.length)}`);

    // Step 3: Submit login form
    console.log('\nStep 3: Submitting login form...');
    await Promise.all([
      page.click('button[type="submit"]'),
      page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 10000 })
    ]);

    // Check if we're on dashboard (successful login redirects there)
    const currentUrl = page.url();
    console.log(`  📍 Current URL: ${currentUrl}`);

    if (currentUrl.includes('/login')) {
      console.log('  ❌ FAILED: Still on login page - credentials rejected');

      // Check for error message
      const errorMsg = await page.evaluate(() => {
        const errorDiv = document.querySelector('.error-message, [role="alert"], .text-red-500');
        return errorDiv ? errorDiv.textContent.trim() : 'No error message found';
      });
      console.log(`  📝 Error message: ${errorMsg}`);

      return { success: false, error: 'Login failed - invalid credentials' };
    }

    console.log('  ✅ Login successful - redirected to dashboard');

    // Step 4: Verify auth state
    console.log('\nStep 4: Verifying authentication state...');
    const authState = await page.evaluate(() => {
      return {
        hasToken: !!localStorage.getItem('auth-token'),
        hasCookie: document.cookie.includes('session_id'),
        token: localStorage.getItem('auth-token')?.substring(0, 20) + '...'
      };
    });

    console.log(`  ✅ Auth token in localStorage: ${authState.hasToken}`);
    console.log(`  ✅ Session cookie: ${authState.hasCookie}`);
    console.log(`  📝 Token: ${authState.token}`);

    // Step 5: Navigate to providers page
    console.log('\nStep 5: Navigating to providers page...');
    await page.goto(`${BASE_URL}/providers`, { waitUntil: 'networkidle2' });

    // Wait for page to settle
    await new Promise(resolve => setTimeout(resolve, 2000));

    const providersPageState = await page.evaluate(() => {
      const hasError = document.body.textContent.includes('Unexpected token') ||
                      document.body.textContent.includes('is not valid JSON') ||
                      document.body.textContent.includes('Failed to fetch');

      const hasProviderTable = !!document.querySelector('table');
      const hasLoadingSpinner = document.body.textContent.includes('Loading');

      // Check for the red error message
      const errorElement = document.querySelector('.text-red-500, [role="alert"]');
      const errorText = errorElement ? errorElement.textContent.trim() : null;

      return {
        url: window.location.href,
        hasError,
        errorText,
        hasProviderTable,
        hasLoadingSpinner,
        bodyPreview: document.body.textContent.substring(0, 500)
      };
    });

    console.log(`  📍 URL: ${providersPageState.url}`);
    console.log(`  📊 Provider table present: ${providersPageState.hasProviderTable ? '✅' : '❌'}`);
    console.log(`  ⏳ Loading spinner: ${providersPageState.hasLoadingSpinner ? 'Yes' : 'No'}`);

    if (providersPageState.hasError || providersPageState.errorText) {
      console.log(`  ❌ ERROR DETECTED: ${providersPageState.errorText || 'JSON parse error'}`);
      console.log(`  📝 Body preview: ${providersPageState.bodyPreview}`);

      // Take screenshot for debugging
      await page.screenshot({ path: '/tmp/providers-error-local.png', fullPage: true });
      console.log('  📸 Screenshot saved: /tmp/providers-error-local.png');

      return { success: false, error: providersPageState.errorText || 'Unknown error on providers page' };
    }

    console.log('  ✅ Providers page loaded successfully - no errors!');

    // Step 6: Test session persistence - refresh page
    console.log('\nStep 6: Testing session persistence (page refresh)...');
    await page.reload({ waitUntil: 'networkidle2' });
    await new Promise(resolve => setTimeout(resolve, 2000));

    const afterRefreshUrl = page.url();
    console.log(`  📍 URL after refresh: ${afterRefreshUrl}`);

    if (afterRefreshUrl.includes('/login')) {
      console.log('  ❌ FAILED: Logged out after refresh - session not persistent');
      return { success: false, error: 'Session not persistent - logged out after refresh' };
    }

    console.log('  ✅ Session persisted - still logged in after refresh');

    // Step 7: Navigate around and back to providers
    console.log('\nStep 7: Testing navigation (dashboard → providers)...');
    await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle2' });
    console.log('  ✅ Navigated to dashboard');

    await page.goto(`${BASE_URL}/providers`, { waitUntil: 'networkidle2' });
    await new Promise(resolve => setTimeout(resolve, 2000));

    const finalUrl = page.url();
    if (finalUrl.includes('/login')) {
      console.log('  ❌ FAILED: Logged out during navigation');
      return { success: false, error: 'Logged out during navigation' };
    }

    console.log('  ✅ Navigation successful - session stable');

    // Take final screenshot
    await page.screenshot({ path: '/tmp/providers-success-local.png', fullPage: true });
    console.log('\n📸 Final screenshot: /tmp/providers-success-local.png');

    console.log('\n╔═══════════════════════════════════════════════════════════╗');
    console.log('║  ✅ LOCAL AUTH TEST PASSED                                ║');
    console.log('╚═══════════════════════════════════════════════════════════╝\n');

    return { success: true };

  } catch (error) {
    console.log(`\n❌ TEST FAILED: ${error.message}`);
    console.error(error.stack);

    // Take error screenshot
    await page.screenshot({ path: '/tmp/providers-error-local.png', fullPage: true });
    console.log('📸 Error screenshot: /tmp/providers-error-local.png');

    return { success: false, error: error.message };

  } finally {
    if (!TEST_CONFIG.headless) {
      console.log('\n🔍 Browser left open for inspection. Press Ctrl+C to close.');
      await new Promise(resolve => {
        process.on('SIGINT', () => {
          console.log('\n\nClosing browser...');
          resolve();
        });
      });
    }
    await browser.close();
  }
}

/**
 * Test OIDC authentication (Keycloak demo-user)
 */
async function testOIDCAuth() {
  console.log('\n╔═══════════════════════════════════════════════════════════╗');
  console.log('║  TEST 2: OIDC Keycloak Authentication                    ║');
  console.log('╚═══════════════════════════════════════════════════════════╝\n');

  const { browser, page } = await setupBrowser();

  try {
    // Step 1: Navigate to main page (should redirect to Keycloak if OIDC enabled)
    console.log('Step 1: Navigating to main page...');
    await page.goto(`${BASE_URL}/`, { waitUntil: 'networkidle2' });

    const initialUrl = page.url();
    console.log(`  📍 Current URL: ${initialUrl}`);

    // Check if OIDC is enabled
    if (!initialUrl.includes('keycloak')) {
      console.log('  ⚠️  OIDC not enabled - skipping Keycloak test');
      console.log('     (Set OIDC_ENABLED=true to enable Keycloak authentication)');
      return { success: true, skipped: true };
    }

    console.log('  ✅ Redirected to Keycloak login');

    // Step 2: Fill in Keycloak credentials
    console.log('\nStep 2: Entering Keycloak credentials...');
    await page.waitForSelector('#username, input[name="username"]', { timeout: 5000 });
    await page.type('#username, input[name="username"]', OIDC_USERNAME);
    await page.type('#password, input[name="password"]', OIDC_PASSWORD);
    console.log(`  ✅ Username: ${OIDC_USERNAME}`);
    console.log(`  ✅ Password: ${'*'.repeat(OIDC_PASSWORD.length)}`);

    // Step 3: Submit Keycloak login
    console.log('\nStep 3: Submitting Keycloak login...');
    await Promise.all([
      page.click('input[type="submit"], button[type="submit"]'),
      page.waitForNavigation({ waitUntil: 'networkidle2', timeout: 15000 })
    ]);

    const afterLoginUrl = page.url();
    console.log(`  📍 URL after login: ${afterLoginUrl}`);

    if (afterLoginUrl.includes('keycloak')) {
      console.log('  ❌ FAILED: Still on Keycloak - credentials rejected or consent needed');
      return { success: false, error: 'OIDC login failed' };
    }

    console.log('  ✅ OIDC login successful - redirected to application');

    // Step 4: Verify auth state
    console.log('\nStep 4: Verifying authentication state...');
    const authState = await page.evaluate(() => {
      return {
        hasToken: !!localStorage.getItem('auth-token'),
        hasCookie: document.cookie.includes('session_id'),
      };
    });

    console.log(`  ✅ Auth token in localStorage: ${authState.hasToken}`);
    console.log(`  ✅ Session cookie: ${authState.hasCookie}`);

    // Step 5: Navigate to providers page
    console.log('\nStep 5: Navigating to providers page...');
    await page.goto(`${BASE_URL}/providers`, { waitUntil: 'networkidle2' });
    await new Promise(resolve => setTimeout(resolve, 2000));

    const providersPageState = await page.evaluate(() => {
      const hasError = document.body.textContent.includes('Unexpected token') ||
                      document.body.textContent.includes('is not valid JSON');

      const hasProviderTable = !!document.querySelector('table');
      const errorElement = document.querySelector('.text-red-500, [role="alert"]');
      const errorText = errorElement ? errorElement.textContent.trim() : null;

      return {
        url: window.location.href,
        hasError,
        errorText,
        hasProviderTable
      };
    });

    console.log(`  📍 URL: ${providersPageState.url}`);
    console.log(`  📊 Provider table present: ${providersPageState.hasProviderTable ? '✅' : '❌'}`);

    if (providersPageState.hasError || providersPageState.errorText) {
      console.log(`  ❌ ERROR DETECTED: ${providersPageState.errorText || 'JSON parse error'}`);
      await page.screenshot({ path: '/tmp/providers-error-oidc.png', fullPage: true });
      return { success: false, error: providersPageState.errorText || 'Error on providers page' };
    }

    console.log('  ✅ Providers page loaded successfully!');

    // Step 6: Test session persistence
    console.log('\nStep 6: Testing session persistence...');
    await page.reload({ waitUntil: 'networkidle2' });
    await new Promise(resolve => setTimeout(resolve, 2000));

    const afterRefreshUrl = page.url();
    if (afterRefreshUrl.includes('keycloak') || afterRefreshUrl.includes('/login')) {
      console.log('  ❌ FAILED: Logged out after refresh');
      return { success: false, error: 'Session not persistent' };
    }

    console.log('  ✅ Session persisted after refresh');

    await page.screenshot({ path: '/tmp/providers-success-oidc.png', fullPage: true });
    console.log('\n📸 Screenshot: /tmp/providers-success-oidc.png');

    console.log('\n╔═══════════════════════════════════════════════════════════╗');
    console.log('║  ✅ OIDC AUTH TEST PASSED                                 ║');
    console.log('╚═══════════════════════════════════════════════════════════╝\n');

    return { success: true };

  } catch (error) {
    console.log(`\n❌ TEST FAILED: ${error.message}`);
    console.error(error.stack);
    await page.screenshot({ path: '/tmp/providers-error-oidc.png', fullPage: true });
    return { success: false, error: error.message };

  } finally {
    if (!TEST_CONFIG.headless) {
      console.log('\n🔍 Browser left open for inspection. Press Ctrl+C to close.');
      await new Promise(resolve => {
        process.on('SIGINT', () => {
          console.log('\n\nClosing browser...');
          resolve();
        });
      });
    }
    await browser.close();
  }
}

/**
 * Main test runner
 */
async function main() {
  console.log('\n╔═══════════════════════════════════════════════════════════╗');
  console.log('║  Innominatus Authentication & Providers Page Tests       ║');
  console.log('╠═══════════════════════════════════════════════════════════╣');
  console.log('║  This test verifies:                                      ║');
  console.log('║  • Local auth (admin/admin123) works                      ║');
  console.log('║  • OIDC auth (demo-user via Keycloak) works               ║');
  console.log('║  • Providers page loads without JSON errors               ║');
  console.log('║  • Sessions persist across page refreshes                 ║');
  console.log('║  • No random logouts during navigation                    ║');
  console.log('╚═══════════════════════════════════════════════════════════╝\n');

  console.log(`📍 Testing against: ${BASE_URL}`);
  console.log(`🔐 Keycloak URL: ${KEYCLOAK_URL}`);
  console.log(`🖥️  Headless: ${TEST_CONFIG.headless}`);
  console.log('');

  const results = [];

  // Test 1: Local Auth
  const localResult = await testLocalAuth();
  results.push({ name: 'Local Auth', ...localResult });

  // Test 2: OIDC Auth
  const oidcResult = await testOIDCAuth();
  results.push({ name: 'OIDC Auth', ...oidcResult });

  // Summary
  console.log('\n╔═══════════════════════════════════════════════════════════╗');
  console.log('║  TEST SUMMARY                                             ║');
  console.log('╚═══════════════════════════════════════════════════════════╝\n');

  let allPassed = true;
  for (const result of results) {
    if (result.skipped) {
      console.log(`⏭️  ${result.name}: SKIPPED`);
    } else if (result.success) {
      console.log(`✅ ${result.name}: PASSED`);
    } else {
      console.log(`❌ ${result.name}: FAILED - ${result.error}`);
      allPassed = false;
    }
  }

  console.log('');

  if (allPassed) {
    console.log('╔═══════════════════════════════════════════════════════════╗');
    console.log('║  🎉 ALL TESTS PASSED                                      ║');
    console.log('╚═══════════════════════════════════════════════════════════╝\n');
    process.exit(0);
  } else {
    console.log('╔═══════════════════════════════════════════════════════════╗');
    console.log('║  ❌ SOME TESTS FAILED                                     ║');
    console.log('╚═══════════════════════════════════════════════════════════╝\n');
    process.exit(1);
  }
}

main().catch(error => {
  console.error('\n💥 Fatal error:', error);
  process.exit(1);
});
