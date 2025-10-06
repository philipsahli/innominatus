# Graph Tracking Integration

## Overview

This document describes the integration of the `innominatus-graph` SDK into the innominatus IDP orchestrator for workflow and resource dependency tracking.

## Integration Status

### ✅ Completed Components

1. **Adapter Implementation** (`internal/graph/adapter.go`)
   - Wraps innominatus-graph SDK repository with clean API
   - Provides methods: `AddNode()`, `AddEdge()`, `UpdateNodeState()`, `ExportGraph()`, `GetGraph()`
   - Graceful degradation: continues operation if graph adapter fails

2. **Workflow Executor Integration** (`internal/workflow/executor.go`)
   - Graph tracking integrated at key workflow execution points:
     - Workflow node creation on execution start
     - Step node creation for each workflow step
     - Edge creation (workflow contains steps)
     - State updates (waiting → running → succeeded/failed)
   - All graph operations wrapped in nil-checks for resilience

3. **Server Handler Integration** (`internal/server/handlers.go`)
   - Graph adapter initialized in `NewServerWithDB()`
   - Set on workflow executor instance
   - HTTP endpoints registered:
     - `GET /api/graph/{app}` - Get graph status and statistics
     - `GET /api/graph/{app}/export?format={format}` - Export graph visualization

4. **CLI Commands** (`cmd/cli/main.go`, `internal/cli/client.go`)
   - `graph-export <app>` - Export graph to SVG/PNG/DOT format
   - `graph-status <app>` - Show graph statistics (nodes by type/state, edges)

5. **Database Migration** (`migrations/001_create_graph_tables.sql`)
   - Creates 4 tables: `apps`, `nodes`, `edges`, `graph_runs`
   - Schema aligned with GORM model expectations (CHAR(36) for UUIDs)
   - Proper indexes and foreign key constraints

### ⚠️ **CRITICAL BLOCKER**

**GORM Column Error**: Graph tracking currently fails with the following error:

```
ERROR: column "state" of relation "nodes" does not exist (SQLSTATE 42703)
```

**Evidence**:
- Manual INSERT queries work perfectly - the column definitely exists
- GORM-generated INSERT includes "state" in column list but PostgreSQL rejects it
- Error persists across:
  - Schema recreation with CHAR(36) UUID types (matching GORM model)
  - New GORM connections (separate from orchestrator's sql.DB)
  - Adding `search_path=public` to DSN
  - Server restarts and prepared statement clearing

**Root Cause Analysis**:
1. Not a schema issue - verified with `\d nodes` shows state column exists
2. Not a UUID type mismatch - changed from native UUID to CHAR(36)
3. Not a search path issue - explicitly set to public schema
4. Likely a GORM prepared statement caching bug or driver incompatibility

**Impact**:
- Graph adapter initializes successfully
- Workflow execution continues normally (graceful degradation)
- NO graph nodes/edges are persisted to database
- Graph visualization and tracking features non-functional

**Next Steps to Resolve**:
1. **Investigate innominatus-graph SDK**:
   - Check if SDK has known GORM/PostgreSQL compatibility issues
   - Review SDK's GORM configuration and model definitions
   - Test SDK in isolation with same PostgreSQL version

2. **Try Alternative Approaches**:
   - Use GORM's `Exec()` with raw SQL instead of model-based INSERT
   - Implement custom repository layer bypassing GORM for nodes table
   - Use database/sql directly instead of GORM for this integration

3. **SDK Modifications** (if needed):
   - Update innominatus-graph to use database/sql instead of GORM
   - Or fix GORM configuration to disable all prepared statements
   - Or switch to different ORM (sqlx, sqlc, etc.)

## Architecture

### Data Flow

```
Workflow Execution
        ↓
WorkflowExecutor.ExecuteWorkflow()
        ↓
graphAdapter.AddNode(workflowNode)  ← Creates workflow node
        ↓
graphAdapter.AddNode(stepNode)      ← Creates step nodes
        ↓
graphAdapter.AddEdge(workflow→step) ← Creates containment edges
        ↓
graphAdapter.UpdateNodeState()      ← Updates states during execution
        ↓
PostgreSQL (apps, nodes, edges tables)
```

### Graph Model

**Node Types** (from SDK):
- `workflow` - Represents entire workflow execution
- `step` - Individual workflow step
- `resource` - Provisioned resources (database, S3, etc.)
- `spec` - Application specifications

**Node States** (from SDK):
- `waiting` - Step waiting for dependencies
- `pending` - Workflow created, not yet running
- `running` - Currently executing
- `succeeded` - Completed successfully
- `failed` - Execution failed

**Edge Types** (from SDK):
- `contains` - Workflow contains steps
- `depends_on` - Step depends on another step
- `provisions` - Step provisions a resource
- `configures` - Resource configures another resource

## Database Schema

### Tables Created by Migration

**apps** - Application metadata
```sql
CREATE TABLE apps (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**nodes** - Graph nodes (workflows, steps, resources)
```sql
CREATE TABLE nodes (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    state VARCHAR(50) NOT NULL DEFAULT 'waiting',
    properties TEXT DEFAULT '{}',  -- JSON serialized properties
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**edges** - Relationships between nodes
```sql
CREATE TABLE edges (
    id VARCHAR(255) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    from_node_id VARCHAR(255) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    to_node_id VARCHAR(255) NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    properties TEXT DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**graph_runs** - Graph execution metadata
```sql
CREATE TABLE graph_runs (
    id CHAR(36) PRIMARY KEY,
    app_id CHAR(36) NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    execution_plan TEXT,
    metadata TEXT DEFAULT '{}'
);
```

## Usage Examples

### CLI Commands

```bash
# Export workflow graph to SVG
./innominatus-ctl graph-export my-app --format svg --output graph.svg

# Export to PNG
./innominatus-ctl graph-export my-app --format png --output graph.png

# Export DOT format (for Graphviz)
./innominatus-ctl graph-export my-app --format dot --output graph.dot

# Show graph statistics
./innominatus-ctl graph-status my-app
```

### Expected Output (when working)

```
Graph Status for Application: my-app

Total Nodes: 12

Node Counts by Type:
  workflow: 1
  step: 8
  resource: 3

Node Counts by State:
  succeeded: 10
  running: 1
  waiting: 1

Total Edges: 11
```

## Configuration

### Environment Variables

The graph adapter uses the same database connection as the orchestrator:

```bash
DB_HOST=localhost        # Database host
DB_PORT=5432            # Database port
DB_USER=postgres        # Database user
DB_PASSWORD=            # Database password
DB_NAME=idp_orchestrator # Database name
DB_SSLMODE=disable      # SSL mode
```

### Adapter Initialization

Graph adapter is initialized automatically during server startup:

```go
// internal/server/handlers.go:178-186
graphAdapter, err := graph.NewAdapter(db.DB())
if err != nil {
    fmt.Printf("Warning: Failed to initialize graph adapter: %v\n", err)
    fmt.Println("Continuing without graph tracking...")
} else {
    fmt.Println("Graph adapter initialized successfully")
    workflowExecutor.SetGraphAdapter(graphAdapter)
}
```

## Testing

### Manual Testing (once GORM issue resolved)

```bash
# 1. Deploy an application
./innominatus-ctl run deploy-app score-spec.yaml

# 2. Check database for graph nodes
psql -h localhost -U postgres -d idp_orchestrator -c \
  "SELECT COUNT(*) as node_count, type FROM nodes GROUP BY type;"

# Expected output:
#  node_count | type
# ------------+----------
#           1 | workflow
#           8 | step
#           2 | resource

# 3. Export graph visualization
./innominatus-ctl graph-export my-app --format svg --output /tmp/graph.svg
open /tmp/graph.svg  # macOS
```

## Troubleshooting

### Adapter Initialization Failed

**Symptom**: Server logs show "Warning: Failed to initialize graph adapter"

**Solution**:
- Check database connectivity
- Verify migration has been run
- Check environment variables are set correctly

### No Nodes Created

**Symptom**: Database tables empty despite successful workflow execution

**Current Status**: This is the known GORM issue described above

**Workaround**: None currently - see "CRITICAL BLOCKER" section

### Graph Export Returns 404

**Symptom**: `./innominatus-ctl graph-export` returns "server returned status 404"

**Solution**:
- Verify application exists: `./innominatus-ctl list`
- Check graph tracking is working: Query nodes table
- Ensure server is running and API endpoints are registered

## Future Enhancements

Once the GORM issue is resolved:

1. **State Propagation Visualization**
   - Show how failure states propagate up from steps to workflows
   - Highlight critical path in workflow execution

2. **Resource Dependency Graphs**
   - Visualize resource provisioning dependencies
   - Show which resources are shared across applications

3. **Web UI Integration**
   - Embed graph visualizations in web dashboard
   - Interactive graph exploration with zoom/pan
   - Real-time updates during workflow execution

4. **Graph Analytics**
   - Identify common failure patterns
   - Calculate workflow execution metrics
   - Detect circular dependencies

5. **Multi-Application Graphs**
   - Visualize dependencies across multiple applications
   - Platform-wide resource utilization views

## References

- innominatus-graph SDK: `/Users/philipsahli/projects/innominatus-graph`
- Migration SQL: `migrations/001_create_graph_tables.sql`
- Adapter Implementation: `internal/graph/adapter.go`
- Executor Integration: `internal/workflow/executor.go`
- CLI Implementation: `cmd/cli/main.go`, `internal/cli/client.go`

---

*Last Updated: 2025-10-04*
*Status: Integration Complete / GORM Blocker Identified*
