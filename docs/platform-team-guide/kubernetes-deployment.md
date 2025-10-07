# Kubernetes Deployment Guide

**Complete guide for platform engineers to deploy innominatus on Kubernetes**

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Installation Options](#installation-options)
- [Configuration](#configuration)
- [Database Setup](#database-setup)
- [OIDC Authentication](#oidc-authentication)
- [demo-time in Kubernetes](#demo-time-in-kubernetes)
- [Monitoring & Operations](#monitoring--operations)
- [Security](#security)
- [Troubleshooting](#troubleshooting)
- [Upgrade Guide](#upgrade-guide)

---

## Overview

innominatus can be deployed to Kubernetes using Helm. The Helm chart includes:

- **innominatus server** deployment with configurable replicas
- **PostgreSQL database** (optional, via Bitnami subchart)
- **RBAC configuration** for demo-time operations
- **Ingress support** for external access
- **Persistent storage** for workspaces
- **Health checks** and monitoring endpoints

**Key Features:**
- Production-ready defaults
- High availability support
- External database integration
- OIDC/SSO authentication
- Automatic Kubernetes mode detection

---

## Prerequisites

### Required

✅ **Kubernetes cluster** (1.24+)
- Minimum 2 CPUs, 4GB RAM per node
- Storage provisioner for PVCs

✅ **kubectl** configured to access your cluster
```bash
kubectl version --client
kubectl cluster-info
```

✅ **Helm 3.x** installed
```bash
helm version
```

### Optional

- **Ingress controller** (nginx recommended) for external access
- **cert-manager** for TLS certificate management
- **External PostgreSQL** database for production deployments

---

## Quick Start

### 1. Add Helm Repository (Future)

```bash
# Once published
helm repo add innominatus https://innominatus.github.io/charts
helm repo update
```

### 2. Install from Local Chart

```bash
# From the innominatus repository root
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace
```

### 3. Verify Installation

```bash
# Check pod status
kubectl get pods -n innominatus-system

# Check service
kubectl get svc -n innominatus-system

# View logs
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus -f
```

### 4. Access innominatus

**Port-forward (local access):**
```bash
kubectl port-forward -n innominatus-system svc/innominatus 8081:8081
```

Then access http://localhost:8081

**Ingress (if enabled):**
```bash
kubectl get ingress -n innominatus-system
```

Access via the configured ingress host.

---

## Installation Options

### Option 1: With Bundled PostgreSQL (Default)

**Best for:** Development, testing, proof-of-concept

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set postgresql.auth.password=myStrongPassword123
```

**Features:**
- PostgreSQL deployed as subchart
- Automatic database configuration
- Persistent storage for database
- Simple setup, single Helm release

### Option 2: With External Database (Production)

**Best for:** Production, enterprise deployments

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set postgresql.enabled=false \
  --set externalDatabase.enabled=true \
  --set externalDatabase.host=postgres.example.com \
  --set externalDatabase.port=5432 \
  --set externalDatabase.user=innominatus \
  --set externalDatabase.password=secretPassword \
  --set externalDatabase.database=idp_orchestrator
```

**Features:**
- Use existing PostgreSQL (managed service or on-prem)
- Independent database lifecycle
- Better for high availability
- Simplified backup/restore

### Option 3: High Availability Setup

**Best for:** Production with high availability requirements

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set replicaCount=3 \
  --set postgresql.enabled=false \
  --set externalDatabase.enabled=true \
  --set externalDatabase.host=postgres-ha.example.com \
  --set resources.requests.memory=512Mi \
  --set resources.requests.cpu=500m \
  --set autoscaling.enabled=true \
  --set autoscaling.minReplicas=3 \
  --set autoscaling.maxReplicas=10
```

**Features:**
- Multiple replicas for redundancy
- Horizontal Pod Autoscaler
- External HA database
- Production resource limits

### Option 4: With OIDC/SSO

**Best for:** Enterprise with SSO requirements

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set env.OIDC_ENABLED=true \
  --set env.OIDC_ISSUER=https://keycloak.example.com/realms/production \
  --set env.OIDC_CLIENT_ID=innominatus \
  --set env.OIDC_CLIENT_SECRET=your-client-secret \
  --set env.OIDC_REDIRECT_URL=https://innominatus.example.com/auth/callback
```

### Option 5: With Custom Ingress

**Best for:** Production with TLS and custom domains

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set ingress.hosts[0].host=innominatus.example.com \
  --set ingress.hosts[0].paths[0].path=/ \
  --set ingress.hosts[0].paths[0].pathType=Prefix \
  --set ingress.tls[0].secretName=innominatus-tls \
  --set ingress.tls[0].hosts[0]=innominatus.example.com
```

---

## Configuration

### Production values.yaml Example

Create a `production-values.yaml` file:

```yaml
# Production Configuration for innominatus

replicaCount: 3

image:
  repository: ghcr.io/philipsahli/innominatus
  pullPolicy: IfNotPresent
  tag: "0.1.0"

# Service configuration
service:
  type: ClusterIP
  port: 8081

# Ingress with TLS
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: innominatus.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: innominatus-tls
      hosts:
        - innominatus.example.com

# Resources (production)
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

# Autoscaling
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

# PostgreSQL disabled (use external)
postgresql:
  enabled: false

# External database
externalDatabase:
  enabled: true
  host: postgres-prod.example.com
  port: 5432
  user: innominatus
  database: idp_orchestrator
  password: "CHANGE-ME-IN-PRODUCTION"
  # Or use existing secret:
  # existingSecret: innominatus-db-credentials

# OIDC Authentication
env:
  RUNNING_IN_KUBERNETES: "true"
  PORT: "8081"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  OIDC_ENABLED: "true"
  OIDC_ISSUER: "https://keycloak.example.com/realms/production"
  OIDC_CLIENT_ID: "innominatus"
  OIDC_CLIENT_SECRET: "CHANGE-ME"
  OIDC_REDIRECT_URL: "https://innominatus.example.com/auth/callback"

# RBAC for demo-time
rbac:
  create: true
  clusterRole: true

# Persistence for workspaces
persistence:
  enabled: true
  storageClass: "fast-ssd"
  size: 50Gi

# Pod disruption budget
podDisruptionBudget:
  enabled: true
  minAvailable: 2

# Service monitor for Prometheus
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
```

**Install with production values:**

```bash
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --create-namespace \
  -f production-values.yaml
```

---

## Database Setup

### Option 1: Bundled PostgreSQL (Subchart)

The default installation includes PostgreSQL via the Bitnami subchart.

**Configuration:**
```yaml
postgresql:
  enabled: true
  auth:
    username: innominatus
    password: "strongPassword123"  # CHANGE THIS
    database: idp_orchestrator
    postgresPassword: "postgresPassword"  # CHANGE THIS
  primary:
    persistence:
      enabled: true
      size: 20Gi
      storageClass: ""  # Use default
    resources:
      requests:
        memory: "512Mi"
        cpu: "500m"
```

**Access the database:**
```bash
# Get password
export POSTGRES_PASSWORD=$(kubectl get secret -n innominatus-system innominatus-postgresql -o jsonpath="{.data.password}" | base64 -d)

# Connect
kubectl exec -it -n innominatus-system innominatus-postgresql-0 -- psql -U innominatus -d idp_orchestrator
```

### Option 2: External PostgreSQL

**Prerequisites:**
- PostgreSQL 15+ database instance
- Database created: `idp_orchestrator`
- User with full privileges

**Create database:**
```sql
CREATE DATABASE idp_orchestrator;
CREATE USER innominatus WITH PASSWORD 'your-secure-password';
GRANT ALL PRIVILEGES ON DATABASE idp_orchestrator TO innominatus;
```

**Helm configuration:**
```yaml
postgresql:
  enabled: false

externalDatabase:
  enabled: true
  host: postgres.example.com
  port: 5432
  user: innominatus
  password: "your-secure-password"
  database: idp_orchestrator
```

**Using Kubernetes Secret:**
```bash
# Create secret
kubectl create secret generic innominatus-db-credentials \
  -n innominatus-system \
  --from-literal=password=your-secure-password

# Reference in values.yaml
externalDatabase:
  existingSecret: innominatus-db-credentials
```

### Database Migration

innominatus automatically applies database migrations on startup. The server will:
1. Connect to the database
2. Create tables if they don't exist
3. Apply any pending migrations
4. Start serving requests

**Monitor migration:**
```bash
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus -f | grep -i migration
```

---

## OIDC Authentication

### Setup with Keycloak

**1. Create OIDC Client in Keycloak**

- Client ID: `innominatus`
- Client Protocol: `openid-connect`
- Access Type: `confidential`
- Valid Redirect URIs: `https://innominatus.example.com/auth/callback`
- Web Origins: `https://innominatus.example.com`

**2. Configure Helm Values**

```yaml
env:
  OIDC_ENABLED: "true"
  OIDC_ISSUER: "https://keycloak.example.com/realms/production"
  OIDC_CLIENT_ID: "innominatus"
  OIDC_CLIENT_SECRET: "client-secret-from-keycloak"
  OIDC_REDIRECT_URL: "https://innominatus.example.com/auth/callback"
```

**3. Install/Upgrade**

```bash
helm upgrade --install innominatus ./charts/innominatus \
  -n innominatus-system \
  -f values-with-oidc.yaml
```

**4. Test Login**

Access: `https://innominatus.example.com/auth/oidc/login`

---

## demo-time in Kubernetes

When innominatus runs in Kubernetes, `demo-time` operates in **Kubernetes mode**.

### Automatic K8s Detection

innominatus automatically detects Kubernetes deployment via:
1. `RUNNING_IN_KUBERNETES=true` environment variable (set by Helm chart)
2. Kubernetes service account token presence
3. `KUBERNETES_SERVICE_HOST` environment variable

### Behavior Changes in K8s Mode

| Feature | Local Mode | Kubernetes Mode |
|---------|-----------|-----------------|
| **Component URLs** | localhost addresses | K8s service DNS names |
| **Database** | localhost:5432 | K8s service DNS |
| **Kube Context** | docker-desktop | In-cluster config |
| **RBAC** | Local kubectl | ServiceAccount with ClusterRole |

### Running demo-time in K8s

**Execute demo-time:**
```bash
# Get pod name
POD=$(kubectl get pod -n innominatus-system -l app.kubernetes.io/name=innominatus -o jsonpath='{.items[0].metadata.name}')

# Run demo-time
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-time
```

**Check demo status:**
```bash
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-status
```

**Clean up demo:**
```bash
kubectl exec -it -n innominatus-system $POD -- /app/innominatus-ctl demo-nuke
```

### RBAC Requirements for demo-time

demo-time requires ClusterRole permissions to install components. The Helm chart creates:

**ClusterRole permissions:**
- Namespaces: create, list, delete
- Deployments, StatefulSets: create, update, delete
- Services, ConfigMaps, Secrets: create, update, delete
- Ingresses: create, update, delete
- RBAC resources: create for component service accounts
- ArgoCD CRDs: create, update (for Applications)

**Verify RBAC:**
```bash
kubectl get clusterrole innominatus
kubectl get clusterrolebinding innominatus
```

---

## Monitoring & Operations

### Health Checks

innominatus exposes three health endpoints:

| Endpoint | Purpose | Kubernetes Use |
|----------|---------|----------------|
| `/health` | Liveness probe | Restart unhealthy pods |
| `/ready` | Readiness probe | Route traffic only when ready |
| `/metrics` | Prometheus metrics | Monitoring and alerting |

**Configured in Deployment:**
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 5
```

### Prometheus Metrics

innominatus exposes metrics at `/metrics` in Prometheus format.

**Key metrics:**
- `innominatus_http_requests_total` - Total HTTP requests
- `innominatus_workflows_executed_total` - Workflow executions
- `innominatus_resources_total{type="..."}` - Resources by type
- `innominatus_db_queries_total` - Database queries

**ServiceMonitor (Prometheus Operator):**
```yaml
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
```

**Manual scrape config:**
```yaml
scrape_configs:
  - job_name: 'innominatus'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - innominatus-system
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        regex: innominatus
        action: keep
```

### Grafana Dashboard

Access the Grafana dashboard included in demo-time:
```bash
http://grafana.localtest.me
```

Pre-configured panels show:
- HTTP request rate and errors
- Workflow execution metrics
- Database query performance
- Resource counts by type

### Logging

**View logs:**
```bash
# All pods
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus -f

# Specific pod
kubectl logs -n innominatus-system innominatus-xxxxx -f

# JSON logs
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus | jq .
```

**Log format (JSON):**
```json
{
  "level": "info",
  "timestamp": "2025-10-06T15:30:00Z",
  "message": "Workflow execution completed",
  "workflow_id": "abc123",
  "duration_ms": 1234
}
```

---

## Security

### Pod Security Context

The Helm chart runs innominatus as non-root user:

```yaml
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532

securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: false
  runAsNonRoot: true
  runAsUser: 65532
```

### Network Policies

Enable network policies to restrict traffic:

```yaml
networkPolicy:
  enabled: true
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
      - namespaceSelector:
          matchLabels:
            name: ingress-nginx
      ports:
      - protocol: TCP
        port: 8081
  egress:
    - to:
      - podSelector:
          matchLabels:
            app.kubernetes.io/name: postgresql
      ports:
      - protocol: TCP
        port: 5432
```

### Secrets Management

**Database credentials:**
```bash
# Create from literal
kubectl create secret generic innominatus-db \
  -n innominatus-system \
  --from-literal=password=secretPassword

# Create from file
kubectl create secret generic innominatus-db \
  -n innominatus-system \
  --from-file=password=./db-password.txt
```

**OIDC client secret:**
```bash
kubectl create secret generic innominatus-oidc \
  -n innominatus-system \
  --from-literal=client-secret=your-oidc-secret
```

### RBAC Best Practices

1. **Principle of Least Privilege**: Only grant demo-time the ClusterRole if needed
2. **Namespace Isolation**: Run innominatus in dedicated namespace
3. **Service Account**: Use dedicated ServiceAccount, not default
4. **Audit Logging**: Enable Kubernetes audit logs for innominatus actions

---

## Troubleshooting

### Pod Not Starting

**Check pod status:**
```bash
kubectl get pods -n innominatus-system
kubectl describe pod -n innominatus-system innominatus-xxxxx
```

**Common causes:**
- Image pull errors: Check imagePullSecrets
- Resource constraints: Increase memory/CPU limits
- PVC not bound: Check storage class and PV availability

### Database Connection Errors

**Symptoms:**
```
Error: failed to connect to database: connection refused
```

**Solutions:**
1. Check database service is running:
   ```bash
   kubectl get svc -n innominatus-system innominatus-postgresql
   ```

2. Verify database credentials:
   ```bash
   kubectl get secret -n innominatus-system innominatus-postgresql -o yaml
   ```

3. Test connection from pod:
   ```bash
   kubectl exec -it -n innominatus-system innominatus-xxxxx -- sh
   # Inside pod:
   wget -qO- http://innominatus-postgresql:5432
   ```

### demo-time RBAC Errors

**Symptoms:**
```
Error: failed to create namespace: forbidden
Error: serviceaccounts is forbidden
```

**Solutions:**
1. Verify ClusterRole exists:
   ```bash
   kubectl get clusterrole innominatus
   ```

2. Check ClusterRoleBinding:
   ```bash
   kubectl describe clusterrolebinding innominatus
   ```

3. Ensure rbac.create=true in values:
   ```yaml
   rbac:
     create: true
     clusterRole: true
   ```

### Ingress Not Working

**Check ingress:**
```bash
kubectl get ingress -n innominatus-system innominatus -o yaml
```

**Verify:**
- Ingress controller is running
- Ingress class matches controller
- DNS points to ingress IP/hostname
- TLS certificate is valid (if using TLS)

### High Memory Usage

**Check resource usage:**
```bash
kubectl top pod -n innominatus-system
```

**Increase limits:**
```yaml
resources:
  limits:
    memory: 2Gi
  requests:
    memory: 1Gi
```

---

## Upgrade Guide

### Helm Upgrade Process

**1. Review changes:**
```bash
helm diff upgrade innominatus ./charts/innominatus \
  -n innominatus-system \
  -f production-values.yaml
```

**2. Backup database (if using bundled PostgreSQL):**
```bash
kubectl exec -n innominatus-system innominatus-postgresql-0 -- \
  pg_dump -U innominatus idp_orchestrator > backup-$(date +%Y%m%d).sql
```

**3. Perform upgrade:**
```bash
helm upgrade innominatus ./charts/innominatus \
  -n innominatus-system \
  -f production-values.yaml \
  --wait \
  --timeout 10m
```

**4. Verify deployment:**
```bash
# Check pod status
kubectl get pods -n innominatus-system

# Check logs
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus -f

# Test health endpoint
kubectl exec -it -n innominatus-system innominatus-xxxxx -- wget -qO- http://localhost:8081/health
```

### Rollback

If upgrade fails, rollback:

```bash
# List revisions
helm history innominatus -n innominatus-system

# Rollback to previous version
helm rollback innominatus -n innominatus-system

# Rollback to specific revision
helm rollback innominatus 2 -n innominatus-system
```

### Database Migrations

Database migrations run automatically on startup. The server will:
- Create new tables if needed
- Add new columns to existing tables
- NOT drop or modify existing data

**Monitor migration logs:**
```bash
kubectl logs -n innominatus-system -l app.kubernetes.io/name=innominatus \
  | grep -i "migration\|schema"
```

---

## Best Practices

### Production Checklist

✅ Use external database (managed PostgreSQL)
✅ Enable OIDC/SSO authentication
✅ Configure TLS/HTTPS via ingress
✅ Set appropriate resource limits
✅ Enable autoscaling
✅ Configure pod disruption budget
✅ Enable Prometheus monitoring
✅ Setup backup for PostgreSQL
✅ Use separate namespaces per environment
✅ Apply network policies
✅ Regular security updates

### High Availability

For HA deployments:

```yaml
replicaCount: 3

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10

podDisruptionBudget:
  enabled: true
  minAvailable: 2

# Use external HA database
postgresql:
  enabled: false

externalDatabase:
  host: postgres-ha.example.com  # HA database endpoint
```

### Resource Sizing

| Environment | Replicas | CPU Request | Memory Request | CPU Limit | Memory Limit |
|-------------|----------|-------------|----------------|-----------|--------------|
| **Development** | 1 | 100m | 128Mi | 500m | 512Mi |
| **Staging** | 2 | 250m | 256Mi | 1000m | 1Gi |
| **Production** | 3 | 500m | 512Mi | 2000m | 2Gi |

---

## Next Steps

1. ✅ Deploy innominatus to your cluster
2. 📊 Setup monitoring and alerting
3. 🔒 Configure OIDC authentication
4. 📝 Create golden path workflows
5. 👥 Onboard development teams
6. 🚀 Deploy your first application via Score

---

## Support & Resources

- 📖 **Documentation**: https://github.com/innominatus/innominatus/tree/main/docs
- 🐛 **Issues**: https://github.com/innominatus/innominatus/issues
- 💬 **Community**: [Slack/Discord]
- 📧 **Contact**: support@innominatus.dev

---

*Last updated: 2025-10-06*
