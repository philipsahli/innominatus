# Monitoring Guide

Set up Prometheus metrics and Grafana dashboards for innominatus.

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

## Custom Dashboards

Create your own dashboards using Grafana UI or via API:

```bash
curl -X POST http://grafana.yourcompany.com/api/dashboards/db \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $GRAFANA_API_KEY" \
  -d @dashboard.json
```

---

## Next Steps

- **[Operations Guide](operations.md)** - Troubleshooting and scaling
- **[Health Monitoring](../HEALTH_MONITORING.md)** - Detailed health check documentation

---

**See Also:**
- [Prometheus Metrics Documentation](https://prometheus.io/docs/)
- [Grafana Dashboard Guide](https://grafana.com/docs/grafana/latest/dashboards/)
