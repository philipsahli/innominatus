# Setup Instructions

## Quick Setup

### 1. Install Go Dependencies

Due to network connectivity during development, some Go dependencies need to be added manually. Run these scripts when you have network access:

```bash
# Add testcontainers dependencies (required for tests)
./scripts/add-testcontainers-dependency.sh

# Add SQLite dependency (required for SQLite support)
./scripts/add-sqlite-dependency.sh
```

Or add them manually:

```bash
# Testcontainers (for PostgreSQL tests)
go get github.com/testcontainers/testcontainers-go@latest
go get github.com/testcontainers/testcontainers-go/modules/postgres@latest

# SQLite (for SQLite development/testing)
go get github.com/mattn/go-sqlite3@latest

# Clean up
go mod tidy
```

### 2. Build the Application

```bash
# Build all components
make build

# Or build individually
go build -o innominatus cmd/server/main.go
go build -o innominatus-ctl cmd/cli/main.go
```

### 3. Run the Server

**With SQLite (fastest for development):**
```bash
DB_DRIVER=sqlite ./innominatus
```

**With PostgreSQL (production-like):**
```bash
# Start PostgreSQL (Docker Compose)
make db-test-up

# Run server
./innominatus
```

## Detailed Setup

See the following guides for detailed setup instructions:

- **Quick Start (SQLite)**: [docs/quick-start/sqlite-development.md](docs/quick-start/sqlite-development.md)
- **Quick Start (PostgreSQL)**: [docs/quick-start/README.md](docs/quick-start/README.md)
- **Database Testing**: [docs/testing/database-testing.md](docs/testing/database-testing.md)
- **SQLite Support**: [docs/development/sqlite-support.md](docs/development/sqlite-support.md)

## Troubleshooting

### Build Errors

**Error: "no required module provides package github.com/testcontainers/testcontainers-go"**

Solution: Run `./scripts/add-testcontainers-dependency.sh`

**Error: "no required module provides package github.com/mattn/go-sqlite3"**

Solution: Run `./scripts/add-sqlite-dependency.sh`

**Error: "pattern migrations/*.sql: no matching files found"**

Solution: Make sure you're running the build from the project root directory.

### Network Issues During Dependency Installation

If you're experiencing network connectivity issues when running `go get`:

1. **Check DNS**: Try `ping storage.googleapis.com`
2. **Use Go Proxy**: Set `GOPROXY=https://proxy.golang.org,direct`
3. **Offline Mode**: If you have go.mod and go.sum with all dependencies already listed, run `go mod download` which may use cached versions

### Runtime Issues

**"database is locked" (SQLite)**

Another process is using the SQLite file. Kill it or use a different path:
```bash
DB_DRIVER=sqlite DB_PATH=./data/innominatus-2.db ./innominatus
```

**"connection refused" (PostgreSQL)**

PostgreSQL is not running. Start it:
```bash
make db-test-up
```

## Next Steps

1. **Run Tests**: `make test-sqlite` or `make test`
2. **Read Documentation**: Start with [docs/quick-start/README.md](docs/quick-start/README.md)
3. **Explore Web UI**: Open http://localhost:8081 after starting the server
4. **Try CLI**: Run `./innominatus-ctl --help`

## Getting Help

- **Documentation**: See `docs/` directory
- **Issues**: https://github.com/philipsahli/innominatus/issues
- **Main README**: [CLAUDE.md](CLAUDE.md)
