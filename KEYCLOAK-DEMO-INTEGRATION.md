# Keycloak Integration - Demo Environment

## Overview

Keycloak has been successfully integrated into the demo environment with local users and ArgoCD OIDC authentication. The integration follows the existing demo infrastructure patterns.

## ✅ Build Status

All components built successfully:
- ✅ **Server binary** (`innominatus`)
- ✅ **CLI binary** (`innominatus-ctl`)
- ✅ **Web UI** (Next.js build)
- ✅ **All Go code** compiles without errors

## Integration Summary

### Components Added/Modified

1. **`internal/demo/components.go`**
   - Added Keycloak component (Bitnami Helm chart v25.2.0)
   - Configured with PostgreSQL, ingress, and keycloakconfigcli
   - Credentials: admin/adminpassword, demo-user/password123, test-user/test123

2. **`internal/demo/health.go`**
   - Added `checkKeycloak()` health check method
   - Parses `/health` endpoint JSON response
   - Integrated into health monitoring system

3. **`internal/demo/installer.go`**
   - **`ApplyKeycloakConfig()`** - Deploys realm config, users, ArgoCD OIDC client, and patches ArgoCD CM
   - **`RestartArgoCDServer()`** - Restarts ArgoCD to apply OIDC configuration

4. **`internal/cli/commands.go`**
   - **`DemoTimeCommand`** - Added Keycloak configuration and ArgoCD restart steps
   - **`DemoNukeCommand`** - Added keycloak namespace deletion and OIDC cleanup

5. **`internal/demo/cheatsheet.go`**
   - Updated Quick Start Guide with Keycloak section
   - Added OIDC login instructions for ArgoCD

## Usage

### Deploy Demo Environment

```bash
./innominatus-ctl demo-time
```

This will deploy:
1. NGINX Ingress Controller
2. Gitea (Git repository)
3. ArgoCD (GitOps) - **with OIDC enabled**
4. Vault & Vault Secrets Operator
5. Prometheus & Grafana
6. Minio (S3 storage)
7. **Keycloak (Identity Provider)** ← NEW!
8. Backstage (Developer Portal)
9. Kubernetes Dashboard
10. Demo Application

### Access Services

**Keycloak Admin Console:**
```
URL:      http://keycloak.localtest.me
Username: admin
Password: adminpassword
Realm:    demo-realm
```

**Demo Users:**
- `demo-user` / `password123`
- `test-user` / `test123`

**ArgoCD with OIDC:**
```
URL: http://argocd.localtest.me

Login Options:
1. Admin: admin / admin123
2. OIDC: Click "LOG IN VIA KEYCLOAK"
   - Use: demo-user / password123
```

### Check Demo Status

```bash
./innominatus-ctl demo-status
```

### Remove Demo Environment

```bash
./innominatus-ctl demo-nuke
```

This will:
- Uninstall all Helm releases (including Keycloak)
- Remove ArgoCD OIDC configuration
- Delete all namespaces (including keycloak)
- Clean database tables

## Configuration Details

### Keycloak Realm: demo-realm

**Realm Settings:**
- Display Name: "Demo Realm"
- Login Theme: keycloak
- SSL Required: external
- Registration: disabled
- Login with Email: enabled
- Brute Force Protection: enabled

**Users:**
| Username | Password | Role | Email |
|----------|----------|------|-------|
| demo-user | password123 | user | demo-user@example.com |
| test-user | test123 | user | test-user@example.com |

**Roles:**
- `user` - Standard user role
- `admin` - Administrator role

### ArgoCD OIDC Client

**Client Configuration:**
- Client ID: `argocd`
- Client Type: Confidential (with secret)
- Client Secret: `argocd-client-secret-change-me`
- Protocol: openid-connect
- Redirect URIs:
  - `http://argocd.localtest.me/auth/callback`
  - `https://argocd.localtest.me/auth/callback`

**Scopes:**
- `openid` - OpenID Connect authentication
- `profile` - User profile information
- `email` - Email address
- `groups` - Group membership (for RBAC)

**Protocol Mappers:**
- `groups` - Group membership mapper
- `email` - Email mapper
- `family_name` - Last name mapper
- `given_name` - First name mapper
- `username` - Username mapper (preferred_username claim)

### ArgoCD OIDC Configuration

**Issuer:**
```
http://keycloak.localtest.me/realms/demo-realm
```

**ConfigMap Patch (`argocd-cm`):**
```yaml
data:
  url: http://argocd.localtest.me
  oidc.config: |
    name: Keycloak
    issuer: http://keycloak.localtest.me/realms/demo-realm
    clientID: argocd
    clientSecret: $argocd-oidc-secret:clientSecret
    requestedScopes:
      - openid
      - profile
      - email
      - groups
    requestedIDTokenClaims:
      groups:
        essential: true
```

**Secret (`argocd-oidc-secret`):**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: argocd-oidc-secret
  namespace: argocd
type: Opaque
stringData:
  clientSecret: argocd-client-secret-change-me
```

## Architecture

### Installation Flow

```
demo-time
  ├── Install Helm components (including Keycloak)
  ├── Install Kubernetes Dashboard
  ├── Install Demo App
  ├── Wait for services healthy
  ├── Configure Keycloak realm + ArgoCD OIDC ← NEW
  │   ├── Deploy keycloak-realm-config ConfigMap
  │   ├── Deploy argocd-oidc-secret Secret
  │   └── Patch argocd-cm ConfigMap
  ├── Restart ArgoCD server ← NEW
  ├── Seed Git repository
  ├── Install Grafana dashboards
  └── Display credentials and status
```

### Cleanup Flow

```
demo-nuke
  ├── Uninstall Helm releases (including Keycloak)
  ├── Remove ArgoCD OIDC configuration ← NEW
  │   ├── Remove oidc.config from argocd-cm
  │   └── Delete argocd-oidc-secret
  ├── Delete namespaces (including keycloak) ← NEW
  └── Clean database tables
```

## Health Monitoring

Keycloak health check:
- Endpoint: `http://keycloak.localtest.me/health`
- Expected response: `{"status": "UP"}`
- Integrated into `demo-status` output

## Security Considerations

⚠️ **This is a DEMO configuration for local development only!**

For production use:
1. **Change all default passwords**
2. **Use strong client secrets** (not `argocd-client-secret-change-me`)
3. **Enable TLS/HTTPS** for all services
4. **Use external PostgreSQL** database
5. **Configure proper RBAC** policies in ArgoCD
6. **Enable Keycloak audit logging**
7. **Set up backup/restore** procedures
8. **Use proper secrets management** (Sealed Secrets, External Secrets, Vault)

## Troubleshooting

### Keycloak Not Starting

```bash
# Check Keycloak pods
kubectl get pods -n keycloak

# Check Keycloak logs
kubectl logs -n keycloak -l app.kubernetes.io/name=keycloak --tail=100

# Check PostgreSQL logs
kubectl logs -n keycloak -l app.kubernetes.io/name=postgresql --tail=100
```

### OIDC Login Not Working

```bash
# Verify ArgoCD ConfigMap
kubectl get configmap argocd-cm -n argocd -o yaml | grep -A 10 oidc.config

# Check ArgoCD server logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server --tail=100

# Verify secret exists
kubectl get secret argocd-oidc-secret -n argocd
```

### Realm Not Created

```bash
# Check keycloak-config-cli logs
kubectl logs -n keycloak -l app.kubernetes.io/name=keycloak-config-cli --tail=100

# Verify ConfigMap exists
kubectl get configmap keycloak-realm-config -n keycloak

# Force realm import by restarting Keycloak
kubectl delete pod -n keycloak -l app.kubernetes.io/name=keycloak
```

### Manual OIDC Configuration Removal

```bash
# Remove OIDC config from ArgoCD
kubectl patch configmap argocd-cm -n argocd --type json \
  -p '[{"op": "remove", "path": "/data/oidc.config"}]'

# Remove OIDC secret
kubectl delete secret argocd-oidc-secret -n argocd

# Restart ArgoCD server
kubectl rollout restart deployment argocd-server -n argocd
```

## Testing OIDC Login

1. Open http://argocd.localtest.me
2. Look for **"LOG IN VIA KEYCLOAK"** button
3. Click the button
4. You'll be redirected to: `http://keycloak.localtest.me/realms/demo-realm/protocol/openid-connect/auth?...`
5. Login with: `demo-user` / `password123`
6. You'll be redirected back to ArgoCD dashboard

## Future Enhancements

Possible improvements (not implemented yet):
- Group-based RBAC in ArgoCD
- Additional users/roles in Keycloak
- Custom Keycloak themes
- Email verification flow
- Password reset flow
- MFA/2FA support
- User self-service registration

## References

- [Keycloak Documentation](https://www.keycloak.org/documentation)
- [ArgoCD OIDC Configuration](https://argo-cd.readthedocs.io/en/stable/operator-manual/user-management/)
- [Bitnami Keycloak Helm Chart](https://github.com/bitnami/charts/tree/main/bitnami/keycloak)
- [keycloak-config-cli](https://github.com/adorsys/keycloak-config-cli)

---

**Status:** ✅ Ready for use
**Last Updated:** 2025-10-03
**Version:** v1.0.0
