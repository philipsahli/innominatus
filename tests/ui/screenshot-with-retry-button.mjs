import puppeteer from 'puppeteer';

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function takeScreenshot() {
  const browser = await puppeteer.launch({
    headless: false,  // Run in visible mode to see what's happening
    args: ['--no-sandbox', '--disable-setuid-sandbox']
  });

  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1920, height: 1080 });

    console.log('Navigating to login page...');
    await page.goto('http://localhost:8081/login', {
      waitUntil: 'networkidle2',
      timeout: 10000
    });

    await sleep(1000);

    // Login
    console.log('Logging in...');
    await page.type('input[name="username"]', 'admin');
    await page.type('input[name="password"]', 'admin123');
    await page.click('button[type="submit"]');
    await sleep(3000);

    // Go to workflows page
    console.log('Navigating to workflows page...');
    await page.goto('http://localhost:8081/workflows', {
      waitUntil: 'networkidle2',
      timeout: 10000
    });

    await sleep(2000);

    // Check if there are any failed workflows
    const failedRows = await page.$$('tr:has([class*="text-red"])');
    console.log(`Found ${failedRows.length} rows with failed status`);

    // Take screenshot
    console.log('Taking screenshot...');
    await page.screenshot({
      path: '/tmp/retry-button-final.png',
      fullPage: true
    });

    console.log('Screenshot saved to /tmp/retry-button-final.png');

    // Keep browser open for 5 seconds so we can see it
    await sleep(5000);

  } catch (error) {
    console.error('Error:', error.message);
    throw error;
  } finally {
    await browser.close();
  }
}

takeScreenshot();
