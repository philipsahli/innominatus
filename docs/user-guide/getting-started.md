# Getting Started

**Your Platform Team has already set up innominatus for you.** This guide will help you start deploying applications.

---

## What You Have

✅ **innominatus Platform**: Running and managed by your Platform Team
✅ **Platform URL**: Ask your Platform Team (usually `https://innominatus.yourcompany.com`)
✅ **Access Control**: OIDC/SSO or API key authentication

---

## Step 1: Get Access

### Option A: SSO Login (Web UI)

1. Open the innominatus Web UI (get URL from your Platform Team)
2. Click **"Login with SSO"** or **"Login with Keycloak"**
3. Use your company credentials
4. You're in! 🎉

### Option B: API Key (CLI)

1. Log into the Web UI (see Option A)
2. Navigate to **Profile → API Keys**
3. Click **"Generate New Key"**
4. Give it a name (e.g., "my-laptop-cli")
5. **Copy the key** (you won't see it again!)
6. Save it securely

```bash
# Set your API key
export INNOMINATUS_API_KEY="your-api-key-here"

# Or save to config file
echo "api_key: your-api-key-here" > ~/.innominatus/config.yaml
```

---

## Step 2: Install the CLI

### Download from Platform Portal

Your Platform Team provides a download link for the `innominatus-ctl` CLI:

```bash
# Example (ask your Platform Team for actual URL)
curl -L https://platform.company.com/downloads/innominatus-ctl -o innominatus-ctl
chmod +x innominatus-ctl

# Move to PATH
sudo mv innominatus-ctl /usr/local/bin/
```

### Verify Installation

```bash
innominatus-ctl --version
# Output: innominatus-ctl v1.0.0
```

---

## Step 3: Configure the CLI

Tell the CLI where your innominatus platform is:

```bash
# Set platform URL
export INNOMINATUS_URL="https://innominatus.yourcompany.com"

# Or create config file
mkdir -p ~/.innominatus
cat <<EOF > ~/.innominatus/config.yaml
url: https://innominatus.yourcompany.com
api_key: your-api-key-here
EOF
```

---

## Step 4: Test Your Setup

```bash
# List deployed applications (should return empty list if you haven't deployed anything)
innominatus-ctl list

# Expected output:
# No applications deployed yet
```

If this works, you're ready to deploy! 🚀

---

## Step 5: Your First Deployment

Create a simple Score specification:

```bash
cat <<EOF > my-first-app.yaml
apiVersion: score.dev/v1b1
metadata:
  name: my-first-app

containers:
  web:
    image: nginx:latest
    ports:
      - port: 80

resources:
  route:
    type: route
    params:
      host: my-first-app.yourcompany.com
EOF
```

Deploy it:

```bash
innominatus-ctl deploy my-first-app.yaml
```

Output:
```
🚀 Starting deployment: my-first-app

[1/5] ✓ validate-spec - 0.5s
[2/5] ✓ provision-namespace - 1.2s
[3/5] ✓ deploy-application - 2.1s
[4/5] ✓ health-check - 3.0s
[5/5] ✓ register-app - 0.3s

✅ Deployed successfully in 7.1 seconds!
🔗 https://my-first-app.yourcompany.com
```

**🎉 SUCCESS!** Visit the URL and see your app running!

**What just happened:**
- Created Kubernetes namespace
- Deployed nginx container
- Configured ingress route
- Verified health checks
- All done automatically by innominatus!

---

## Common Commands

```bash
# Deploy application
innominatus-ctl deploy my-app.yaml

# Check status
innominatus-ctl status my-first-app

# View logs
innominatus-ctl logs my-first-app

# List all apps
innominatus-ctl list

# Delete app
innominatus-ctl delete my-first-app
```

---

## What's Next?

- **[First Deployment](first-deployment.md)** - Detailed walkthrough with troubleshooting
- **[CLI Reference](cli-reference.md)** - Complete command documentation
- **[Troubleshooting](troubleshooting.md)** - Fix common issues

---

## Getting Help

**Platform Team**: Your first point of contact for any issues
**Self-Service**: [Troubleshooting Guide](troubleshooting.md), CLI `--help`, Platform documentation portal

---

**Happy deploying!** 🚀
