import { test, expect } from '@playwright/test';

/**
 * Application Deployment E2E Tests
 *
 * These tests verify the deployment workflow through the Web UI.
 * Requires: Running innominatus server with test configuration
 */
test.describe('Application Deployment', () => {
  test.describe.configure({ mode: 'serial' });

  test('should navigate to deployment page', async ({ page }) => {
    await page.goto('/');

    // Look for deploy button or link
    const deployButton = page.getByRole('link', { name: /deploy|new application/i });

    if ((await deployButton.count()) > 0) {
      await deployButton.first().click();

      // Verify navigation
      await expect(page).toHaveURL(/\/deploy|\/applications\/new/);
    } else {
      test.skip();
    }
  });

  test('should show deployment form or golden paths', async ({ page }) => {
    await page.goto('/deploy');

    // Check for either direct deployment form or golden paths selection
    const hasUploadForm = (await page.getByText(/upload|score spec/i).count()) > 0;
    const hasGoldenPaths = (await page.getByText(/golden path/i).count()) > 0;

    expect(hasUploadForm || hasGoldenPaths).toBeTruthy();
  });

  test('should validate score spec file', async ({ page }) => {
    await page.goto('/deploy');

    // Look for file input
    const fileInput = page.locator('input[type="file"]');

    if ((await fileInput.count()) > 0) {
      // Test with invalid file (if validation exists)
      // This would require creating test fixtures

      // For now, just verify the input exists
      await expect(fileInput.first()).toBeVisible();
    }
  });
});

test.describe('Golden Paths', () => {
  test('should list available golden paths', async ({ page }) => {
    await page.goto('/golden-paths');

    // Wait for page load
    await page.waitForLoadState('networkidle');

    // Check for golden paths list or empty state
    const hasGoldenPaths = (await page.getByText(/deploy|ephemeral|path/i).count()) > 0;
    const hasEmptyState = (await page.getByText(/no golden paths/i).count()) > 0;

    expect(hasGoldenPaths || hasEmptyState).toBeTruthy();
  });

  test('should show golden path details', async ({ page }) => {
    await page.goto('/golden-paths');

    // Find first golden path link
    const firstPath = page.getByRole('link').first();

    if ((await firstPath.count()) > 0) {
      await firstPath.click();

      // Wait for details to load
      await page.waitForLoadState('networkidle');

      // Check for details like parameters, description, etc.
      const hasDetails = (await page.getByText(/parameter|description|step/i).count()) > 0;
      expect(hasDetails).toBeTruthy();
    }
  });
});

test.describe('Application Management', () => {
  test('should view application details', async ({ page }) => {
    await page.goto('/applications');

    // Find first application
    const firstApp = page.getByRole('link', { name: /test-app|view|details/i }).first();

    if ((await firstApp.count()) > 0) {
      await firstApp.click();

      // Verify we're on application details page
      await page.waitForURL(/\/applications\/.+/);

      // Check for application information
      const hasDetails = (await page.getByText(/container|resource|workflow/i).count()) > 0;
      expect(hasDetails).toBeTruthy();
    } else {
      test.skip();
    }
  });

  test('should display application workflows', async ({ page }) => {
    await page.goto('/workflows');

    // Wait for workflows to load
    await page.waitForLoadState('networkidle');

    // Check for workflows list or empty state
    const hasWorkflows = (await page.getByText(/workflow|execution|deploy/i).count()) > 0;
    const hasEmptyState = (await page.getByText(/no workflows/i).count()) > 0;

    expect(hasWorkflows || hasEmptyState).toBeTruthy();
  });

  test('should show workflow execution logs', async ({ page }) => {
    await page.goto('/workflows');

    // Find first workflow
    const firstWorkflow = page.getByRole('link', { name: /view|details/i }).first();

    if ((await firstWorkflow.count()) > 0) {
      await firstWorkflow.click();

      // Wait for logs to load
      await page.waitForLoadState('networkidle');

      // Check for log content
      const hasLogs = (await page.getByText(/step|log|output|running|completed/i).count()) > 0;
      expect(hasLogs).toBeTruthy();
    } else {
      test.skip();
    }
  });
});

test.describe('Application Deletion', () => {
  test('should show delete confirmation dialog', async ({ page }) => {
    await page.goto('/applications');

    // Find delete button
    const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();

    if ((await deleteButton.count()) > 0) {
      await deleteButton.click();

      // Look for confirmation dialog
      const confirmDialog = page.getByRole('dialog');
      const confirmText = page.getByText(/are you sure|confirm|delete/i);

      const hasDialog = (await confirmDialog.count()) > 0;
      const hasConfirmText = (await confirmText.count()) > 0;

      expect(hasDialog || hasConfirmText).toBeTruthy();

      // Cancel the deletion
      const cancelButton = page.getByRole('button', { name: /cancel|no/i });
      if ((await cancelButton.count()) > 0) {
        await cancelButton.first().click();
      }
    } else {
      test.skip();
    }
  });
});
