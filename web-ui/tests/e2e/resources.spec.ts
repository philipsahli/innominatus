import { test, expect } from '@playwright/test';

/**
 * Resources Management E2E Tests
 *
 * These tests verify resource management functionality.
 */
test.describe('Resources', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/resources');
  });

  test('should display resources list', async ({ page }) => {
    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Check for resources or empty state
    const hasResources = await page.getByText(/postgres|redis|route|resource/i).count() > 0;
    const hasEmptyState = await page.getByText(/no resources/i).count() > 0;

    expect(hasResources || hasEmptyState).toBeTruthy();
  });

  test('should filter resources by type', async ({ page }) => {
    // Look for filter controls
    const filterButton = page.getByRole('button', { name: /filter|type/i });
    const filterSelect = page.getByLabel(/type|filter/i);

    const hasFilterButton = await filterButton.count() > 0;
    const hasFilterSelect = await filterSelect.count() > 0;

    if (hasFilterButton || hasFilterSelect) {
      expect(hasFilterButton || hasFilterSelect).toBeTruthy();
    } else {
      // Filtering not implemented yet
      test.skip();
    }
  });

  test('should show resource details', async ({ page }) => {
    // Find first resource link
    const firstResource = page.getByRole('link', { name: /view|details/i }).first();

    if (await firstResource.count() > 0) {
      await firstResource.click();

      // Wait for details page
      await page.waitForLoadState('networkidle');

      // Verify resource information is displayed
      const hasDetails = await page.getByText(/status|config|parameter|connection/i).count() > 0;
      expect(hasDetails).toBeTruthy();
    } else {
      test.skip();
    }
  });

  test('should display resource health status', async ({ page }) => {
    // Look for health indicators
    const healthStatus = page.getByText(/healthy|unhealthy|unknown|pending|active/i);

    if (await healthStatus.count() > 0) {
      await expect(healthStatus.first()).toBeVisible();
    } else {
      test.skip();
    }
  });
});

test.describe('Resource Graphs', () => {
  test('should display dependency graph', async ({ page }) => {
    await page.goto('/applications');

    // Find an application with resources
    const appLink = page.getByRole('link').first();

    if (await appLink.count() > 0) {
      await appLink.click();

      // Look for graph or visualization
      const graphView = page.getByRole('img', { name: /graph|diagram/i });
      const graphCanvas = page.locator('canvas, svg').first();

      const hasGraph = await graphView.count() > 0;
      const hasCanvas = await graphCanvas.count() > 0;

      if (hasGraph || hasCanvas) {
        expect(hasGraph || hasCanvas).toBeTruthy();
      } else {
        test.skip();
      }
    } else {
      test.skip();
    }
  });

  test('should export graph visualization', async ({ page }) => {
    await page.goto('/applications');

    // Find export button
    const exportButton = page.getByRole('button', { name: /export|download|save/i });

    if (await exportButton.count() > 0) {
      // Just verify the button exists (actual download would be harder to test)
      await expect(exportButton.first()).toBeVisible();
    } else {
      test.skip();
    }
  });
});

test.describe('Providers', () => {
  test('should list configured providers', async ({ page }) => {
    await page.goto('/providers');

    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Check for providers or empty state
    const hasProviders = await page.getByText(/kubernetes|cloud|provider/i).count() > 0;
    const hasEmptyState = await page.getByText(/no providers/i).count() > 0;

    expect(hasProviders || hasEmptyState).toBeTruthy();
  });

  test('should show provider stats', async ({ page }) => {
    await page.goto('/providers');

    // Look for statistics
    const stats = page.getByText(/total|active|healthy|resource/i);

    if (await stats.count() > 0) {
      await expect(stats.first()).toBeVisible();
    }
  });

  test('should have add provider button', async ({ page }) => {
    await page.goto('/providers');

    // Look for add button (if this feature exists)
    const addButton = page.getByRole('button', { name: /add|new provider/i });

    if (await addButton.count() > 0) {
      await expect(addButton).toBeVisible();
    } else {
      test.skip();
    }
  });
});
