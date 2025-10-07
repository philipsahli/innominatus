# Configuration Guide

Configure innominatus for production use with OIDC authentication, RBAC, and security.

---

## OIDC Authentication

innominatus supports enterprise SSO via OpenID Connect (OIDC).

### Quick Start

```bash
export OIDC_ENABLED=true
export OIDC_ISSUER=https://keycloak.company.com/realms/production
export OIDC_CLIENT_ID=innominatus
export OIDC_CLIENT_SECRET=your-client-secret
export OIDC_REDIRECT_URL=https://innominatus.company.com/auth/oidc/callback
```

### Kubernetes Configuration

```yaml
env:
- name: OIDC_ENABLED
  value: "true"
- name: OIDC_ISSUER
  value: "https://keycloak.company.com/realms/production"
- name: OIDC_CLIENT_ID
  value: "innominatus"
- name: OIDC_CLIENT_SECRET
  valueFrom:
    secretKeyRef:
      name: innominatus-oidc
      key: client-secret
- name: OIDC_REDIRECT_URL
  value: "https://innominatus.company.com/auth/oidc/callback"
```

### Detailed OIDC Setup

**See:** [OIDC Authentication Guide](../OIDC_AUTHENTICATION.md) for complete setup instructions including:
- Keycloak client configuration
- API key management for OIDC users
- Dual authentication (local + OIDC users)
- Database-backed API keys

---

## Role-Based Access Control (RBAC)

### Admin Configuration

Create `admin-config.yaml`:

```yaml
admin:
  defaultRuntime: "kubernetes"

rbac:
  roles:
    - name: "platform-admin"
      permissions:
        - "workflows:*"
        - "specs:*"
        - "admin:*"

    - name: "developer"
      permissions:
        - "workflows:read"
        - "specs:create"
        - "specs:read"

    - name: "viewer"
      permissions:
        - "workflows:read"
        - "specs:read"

resourceDefinitions:
  postgres: "managed-postgres-cluster"
  redis: "redis-cluster"
  volume: "persistent-volume-claim"
  route: "ingress-route"

policies:
  enforceBackups: true
  allowedEnvironments:
    - "development"
    - "staging"
    - "production"
```

Mount as ConfigMap:

```bash
kubectl create configmap innominatus-config \
  --namespace platform \
  --from-file=admin-config.yaml
```

---

## Secrets Management

### Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: innominatus-secrets
  namespace: platform
type: Opaque
stringData:
  db-password: "postgres_password_here"
  jwt-secret: "random_jwt_secret_here"
  oidc-client-secret: "oidc_client_secret_here"
```

### Vault Integration

```yaml
env:
- name: VAULT_ADDR
  value: "https://vault.company.com"
- name: VAULT_TOKEN
  valueFrom:
    secretKeyRef:
      name: vault-token
      key: token
```

---

## Environment Variables

### Required

```bash
DB_HOST=postgres.production.internal
DB_PORT=5432
DB_USER=orchestrator_service
DB_PASSWORD=secure_password
DB_NAME=idp_orchestrator
DB_SSLMODE=require
```

### Optional

```bash
# Server
PORT=8081

# OIDC
OIDC_ENABLED=true
OIDC_ISSUER=https://keycloak.company.com/realms/production
OIDC_CLIENT_ID=innominatus
OIDC_CLIENT_SECRET=client-secret
OIDC_REDIRECT_URL=https://innominatus.company.com/auth/oidc/callback

# Metrics
PUSHGATEWAY_URL=http://pushgateway.monitoring.svc:9091

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## Network Policies

Restrict network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: innominatus-network-policy
  namespace: platform
spec:
  podSelector:
    matchLabels:
      app: innominatus
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
    - namespaceSelector:
        matchLabels:
          name: platform
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
```

---

## Next Steps

- **[Database](database.md)** - Configure PostgreSQL for production
- **[Operations](operations.md)** - Monitor and scale your deployment
- **[Monitoring](monitoring.md)** - Set up Prometheus and Grafana

---

**See Also:**
- [OIDC Authentication Guide](../OIDC_AUTHENTICATION.md)
- [API Security Phase 1](../API_SECURITY_PHASE1.md)
