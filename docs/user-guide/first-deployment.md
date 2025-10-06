# Your First Deployment

Deploy your first application to the innominatus platform in 5 minutes.

---

## Prerequisites

✅ Access to innominatus platform (see [Getting Started](getting-started.md))
✅ `innominatus-ctl` CLI installed
✅ API key configured

---

## Step 1: Create Score Specification

Create `my-app.yaml`:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app
  version: "1.0.0"

containers:
  web:
    image: nginx:latest
    ports:
      - port: 80
        protocol: TCP

resources:
  route:
    type: route
    params:
      host: my-app.yourcompany.com
      port: 80
```

---

## Step 2: Deploy

```bash
innominatus-ctl deploy my-app.yaml
```

**What happens:**
1. innominatus validates your Score spec
2. Creates Kubernetes namespace
3. Provisions route/ingress
4. Deploys your container
5. Runs health checks
6. Returns deployment URL

---

## Step 3: Verify Deployment

```bash
# Check status
innominatus-ctl status my-app

# View workflow history
innominatus-ctl workflows my-app

# Check logs
innominatus-ctl logs my-app
```

---

## Step 4: Access Your App

Open the URL provided in deployment output:
```
🔗 https://my-app.yourcompany.com
```

---

## What's Next?

### Add a Database

```yaml
resources:
  db:
    type: postgres
    params:
      version: "15"
      storage: "10Gi"
```

### Add Environment Variables

```yaml
containers:
  web:
    variables:
      DATABASE_URL: "${resources.db.connection_string}"
      APP_ENV: "production"
```

### Use Golden Paths

```bash
# List available workflows
innominatus-ctl list-goldenpaths

# Deploy with specific golden path
innominatus-ctl run deploy-app my-app.yaml
```

---

## Troubleshooting

**Deployment fails?** See [Troubleshooting Guide](troubleshooting.md)

**Questions?** Contact your Platform Team

---

**Continue:** [CLI Reference](cli-reference.md) | [Troubleshooting](troubleshooting.md)
