# innominatus - UI Tests (Puppeteer)

This directory contains browser automation tests for the innominatus Web UI using Puppeteer.

## Test Suites

### `graph-visualization.test.js`

Comprehensive end-to-end tests for the workflow graph visualization feature.

**Test Coverage:**
1. **Login Flow** - Authentication and session management
2. **Graph List Page** - Application listing and navigation
3. **Graph Visualization** - React Flow rendering, nodes, edges
4. **Real-Time Updates** - Server-Sent Events (SSE) verification
5. **Responsive Design** - Mobile, tablet, desktop viewports

**Features Tested:**
- User authentication (login/logout)
- Application discovery and listing
- Graph rendering with React Flow
- Node and edge visualization
- Zoom and pan controls
- Node selection and interaction
- Export functionality (if implemented)
- SSE real-time updates
- Responsive layouts across devices

## Prerequisites

- Node.js 18+ (for Puppeteer)
- innominatus server running on http://localhost:8081
- Valid test credentials (admin/admin123 by default)
- Applications deployed (world-app3, test-graph-app)

## Installation

```bash
cd tests/ui
npm install
```

## Running Tests

### Default (Headless Mode)
```bash
npm test
```

### Debug Mode (with Browser Window)
```bash
HEADLESS=false npm test
```

### Custom Configuration
```bash
# Custom base URL
BASE_URL=http://innominatus.localtest.me npm test

# Custom credentials
TEST_USERNAME=alice TEST_PASSWORD=alice123 npm test

# Debug with Node.js inspector
npm run test:debug
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8081` | innominatus server URL |
| `TEST_USERNAME` | `admin` | Login username |
| `TEST_PASSWORD` | `admin123` | Login password |
| `HEADLESS` | `true` | Run browser in headless mode |

## Screenshots

All test runs automatically capture screenshots at key points:

- `screenshots/01-login-page.png` - Initial login page
- `screenshots/02-login-credentials-entered.png` - Before submit
- `screenshots/03-dashboard-after-login.png` - Post-login redirect
- `screenshots/04-graph-list-page.png` - Application list
- `screenshots/05-graph-viz-{app}-initial.png` - Graph visualization
- `screenshots/06-graph-viz-{app}-zoomed.png` - Zoom controls test
- `screenshots/07-graph-viz-{app}-node-selected.png` - Node interaction
- `screenshots/08-responsive-{mobile|tablet|desktop}.png` - Responsive layouts

Screenshots are saved to `tests/ui/screenshots/` directory.

## Test Output

Tests provide colored terminal output:
- ✓ Green - Passed tests
- ✗ Red - Failed tests
- ○ Yellow - Skipped tests

Example output:
```
================================================
  innominatus - Puppeteer UI Tests
  Graph Visualization Feature
================================================

=== Test 1: Login Flow ===
✓ Login page loaded: Username and password inputs found
✓ Login successful: Redirected to http://localhost:8081/dashboard

=== Test 2: Graph List Page ===
✓ Applications visible in list: Found: world-app3 test-graph-app
✓ Navigation to graph detail: Clicked world-app3 card

=== Test 3: Graph Visualization (world-app3) ===
✓ Graph visualization rendered: 27 nodes, 24 edges
✓ Zoom controls functional: Zoom in/out buttons clickable
✓ Node interaction: Node clickable and selectable

================================================
  Test Summary
================================================
Total: 12 tests
Passed: 11
Failed: 0
Skipped: 1
Pass Rate: 91.7%
```

## Debugging Failed Tests

1. **Run with visible browser:**
   ```bash
   HEADLESS=false npm test
   ```

2. **Check error screenshots:**
   ```bash
   ls -la screenshots/error-*.png
   ```

3. **Enable Node.js inspector:**
   ```bash
   npm run test:debug
   # Open chrome://inspect in Chrome
   ```

4. **Verify server is running:**
   ```bash
   curl http://localhost:8081/health
   ```

5. **Check browser console errors:**
   Test script logs browser console errors automatically

## CI/CD Integration

### GitHub Actions Example
```yaml
name: UI Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '20'

      - name: Start innominatus server
        run: |
          go build -o innominatus cmd/server/main.go
          ./innominatus &
          sleep 5

      - name: Run UI tests
        run: |
          cd tests/ui
          npm install
          npm test

      - name: Upload screenshots
        if: failure()
        uses: actions/upload-artifact@v3
        with:
          name: test-screenshots
          path: tests/ui/screenshots/
```

## Troubleshooting

### Issue: "Navigation timeout"
- **Cause:** Server not responding or slow to start
- **Fix:** Increase timeout in test or verify server health

### Issue: "Element not found"
- **Cause:** UI structure changed or slow rendering
- **Fix:** Increase wait times or update selectors

### Issue: "Login failed"
- **Cause:** Invalid credentials or auth system changed
- **Fix:** Verify credentials in users.yaml or OIDC config

### Issue: "No applications found"
- **Cause:** No apps deployed to test database
- **Fix:** Run `./innominatus-ctl run deploy-app score-spec-k8s.yaml`

## Writing New Tests

Example test function:
```javascript
async function testNewFeature(page) {
  log('\n=== Test X: New Feature ===', 'blue');

  try {
    // Navigate to page
    await page.goto(`${BASE_URL}/feature`, { waitUntil: 'networkidle2' });

    // Take screenshot
    await takeScreenshot(page, 'feature-test');

    // Test functionality
    const result = await page.evaluate(() => {
      // Run code in browser context
      return document.querySelector('.feature').textContent;
    });

    // Assert and log
    if (result.includes('expected')) {
      logTest('New feature', 'PASS', 'Feature works correctly');
    } else {
      throw new Error('Feature not working as expected');
    }

  } catch (err) {
    logTest('New feature', 'FAIL', err.message);
    await takeScreenshot(page, 'error-feature-failed');
    throw err;
  }
}
```

## Best Practices

1. **Always take screenshots** - Especially on failures
2. **Wait for network idle** - Before assertions
3. **Handle timeouts gracefully** - Don't fail on slow responses
4. **Log progress** - Help debugging without running browser visible
5. **Clean up resources** - Close browser in finally block
6. **Test in multiple viewports** - Ensure responsive design works

## Resources

- [Puppeteer Documentation](https://pptr.dev/)
- [React Flow Testing](https://reactflow.dev/learn/troubleshooting)
- [innominatus API Documentation](../../docs/api/README.md)

## Maintenance

**Created:** 2025-10-14
**Last Updated:** 2025-10-14
**Maintained by:** innominatus core team

For questions or issues, open a GitHub issue or contact the platform team.
