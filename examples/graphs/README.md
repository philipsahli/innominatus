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
# Get graph for an application
curl -H "Authorization: Bearer $API_TOKEN" \
  http://localhost:8081/api/graph/my-app

# Returns:
{
  "nodes": [
    {
      "id": "workflow-1",
      "name": "deploy-app",
      "type": "workflow",
      "status": "running",
      "metadata": {...}
    },
    {
      "id": "step-1",
      "name": "provision-database",
      "type": "step",
      "status": "succeeded",
      "metadata": {...}
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "source_id": "workflow-1",
      "target_id": "step-1",
      "type": "contains"
    }
  ]
}
```

### 2. Export Graph (Multiple Formats)

#### Mermaid Diagram (Default)

```bash
# Export as Mermaid flowchart
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid" \
  -o my-app-graph.mmd

# Mermaid diagram with full styling and state indicators
```

#### Simplified Mermaid

```bash
# Export as simplified horizontal layout
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=mermaid-simple" \
  -o my-app-graph-simple.mmd
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

#### JSON Export

```bash
# Export as JSON (with download headers)
curl -H "Authorization: Bearer $API_TOKEN" \
  "http://localhost:8081/api/graph/my-app/export?format=json" \
  -o my-app-graph.json
```

## Supported Export Formats

| Format | Content-Type | Use Case |
|--------|-------------|----------|
| `mermaid` | `text/plain` | Documentation, Markdown files, interactive diagrams |
| `mermaid-simple` | `text/plain` | Simplified horizontal layout for quick reference |
| `svg` | `image/svg+xml` | Vector graphics, web embedding, high-quality prints |
| `png` | `image/png` | Raster images, presentations, reports |
| `dot` | `text/plain` | Graphviz processing, custom rendering |
| `json` | `application/json` | Data analysis, custom visualization tools |

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
- **Real-time updates**: Nodes update as workflow progresses
- **Interactive**: Zoom, pan, drag nodes
- **Legend**: Color-coded node types and states
- **Export**: Download graph data as JSON
- **Refresh**: Manual refresh to sync latest state

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

**Solution**: Use one of the supported formats: `mermaid`, `mermaid-simple`, `svg`, `png`, `dot`, `json`

## Best Practices

1. **Documentation**: Export Mermaid diagrams for technical documentation
2. **Monitoring**: Use PNG exports for dashboards and reports
3. **Analysis**: Use JSON exports for custom analytics
4. **Troubleshooting**: View real-time graphs during deployments to identify bottlenecks

## See Also

- [Web UI Graph Visualization](../../web-ui/README.md)
- [Workflow Execution](../workflows/README.md)
- [Golden Paths](../../docs/GOLDEN_PATHS_METADATA.md)
