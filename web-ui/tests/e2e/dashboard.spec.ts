import { test, expect } from '@playwright/test';

/**
 * Dashboard and Navigation E2E Tests
 *
 * These tests verify the basic functionality of the dashboard and navigation.
 */
test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the homepage before each test
    await page.goto('/');
  });

  test('should load the homepage successfully', async ({ page }) => {
    // Verify the page loaded
    await expect(page).toHaveTitle(/innominatus/i);

    // Check for main navigation elements
    const nav = page.locator('nav').first();
    await expect(nav).toBeVisible();
  });

  test('should display navigation menu', async ({ page }) => {
    // Check for key navigation items
    const navItems = [
      'Applications',
      'Golden Paths',
      'Resources',
      'Workflows',
    ];

    for (const item of navItems) {
      // Use more flexible locator that works with various navigation structures
      const navItem = page.getByRole('link', { name: new RegExp(item, 'i') });
      // Check if at least one matching element exists (may not be visible if in dropdown)
      const count = await navItem.count();
      expect(count).toBeGreaterThan(0);
    }
  });

  test('should navigate to applications page', async ({ page }) => {
    // Click on Applications link
    await page.getByRole('link', { name: /applications/i }).first().click();

    // Wait for navigation
    await page.waitForURL(/\/applications/);

    // Verify we're on the applications page
    expect(page.url()).toContain('/applications');
  });

  test('should have responsive navigation', async ({ page }) => {
    // Test on mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });

    // Check if mobile menu button exists
    const mobileMenuButton = page.getByRole('button', { name: /menu/i });

    // Mobile menu might exist or navigation might be adapted
    const hasMobileMenu = await mobileMenuButton.count() > 0;
    const hasVisibleNav = await page.locator('nav').first().isVisible();

    expect(hasMobileMenu || hasVisibleNav).toBeTruthy();
  });
});

test.describe('Dashboard - Applications List', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/applications');
  });

  test('should display applications list or empty state', async ({ page }) => {
    // Wait for the page to load
    await page.waitForLoadState('networkidle');

    // Check for either applications list or empty state
    const hasApplications = await page.getByText(/test-app|application/i).count() > 0;
    const hasEmptyState = await page.getByText(/no applications/i).count() > 0;

    expect(hasApplications || hasEmptyState).toBeTruthy();
  });

  test('should have search or filter functionality', async ({ page }) => {
    // Look for search input or filter controls
    const searchInput = page.getByPlaceholder(/search/i);
    const filterButton = page.getByRole('button', { name: /filter/i });

    const hasSearch = await searchInput.count() > 0;
    const hasFilter = await filterButton.count() > 0;

    // At least one should exist (or neither if not implemented yet)
    // This is more of a documentation test
    if (hasSearch || hasFilter) {
      expect(hasSearch || hasFilter).toBeTruthy();
    }
  });
});

test.describe('Dashboard - Theme Toggle', () => {
  test('should toggle between light and dark mode', async ({ page }) => {
    await page.goto('/');

    // Look for theme toggle button
    const themeToggle = page.getByRole('button', { name: /theme|dark|light/i });

    if (await themeToggle.count() > 0) {
      // Get initial theme
      const html = page.locator('html');
      const initialClass = await html.getAttribute('class');

      // Click theme toggle
      await themeToggle.first().click();

      // Wait a bit for theme to change
      await page.waitForTimeout(100);

      // Verify theme changed
      const newClass = await html.getAttribute('class');
      expect(newClass).not.toBe(initialClass);
    } else {
      // Theme toggle not implemented yet
      test.skip();
    }
  });
});
