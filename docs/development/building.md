# Building from Source

Build innominatus components from source code.

---

## Prerequisites

- **Go 1.21+**
- **Node.js 18+** (for Web UI)
- **PostgreSQL 13+** (for local testing)

---

## Build Server

```bash
go build -o innominatus cmd/server/main.go
```

---

## Build CLI

```bash
go build -o innominatus-ctl cmd/cli/main.go
```

---

## Build Web UI

```bash
cd web-ui
npm install
npm run build
cd ..
```

---

## Run Locally

```bash
# Start server
export DB_USER=postgres
export DB_NAME=idp_orchestrator
./innominatus

# Access Web UI
open http://localhost:8081
```

---

## Development Mode

```bash
# Server with hot reload
go run cmd/server/main.go

# Web UI with hot reload
cd web-ui && npm run dev
```

---

**See:** [CLAUDE.md](../../CLAUDE.md) for detailed development setup.
