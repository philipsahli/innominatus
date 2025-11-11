# Developer UI (`/dev`) - Documentation

## Overview

Minimalistic, data-focused UI for developers to interact with innominatus platform. Coexists with the main UI at `/dev` endpoint.

**Design Philosophy:**
- Table-centric layouts (GitHub Issues / Grafana style)
- Essential features only: Deploy, Monitor, Track, Visualize
- Clean monospace typography for technical data
- Real-time updates for running workflows

**Built with:**
- Next.js 15.5.6 (App Router, React Server Components)
- TypeScript strict mode
- Tailwind CSS + Radix UI primitives
- Embedded in Go binary via `go:embed`

## Pages

### 1. Home - `/dev`
Dashboard with quick stats:
- Total applications
- Active resources
- Recent workflows
- Quick action: Deploy new app

### 2. Applications - `/dev/applications`
List all deployed applications:
- Application name (linked)
- Environment
- Resource count
- Deploy timestamp
- Actions: Deploy new, View details

**Features:**
- Search/filter applications
- Deploy button opens YAML editor
- Click app name to view details

### 3. Resources - `/dev/resources`
Infrastructure resources provisioned by platform:
- Resource name
- Type badge (postgres, s3-bucket, namespace, etc.)
- State with status dot (active, provisioning, failed, etc.)
- Provider ID
- Connection hints preview

**Features:**
- Status dot color coding (green=active, yellow=provisioning, red=failed)
- Click resource name to open sliding detail pane
- Detail pane shows: Full status, Connection details (copyable), Configuration JSON
- Stats footer: Total, Active, Provisioning, Failed counts

**State Types Supported:**
- `active` - Resource ready to use
- `provisioning` - Being created (animated spinner)
- `scaling` - Scaling resources (animated spinner)
- `updating` - Being updated (animated spinner)
- `terminating` - Being deleted (animated spinner)
- `degraded` - Partial functionality
- `terminated` - Deleted
- `requested` - Awaiting provisioning
- `failed` - Provisioning failed

### 4. Workflows - `/dev/workflows`
Track workflow execution history:
- Workflow ID (short hash)
- Workflow name
- Status badge (running, completed, failed, pending)
- Associated application (linked)
- Start timestamp
- Duration
- Actions: Retry (if failed), View graph

**Features:**
- Auto-refresh every 5s when workflows are running
- Status indicator with animation for active workflows
- "Graph" link to view workflow in graph visualization
- Stats footer: Total, Running, Completed, Failed counts

**Status Types:**
- `running` - Executing (animated, pulsing indicator)
- `completed` - Finished successfully
- `failed` - Execution failed
- `pending` - Queued for execution

### 5. Graph - `/dev/graph`
Visual dependency graph:
- Search by application name
- Interactive graph visualization
- Shows: Specs → Resources → Providers → Workflows
- Color-coded nodes by type
- Edges show relationships

## Components Library

### Data Table Components
Reusable table primitives in `src/components/dev/data-table.tsx`:
- `<DataTable>` - Container
- `<DataTableHeader>` - Header row
- `<DataTableHeaderCell>` - Header cell with uppercase styling
- `<DataTableBody>` - Table body
- `<DataTableRow>` - Row with hover state
- `<DataTableCell>` - Cell with optional `mono` className
- `<DataTableEmpty>` - Empty state message
- `<DataTableLoading>` - Loading skeleton

### Status Components
Status indicators in `src/components/dev/status-badge.tsx`:
- `<StatusBadge>` - Full status with icon + label
- `<StatusDot>` - Minimal colored dot

**Features:**
- 12 status types supported
- Automatic spinner animation for transitional states
- Color-coded (green, blue, amber, red, gray)
- Icons from `lucide-react`

### Utility Components
- `<CopyableText>` - Copyable text with label (for connection strings)
- `<CodeBlock>` - Syntax-highlighted code block
- `<NavLink>` - Active state navigation link

## Current Database State

**PostgreSQL Database:** `idp_orchestrator2`

```
applications:         1 row
resource_instances:   0 rows
workflow_executions:  0 rows
```

This means:
- Applications page will show 1 existing application
- Resources page will show "No resources provisioned yet" empty state
- Workflows page will show "No workflow executions yet" empty state

## Screenshots Instructions

### Prerequisites
1. Server running: `./innominatus` (should already be running on http://localhost:8081)
2. Login credentials: `admin` / `admin123` (from users.yaml)

### Step-by-Step

**1. Login**
- Navigate to: http://localhost:8081
- Enter credentials: admin / admin123
- You'll be redirected to main UI

**2. Navigate to Dev UI**
- Click the `/dev` link or navigate directly to: http://localhost:8081/dev

**3. Screenshot Each Page**

Take screenshots of:

a) **Home Dashboard** - http://localhost:8081/dev
   - Shows stat cards (likely all 0s or minimal)

b) **Applications List** - http://localhost:8081/dev/applications
   - Shows 1 application in table
   - Screenshot full page with header + table + stats

c) **Resources List (Empty State)** - http://localhost:8081/dev/resources
   - Shows "No resources provisioned yet" message
   - Demonstrates empty state design

d) **Workflows List (Empty State)** - http://localhost:8081/dev/workflows
   - Shows "No workflow executions yet" message
   - Auto-refresh indicator visible in header

e) **Graph Visualization** - http://localhost:8081/dev/graph
   - Enter the application name from the applications page
   - Click "View Graph"
   - Screenshot the graph visualization

### Optional: Deploy Application for Full State

To see the UI with real data (resources + workflows):

1. Navigate to http://localhost:8081/dev/applications
2. Click "Deploy New Application"
3. Paste a Score spec (example below)
4. Submit
5. Wait 10-30 seconds for provisioning
6. Refresh pages to see populated resources and workflows

**Example Score Spec:**
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: test-app
containers:
  main:
    image: nginx:latest
resources:
  db:
    type: postgres
    properties:
      version: "15"
```

After deployment:
- Applications page: Shows test-app
- Resources page: Shows postgres database (provisioning → active)
- Workflows page: Shows provision-postgres execution
- Graph page: Shows complete dependency graph

## Implementation Details

### Files Modified/Created
- `web-ui/src/app/dev/layout.tsx` - Dev UI layout with top nav
- `web-ui/src/app/dev/page.tsx` - Home dashboard
- `web-ui/src/app/dev/applications/page.tsx` - Applications list
- `web-ui/src/app/dev/resources/page.tsx` - Resources list + detail pane
- `web-ui/src/app/dev/workflows/page.tsx` - Workflows list with auto-refresh
- `web-ui/src/app/dev/graph/page.tsx` - Graph visualization wrapper
- `web-ui/src/components/dev/data-table.tsx` - Reusable table components
- `web-ui/src/components/dev/status-badge.tsx` - Status indicators
- `web-ui/src/components/dev/code-block.tsx` - Code display (already existed)

### Build Process
1. `cd web-ui && npm run build` - Build Next.js static export
2. `./scripts/prepare-embed.sh` - Copy static files to `web-ui-out/`
3. `go build -o innominatus cmd/server/main.go` - Embed UI in binary

### API Integration
All pages use `src/lib/api.ts` client:
- `api.getApplications()` → `ApiResponse<Application[]>`
- `api.getResources()` → `ApiResponse<Record<string, ResourceInstance[]>>` (grouped)
- `api.getWorkflows()` → `ApiResponse<PaginatedResponse<WorkflowExecution>>`

### Type Safety
Strict TypeScript interfaces in `src/lib/api.ts`:
```typescript
interface Application {
  name: string;
  resources: number;
  environment: string;
  created_at: string;
}

interface ResourceInstance {
  id: string;
  resource_name: string;
  resource_type: string;
  state: string;
  provider_id: string;
  hints: Record<string, any>;
  configuration: Record<string, any>;
}

interface WorkflowExecution {
  id: string;
  name: string;
  status: string;
  app_name: string;
  timestamp: string;
  duration: string;
}
```

## Key Features Implemented

1. **Progressive Disclosure**
   - Resources: Click name → Sliding detail pane
   - Workflows: Graph link for each execution

2. **Real-time Updates**
   - Workflows: Auto-refresh every 5s when running workflows exist
   - Pulsing indicator shows active refresh

3. **Empty States**
   - Resources: "No resources provisioned yet"
   - Workflows: "No workflow executions yet"
   - Clean, centered messaging

4. **Status Management**
   - 12 status types with color coding
   - Animated spinners for transitional states
   - Consistent across resources and workflows

5. **Copy to Clipboard**
   - Connection details in resource detail pane
   - One-click copy for passwords, endpoints, etc.

6. **Responsive Stats**
   - Footer stats on all list pages
   - Real-time count updates
   - Failed items highlighted in red

## Next Steps (Future Enhancements)

- Add deployment wizard with Score spec templates
- Real-time WebSocket updates instead of polling
- Filter/search on all list pages
- Export workflow logs
- Resource health checks visualization
- Workflow retry with parameter override
- Dark mode toggle
- Keyboard shortcuts (vim-style navigation)
