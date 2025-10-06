# Monitoring Guide

Comprehensive monitoring setup for innominatus including health checks, Prometheus metrics, and Grafana dashboards.

---

## Health Check Endpoints

innominatus provides standard health check and metrics endpoints for integration with monitoring systems, load balancers, and Kubernetes health probes.

### /health - Liveness Probe

**Purpose**: Indicates whether the service is alive and functioning

**URL**: `GET /health`

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

### /ready - Readiness Probe

**Purpose**: Indicates whether the service is ready to accept traffic

**URL**: `GET /ready`

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

**Difference from /health**:
- `/health` checks if the service process is alive
- `/ready` checks if the service can handle requests
- A service can be healthy but not ready (e.g., during initialization)
- A service should be both healthy and ready to receive production traffic

---

## Prometheus Metrics

innominatus exposes metrics at `/metrics` endpoint.

### Metrics Endpoint

```bash
curl http://innominatus.platform.svc:8081/metrics
```

### Key Metrics

**Application Metrics:**
- `innominatus_uptime_seconds` - Server uptime
- `innominatus_workflows_executed_total` - Total workflow executions
- `innominatus_workflows_succeeded_total` - Successful workflows
- `innominatus_workflows_failed_total` - Failed workflows
- `innominatus_workflow_duration_seconds_avg` - Average workflow duration
- `innominatus_http_requests_total` - Total HTTP requests
- `innominatus_db_queries_total` - Total database queries
- `innominatus_build_info` - Build information (version, commit)

**Go Runtime Metrics:**
- `go_goroutines` - Number of goroutines
- `go_memstats_alloc_bytes` - Memory allocated
- `go_gc_duration_seconds` - GC duration

**Process Metrics:**
- `process_cpu_seconds_total` - CPU usage
- `process_resident_memory_bytes` - Memory usage
- `process_open_fds` - Open file descriptors

---

## Prometheus Configuration

### Scrape Configuration

```yaml
scrape_configs:
  - job_name: 'innominatus'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names: ['platform']
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        action: keep
        regex: innominatus
      - source_labels: [__meta_kubernetes_pod_name]
        target_label: pod
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
```

### Pushgateway (Optional)

innominatus can push metrics to Prometheus Pushgateway:

```bash
export PUSHGATEWAY_URL=http://pushgateway.monitoring.svc:9091
```

---

## Grafana Dashboards

### Dashboard 1: Innominatus Platform Metrics

**Panels:**
1. Server Uptime
2. Total Workflows
3. Workflow Success Rate
4. HTTP Requests
5. Workflow Executions (timeseries)
6. Average Workflow Duration
7. Database Queries
8. HTTP Requests & Errors

**Import:**
```bash
kubectl exec -n platform deployment/innominatus -- \
  curl http://localhost:8081/api/grafana/dashboards/innominatus
```

### Dashboard 2: Runtime & Performance

**Panels:**
9. Go Goroutines
10. Memory Allocated
11. Process Resident Memory
12. Build Info
13. GC Duration Rate
14. CPU Usage Rate

---

## Alerting

### Prometheus Alert Rules

```yaml
groups:
  - name: innominatus
    rules:
      - alert: InnominatusDown
        expr: up{job="innominatus"} == 0
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "innominatus is down"

      - alert: HighWorkflowFailureRate
        expr: rate(innominatus_workflows_failed_total[5m]) > 0.1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High workflow failure rate"

      - alert: DatabaseConnectionIssue
        expr: innominatus_db_query_errors_total > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Database connection issues detected"
```

---

## Demo Environment Dashboards

The demo environment automatically installs Grafana dashboards:

```bash
./innominatus-ctl demo-time
```

Access Grafana at http://grafana.localtest.me (admin/admin)

---

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

### Complete Deployment Example

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

## Custom Dashboards

Create your own dashboards using Grafana UI or via API:

```bash
curl -X POST http://grafana.yourcompany.com/api/dashboards/db \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -d @dashboard.json
```

---

## Monitoring Best Practices

### 1. Alert on Unhealthy Status

```yaml
# Prometheus alerting rule
- alert: InnominatusUnhealthy
  expr: up{job="innominatus"} == 0
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

---

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
kubectl logs -n platform deployment/innominatus --tail=100
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
kubectl rollout restart deployment/innominatus -n platform
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
kubectl get servicemonitor innominatus -n platform -o yaml

# Verify service labels match selector
kubectl get service innominatus -n platform --show-labels
```

---

## Security Considerations

1. **No Authentication Required**: Health endpoints are intentionally unauthenticated for monitoring systems
2. **Read-Only**: All endpoints only expose status information, no mutations
3. **Rate Limiting**: Consider adding rate limiting if exposed publicly
4. **Network Policies**: Restrict access to monitoring namespaces in Kubernetes

---

## Next Steps

- **[Operations Guide](operations.md)** - Troubleshooting and scaling
- **[Quick Install](quick-install.md)** - Production deployment guide
- **[Authentication](authentication.md)** - OIDC and security setup

---

**See Also:**
- [Prometheus Metrics Documentation](https://prometheus.io/docs/)
- [Grafana Dashboard Guide](https://grafana.com/docs/grafana/latest/dashboards/)
- [Kubernetes Health Checks](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
