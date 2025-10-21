# Embedded Files Architecture

This document describes how innominatus handles static files (migrations, swagger specs, web-ui) for both development and production environments.

## Overview

The application supports two modes:
- **Development**: Reads files from filesystem (migrations/, swagger-*.yaml, web-ui/dist/)
- **Production**: Falls back to embedded files when filesystem files are not available

## Architecture

### Filesystem-First Fallback Pattern

All file loading follows this pattern:
1. Try to read from filesystem (development)
2. If filesystem read fails, fall back to embedded FS (production)
3. Return error if both fail

### Implementation

#### Database Migrations (`internal/database/database.go`)

```go
type Database struct {
    db           *sql.DB
    migrationsFS fs.FS  // Optional: embedded migrations filesystem
}

func (d *Database) RunMigrations() error {
    // Try filesystem first
    if _, err := os.Stat("migrations"); err == nil {
        // Use filesystem migrations
        files, _ := filepath.Glob(filepath.Join("migrations", "*.sql"))
    } else {
        // Use embedded migrations
        // Reads from d.migrationsFS if provided
    }
}
```

**Initialization**:
```go
// Development mode (no embed)
db := database.NewDatabase(connStr, nil)

// Production mode (with embed)
db := database.NewDatabase(connStr, migrationsFS)
```

#### Swagger API Docs (`internal/server/web.go`)

```go
type Server struct {
    swaggerFS fs.FS  // Optional: embedded swagger files
}

func (s *Server) readSwaggerFile(filename string) ([]byte, error) {
    // Try filesystem first (for development)
    if data, err := os.ReadFile(filename); err == nil {
        return data, nil
    }

    // Fallback to embedded FS (for production) if available
    if s.swaggerFS != nil {
        return fs.ReadFile(s.swaggerFS, filename)
    }

    return nil, fmt.Errorf("swagger file %s not found", filename)
}
```

#### Web UI Static Files (`internal/server/handlers.go`)

```go
type Server struct {
    webUIFS fs.FS  // Optional: embedded web-ui files
}

// To be implemented in cmd/server/main.go:
// http.FileServer(http.FS(webUIFS)) instead of http.Dir("web-ui/dist")
```

## Building with Embedded Files

### Step 1: Create embed directives in cmd/server/main.go

```go
package main

import (
    "embed"
    "io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

//go:embed swagger-user.yaml swagger-admin.yaml
var swaggerFS embed.FS

//go:embed web-ui/dist/*
var webUIFS embed.FS
```

**Note**: Go embed directives cannot use `..` paths. Files must be in the same directory or subdirectories of the Go file containing the embed directive.

### Step 2: Copy files to embeddable location (if needed)

If files are in parent directories, create a build script:

```bash
#!/bin/bash
# scripts/prepare-embed.sh

mkdir -p cmd/server/static
cp -r migrations cmd/server/static/
cp swagger-*.yaml cmd/server/static/
cp -r web-ui/dist cmd/server/static/web-ui/
```

Then update embed directives:
```go
//go:embed static/migrations/*.sql
//go:embed static/swagger-*.yaml
//go:embed static/web-ui/*
```

### Step 3: Pass embedded FS to components

```go
func main() {
    // Extract subdirectories from embed.FS
    migrationsSubFS, _ := fs.Sub(migrationsFS, "migrations")
    swaggerSubFS, _ := fs.Sub(swaggerFS, ".")
    webUISubFS, _ := fs.Sub(webUIFS, "web-ui/dist")

    // Initialize with embedded FS
    db := database.NewDatabase(connStr, migrationsSubFS)
    server := server.NewServer(db, /* ... */)
    server.swaggerFS = swaggerSubFS
    server.webUIFS = webUISubFS
}
```

## Development Workflow

### Without Embeds (Current)
```bash
go build -o innominatus cmd/server/main.go
./innominatus  # Reads from filesystem: migrations/, swagger-*.yaml, web-ui/dist/
```

### With Embeds (Future)
```bash
./scripts/prepare-embed.sh  # Copy files to cmd/server/static/
go build -o innominatus cmd/server/main.go
./innominatus  # Falls back to embedded files if filesystem files missing
```

## Verification

To verify embedded build works without repository:

```bash
# Build with embeds
go build -o /tmp/innominatus-standalone cmd/server/main.go

# Test in empty directory
cd /tmp/test-standalone
/tmp/innominatus-standalone

# Should run without errors, using embedded files
```

## Trade-offs

### Current Approach (Filesystem-First)

**Pros:**
- No build-time file copying required
- Fast development cycle (no embedding)
- Can update static files without rebuilding

**Cons:**
- Requires repository structure when running
- Not suitable for standalone binary distribution

### With Embedding

**Pros:**
- Standalone binary distribution
- No external file dependencies
- Simpler deployment (single file)

**Cons:**
- Requires embedding step in build
- Binary size increases
- Changes to static files require rebuild

## Implementation Status

### Completed (P0)
- ✅ Database struct accepts optional migrationsFS
- ✅ Server struct accepts optional swaggerFS and webUIFS
- ✅ Filesystem-first fallback logic implemented
- ✅ Builds successfully without embeds
- ✅ Migration execution supports both filesystem and embedded FS

### Not Yet Implemented
- ⏳ Actual embed directives in cmd/server/main.go
- ⏳ Build script to prepare files for embedding
- ⏳ Server initialization to pass embedded FS
- ⏳ Web UI FileServer integration with webUIFS
- ⏳ CLI embedding for golden paths and other local files

## References

- Go embed documentation: https://pkg.go.dev/embed
- io/fs package: https://pkg.go.dev/io/fs
