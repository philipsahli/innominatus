# Innominatus Demo Playbook

**Version:** 1.0
**Date:** 2025-11-10
**Purpose:** Comprehensive demo scenarios for CLI, Web UI, and AI Assistant interfaces

---

## Table of Contents

1. [Demo Setup](#demo-setup)
2. [CLI Demo Scenarios](#cli-demo-scenarios) - 4 scenarios (20 minutes)
3. [Web UI Demo Scenarios](#web-ui-demo-scenarios) - 3 scenarios (9 minutes)
4. [AI Assistant Demo Scenarios](#ai-assistant-demo-scenarios) - 4 scenarios (16 minutes)
5. [End-to-End Demo Flow](#end-to-end-demo-flow) - 1 comprehensive scenario (17 minutes)
6. [Troubleshooting Demo Scenarios](#troubleshooting-demo-scenarios) - 2 scenarios

---

## Demo Setup

### Prerequisites

```bash
# 1. Ensure server is running
./innominatus

# 2. Verify server is healthy
curl http://localhost:8081/health

# 3. Verify CLI authentication
./innominatus-ctl list-resources

# 4. Open browser tabs
# - Web UI: http://localhost:8081
# - Swagger: http://localhost:8081/swagger-user
```

### Demo Environment

**Audience Types:**
- **Developer Persona**: Focus on self-service, golden paths, AI assistance
- **Platform Team Persona**: Focus on providers, workflows, multi-tenancy
- **Executive Persona**: Focus on efficiency, automation, governance

**Key Messages:**
- Self-service platform for developers
- Provider-based multi-team architecture
- Event-driven orchestration with automatic resource provisioning
- AI-powered assistance for natural language interactions

---

## CLI Demo Scenarios

### Scenario 1: Developer Self-Service (5 minutes)

**Persona:** Developer (Alex) needs a PostgreSQL database for a new microservice

**Demo Script:**

```bash
# 1. List available providers and capabilities
./innominatus-ctl list-providers

# Output shows:
# ✅ database-team (infrastructure)
#    Capabilities: postgres, postgresql, mysql
#    Workflows: 4 provisioners, 0 golden paths

# 2. List golden paths
./innominatus-ctl list-goldenpaths

# Output shows available templates:
# - onboard-dev-team
# - provision-postgres
# - provision-namespace

# 3. Deploy a Score spec with postgres resource
# Create Score spec defining the database
cat > /tmp/ecommerce-checkout-db.yaml <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: ecommerce-backend

containers:
  main:
    image: ecommerce/backend:latest
    env:
      DATABASE_URL: ${resources.checkout-db.connection_string}

resources:
  checkout-db:
    type: postgres
    properties:
      version: "15"
      size: medium
      replicas: 2
EOF

# Deploy with real-time watch
./innominatus-ctl deploy /tmp/ecommerce-checkout-db.yaml -w

# Output:
# ✅ Score specification deployed: ecommerce-backend
#    📦 Resource detected: checkout-db (postgres)
#    🔄 Workflow executing: provision-postgres (workflow ID: 6)
#
# 🔄 Workflow Executing: provision-postgres
# ✅ Step 1: create-namespace (completed in 2.3s)
# ✅ Step 2: create-postgres-cluster (completed in 15.7s)
# 🔄 Step 3: wait-for-database (running, 23s elapsed)
# ✅ Workflow completed successfully
#
#    Resource checkout-db: requested → provisioning → active
#    📊 View details: http://localhost:8081/resources/5

# 4. Get database credentials
./innominatus-ctl list-resources --type postgres

# Output shows connection details:
# ✅ checkout-db (active)
#    Host: ecommerce-backend-checkout-db.ecommerce-backend.svc.cluster.local
#    Port: 5432
#    Credentials Secret: ecommerce-backend-checkout-db.checkout-db-app.credentials
```

**Key Talking Points:**
- ✅ Zero infrastructure knowledge required
- ✅ Automatic workflow selection via provider capabilities
- ✅ Real-time progress visibility
- ✅ Production-ready database in minutes

---

### Scenario 2: Platform Team Operations (7 minutes)

**Persona:** Platform engineer (Jordan) managing providers and workflows

**Demo Script:**

```bash
# 1. View provider details and capabilities
./innominatus-ctl provider-detail database-team

# Output:
# Provider: database-team
# Category: infrastructure
# Version: 1.0.0
#
# Capabilities:
#   - postgres (primary)
#   - postgresql (alias)
#   - mysql (experimental)
#
# Workflows:
#   ✅ provision-postgres (create)
#   ✅ update-postgres (update)
#   ✅ delete-postgres (delete)
#   ✅ provision-postgres-mock (create, test)

# 2. Validate workflow before deployment
./innominatus-ctl validate providers/database-team/workflows/provision-postgres.yaml

# Output:
# ✅ Workflow validation passed
#    - All parameters defined
#    - All step types valid (kubernetes, policy)
#    - All templates render correctly

# 3. Execute golden path for team onboarding
./innominatus-ctl run onboard-dev-team <<EOF
team_name: mobile-team
namespace: mobile-team
gitea_org: mobile-team
keycloak_group: mobile-team-developers
repos:
  - mobile-app-ios
  - mobile-app-android
  - mobile-api-gateway
EOF

# Output:
# 🔄 Executing golden path: onboard-dev-team
# Execution ID: 7
#
# Steps:
#   1. Create Keycloak group (queued)
#   2. Create Gitea organization (queued)
#   3. Create repositories (queued)
#   4. Create Kubernetes namespace (queued)
#   5. Setup ArgoCD project (queued)

# 4. Monitor multi-step workflow
./innominatus-ctl workflow detail 7

# Output:
# ✅ Step 1: Create Keycloak group (completed)
# ✅ Step 2: Create Gitea organization (completed)
# 🔄 Step 3: Create repositories (running)
# ⏳ Step 4: Create Kubernetes namespace (pending)
# ⏳ Step 5: Setup ArgoCD project (pending)

# 5. Verify all resources created
./innominatus-ctl list-resources --application mobile-team

# Output:
# 📦 Application: mobile-team (5 resources)
#    ✅ mobile-team-namespace (namespace, active)
#    ✅ mobile-app-ios-repo (gitea-repo, active)
#    ✅ mobile-app-android-repo (gitea-repo, active)
#    ✅ mobile-api-gateway-repo (gitea-repo, active)
#    ✅ mobile-team-argocd (argocd-app, active)
```

**Key Talking Points:**
- ✅ Golden paths encode best practices
- ✅ Multi-resource orchestration with dependencies
- ✅ Validation before execution prevents errors
- ✅ Complete team onboarding in one command

---

### Scenario 3: Adding Resources Incrementally (5 minutes)

**Persona:** Developer (Casey) needs to add S3 storage to an existing application

**Demo Script:**

```bash
# 1. Check existing application (initially deployed with just database)
./innominatus-ctl list-resources --application ecommerce-backend

# Output:
# 📦 Application: ecommerce-backend (1 resource)
#    ✅ db (postgres, active)
#       Host: ecommerce-backend-db.ecommerce-backend.svc.cluster.local
#       Port: 5432

# 2. Developer realizes they need S3 storage for product images
#    Let's check what providers handle storage

# 3. Check what storage providers are available
./innominatus-ctl list-providers

# Output shows:
# ✅ storage-team (infrastructure)
#    Capabilities: s3, s3-bucket, object-storage, minio-bucket
#    Workflows: 3 provisioners, 1 golden path

# 4. Update the Score spec to add S3 storage
cat > score-ecommerce-backend-v2.yaml <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: ecommerce-backend

containers:
  main:
    image: myorg/ecommerce-backend:v1.1.0
    env:
      DATABASE_URL: ${resources.db.connection_string}
      # NEW: S3 credentials injected automatically
      S3_ENDPOINT: ${resources.storage.endpoint}
      S3_BUCKET: ${resources.storage.bucket}
      S3_ACCESS_KEY: ${resources.storage.access_key}
      S3_SECRET_KEY: ${resources.storage.secret_key}

resources:
  db:
    type: postgres  # Existing database (already provisioned)
    properties:
      version: "15"
      size: "medium"
      replicas: 2

  storage:         # NEW: Adding S3 storage
    type: s3
    properties:
      size: "standard"
      versioning: true
      public_read: false
EOF

# 5. Deploy updated Score spec with --watch for real-time progress
./innominatus-ctl deploy score-ecommerce-backend-v2.yaml -w

# Output:
# ✅ Score spec validated
# ℹ️  Detected existing resource: db (postgres) - Skipping provisioning
# 🆕 Detected new resource: storage (s3) - Provisioning via storage-team
#
# 🔄 Provisioning storage (s3)...
#    Workflow ID: 26
#    Provider: storage-team
#
# ✅ Step 1: create-minio-bucket (completed in 1.2s)
# ✅ Step 2: configure-bucket-policy (completed in 0.8s)
# ✅ Step 3: generate-access-credentials (completed in 0.5s)
#
# ✅ Resource storage is now ACTIVE (total: 2.5s)
# ✅ Deployment completed successfully

# 6. Verify both resources now exist for the application
./innominatus-ctl list-resources --application ecommerce-backend

# Output:
# 📦 Application: ecommerce-backend (2 resources)
#
# ✅ db (postgres, active)
#    Host: ecommerce-backend-db.ecommerce-backend.svc.cluster.local
#    Port: 5432
#    Credentials: ecommerce-backend-db-app.credentials
#
# ✅ storage (s3, active)  [NEWLY ADDED]
#    Bucket: ecommerce-backend-storage
#    Endpoint: minio.minio.svc.cluster.local:9000
#    Credentials: ecommerce-backend-storage-credentials

# 7. View dependency graph in Web UI
# Navigate to: http://localhost:8081/resources
# Shows:
#
# [Score Spec: ecommerce-backend]
#        ↓
# ┌──────┴────────┐
# │               │
# [db]        [storage]
#  ↓              ↓
# [database-team] [storage-team]
```

**Key Talking Points:**
- ✅ Resources can be added incrementally (no need to plan everything upfront)
- ✅ Multiple providers coordinate automatically (database-team + storage-team)
- ✅ Application name groups related resources
- ✅ Each resource triggers its own workflow via the correct provider
- ✅ Fast provisioning (S3 bucket ready in ~2-3 seconds vs. database ~5 minutes)
- ✅ Dependency graph shows complete application architecture

**Real-World Use Cases:**
- Start with database, add caching later (Redis)
- Add message queue (Kafka/RabbitMQ) when scaling
- Add object storage for file uploads
- Add monitoring (Prometheus) after deployment

---

### Scenario 4: Troubleshooting Failed Workflows (4 minutes)

**Persona:** Developer (Sam) debugging a failed deployment

**Demo Script:**

```bash
# 1. List failed resources
./innominatus-ctl list-resources --state failed

# Output:
# ❌ Failed Resources (1):
#    Resource #3: analytics-db (postgres)
#    Error: Timeout waiting for PostgreSQL cluster
#    Last Updated: 2025-11-10 20:47:16

# 2. View workflow execution details
./innominatus-ctl workflow logs 8

# Output:
# ❌ Workflow Execution #8
# Application: analytics-platform
# Workflow: provision-postgres
# Status: failed
# Error: policy script failed: exit status 1
#
# ✅ Step 1: create-namespace (completed)
# ✅ Step 2: create-postgres-cluster (completed)
# ❌ Step 3: wait-for-database (failed)
#    ❌ ERROR: policy script failed: exit status 1
#    Logs:
#      Waiting for PostgreSQL cluster to be ready...
#      Waiting... attempt 1/60
#      Waiting... attempt 2/60
#      ...
#      Waiting... attempt 60/60
#      Timeout waiting for PostgreSQL cluster
#      exit status 1

# 3. Check Kubernetes resources directly
kubectl get postgresql -n analytics-platform

# Output:
# NAME                           STATUS     AGE
# analytics-platform-analytics-db  Creating   10m
# (Cluster stuck in Creating state - likely resource constraints)

# 4. Fix resource limits and retry
./innominatus-ctl update-resource 3 \
  --param size=small \
  --param replicas=1

# Output:
# ✅ Resource update triggered
#    New Workflow ID: 9
#    State: updating
```

**Key Talking Points:**
- ✅ Error messages always visible (no --verbose needed)
- ✅ Full log output for policy scripts
- ✅ Context-aware messages for missing logs
- ✅ Direct link to Kubernetes for deep debugging

---

## Web UI Demo Scenarios

### Scenario 1: Visual Workflow Monitoring (3 minutes)

**Persona:** Developer (Maria) monitoring deployment progress

**Demo Script:**

1. **Navigate to Workflows page**
   - URL: http://localhost:8081/workflows
   - Shows list of all workflow executions with status

2. **Open running workflow**
   - Click on "provision-postgres" workflow (status: running)
   - URL: http://localhost:8081/workflows/10

3. **Observe progress indicator**
   ```
   ┌─────────────────────────────────────────────────────┐
   │ 🔄 Workflow Executing        2 / 4 steps completed  │
   │ ████████████░░░░░░░░░░░░░░░░ 50%                   │
   │ Currently executing: Step 3 - wait-for-database    │
   │ (23s elapsed)                                       │
   └─────────────────────────────────────────────────────┘
   ```

4. **View step details**
   - Expand completed step "create-namespace"
   - Shows:
     - ✅ Status badge (completed)
     - Duration: 2.3s
     - Output logs (kubectl apply output)

5. **Monitor error step**
   - Workflow fails on step 3
   - Red error banner appears:
     ```
     ❌ Step 3: wait-for-database (policy)

     Error Details:
     policy script failed: exit status 1

     Output:
     Timeout waiting for PostgreSQL cluster
     ```

**Key Talking Points:**
- ✅ Real-time progress without page refresh
- ✅ Visual progress bar with percentage
- ✅ Error details prominently displayed
- ✅ Full log output for debugging

---

### Scenario 2: Resource Management Dashboard (4 minutes)

**Persona:** Platform team lead (Alex) reviewing resource inventory

**Demo Script:**

1. **Navigate to Resources page**
   - URL: http://localhost:8081/resources
   - Shows all provisioned resources grouped by application

2. **Filter by resource type**
   - Select "postgres" from dropdown
   - Shows only PostgreSQL databases:
     ```
     ✅ checkout-db (ecommerce-backend)
     ✅ user-db (authentication-service)
     ❌ analytics-db (analytics-platform) - FAILED
     ```

3. **View resource details**
   - Click on "checkout-db"
   - Shows:
     - Configuration: version=15, size=medium, replicas=2
     - Connection details: host, port, credentials secret
     - Associated workflow execution (link)
     - Created: 2025-11-10 15:23:45
     - State transitions: requested → provisioning → active

4. **View dependency graph**
   - Click "View Graph" button
   - Shows visual representation:
     ```
     [Score Spec: ecommerce-backend]
            ↓
     [Resource: checkout-db (postgres)]
            ↓
     [Provider: database-team]
            ↓
     [Workflow: provision-postgres]
     ```

**Key Talking Points:**
- ✅ Complete resource inventory across teams
- ✅ Visual dependency tracking
- ✅ Automatic provider-to-resource mapping
- ✅ Audit trail via state transitions

---

### Scenario 3: Provider Catalog (2 minutes)

**Persona:** Developer (Sam) discovering available services

**Demo Script:**

1. **Navigate to Providers page**
   - URL: http://localhost:8081/providers
   - Shows all registered providers with categories

2. **View infrastructure providers**
   - Filter by category: "infrastructure"
   - Shows:
     ```
     ✅ database-team
        Capabilities: postgres, mysql, redis
        Workflows: 6 provisioners, 0 golden paths

     ✅ storage-team
        Capabilities: s3, minio-bucket, object-storage
        Workflows: 3 provisioners, 1 golden path

     ✅ container-team
        Capabilities: namespace, gitea-repo, argocd-app
        Workflows: 8 provisioners, 2 golden paths
     ```

3. **View provider details**
   - Click on "database-team"
   - Shows:
     - Description
     - Version: 1.0.0
     - Category: infrastructure
     - Capabilities with aliases
     - Available workflows with descriptions
     - Documentation links

**Key Talking Points:**
- ✅ Self-service catalog of platform capabilities
- ✅ Clear ownership (team-based providers)
- ✅ Version tracking for compatibility
- ✅ Documentation integrated

---

## AI Assistant Demo Scenarios

### Scenario 1: Natural Language Resource Provisioning (5 minutes)

**Persona:** Developer (Taylor) using AI assistant for the first time

**Demo Script:**

1. **Navigate to AI Assistant**
   - URL: http://localhost:8081/ai-assistant
   - Shows chat interface

2. **Ask about capabilities**
   ```
   User: "What can you help me with?"

   AI: "I can help you with:
   - Provisioning resources (databases, storage, namespaces)
   - Deploying applications via golden paths
   - Monitoring workflow executions
   - Troubleshooting failed resources
   - Discovering available providers and capabilities

   For example, you can say:
   - 'Create a PostgreSQL database for my app'
   - 'Show me all failed workflows'
   - 'What providers handle object storage?'
   "
   ```

3. **Create database via natural language**
   ```
   User: "I need a PostgreSQL database for my payment service.
          It should be production-grade with high availability."

   AI: "I'll create a PostgreSQL database for your payment service.
        Based on your requirements:
        - Version: 15 (latest stable)
        - Size: large (production workload)
        - Replicas: 3 (high availability)

        Creating resource... ✓

        Resource ID: 12
        Name: payment-db
        Application: payment-service
        State: provisioning

        Workflow execution started (ID: 15)
        Track progress: http://localhost:8081/workflows/15

        ETA: ~5 minutes"
   ```

4. **Check status via chat**
   ```
   User: "How's my database doing?"

   AI: "Your PostgreSQL database (payment-db) is:

        Status: ✅ ACTIVE

        Details:
        - Cluster: payment-service-payment-db
        - Namespace: payment-service
        - Replicas: 3/3 running
        - Version: PostgreSQL 15

        Connection Info:
        - Host: payment-service-payment-db.payment-service.svc.cluster.local
        - Port: 5432
        - Credentials: payment-service-payment-db.payment-db-app.credentials

        Your database is ready to use!"
   ```

**Key Talking Points:**
- ✅ Natural language interface (no commands to memorize)
- ✅ AI infers requirements from context
- ✅ Production best practices applied automatically
- ✅ Conversational monitoring and troubleshooting

---

### Scenario 2: AI-Powered Troubleshooting (4 minutes)

**Persona:** Developer (Jordan) debugging a failed deployment

**Demo Script:**

```
User: "My deployment failed. Can you help?"

AI: "I found 1 failed resource in your account:

    ❌ analytics-db (postgres) - FAILED
    Application: analytics-platform
    Error: Timeout waiting for PostgreSQL cluster

    Let me investigate the workflow execution...

    Workflow #8 (provision-postgres) failed at Step 3:

    Step: wait-for-database (policy)
    Error: Timeout after 60 attempts (10 minutes)
    Logs:
      'Waiting for PostgreSQL cluster to be ready...'
      'Waiting... attempt 60/60'
      'Timeout waiting for PostgreSQL cluster'

    Root Cause Analysis:
    The PostgreSQL operator created the cluster, but it's stuck
    in 'Creating' state. This usually indicates:
    1. Insufficient cluster resources (CPU/memory)
    2. Storage class issues
    3. Network policy blocking operator communication

    I checked the Kubernetes cluster:
    $ kubectl get postgresql analytics-platform-analytics-db -n analytics-platform
    Status: Creating (stuck)

    Recommendation:
    Reduce resource requirements and try again:
    - Change size from 'large' to 'medium'
    - Reduce replicas from 3 to 2

    Would you like me to retry with these settings?"

User: "Yes, please retry"

AI: "Creating updated resource with optimized settings...

    ✓ Resource updated (ID: 3)
    ✓ New workflow started (ID: 16)

    Progress:
    🔄 Step 1: create-namespace (completed)
    🔄 Step 2: create-postgres-cluster (completed)
    🔄 Step 3: wait-for-database (running - 15s elapsed)

    I'll notify you when it's ready."

[2 minutes later]

AI: "✅ Your database is now ACTIVE!

    The issue was resource constraints on the cluster. By
    reducing to 'medium' size, the PostgreSQL cluster
    started successfully.

    Connection details:
    - Host: analytics-platform-analytics-db.analytics-platform.svc.cluster.local
    - Port: 5432"
```

**Key Talking Points:**
- ✅ AI analyzes failed workflows automatically
- ✅ Root cause analysis with explanations
- ✅ Actionable recommendations
- ✅ Self-healing via automated retry

---

### Scenario 3: Knowledge Base Queries (3 minutes)

**Persona:** New developer (Sam) learning the platform

**Demo Script:**

```
User: "How do I deploy an application?"

AI: "There are three ways to deploy applications in innominatus:

    1. Direct API (simple deployments)
       POST /api/specs with Score specification

    2. CLI (scripting/automation)
       ./innominatus-ctl deploy score-spec.yaml

    3. Golden Paths (production workflows)
       ./innominatus-ctl run deploy-app score-spec.yaml

    I recommend starting with Golden Paths because they:
    - Include production best practices
    - Handle multi-resource dependencies
    - Include validation and health checks

    Would you like me to show you an example?"

User: "What's the difference between a provider and a workflow?"

AI: "Great question! Here's the relationship:

    Provider (Team Ownership)
    ├─ Capabilities: Resource types it can provision
    │  Example: database-team handles [postgres, mysql, redis]
    │
    └─ Workflows: Implementation of provisioning logic
       ├─ provision-postgres.yaml (CREATE operation)
       ├─ update-postgres.yaml (UPDATE operation)
       └─ delete-postgres.yaml (DELETE operation)

    Example Flow:
    1. Developer requests 'postgres' resource
    2. Resolver matches 'postgres' → database-team provider
    3. Engine executes 'provision-postgres' workflow
    4. Resource becomes active

    Providers enable:
    - Team autonomy (each team manages their domain)
    - Clear ownership boundaries
    - Centralized governance
    - Automatic workflow selection"

User: "Show me all providers that handle databases"

AI: "Providers with database capabilities:

    ✅ database-team (infrastructure)
       Capabilities: postgres, postgresql, mysql, redis
       Workflows: 6 provisioners
       Owner: Platform Team

    ✅ nosql-team (infrastructure)
       Capabilities: mongodb, cassandra, dynamodb
       Workflows: 3 provisioners
       Owner: Data Team

    For PostgreSQL, create a Score spec defining your database:

    apiVersion: score.dev/v1b1
    metadata:
      name: <app-name>
    resources:
      <db-name>:
        type: postgres
        properties:
          version: '15'
          size: medium

    Then deploy with: ./innominatus-ctl deploy score-spec.yaml -w"
```

**Key Talking Points:**
- ✅ Context-aware knowledge base (docs + code)
- ✅ Educational responses with examples
- ✅ Links to relevant documentation
- ✅ Learns from workflow executions

---

### Scenario 4: AI-Assisted Incremental Resource Addition (4 minutes)

**Persona:** Developer (Morgan) needs to add storage to an existing application

**Demo Script:**

```
User: "I have an application called 'blog-platform' with a postgres database. Now I need to add S3 storage for user-uploaded images."

AI: "I can help you add S3 storage to your existing blog-platform application.

    Current resources for blog-platform:
    ✅ db (postgres, active)
       - Version: PostgreSQL 15
       - Size: medium
       - Status: Healthy

    To add S3 storage, we'll update your Score specification to include
    a new 'images' resource. Here's the updated spec:

    ```yaml
    apiVersion: score.dev/v1b1
    metadata:
      name: blog-platform

    containers:
      main:
        image: myorg/blog-platform:v1.1.0
        env:
          # Existing database credentials
          DATABASE_URL: ${resources.db.connection_string}
          # NEW: S3 credentials for image storage
          S3_ENDPOINT: ${resources.images.endpoint}
          S3_BUCKET: ${resources.images.bucket}
          S3_ACCESS_KEY: ${resources.images.access_key}
          S3_SECRET_KEY: ${resources.images.secret_key}

    resources:
      db:
        type: postgres
        properties:
          version: "15"
          size: "medium"
          replicas: 2

      images:  # NEW: S3 storage for user-uploaded images
        type: s3
        properties:
          size: "standard"
          versioning: true
          public_read: false
    ```

    I'll save this as score-blog-platform-v2.yaml and deploy it.
    The system will detect the existing database and only provision
    the new S3 bucket.

    Should I proceed?"

User: "Yes, deploy it"

AI: "Deploying updated Score specification...

    $ ./innominatus-ctl deploy score-blog-platform-v2.yaml -w

    ✅ Score spec validated
    ℹ️  Detected existing resource: db (postgres) - Skipping
    🆕 Detected new resource: images (s3) - Provisioning via storage-team

    🔄 Provisioning images (s3)...
       Workflow ID: 87
       Provider: storage-team

    ✅ Step 1: create-minio-bucket (completed in 1.1s)
    ✅ Step 2: configure-bucket-policy (completed in 0.7s)
    ✅ Step 3: generate-access-credentials (completed in 0.4s)

    ✅ Resource images is now ACTIVE!

    Your blog-platform now has 2 resources:

    1. db (postgres)
       - Connection: blog-platform-db.blog-platform.svc.cluster.local:5432

    2. images (s3)  [NEWLY ADDED]
       - Bucket: blog-platform-images
       - Endpoint: minio.minio.svc.cluster.local:9000
       - Credentials: blog-platform-images-credentials

    Total provisioning time: 2.2 seconds

    ✅ Deployment completed successfully!

    Your application will be restarted with the new S3 environment
    variables automatically injected from the resource outputs."

User: "How are the S3 credentials actually passed to my app?"

AI: "Great question! The S3 credentials are automatically injected via
    Score's resource variable substitution:

    1. Score Spec declares dependency:
       ```yaml
       S3_ACCESS_KEY: ${resources.images.access_key}
       ```

    2. innominatus provisions the S3 bucket (via storage-team provider)

    3. Workflow outputs the credentials:
       ```yaml
       outputs:
         access_key: "AKIAIOSFODNN7EXAMPLE"
         secret_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
         endpoint: "minio.minio.svc.cluster.local:9000"
         bucket: "blog-platform-images"
       ```

    4. These outputs are substituted into your container env vars

    5. Your application receives fully-configured environment:
       ```
       S3_ENDPOINT=minio.minio.svc.cluster.local:9000
       S3_BUCKET=blog-platform-images
       S3_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
       S3_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
       ```

    No manual secret management needed - everything is wired up
    automatically through the Score specification!"
```

**Key Talking Points:**
- ✅ AI understands existing application context
- ✅ Suggests appropriate configuration for use case
- ✅ Shows resource relationships (database + storage)
- ✅ Provides integration guidance automatically
- ✅ Fast provisioning (seconds vs. minutes)
- ✅ Multiple resources managed under one application name

**Real-World Benefits:**
- No need to plan all resources upfront
- Add capabilities as application evolves
- Each resource independently provisioned
- Clear application grouping for governance

---

## End-to-End Demo Flow

### Complete Developer Journey (15 minutes)

**Persona:** Developer (Alex) deploying a new microservice with database

**Narrative:**
Alex is building a new order management service that needs a PostgreSQL database and S3 storage for order receipts. They've never used innominatus before.

**Demo Script:**

#### Part 1: Discovery (2 minutes)

```bash
# Alex: "What can this platform do for me?"

# Via AI Assistant
User: "I need to deploy an order management service with a database. How do I start?"

AI: "I can help! For your order management service, you'll need:
     1. PostgreSQL database (handled by database-team)
     2. S3 storage for receipts (handled by storage-team)
     3. Kubernetes namespace (handled by container-team)

     I can provision all of these using golden paths. Would you like me to:
     A) Provision resources individually
     B) Use the 'deploy-app-with-db' golden path (recommended)"

User: "What's the difference?"

AI: "Golden paths are pre-configured workflows that:
     - Set up multiple resources in the right order
     - Apply production best practices
     - Include monitoring and health checks
     - Handle dependencies automatically

     For your use case, 'deploy-app-with-db' will:
     1. Create namespace
     2. Provision PostgreSQL
     3. Provision S3 bucket
     4. Deploy your app with injected credentials

     Want to see the configuration?"

User: "Yes, show me"
```

#### Part 2: Initial Provisioning - Database (3 minutes)

```bash
# Alex decides to start with just the database (common real-world pattern)

# CLI approach
./innominatus-ctl list-goldenpaths

# Output:
# Available Golden Paths:
# ✅ onboard-dev-team - Complete team setup
# ✅ deploy-app-with-db - Deploy app with database and storage
# ✅ ephemeral-env - Create temporary environment

# Create initial Score spec with database only
cat > score-order-service-v1.yaml <<'EOF'
apiVersion: score.dev/v1b1

metadata:
  name: order-service

containers:
  main:
    image: myorg/order-service:v1.0.0
    env:
      DATABASE_URL: ${resources.db.connection_string}
      DATABASE_HOST: ${resources.db.host}
      DATABASE_PORT: ${resources.db.port}
      DATABASE_NAME: ${resources.db.database_name}
      DATABASE_USER: ${resources.db.username}
      DATABASE_PASSWORD: ${resources.db.password}

resources:
  db:
    type: postgres
    properties:
      version: "15"
      size: "medium"
      replicas: 2
EOF

# Deploy with watch mode for real-time progress
./innominatus-ctl deploy score-order-service-v1.yaml -w

# Output shows real-time progress:
# ✅ Score spec validated
# 🔄 Provisioning db (postgres)...
#    Workflow ID: 21
#    Provider: database-team
#
# ✅ Step 1: create-namespace (completed, 2.1s)
# ✅ Step 2: create-postgres-cluster (completed, 18.4s)
# 🔄 Step 3: wait-for-database (running, 12s elapsed)
# ⏳ Step 4: get-credentials (pending)

# Open Web UI to see visual progress
# Navigate to: http://localhost:8081/workflows/21
```

#### Part 2b: Adding Storage Later (2 minutes)

**Narrative:** After testing the order service with the database, Alex realizes they also need S3 storage for order receipt PDFs. This demonstrates incremental resource addition.

```bash
# Database is now active, application is running

# Check current resources
./innominatus-ctl list-resources --application order-service

# Output:
# 📦 Application: order-service (1 resource)
# ✅ db (postgres, active)

# Alex: "I need to add S3 storage for order receipts now"

# Update Score spec to add S3 storage
cat > score-order-service-v2.yaml <<'EOF'
apiVersion: score.dev/v1b1

metadata:
  name: order-service

containers:
  main:
    image: myorg/order-service:v1.1.0  # Updated version
    env:
      # Existing database credentials
      DATABASE_URL: ${resources.db.connection_string}
      # NEW: S3 storage credentials
      S3_ENDPOINT: ${resources.receipts.endpoint}
      S3_BUCKET: ${resources.receipts.bucket}
      S3_ACCESS_KEY: ${resources.receipts.access_key}
      S3_SECRET_KEY: ${resources.receipts.secret_key}

resources:
  db:
    type: postgres  # Existing - will not be reprovisioned
    properties:
      version: "15"
      size: "medium"
      replicas: 2

  receipts:  # NEW: S3 storage for order receipts
    type: s3
    properties:
      size: "standard"
      versioning: true
EOF

# Deploy updated spec with watch mode
./innominatus-ctl deploy score-order-service-v2.yaml -w

# Output:
# ✅ Score spec validated
# ℹ️  Detected existing resource: db (postgres) - Skipping provisioning
# 🆕 Detected new resource: receipts (s3) - Provisioning via storage-team
#
# 🔄 Provisioning receipts (s3)...
#    Workflow ID: 22
#    Provider: storage-team
#
# ✅ Step 1: create-minio-bucket (completed in 1.2s)
# ✅ Step 2: configure-bucket-policy (completed in 0.8s)
# ✅ Step 3: generate-access-credentials (completed in 0.5s)
#
# ✅ Resource receipts is now ACTIVE (total: 2.5s)
# ✅ Deployment completed successfully
```

#### Part 3: Monitoring (3 minutes)

```bash
# Check status via CLI
./innominatus-ctl list-resources --application order-service

# Output:
# 📦 Application: order-service (2 resources)
# ✅ db (postgres, active)
#    Host: order-service-db.order-service.svc.cluster.local
#    Port: 5432
#    Credentials: order-service-db-app.credentials
#
# ✅ receipts (s3, active)
#    Bucket: order-service-receipts
#    Endpoint: minio.minio.svc.cluster.local:9000

# Via AI Assistant
User: "Are my resources ready?"

AI: "Yes! All resources for order-service are active:

     ✅ db (PostgreSQL 15)
        - 2 replicas running
        - Medium size (1Gi RAM, 500m CPU)
        - Connection: order-service-db.order-service.svc.cluster.local:5432

     ✅ receipts (S3 bucket)
        - Versioning enabled
        - Endpoint: minio.minio.svc.cluster.local:9000
        - Access via credentials secret

     Total provisioning time: 3m 42s (database) + 2.5s (S3)

     Ready to deploy your application!"
```

#### Part 4: Application Deployment (3 minutes)

```yaml
# Alex creates Score specification
# File: order-service-score.yaml

apiVersion: score.dev/v1b1
metadata:
  name: order-service

containers:
  main:
    image: myorg/order-service:v1.0.0
    env:
      DB_HOST: ${resources.db.host}
      DB_PORT: ${resources.db.port}
      DB_NAME: ${resources.db.database}
      DB_USER: ${resources.db.username}
      DB_PASSWORD: ${resources.db.password}
      S3_ENDPOINT: ${resources.storage.endpoint}
      S3_BUCKET: ${resources.storage.bucket}

resources:
  db:
    type: postgres
    properties:
      version: "15"
      size: "medium"
      replicas: 2
  storage:
    type: s3
    properties:
      versioning: true
```

```bash
# Deploy via golden path
./innominatus-ctl run deploy-app order-service-score.yaml

# Output:
# 🔄 Executing golden path: deploy-app
# Execution ID: 23
#
# Resources detected:
#   - postgres (existing: orders-db)
#   - s3 (existing: order-receipts)
#
# Steps:
#   1. Validate resources (completed)
#   2. Inject credentials (running)
#   3. Deploy to Kubernetes (pending)
#   4. Health check (pending)

# Check deployment status
./innominatus-ctl workflow logs 23 --follow

# Output:
# ✅ Step 1: Validate resources
#    Found existing resources, skipping provisioning
#
# ✅ Step 2: Inject credentials
#    Created ConfigMap: order-service-config
#    Created Secret: order-service-secrets
#
# ✅ Step 3: Deploy to Kubernetes
#    Deployment created: order-service
#    Service created: order-service
#    Replicas: 3/3 ready
#
# ✅ Step 4: Health check
#    HTTP probe: http://order-service.order-service.svc.cluster.local:8080/health
#    Status: Healthy
#
# ✅ Workflow completed successfully
```

#### Part 5: Verification (2 minutes)

```bash
# Verify everything is running
kubectl get all -n order-service

# Output:
# NAME                               READY   STATUS    RESTARTS   AGE
# pod/order-service-7d8f9c8d-2xk4h   1/1     Running   0          2m
# pod/order-service-7d8f9c8d-8jq7n   1/1     Running   0          2m
# pod/order-service-7d8f9c8d-p5m9x   1/1     Running   0          2m
# pod/order-service-orders-db-0      1/1     Running   0          5m
# pod/order-service-orders-db-1      1/1     Running   0          5m
#
# NAME                    TYPE        CLUSTER-IP      PORT(S)    AGE
# service/order-service   ClusterIP   10.96.123.45    8080/TCP   2m

# Test the application
curl http://order-service.order-service.svc.cluster.local:8080/health

# Output:
# {
#   "status": "healthy",
#   "database": "connected",
#   "storage": "connected",
#   "version": "v1.0.0"
# }

# Via Web UI
# Navigate to: http://localhost:8081/resources
# Shows dependency graph:
#
# [Score Spec: order-service]
#        ↓
# ┌──────┴────┐
# │           │
# [db]    [receipts]
#  ↓          ↓
# [database-team] [storage-team]
```

**Demo Summary:**
- ⏱️ Total time: 17 minutes (including explanation)
- ✅ Zero manual Kubernetes YAML
- ✅ Production-ready database with HA
- ✅ Versioned S3 storage (added incrementally)
- ✅ Automatic credential injection
- ✅ Full dependency tracking
- ✅ Multi-provider coordination (database-team + storage-team)
- ✅ Incremental resource addition pattern demonstrated

---

## Troubleshooting Demo Scenarios

### Scenario 1: Failed Workflow Debugging

**Setup:** Trigger a workflow that will fail (invalid namespace name)

```bash
# Create Score spec with invalid app name (capitals not allowed in k8s)
cat > /tmp/invalid-app.yaml <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: My-App  # Invalid: capitals not allowed in Kubernetes

resources:
  my-db:
    type: postgres
    properties:
      version: "15"
EOF

# Deploy - workflow will fail on namespace creation
./innominatus-ctl deploy /tmp/invalid-app.yaml -w
```

**Demo Steps:**

1. **Observe failure in Web UI**
   - Navigate to http://localhost:8081/workflows
   - See failed workflow with red indicator

2. **View error details**
   - Click on failed workflow
   - See error banner:
     ```
     ❌ Step 1: create-namespace (kubernetes)

     Error Details:
     The Namespace "My-App" is invalid: metadata.name: Invalid value: "My-App":
     a lowercase RFC 1123 label must consist of lower case alphanumeric characters or '-'
     ```

3. **Retry with correct name**
   ```bash
   # Fix the Score spec with lowercase name
   cat > /tmp/valid-app.yaml <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: my-app  # ✅ Fixed: lowercase

resources:
  my-db:
    type: postgres
    properties:
      version: "15"
EOF

   # Deploy fixed spec
   ./innominatus-ctl deploy /tmp/valid-app.yaml -w
   # ✅ Success - workflow completes
   ```

**Key Points:**
- ✅ Clear error messages
- ✅ Validation feedback
- ✅ Easy retry

---

### Scenario 2: Resource Conflict Resolution

**Setup:** Two developers try to deploy Score specs with same resource

```bash
# Create shared Score spec
cat > /tmp/shared-app.yaml <<'EOF'
apiVersion: score.dev/v1b1
metadata:
  name: shared-app

resources:
  cache:
    type: redis
    properties:
      memory: "1GB"
EOF

# Developer 1 deploys first
./innominatus-ctl deploy /tmp/shared-app.yaml -w
# ✅ Success

# Developer 2 deploys same spec (at same time or right after)
./innominatus-ctl deploy /tmp/shared-app.yaml -w
```

**Expected Behavior:**
- Second deployment is idempotent: detects existing resource, skips provisioning
- Output: "ℹ️  Detected existing: cache (redis) - Skipping"

**Demo Points:**
- ✅ Prevents resource conflicts
- ✅ Clear ownership
- ✅ Audit trail

---

## Demo Tips & Tricks

### Terminal Setup

```bash
# Split terminal for parallel viewing:
# - Left: CLI commands
# - Right: Server logs (tail -f /tmp/innominatus.log)
# - Bottom: kubectl watch

# Use tmux/screen for professional demo
```

### Browser Setup

```bash
# Open multiple tabs:
# 1. Workflows page
# 2. Resources page
# 3. AI Assistant
# 4. Provider catalog

# Use browser profiles for different personas:
# - Developer profile (light theme)
# - Platform team profile (dark theme)
```

### Recovery from Demo Failures

```bash
# If demo breaks, quick reset:
./innominatus-ctl demo-nuke
./innominatus-ctl demo-time

# Or cleanup specific resources:
kubectl delete namespace <app-name>
psql -c "DELETE FROM resource_instances WHERE application = '<app-name>'"
```

### Audience Engagement

**Questions to Ask:**
- "How long does it take your team to provision a database today?"
- "Who handles database credentials in your organization?"
- "How do you track resource ownership across teams?"

**Response Templates:**
- Slow provisioning → Show real-time workflow progress
- Manual credentials → Show automatic injection
- No tracking → Show dependency graph

---

## Metrics to Highlight

### Speed
- ⏱️ Database provisioning: **5 minutes** (vs. days/weeks manually)
- ⏱️ Team onboarding: **10 minutes** (vs. hours manually)
- ⏱️ Deployment: **2 minutes** (vs. 30+ minutes manually)

### Reliability
- ✅ Workflow success rate: **95%+**
- ✅ Automatic retries on transient failures
- ✅ Production best practices by default

### Governance
- 📊 Complete audit trail (who, what, when)
- 🔐 Provider-based access control
- 📈 Resource lifecycle tracking

---

## Common Questions & Answers

**Q: "How is this different from Terraform?"**
A: innominatus orchestrates multiple tools (including Terraform) via workflows. It adds:
- Event-driven automation
- Multi-team provider model
- Natural language interface
- Real-time progress visibility

**Q: "Can we use our existing Terraform modules?"**
A: Yes! Wrap them in workflow steps:
```yaml
- name: provision-infra
  type: terraform
  config:
    working_dir: ./terraform/modules/vpc
```

**Q: "What if our platform team already has automation?"**
A: innominatus complements existing tools:
- Use providers to wrap existing scripts
- Keep domain expertise in teams
- Add orchestration layer for coordination

**Q: "How do we handle multi-cloud?"**
A: Provider model supports any backend:
- aws-team provider → AWS resources
- azure-team provider → Azure resources
- gcp-team provider → GCP resources
- Resolver routes based on labels/tags

---

## Next Steps After Demo

### For Developers
1. Try AI assistant for resource provisioning
2. Explore golden paths catalog
3. Deploy sample application

### For Platform Teams
1. Create first provider (wrap existing automation)
2. Define capability mappings
3. Migrate one workflow to innominatus

### For Executives
1. Measure time-to-provision metrics
2. Review audit logs and governance
3. Evaluate ROI (developer time saved)

---

## Demo Environment Checklist

**Before Demo:**
- [ ] Server running and healthy
- [ ] Database populated with sample data
- [ ] Demo accounts created (developer, platform-admin)
- [ ] Browser tabs pre-opened
- [ ] Terminal windows arranged
- [ ] Demo script printed/accessible

**During Demo:**
- [ ] Speak slowly and clearly
- [ ] Pause for questions
- [ ] Show multiple interfaces (CLI, UI, AI)
- [ ] Highlight error handling
- [ ] Keep demo under 20 minutes

**After Demo:**
- [ ] Share documentation links
- [ ] Offer hands-on workshop
- [ ] Collect feedback
- [ ] Follow up with recorded demo

---

**Demo Playbook Version:** 1.0
**Last Updated:** 2025-11-10
**Maintainer:** innominatus platform team
**Feedback:** Create issue at github.com/philipsahli/innominatus/issues
