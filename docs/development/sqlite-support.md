# SQLite Support for Development

Innominatus supports **SQLite** as an alternative to PostgreSQL for development and testing. This allows developers to run the application with **zero external dependencies** - no Docker, no PostgreSQL installation required.

## Quick Start

### For Testing

```bash
# Run tests with SQLite (fastest)
TEST_DB_DRIVER=sqlite go test ./...

# Or using Make
make test-sqlite
```

### For Development Server

```bash
# Run server with SQLite
DB_DRIVER=sqlite ./innominatus

# Or specify custom database path
DB_DRIVER=sqlite DB_PATH=./my-data.db ./innominatus
```

## When to Use SQLite vs PostgreSQL

### Use SQLite For:

✅ **Quick local development** - No setup required
✅ **Fast test execution** - In-memory database
✅ **CI environments** - Faster, no Docker overhead
✅ **Onboarding** - New developers can start immediately
✅ **Demos** - Self-contained, portable database file

### Use PostgreSQL For:

✅ **Production** - Production database engine
✅ **Production-like testing** - Catch PostgreSQL-specific issues
✅ **Performance testing** - Real database performance characteristics
✅ **Transaction testing** - PostgreSQL-specific isolation levels
✅ **Schema compatibility** - Validate migrations work on PostgreSQL

## Configuration

### Test Configuration

Control test database driver with environment variable:

```bash
# PostgreSQL (default) - Uses testcontainers
go test ./...

# SQLite - In-memory, fastest
TEST_DB_DRIVER=sqlite go test ./...
```

### Runtime Configuration

Control server database driver with environment variables:

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `DB_DRIVER` | `postgres`, `sqlite` | `postgres` | Database driver |
| `DB_PATH` | File path or `:memory:` | `./data/innominatus.db` | SQLite database path (when `DB_DRIVER=sqlite`) |

**PostgreSQL environment variables** (when `DB_DRIVER=postgres`):
- `DB_HOST` - PostgreSQL host (default: `localhost`)
- `DB_PORT` - PostgreSQL port (default: `5432`)
- `DB_USER` - Database user (default: `postgres`)
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (default: `idp_orchestrator`)
- `DB_SSLMODE` - SSL mode (default: `disable`)

## Usage Examples

### Development Server

**Option 1: In-memory SQLite (fastest, no persistence)**
```bash
DB_DRIVER=sqlite DB_PATH=:memory: ./innominatus
```

**Option 2: File-based SQLite (persists data)**
```bash
DB_DRIVER=sqlite DB_PATH=./dev-data.db ./innominatus
```

**Option 3: PostgreSQL (production-like)**
```bash
# Uses default PostgreSQL settings or testcontainers
./innominatus
```

### Testing

**SQLite tests (no Docker)**
```bash
# All tests
make test-sqlite

# Database tests only
make test-sqlite-only

# Specific test
TEST_DB_DRIVER=sqlite go test ./internal/database -run TestWorkflowRepository
```

**PostgreSQL tests (with Docker)**
```bash
# All tests with testcontainers
make test

# Database tests only
make test-db-only
```

**Both databases (validation)**
```bash
# Run database tests against both PostgreSQL and SQLite
make test-both
```

### CI/CD

**Fast CI with SQLite**
```yaml
# .github/workflows/test.yml
- name: Run fast tests with SQLite
  env:
    TEST_DB_DRIVER: sqlite
  run: go test ./... -v -short
```

**Full CI with PostgreSQL**
```yaml
# Default behavior - testcontainers manages PostgreSQL
- name: Run comprehensive tests
  run: go test ./... -v
```

## Compatibility

### SQL Compatibility

Both databases support:
- ✅ `RETURNING` clause (SQLite 3.35+)
- ✅ JSON columns (`TEXT` in SQLite)
- ✅ Foreign keys (enabled by default in our SQLite config)
- ✅ Transactions
- ✅ Concurrent reads

### Differences

| Feature | PostgreSQL | SQLite |
|---------|-----------|--------|
| **Concurrency** | Multiple writers | Single writer |
| **Data types** | Rich (JSONB, UUID, arrays) | Limited (uses TEXT/BLOB) |
| **Performance** | Better for large datasets | Better for small datasets |
| **Deployment** | Separate service | Embedded (no setup) |
| **Isolation** | Full ACID compliance | ACID with limitations |

### Known Limitations

**SQLite limitations** (acceptable for dev/test):
1. **Single writer** - Writes are serialized (fine for dev, not for high-traffic prod)
2. **No JSONB** - Uses TEXT with JSON functions (slower for JSON queries)
3. **Limited ALTER TABLE** - Some schema changes require table recreation
4. **No UUID type** - Uses TEXT representation

**Mitigation**:
- Use PostgreSQL for production
- Use SQLite only for development and testing
- Tests pass on both drivers to ensure compatibility

## Migration Compatibility

All migrations work on both PostgreSQL and SQLite:

```sql
-- migrations/001_initial_schema.sql
-- Uses compatible SQL syntax
CREATE TABLE IF NOT EXISTS workflow_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,  -- Works in SQLite
    -- or
    id SERIAL PRIMARY KEY,                  -- Works in PostgreSQL
    ...
);
```

**Auto-detection**:
- `SERIAL` → SQLite uses `INTEGER PRIMARY KEY AUTOINCREMENT`
- `JSONB` → SQLite uses `TEXT` with JSON functions
- `NOW()` → SQLite uses `datetime('now')`

## Performance

### Benchmark (100 test executions)

| Database | Test Duration | Startup Time | Total Time |
|----------|--------------|--------------|------------|
| SQLite (in-memory) | 1.2s | 0.01s | **1.21s** |
| PostgreSQL (testcontainers) | 1.5s | 30s* | **31.5s** |
| PostgreSQL (running) | 1.5s | 0s | **1.5s** |

*First run pulls Docker image

**Recommendation**: Use SQLite for rapid local testing, PostgreSQL for CI/CD.

## Troubleshooting

### "no such table" Error

**Cause**: Schema not initialized

**Solution**:
```bash
# Delete SQLite file and restart
rm ./data/innominatus.db
DB_DRIVER=sqlite ./innominatus
```

### "database is locked" Error

**Cause**: Another process has the SQLite file open

**Solution**:
```bash
# Check for processes using the database
lsof ./data/innominatus.db

# Or use in-memory database
DB_DRIVER=sqlite DB_PATH=:memory: ./innominatus
```

### Tests Fail on SQLite but Pass on PostgreSQL

**Cause**: Using PostgreSQL-specific features

**Solution**:
1. Check SQL queries for PostgreSQL-specific syntax
2. Use database-agnostic abstractions
3. Add conditional logic for driver-specific features

**Example**:
```go
// Bad - PostgreSQL specific
db.Exec("SELECT current_database()")

// Good - Works on both
var dbName string
if driver == "postgres" {
    db.QueryRow("SELECT current_database()").Scan(&dbName)
} else {
    dbName = "sqlite_db"
}
```

## Best Practices

### For Development

1. **Use SQLite by default** for quick local dev
2. **Test with PostgreSQL** before pushing to ensure compatibility
3. **Use file-based SQLite** if you need to inspect the database
4. **Use in-memory SQLite** for fastest iteration

### For Testing

1. **Use SQLite in CI** for fast feedback
2. **Use PostgreSQL in nightly tests** for comprehensive validation
3. **Test both** before releases with `make test-both`

### For Production

1. **Always use PostgreSQL** in production
2. **Never use SQLite** in production (not designed for multi-user scenarios)
3. **Validate migrations** on PostgreSQL before deploying

## Migration from PostgreSQL-only

If you have existing code assuming PostgreSQL:

**Before**:
```go
db, err := database.NewDatabase() // PostgreSQL only
```

**After**:
```go
db, err := database.NewDatabaseAuto() // Auto-detects based on DB_DRIVER
// or
db, err := database.NewSQLiteDatabase(":memory:") // Explicit SQLite
```

## FAQ

**Q: Is SQLite production-ready?**
A: SQLite is incredibly robust, but innominatus is designed for multi-user orchestration. Use PostgreSQL in production.

**Q: Can I switch between SQLite and PostgreSQL?**
A: No direct migration tool. Export data from one, import to the other if needed.

**Q: Does this add complexity?**
A: Minimal - we use standard SQL that works on both. Tests validate compatibility.

**Q: Why not just use PostgreSQL everywhere?**
A: Developer experience - SQLite requires zero setup, making onboarding instant.

**Q: What about other databases (MySQL, MSSQL)?**
A: Not currently supported. PostgreSQL is production, SQLite is dev convenience.

## Additional Resources

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [SQLite vs PostgreSQL](https://www.sqlite.org/whentouse.html)
- [Go SQLite Driver](https://github.com/mattn/go-sqlite3)
- [Database Testing Guide](./database-testing.md)

---

*Last updated: 2025-10-22*
