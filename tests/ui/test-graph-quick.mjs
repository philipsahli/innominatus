#!/usr/bin/env node

import puppeteer from 'puppeteer';

const browser = await puppeteer.launch({headless: true, args: ['--no-sandbox', '--disable-cache']});
const page = await browser.newPage();
await page.setCacheEnabled(false);

const errors = [];
page.on('pageerror', err => errors.push(err.message));

await page.setCookie({
  name: 'session_id',
  value: 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46',
  domain: 'localhost',
  path: '/',
  httpOnly: true,
  secure: false
});

await page.goto('http://localhost:8081/login', {waitUntil: 'networkidle2'});
await page.evaluate(() => {
  localStorage.setItem('auth-token', 'bdc213029df1b9362937ece4467c12022100e619e037479ce73005f1860a8e46');
});

console.log('Testing app-with-storage...');
await page.goto('http://localhost:8081/graph/app-with-storage', {waitUntil: 'networkidle2'});
await new Promise(r => setTimeout(r, 5000));

const hasReactFlow = await page.evaluate(() => document.querySelectorAll('.react-flow').length > 0);
const nodes = await page.evaluate(() => document.querySelectorAll('.react-flow__node').length);

console.log(`React Flow present: ${hasReactFlow}`);
console.log(`Nodes found: ${nodes}`);
console.log(`Errors: ${errors.length}`);
if (errors.length) console.log(`  First error: ${errors[0]}`);

await page.screenshot({path: '/tmp/quick-test.png', fullPage: true});
console.log('Screenshot: /tmp/quick-test.png');

await browser.close();
