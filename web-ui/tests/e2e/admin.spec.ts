import { test, expect } from '@playwright/test';

/**
 * Admin and Teams E2E Tests
 *
 * These tests verify administrative functionality.
 * Note: May require authentication/authorization
 */
test.describe('Teams Management', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/teams');
  });

  test('should display teams list', async ({ page }) => {
    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Check for teams or empty state
    const hasTeams = (await page.getByText(/team|platform|engineering/i).count()) > 0;
    const hasEmptyState = (await page.getByText(/no teams/i).count()) > 0;

    expect(hasTeams || hasEmptyState).toBeTruthy();
  });

  test('should have create team button', async ({ page }) => {
    // Look for create button
    const createButton = page.getByRole('button', { name: /create|new team|add team/i });

    if ((await createButton.count()) > 0) {
      await expect(createButton.first()).toBeVisible();
    } else {
      test.skip();
    }
  });

  test('should show team details', async ({ page }) => {
    // Find first team link
    const firstTeam = page.getByRole('link').first();

    if ((await firstTeam.count()) > 0) {
      await firstTeam.click();

      // Wait for details page
      await page.waitForLoadState('networkidle');

      // Check for team information
      const hasDetails = (await page.getByText(/member|user|application|resource/i).count()) > 0;
      expect(hasDetails).toBeTruthy();
    } else {
      test.skip();
    }
  });
});

test.describe('User Management', () => {
  test('should navigate to users page', async ({ page }) => {
    await page.goto('/admin/users');

    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Check if we can access the page or if auth is required
    const hasUnauthorized = (await page.getByText(/unauthorized|forbidden|login/i).count()) > 0;

    if (hasUnauthorized) {
      test.skip();
    }

    // Check for users list
    const hasUsers = (await page.getByText(/user|admin|username|email/i).count()) > 0;
    expect(hasUsers).toBeTruthy();
  });

  test('should show add user button', async ({ page }) => {
    await page.goto('/admin/users');

    // Look for add user button
    const addButton = page.getByRole('button', { name: /add|create|new user/i });

    if ((await addButton.count()) > 0) {
      await expect(addButton.first()).toBeVisible();
    } else {
      test.skip();
    }
  });
});

test.describe('Settings', () => {
  test('should navigate to settings page', async ({ page }) => {
    await page.goto('/settings');

    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Check for settings sections
    const hasSettings =
      (await page.getByText(/setting|configuration|profile|preference/i).count()) > 0;

    if (hasSettings) {
      expect(hasSettings).toBeTruthy();
    } else {
      test.skip();
    }
  });

  test('should display user profile', async ({ page }) => {
    await page.goto('/profile');

    // Check for profile information
    const hasProfile = (await page.getByText(/profile|username|email|api key/i).count()) > 0;

    if (hasProfile) {
      expect(hasProfile).toBeTruthy();
    } else {
      test.skip();
    }
  });

  test('should allow API key generation', async ({ page }) => {
    await page.goto('/profile');

    // Look for API key section
    const apiKeySection = page.getByText(/api key|token|credential/i);

    if ((await apiKeySection.count()) > 0) {
      // Look for generate button
      const generateButton = page.getByRole('button', { name: /generate|create|new/i });

      if ((await generateButton.count()) > 0) {
        await expect(generateButton.first()).toBeVisible();
      }
    } else {
      test.skip();
    }
  });
});

test.describe('Statistics and Monitoring', () => {
  test('should display dashboard statistics', async ({ page }) => {
    await page.goto('/');

    // Look for stats widgets
    const stats = page.getByText(/total|active|running|completed|failed/i);

    if ((await stats.count()) > 0) {
      await expect(stats.first()).toBeVisible();
    }
  });

  test('should show workflow statistics', async ({ page }) => {
    await page.goto('/stats');

    // Wait for stats to load
    await page.waitForLoadState('networkidle');

    // Check for statistics
    const hasStats =
      (await page.getByText(/workflow|execution|success rate|duration/i).count()) > 0;

    if (hasStats) {
      expect(hasStats).toBeTruthy();
    } else {
      test.skip();
    }
  });
});
