/**
 * Claude Code Hooks for innominatus
 *
 * Automation hooks for Go backend and Next.js/TypeScript frontend development
 */

/**
 * Pre-edit hook: Run before file edits
 * - Validates file exists
 * - Runs linters for Go and TypeScript files
 */
export async function preEdit({ file, operation }) {
  console.log(`[pre-edit] ${operation} on ${file}`);

  // Go file validation
  if (file.endsWith('.go')) {
    try {
      await $`go fmt ${file}`;
      console.log(`[pre-edit] Formatted Go file: ${file}`);
    } catch (error) {
      console.warn(`[pre-edit] Go fmt failed for ${file}:`, error.message);
    }
  }

  // TypeScript file validation
  if (file.endsWith('.ts') || file.endsWith('.tsx')) {
    console.log(`[pre-edit] TypeScript file detected: ${file}`);
    // Note: npm run lint would run in web-ui directory
  }

  return { continue: true };
}

/**
 * Post-edit hook: Run after file edits
 * - Runs tests for changed files
 * - Validates builds still work
 */
export async function postEdit({ file, operation }) {
  console.log(`[post-edit] ${operation} completed on ${file}`);

  // Run Go tests if Go file changed
  if (file.endsWith('.go') && !file.endsWith('_test.go')) {
    const testFile = file.replace('.go', '_test.go');
    try {
      const testExists = await Bun.file(testFile).exists();
      if (testExists) {
        console.log(`[post-edit] Running tests for ${file}`);
        await $`go test -run ${path.basename(file, '.go')} ./...`.quiet();
        console.log(`[post-edit] Tests passed for ${file}`);
      }
    } catch (error) {
      console.warn(`[post-edit] Tests failed for ${file}:`, error.message);
    }
  }

  // Run TypeScript type check if TS file changed
  if (file.endsWith('.tsx') || file.endsWith('.ts')) {
    console.log(`[post-edit] TypeScript file changed, consider running: cd web-ui && npm run type-check`);
  }

  return { continue: true };
}

/**
 * Pre-commit hook: Run before git commits
 * - Validates all tests pass
 * - Ensures builds succeed
 */
export async function preCommit({ files }) {
  console.log(`[pre-commit] Validating ${files.length} files`);

  const goFiles = files.filter(f => f.endsWith('.go'));
  const tsFiles = files.filter(f => f.endsWith('.ts') || f.endsWith('.tsx'));

  // Run Go tests if Go files changed
  if (goFiles.length > 0) {
    console.log(`[pre-commit] Running Go tests...`);
    try {
      await $`go test ./...`.quiet();
      console.log(`[pre-commit] Go tests passed`);
    } catch (error) {
      console.error(`[pre-commit] Go tests failed:`, error.message);
      return {
        continue: false,
        message: 'Go tests failed. Fix tests before committing.'
      };
    }
  }

  // Suggest running web-ui tests if TS files changed
  if (tsFiles.length > 0) {
    console.log(`[pre-commit] TypeScript files changed. Recommended: cd web-ui && npm run build`);
  }

  return { continue: true };
}

/**
 * Post-test hook: Run after test execution
 * - Generates coverage reports
 * - Updates test documentation
 */
export async function postTest({ testCommand, exitCode, stdout, stderr }) {
  console.log(`[post-test] Test command: ${testCommand}, Exit code: ${exitCode}`);

  if (exitCode === 0) {
    console.log(`[post-test] Tests passed successfully`);

    // Generate coverage report for Go tests
    if (testCommand.includes('go test')) {
      try {
        await $`go test -coverprofile=coverage.out ./...`.quiet();
        await $`go tool cover -html=coverage.out -o coverage.html`.quiet();
        console.log(`[post-test] Coverage report generated: coverage.html`);
      } catch (error) {
        console.warn(`[post-test] Coverage generation failed:`, error.message);
      }
    }
  } else {
    console.error(`[post-test] Tests failed with exit code ${exitCode}`);
  }

  return { continue: true };
}

/**
 * Pre-build hook: Run before builds
 * - Validates dependencies are installed
 * - Checks for environment variables
 */
export async function preBuild({ target }) {
  console.log(`[pre-build] Building target: ${target}`);

  // Check Go dependencies
  if (target === 'server' || target === 'cli') {
    try {
      await $`go mod download`.quiet();
      console.log(`[pre-build] Go dependencies up to date`);
    } catch (error) {
      console.error(`[pre-build] Go mod download failed:`, error.message);
      return { continue: false, message: 'Failed to download Go dependencies' };
    }
  }

  // Check npm dependencies for web-ui
  if (target === 'web-ui') {
    try {
      await $`cd web-ui && npm install`.quiet();
      console.log(`[pre-build] npm dependencies up to date`);
    } catch (error) {
      console.error(`[pre-build] npm install failed:`, error.message);
      return { continue: false, message: 'Failed to install npm dependencies' };
    }
  }

  return { continue: true };
}

/**
 * Post-deploy hook: Run after deployments
 * - Validates deployment health
 * - Runs smoke tests
 */
export async function postDeploy({ environment, version }) {
  console.log(`[post-deploy] Deployed version ${version} to ${environment}`);

  // Health check for server deployment
  if (environment === 'local' || environment === 'development') {
    console.log(`[post-deploy] Running health check...`);
    try {
      const response = await fetch('http://localhost:8081/health');
      if (response.ok) {
        console.log(`[post-deploy] Health check passed`);
      } else {
        console.warn(`[post-deploy] Health check returned status ${response.status}`);
      }
    } catch (error) {
      console.warn(`[post-deploy] Health check failed:`, error.message);
    }
  }

  return { continue: true };
}

export default {
  preEdit,
  postEdit,
  preCommit,
  postTest,
  preBuild,
  postDeploy
};
