# Troubleshooting Guide

Common issues and solutions for innominatus platform orchestration.

## Server Won't Start

### Database Connection Failed

**Symptom:**
```
FATAL: database connection failed: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solutions:**

1. **Check PostgreSQL is running:**
   ```bash
   # macOS (Homebrew)
   brew services list | grep postgresql
   brew services start postgresql@15

   # Docker
   docker ps | grep postgres
   docker-compose up -d postgres

   # Linux (systemd)
   sudo systemctl status postgresql
   sudo systemctl start postgresql
   ```

2. **Verify connection details:**
   ```bash
   # Test connection
   psql -h localhost -U postgres -d innominatus -c "SELECT 1"

   # Check environment variables
   env | grep DB_
   ```

3. **Create database if missing:**
   ```bash
   psql -h localhost -U postgres -c "CREATE DATABASE innominatus"
   ```

4. **Check credentials:**
   ```bash
   # Default: postgres/postgres
   export DB_PASSWORD=postgres
   ./innominatus
   ```

### Port Already in Use

**Symptom:**
```
FATAL: failed to start server: listen tcp :8081: bind: address already in use
```

**Solutions:**

1. **Find process using port:**
   ```bash
   # macOS/Linux
   lsof -i :8081

   # Kill process
   kill -9 <PID>
   ```

2. **Use different port:**
   ```bash
   export SERVER_PORT=8082
   ./innominatus
   ```

### OIDC Configuration Error

**Symptom:**
```
FATAL: OIDC issuer validation failed: failed to fetch discovery document
```

**Solutions:**

1. **Verify OIDC issuer URL:**
   ```bash
   curl https://keycloak.example.com/realms/production/.well-known/openid-configuration
   ```

2. **Check network connectivity:**
   ```bash
   ping keycloak.example.com
   curl -v https://keycloak.example.com
   ```

3. **Disable OIDC for testing:**
   ```bash
   export OIDC_ENABLED=false
   export AUTH_TYPE=file
   ./innominatus
   ```

4. **Validate client credentials:**
   ```bash
   # Check client exists in Keycloak
   # Admin Console → Clients → innominatus

   export OIDC_CLIENT_ID=innominatus
   export OIDC_CLIENT_SECRET=your-secret
   ```

## Workflow Execution Failures

### Terraform Step Failed

**Symptom:**
```
ERROR: terraform apply failed: exit status 1
```

**Solutions:**

1. **Check Terraform version:**
   ```bash
   terraform version
   # Requires: 1.0.0+
   ```

2. **Review step logs:**
   ```bash
   innominatus-ctl workflow logs <execution-id> --step terraform-apply
   ```

3. **Verify Terraform state:**
   ```bash
   # Check working directory
   ls -la /tmp/innominatus/workflows/<execution-id>/terraform/

   # View state
   cd /tmp/innominatus/workflows/<execution-id>/terraform/
   terraform show
   ```

4. **Common Terraform errors:**

   **Provider authentication:**
   ```bash
   # AWS
   export AWS_ACCESS_KEY_ID=...
   export AWS_SECRET_ACCESS_KEY=...

   # Azure
   az login

   # GCP
   export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
   ```

   **Resource already exists:**
   ```bash
   # Import existing resource
   terraform import aws_s3_bucket.example my-bucket
   ```

### Kubernetes Step Failed

**Symptom:**
```
ERROR: kubernetes apply failed: namespaces "my-namespace" already exists
```

**Solutions:**

1. **Check kubeconfig:**
   ```bash
   kubectl config current-context
   kubectl cluster-info
   ```

2. **Verify permissions:**
   ```bash
   kubectl auth can-i create namespaces
   kubectl auth can-i create deployments
   ```

3. **Check resource conflicts:**
   ```bash
   kubectl get all -n my-namespace
   kubectl delete namespace my-namespace  # If safe to delete
   ```

4. **Review manifest:**
   ```bash
   innominatus-ctl workflow logs <execution-id> --step kubectl-apply

   # Extract manifest
   grep -A 100 "Applying manifest" workflow.log
   ```

### Ansible Step Failed

**Symptom:**
```
ERROR: ansible-playbook failed: unreachable hosts
```

**Solutions:**

1. **Check Ansible version:**
   ```bash
   ansible --version
   # Requires: 2.10.0+
   ```

2. **Verify SSH connectivity:**
   ```bash
   ssh -i ~/.ssh/id_rsa user@target-host
   ```

3. **Review inventory:**
   ```bash
   innominatus-ctl workflow logs <execution-id> --step ansible-playbook

   # Check generated inventory
   cat /tmp/innominatus/workflows/<execution-id>/ansible/inventory
   ```

4. **Test playbook manually:**
   ```bash
   cd /tmp/innominatus/workflows/<execution-id>/ansible/
   ansible-playbook -i inventory playbook.yml -vvv
   ```

## Deployment Issues

### Deploy Command Not Found

**Symptom:**
```
innominatus-ctl: unknown command "deploy" for "innominatus-ctl"
```

**Solutions:**

1. **Check CLI version:**
   ```bash
   ./innominatus-ctl --version
   # Should be v0.2.0 or later
   ```

2. **Rebuild CLI if needed:**
   ```bash
   go build -o innominatus-ctl cmd/cli/main.go
   ```

3. **Use golden path as alternative:**
   ```bash
   ./innominatus-ctl run deploy-app score-spec.yaml
   ```

### Deployment Timeout

**Symptom:**
```
Error: deployment timed out after 5m0s
```

**Solutions:**

1. **Increase timeout:**
   ```bash
   ./innominatus-ctl deploy score-spec.yaml -w --timeout 10m
   ```

2. **Check workflow status:**
   ```bash
   ./innominatus-ctl list-workflows
   ./innominatus-ctl workflow detail <workflow-id>
   ```

3. **Check resource provisioning:**
   ```bash
   ./innominatus-ctl list-resources --type <resource-type>
   ```

### Resource Already Exists

**Symptom:**
```
ℹ️  Detected existing: db (postgres) - Skipping
```

**This is normal behavior** - Incremental deployments are idempotent:

**Examples:**
```bash
# Deploy v1 (database only)
./innominatus-ctl deploy app-v1.yaml -w
# ✅ Creates: db (postgres)

# Deploy v2 (database + S3)
./innominatus-ctl deploy app-v2.yaml -w
# ℹ️  Detected existing: db (postgres) - Skipping
# 🆕 Detected new: storage (s3) - Provisioning
```

**Force recreation (destructive):**
```bash
# Delete existing resource first
./innominatus-ctl delete-resource app-name resource-name

# Then redeploy
./innominatus-ctl deploy app-v2.yaml -w
```

### Invalid Score Specification

**Symptom:**
```
Error: invalid Score spec: missing required field 'apiVersion'
```

**Solutions:**

1. **Verify Score spec format:**
   ```yaml
   apiVersion: score.dev/v1b1  # Required
   metadata:
     name: my-app              # Required

   containers:
     main:
       image: nginx:latest     # Required

   resources:
     db:
       type: postgres
       properties:              # Use 'properties', not 'params'
         version: "15"
   ```

2. **Validate locally:**
   ```bash
   ./innominatus-ctl validate score-spec.yaml
   ```

3. **Common mistakes:**
   - Using `params:` instead of `properties:`
   - Missing `apiVersion: score.dev/v1b1`
   - Invalid resource type (unknown provider)
   - Missing required fields (name, image)

### Watch Mode Connection Lost

**Symptom:**
```
Warning: lost connection to server, deployment may still be in progress
```

**Solutions:**

1. **Check server is running:**
   ```bash
   curl http://localhost:8081/health
   ```

2. **Check deployment status:**
   ```bash
   ./innominatus-ctl status my-app
   ./innominatus-ctl list-workflows
   ```

3. **Reconnect to logs:**
   ```bash
   ./innominatus-ctl workflow logs <workflow-id> --follow
   ```

## Provider Resolution Errors

### Unknown Resource Type

**Symptom:**
```
ERROR: provider resolution failed: unknown resource type 'redis'
```

**Solutions:**

1. **List available providers:**
   ```bash
   innominatus-ctl list-providers
   ```

2. **Check provider capabilities:**
   ```bash
   innominatus-ctl provider detail database-team
   # Look for "Resource Types" section
   ```

3. **Add provider for resource type:**

   Create `providers/my-provider/provider.yaml`:
   ```yaml
   capabilities:
     resourceTypes:
       - redis
       - redis-cluster

   workflows:
     - name: provision-redis
       category: provisioner
   ```

4. **Reload providers:**
   ```bash
   curl -X POST http://localhost:8081/api/admin/providers/reload \
     -H "Authorization: Bearer <admin-token>"
   ```

### Capability Conflict

**Symptom:**
```
FATAL: capability conflict: resource type 'postgres' claimed by multiple providers:
  - database-team
  - backup-team
```

**Solutions:**

1. **Review provider manifests:**
   ```bash
   # Check which providers claim 'postgres'
   grep -r "postgres" providers/*/provider.yaml
   ```

2. **Remove duplicate capability:**

   Edit `providers/backup-team/provider.yaml`:
   ```yaml
   capabilities:
     resourceTypes:
       # Remove: - postgres
       - postgres-backup  # Use alias instead
   ```

3. **Use resource type aliases:**
   ```yaml
   # database-team: primary handler
   resourceTypes: [postgres, postgresql]

   # backup-team: different capability
   resourceTypes: [postgres-backup, db-backup]
   ```

4. **Restart server:**
   ```bash
   pkill innominatus
   ./innominatus
   ```

### No Provisioner Workflow

**Symptom:**
```
ERROR: provider 'storage-team' has no provisioner workflows for resource type 's3'
```

**Solutions:**

1. **Check provider workflows:**
   ```bash
   innominatus-ctl provider detail storage-team
   ```

2. **Add provisioner workflow:**

   Edit `providers/storage-team/provider.yaml`:
   ```yaml
   workflows:
     - name: provision-s3-bucket
       file: ./workflows/provision-s3.yaml
       category: provisioner  # MUST be 'provisioner', not 'goldenpath'
   ```

3. **Verify workflow file exists:**
   ```bash
   ls -la providers/storage-team/workflows/provision-s3.yaml
   ```

## Authentication Issues

### OIDC Token Expired

**Symptom:**
```
ERROR: authentication failed: token is expired
```

**Solutions:**

1. **Re-authenticate:**
   ```bash
   # Web UI: Log out and log in again
   # CLI will auto-redirect to OIDC login
   ```

2. **Increase session duration:**

   In Keycloak:
   ```
   Realm → Tokens → Access Token Lifespan: 1 hour
   Realm → Tokens → SSO Session Idle: 8 hours
   ```

3. **Use API key instead:**
   ```bash
   # Web UI → Profile → Generate API Key
   export INNOMINATUS_API_KEY=<key>
   innominatus-ctl list
   ```

### API Key Invalid

**Symptom:**
```
ERROR: authentication failed: invalid API key
```

**Solutions:**

1. **Regenerate API key:**
   ```bash
   # Web UI → Profile → Revoke Old Key → Generate New Key
   ```

2. **Check key format:**
   ```bash
   # Should be: inn_<base64_string>
   echo $INNOMINATUS_API_KEY
   ```

3. **Verify key in database:**
   ```bash
   psql -h localhost -U postgres -d innominatus \
     -c "SELECT id, user_id, key_hash, created_at FROM api_keys WHERE revoked_at IS NULL"
   ```

### Session Expired

**Symptom:**
```
ERROR: session expired, please log in again
```

**Solutions:**

1. **Clear browser cookies:**
   ```
   Browser → DevTools → Application → Cookies → Clear
   ```

2. **Increase session timeout:**
   ```bash
   export SESSION_TIMEOUT=86400  # 24 hours
   ./innominatus
   ```

3. **Check SESSION_SECRET:**
   ```bash
   # Must be consistent across restarts
   export SESSION_SECRET=changeme-to-random-string
   ```

## Database Migration Problems

### Migration Failed

**Symptom:**
```
FATAL: migration 010_add_workflow_execution_id failed: column already exists
```

**Solutions:**

1. **Check migration status:**
   ```bash
   psql -h localhost -U postgres -d innominatus \
     -c "SELECT * FROM schema_migrations ORDER BY version"
   ```

2. **Rollback to specific version:**
   ```bash
   # Edit internal/database/database.go to skip failing migration
   # Or manually apply SQL
   ```

3. **Reset database (DEVELOPMENT ONLY):**
   ```bash
   psql -h localhost -U postgres -c "DROP DATABASE innominatus"
   psql -h localhost -U postgres -c "CREATE DATABASE innominatus"
   ./innominatus  # Migrations run automatically
   ```

### Migration Out of Order

**Symptom:**
```
FATAL: migration ordering error: 012 cannot run before 011
```

**Solutions:**

1. **Check migration files:**
   ```bash
   ls -la internal/database/migrations/
   # Should be: 001_, 002_, 003_, etc.
   ```

2. **Rename migrations sequentially:**
   ```bash
   mv 012_new.sql 011_new.sql
   mv 011_old.sql 012_old.sql
   ```

## Demo Environment Issues

### Demo Install Failed

**Symptom:**
```
ERROR: demo-time failed: Kubernetes cluster not found
```

**Solutions:**

1. **Check Docker Desktop:**
   ```bash
   docker version
   docker ps
   ```

2. **Enable Kubernetes:**
   ```
   Docker Desktop → Settings → Kubernetes → Enable Kubernetes → Apply
   ```

3. **Verify kubectl:**
   ```bash
   kubectl cluster-info
   kubectl get nodes
   ```

4. **Check demo status:**
   ```bash
   innominatus-ctl demo-status
   ```

### Demo Service Unreachable

**Symptom:**
```
ERROR: cannot connect to http://gitea.localtest.me
```

**Solutions:**

1. **Check service pods:**
   ```bash
   kubectl get pods -n gitea
   kubectl get pods -n argocd
   ```

2. **Check ingress:**
   ```bash
   kubectl get ingress -A
   ```

3. **Verify DNS resolution:**
   ```bash
   ping gitea.localtest.me
   # Should resolve to 127.0.0.1
   ```

4. **Port forward manually:**
   ```bash
   kubectl port-forward -n gitea svc/gitea-http 3000:3000
   # Access: http://localhost:3000
   ```

## Performance Debugging

### Slow Workflow Execution

**Symptom:**
```
Workflow taking > 10 minutes (expected: 2-3 minutes)
```

**Solutions:**

1. **Check step timings:**
   ```bash
   innominatus-ctl workflow detail <execution-id>
   # Look for long-running steps
   ```

2. **Review resource limits:**
   ```bash
   # Kubernetes deployments
   kubectl top pods -n innominatus-system

   # Server resources
   ps aux | grep innominatus
   ```

3. **Check database performance:**
   ```bash
   psql -h localhost -U postgres -d innominatus \
     -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10"
   ```

4. **Enable debug logging:**
   ```bash
   export LOG_LEVEL=debug
   ./innominatus
   ```

### High Memory Usage

**Symptom:**
```
Server consuming > 2GB RAM (expected: 200-500MB)
```

**Solutions:**

1. **Check goroutine leaks:**
   ```bash
   # Enable pprof
   curl http://localhost:8081/debug/pprof/goroutine?debug=1
   ```

2. **Review database connection pool:**
   ```bash
   psql -h localhost -U postgres -d innominatus \
     -c "SELECT count(*) FROM pg_stat_activity WHERE datname='innominatus'"
   ```

3. **Reduce concurrent workflows:**
   ```bash
   export WORKFLOW_MAX_CONCURRENT=5
   ./innominatus
   ```

## Common Error Messages

### "workflow_execution_id cannot be null"

**Cause:** Resource state updated without setting workflow_execution_id

**Solution:**
```go
// Always set workflow_execution_id when changing state to 'provisioning'
resource.WorkflowExecutionID = &executionID
resource.State = "provisioning"
db.Save(&resource)
```

### "graph edge creation failed: source node not found"

**Cause:** Trying to create edge before nodes exist

**Solution:**
```go
// Create nodes first
graph.CreateNode(spec)
graph.CreateNode(resource)

// Then create edge
graph.CreateEdge(spec, resource, "contains")
```

### "provider not found for resource type"

**Cause:** No provider declares capability for requested resource type

**Solution:**
1. Add provider with capability
2. Or use different resource type that exists
3. Check for typos in resource type name

### "OIDC discovery failed"

**Cause:** Cannot reach OIDC issuer URL

**Solution:**
1. Verify URL format: `https://keycloak.example.com/realms/production`
2. Check network connectivity
3. Verify SSL certificates (use `curl -v`)

---

## Getting Help

**Logs:**
```bash
# Server logs
./innominatus 2>&1 | tee server.log

# Workflow logs
innominatus-ctl workflow logs <execution-id>

# System logs (Kubernetes)
kubectl logs -n innominatus-system deployment/innominatus
```

**Debug Mode:**
```bash
export LOG_LEVEL=debug
export ENABLE_PROFILING=true
./innominatus
```

**Health Checks:**
```bash
curl http://localhost:8081/health | jq
curl http://localhost:8081/ready | jq
```

**Report Issues:**
- GitHub: https://github.com/philipsahli/innominatus/issues
- Include: logs, error messages, environment details

---

**Related Docs:**
- [CLAUDE.md](../CLAUDE.md) - Development guide
- [ARCHITECTURE.md](./ARCHITECTURE.md) - System architecture
- [QUICKREF.md](../QUICKREF.md) - Command reference
