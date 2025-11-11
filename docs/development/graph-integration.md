# Graph Tracking Integration

## Overview

This document describes the integration of the enhanced `innominatus-graph` SDK into the innominatus IDP orchestrator for workflow and resource dependency tracking with advanced visualization and real-time updates.

## Integration Status: ✅ COMPLETE

### Phase 1: Core Integration ✅

1. **Adapter Implementation** (`internal/graph/adapter.go`)
   - Wraps innominatus-graph SDK repository with clean API
   - Core methods: `AddNode()`, `AddEdge()`, `UpdateNodeState()`, `GetGraph()`
   - Enhanced methods: `ExportGraphJSON()`, `ExportGraphMermaid()`, `ComputeGraphLayout()`
   - Observer support: `AddObserver()`, `RemoveObserver()`
   - Graceful degradation: continues operation if graph adapter fails

2. **Workflow Executor Integration** (`internal/workflow/executor.go`)
   - Graph tracking integrated at key workflow execution points:
     - Workflow node creation on execution start
     - Step node creation for each workflow step
     - Edge creation (workflow contains steps)
     - State updates (waiting → running → succeeded/failed)
   - All graph operations wrapped in nil-checks for resilience

3. **Database Migration** (`migrations/001_create_graph_tables.sql`)
   - Creates prefixed tables: `graph_apps`, `graph_nodes`, `graph_edges`
   - Additional table for timing: `workflow_step_executions`
   - Schema optimized for GORM compatibility
   - Proper indexes and foreign key constraints

### Phase 2: Enhanced SDK Integration ✅ (NEW)

4. **Observer Pattern Implementation** (`internal/orchestration/graph_observer.go`)
   - Implements SDK's `GraphObserver` interface
   - Receives real-time notifications for:
     - Node state changes (e.g., waiting → running → completed)
     - Node updates (timing information changes)
     - Edge additions
     - Graph structure changes
   - Forwards events to WebSocket hub for client broadcasting
   - Converts SDK graph format to frontend-compatible JSON

5. **WebSocket Real-time Updates** (`internal/server/websocket.go`, `handlers.go`)
   - Observer automatically registered on server startup
   - WebSocket endpoint: `GET /api/graph/{app}/ws`
   - Broadcasts graph changes to all connected clients
   - Initial graph state sent on connection
   - Ping/pong keep-alive mechanism

6. **Enhanced API Endpoints** (`internal/server/handlers.go`)
   - **Base endpoints**:
     - `GET /api/graph/{app}` - Get graph status and statistics
   - **Layout computation** (NEW):
     - `GET /api/graph/{app}/layout?type={algorithm}` - Compute node positions
     - Algorithms: hierarchical, radial, force-directed, grid
     - Configurable: nodeSpacing, levelSpacing, width, height
   - **Enhanced export** (NEW):
     - `GET /api/graph/{app}/export?format=json` - JSON with timing metadata
     - `GET /api/graph/{app}/export?format=mermaid` - Flowchart diagram
     - `GET /api/graph/{app}/export?format=mermaid-state` - State diagram
     - `GET /api/graph/{app}/export?format=mermaid-gantt` - Gantt timeline
     - `GET /api/graph/{app}/export?format=svg` - SVG image
     - `GET /api/graph/{app}/export?format=png` - PNG image
     - `GET /api/graph/{app}/export?format=dot` - GraphViz DOT
   - **Real-time updates**:
     - `WS /api/graph/{app}/ws` - WebSocket for live updates

7. **CLI Commands** (`cmd/cli/main.go`, `internal/cli/client.go`)
   - `graph-export <app>` - Export graph with enhanced formats
   - `graph-status <app>` - Show graph statistics (nodes by type/state, edges)

## Architecture

### Data Flow (with Observer Pattern)

```
Workflow Execution
        ↓
WorkflowExecutor.ExecuteWorkflow()
        ↓
graphAdapter.AddNode(workflowNode)  ← Creates workflow node
        ↓                              ↓
graphAdapter.AddNode(stepNode)      ObservableGraph wraps Graph
        ↓                              ↓
graphAdapter.AddEdge(workflow→step) Notifies registered observers
        ↓                              ↓
graphAdapter.UpdateNodeState()      GraphObserver.OnNodeStateChanged()
        ↓                              ↓
PostgreSQL (persistence)            WebSocketHub.BroadcastGraphUpdate()
                                       ↓
                                    Connected WebSocket clients (real-time UI updates)
```

**Observer Pattern Flow:**
1. Adapter wraps each loaded Graph as ObservableGraph
2. Registers all observers (currently: GraphObserver for WebSocket broadcasting)
3. Graph operations trigger observer callbacks automatically
4. Observer converts SDK format to frontend JSON
5. WebSocket hub broadcasts to all clients watching that application

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

# NEW: Export to Mermaid flowchart
./innominatus-ctl graph-export my-app --format mermaid --output graph.mmd

# NEW: Export to Mermaid state diagram
./innominatus-ctl graph-export my-app --format mermaid-state --output state.mmd

# NEW: Export to Mermaid Gantt chart (timeline)
./innominatus-ctl graph-export my-app --format mermaid-gantt --output timeline.mmd

# NEW: Export to JSON with timing metadata
./innominatus-ctl graph-export my-app --format json --output graph.json

# Show graph statistics
./innominatus-ctl graph-status my-app
```

### API Examples

```bash
# Get graph data (frontend format)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8081/api/graph/my-app

# Compute hierarchical layout
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/layout?type=hierarchical"

# Compute radial layout with custom spacing
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/layout?type=radial&nodeSpacing=120&width=800&height=600"

# Export as Mermaid flowchart
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid"

# Export as JSON with timing
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=json"

# Export as Gantt chart
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid-gantt"
```

### WebSocket Example (JavaScript)

```javascript
// Connect to WebSocket for real-time graph updates
const ws = new WebSocket('ws://localhost:8081/api/graph/my-app/ws');

ws.onopen = () => {
  console.log('Connected to graph updates');
};

ws.onmessage = (event) => {
  const graphData = JSON.parse(event.data);
  console.log('Graph updated:', graphData);
  // Update UI with new graph state
  updateGraphVisualization(graphData);
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

### Expected Output

**Graph Status:**
```
Graph Status for Application: ecommerce-backend

Total Nodes: 3

Node Counts by Type:
  workflow: 1
  resource: 2

Node Counts by State:
  running: 3

Total Edges: 2
```

**Mermaid Flowchart Export:**
```mermaid
---
title: ecommerce-backend
---
flowchart TB
    resource_1((postgres [running]))
    class resource_1 running
    resource_2((redis [running]))
    class resource_2 running

    classDef running fill:#bbdefb,stroke:#1976d2,stroke-width:3px
    classDef succeeded fill:#c8e6c9,stroke:#388e3c,stroke-width:2px
    classDef failed fill:#ffcdd2,stroke:#d32f2f,stroke-width:3px
```

**Layout Computation Response:**
```json
{
  "nodes": {
    "workflow-123": {
      "node_id": "workflow-123",
      "position": {"x": 600, "y": 50},
      "level": 0
    },
    "step-postgres": {
      "node_id": "step-postgres",
      "position": {"x": 400, "y": 200},
      "level": 1
    },
    "step-redis": {
      "node_id": "step-redis",
      "position": {"x": 800, "y": 200},
      "level": 1
    }
  }
}
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

Graph adapter and observer are initialized automatically during server startup:

```go
// internal/server/handlers.go
// Initialize WebSocket hub for real-time graph updates
wsHub := NewGraphWebSocketHub()
go wsHub.Run()

// Initialize graph adapter
graphAdapter, err := graph.NewAdapter(db.DB())
if err != nil {
    fmt.Printf("Warning: Failed to initialize graph adapter: %v\n", err)
    fmt.Println("Continuing without graph tracking...")
} else {
    fmt.Println("Graph adapter initialized successfully")

    // Register graph observer for real-time WebSocket updates
    graphObserver := orchestration.NewGraphObserver(graphAdapter, wsHub)
    graphAdapter.AddObserver(graphObserver)
    fmt.Println("Graph observer registered for real-time updates")

    workflowExecutor.SetGraphAdapter(graphAdapter)
    resourceManager.SetGraphAdapter(graphAdapter)
}
```

## Testing

### Manual Testing

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
#           2 | resource

# 3. Export graph visualization (multiple formats)
./innominatus-ctl graph-export my-app --format svg --output /tmp/graph.svg
./innominatus-ctl graph-export my-app --format mermaid --output /tmp/graph.mmd
./innominatus-ctl graph-export my-app --format json --output /tmp/graph.json
open /tmp/graph.svg  # macOS

# 4. Test layout computation
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8081/api/graph/my-app/layout?type=hierarchical"

# 5. Test WebSocket updates (in browser console)
const ws = new WebSocket('ws://localhost:8081/api/graph/my-app/ws');
ws.onmessage = (e) => console.log('Graph update:', JSON.parse(e.data));
```

### Testing Observer Pattern

```bash
# 1. Start server with observer enabled (automatic)
./innominatus

# Expected logs:
# Graph adapter initialized successfully
# Graph observer registered for real-time updates

# 2. Deploy application and watch for observer notifications
./innominatus-ctl run deploy-app score-spec.yaml

# Expected server logs:
# Graph: registered observer (total: 1)
# Node state changed: workflow-xxx pending → running
# Node state changed: resource-postgres waiting → running
# Broadcasting graph update to 0 clients
```

### Integration Testing

```bash
# Test complete workflow with all features
./scripts/test-graph-integration.sh

# This script:
# 1. Deploys test application
# 2. Exports graph in all formats
# 3. Computes layouts with all algorithms
# 4. Verifies observer notifications
# 5. Tests WebSocket connectivity
```

## Troubleshooting

### Adapter Initialization Failed

**Symptom**: Server logs show "Warning: Failed to initialize graph adapter"

**Solution**:
- Check database connectivity: `psql -h localhost -U postgres -l`
- Verify migration has been run: `./innominatus migrate up`
- Check environment variables are set correctly
- Review database logs for connection errors

### Observer Not Registered

**Symptom**: No "Graph observer registered" message in server logs

**Solution**:
- Ensure WebSocket hub is initialized before graph adapter
- Check for errors during observer registration
- Verify orchestration package imports correctly

### No Nodes Created

**Symptom**: Database tables empty despite successful workflow execution

**Solution**:
- Check if graph adapter is set on workflow executor
- Verify SetGraphAdapter() was called during initialization
- Check server logs for graph operation errors
- Ensure database migration created tables with correct schema

### Graph Export Returns 404

**Symptom**: `./innominatus-ctl graph-export` returns "server returned status 404"

**Solution**:
- Verify application exists: `./innominatus-ctl list`
- Check graph tracking is working: `psql -c "SELECT * FROM nodes;"`
- Ensure server is running: `curl http://localhost:8081/health`
- Verify API endpoints are registered correctly

### WebSocket Connection Fails

**Symptom**: WebSocket connection returns 404 or immediately closes

**Solution**:
- Verify WebSocket hub is running: Check "WebSocket hub started" in logs
- Ensure application name in URL is correct
- Check browser console for CORS or connection errors
- Verify server is accessible at specified URL

## Future Enhancements

### Completed (Phase 2)
- ✅ Real-time WebSocket updates via observer pattern
- ✅ Enhanced JSON export with timing metadata
- ✅ Mermaid diagram export (flowchart, state, Gantt)
- ✅ Layout computation (4 algorithms: hierarchical, radial, force, grid)
- ✅ GraphObserver for event-driven updates

### Planned (Phase 3)

1. **Web UI Integration** (High Priority)
   - Embed graph visualizations in web dashboard
   - Interactive graph exploration with zoom/pan using computed layouts
   - Real-time updates during workflow execution (observer already implemented)
   - Integration with existing React/TypeScript frontend

2. **State Propagation Visualization**
   - Show how failure states propagate up from steps to workflows
   - Highlight critical path in workflow execution
   - Visual diff showing state changes over time

3. **Resource Dependency Graphs**
   - Visualize resource provisioning dependencies
   - Show which resources are shared across applications
   - Cross-application dependency tracking

4. **Graph Analytics**
   - Identify common failure patterns using historical data
   - Calculate workflow execution metrics (avg duration, success rate)
   - Detect circular dependencies
   - Performance bottleneck identification

5. **Multi-Application Graphs**
   - Visualize dependencies across multiple applications
   - Platform-wide resource utilization views
   - Tenant/team-level aggregated views

6. **Enhanced Export Formats**
   - PDF reports with embedded graphs
   - Interactive HTML export
   - Prometheus metrics export for monitoring

7. **Graph Versioning**
   - Track graph evolution over time
   - Compare graph structures across versions
   - Rollback support for workflow definitions

## References

### SDK and Core Components
- **innominatus-graph SDK**: `/Users/philipsahli/projects/innominatus-graph`
  - `pkg/graph/types.go` - Graph data structures with timing fields
  - `pkg/graph/observers.go` - Observer pattern implementation
  - `pkg/export/json.go` - JSON export with metadata
  - `pkg/export/mermaid.go` - Mermaid diagram generation
  - `pkg/layout/` - Layout algorithms (hierarchical, radial, force, grid)

### Integration Components
- **Graph Adapter**: `internal/graph/adapter.go` - SDK wrapper with enhanced methods
- **Observer**: `internal/orchestration/graph_observer.go` - WebSocket bridge
- **API Handlers**: `internal/server/handlers.go` - Graph endpoints
- **WebSocket**: `internal/server/websocket.go` - Real-time updates
- **Workflow Executor**: `internal/workflow/executor.go` - Graph tracking integration
- **Resource Manager**: `internal/resources/manager.go` - Resource graph tracking

### Database
- **Migration**: `migrations/001_create_graph_tables.sql`
- **Tables**: `apps`, `nodes`, `edges`, `graph_runs`

### Documentation and API
- **API Documentation**: `swagger-user.yaml` - Graph endpoints
- **CLI Reference**: `cmd/cli/main.go`, `internal/cli/client.go`
- **Examples**: `examples/graphs/` - Usage examples

### Testing
- **Demo Script**: `/tmp/demo-graph-features.sh` - Feature demonstration
- **Test Deployment**: `/tmp/postgres-deployment.yaml` - Sample Score spec

---

*Last Updated: 2025-10-30*
*Status: ✅ Phase 2 Integration Complete - Observer Pattern & Enhanced SDK*
