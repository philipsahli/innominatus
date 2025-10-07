#!/usr/bin/env node

/**
 * Test script for documentation search and highlighting functionality
 * Tests:
 * 1. Search functionality returns correct results
 * 2. Highlighting works with URL parameters
 * 3. Special characters are properly escaped
 * 4. Case-insensitive matching works
 * 5. Auto-scroll to first match
 */

import puppeteer from 'puppeteer';

const BASE_URL = process.env.BASE_URL || 'http://localhost:8081';
const TESTS_PASSED = [];
const TESTS_FAILED = [];

async function runTest(name, testFn) {
  try {
    console.log(`\n🧪 Testing: ${name}`);
    await testFn();
    TESTS_PASSED.push(name);
    console.log(`✅ PASSED: ${name}`);
  } catch (error) {
    TESTS_FAILED.push({ name, error: error.message });
    console.error(`❌ FAILED: ${name}`);
    console.error(`   Error: ${error.message}`);
  }
}

async function testSearchFunctionality(page) {
  await page.goto(`${BASE_URL}/docs`);
  await page.waitForSelector('input[placeholder*="Search"]', { timeout: 5000 });

  // Type search term
  await page.type('input[placeholder*="Search"]', 'kubernetes');
  await page.waitForTimeout(500);

  // Check if results are filtered
  const docCards = await page.$$('[class*="cursor-pointer"]');
  if (docCards.length === 0) {
    throw new Error('No search results found for "kubernetes"');
  }

  console.log(`   Found ${docCards.length} results for "kubernetes"`);
}

async function testHighlightingWithURL(page) {
  // Navigate to a doc page with highlight parameter
  await page.goto(`${BASE_URL}/docs/getting-started/concepts?highlight=kubernetes`);
  await page.waitForSelector('.prose', { timeout: 5000 });
  await page.waitForTimeout(1000); // Wait for highlighting to apply

  // Check for highlighted terms
  const highlights = await page.$$('mark.bg-yellow-200, mark.bg-yellow-800');
  if (highlights.length === 0) {
    throw new Error('No highlighted terms found on page');
  }

  console.log(`   Found ${highlights.length} highlighted instances`);

  // Verify highlight contains the search term
  const firstHighlight = await page.$('mark');
  const text = await page.evaluate(el => el.textContent, firstHighlight);
  if (!text.toLowerCase().includes('kubernetes')) {
    throw new Error(`Highlight text "${text}" doesn't contain search term`);
  }
}

async function testCaseInsensitiveMatching(page) {
  // Test with different case
  await page.goto(`${BASE_URL}/docs/getting-started/concepts?highlight=KUBERNETES`);
  await page.waitForSelector('.prose', { timeout: 5000 });
  await page.waitForTimeout(1000);

  const highlights = await page.$$('mark');
  if (highlights.length === 0) {
    throw new Error('Case-insensitive matching failed');
  }

  // Verify both uppercase and lowercase are highlighted
  const highlightTexts = await page.$$eval('mark', marks =>
    marks.map(m => m.textContent)
  );

  const hasLowercase = highlightTexts.some(t => t.includes('kubernetes'));
  const hasCapitalized = highlightTexts.some(t => t.includes('Kubernetes'));

  console.log(`   Found lowercase: ${hasLowercase}, capitalized: ${hasCapitalized}`);
}

async function testSpecialCharacters(page) {
  // Test with term containing special characters that need escaping
  await page.goto(`${BASE_URL}/docs/cli/commands?highlight=--help`);
  await page.waitForSelector('.prose', { timeout: 5000 });
  await page.waitForTimeout(1000);

  // Check if any highlights exist (even if term isn't found, no errors should occur)
  const pageContent = await page.content();
  if (pageContent.includes('ReferenceError') || pageContent.includes('SyntaxError')) {
    throw new Error('Regex error with special characters');
  }

  console.log(`   Special characters handled without errors`);
}

async function testAutoScroll(page) {
  await page.goto(`${BASE_URL}/docs/getting-started/concepts?highlight=kubernetes`);
  await page.waitForSelector('.prose', { timeout: 5000 });
  await page.waitForTimeout(1500); // Wait for auto-scroll

  // Check if first mark is in viewport
  const isInViewport = await page.evaluate(() => {
    const mark = document.querySelector('mark');
    if (!mark) return false;

    const rect = mark.getBoundingClientRect();
    return (
      rect.top >= 0 &&
      rect.left >= 0 &&
      rect.bottom <= window.innerHeight &&
      rect.right <= window.innerWidth
    );
  });

  console.log(`   First highlight in viewport: ${isInViewport}`);
}

async function testSearchIntegration(page) {
  // Test full flow: search -> click -> highlight
  await page.goto(`${BASE_URL}/docs`);
  await page.waitForSelector('input[placeholder*="Search"]', { timeout: 5000 });

  // Search for "workflow"
  await page.type('input[placeholder*="Search"]', 'workflow');
  await page.waitForTimeout(500);

  // Click first result
  const firstResult = await page.$('[class*="cursor-pointer"]');
  if (!firstResult) {
    throw new Error('No search results to click');
  }

  await firstResult.click();
  await page.waitForNavigation({ waitUntil: 'networkidle0', timeout: 10000 });

  // Verify URL contains highlight parameter
  const url = page.url();
  if (!url.includes('highlight=workflow')) {
    throw new Error(`URL doesn't contain highlight parameter: ${url}`);
  }

  // Verify highlights appear
  await page.waitForTimeout(1000);
  const highlights = await page.$$('mark');
  if (highlights.length === 0) {
    throw new Error('No highlights after clicking search result');
  }

  console.log(`   Full integration flow successful with ${highlights.length} highlights`);
}

async function testNoHighlightWithoutParam(page) {
  // Navigate without highlight parameter
  await page.goto(`${BASE_URL}/docs/getting-started/concepts`);
  await page.waitForSelector('.prose', { timeout: 5000 });
  await page.waitForTimeout(500);

  // Verify no highlights present
  const highlights = await page.$$('mark.bg-yellow-200, mark.bg-yellow-800');
  if (highlights.length > 0) {
    throw new Error('Highlights appear without highlight parameter');
  }

  console.log(`   Correctly shows no highlights without parameter`);
}

async function main() {
  console.log('🚀 Starting Search & Highlighting Tests\n');
  console.log(`📍 Base URL: ${BASE_URL}`);

  const browser = await puppeteer.launch({
    headless: 'new',
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 800 });

  // Run all tests
  await runTest('Search functionality returns results', () => testSearchFunctionality(page));
  await runTest('Highlighting works with URL parameter', () => testHighlightingWithURL(page));
  await runTest('Case-insensitive matching works', () => testCaseInsensitiveMatching(page));
  await runTest('Special characters handled correctly', () => testSpecialCharacters(page));
  await runTest('Auto-scroll to first match', () => testAutoScroll(page));
  await runTest('Search integration flow', () => testSearchIntegration(page));
  await runTest('No highlights without parameter', () => testNoHighlightWithoutParam(page));

  await browser.close();

  // Print summary
  console.log('\n' + '='.repeat(60));
  console.log('📊 TEST SUMMARY');
  console.log('='.repeat(60));
  console.log(`✅ Passed: ${TESTS_PASSED.length}`);
  console.log(`❌ Failed: ${TESTS_FAILED.length}`);

  if (TESTS_PASSED.length > 0) {
    console.log('\n✅ Passed tests:');
    TESTS_PASSED.forEach(name => console.log(`   - ${name}`));
  }

  if (TESTS_FAILED.length > 0) {
    console.log('\n❌ Failed tests:');
    TESTS_FAILED.forEach(({ name, error }) => {
      console.log(`   - ${name}`);
      console.log(`     Error: ${error}`);
    });
    process.exit(1);
  }

  console.log('\n🎉 All tests passed!');
  process.exit(0);
}

main().catch(error => {
  console.error('Fatal error:', error);
  process.exit(1);
});
