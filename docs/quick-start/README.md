# Quick Start Guides

Choose your development path based on your needs:

## 🚀 [SQLite Development (Recommended for Getting Started)](./sqlite-development.md)

**Start developing in 2 minutes** with zero external dependencies.

- ✅ No Docker required
- ✅ No PostgreSQL installation
- ✅ Fastest setup and test execution
- ✅ Perfect for onboarding and local development

```bash
make build
./scripts/add-sqlite-dependency.sh
DB_DRIVER=sqlite ./innominatus
```

**When to use:** Learning, prototyping, rapid local development, demos

---

## 🐘 PostgreSQL Development (Production-like)

**Production-ready setup** with PostgreSQL for comprehensive testing.

- ✅ Production-like environment
- ✅ Full PostgreSQL feature set
- ✅ Comprehensive testing
- ✅ Migration validation

**Option 1: Docker Compose**
```bash
make db-test-up
./innominatus
```

**Option 2: Local PostgreSQL**
```bash
# Requires PostgreSQL installed on your system
./innominatus  # Uses default PostgreSQL config
```

**When to use:** Final testing before commits, validating migrations, performance testing

---

## 📚 Additional Resources

### Configuration
- [Example SQLite .env](../../.env.sqlite.example)
- [Example PostgreSQL .env](../../.env.postgres.example)

### Development Guides
- [SQLite Support Guide](../development/sqlite-support.md)
- [Database Testing](../testing/database-testing.md)
- [PostgreSQL Configuration](../platform-team-guide/database.md)

### Deployment
- [Kubernetes Deployment](../platform-team-guide/kubernetes-deployment.md)
- [Docker Deployment](../platform-team-guide/docker-deployment.md)

---

## 🎯 Recommended Workflow

### Phase 1: Getting Started (Day 1)
1. Follow [SQLite Development Guide](./sqlite-development.md)
2. Build and run with SQLite
3. Explore the Web UI and API
4. Run tests with SQLite

### Phase 2: Active Development (Week 1-4)
1. Continue using SQLite for daily development
2. Use `TEST_DB_DRIVER=sqlite make test` for fast testing
3. Periodically test with PostgreSQL: `make test`
4. Before committing: `make test-both`

### Phase 3: Production Preparation
1. Full PostgreSQL testing: `make test`
2. Deploy with PostgreSQL configuration
3. Performance testing with PostgreSQL
4. Production deployment

---

## ⚡ Quick Reference

| Command | Database | Use Case |
|---------|----------|----------|
| `DB_DRIVER=sqlite ./innominatus` | SQLite | Daily development |
| `./innominatus` | PostgreSQL | Production-like testing |
| `TEST_DB_DRIVER=sqlite make test` | SQLite | Fast tests |
| `make test` | PostgreSQL | Comprehensive tests |
| `make test-both` | Both | Pre-commit validation |

---

## 💡 Tips

- **Start with SQLite** - Fastest way to get productive
- **Test with both** - Run `make test-both` before pushing
- **Deploy with PostgreSQL** - Always use PostgreSQL in production
- **Use .env files** - Copy `.env.sqlite.example` or `.env.postgres.example`

---

*Happy coding! 🎉*
