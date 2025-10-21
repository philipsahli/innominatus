# Makefile Guide

The innominatus project includes a comprehensive Makefile to simplify development, testing, and build workflows.

## Quick Start

```bash
make help          # Show all available commands (colored output)
make deps-check    # Verify all dependencies are installed
make install       # Install all dependencies
make build         # Build everything
make test          # Run all tests
```

## Common Commands

### Development

```bash
make dev           # Start server + web UI in development mode (parallel)
make run           # Run the server only
make build         # Build all components (server, CLI, web UI)
make clean         # Remove build artifacts
```

### Testing

```bash
make test          # Run all local tests (unit + e2e, no K8s)
make test-unit     # Run Go unit tests with race detection
make test-e2e      # Run Go E2E tests (skip K8s tests)
make test-e2e-k8s  # Run full E2E tests including K8s demo
make test-ui       # Run Web UI Playwright tests
make test-all      # Run complete test suite (includes K8s)
make test-ci       # Simulate CI test run
make coverage      # Generate and open HTML coverage report
```

### Code Quality

```bash
make lint          # Lint all code (Go + TypeScript)
make fmt           # Format all code
make vet           # Run go vet
```

### Web UI Specific

```bash
make test-ui       # Run Playwright tests headless
make test-ui-ui    # Run Playwright tests with UI mode
make test-ui-debug # Run Playwright tests in debug mode
```

### Demo Environment

```bash
make demo-time     # Install demo environment (requires K8s)
make demo-status   # Check demo environment status
make demo-nuke     # Remove demo environment
```

### Utilities

```bash
make setup-playwright  # Install Playwright browsers
make clean            # Remove build artifacts
make clean-all        # Deep clean (removes dependencies too)
make version          # Show version information
make deps-check       # Check if all dependencies are installed
make install-hooks    # Install git pre-commit hooks
```

## Environment Variables

The Makefile respects these environment variables:

- `SKIP_DEMO_TESTS=1` - Skip Kubernetes demo tests
- `SKIP_INTEGRATION_TESTS=1` - Skip integration tests
- `CI=true` - Run in CI mode

Example:
```bash
export SKIP_DEMO_TESTS=1
make test-e2e
```

## Features

### Colored Output

All commands provide colored output for better readability:
- 🟢 Green: Success messages and status
- 🟡 Yellow: Warnings and informational messages
- 🔵 Blue: Section headers
- 🔴 Red: Errors (when applicable)

### Self-Documenting

Run `make help` or just `make` to see all available commands with descriptions.

### Parallel Execution

The `make dev` command runs the server and web UI in parallel for faster development.

### Matches CI/CD

The `make test-ci` command simulates the exact CI pipeline for local testing.

### Phony Targets

All targets are properly declared as phony to avoid conflicts with files.

## Tips

1. **Fast Iteration**: Use `make test-unit` during development for quick feedback
2. **Before Commit**: Run `make lint && make test` to ensure code quality
3. **Full Validation**: Run `make test-all` before creating a PR
4. **CI Debugging**: Use `make test-ci` to replicate CI environment locally
5. **Pre-commit Hooks**: Run `make install-hooks` to automatically run tests on commit

## Troubleshooting

**"No rule to make target"**
- Make sure you're in the project root directory
- Check for typos in the target name

**"command not found"**
- Run `make deps-check` to verify dependencies
- Install missing dependencies with `make install`

**Tests failing**
- Check prerequisites: `make deps-check`
- For demo tests, ensure Docker Desktop with K8s is running
- Use `make test-e2e` instead of `make test-e2e-k8s` to skip K8s tests

**Web UI tests failing**
- Install Playwright browsers: `make setup-playwright`
- Ensure web UI dependencies are installed: `cd web-ui && npm install`

## Integration with IDE

Most IDEs can integrate with Makefiles:

**VS Code:**
- Install "Makefile Tools" extension
- Tasks will appear in the command palette

**JetBrains IDEs (GoLand, IntelliJ):**
- Right-click on Makefile → "Run Make"
- Or use the Make tool window

**Vim/Neovim:**
- Use `:make target` command
- Configure with `:set makeprg=make`

## See Also

- [Testing Guide](docs/development/testing.md) - Comprehensive testing documentation
- [CLAUDE.md](CLAUDE.md) - Project overview and quick start
- [CI/CD Workflow](.github/workflows/e2e-tests.yml) - GitHub Actions configuration
