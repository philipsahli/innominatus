/**
 * E2E tests for embedded web UI served by Go server
 *
 * These tests verify that the Go server correctly serves the embedded
 * Next.js web UI files (HTML, JS, CSS, etc.)
 *
 * To run against Go server:
 * 1. Start Go server: go run ./cmd/server/main.go --disable-db --port 8082
 * 2. Run tests: BASE_URL=http://localhost:8082 npx playwright test go-server.spec.ts --config=playwright.config.goserver.ts
 */

import { test, expect } from '@playwright/test';

test.describe('Go Server - Embedded Web UI', () => {
  test('should serve HTML index page', async ({ page }) => {
    await page.goto('/');

    // Check that HTML loads
    await expect(page).toHaveTitle(/innominatus/);

    // Check for essential HTML elements
    const html = await page.content();
    expect(html).toContain('<!DOCTYPE html>');
    expect(html).toContain('_next/static');
  });

  test('should load JavaScript chunks', async ({ page, request }) => {
    // Test webpack chunk
    const webpackResponse = await request.get('/_next/static/chunks/webpack-5289856b5489d9f4.js');
    expect(webpackResponse.status()).toBe(200);
    expect(webpackResponse.headers()['content-type']).toContain('javascript');

    // Test main app chunk
    const mainAppResponse = await request.get('/_next/static/chunks/main-app-34e5f21fab611491.js');
    expect(mainAppResponse.status()).toBe(200);
    expect(mainAppResponse.headers()['content-type']).toContain('javascript');
  });

  test('should load CSS files', async ({ request }) => {
    const cssResponse = await request.get('/_next/static/css/7e7d96b1e6991756.css');
    expect(cssResponse.status()).toBe(200);
    expect(cssResponse.headers()['content-type']).toContain('text/css');
  });

  test('should serve favicon', async ({ request }) => {
    const faviconResponse = await request.get('/favicon.ico');
    expect(faviconResponse.status()).toBe(200);
    expect(faviconResponse.headers()['content-type']).toContain('image');
  });

  test('should serve Swagger YAML', async ({ request }) => {
    const swaggerResponse = await request.get('/swagger-user.yaml');
    expect(swaggerResponse.status()).toBe(200);

    const yaml = await swaggerResponse.text();
    expect(yaml).toContain('openapi:');
    expect(yaml).toContain('innominatus User API');
  });

  test('should serve Swagger UI', async ({ page }) => {
    await page.goto('/swagger');

    // Check that Swagger UI loads
    await expect(page.locator('h1')).toContainText('innominatus');

    // Check for Swagger UI elements
    const html = await page.content();
    expect(html).toContain('swagger-ui');
  });

  test('should serve health endpoint', async ({ request }) => {
    const healthResponse = await request.get('/health');
    expect(healthResponse.status()).toBe(200);

    const health = await healthResponse.json();
    expect(health.status).toBe('healthy');
  });

  test('should handle SPA routing', async ({ page }) => {
    // Navigate to a route that doesn't have a physical file
    await page.goto('/dashboard');

    // Should still serve the index.html and let React Router handle it
    await expect(page).toHaveTitle(/innominatus/);
  });

  test('should serve fonts', async ({ request }) => {
    const fontResponse = await request.get('/_next/static/media/e4af272ccee01ff0-s.p.woff2');
    expect(fontResponse.status()).toBe(200);
    expect(fontResponse.headers()['content-type']).toContain('font');
  });
});
