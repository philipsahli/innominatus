# Operations Guide

Monitor, scale, and troubleshoot innominatus in production.

---

## Health Monitoring

### Health Endpoints

```bash
# Liveness probe
curl http://innominatus.platform.svc:8081/health

# Readiness probe
curl http://innominatus.platform.svc:8081/ready

# Metrics
curl http://innominatus.platform.svc:8081/metrics
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Scaling

### Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: innominatus-hpa
  namespace: platform
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: innominatus
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Manual Scaling

```bash
kubectl scale deployment innominatus -n platform --replicas=5
```

---

## Logging

### View Logs

```bash
# Tail logs
kubectl logs -n platform deployment/innominatus --tail=100 -f

# Filter by workflow
kubectl logs -n platform deployment/innominatus | grep workflow_id=wf-123
```

### Structured Logging

innominatus logs in JSON format:

```json
{
  "level": "info",
  "timestamp": "2025-10-06T08:00:00Z",
  "workflow_id": "wf-abc123",
  "app_name": "my-app",
  "step": "kubernetes-deploy",
  "message": "Deployment successful"
}
```

---

## Backup & Recovery

### Database Backup

```bash
# Manual backup
pg_dump -h $DB_HOST -U $DB_USER -d idp_orchestrator \
  -F c -f backup-$(date +%Y%m%d).dump

# Restore
pg_restore -h $DB_HOST -U $DB_USER -d idp_orchestrator \
  -c backup-20251006.dump
```

### Configuration Backup

```bash
# Backup ConfigMaps and Secrets
kubectl get configmap innominatus-config -n platform -o yaml > config-backup.yaml
kubectl get secret innominatus-secrets -n platform -o yaml > secrets-backup.yaml
```

---

## Troubleshooting

### Pod Not Starting

```bash
kubectl describe pod -n platform -l app=innominatus
kubectl logs -n platform -l app=innominatus --previous
```

### Database Connection Failed

```bash
# Test database connectivity from pod
kubectl exec -n platform deployment/innominatus -- \
  psql -h $DB_HOST -U $DB_USER -d idp_orchestrator -c "SELECT 1;"
```

### Workflow Stuck

```bash
# Query workflow status in database
kubectl exec -n platform deployment/innominatus -- \
  psql -h $DB_HOST -U $DB_USER -d idp_orchestrator \
  -c "SELECT id, app_name, status, started_at FROM workflow_executions WHERE status='running' ORDER BY started_at DESC;"
```

### High CPU/Memory Usage

```bash
# Check resource usage
kubectl top pod -n platform -l app=innominatus

# Adjust resource limits
kubectl edit deployment innominatus -n platform
```

---

## Security

### Update Secrets

```bash
# Rotate database password
kubectl create secret generic innominatus-db-new \
  --namespace platform \
  --from-literal=password=new-secure-password

# Update deployment
kubectl set env deployment/innominatus -n platform \
  DB_PASSWORD_SECRET=innominatus-db-new
```

### Security Scanning

```bash
# Scan container image
trivy image ghcr.io/philipsahli/innominatus:latest

# Check for vulnerabilities
kubectl kube-bench run --targets node,policies
```

---

## Maintenance Windows

### Rolling Update

```bash
# Update image
kubectl set image deployment/innominatus -n platform \
  server=ghcr.io/philipsahli/innominatus:v1.2.0

# Monitor rollout
kubectl rollout status deployment/innominatus -n platform
```

### Rollback

```bash
# Rollback to previous version
kubectl rollout undo deployment/innominatus -n platform
```

---

## Monitoring

**See:** [Monitoring Guide](monitoring.md) for Prometheus and Grafana setup.

---

## Next Steps

- **[Monitoring](monitoring.md)** - Set up metrics and dashboards
- **[Configuration](configuration.md)** - Update OIDC and RBAC settings
- **[Database](database.md)** - Database maintenance

---

**See Also:**
- [Health Monitoring Guide](../HEALTH_MONITORING.md)
- [Grafana Dashboard Guide](../demo/grafana.go)
