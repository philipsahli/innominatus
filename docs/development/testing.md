# Testing Guide

## Overview

This document describes the testing infrastructure for the innominatus project, including unit tests, integration tests, and end-to-end (E2E) tests.

## Test Structure

```
innominatus/
├── tests/
│   └── e2e/                      # Go E2E tests
│       ├── demo_test.go          # Demo environment lifecycle tests
│       └── deployment_test.go    # Deployment workflow tests
├── internal/
│   └── */                        # Unit tests alongside source
│       └── *_test.go
├── web-ui/
│   ├── tests/
│   │   └── e2e/                  # Playwright E2E tests
│   │       ├── dashboard.spec.ts
│   │       ├── deploy.spec.ts
│   │       ├── resources.spec.ts
│   │       └── admin.spec.ts
│   └── playwright.config.ts
└── .github/
    └── workflows/
        └── e2e-tests.yml         # CI/CD pipeline
```

## Quick Start (Makefile)

The easiest way to run tests locally is using the Makefile:

```bash
make help           # Show all available commands
make test           # Run all local tests (unit + e2e, no K8s)
make test-unit      # Run Go unit tests only
make test-e2e       # Run Go E2E tests (skip K8s)
make test-ui        # Run Web UI Playwright tests
make test-all       # Run complete test suite (including K8s)
make coverage       # Generate coverage report with HTML
```

## Go Tests

### Unit Tests

Unit tests are located alongside their source files with the `_test.go` suffix.

**Run all unit tests:**
```bash
make test-unit
# Or directly with go:
go test ./... -v
```

**Run with coverage:**
```bash
go test ./... -v -race -coverprofile=coverage.out
go tool cover -html=coverage.out
```

**Run short tests only (skip integration):**
```bash
go test ./... -v -short
```

### E2E Tests

E2E tests are located in `tests/e2e/` and cover:

1. **Demo Environment Tests** (`demo_test.go`)
   - Demo installation (`demo-time`)
   - Status checking (`demo-status`)
   - Service health verification
   - Cleanup (`demo-nuke`)
   - Idempotent installation
   - Component filtering

2. **Deployment Workflow Tests** (`deployment_test.go`)
   - Application deployment via golden paths
   - Validation and analysis
   - Resource management
   - Application updates
   - Application deletion

**Run E2E tests:**
```bash
# Using Makefile (recommended)
make test-e2e           # Skip Kubernetes tests
make test-e2e-k8s       # Include Kubernetes tests

# Or directly with go:
# All E2E tests
go test ./tests/e2e -v

# Skip tests requiring Kubernetes
export SKIP_DEMO_TESTS=1
go test ./tests/e2e -v

# Skip integration tests
export SKIP_INTEGRATION_TESTS=1
go test ./tests/e2e -v

# Run specific test
go test ./tests/e2e -v -run TestDemoEnvironmentLifecycle
```

**Prerequisites for E2E tests:**
- Docker Desktop with Kubernetes enabled (for demo tests)
- kubectl and helm in PATH (for demo tests)
- Running innominatus server at http://localhost:8081 (for deployment tests)

## Web UI Tests (Playwright)

### Setup

**Install dependencies:**
```bash
cd web-ui
npm install
```

**Install Playwright browsers:**
```bash
npx playwright install
```

### Running Tests

**Run all tests (headless):**
```bash
make test-ui
# Or directly with npm:
cd web-ui && npm run test:e2e
```

**Run with UI mode:**
```bash
make test-ui-ui
# Or: cd web-ui && npm run test:e2e:ui
```

**Run with debug mode:**
```bash
make test-ui-debug
# Or: cd web-ui && npm run test:e2e:debug
```

**Run specific test file:**
```bash
npx playwright test tests/e2e/dashboard.spec.ts
```

**Run tests in specific browser:**
```bash
npx playwright test --project=chromium
```

**View test report:**
```bash
npm run test:e2e:report
```

### Test Suites

1. **Dashboard Tests** (`dashboard.spec.ts`)
   - Homepage loading
   - Navigation menu
   - Theme toggle
   - Responsive design

2. **Deployment Tests** (`deploy.spec.ts`)
   - Deployment form
   - Golden paths
   - Application management
   - Workflow execution logs
   - Delete confirmation

3. **Resources Tests** (`resources.spec.ts`)
   - Resource listing
   - Resource filtering
   - Resource details
   - Health status
   - Dependency graphs
   - Provider management

4. **Admin Tests** (`admin.spec.ts`)
   - Team management
   - User management
   - Settings
   - API key generation
   - Statistics

### Configuration

Playwright configuration is in `web-ui/playwright.config.ts`:
- Base URL: `http://localhost:3000` (configurable via `BASE_URL` env var)
- Browsers: Chromium, Firefox, WebKit
- Mobile viewports: Pixel 5, iPhone 12
- Screenshots on failure
- Video recording on first retry
- Trace collection on first retry

## CI/CD Pipeline

### GitHub Actions

The E2E test pipeline is defined in `.github/workflows/e2e-tests.yml` and includes:

1. **Go E2E Tests** (`test-go-e2e`)
   - Runs on: push to main/develop, pull requests
   - Executes: Unit tests, E2E tests (no K8s)
   - Coverage: Uploads to Codecov

2. **Web UI E2E Tests** (`test-web-ui-e2e`)
   - Runs on: push to main/develop, pull requests
   - Executes: Playwright tests with Chromium
   - Artifacts: Playwright HTML report

3. **Integration Tests (K8s)** (`test-integration-kubernetes`)
   - Runs on: Manual trigger or `[k8s]` in commit message
   - Setup: Creates kind cluster
   - Executes: Demo environment tests with Kubernetes

4. **Coverage Report** (`coverage-report`)
   - Generates: Coverage summary
   - Comments: Coverage percentage on PRs

### Running CI Locally

**Easiest way (using Makefile):**
```bash
make test-ci            # Simulates complete CI run
```

**Manual steps:**
```bash
# Go tests
go test ./... -v -race -coverprofile=coverage.out -short
export SKIP_DEMO_TESTS=1
export SKIP_INTEGRATION_TESTS=1
go test ./tests/e2e -v

# Web UI tests
cd web-ui
npm ci
npx playwright install --with-deps chromium
npm run build
CI=true npm run test:e2e
```

## Test Coverage Goals

### Current Coverage Targets

- **Go Packages:**
  - `internal/server`: 80%+
  - `internal/database`: 80%+
  - `internal/workflow`: 80%+
  - `internal/cli`: 70%+
  - Overall: 60%+

- **Web UI:**
  - Critical user paths: 100%
  - Component tests: (to be added)

### Checking Coverage

**Go:**
```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

**View coverage by package:**
```bash
go tool cover -func=coverage.out | grep -E "^github.com/.*total:"
```

## Writing Tests

### Go Test Guidelines

**Use table-driven tests:**
```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("MyFunction() error = %v, wantErr %v", err, tt.wantErr)
            }
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Skip tests conditionally:**
```go
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

**Use testify for assertions:**
```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

assert.Equal(t, expected, actual)
require.NoError(t, err)  // Fail immediately
```

### Playwright Test Guidelines

**Use page object pattern:**
```typescript
test('should navigate to page', async ({ page }) => {
    await page.goto('/applications');
    await page.getByRole('button', { name: /deploy/i }).click();
    await expect(page).toHaveURL(/\/deploy/);
});
```

**Handle async operations:**
```typescript
await page.waitForLoadState('networkidle');
await page.waitForURL(/\/applications/);
```

**Skip tests gracefully:**
```typescript
if (await element.count() === 0) {
    test.skip();
}
```

## Troubleshooting

### Go Tests

**Tests hanging:**
- Check for missing context cancellation
- Use `-timeout` flag: `go test ./... -timeout 5m`

**Race condition detected:**
- Fix synchronization issues
- Don't skip: `go test ./... -race`

**Import cycle:**
- Refactor packages to break cycles
- Use interfaces for decoupling

### Playwright Tests

**Element not found:**
- Use `waitForSelector()` or `waitForLoadState()`
- Check if element is in shadow DOM
- Verify selector is correct

**Tests flaky:**
- Add explicit waits
- Avoid hard-coded timeouts
- Use `waitForLoadState('networkidle')`

**Browser not installed:**
```bash
npx playwright install chromium
```

## Best Practices

### General

1. **Test naming:** Use descriptive names that explain what is being tested
2. **Isolation:** Tests should be independent and not rely on execution order
3. **Cleanup:** Always clean up resources (use `t.Cleanup()` or `defer`)
4. **Speed:** Keep unit tests fast (< 1s), integration tests reasonable (< 30s)
5. **Assertions:** Use meaningful assertion messages

### E2E Tests

1. **Flakiness:** Avoid flaky tests - add proper waits and retries
2. **Test data:** Use fixtures in `testdata/` directory
3. **Environment:** Make tests work in CI and locally
4. **Coverage:** Focus on critical user journeys
5. **Documentation:** Document test prerequisites and setup

### CI/CD

1. **Fast feedback:** Run unit tests before E2E tests
2. **Parallelization:** Run independent test suites in parallel
3. **Artifacts:** Save logs, screenshots, and reports
4. **Notifications:** Alert on test failures
5. **Coverage:** Track coverage trends over time

## Future Improvements

- [ ] Add component tests for Web UI (React Testing Library)
- [ ] Add performance tests (load testing)
- [ ] Add API contract tests
- [ ] Increase Go test coverage to 70%+
- [ ] Add mutation testing
- [ ] Add visual regression tests
- [ ] Integrate with test reporting tools (Allure, TestRail)
- [ ] Add chaos engineering tests

## References

- Go Testing: https://go.dev/doc/tutorial/add-a-test
- Testify: https://github.com/stretchr/testify
- Playwright: https://playwright.dev/
- Testing Best Practices: https://github.com/goldbergyoni/javascript-testing-best-practices
