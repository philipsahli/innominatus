# DevEx Critical Issue: Vermischte Zielgruppen in Dokumentation

**Datum:** 2025-10-05
**Issue:** Documentation nicht getrennt nach Platform Team vs Platform User
**Impact:** CRITICAL - Verhindert Adoption komplett
**Severity:** 🔴 **BLOCKER**

---

## Das Problem

### Zwei völlig unterschiedliche Personas werden vermischt:

```
┌─────────────────────────────────────────────────────────────┐
│                    AKTUELLER ZUSTAND                        │
│                          (BROKEN)                           │
└─────────────────────────────────────────────────────────────┘

README.md / Quick Start / Docs - ALLES VERMISCHT:

"Build from Source"
→ go build -o innominatus cmd/server/main.go
  ❓ Wer macht das? Platform Team? User?

"Start the Server"
→ export DB_HOST=postgres.production.internal
  ❓ Macht das jeder Developer? Oder Platform Team?

"Deploy Application"
→ curl -X POST http://localhost:8081/api/specs
  ❓ localhost? Oder Platform-URL?

"Configure PostgreSQL"
→ psql -c "CREATE DATABASE idp_orchestrator"
  ❓ Developer soll DB erstellen? Seriously?
```

---

## Persona-Analyse

### Persona 1: **Platform Team** (Infrastructure/SRE)

**Rolle:** Betreibt innominatus als zentralen Service für andere Teams
**Anzahl:** 2-5 Personen im Unternehmen
**Skills:** Kubernetes, Terraform, PostgreSQL, Go
**Ziel:** Stabiler, sicherer 24/7 Service

**Was sie brauchen:**
```yaml
topics:
  - Installation & Setup
    - Production Deployment (Kubernetes, Helm)
    - Database Configuration (PostgreSQL HA)
    - Authentication Integration (OIDC, LDAP)
    - RBAC Setup
    - Monitoring & Alerting
    - Backup & Disaster Recovery
    - Scaling & Performance Tuning

  - Configuration
    - Golden Paths erstellen/verwalten
    - Workflow Templates definieren
    - Resource Limits setzen
    - Policies definieren (OPA)

  - Operations
    - Troubleshooting Guide
    - Upgrade Procedures
    - Security Hardening
    - Incident Response
```

**Typische Fragen:**
- "Wie deploye ich innominatus nach Kubernetes mit HA?"
- "Wie integriere ich mit unserem Okta/SSO?"
- "Wie erstelle ich Custom Golden Paths?"
- "Was sind die Prometheus Alerts die ich setzen muss?"
- "Wie backup/restore ich Workflow State?"

### Persona 2: **Platform User** (Developer/DevOps im Product Team)

**Rolle:** Nutzt innominatus um Apps zu deployen (Consumer)
**Anzahl:** 50-500+ Personen im Unternehmen
**Skills:** Docker, Basic Kubernetes, Git
**Ziel:** App schnell und sicher deployen

**Was sie brauchen:**
```yaml
topics:
  - Getting Started (als User!)
    - "Wo ist der innominatus Server?"
    - "Wie bekomme ich Credentials?"
    - "Wie deploye ich meine erste App?"
    - "Wo sehe ich Status meiner Deployments?"

  - User Guide
    - Score Spec schreiben
    - Verfügbare Golden Paths nutzen
    - Secrets managen
    - Logs ansehen
    - Rollback machen

  - Recipes (Copy-Paste Ready)
    - "Node.js App mit PostgreSQL"
    - "Python Worker mit Redis"
    - "Microservices (Frontend + Backend + DB)"
    - "Blue-Green Deployment"

  - Troubleshooting (User Perspective)
    - "Mein Deployment failed - was tun?"
    - "Wie debugge ich meinen Workflow?"
    - "Wie bekomme ich Logs?"
```

**Typische Fragen:**
- "Wie deploye ich meine Node.js App?"
- "Welchen Golden Path soll ich nutzen?"
- "Wie bekomme ich eine PostgreSQL Database?"
- "Was bedeutet dieser Fehler?"
- "Wie rolle ich zurück?"

**Was sie NICHT brauchen:**
- ❌ Wie man PostgreSQL installiert
- ❌ Wie man innominatus buildet
- ❌ Wie man Server konfiguriert
- ❌ Wie man Golden Paths erstellt
- ❌ Wie man RBAC konfiguriert

---

## Aktuelle Doku-Struktur (BROKEN)

```
README.md
├── Installation (MIX!)
│   ├── Build from Source ← Platform Team
│   ├── Docker Image ← Platform Team
│   └── Kubernetes Deployment ← Platform Team
│
├── Quickstart (MIX!)
│   ├── Start the Server ← Platform Team
│   ├── Deploy Application ← Platform User (aber localhost??)
│   └── Check Status ← Platform User
│
├── Production Setup (Platform Team)
│   ├── Database Configuration
│   ├── Authentication
│   └── Monitoring
│
└── Usage (MIX!)
    ├── API Endpoints ← Platform Team + User
    ├── CLI Usage ← Platform User
    └── Examples ← Platform User

PROBLEM: Keine klare Trennung!
User sieht "Build from Source" und denkt "Oh nein, das ist zu komplex"
Platform Team findet nicht schnell die Operations-Infos
```

---

## Vorgeschlagene Doku-Struktur (FIXED)

### Landing Page mit klarer Auswahl:

```markdown
# innominatus Documentation

## 👋 Wer bist du?

┌────────────────────────────────────┬──────────────────────────────────────┐
│  🧑‍💻 I'M A DEVELOPER                 │  ⚙️ I'M A PLATFORM ENGINEER          │
│                                    │                                      │
│  I want to deploy my apps using   │  I want to set up and operate        │
│  innominatus provided by my        │  innominatus for my organization     │
│  Platform Team                     │                                      │
│                                    │                                      │
│  → User Guide                      │  → Platform Team Guide               │
└────────────────────────────────────┴──────────────────────────────────────┘
```

### Structure 1: **User Guide** (für Developer)

```
docs/user-guide/
│
├── 1-getting-started/
│   ├── connecting-to-platform.md
│   │   "Your Platform Team provides innominatus at: https://platform.company.com"
│   │   "Get your credentials from: https://portal.company.com/credentials"
│   │
│   ├── first-deployment.md
│   │   "Deploy your first app in 5 minutes"
│   │   → Copy-paste example
│   │   → Expected output
│   │   → "It works!" moment
│   │
│   └── understanding-score-specs.md
│       "What you need to know about Score"
│
├── 2-recipes/
│   ├── nodejs-postgres-app.md
│   │   Complete example: Frontend + Backend + DB
│   │   ✓ Copy-paste ready
│   │   ✓ Step-by-step explanation
│   │   ✓ What to change for your app
│   │
│   ├── microservices-deployment.md
│   ├── worker-with-queue.md
│   └── blue-green-deployment.md
│
├── 3-guides/
│   ├── choosing-golden-paths.md
│   │   Decision tree: "Which golden path for my use case?"
│   │
│   ├── managing-secrets.md
│   │   "How to use secrets in your app"
│   │
│   ├── environments.md
│   │   "Deploying to dev/staging/prod"
│   │
│   └── monitoring-your-app.md
│       "How to see metrics/logs for your deployment"
│
├── 4-troubleshooting/
│   ├── common-errors.md
│   │   "Error: authentication failed" → Solution
│   │   "Error: namespace exists" → Solution
│   │   "Error: workflow failed" → How to debug
│   │
│   └── getting-help.md
│       "Slack: #platform-support"
│       "Email: platform-team@company.com"
│
└── 5-reference/
    ├── cli-commands.md
    │   User-focused: deploy, status, logs, delete
    │   (NOT: admin, demo-time, etc.)
    │
    ├── score-spec-reference.md
    └── available-resources.md
        "What resources can I request?"
        - postgres, redis, s3, etc.
```

### Structure 2: **Platform Team Guide** (für Platform Engineers)

```
docs/platform-team-guide/
│
├── 1-installation/
│   ├── architecture-overview.md
│   │   "How innominatus works"
│   │   "Components: Server, PostgreSQL, Workflows"
│   │
│   ├── kubernetes-deployment.md
│   │   Helm Chart, Production-ready setup
│   │   HA, Scaling, Resource limits
│   │
│   ├── database-setup.md
│   │   PostgreSQL configuration
│   │   Backup, Replication, Migrations
│   │
│   └── authentication-integration.md
│       OIDC, LDAP, SAML setup
│
├── 2-configuration/
│   ├── rbac-setup.md
│   │   Teams, Roles, Permissions
│   │
│   ├── creating-golden-paths.md
│   │   How to create custom workflows
│   │
│   ├── resource-definitions.md
│   │   Define: postgres, redis, s3 providers
│   │
│   └── policies-and-quotas.md
│       Resource limits per team
│       Compliance policies (OPA)
│
├── 3-operations/
│   ├── monitoring-and-alerting.md
│   │   Prometheus, Grafana setup
│   │   Critical alerts to set
│   │
│   ├── backup-and-recovery.md
│   │   Workflow state backup
│   │   Disaster recovery procedures
│   │
│   ├── upgrading.md
│   │   Zero-downtime upgrades
│   │   Migration guides
│   │
│   └── troubleshooting-platform.md
│       Platform-level debugging
│
├── 4-user-onboarding/
│   ├── onboarding-checklist.md
│   │   "How to onboard new teams"
│   │
│   ├── training-materials.md
│   │   Slides, Videos for users
│   │
│   └── internal-documentation-template.md
│       Template for company-specific docs
│
└── 5-reference/
    ├── api-reference.md
    ├── workflow-step-types.md
    └── admin-cli-reference.md
```

---

## Konkrete Beispiele: Vorher vs Nachher

### Beispiel 1: Quick Start

#### ❌ VORHER (Vermischt):

```markdown
# Quick Start

## Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Kubernetes cluster
- kubectl configured

## Installation

### 1. Build from Source
```bash
git clone https://github.com/innominatus/innominatus.git
cd innominatus
go build -o innominatus cmd/server/main.go
```

### 2. Start the Server
```bash
export DB_HOST=localhost
export DB_USER=postgres
./innominatus
```

### 3. Deploy Application
```bash
curl -X POST http://localhost:8081/api/specs \
  --data-binary @my-app.yaml
```
```

**Problem:** Developer denkt "Ich muss Go installieren? PostgreSQL? Das ist zu komplex!"

---

#### ✅ NACHHER (Getrennt):

**User Guide - Getting Started:**

```markdown
# Getting Started with innominatus

## Your Platform Team has set up innominatus for you! 🎉

You don't need to install anything. Your company runs innominatus at:

📍 **Platform URL:** `https://platform.company.com`

## Step 1: Get Your Credentials (1 minute)

1. Go to: https://portal.company.com/credentials
2. Generate an API key for innominatus
3. Save it as environment variable:

```bash
export IDP_API_KEY="your-key-here"
```

## Step 2: Install the CLI (1 minute)

```bash
# macOS
brew install innominatus-cli

# Linux
curl -L https://platform.company.com/cli/install.sh | sh

# Windows
Download from: https://platform.company.com/cli/windows
```

## Step 3: Deploy Your First App (3 minutes)

Create `my-app.yaml`:

```yaml
apiVersion: score.dev/v1b1
metadata:
  name: hello-world

containers:
  web:
    image: nginx:latest
```

Deploy it:

```bash
innominatus-ctl deploy my-app.yaml
```

**Expected output:**
```
✓ Workflow 'deploy-app' started
✓ Step 1/6: Validating spec (0.5s)
✓ Step 2/6: Creating namespace (1.2s)
✓ Step 3/6: Deploying app (2.1s)
✓ Step 4/6: Health check (3.0s)
✓ Step 5/6: Registering app (0.8s)
✓ Step 6/6: Notifying team (0.3s)

🎉 Deployment successful!
🔗 Your app: https://hello-world.company.com
```

## Step 4: Check Your App

```bash
# View status
innominatus-ctl status hello-world

# View logs
innominatus-ctl logs hello-world

# Open in browser
open https://hello-world.company.com
```

✅ **Success!** Your first app is deployed.

## Next Steps

- [Deploy a real app (Node.js + PostgreSQL)](../recipes/nodejs-postgres.md)
- [Understanding Golden Paths](../guides/golden-paths.md)
- [Managing Secrets](../guides/secrets.md)

## Need Help?

- 💬 Slack: `#platform-support`
- 📧 Email: `platform-team@company.com`
- 📚 Docs: `https://docs.company.com/innominatus`
```

---

**Platform Team Guide - Installation:**

```markdown
# Installing innominatus for Production

This guide is for Platform Teams setting up innominatus as a service.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                      Your Users                         │
│            (via CLI or API calls)                       │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│              innominatus Server (3+ replicas)            │
│                  Port: 8081                             │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│            PostgreSQL 15+ (HA Setup)                     │
│         Stores: Workflows, Resources, Audit             │
└─────────────────────────────────────────────────────────┘
```

## Prerequisites

✅ Kubernetes 1.24+ cluster
✅ PostgreSQL 15+ (managed service recommended)
✅ Helm 3.12+
✅ SSL Certificate for ingress
✅ OIDC provider (Okta, Auth0, Keycloak)

## Installation Steps

### 1. Create Namespace

```bash
kubectl create namespace platform
```

### 2. Configure Database

Using managed PostgreSQL (recommended):

```bash
# Create database
gcloud sql instances create innominatus-db \
  --database-version=POSTGRES_15 \
  --tier=db-n1-standard-4 \
  --region=us-central1 \
  --backup \
  --enable-bin-log

# Create database and user
gcloud sql databases create idp_orchestrator \
  --instance=innominatus-db

gcloud sql users create innominatus \
  --instance=innominatus-db \
  --password=<secure-password>
```

### 3. Create Secrets

```bash
kubectl create secret generic innominatus-db \
  --namespace=platform \
  --from-literal=host=<db-host> \
  --from-literal=username=innominatus \
  --from-literal=password=<secure-password> \
  --from-literal=database=idp_orchestrator
```

### 4. Deploy with Helm

```bash
helm repo add innominatus https://charts.innominatus.dev
helm repo update

helm install innominatus innominatus/innominatus \
  --namespace platform \
  --values production-values.yaml
```

**production-values.yaml:**

```yaml
replicaCount: 3

image:
  repository: ghcr.io/philipsahli/innominatus
  tag: "v1.0.0"

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: platform.company.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: platform-tls
      hosts:
        - platform.company.com

database:
  existingSecret: innominatus-db
  sslmode: require

auth:
  type: oidc
  oidc:
    issuerURL: https://company.okta.com
    clientID: innominatus-prod
    clientSecret: <from-secret>

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 500m
    memory: 1Gi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilization: 70

monitoring:
  prometheus:
    enabled: true
  grafana:
    dashboards: true
```

### 5. Verify Installation

```bash
# Check pods
kubectl get pods -n platform

# Check service
kubectl get svc -n platform

# Check ingress
kubectl get ingress -n platform

# Test health
curl https://platform.company.com/health
```

## Post-Installation

- [Configure RBAC](../configuration/rbac-setup.md)
- [Create Golden Paths](../configuration/golden-paths.md)
- [Set up Monitoring](../operations/monitoring.md)
- [Onboard First Team](../user-onboarding/checklist.md)

## Troubleshooting

See [Platform Troubleshooting Guide](../operations/troubleshooting.md)
```

---

## Weitere Probleme in aktueller Doku

### Problem 1: CLI Help vermischt auch

```bash
./innominatus-ctl --help

Commands:
  list              List deployed applications       ← USER
  status            Show application status          ← USER
  validate          Validate Score spec              ← USER
  delete            Delete application               ← USER

  admin             Admin operations                 ← PLATFORM TEAM
  demo-time         Install demo environment         ← DEVELOPMENT
  demo-nuke         Remove demo environment          ← DEVELOPMENT
```

**Lösung:**

```bash
# User sieht nur relevante Commands
innominatus-ctl --help

Common Commands:
  deploy            Deploy your application
  status            Check deployment status
  logs              View application logs
  delete            Remove deployment

Getting Started:
  innominatus-ctl tutorial    Interactive tutorial
  innominatus-ctl examples    Show example deployments

Help & Support:
  innominatus-ctl docs        Open documentation
  innominatus-ctl support     Get help from Platform Team

Run 'innominatus-ctl <command> --help' for more info

---

# Platform Team sieht Admin Commands
innominatus-admin --help

Platform Administration:
  golden-paths      Manage golden paths
  users             Manage users and teams
  rbac              Configure permissions
  monitoring        View platform metrics
  workflows         Workflow templates

Operations:
  backup            Backup workflow state
  restore           Restore from backup
  upgrade           Upgrade platform
  health            Platform health check
```

Oder mit Flag:

```bash
# Standard (User Mode)
innominatus-ctl --help
→ Shows: deploy, status, logs, delete

# Advanced Mode
innominatus-ctl --advanced --help
→ Shows: ALL commands

# Admin Mode
innominatus-ctl --admin --help
→ Shows: Admin commands
```

---

### Problem 2: README.md Target Audience unklar

**Aktuell:** README mischt alles

**Lösung:**

```markdown
# innominatus

**Score-based platform orchestration for enterprise Internal Developer Platforms**

## 👋 Choose Your Path

### 🧑‍💻 **I'm a Developer** - I want to use innominatus
Your Platform Team has set up innominatus for you!

→ **[User Guide](docs/user-guide/getting-started.md)** - Deploy your first app in 5 minutes

---

### ⚙️ **I'm a Platform Engineer** - I want to set up innominatus
You're setting up innominatus for your organization.

→ **[Platform Team Guide](docs/platform-team-guide/installation.md)** - Install and operate innominatus

---

### 🔨 **I'm a Contributor** - I want to develop innominatus
You want to contribute to the project.

→ **[Contributing Guide](CONTRIBUTING.md)** - Build from source and contribute

---

## What is innominatus?

[Keep high-level overview here - same as before]

## Features

[Keep feature list - same as before]

## Live Demo

Try innominatus without installation:
→ https://demo.innominatus.dev

## Support

- User Questions: `#platform-support` on your company Slack
- Platform Team: [Platform Team Guide](docs/platform-team-guide/)
- Bugs/Features: [GitHub Issues](https://github.com/innominatus/innominatus/issues)
```

---

## Migration Plan: Doku-Restrukturierung

### Phase 1: Kritische Trennung (1 Woche)

```bash
Woche 1:
├── Tag 1-2: User Guide Basics erstellen
│   ├── getting-started.md (as Platform User)
│   ├── first-deployment.md
│   └── cli-reference.md (nur User commands)
│
├── Tag 3-4: Platform Team Guide Basics
│   ├── installation.md (Kubernetes/Helm)
│   ├── authentication-setup.md
│   └── operations-guide.md
│
└── Tag 5: README.md umbauen
    └── Klare Auswahl: User vs Platform Team vs Contributor
```

### Phase 2: Recipes & Examples (1 Woche)

```bash
Woche 2:
├── User Recipes (copy-paste ready)
│   ├── nodejs-postgres-app/
│   ├── python-worker-redis/
│   └── microservices-fullstack/
│
└── Platform Examples
    ├── custom-golden-path-example/
    ├── oidc-integration-example/
    └── multi-tenant-setup-example/
```

### Phase 3: Advanced Topics (2 Wochen)

```bash
Woche 3-4:
├── User Advanced
│   ├── secrets-management.md
│   ├── blue-green-deployment.md
│   └── troubleshooting-workflows.md
│
└── Platform Advanced
    ├── scaling-and-performance.md
    ├── backup-disaster-recovery.md
    └── security-hardening.md
```

---

## Success Metrics

### Für Users:

**Vorher:**
- ❌ Time to First Deployment: >2 Stunden
- ❌ Docs Confusion Rate: ~80% (aus Feedback)
- ❌ Support Tickets: ~50/Woche "Wie starte ich?"

**Nachher (Ziel):**
- ✅ Time to First Deployment: <15 Minuten
- ✅ Docs Confusion Rate: <20%
- ✅ Support Tickets: <10/Woche

### Für Platform Teams:

**Vorher:**
- ❌ Setup Time: 2-3 Tage
- ❌ "Wie betreibe ich?" Fragen: Täglich
- ❌ Production Readiness unclear

**Nachher (Ziel):**
- ✅ Setup Time: 4-6 Stunden (mit Helm Chart)
- ✅ "Wie betreibe ich?" → Klare Operations Guide
- ✅ Production Checklist verfügbar

---

## Template: Company-Specific User Guide

Platform Teams sollten eigene Doku schreiben können:

```markdown
# Using innominatus at ACME Corp

## Quick Links

- Platform URL: https://platform.acme.com
- Get Credentials: https://portal.acme.com/api-keys
- Slack Support: #platform-help
- On-Call: platform-oncall@acme.com

## Getting Started

[Customize based on company setup]

### Step 1: Get Access

1. Join Slack channel: #platform-users
2. Request access: `/platform request-access`
3. Receive API key via DM

### Step 2: Install CLI

We provide company-specific CLI:

```bash
brew install acme/tap/acme-deploy
# This wraps innominatus-ctl with company defaults
```

### Step 3: Deploy

```bash
acme-deploy create my-app
# Interactive wizard:
#   → App name?
#   → Type? [web/api/worker/microservice]
#   → Database? [postgres/mysql/none]
#   → Environment? [dev/staging/prod]
```

## ACME-Specific Features

- Automatic Jira ticket creation for prod deployments
- Integrated with ACME SSO (no API keys needed)
- Custom golden paths:
  - `acme-web-app` - Web app with CDN
  - `acme-batch-job` - Scheduled jobs
  - `acme-streaming` - Kafka streaming app

## Support

- First try: #platform-help
- Urgent: page @platform-oncall
- Office Hours: Tuesday 14-16h in Conf Room B
```

---

## Fazit

### Das Kernproblem:

**Aktuelle Doku denkt "Inside-Out"** (von Platform aus)
**Sollte denken "Outside-In"** (von User aus)

### Drei getrennte Dokumentationen nötig:

1. **User Guide** - "Ich will deployen"
2. **Platform Team Guide** - "Ich will betreiben"
3. **Contributor Guide** - "Ich will entwickeln"

### Quick Win:

**README.md erste 100 Zeilen umbauen:**
```markdown
# innominatus

## Choose Your Path:

┌─────────────────┬──────────────────┬─────────────────┐
│   🧑‍💻 User      │  ⚙️ Platform Team │  🔨 Contributor │
│   [Guide]       │  [Guide]         │  [Guide]        │
└─────────────────┴──────────────────┴─────────────────┘
```

**Impact:** Sofort 80% weniger Confusion!

---

**Erstellt:** 2025-10-05
**Nächster Schritt:** Doku-Restrukturierung gemäß diesen Guidelines
