# Health Check and Monitoring Endpoints

## Overview

innominatus provides standard health check and metrics endpoints for integration with monitoring systems, load balancers, and Kubernetes health probes.

## Endpoints

### /health - Liveness Probe

**Purpose**: Indicates whether the service is alive and functioning

**URL**: `GET /health`

**Response Format**: JSON

**Status Codes**:
- `200 OK` - Service is healthy or degraded (can still serve traffic)
- `503 Service Unavailable` - Service is unhealthy (critical dependencies down)

**Response Example**:
```json
{
  "status": "healthy",
  "timestamp": "2025-01-15T10:30:00Z",
  "uptime_seconds": 3600.5,
  "checks": {
    "server": {
      "name": "server",
      "status": "healthy",
      "message": "OK",
      "latency_ms": 0,
      "timestamp": "2025-01-15T10:30:00Z"
    },
    "database": {
      "name": "database",
      "status": "healthy",
      "message": "5 active connections",
      "latency_ms": 2,
      "timestamp": "2025-01-15T10:30:00Z"
    }
  }
}
```

**Health States**:
- `healthy` - All dependencies are functioning normally
- `degraded` - Some non-critical dependencies are impaired (service can still operate)
- `unhealthy` - Critical dependencies are down (service cannot operate properly)

**Use Cases**:
- Kubernetes liveness probes
- Load balancer health checks
- Service mesh health verification
- Monitoring system alerts

### /ready - Readiness Probe

**Purpose**: Indicates whether the service is ready to accept traffic

**URL**: `GET /ready`

**Response Format**: JSON

**Status Codes**:
- `200 OK` - Service is ready to accept traffic
- `503 Service Unavailable` - Service is not ready (still initializing or dependencies unavailable)

**Response Example**:
```json
{
  "ready": true,
  "timestamp": "2025-01-15T10:30:00Z",
  "message": "Service is ready"
}
```

**Use Cases**:
- Kubernetes readiness probes
- Load balancer pool membership
- Rolling deployment health checks
- Traffic routing decisions

**Difference from /health**:
- `/health` checks if the service process is alive
- `/ready` checks if the service can handle requests
- A service can be healthy but not ready (e.g., during initialization)
- A service should be both healthy and ready to receive production traffic

### /metrics - Prometheus Metrics

**Purpose**: Exposes application metrics in Prometheus exposition format

**URL**: `GET /metrics`

**Response Format**: Text (Prometheus format)

**Status Code**: `200 OK`

**Response Example**:
```
# HELP innominatus_build_info Build information
# TYPE innominatus_build_info gauge
innominatus_build_info{version="1.0.0",go_version="go1.21.0"} 1

# HELP innominatus_uptime_seconds Server uptime in seconds
# TYPE innominatus_uptime_seconds gauge
innominatus_uptime_seconds 3600.50

# HELP innominatus_http_requests_total Total HTTP requests
# TYPE innominatus_http_requests_total counter
innominatus_http_requests_total{method="GET",path="/api/specs"} 150
innominatus_http_requests_total{method="POST",path="/api/specs"} 25

# HELP innominatus_workflows_executed_total Total workflow executions
# TYPE innominatus_workflows_executed_total counter
innominatus_workflows_executed_total 42

# HELP innominatus_workflows_succeeded_total Total successful workflow executions
# TYPE innominatus_workflows_succeeded_total counter
innominatus_workflows_succeeded_total 38

# HELP innominatus_workflows_failed_total Total failed workflow executions
# TYPE innominatus_workflows_failed_total counter
innominatus_workflows_failed_total 4

# HELP innominatus_go_goroutines Number of goroutines
# TYPE innominatus_go_goroutines gauge
innominatus_go_goroutines 45

# HELP innominatus_go_memory_alloc_bytes Bytes allocated and in use
# TYPE innominatus_go_memory_alloc_bytes gauge
innominatus_go_memory_alloc_bytes 12582912
```

**Metrics Provided**:

1. **Build Info**
   - `innominatus_build_info` - Version and Go runtime information

2. **Server Metrics**
   - `innominatus_uptime_seconds` - Time since server started

3. **HTTP Metrics**
   - `innominatus_http_requests_total` - Total requests by method and path
   - `innominatus_http_errors_total` - Total 5xx errors by path

4. **Workflow Metrics**
   - `innominatus_workflows_executed_total` - Total workflow executions
   - `innominatus_workflows_succeeded_total` - Successful executions
   - `innominatus_workflows_failed_total` - Failed executions
   - `innominatus_workflow_duration_seconds_avg` - Average duration (last 100)

5. **Database Metrics**
   - `innominatus_db_queries_total` - Total database queries
   - `innominatus_db_query_errors_total` - Database query errors

6. **Go Runtime Metrics**
   - `innominatus_go_goroutines` - Number of goroutines
   - `innominatus_go_memory_alloc_bytes` - Allocated memory
   - `innominatus_go_memory_total_alloc_bytes` - Cumulative allocated memory
   - `innominatus_go_memory_sys_bytes` - Memory from OS
   - `innominatus_go_gc_runs_total` - Garbage collection runs

**Use Cases**:
- Prometheus scraping
- Grafana dashboards
- Performance monitoring
- Capacity planning
- SLI/SLO tracking

## Kubernetes Integration

### Liveness Probe Configuration

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Readiness Probe Configuration

```yaml
readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 2
```

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: innominatus
  namespace: platform
spec:
  selector:
    matchLabels:
      app: innominatus
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

## Health Check Implementation

### Components Checked

1. **Server**: Always healthy (basic process health)
2. **Database**: Connection and query health
   - Ping test with 5-second timeout
   - Connection pool stats
   - Returns `degraded` if not configured
   - Returns `unhealthy` if connection fails

### Adding Custom Health Checks

Health checks can be added during server initialization:

```go
import "innominatus/internal/health"

// Create health checker
healthChecker := health.NewHealthChecker()

// Register custom checker
healthChecker.Register(health.NewDatabaseChecker(db.DB(), 5*time.Second))

// Custom checker implementation
type MyChecker struct {
    name string
}

func (c *MyChecker) Name() string {
    return c.name
}

func (c *MyChecker) Check(ctx context.Context) health.Check {
    // Perform health check logic
    return health.Check{
        Name:      c.name,
        Status:    health.StatusHealthy,
        Message:   "OK",
        Timestamp: time.Now(),
        Latency:   duration,
    }
}
```

## Monitoring Best Practices

### 1. Alert on Unhealthy Status

```yaml
# Prometheus alerting rule
- alert: InnominatusUnhealthy
  expr: up{job="innominatus"} == 0 OR innominatus_health_status == 0
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: "Innominatus service is unhealthy"
```

### 2. Track Workflow Success Rate

```promql
# Success rate over 5 minutes
rate(innominatus_workflows_succeeded_total[5m]) /
rate(innominatus_workflows_executed_total[5m])
```

### 3. Monitor HTTP Error Rate

```promql
# Error rate over 5 minutes
rate(innominatus_http_errors_total[5m]) /
rate(innominatus_http_requests_total[5m])
```

### 4. Database Health Tracking

```promql
# Database health check latency
innominatus_health_check_latency_ms{component="database"}
```

### 5. Memory Growth Detection

```promql
# Memory growth rate
rate(innominatus_go_memory_alloc_bytes[1h])
```

## Troubleshooting

### Health Check Failing

**Symptom**: `/health` returns 503

**Possible Causes**:
1. Database connection lost
2. Critical dependencies unavailable
3. Service initialization not complete

**Resolution**:
```bash
# Check health status
curl http://localhost:8081/health

# Check individual components in response
# Look for "status": "unhealthy" entries
# Check "error" fields for details

# Verify database connectivity
psql -h localhost -U postgres -d idp_orchestrator -c "SELECT 1"

# Check server logs
tail -f /var/log/innominatus.log
```

### Readiness Check Failing

**Symptom**: `/ready` returns 503

**Possible Causes**:
1. Service still initializing
2. Database migrations running
3. Required dependencies not available

**Resolution**:
```bash
# Check readiness status
curl http://localhost:8081/ready

# Wait for initialization
# Check if service becomes ready after a few seconds

# Force restart if stuck
kubectl rollout restart deployment/innominatus
```

### Metrics Not Scraping

**Symptom**: Prometheus not collecting metrics

**Possible Causes**:
1. Incorrect ServiceMonitor configuration
2. Network policy blocking scraping
3. Metrics endpoint not exposed

**Resolution**:
```bash
# Test metrics endpoint
curl http://localhost:8081/metrics

# Check Prometheus targets
# Navigate to Prometheus UI > Status > Targets
# Verify innominatus target is UP

# Check ServiceMonitor
kubectl get servicemonitor innominatus -o yaml

# Verify service labels match selector
kubectl get service innominatus --show-labels
```

## Security Considerations

1. **No Authentication Required**: Health endpoints are intentionally unauthenticated for monitoring systems
2. **Read-Only**: All endpoints only expose status information, no mutations
3. **Rate Limiting**: Consider adding rate limiting if exposed publicly
4. **Network Policies**: Restrict access to monitoring namespaces in Kubernetes

## Example: Complete Monitoring Stack

```yaml
# Deployment with health checks
apiVersion: apps/v1
kind: Deployment
metadata:
  name: innominatus
spec:
  template:
    spec:
      containers:
        - name: innominatus
          image: innominatus:latest
          ports:
            - name: http
              containerPort: 8081
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
---
# Service for scraping
apiVersion: v1
kind: Service
metadata:
  name: innominatus
  labels:
    app: innominatus
spec:
  ports:
    - name: http
      port: 8081
  selector:
    app: innominatus
---
# ServiceMonitor for Prometheus
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: innominatus
spec:
  selector:
    matchLabels:
      app: innominatus
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
```

---

*Last updated: 2025-01-15*
