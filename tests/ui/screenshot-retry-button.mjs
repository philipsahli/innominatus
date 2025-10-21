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

    // Login with admin/admin123
    console.log('Attempting login with admin/admin123...');
    await page.type('input[name="username"]', 'admin');
    await page.type('input[name="password"]', 'admin123');

    // Click login button
    await page.click('button[type="submit"]');
    await sleep(3000);

    console.log('After login URL:', page.url());

    // Navigate to workflows page
    console.log('Navigating to workflows page...');
    await page.goto('http://localhost:8081/workflows', {
      waitUntil: 'networkidle2',
      timeout: 10000
    });

    await sleep(3000);
    console.log('Final URL:', page.url());

    console.log('Taking screenshot...');
    await page.screenshot({
      path: '/tmp/retry-button-screenshot.png',
      fullPage: true
    });

    console.log('Screenshot saved to /tmp/retry-button-screenshot.png');

  } catch (error) {
    console.error('Error:', error.message);
    throw error;
  } finally {
    await browser.close();
  }
}

takeScreenshot();
