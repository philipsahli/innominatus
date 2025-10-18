import puppeteer from 'puppeteer';

async function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function takeScreenshot() {
  const browser = await puppeteer.launch({
    headless: true,
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

    // Check for failed workflows
    const failedCount = await page.$$eval('tr', rows => {
      return rows.filter(row => row.textContent.includes('failed')).length;
    });
    console.log(`Found ${failedCount} failed workflow(s)`);

    // Take full page screenshot
    console.log('Taking full page screenshot...');
    await page.screenshot({
      path: '/tmp/retry-button-with-failed-workflow.png',
      fullPage: true
    });

    // Also take a focused screenshot of just the table area
    const table = await page.$('table');
    if (table) {
      await table.screenshot({
        path: '/tmp/retry-button-table-only.png'
      });
      console.log('Table screenshot saved to /tmp/retry-button-table-only.png');
    }

    console.log('Full screenshot saved to /tmp/retry-button-with-failed-workflow.png');

  } catch (error) {
    console.error('Error:', error.message);
    throw error;
  } finally {
    await browser.close();
  }
}

takeScreenshot();
