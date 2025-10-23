# Quick Start: SQLite Development (Zero Setup)

Get started with innominatus in **under 2 minutes** with SQLite - no Docker, no PostgreSQL installation required.

## 🚀 5-Minute Quick Start

### 1. Build the Server

```bash
# Clone the repository
git clone https://github.com/philipsahli/innominatus.git
cd innominatus

# Install dependencies
make install

# Build the server
make build
```

### 2. Add SQLite Dependency

```bash
# Add the SQLite driver
./scripts/add-sqlite-dependency.sh

# Or manually
go get github.com/mattn/go-sqlite3@latest
go mod tidy
```

### 3. Run with SQLite

```bash
# Start the server with SQLite
DB_DRIVER=sqlite ./innominatus
```

**That's it!** Your server is now running at:
- Web UI: http://localhost:8081
- API: http://localhost:8081/api
- Health: http://localhost:8081/health

### 4. Test It

```bash
# Create a test workflow
curl -X POST http://localhost:8081/api/workflows/test-app/test-workflow/execute \
  -H "Content-Type: application/json" \
  -d '{
    "steps": [
      {
        "name": "hello",
        "type": "shell",
        "command": "echo \"Hello from SQLite!\""
      }
    ]
  }'
```

## 📋 Configuration Options

### In-Memory Database (Fastest)

**No persistence** - data lost when server restarts:

```bash
DB_DRIVER=sqlite DB_PATH=:memory: ./innominatus
```

**Use for:**
- Quick testing
- Demos
- Development experiments

### File-Based Database (Default)

**Persistent** - data saved to file:

```bash
# Default location: ./data/innominatus.db
DB_DRIVER=sqlite ./innominatus

# Custom location
DB_DRIVER=sqlite DB_PATH=/tmp/my-innominatus.db ./innominatus
```

**Use for:**
- Local development
- Inspecting database contents
- Sharing state between restarts

### Using .env File

Create `.env` in project root:

```bash
# .env file for SQLite development
DB_DRIVER=sqlite
DB_PATH=./data/innominatus.db
PORT=8081
```

Then run:

```bash
# Load .env automatically (if using direnv or similar)
./innominatus

# Or export manually
export $(cat .env | xargs) && ./innominatus
```

## 🔧 Development Workflow

### Daily Development

```bash
# 1. Start server with file-based SQLite
DB_DRIVER=sqlite ./innominatus &

# 2. Make your code changes

# 3. Run tests with SQLite (fastest)
TEST_DB_DRIVER=sqlite make test

# 4. Restart server
pkill innominatus
DB_DRIVER=sqlite ./innominatus &
```

### Before Committing

```bash
# Test with both databases to ensure compatibility
make test-both

# Or test with PostgreSQL only
make test
```

### Reset Database

```bash
# Stop server
pkill innominatus

# Delete SQLite file
rm ./data/innominatus.db

# Restart server (schema auto-created)
DB_DRIVER=sqlite ./innominatus
```

## 🎯 Common Development Scenarios

### Scenario 1: Quick Feature Development

```bash
# Use in-memory SQLite for fastest iteration
DB_DRIVER=sqlite DB_PATH=:memory: ./innominatus

# Code, test, repeat - no cleanup needed!
```

### Scenario 2: Inspecting Database

```bash
# Use file-based SQLite
DB_DRIVER=sqlite DB_PATH=./dev.db ./innominatus

# In another terminal, inspect database
sqlite3 ./dev.db
sqlite> .tables
sqlite> SELECT * FROM workflow_executions;
sqlite> .quit
```

### Scenario 3: Sharing Development State

```bash
# Create database with test data
DB_DRIVER=sqlite DB_PATH=./demo.db ./innominatus
# ... create some workflows ...
pkill innominatus

# Share the file
tar czf demo-db.tar.gz ./demo.db

# Teammate can load it
tar xzf demo-db.tar.gz
DB_DRIVER=sqlite DB_PATH=./demo.db ./innominatus
```

### Scenario 4: Testing Migrations

```bash
# Start with SQLite
DB_DRIVER=sqlite DB_PATH=./migration-test.db ./innominatus

# Migrations run automatically on startup
# Check logs for migration execution

# Inspect migrated schema
sqlite3 ./migration-test.db ".schema"
```

## 🛠️ Tools & Tips

### SQLite Browser

Install a GUI tool for easier database inspection:

```bash
# macOS
brew install --cask db-browser-for-sqlite

# Ubuntu/Debian
sudo apt install sqlitebrowser

# Or use web-based
# https://sqliteviewer.app/
```

### SQLite CLI Tips

```bash
# Open database
sqlite3 ./data/innominatus.db

# Useful commands
.tables              # List all tables
.schema table_name   # Show table schema
.mode column         # Pretty column output
.headers on          # Show column names
.quit                # Exit

# Query examples
SELECT * FROM workflow_executions ORDER BY created_at DESC LIMIT 10;
SELECT COUNT(*) FROM resource_instances;
```

### Backup & Restore

```bash
# Backup
cp ./data/innominatus.db ./backups/innominatus-$(date +%Y%m%d).db

# Restore
cp ./backups/innominatus-20251022.db ./data/innominatus.db

# Export to SQL
sqlite3 ./data/innominatus.db .dump > backup.sql

# Import from SQL
sqlite3 ./data/innominatus-new.db < backup.sql
```

## 📊 SQLite vs PostgreSQL

### When to Use SQLite

✅ **Local development** - Fastest setup
✅ **Quick prototyping** - No configuration needed
✅ **Onboarding** - New developers start immediately
✅ **Unit testing** - Fast, isolated tests
✅ **Demos** - Self-contained, portable

### When to Switch to PostgreSQL

⚠️ **Before production** - PostgreSQL required for production
⚠️ **Performance testing** - Test with production database
⚠️ **Multi-user testing** - SQLite serializes writes
⚠️ **Advanced features** - PostgreSQL-specific features
⚠️ **Final validation** - Always test with PostgreSQL before deploy

### Performance Comparison

| Operation | SQLite | PostgreSQL |
|-----------|--------|------------|
| **Startup** | < 1 second | ~5 seconds (local) |
| **Workflow execution** | Fast | Fast |
| **Test suite** | ~2 seconds | ~5 seconds (testcontainers) |
| **Database size** | ~1 MB | ~5 MB |

## 🚨 Troubleshooting

### Server Won't Start

**Error: "database is locked"**

```bash
# Another process is using the database
lsof ./data/innominatus.db

# Kill the process or use a different path
DB_DRIVER=sqlite DB_PATH=./data/innominatus-2.db ./innominatus
```

**Error: "no such file or directory"**

```bash
# Database directory doesn't exist
mkdir -p ./data
DB_DRIVER=sqlite ./innominatus
```

### Database Corruption

```bash
# Check integrity
sqlite3 ./data/innominatus.db "PRAGMA integrity_check;"

# If corrupted, restore from backup or recreate
rm ./data/innominatus.db
DB_DRIVER=sqlite ./innominatus  # Schema auto-created
```

### Missing Data

**Remember:** In-memory databases (`:memory:`) lose all data on restart!

```bash
# Switch to file-based for persistence
DB_DRIVER=sqlite DB_PATH=./data/innominatus.db ./innominatus
```

### Tests Fail with SQLite

```bash
# Some tests might be PostgreSQL-specific
# Run with PostgreSQL to verify
make test

# Report incompatibilities as bugs
```

## 🎓 Learning Path

### Day 1: Get Started

1. Build the server
2. Run with in-memory SQLite
3. Create a workflow via API
4. Browse Web UI at http://localhost:8081

### Week 1: Development

1. Switch to file-based SQLite
2. Inspect database with sqlite3
3. Run tests with SQLite
4. Make code changes

### Week 2: Production Prep

1. Test with PostgreSQL: `make test`
2. Run both databases: `make test-both`
3. Deploy with PostgreSQL configuration

## 📚 Additional Resources

- [SQLite Support Guide](../development/sqlite-support.md) - Comprehensive SQLite documentation
- [Database Testing](../testing/database-testing.md) - Testing with both databases
- [PostgreSQL Setup](../platform-team-guide/database.md) - Production PostgreSQL configuration
- [SQLite Documentation](https://www.sqlite.org/docs.html) - Official SQLite docs

## ❓ FAQ

**Q: Is SQLite good enough for production?**
A: No. SQLite is excellent for development, but innominatus requires PostgreSQL for production due to concurrency and performance requirements.

**Q: Can I migrate from SQLite to PostgreSQL?**
A: No automatic migration. Start with SQLite for development, then deploy to PostgreSQL. Data doesn't transfer (by design - dev vs prod separation).

**Q: What if I already have PostgreSQL installed?**
A: You can use either! SQLite is optional for convenience. Your existing PostgreSQL setup will continue to work.

**Q: Does SQLite support all features?**
A: Yes, all application features work with SQLite. Some edge cases may behave slightly differently (see [compatibility docs](../development/sqlite-support.md)).

**Q: How do I switch back to PostgreSQL?**
A: Just remove the `DB_DRIVER=sqlite` environment variable:
```bash
./innominatus  # Uses PostgreSQL by default
```

## 🎉 Benefits Summary

- ⚡ **Fast**: Start developing in under 2 minutes
- 🔧 **Simple**: One binary, no external dependencies
- 📦 **Portable**: Share database files easily
- 🧪 **Testable**: Fast test execution
- 👨‍💻 **Developer-friendly**: Perfect for local development
- 🔄 **Flexible**: Switch to PostgreSQL anytime

---

**Ready to develop?** Run this now:

```bash
make build
DB_DRIVER=sqlite ./innominatus
```

Open http://localhost:8081 and start building! 🚀

---

*Last updated: 2025-10-22*
