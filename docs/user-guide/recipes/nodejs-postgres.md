# Recipe: Node.js API with PostgreSQL

Deploy a production-ready Node.js REST API with PostgreSQL database.

## What You'll Build

A Todo List API with:
- ✅ Express.js REST API
- ✅ PostgreSQL database (managed by platform)
- ✅ Database migrations
- ✅ Health checks
- ✅ Logging
- ✅ Auto-scaling
- ✅ HTTPS ingress

**Time to deploy:** 15 minutes

## Prerequisites

- Node.js app with Dockerfile
- Basic understanding of environment variables
- Access to your platform (see [Getting Started](../getting-started.md))

---

## Architecture

```mermaid
graph LR
    A[Internet] -->|HTTPS| B[Ingress]
    B --> C[Node.js API]
    C --> D[PostgreSQL]
    C --> E[Redis]

    style C fill:#61DAFB
    style D fill:#336791
    style E fill:#DC382D
```

**Components:**
- **Node.js API** - Your Express.js application (3 replicas)
- **PostgreSQL** - Managed database (provisioned by platform)
- **Redis** - Session store (optional, provisioned by platform)
- **Ingress** - HTTPS endpoint with SSL certificate

---

## Score Specification

Create `score.yaml`:

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: todo-api
  annotations:
    team: backend-team
    owner: mike@company.com

# Define your application container
containers:
  api:
    image: registry.company.com/todo-api:latest

    # Environment variables for your app
    variables:
      NODE_ENV: production
      PORT: "3000"

      # Database connection (automatically injected)
      DATABASE_URL: "postgresql://${resources.database.username}:${resources.database.password}@${resources.database.host}:${resources.database.port}/${resources.database.name}"

      # Redis connection (automatically injected)
      REDIS_URL: "redis://${resources.cache.host}:${resources.cache.port}"

      # App-specific variables
      JWT_SECRET: "${secrets.jwt-secret}"
      API_KEY: "${secrets.api-key}"

    # Health check configuration
    livenessProbe:
      httpGet:
        path: /health
        port: 3000
      initialDelaySeconds: 30
      periodSeconds: 10

    readinessProbe:
      httpGet:
        path: /ready
        port: 3000
      initialDelaySeconds: 5
      periodSeconds: 5

# Define required infrastructure resources
resources:
  # PostgreSQL database
  database:
    type: postgres
    params:
      version: "15"
      size: small           # small, medium, large
      storage: 20Gi
      backup_enabled: true
      backup_retention: 7   # days

  # Redis cache (optional)
  cache:
    type: redis
    params:
      version: "7"
      memory: 1Gi

  # HTTPS ingress
  route:
    type: route
    params:
      host: todo-api.company.com
      port: 3000
      tls: true
      annotations:
        rate-limit: "100/minute"

# Deployment configuration
deployment:
  replicas: 3
  resources:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 1000m
      memory: 512Mi

  # Auto-scaling
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilization: 70

# Environment-specific settings
environment:
  type: kubernetes
  namespace: backend-team
```

---

## Deployment Guide

### Step 1: Prepare Your Application

Your Node.js app needs:

**1. Dockerfile**

Create `Dockerfile` in your repo:

```dockerfile
FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production

# Copy application code
COPY . .

# Create non-root user
RUN addgroup -g 1001 -S nodejs && \
    adduser -S nodejs -u 1001
USER nodejs

EXPOSE 3000

CMD ["node", "server.js"]
```

**2. Health Check Endpoints**

Add to your Express app:

```javascript
// server.js
const express = require('express');
const app = express();

// Health check endpoints
app.get('/health', (req, res) => {
  res.status(200).json({ status: 'healthy' });
});

app.get('/ready', async (req, res) => {
  // Check database connection
  try {
    await db.query('SELECT 1');
    res.status(200).json({ status: 'ready' });
  } catch (err) {
    res.status(503).json({ status: 'not ready', error: err.message });
  }
});

// Your API routes
app.get('/api/todos', getTodos);
app.post('/api/todos', createTodo);
// ...

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
```

**3. Database Connection**

Use environment variable from Score spec:

```javascript
// db.js
const { Pool } = require('pg');

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  ssl: process.env.NODE_ENV === 'production' ? { rejectUnauthorized: false } : false,
  max: 20,
  idleTimeoutMillis: 30000,
  connectionTimeoutMillis: 2000,
});

module.exports = pool;
```

### Step 2: Build and Push Docker Image

```bash
# Build image
docker build -t registry.company.com/todo-api:latest .

# Push to your company's registry
docker push registry.company.com/todo-api:latest
```

> 💡 **Tip:** Your Platform Team provides a container registry. Ask for credentials if needed.

### Step 3: Create Secrets

Your app needs secrets (JWT secret, API keys).

**Option A: Using Platform Portal**
1. Go to https://portal.company.com/secrets
2. Create secrets:
   - `jwt-secret`: Your JWT signing secret
   - `api-key`: Your API key

**Option B: Using CLI**
```bash
innominatus-ctl secrets create jwt-secret --value "your-secret-here"
innominatus-ctl secrets create api-key --value "your-api-key"
```

### Step 4: Deploy Application

```bash
# Deploy with the Score specification
innominatus-ctl deploy score.yaml
```

**Expected output:**

```
🚀 Starting deployment: todo-api

Workflow: deploy-app (8 steps)
═══════════════════════════════════════

[1/8] ✓ validate-spec          (0.5s)
      Validating Score specification...
      ✓ Valid

[2/8] ✓ provision-database      (45s)
      Creating PostgreSQL database...
      Database: todo-api-db
      Version: 15
      Size: small (2 vCPU, 4GB RAM)
      Storage: 20GB SSD
      ✓ Database provisioned
      ✓ Backup enabled (7 days retention)

[3/8] ✓ provision-cache         (15s)
      Creating Redis cache...
      ✓ Redis provisioned (1GB memory)

[4/8] ✓ run-migrations          (8s)
      Running database migrations...
      ✓ 3 migrations applied

[5/8] ✓ deploy-application      (30s)
      Deploying container: todo-api:latest
      Replicas: 3
      ✓ Deployment successful
      ✓ 3/3 pods healthy

[6/8] ✓ create-ingress          (5s)
      Creating HTTPS ingress...
      Host: todo-api.company.com
      ✓ SSL certificate configured
      ✓ Rate limiting: 100/min

[7/8] ✓ health-check            (15s)
      Waiting for pods to be ready...
      ✓ All health checks passing

[8/8] ✓ register-monitoring     (2s)
      Configuring monitoring...
      ✓ Prometheus metrics enabled
      ✓ Grafana dashboard created

═══════════════════════════════════════
✅ Deployment completed successfully!
⏱  Total time: 2m 5s

📊 Application Details:
   Name: todo-api
   Replicas: 3/3 healthy
   URL: https://todo-api.company.com

🔗 Resources Created:
   Database: todo-api-db (PostgreSQL 15, 20GB)
   Cache: todo-api-cache (Redis 7, 1GB)
   Ingress: https://todo-api.company.com

📈 Monitoring:
   Metrics: https://grafana.company.com/d/todo-api
   Logs: innominatus-ctl logs todo-api --follow

🎯 Next Steps:
   • Test API: curl https://todo-api.company.com/health
   • View status: innominatus-ctl status todo-api
   • View logs: innominatus-ctl logs todo-api
```

### Step 5: Verify Deployment

```bash
# Test health endpoint
curl https://todo-api.company.com/health

# Expected response:
# {"status":"healthy"}

# Test your API
curl https://todo-api.company.com/api/todos

# View logs
innominatus-ctl logs todo-api --follow

# Check status
innominatus-ctl status todo-api
```

✅ **Success!** Your API is deployed and running with database!

---

## Common Operations

### Update Your Application

1. Make code changes
2. Build new image:
   ```bash
   docker build -t registry.company.com/todo-api:v2 .
   docker push registry.company.com/todo-api:v2
   ```

3. Update `score.yaml`:
   ```yaml
   containers:
     api:
       image: registry.company.com/todo-api:v2
   ```

4. Deploy update:
   ```bash
   innominatus-ctl deploy score.yaml
   ```

**Result:** Zero-downtime rolling update (one pod at a time)

### Scale Your Application

**Auto-scaling is already configured!** Platform will scale based on CPU.

**Manual scaling:**
```yaml
deployment:
  replicas: 5  # Change from 3 to 5
```

```bash
innominatus-ctl deploy score.yaml
```

### View Logs

```bash
# Recent logs
innominatus-ctl logs todo-api

# Live logs (follow mode)
innominatus-ctl logs todo-api --follow

# Filter by level
innominatus-ctl logs todo-api --level error

# Last 100 lines
innominatus-ctl logs todo-api --tail 100
```

### Database Operations

**Connect to Database:**
```bash
# Get database credentials
innominatus-ctl resources todo-api

# Connect with psql
psql "postgresql://user:pass@host:5432/dbname"
```

**Run Migrations:**
```bash
# SSH into pod and run migrations
innominatus-ctl exec todo-api -- npm run migrate
```

**Backup Database:**
```bash
# Trigger manual backup
innominatus-ctl backup create todo-api-db
```

### Monitor Your Application

**Grafana Dashboard:**
```bash
innominatus-ctl dashboard todo-api
# Opens Grafana dashboard in browser
```

**Metrics you'll see:**
- Request rate (requests/second)
- Response time (p50, p95, p99)
- Error rate
- Pod health
- Database connections
- Memory/CPU usage

**Set up Alerts:**
Your Platform Team has configured default alerts:
- 🚨 Error rate > 5%
- 🚨 Response time > 1s (p95)
- 🚨 Pod restart > 3 times/hour

Alerts go to: `#backend-alerts` Slack channel

---

## Troubleshooting

### Deployment Failed

**Error:** `Error: health check failed`

**Debug:**
```bash
# Check pod status
innominatus-ctl status todo-api

# View logs
innominatus-ctl logs todo-api

# Common issues:
# 1. Database connection failed
#    → Check DATABASE_URL is set correctly
#    → Check database is provisioned
# 2. Port mismatch
#    → App must listen on PORT from env var
# 3. Health endpoint not responding
#    → Add /health endpoint to your app
```

### Database Connection Issues

**Error:** `Error: connection to database failed`

**Fix:**
```javascript
// Ensure your app uses DATABASE_URL from environment
const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  // Add connection pooling
  max: 20,
  connectionTimeoutMillis: 5000,
});

// Add retry logic
async function connectWithRetry() {
  const maxRetries = 5;
  for (let i = 0; i < maxRetries; i++) {
    try {
      await pool.query('SELECT 1');
      console.log('Database connected');
      return;
    } catch (err) {
      console.log(`Retry ${i + 1}/${maxRetries}...`);
      await new Promise(resolve => setTimeout(resolve, 2000));
    }
  }
  throw new Error('Could not connect to database');
}
```

### Secrets Not Available

**Error:** `Error: JWT_SECRET is undefined`

**Fix:**
```bash
# Check secrets exist
innominatus-ctl secrets list

# If missing, create them
innominatus-ctl secrets create jwt-secret --value "your-secret"

# Redeploy
innominatus-ctl deploy score.yaml
```

### Need More Help?

→ [Full Troubleshooting Guide](../troubleshooting.md)

---

## Complete Example

Full working example available at:
→ https://github.com/philipsahli/innominatus/tree/main/examples/nodejs-postgres

**What's included:**
- ✅ Express.js app with REST API
- ✅ PostgreSQL connection with migrations
- ✅ Redis session store
- ✅ Health check endpoints
- ✅ Dockerfile (multi-stage build)
- ✅ Score specification
- ✅ Database migrations
- ✅ Tests

**Quick start:**
```bash
# Clone example
git clone https://github.com/philipsahli/innominatus
cd examples/nodejs-postgres

# Review the code
cat app/server.js
cat score.yaml

# Build and deploy
docker build -t registry.company.com/my-api:latest app/
docker push registry.company.com/my-api:latest
innominatus-ctl deploy score.yaml
```

---

## Next Steps

🎉 **Congratulations!** You've deployed a production-ready Node.js API with database!

**What to do next:**

**Add More Features:**
- [Add Redis Caching](./caching.md)
- [Set up CI/CD](./cicd-integration.md)
- [Blue-Green Deployment](./blue-green-deployment.md)
- [Database Migrations](./database-migrations.md)

**Explore Other Recipes:**
- [Python Worker with Queue](./python-worker.md)
- [Microservices Architecture](./microservices.md)
- [Serverless Functions](./serverless.md)

**Advanced Topics:**
- [Custom Golden Paths](../../platform-team-guide/golden-paths.md)
- [Multi-Environment Setup](../guides/environments.md)
- [Monitoring & Alerting](../guides/monitoring.md)

**Get Help:**
- 💬 Slack: `#platform-support`
- 📚 [User Guide](../README.md)
- 📖 [CLI Reference](../cli-reference.md)

---

**Deployment time:** 15 minutes ✓
**Production-ready:** ✓
**Auto-scaling:** ✓
**Monitoring:** ✓

You're ready to ship! 🚀
