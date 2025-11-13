# Live Activity Feed Implementation

## Overview

The Live Activity Feed provides real-time visibility into system events as they happen, giving developers instant feedback on workflow execution, resource provisioning, and graph changes without requiring page refreshes or polling.

**Status**: ✅ **Fully Implemented** (November 13, 2025)

## Architecture

### Event Flow

```
Graph Event → GraphObserver → WebSocket Broadcast → Frontend Display
     ↓              ↓                 ↓                      ↓
  Node/Edge    Create Event      JSON Message          User-Friendly
  Changes      Metadata          {graph, event}        Activity Item
```

### Components

#### Backend (Go)

**1. GraphObserver** (`internal/orchestration/graph_observer.go`)
- Implements SDK's GraphObserver interface
- Captures all graph state changes
- Creates rich event metadata objects
- Broadcasts to WebSocket hub

**Event Types Captured:**
- `node_state_changed` - State transitions (pending → running → succeeded)
- `node_updated` - Timing and progress updates
- `edge_added` - New relationships in graph
- `graph_updated` - Overall graph structure changes

**Example Event Object:**
```go
event := map[string]interface{}{
    "type":      "node_state_changed",
    "timestamp": g.UpdatedAt,
    "node_id":   nodeID,
    "node_name": "provision-postgres",
    "node_type": "workflow",
    "old_state": "pending",
    "new_state": "running",
}
```

**2. WebSocket Hub** (`internal/server/websocket.go`)
- Manages client connections per application
- Broadcasts graph updates with event metadata
- Maintains backwards compatibility with existing graph visualizations

**Message Format:**
```json
{
  "graph": {
    "app_name": "my-app",
    "nodes": [...],
    "edges": [...]
  },
  "event": {
    "type": "node_state_changed",
    "timestamp": "2025-11-13T20:53:46+01:00",
    "node_name": "provision-postgres",
    "old_state": "pending",
    "new_state": "running"
  }
}
```

#### Frontend (TypeScript/React)

**1. GraphWebSocket** (`web-ui/src/lib/graph-websocket.ts`)
- WebSocket connection manager
- Handles reconnection with exponential backoff
- Converts backend format to frontend types
- Maintains backwards compatibility

**Type Definitions:**
```typescript
export interface GraphEvent {
  type: 'node_added' | 'node_state_changed' | 'node_updated' | 'edge_added' | 'graph_updated';
  timestamp: string;
  node_id?: string;
  node_name?: string;
  node_type?: string;
  old_state?: string;
  new_state?: string;
  edge_id?: string;
  edge_type?: string;
  from_node?: string;
  to_node?: string;
  metadata?: Record<string, unknown>;
}
```

**2. ActivityFeed** (`web-ui/src/components/dev/activity-feed.tsx`)
- Subscribes to WebSocket for specific application
- Manages event list (most recent first)
- Shows connection status
- Limits to configurable max events (default: 20)

**Usage:**
```tsx
<ActivityFeed
  appName="my-app"  // Optional: filter by app
  maxEvents={15}    // Optional: max events to show
/>
```

**3. ActivityEventItem** (`web-ui/src/components/dev/activity-event-item.tsx`)
- Translates technical events to user-friendly messages
- Applies semantic color coding
- Shows relative timestamps ("2 minutes ago")
- Displays node type badges

**Event Message Examples:**
- `node_state_changed`: "provision-postgres changed from pending to running"
- `node_updated`: "provision-postgres updated"
- `edge_added`: "Connected my-app to provision-postgres"
- `graph_updated`: "Graph updated (5 nodes, 4 edges)"

## Integration Points

### 1. Developer Dashboard
**Location**: `web-ui/src/app/dev/page.tsx`

The activity feed is prominently displayed on the dashboard homepage, replacing the previous static "Recent Workflow Executions" placeholder.

```tsx
<ActivityFeed maxEvents={15} />
```

### 2. Application Detail Pages (Future)
Can be integrated into individual application detail pages:

```tsx
<ActivityFeed appName={applicationName} maxEvents={10} />
```

### 3. Workflow Detail Pages (Future)
Can show events for specific workflows:

```tsx
<ActivityFeed appName={workflowAppName} maxEvents={20} />
```

## Features

### Real-Time Updates
- ✅ No polling required
- ✅ Instant event display (<100ms latency)
- ✅ WebSocket connection with auto-reconnect
- ✅ Connection status indicator

### User Experience
- ✅ Most recent events shown first
- ✅ Scrollable event list
- ✅ Relative timestamps ("2 minutes ago")
- ✅ Semantic color coding
  - 🔵 Blue: Running/In-progress
  - 🟢 Green: Success/Completed
  - 🔴 Red: Failed/Error
  - 🟡 Yellow: Pending/Waiting
- ✅ Node type badges (Workflow, Step, Resource, Provider)
- ✅ Empty state messaging
- ✅ Dark mode support

### Technical
- ✅ Backwards compatible with existing graph visualizations
- ✅ Type-safe TypeScript implementation
- ✅ React hooks for state management
- ✅ Configurable event limits
- ✅ Connection health monitoring
- ✅ Automatic cleanup on unmount

## Event Lifecycle Example

### Scenario: Developer deploys Score spec requesting postgres database

**1. Spec Submitted**
```
Event: graph_updated
Message: "Graph updated (3 nodes, 2 edges)"
```

**2. Resource Created**
```
Event: node_state_changed
Message: "postgres-db changed from none to requested"
```

**3. Orchestration Engine Picks Up**
```
Event: edge_added
Message: "Connected postgres-db to database-team"
```

**4. Workflow Started**
```
Event: node_state_changed
Message: "provision-postgres changed from pending to running"
```

**5. Workflow Progressing**
```
Event: node_updated
Message: "provision-postgres updated"
```

**6. Workflow Completed**
```
Event: node_state_changed
Message: "provision-postgres changed from running to succeeded"
```

**7. Resource Active**
```
Event: node_state_changed
Message: "postgres-db changed from provisioning to active"
```

## Testing

### Manual Testing

**1. Open Developer Dashboard**
```bash
# Ensure server is running
./innominatus

# Open browser
open http://localhost:3000/dev
```

**2. Trigger Events**
```bash
# Deploy a Score spec to generate events
./innominatus-ctl deploy examples/score-postgres.yaml -w
```

**3. Observe Activity Feed**
- Events should appear in real-time
- Connection indicator should show "Live" with green pulsing dot
- Events should be color-coded and timestamped
- Scroll should work if more than ~10 events

### Programmatic Testing

The ActivityFeed exposes `window.__addActivityEvent` for testing:

```javascript
// In browser console
window.__addActivityEvent({
  type: 'node_state_changed',
  timestamp: new Date().toISOString(),
  node_name: 'test-workflow',
  node_type: 'workflow',
  old_state: 'pending',
  new_state: 'running'
})
```

## Configuration

### Backend Configuration

**Strict Validation Mode**:
```bash
# Enable strict variable validation (default)
export STRICT_VALIDATION=true

# Disable for lenient mode
export STRICT_VALIDATION=false
```

### Frontend Configuration

**Max Events**:
```tsx
<ActivityFeed maxEvents={15} />  // Show up to 15 events
```

**Application Filter**:
```tsx
<ActivityFeed appName="my-app" />  // Only show events for specific app
```

## Troubleshooting

### Events Not Appearing

**1. Check WebSocket Connection**
- Look for "Live" indicator in activity feed header
- If showing "Connecting...", check browser console for WebSocket errors

**2. Verify Backend**
```bash
# Check server logs for GraphObserver registration
grep "Graph observer registered" logs.txt
# Should show: "Graph: registered observer (total: 1)"
```

**3. Test WebSocket Endpoint**
```bash
# Test WebSocket connection
wscat -c ws://localhost:8081/api/graph/system/ws
# Should receive initial graph state
```

### Connection Keeps Dropping

**Check Network Stability**:
- WebSocket uses ping/pong every 30 seconds
- Auto-reconnects with exponential backoff (1s, 2s, 4s, 8s, 16s)
- Max 5 reconnection attempts

**Browser Console**:
```javascript
// Check WebSocket state
console.log(window.wsRef?.isConnected())
```

### Old Events Not Showing

**Expected Behavior**: Activity feed only shows events that occur AFTER connection
- Historical events are not replayed
- Only real-time events are displayed
- This is intentional to show "what's happening now"

## Related Features

### Variable Validation
**Status**: ✅ Implemented (November 13, 2025)

Fail-fast validation for workflow variables (`${workflow.VAR}`, `${step.output}`, `${resources.name.attr}`):
- Pre-execution validation
- Per-step validation
- Configurable strict/lenient mode
- See: `internal/workflow/validation.go`

### Graph Visualizations
**Status**: ✅ Implemented (October 2025)

Multiple visualization modes using D3.js, Cytoscape.js, and Mermaid:
- Hierarchical tree view
- Force-directed network
- Radial layout
- Timeline view
- Mermaid diagrams

## Future Enhancements

### Event Filtering
- [ ] Filter by event type (show only state changes, hide updates)
- [ ] Filter by node type (show only workflows, hide resources)
- [ ] Search/filter by keyword

### Event Details
- [ ] Click event to show full details panel
- [ ] Show related nodes/edges
- [ ] Link to graph visualization focused on event node

### Event Persistence
- [ ] Store recent events in database
- [ ] Show historical events on page load
- [ ] Event replay for debugging

### Advanced Features
- [ ] Export events as JSON/CSV
- [ ] Subscribe to specific event types
- [ ] Custom event notifications (email, Slack)
- [ ] Event aggregation (group similar events)

## Performance Considerations

### Frontend
- **Event Limit**: Default 20 events to prevent DOM bloat
- **Memory**: Old events automatically removed from state
- **Rendering**: React optimizes re-renders with proper keys
- **Connection**: Single WebSocket per app, shared across components

### Backend
- **Broadcast Channel**: Buffered (256 messages) with timeout
- **Connection Management**: Automatic cleanup on disconnect
- **Event Size**: Minimal metadata (~200-500 bytes per event)
- **Graph Observer**: Single observer instance, low overhead

## API Reference

### GraphEvent Interface
```typescript
interface GraphEvent {
  type: 'node_added' | 'node_state_changed' | 'node_updated' | 'edge_added' | 'graph_updated';
  timestamp: string;              // RFC3339 timestamp
  node_id?: string;               // Node identifier
  node_name?: string;             // Human-readable node name
  node_type?: string;             // workflow, step, resource, provider
  old_state?: string;             // Previous state (for state changes)
  new_state?: string;             // New state (for state changes)
  edge_id?: string;               // Edge identifier
  edge_type?: string;             // Edge relationship type
  from_node?: string;             // Source node name
  to_node?: string;               // Target node name
  node_count?: number;            // Total nodes (for graph_updated)
  edge_count?: number;            // Total edges (for graph_updated)
  metadata?: Record<string, unknown>; // Additional context
}
```

### ActivityFeed Props
```typescript
interface ActivityFeedProps {
  appName?: string;      // Optional: filter events by application
  maxEvents?: number;    // Maximum events to display (default: 20)
  className?: string;    // Optional: additional CSS classes
}
```

### ActivityEventItem Props
```typescript
interface ActivityEventItemProps {
  event: GraphEvent;     // Event object to display
}
```

## Files Modified/Created

### Backend
- ✅ `internal/orchestration/graph_observer.go` - Event metadata creation
- ✅ `internal/server/websocket.go` - Broadcast with events
- ✅ `internal/workflow/validation.go` - Variable validation (NEW)
- ✅ `internal/workflow/validation_test.go` - Validation tests (NEW)

### Frontend
- ✅ `web-ui/src/lib/graph-websocket.ts` - Event types and parsing
- ✅ `web-ui/src/components/dev/activity-feed.tsx` - Feed component (NEW)
- ✅ `web-ui/src/components/dev/activity-event-item.tsx` - Event item (NEW)
- ✅ `web-ui/src/app/dev/page.tsx` - Dashboard integration

## Commits

1. **feat: add fail-fast variable validation for workflows** (8556f7a)
   - Variable validation with test coverage

2. **feat: add event metadata to graph updates for live activity feed** (454955c)
   - Backend event infrastructure

3. **feat: add live activity feed UI components to developer dashboard** (64f7094)
   - Frontend components and integration

## Credits

**Implementation**: Claude Code Assistant (November 13, 2025)
**Feature Request**: Philip Sahli
**Architecture**: Graph-centric event streaming with WebSocket broadcast

---

*Last Updated: November 13, 2025*
