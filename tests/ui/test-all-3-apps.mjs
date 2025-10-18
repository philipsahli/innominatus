#!/usr/bin/env node

import puppeteer from 'puppeteer';

const BASE_URL = 'http://localhost:8081';
const APPS = ['web-application', 'world-app3', 'app-with-storage'];
const SESSION_ID = 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46';

async function testApp(appName) {
  const browser = await puppeteer.launch({ headless: true, args: ['--no-sandbox', '--disable-cache'] });
  const page = await browser.newPage();
  await page.setCacheEnabled(false);

  const errors = [];
  page.on('pageerror', err => errors.push(err.message));

  try {
    await page.setCookie({
      name: 'session_id',
      value: SESSION_ID,
      domain: 'localhost',
      path: '/',
      httpOnly: true,
      secure: false
    });

    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle2' });
    await page.evaluate(() => {
      localStorage.setItem('auth-token', 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46');
    });

    await page.goto(`${BASE_URL}/graph/${appName}`, { waitUntil: 'networkidle2' });
    await new Promise(r => setTimeout(r, 3000));

    const hasGraph = await page.evaluate(() => {
      const text = document.body.innerText;
      return text.includes('workflow') || text.includes('step') || text.includes('waiting') || text.includes('failed');
    });

    const hasError = errors.length > 0;

    console.log(`${appName.padEnd(20)} ${hasError ? '❌ Error' : hasGraph ? '✅ OK' : '⚠️  No graph'} (${errors.length} errors)`);

    await browser.close();
    return !hasError && hasGraph;
  } catch (err) {
    console.log(`${appName.padEnd(20)} ❌ Exception: ${err.message}`);
    await browser.close();
    return false;
  }
}

console.log('\n🧪 Testing all graph pages with text visualization...\n');
let allPassed = true;

for (const app of APPS) {
  const passed = await testApp(app);
  if (!passed) allPassed = false;
}

console.log('\n' + (allPassed ? '✅ All tests passed!' : '❌ Some tests failed'));
process.exit(allPassed ? 0 : 1);
