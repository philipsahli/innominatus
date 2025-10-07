# User Guide

**Welcome, Developer!** 👋

Your Platform Team has set up innominatus for you. This guide will help you deploy your applications using Score specifications.

---

## What You Need to Know

innominatus is a **platform orchestration service** that your Platform Team manages. You don't need to install or configure it - it's already running and ready for you to use.

**What innominatus does for you:**
- Takes your Score specification (simple YAML file)
- Orchestrates complex multi-step deployments
- Provisions infrastructure (databases, storage, networks)
- Deploys your application to Kubernetes
- Sets up monitoring and logging

**What you need:**
- ✅ Access credentials from your Platform Team
- ✅ The **innominatus-ctl** CLI tool (download from your platform portal)
- ✅ A Score specification for your application

---

## Quick Start

### 1. Get Your Credentials

Contact your Platform Team to get:
- **Platform URL**: https://innominatus.yourcompany.com
- **API Key**: Generate via Web UI → Profile → API Keys

### 2. Install the CLI

```bash
# Download from your platform portal or use the binary
# Your Platform Team will provide the download link

# Verify installation
innominatus-ctl --version
```

### 3. Deploy Your First Application

See the [First Deployment Guide](first-deployment.md) for a complete walkthrough.

---

## Documentation

| Guide | Description |
|-------|-------------|
| **[Getting Started](getting-started.md)** | First steps with innominatus - connect, install CLI, deploy |
| **[First Deployment](first-deployment.md)** | Deploy your first app in 5 minutes |
| **[CLI Reference](cli-reference.md)** | Complete CLI command reference |
| **[Recipes](recipes/README.md)** | Real-world deployment examples (Node.js, Python, etc.) |
| **[Troubleshooting](troubleshooting.md)** | Common issues and solutions |

### Featured Recipes

**[Node.js API with PostgreSQL](recipes/nodejs-postgres.md)** - Deploy a production-ready REST API with database, Redis cache, and auto-scaling in 15 minutes.

---

## Common Tasks

### Deploy an Application

```bash
innominatus-ctl deploy my-app.yaml
```

### Check Application Status

```bash
innominatus-ctl status my-app
```

### List Your Applications

```bash
innominatus-ctl list
```

### Delete an Application

```bash
innominatus-ctl delete my-app
```

---

## Getting Help

**First, contact your Platform Team** - they manage innominatus and can help with:
- Access issues
- Deployment failures
- Resource provisioning

**Self-Service Resources:**
- [Troubleshooting Guide](troubleshooting.md)
- Platform documentation portal (ask your Platform Team for the link)
- CLI help: **innominatus-ctl --help**

---

## What's a Score Specification?

Score is a platform-agnostic workload specification format. You describe **what** you need (containers, databases, routes), not **how** to provision them.

**Example Score Spec:**
```yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-app

containers:
  web:
    image: nginx:latest

resources:
  db:
    type: postgres
  route:
    type: route
    params:
      host: my-app.company.com
```

Learn more at [score.dev](https://score.dev)

---

## Next Steps

1. **[Getting Started](getting-started.md)** - Set up your environment in 15 minutes
2. **[First Deployment](first-deployment.md)** - Deploy your first app in 5 minutes
3. **[Recipes](recipes/README.md)** - Real-world deployment examples
   - [Node.js + PostgreSQL](recipes/nodejs-postgres.md) - Production-ready API with database
4. **[CLI Reference](cli-reference.md)** - Learn all CLI commands
5. **[Troubleshooting](troubleshooting.md)** - Fix common issues

---

**Questions?** Ask your Platform Team! They're here to help. 🚀
