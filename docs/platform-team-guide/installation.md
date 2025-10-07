# Installation Guide

Deploy innominatus to your Kubernetes cluster.

---

## Prerequisites

- Kubernetes 1.24+ cluster
- PostgreSQL 13+ database
- kubectl configured with admin access
- Helm 3.x (optional)

---

## Option 1: Docker Image

### Pull Image

```bash
docker pull ghcr.io/philipsahli/innominatus:latest
```

### Run Locally

```bash
docker run -p 8081:8081 \
  -e DB_HOST=postgres.example.com \
  -e DB_USER=orchestrator \
  -e DB_NAME=idp_orchestrator \
  -e DB_PASSWORD=secure_password \
  ghcr.io/philipsahli/innominatus:latest
```

---

## Option 2: Kubernetes Deployment

### Create Namespace

```bash
kubectl create namespace platform
```

### Deploy PostgreSQL (if needed)

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install postgres bitnami/postgresql \
  --namespace platform \
  --set auth.database=idp_orchestrator \
  --set auth.username=orchestrator \
  --set auth.password=secure_password
```

### Create Secret

```bash
kubectl create secret generic innominatus-db \
  --namespace platform \
  --from-literal=username=orchestrator \
  --from-literal=password=secure_password
```

### Deploy innominatus

Create `deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: innominatus
  namespace: platform
spec:
  replicas: 3
  selector:
    matchLabels:
      app: innominatus
  template:
    metadata:
      labels:
        app: innominatus
    spec:
      containers:
      - name: server
        image: ghcr.io/philipsahli/innominatus:latest
        ports:
        - containerPort: 8081
        env:
        - name: DB_HOST
          value: "postgres-postgresql.platform.svc.cluster.local"
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: "idp_orchestrator"
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: innominatus-db
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: innominatus-db
              key: password
        - name: DB_SSLMODE
          value: "require"
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
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: innominatus
  namespace: platform
spec:
  selector:
    app: innominatus
  ports:
  - port: 8081
    targetPort: 8081
  type: ClusterIP
```

Apply:

```bash
kubectl apply -f deployment.yaml
```

### Verify Deployment

```bash
kubectl get pods -n platform
kubectl logs -n platform deployment/innominatus
```

---

## Option 3: Build from Source

See [Development Guide](../development/building.md) for building from source.

---

## Post-Installation

### 1. Verify Health

```bash
kubectl port-forward -n platform svc/innominatus 8081:8081

curl http://localhost:8081/health
# Expected: {"status": "ok"}
```

### 2. Access Web UI

```bash
kubectl port-forward -n platform svc/innominatus 8081:8081
```

Open: http://localhost:8081

### 3. Configure Ingress (Production)

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: innominatus
  namespace: platform
spec:
  rules:
  - host: innominatus.yourcompany.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: innominatus
            port:
              number: 8081
```

---

## Next Steps

- **[Configuration](configuration.md)** - Set up OIDC and RBAC
- **[Database](database.md)** - Production database configuration
- **[Operations](operations.md)** - Monitoring and scaling

---

**Installation complete!** 🎉
