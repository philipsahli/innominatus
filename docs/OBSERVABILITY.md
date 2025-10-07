# Observability

This document describes the observability features in innominatus, including structured logging, distributed tracing, and metrics collection.

## Table of Contents

- [Overview](#overview)
- [Structured Logging](#structured-logging)
- [Distributed Tracing](#distributed-tracing)
- [Metrics](#metrics)
- [Configuration](#configuration)
- [Integration Examples](#integration-examples)

## Overview

innominatus provides comprehensive observability through three pillars:

1. **Structured Logging**: JSON-formatted logs with trace correlation
2. **Distributed Tracing**: OpenTelemetry-based request tracing
3. **Metrics**: Prometheus metrics with Pushgateway integration

## Structured Logging

### Log Formats

innominatus supports three logging formats configured via the `LOG_FORMAT` environment variable:

- **`json`** (Production): Machine-parseable JSON logs
- **`console`**: Plain text logs without colors
- **`pretty`** (Default): Human-readable colored logs with emojis

### Log Levels

Configure log verbosity via `LOG_LEVEL` environment variable:

- `DEBUG`: Detailed debugging information
- `INFO`: General informational messages (default)
- `WARN`: Warning messages
- `ERROR`: Error messages
- `FATAL`: Critical errors (exits application)

### Example Configuration

```bash
# Production JSON logging
export LOG_FORMAT=json
export LOG_LEVEL=info
./innominatus

# Development pretty logging with debug
export LOG_FORMAT=pretty
export LOG_LEVEL=debug
./innominatus
```

### JSON Log Format

```json
{
  "level": "info",
  "component": "workflow",
  "message": "Workflow execution started",
  "trace_id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "request_id": "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "app.name": "my-app",
  "workflow.name": "deploy",
  "workflow.execution_id": 123,
  "timestamp": "2025-10-06T10:30:00Z"
}
```

### Using Structured Logging

#### In Application Code

```go
import "innominatus/internal/logging"

// Create a structured logger
logger := logging.NewStructuredLogger("my-component")

// Log with fields
logger.InfoWithFields("Operation completed", map[string]interface{}{
    "duration_ms": 150,
    "status": "success",
})

// Log errors with context
logger.ErrorWithError("Failed to process request", err)
```

#### Context-Aware Logging

```go
import "innominatus/internal/logging"

// Create context logger (auto-populates trace_id, request_id, user_id)
ctx := r.Context() // from HTTP request
logger := logging.NewContextLogger(ctx, "handler")

logger.Info("Processing request")  // Automatically includes trace_id
```

## Distributed Tracing

innominatus uses OpenTelemetry for distributed tracing, providing visibility into request flows across the entire platform.

### Enabling Tracing

```bash
# Enable OpenTelemetry tracing
export OTEL_ENABLED=true

# Configure OTLP endpoint (default: http://localhost:4318)
export OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo.monitoring.svc.cluster.local:4318

# Set service name and version
export OTEL_SERVICE_NAME=innominatus
export OTEL_SERVICE_VERSION=v0.1.0

# Start server
./innominatus
```

### Trace ID Propagation

Every HTTP request receives a unique trace ID:

**Request Header:**
```
X-Trace-Id: <provided-trace-id>  # If provided by client
```

**Response Header:**
```
X-Trace-Id: a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
```

The trace ID is:
1. Extracted from `X-Trace-Id` request header (if provided)
2. Derived from OpenTelemetry span context (if tracing enabled)
3. Generated as a new UUID (fallback)

### Trace Context

All traces include these attributes:

**HTTP Spans:**
- `http.method`: HTTP method (GET, POST, etc.)
- `http.url`: Full request URL
- `http.host`: Request host
- `http.target`: Request path
- `http.status_code`: Response status code
- `http.user_agent`: Client user agent
- `http.client_ip`: Client IP address
- `http.error`: true if status >= 400

**Workflow Spans:**
- `app.name`: Application name
- `workflow.name`: Workflow name
- `workflow.steps`: Number of steps
- `workflow.execution_id`: Database execution ID

### Sampling

Sampling is configurable based on environment:

**Development (default):**
```bash
# Always sample all traces
ENV=development
```

**Production:**
```bash
# Sample 10% of traces
ENV=production
OTEL_TRACE_SAMPLE_RATE=0.1
```

### Integration with Tempo/Jaeger

innominatus exports traces via OTLP HTTP to compatible backends:

**Grafana Tempo:**
```bash
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318
```

**Jaeger (with OTLP support):**
```bash
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger-collector:4318
```

**Cloud Services:**
```bash
# Honeycomb
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=https://api.honeycomb.io:443
export OTEL_EXPORTER_OTLP_HEADERS="x-honeycomb-team=YOUR_API_KEY"

# New Relic
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp.nr-data.net:4318
export OTEL_EXPORTER_OTLP_HEADERS="api-key=YOUR_LICENSE_KEY"
```

## Metrics

innominatus exposes Prometheus metrics and pushes them to Pushgateway.

### Metrics Endpoint

```bash
curl http://localhost:8081/metrics
```

### Available Metrics

**Application Metrics:**
- `innominatus_uptime_seconds`: Server uptime
- `innominatus_workflows_total`: Total workflow executions
- `innominatus_workflows_succeeded_total`: Successful workflows
- `innominatus_workflows_failed_total`: Failed workflows
- `innominatus_http_requests_total`: Total HTTP requests
- `innominatus_http_request_errors_total`: HTTP errors
- `innominatus_database_queries_total`: Database queries
- `innominatus_database_query_errors_total`: Database errors

**Go Runtime Metrics:**
- `go_goroutines`: Number of goroutines
- `go_memstats_alloc_bytes`: Allocated memory
- `go_memstats_sys_bytes`: System memory
- `go_gc_duration_seconds`: GC pause duration

### Pushgateway Integration

```bash
# Configure Pushgateway
export PUSHGATEWAY_URL=http://pushgateway.monitoring.svc.cluster.local

# Disable Pushgateway
export PUSHGATEWAY_URL=disabled

# Start server (pushes metrics every 15 seconds)
./innominatus
```

### Grafana Dashboard

Import the pre-built dashboard:
```bash
kubectl apply -f docs/grafana-dashboard-innominatus.json
```

Or access the demo Grafana:
```
http://grafana.localtest.me
```

Dashboard includes:
- Server uptime and version
- Workflow execution rates (total, success, failure)
- HTTP request rates and error rates
- Database query performance
- Go runtime metrics (goroutines, memory, GC)

## Configuration

### Complete Environment Variables

```bash
# Logging Configuration
export LOG_LEVEL=info                    # debug, info, warn, error, fatal
export LOG_FORMAT=json                   # json, console, pretty
export ENV=production                    # Affects default log format

# OpenTelemetry Tracing
export OTEL_ENABLED=true                 # Enable distributed tracing
export OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318
export OTEL_SERVICE_NAME=innominatus
export OTEL_SERVICE_VERSION=v0.1.0
export OTEL_TRACE_SAMPLE_RATE=0.1       # Sample 10% in production

# Metrics
export PUSHGATEWAY_URL=http://pushgateway:9091
```

### Production Recommended Settings

```bash
# Structured JSON logs
export LOG_FORMAT=json
export LOG_LEVEL=info

# Distributed tracing with sampling
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo.monitoring.svc:4318
export OTEL_TRACE_SAMPLE_RATE=0.1

# Metrics with Pushgateway
export PUSHGATEWAY_URL=http://pushgateway.monitoring.svc:9091
```

### Development Settings

```bash
# Pretty console logs with debug level
export LOG_FORMAT=pretty
export LOG_LEVEL=debug

# Full trace sampling for debugging
export OTEL_ENABLED=true
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
# No sample rate = 100% sampling

# Local Pushgateway
export PUSHGATEWAY_URL=http://localhost:9091
```

## Integration Examples

### Log Aggregation with Loki

**1. Deploy Loki Stack:**
```bash
helm repo add grafana https://grafana.github.io/helm-charts
helm install loki grafana/loki-stack -n monitoring
```

**2. Configure Promtail to scrape logs:**
```yaml
# promtail-config.yaml
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: innominatus
    static_configs:
      - targets:
          - localhost
        labels:
          job: innominatus
          __path__: /var/log/innominatus/*.log
```

**3. Run innominatus with JSON logs:**
```bash
export LOG_FORMAT=json
./innominatus > /var/log/innominatus/app.log
```

### Full Observability Stack (Demo)

Deploy complete observability stack with demo environment:

```bash
# Start demo environment with all observability components
./innominatus-ctl demo-time
```

This includes:
- **Prometheus**: Metrics collection (http://prometheus.localtest.me)
- **Grafana**: Dashboards (http://grafana.localtest.me)
- **Pushgateway**: Metrics push endpoint
- Pre-configured Grafana dashboard for innominatus

### Kubernetes Deployment

**With Tempo and Loki:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: innominatus
spec:
  template:
    spec:
      containers:
      - name: innominatus
        image: ghcr.io/philipsahli/innominatus:latest
        env:
        - name: LOG_FORMAT
          value: "json"
        - name: LOG_LEVEL
          value: "info"
        - name: OTEL_ENABLED
          value: "true"
        - name: OTEL_EXPORTER_OTLP_ENDPOINT
          value: "http://tempo.monitoring:4318"
        - name: OTEL_SERVICE_NAME
          value: "innominatus"
        - name: PUSHGATEWAY_URL
          value: "http://pushgateway.monitoring:9091"
```

### Querying Traces in Grafana

**Find traces by trace ID:**
```
{traceID="a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"}
```

**Find slow workflow executions:**
```
{service.name="innominatus"}
  | duration > 5s
  | select(workflow.name, duration)
```

**Find failed HTTP requests:**
```
{service.name="innominatus" http.error=true}
  | select(http.method, http.target, http.status_code)
```

### Querying Logs in Loki

**All innominatus logs:**
```
{job="innominatus"}
```

**Error logs with trace ID:**
```
{job="innominatus"}
  | json
  | level="error"
  | line_format "{{.trace_id}}: {{.message}}"
```

**Workflow execution logs:**
```
{job="innominatus"}
  | json
  | workflow_name=~".+"
  | line_format "{{.workflow_name}} ({{.workflow_execution_id}}): {{.message}}"
```

## Best Practices

### 1. Always Include Trace IDs

```go
// Get trace ID from context
traceID := logging.GetTraceID(ctx)

// Include in error responses
w.Header().Set("X-Trace-Id", traceID)
```

### 2. Use Structured Logging

```go
// ❌ Bad
fmt.Printf("User %s logged in", username)

// ✅ Good
logger.InfoWithFields("User logged in", map[string]interface{}{
    "username": username,
    "ip": clientIP,
})
```

### 3. Add Context to Spans

```go
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.String("team.id", teamID),
)
```

### 4. Record Errors in Spans

```go
if err != nil {
    span.RecordError(err)
    span.SetAttributes(attribute.Bool("error", true))
    return err
}
```

### 5. Use Correlation

Link logs to traces using trace IDs:
```go
logger := logging.NewContextLogger(ctx, "component")
logger.Info("Message")  // Automatically includes trace_id
```

## Troubleshooting

### Logs Not Appearing

1. Check log level: `export LOG_LEVEL=debug`
2. Verify log format: `export LOG_FORMAT=json`
3. Check output: Logs go to stdout by default

### Traces Not Exported

1. Verify OTEL is enabled: `export OTEL_ENABLED=true`
2. Check endpoint: `export OTEL_EXPORTER_OTLP_ENDPOINT=http://tempo:4318`
3. Verify network connectivity to OTLP endpoint
4. Check Tempo/Jaeger is receiving traces

### Metrics Not Available

1. Verify metrics endpoint: `curl http://localhost:8081/metrics`
2. Check Pushgateway URL: `echo $PUSHGATEWAY_URL`
3. Verify Pushgateway is reachable
4. Check Prometheus scrape configuration

## See Also

- [Health Monitoring](HEALTH_MONITORING.md)
- [Metrics Documentation](../internal/metrics/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Grafana Tempo](https://grafana.com/docs/tempo/latest/)
- [Grafana Loki](https://grafana.com/docs/loki/latest/)
