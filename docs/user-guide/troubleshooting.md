# Troubleshooting

Common issues and solutions for innominatus users.

---

## Authentication Issues

### "Invalid API key"

**Cause:** API key expired or incorrect

**Solution:**
1. Log into Web UI
2. Navigate to Profile → API Keys
3. Generate new API key
4. Update your configuration:

```bash
export INNOMINATUS_API_KEY="new-key-here"
# or update ~/.innominatus/config.yaml
```

### "Unauthorized"

**Cause:** Missing API key or incorrect platform URL

**Solution:**
```bash
# Check configuration
innominatus-ctl --url https://innominatus.yourcompany.com \
  --api-key your-key \
  list
```

---

## Deployment Issues

### "Deployment failed"

**Check workflow status:**
```bash
innominatus-ctl workflows my-app
```

**Common causes:**
- Invalid Score specification → Run `innominatus-ctl validate`
- Resource quota exceeded → Contact Platform Team
- Image pull failure → Verify container image exists

### "Namespace already exists"

**Cause:** Previous deployment wasn't cleaned up

**Solution:**
```bash
# Delete old deployment
innominatus-ctl delete my-app

# Redeploy
innominatus-ctl deploy my-app.yaml
```

---

## Resource Issues

### "Database provisioning failed"

**Cause:** Database resource type not supported or quota exceeded

**Solution:**
1. Check supported resource types with Platform Team
2. Verify resource parameters:

```yaml
resources:
  db:
    type: postgres  # Check if this type is configured
    params:
      version: "15"
      storage: "10Gi"  # Check if this quota is available
```

### "Route already in use"

**Cause:** Hostname conflict with another application

**Solution:**
Use a unique hostname:

```yaml
resources:
  route:
    params:
      host: my-app-unique.yourcompany.com  # Make it unique
```

---

## CLI Issues

### "Command not found: innominatus-ctl"

**Cause:** CLI not in PATH

**Solution:**
```bash
# Add to PATH
export PATH=$PATH:/path/to/innominatus-ctl

# Or move to system path
sudo mv innominatus-ctl /usr/local/bin/
```

### "Connection refused"

**Cause:** Platform URL incorrect or platform is down

**Solution:**
1. Verify platform URL:
```bash
curl https://innominatus.yourcompany.com/health
```

2. Contact Platform Team if platform is unreachable

---

## Score Specification Issues

### "Invalid Score specification"

**Validate your spec:**
```bash
innominatus-ctl validate my-app.yaml
```

**Common mistakes:**
- Missing `apiVersion`
- Invalid resource type
- Incorrect YAML syntax

**Example fix:**
```yaml
apiVersion: score.dev/v1b1  # Required
metadata:
  name: my-app  # Required
containers:
  web:  # At least one container required
    image: nginx:latest
```

---

## Workflow Issues

### "Workflow stuck in 'running' state"

**Check workflow details:**
```bash
innominatus-ctl workflows my-app --output json | jq
```

**Solution:**
- Wait for timeout (usually 15-30 minutes)
- Contact Platform Team if stuck longer

### "Step failed: terraform apply"

**Cause:** Infrastructure provisioning error

**Solution:**
- Check workflow logs for specific error
- Contact Platform Team (they have access to Terraform state)

---

## Performance Issues

### "Deployment is slow"

**Expected times:**
- Simple deployment: 1-3 minutes
- With database: 3-5 minutes
- Complex workflow: 5-10 minutes

**If slower:**
- Check platform status with Platform Team
- Verify network connectivity

---

## Getting Help

### 1. Check Documentation

- [Getting Started](getting-started.md)
- [CLI Reference](cli-reference.md)
- Platform documentation portal (ask Platform Team)

### 2. Self-Diagnosis

```bash
# Check platform health
curl https://innominatus.yourcompany.com/health

# Validate your Score spec
innominatus-ctl validate my-app.yaml

# Check workflow details
innominatus-ctl workflows my-app --output json
```

### 3. Contact Platform Team

**What to include:**
- Application name
- Error message (full output)
- Score specification (sanitized)
- Workflow ID

**Example:**
```
Hi Platform Team,

Application: my-app
Error: "Database provisioning failed"
Workflow ID: wf-abc123
Score spec: [attached]

Please help!
```

---

## FAQ

**Q: Can I deploy to production?**
A: Ask your Platform Team about environment policies and approval workflows.

**Q: How do I update my application?**
A: Modify your Score spec and run `deploy` again. innominatus handles updates automatically.

**Q: Can I rollback a deployment?**
A: Ask your Platform Team about rollback procedures.

**Q: How do I see application logs?**
A: Use `innominatus-ctl logs my-app` or check your platform's logging dashboard.

---

**Still stuck?** Contact your Platform Team - they're here to help! 🚀
