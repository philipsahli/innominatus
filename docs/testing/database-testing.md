# Database Testing with Testcontainers

This document explains how database persistence is handled in Go tests using **testcontainers-go** and **SQLite**.

## Overview

Database tests support two drivers:

1. **PostgreSQL** (default) - Production-like testing with testcontainers
2. **SQLite** (optional) - Fast, zero-dependency testing

Benefits:

- **Zero manual setup**: No need to install PostgreSQL locally
- **Multiple options**: Choose PostgreSQL (accuracy) or SQLite (speed)
- **Isolation**: Each test gets a fresh database
- **Portability**: Tests run consistently across all environments
- **CI/CD ready**: Works seamlessly in GitHub Actions

## How It Works

### Automatic Container Management

When you run a database test:

1. **Container starts**: Testcontainers pulls `postgres:15-alpine` and starts it
2. **Schema initialized**: Tables and migrations are applied automatically
3. **Test runs**: Your test executes against the real PostgreSQL database
4. **Cleanup**: Container is automatically removed when test completes

### Example Test

```go
func TestMyDatabaseFeature(t *testing.T) {
    // SetupTestDatabase handles everything:
    // - Starts PostgreSQL container
    // - Creates database schema
    // - Registers cleanup via t.Cleanup()
    testDB := SetupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database setup failed, test skipped")
    }

    // Use the database
    repo := NewWorkflowRepository(testDB.Database)

    // Your test logic here
    exec, err := repo.CreateWorkflowExecution("my-app", "deploy", 3)
    if err != nil {
        t.Fatalf("Failed to create execution: %v", err)
    }

    // No manual cleanup needed - testcontainers handles it
}
```

## Running Tests

### Choose Your Database Driver

**Option 1: SQLite (fastest, no Docker required)**
```bash
# All tests with SQLite
TEST_DB_DRIVER=sqlite go test ./...

# Or using Make
make test-sqlite
```

**Option 2: PostgreSQL (default, production-like)**
```bash
# All tests with PostgreSQL testcontainers
go test ./...

# Or using Make
make test
```

**Option 3: Both (comprehensive validation)**
```bash
# Run tests against both databases
make test-both
```

### Prerequisites

**For PostgreSQL tests** (default):
- **Docker must be installed and running**:
  - **Linux**: Docker Engine
  - **macOS**: Docker Desktop
  - **Windows**: Docker Desktop (WSL2 backend)

Check Docker is running:
```bash
docker ps
```

**For SQLite tests**:
- No prerequisites! Works out of the box.

### Run All Tests

```bash
# PostgreSQL (default) - Uses testcontainers
go test ./...

# SQLite (faster) - In-memory database
TEST_DB_DRIVER=sqlite go test ./...

# Run with coverage
go test ./... -race -coverprofile=coverage.out

# Run only database tests
go test ./internal/database/... -v
```

### Using Make Commands

```bash
# Run all local tests (PostgreSQL default)
make test

# Run unit tests only
make test-unit

# Run with SQLite (no Docker)
make test-sqlite
make test-sqlite-only

# Run with both databases
make test-both

# Run tests with Docker Compose database (alternative approach)
make test-with-db

# Start standalone test database (for manual testing)
make db-test-up
make db-test-down
```

## Test Patterns

### Pattern 1: Single Test with Fresh Database

Each test gets its own database container:

```go
func TestSomething(t *testing.T) {
    testDB := SetupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database setup failed")
    }

    // Test runs with fresh database
    // Container automatically removed after test
}
```

### Pattern 2: Shared Database for Test Suite

Share one container across all tests in a package using `TestMain`:

```go
var sharedDB *Database

func TestMain(m *testing.M) {
    // Setup shared database
    db, cleanup, err := SetupTestDatabaseShared()
    if err != nil {
        log.Fatalf("Failed to setup test database: %v", err)
    }
    sharedDB = db

    // Run tests
    code := m.Run()

    // Cleanup
    cleanup()
    os.Exit(code)
}

func TestWithSharedDB(t *testing.T) {
    repo := NewWorkflowRepository(sharedDB)

    // Clean data between tests if needed
    if err := sharedDB.TruncateAllTables(); err != nil {
        t.Fatalf("Failed to cleanup: %v", err)
    }

    // Your test logic
}
```

### Pattern 3: Parallel Tests with Unique Data

Use unique identifiers to avoid conflicts:

```go
func TestParallel(t *testing.T) {
    t.Parallel() // Enable parallel execution

    testDB := SetupTestDatabase(t)
    if testDB == nil {
        t.Skip("Database setup failed")
    }

    // Use unique names to avoid conflicts
    appName := fmt.Sprintf("test-app-%d", time.Now().UnixNano())

    repo := NewWorkflowRepository(testDB.Database)
    exec, _ := repo.CreateWorkflowExecution(appName, "deploy", 1)

    // Test continues...
}
```

## CI/CD (GitHub Actions)

Tests run automatically in GitHub Actions with **no configuration needed**:

```yaml
# .github/workflows/test.yml
- name: Run tests (testcontainers will manage database)
  run: go test -v -race -coverprofile=coverage ./...
```

GitHub Actions runners have Docker pre-installed, so testcontainers works out-of-the-box.

### Matrix Testing

Tests run on both Ubuntu and macOS:

- **Ubuntu**: Database tests run (Docker available)
- **macOS**: Database tests skip gracefully (Docker Desktop required, not in CI)

## Development Workflow

### Option 1: Testcontainers (Recommended)

Just run tests - containers are managed automatically:

```bash
go test ./internal/database/... -v
```

### Option 2: Docker Compose (Manual Control)

Start a persistent database for manual testing:

```bash
# Start database
make db-test-up

# Run tests (connects to Docker Compose database)
DB_HOST=localhost DB_PORT=5432 DB_USER=postgres DB_PASSWORD=postgres \
  DB_NAME=idp_orchestrator_test DB_SSLMODE=disable \
  go test ./internal/database/... -v

# Stop database
make db-test-down
```

### Option 3: Local PostgreSQL

Connect to your macOS PostgreSQL installation:

```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=your_password
export DB_NAME=idp_orchestrator_test
export DB_SSLMODE=disable

# Run tests
go test ./internal/database/... -v
```

**Note**: Tests will use testcontainers by default. To force local PostgreSQL, temporarily disable testcontainers by modifying the test helper.

## Troubleshooting

### "Docker not available"

**Symptom**: Tests skip with message like:
```
Database setup failed (Docker not available?): connection refused
```

**Solution**: Start Docker:
```bash
# macOS/Windows
open -a Docker  # or start Docker Desktop

# Linux
sudo systemctl start docker
```

### Slow Test Startup

**Symptom**: First test run takes 30-60 seconds

**Cause**: Pulling `postgres:15-alpine` image (happens once)

**Solution**: Pre-pull the image:
```bash
docker pull postgres:15-alpine
```

### Port Conflicts

**Symptom**: Tests fail with "port already in use"

**Cause**: Previous container not cleaned up

**Solution**: Clean up Docker containers:
```bash
docker ps -a  # List all containers
docker rm -f $(docker ps -aq)  # Remove all containers
```

### macOS Performance

**Symptom**: Tests are slower on macOS than Linux

**Cause**: Docker Desktop on macOS uses a VM

**Solution**:
- Increase Docker Desktop resources (Preferences → Resources)
- Use shared database pattern (`TestMain`) to reuse containers

## Helper Functions Reference

### `SetupTestDatabase(t *testing.T)`

Creates a fresh PostgreSQL container for a single test.

**Returns**: `*TestDatabase` with initialized schema
**Cleanup**: Automatic via `t.Cleanup()`

### `SetupTestDatabaseShared()`

Creates a PostgreSQL container shared across multiple tests.

**Returns**: `*Database, cleanup func(), error`
**Cleanup**: Call `cleanup()` manually in `TestMain`

### `CleanupTestData(t *testing.T)`

Truncates all tables while preserving schema.

**Use case**: Reset database between tests when sharing a container

## Migration to Testcontainers

### Before (Manual Setup)

```go
func setupTestRepo(t *testing.T) *WorkflowRepository {
    db, err := NewDatabase()  // Connects to local PostgreSQL
    if err != nil {
        t.Skipf("Database connection failed: %v", err)
    }

    db.InitSchema()
    return NewWorkflowRepository(db)
}
```

Required manual setup:
- Install PostgreSQL locally
- Set environment variables
- Create database manually

### After (Testcontainers)

```go
func setupTestRepo(t *testing.T) *WorkflowRepository {
    testDB := SetupTestDatabase(t)  // Automatic container
    if testDB == nil {
        t.Skip("Database setup failed")
    }

    return NewWorkflowRepository(testDB.Database)
}
```

No manual setup required - just run `go test`.

## Best Practices

1. **Use `t.Helper()`** in test setup functions for better error messages
2. **Skip gracefully** when Docker unavailable (don't fail the test)
3. **Use unique names** for parallel tests to avoid conflicts
4. **Share containers** for test suites with many tests (use `TestMain`)
5. **Clean up data** between tests if sharing a container
6. **Pre-pull images** in CI to avoid timeout issues

## Performance Tips

### For Local Development

```bash
# Pre-pull image once
docker pull postgres:15-alpine

# Use shared container for package tests
# (implement TestMain pattern)
```

### For CI/CD

```yaml
# Cache Docker images
- name: Cache Docker images
  uses: actions/cache@v4
  with:
    path: /var/lib/docker
    key: ${{ runner.os }}-docker-${{ hashFiles('**/Dockerfile') }}
```

## Additional Resources

- [Testcontainers Go Documentation](https://golang.testcontainers.org/)
- [PostgreSQL Module Documentation](https://golang.testcontainers.org/modules/postgres/)
- [Docker Desktop Downloads](https://www.docker.com/products/docker-desktop/)
- [SQLite Support Guide](../development/sqlite-support.md)

## Questions?

- **Why testcontainers instead of mocks?** Real database catches SQL bugs and migration issues
- **Why PostgreSQL 15 Alpine?** Fast startup, small image size (40MB compressed)
- **Can I use SQLite for tests?** Yes! Set `TEST_DB_DRIVER=sqlite` - fastest option, no Docker required
- **When should I use SQLite vs PostgreSQL?** SQLite for speed/convenience, PostgreSQL for production-like testing
- **Do I need to clean up containers?** No, testcontainers handles cleanup automatically
- **Can I use both databases?** Yes! Run `make test-both` to validate against both

## Database Driver Comparison

| Feature | PostgreSQL | SQLite |
|---------|-----------|--------|
| **Setup** | Docker required | Zero setup |
| **Speed** | ~30s first run, ~1.5s after | ~1.2s always |
| **Accuracy** | Production-like | Good enough for dev |
| **CI/CD** | Full validation | Fast feedback |
| **Recommended for** | Pre-commit, CI/CD | Local dev, quick tests |

**Recommendation**: Use SQLite for rapid iteration, PostgreSQL before committing.

---

*Last updated: 2025-10-22*
