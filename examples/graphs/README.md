# Workflow Graph Visualization Examples

This directory contains examples of workflow graph visualization and export capabilities.

## Overview

Innominatus provides powerful graph visualization to help understand:
- Workflow execution flow
- Step dependencies
- Resource provisioning relationships
- Real-time status updates

## Graph API Endpoints

### 1. View Graph Data (JSON)

```bash
# Get graph for an application (frontend format)
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8081/api/graph/my-app

# Returns: (NEW: includes timing metadata)
{
  "app_name": "my-app",
  "nodes": [
    {
      "id": "workflow-1",
      "name": "deploy-app",
      "type": "workflow",
      "state": "running",
      "started_at": "2025-10-30T10:00:00Z",
      "created_at": "2025-10-30T09:59:55Z",
      "updated_at": "2025-10-30T10:00:00Z"
    },
    {
      "id": "step-1",
      "name": "provision-database",
      "type": "step",
      "state": "succeeded",
      "started_at": "2025-10-30T10:00:05Z",
      "completed_at": "2025-10-30T10:02:30Z",
      "duration": "2m25s"
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "from_node_id": "workflow-1",
      "to_node_id": "step-1",
      "type": "contains"
    }
  ]
}
```

### 2. Compute Graph Layout (NEW)

```bash
# Compute hierarchical layout (default)
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/layout?type=hierarchical"

# Compute radial layout with custom spacing
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/layout?type=radial&nodeSpacing=120&width=800&height=600"

# Returns: Node positions for visualization
{
  "nodes": {
    "workflow-1": {
      "node_id": "workflow-1",
      "position": {"x": 600, "y": 50},
      "level": 0
    },
    "step-1": {
      "node_id": "step-1",
      "position": {"x": 400, "y": 200},
      "level": 1
    }
  }
}
```

**Supported Layout Algorithms:**
- `hierarchical` - Top-down tree layout (default)
- `radial` - Circular layout from center
- `force` - Force-directed physics simulation
- `grid` - Grid-based positioning

**Query Parameters:**
- `type` - Layout algorithm (default: hierarchical)
- `nodeSpacing` - Space between nodes (default: 100)
- `levelSpacing` - Space between levels (default: 150)
- `width` - Canvas width (default: 1200)
- `height` - Canvas height (default: 800)

### 3. Export Graph (Multiple Formats)

#### Mermaid Flowchart (Default)

```bash
# Export as Mermaid flowchart with styling
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid" \
  -o my-app-graph.mmd

# Result: Flowchart diagram with node states and color coding
```

#### Mermaid State Diagram (NEW)

```bash
# Export as Mermaid state diagram showing transitions
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid-state" \
  -o my-app-state.mmd

# Result: State diagram showing workflow progression
```

#### Mermaid Gantt Chart (NEW)

```bash
# Export as Mermaid Gantt chart showing timeline
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid-gantt" \
  -o my-app-timeline.mmd

# Result: Gantt chart with execution timeline and durations
```

#### SVG Export

```bash
# Export as SVG image
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=svg" \
  -o my-app-graph.svg
```

#### PNG Export

```bash
# Export as PNG image
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=png" \
  -o my-app-graph.png
```

#### DOT Format

```bash
# Export as Graphviz DOT format
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=dot" \
  -o my-app-graph.dot
```

#### JSON Export (Enhanced)

```bash
# Export as JSON with timing metadata and pretty formatting (NEW)
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=json" \
  -o my-app-graph.json

# Returns enriched JSON with:
# - Node timing (started_at, completed_at, duration)
# - Execution metadata
# - Complete graph structure
```

### 4. Real-time WebSocket Updates (NEW)

```bash
# Connect to WebSocket for live graph updates
wscat -H "Authorization: Bearer $API_TOKEN" \
  -c "ws://localhost:8081/api/graph/my-app/ws"

# Receive real-time notifications when:
# - Node states change (waiting → running → succeeded)
# - Edges are added
# - Timing information updates
```

**JavaScript Example:**
```javascript
const ws = new WebSocket('ws://localhost:8081/api/graph/my-app/ws');
ws.onmessage = (event) => {
  const graphData = JSON.parse(event.data);
  console.log('Graph updated:', graphData);
  updateVisualization(graphData);
};
```

## Supported Export Formats

| Format | Content-Type | Use Case | New |
|--------|-------------|----------|-----|
| `mermaid` | `text/plain` | Flowchart diagrams for documentation | |
| `mermaid-state` | `text/plain` | State transition diagrams | ✅ |
| `mermaid-gantt` | `text/plain` | Timeline/Gantt charts | ✅ |
| `svg` | `image/svg+xml` | Vector graphics, web embedding | |
| `png` | `image/png` | Raster images, presentations | |
| `dot` | `text/plain` | Graphviz processing, custom rendering | |
| `json` | `application/json` | Enhanced with timing metadata | ✅ |

## Mermaid Diagram Features

### Node Types & Colors

- **Spec Nodes** (Blue): Score specification
- **Workflow Nodes** (Yellow): Workflow execution
- **Step Nodes** (Orange): Individual workflow steps
- **Resource Nodes** (Green): Infrastructure resources

### State Indicators

- `✓` - Succeeded
- `✗` - Failed
- `▶` - Running
- `⏸` - Waiting
- `○` - Pending

### Edge Types

- **contains** - Workflow contains steps
- **depends on** - Dependency relationship
- **provisions** - Resource provisioning
- **creates** - Resource creation
- **binds to** - Resource binding
- **configures** - Configuration relationship

## Example Mermaid Output

```mermaid
flowchart TD
    %% Workflow Execution Graph

    %% Nodes
    workflow_1{["▶ deploy-app<br/>workflow"]}
    step_1[["✓ provision-database<br/>step"]]
    resource_1(["⏸ postgres-db<br/>resource"])

    %% Edges
    workflow_1 -->|contains| step_1
    step_1 -->|configures| resource_1

    %% Styling
    classDef workflow fill:#eab308,stroke:#ca8a04,stroke-width:2px,color:#fff
    classDef step fill:#fb923c,stroke:#ea580c,stroke-width:2px,color:#fff
    classDef resource fill:#22c55e,stroke:#16a34a,stroke-width:2px,color:#fff
    classDef running fill:#06b6d4,stroke:#0891b2,stroke-width:2px,color:#fff

    %% Apply styles
    class workflow_1 running
    class step_1 step
    class resource_1 resource
```

## Web UI Visualization

Access the interactive graph visualization in the Web UI:

```
http://localhost:8081/graph/<app-name>
```

Features:
- **Real-time updates**: Nodes update automatically via WebSocket (NEW)
- **Layout algorithms**: Choose from 4 layout types (NEW)
- **Interactive**: Zoom, pan, drag nodes
- **Legend**: Color-coded node types and states
- **Export**: Download graph in multiple formats (NEW)
- **Timing information**: View execution duration for completed nodes (NEW)

## Integration Examples

### Markdown Documentation

```markdown
# My App Deployment Graph

\`\`\`mermaid
<!-- Paste exported Mermaid diagram here -->
\`\`\`
```

### GitHub Actions

```yaml
- name: Export deployment graph
  run: |
    curl -H "Authorization: Bearer $API_TOKEN" \
      "$API_URL/api/graph/my-app/export?format=mermaid" \
      -o deployment-graph.mmd

    # Commit to repository
    git add deployment-graph.mmd
    git commit -m "Update deployment graph"
```

### Slack Notifications

```bash
# Export as PNG and upload to Slack
curl -H "Authorization: Bearer $API_TOKEN" \
  "$API_URL/api/graph/my-app/export?format=png" \
  -o graph.png

curl -F file=@graph.png \
  -F channels=$SLACK_CHANNEL \
  -H "Authorization: Bearer $SLACK_TOKEN" \
  https://slack.com/api/files.upload
```

## Troubleshooting

### Graph Not Found

```
HTTP 404 - Application 'my-app' not found
```

**Solution**: Ensure the application name matches exactly. List applications:

```bash
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8081/api/specs
```

### Empty Graph

If the graph has no nodes/edges, the application may not have been deployed yet, or the workflow hasn't created graph tracking data.

**Solution**: Deploy the application first:

```bash
./innominatus-ctl run deploy-app score-spec.yaml
```

### Format Not Supported

```
HTTP 400 - Unsupported format
```

**Solution**: Use one of the supported formats: `mermaid`, `mermaid-state`, `mermaid-gantt`, `svg`, `png`, `dot`, `json`

### WebSocket Connection Fails

```
WebSocket connection failed or immediately closes
```

**Solution**:
- Ensure application exists and has graph data
- Check WebSocket hub is running (server logs: "WebSocket hub started")
- Verify authorization token is valid
- Check browser console for CORS errors

## Best Practices

1. **Documentation**: Export Mermaid diagrams for technical documentation
   - Use `mermaid-state` for workflow progression
   - Use `mermaid-gantt` for timeline visualization
2. **Monitoring**: Use PNG/SVG exports for dashboards and reports
3. **Analysis**: Use enhanced JSON exports with timing metadata for custom analytics
4. **Real-time Monitoring**: Connect WebSocket for live deployment tracking
5. **Troubleshooting**: View real-time graphs during deployments to identify bottlenecks
6. **Layout Selection**:
   - Use `hierarchical` for workflow dependencies
   - Use `radial` for resource relationships
   - Use `force` for complex interconnected systems

## See Also

- [Web UI Graph Visualization](../../web-ui/README.md)
- [Workflow Execution](../workflows/README.md)
- [Golden Paths](../../docs/GOLDEN_PATHS_METADATA.md)
