# Watch Feature - Real-Time Deployment Streaming

## Overview

The watch feature provides real-time event streaming for deployment operations, similar to `kubectl apply --watch`. It addresses the user's feedback that the system was "too complex to understand what's happening" by providing live updates on deployment progress.

## User Experience

### Simple Deployment (Without Watch)
```bash
innominatus-ctl deploy myapp.yaml
```
Output:
```
📤 Submitting Score specification: myapp
✅ Spec submitted successfully!

To watch deployment progress, use:
  innominatus-ctl deploy myapp.yaml -w
```

### Real-Time Watch Mode
```bash
innominatus-ctl deploy myapp.yaml -w
```
Output:
```
📤 Submitting Score specification: myapp
✅ Spec submitted, starting watch mode...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🔍 Watching deployment: myapp
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

[15:04:01] 📝 Resource created: database (postgres)
[15:04:01] ⏳ Provisioning resource: database (postgres)
[15:04:02] 🔍 Provider resolved: database-team for postgres (workflow: provision-postgres)
[15:04:02] 🚀 Workflow started: provision-postgres (3 steps)
[15:04:05] 🟢 Resource active: database
[15:04:05] ✅ Workflow completed: provision-postgres

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Deployment completed in 5s
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

### Verbose Mode
```bash
innominatus-ctl deploy myapp.yaml -w --verbose
```
Shows additional event details like resource IDs, provider names, and workflow metadata.

## Architecture

### Backend Components

1. **Event System** (`internal/events/`)
   - `types.go` - Event type definitions (spec.*, resource.*, workflow.*, step.*, deployment.*)
   - `bus.go` - Channel-based pub/sub event bus
   - `sse.go` - Server-Sent Events (SSE) broker for HTTP streaming

2. **Event Publishers**
   - **Orchestration Engine** - Publishes provider resolution, resource provisioning events
   - **Resource Manager** - Publishes resource lifecycle state transitions
   - **Workflow Executor** - Publishes workflow and step execution events

3. **SSE Endpoint**
   - `GET /api/events/stream?app={appName}` - Stream events for specific app
   - `GET /api/events/stream` - Stream all events (admin)

### Frontend Components

1. **SSE Client** (`internal/client/sse.go`)
   - Connects to SSE endpoint
   - Handles reconnection and error recovery
   - Filters events by app name

2. **Event Formatter** (`internal/client/formatter.go`)
   - Formats events for CLI display
   - Provides icons for different event types
   - Supports verbose and show-all modes

3. **Deploy Command** (`cmd/cli/deploy.go`)
   - Submits Score spec to server
   - Optionally streams deployment events with `-w/--watch`
   - Shows formatted real-time progress

## Event Types

### Resource Events
- `resource.created` - Resource instance created in database
- `resource.requested` - Resource state set to requested
- `resource.provisioning` - Provider workflow started
- `resource.active` - Resource provisioned successfully
- `resource.failed` - Resource provisioning failed

### Workflow Events
- `workflow.started` - Workflow execution started
- `workflow.completed` - Workflow completed successfully
- `workflow.failed` - Workflow failed

### Provider Events
- `provider.resolved` - Provider matched to resource type

### Step Events (Future)
- `step.started` - Workflow step started
- `step.completed` - Step completed
- `step.failed` - Step failed
- `step.progress` - Step progress update

### Deployment Events (Future)
- `deployment.started` - Overall deployment started
- `deployment.completed` - All resources provisioned
- `deployment.failed` - Deployment failed

## Implementation Details

### Event Bus
- **Pattern**: Channel-based pub/sub with goroutine-per-subscriber
- **Buffering**: 256-event buffer per subscriber (prevents blocking)
- **Filtering**: By app name and event types
- **Lifecycle**: Graceful shutdown with proper cleanup
- **Thread Safety**: RWMutex for concurrent access

### SSE Broker
- **Protocol**: Server-Sent Events (HTTP streaming)
- **Format**: JSON events with `data:` prefix
- **Keepalive**: 30-second pings to keep connections alive
- **Disconnection**: Automatic cleanup on client disconnect
- **Filtering**: Per-client app name and event type filters

### CLI Watch
- **Timeout**: Default 10 minutes (configurable)
- **Formatting**: Icon-based visual feedback
- **Verbose Mode**: Shows detailed event metadata
- **Error Handling**: Graceful failure with error messages

## Usage Examples

### Basic Deploy with Watch
```bash
innominatus-ctl deploy score.yaml -w
```

### Verbose Watch
```bash
innominatus-ctl deploy score.yaml -w --verbose
```

### Show All Events (Including Internal)
```bash
innominatus-ctl deploy score.yaml -w --all
```

### Custom Timeout
```bash
innominatus-ctl deploy score.yaml -w --timeout 30m
```

## Testing

### Unit Tests
```bash
# Test event bus
go test ./internal/events/

# Test SSE client
go test ./internal/client/
```

### Integration Test
1. Start server: `./innominatus`
2. Deploy with watch: `./innominatus-ctl deploy examples/score.yaml -w`
3. Observe real-time events in terminal

### Manual SSE Test
```bash
# Connect to SSE endpoint directly
curl -N -H "Accept: text/event-stream" \
  http://localhost:8081/api/events/stream?app=myapp
```

## Benefits

### For Users
- **Visibility**: See exactly what's happening during deployment
- **Debugging**: Immediate feedback on failures with error messages
- **Simplicity**: Single command with `-w` flag (like kubectl)
- **Familiar UX**: Icon-based progress indicators

### For Operators
- **Monitoring**: Real-time view of platform activity
- **Troubleshooting**: Live event stream for debugging
- **Observability**: Complete audit trail of operations

## Future Enhancements

1. **Web UI Integration** - Live dashboard with event streaming
2. **Event Filtering** - Advanced filters (event types, sources, severity)
3. **Event History** - Query historical events from database
4. **Webhook Integration** - Send events to external systems
5. **Metrics** - Prometheus metrics from events
6. **Step Progress** - Detailed workflow step execution progress
7. **Multi-App Watch** - Watch multiple apps simultaneously
8. **Event Replay** - Replay events for debugging

## Files Created/Modified

### New Files
- `internal/events/types.go` - Event type definitions
- `internal/events/bus.go` - EventBus implementation
- `internal/events/bus_test.go` - EventBus unit tests
- `internal/events/sse.go` - SSE broker
- `internal/client/sse.go` - SSE client
- `internal/client/formatter.go` - Event formatter
- `internal/client/helpers.go` - Client utilities
- `cmd/cli/deploy.go` - Deploy command with watch
- `docs/WATCH_FEATURE.md` - This document

### Modified Files
- `cmd/server/main.go` - EventBus and SSE broker initialization
- `internal/server/handlers.go` - SSE broker integration
- `internal/orchestration/engine.go` - Event publishing
- `internal/resources/manager.go` - Event publishing
- `internal/workflow/executor.go` - Event publishing
- `internal/cli/client.go` - Helper methods

## Performance Considerations

- **Event Bus**: Non-blocking publish (drops events if subscriber is slow)
- **SSE Broker**: Buffered channels prevent blocking the event bus
- **Network**: SSE uses HTTP/1.1 chunked transfer encoding (efficient)
- **Memory**: Minimal overhead (256 events × ~1KB = ~256KB per subscriber)
- **CPU**: Goroutine-per-subscriber scales to hundreds of clients

## Security

- **Authentication**: SSE endpoint requires valid API key/session
- **Authorization**: App-level filtering (users only see their apps)
- **Rate Limiting**: Standard server rate limits apply
- **CORS**: Enabled for web UI integration

## Conclusion

The watch feature transforms the deployment experience from "submit and wait" to "submit and watch", providing real-time visibility into the orchestration process. This aligns with the user's desire for simplicity: **one command** (`deploy -w`) that shows **exactly what's happening**.
