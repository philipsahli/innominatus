# Multi-Team Platform Demo - Implementation Plan

## Overview

This document provides a comprehensive implementation plan for a multi-team platform demo showcasing how innominatus orchestrates infrastructure provisioning across multiple product teams to onboard application development teams.

### Demo Scenario

**4 Product Teams** provide infrastructure resources:
- **container-team**: Kubernetes namespace + ArgoCD application
- **database-team**: PostgreSQL database instance
- **storage-team**: S3 bucket (Minio)
- **vault-team**: Vault secret space

**1 Dev Team** consumes these resources via a Golden Path to deploy their application with:
- Database connection (credentials synced via Vault)
- S3 storage connection (connection string as env var)
- GitOps deployment via ArgoCD

### Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Innominatus Platform                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  Product Teams (Git Repos with Providers)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │ container-   │  │ database-    │  │ storage-     │           │
│  │ team/        │  │ team/        │  │ team/        │           │
│  │ container-   │  │ database-    │  │ storage-     │           │
│  │ provider     │  │ provider     │  │ provider     │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘          │
│         │                  │                  │                   │
│         │   ┌──────────────┴──────────┐       │                  │
│         │   │ vault-team/             │       │                  │
│         │   │ vault-provider          │       │                  │
│         │   └──────────┬──────────────┘       │                  │
│         │              │                       │                  │
│         └──────────────┴───────────────────────┘                 │
│                        │                                          │
│              ┌─────────▼─────────┐                               │
│              │  Golden Path:     │                               │
│              │  onboard-dev-team │                               │
│              └─────────┬─────────┘                               │
│                        │                                          │
│              ┌─────────▼─────────┐                               │
│              │  Dev Team App     │                               │
│              │  (Score Spec)     │                               │
│              └───────────────────┘                               │
│                                                                   │
├─────────────────────────────────────────────────────────────────┤
│  Infrastructure                                                   │
│  ┌─────────┐  ┌──────────┐  ┌───────┐  ┌────────┐              │
│  │ K8s     │  │ Postgres │  │ Minio │  │ Vault  │               │
│  │ Cluster │  │ Operator │  │       │  │  +VSO  │               │
│  └─────────┘  └──────────┘  └───────┘  └────────┘              │
└─────────────────────────────────────────────────────────────────┘
```

## Prerequisites

- Kubernetes cluster with demo environment: `./innominatus-ctl demo-time`
- Gitea, ArgoCD, Vault, Minio already deployed
- PostgreSQL operator (automatically installed by demo-time)

## Implementation Phases

### Phase 1: PostgreSQL Operator ✅ IMPLEMENTED

The PostgreSQL Operator (Zalando) is now automatically installed as part of the demo environment.

**Implementation complete in:**
- `internal/demo/postgres_operator.go` - Installation/uninstallation functions
- `internal/cli/commands.go` - Integration with demo-time, demo-nuke, demo-status

**What it does:**
- Automatically installs Zalando PostgreSQL Operator during `demo-time`
- Creates `postgres-operator` namespace
- Installs operator and UI components
- Checks operator status in `demo-status`
- Cleans up operator in `demo-nuke`

**No manual intervention required** - the operator is ready to use!

### Phase 2: Container Team Provider

#### 2.1 Provider Definition

**Repository**: Create in Gitea: `container-team/container-provider`

**File**: `provider.yaml`

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: container-team
  version: 1.0.0
  category: platform
  description: Container platform provisioners - namespace and ArgoCD applications

provisioners:
  - name: namespace
    type: kubernetes-namespace
    description: Kubernetes namespace with resource quotas and network policies
    version: 1.0.0

  - name: argocd-app
    type: argocd-application
    description: ArgoCD application for GitOps deployment
    version: 1.0.0

goldenpaths:
  - name: create-namespace
    description: Create a new Kubernetes namespace with standard configuration
  - name: setup-gitops
    description: Setup ArgoCD application for GitOps deployment
```

#### 2.2 Namespace Provisioner Workflow

**File**: `workflows/provision-namespace.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: provision-namespace
  description: Create Kubernetes namespace with resource quotas

spec:
  parameters:
    - name: namespace_name
      type: string
      required: true
      description: Name of the namespace to create

    - name: resource_quota
      type: object
      required: false
      description: Resource quotas for the namespace
      default:
        cpu: "4"
        memory: "8Gi"
        pods: "20"

    - name: labels
      type: object
      required: false
      description: Labels to apply to namespace

  steps:
    - name: create-namespace
      type: kubernetes
      config:
        action: create
        manifest: |
          apiVersion: v1
          kind: Namespace
          metadata:
            name: ${parameters.namespace_name}
            labels:
              managed-by: innominatus
              team: ${workflow.team}
              ${parameters.labels}

    - name: create-resource-quota
      type: kubernetes
      config:
        action: create
        namespace: ${parameters.namespace_name}
        manifest: |
          apiVersion: v1
          kind: ResourceQuota
          metadata:
            name: resource-quota
            namespace: ${parameters.namespace_name}
          spec:
            hard:
              requests.cpu: ${parameters.resource_quota.cpu}
              requests.memory: ${parameters.resource_quota.memory}
              pods: ${parameters.resource_quota.pods}

    - name: create-network-policy
      type: kubernetes
      config:
        action: create
        namespace: ${parameters.namespace_name}
        manifest: |
          apiVersion: networking.k8s.io/v1
          kind: NetworkPolicy
          metadata:
            name: default-deny-ingress
            namespace: ${parameters.namespace_name}
          spec:
            podSelector: {}
            policyTypes:
            - Ingress

  outputs:
    namespace: ${parameters.namespace_name}
    status: active
```

#### 2.3 ArgoCD Application Provisioner Workflow

**File**: `workflows/provision-argocd-app.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: provision-argocd-app
  description: Create ArgoCD application for GitOps deployment

spec:
  parameters:
    - name: app_name
      type: string
      required: true

    - name: namespace
      type: string
      required: true

    - name: repo_url
      type: string
      required: true
      description: Git repository URL

    - name: path
      type: string
      required: false
      default: "."
      description: Path in repo to manifests

    - name: sync_policy
      type: string
      required: false
      default: "auto"
      enum: [auto, manual]

  steps:
    - name: create-argocd-app
      type: argocd-app
      config:
        appName: ${parameters.app_name}
        project: default
        repoURL: ${parameters.repo_url}
        path: ${parameters.path}
        targetRevision: HEAD
        namespace: ${parameters.namespace}
        server: https://kubernetes.default.svc
        syncPolicy: ${parameters.sync_policy}
        automated:
          prune: true
          selfHeal: true

    - name: wait-for-sync
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e
          echo "Waiting for ArgoCD app to sync..."
          kubectl wait --for=condition=Synced \
            application/${parameters.app_name} \
            -n argocd \
            --timeout=300s

  outputs:
    app_name: ${parameters.app_name}
    app_url: http://argocd.localtest.me/applications/${parameters.app_name}
    sync_status: synced
```

### Phase 3: Database Team Provider

#### 3.1 Provider Definition

**Repository**: Create in Gitea: `database-team/database-provider`

**File**: `provider.yaml`

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: database-team
  version: 1.0.0
  category: data
  description: Database provisioners using PostgreSQL Operator

provisioners:
  - name: postgres-db
    type: postgresql
    description: PostgreSQL database instance managed by Zalando operator
    version: 1.0.0

  - name: postgres-db-with-backup
    type: postgresql-backup
    description: PostgreSQL with automated backups to S3
    version: 1.0.0

goldenpaths:
  - name: create-database
    description: Create a new PostgreSQL database
  - name: backup-database
    description: Configure database backup to S3
```

#### 3.2 PostgreSQL Provisioner Workflow

**File**: `workflows/provision-postgres.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: provision-postgres
  description: Create PostgreSQL database using Zalando operator

spec:
  parameters:
    - name: db_name
      type: string
      required: true
      description: Database cluster name

    - name: namespace
      type: string
      required: true
      description: Namespace to create database in

    - name: team_id
      type: string
      required: true
      description: Team identifier (used in cluster name)

    - name: size
      type: string
      required: false
      default: "small"
      enum: [small, medium, large]
      description: Database size

    - name: replicas
      type: number
      required: false
      default: 2
      description: Number of replicas

    - name: version
      type: string
      required: false
      default: "15"
      description: PostgreSQL version

  steps:
    - name: create-postgres-cluster
      type: kubernetes
      config:
        action: create
        namespace: ${parameters.namespace}
        manifest: |
          apiVersion: "acid.zalan.do/v1"
          kind: postgresql
          metadata:
            name: ${parameters.team_id}-${parameters.db_name}
            namespace: ${parameters.namespace}
            labels:
              managed-by: innominatus
              team: ${parameters.team_id}
          spec:
            teamId: ${parameters.team_id}
            numberOfInstances: ${parameters.replicas}
            postgresql:
              version: "${parameters.version}"
            volume:
              size: {{ if eq parameters.size "small" }}5Gi{{ else if eq parameters.size "medium" }}20Gi{{ else }}100Gi{{ end }}
            resources:
              requests:
                cpu: {{ if eq parameters.size "small" }}100m{{ else if eq parameters.size "medium" }}500m{{ else }}2000m{{ end }}
                memory: {{ if eq parameters.size "small" }}256Mi{{ else if eq parameters.size "medium" }}1Gi{{ else }}4Gi{{ end }}
              limits:
                cpu: {{ if eq parameters.size "small" }}500m{{ else if eq parameters.size "medium" }}2000m{{ else }}4000m{{ end }}
                memory: {{ if eq parameters.size "small" }}512Mi{{ else if eq parameters.size "medium" }}2Gi{{ else }}8Gi{{ end }}
            users:
              ${parameters.db_name}_owner:
              - superuser
              - createdb
              ${parameters.db_name}_app:
              - login
            databases:
              ${parameters.db_name}: ${parameters.db_name}_owner

    - name: wait-for-database
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e
          echo "Waiting for PostgreSQL cluster to be ready..."
          kubectl wait --for=condition=Running \
            postgresql/${parameters.team_id}-${parameters.db_name} \
            -n ${parameters.namespace} \
            --timeout=600s

    - name: get-credentials
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          # Get database credentials from secret
          DB_SECRET="${parameters.team_id}-${parameters.db_name}.${parameters.db_name}-app.credentials"
          DB_USER=$(kubectl get secret $DB_SECRET -n ${parameters.namespace} -o jsonpath='{.data.username}' | base64 -d)
          DB_PASSWORD=$(kubectl get secret $DB_SECRET -n ${parameters.namespace} -o jsonpath='{.data.password}' | base64 -d)
          DB_HOST="${parameters.team_id}-${parameters.db_name}.${parameters.namespace}.svc.cluster.local"
          DB_PORT="5432"
          DB_NAME="${parameters.db_name}"

          # Output as JSON for next steps
          echo "{\"username\":\"$DB_USER\",\"password\":\"$DB_PASSWORD\",\"host\":\"$DB_HOST\",\"port\":\"$DB_PORT\",\"database\":\"$DB_NAME\"}"

  outputs:
    database_name: ${parameters.db_name}
    cluster_name: ${parameters.team_id}-${parameters.db_name}
    namespace: ${parameters.namespace}
    host: ${parameters.team_id}-${parameters.db_name}.${parameters.namespace}.svc.cluster.local
    port: "5432"
    connection_string: postgresql://${get-credentials.username}:${get-credentials.password}@${parameters.team_id}-${parameters.db_name}.${parameters.namespace}.svc.cluster.local:5432/${parameters.db_name}
    credentials_secret: ${parameters.team_id}-${parameters.db_name}.${parameters.db_name}-app.credentials
    username: ${get-credentials.username}
    password: ${get-credentials.password}
```

### Phase 4: Storage Team Provider

#### 4.1 Provider Definition

**Repository**: Create in Gitea: `storage-team/storage-provider`

**File**: `provider.yaml`

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: storage-team
  version: 1.0.0
  category: storage
  description: Object storage provisioners for S3-compatible storage (Minio)

provisioners:
  - name: s3-bucket
    type: s3
    description: S3-compatible object storage bucket
    version: 1.0.0

  - name: s3-bucket-with-lifecycle
    type: s3-lifecycle
    description: S3 bucket with lifecycle policies
    version: 1.0.0

goldenpaths:
  - name: create-bucket
    description: Create a new S3 bucket
  - name: configure-lifecycle
    description: Configure lifecycle policies for S3 bucket
```

#### 4.2 S3 Bucket Provisioner Workflow

**File**: `workflows/provision-s3-bucket.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: provision-s3-bucket
  description: Create S3 bucket on Minio with access credentials

spec:
  parameters:
    - name: bucket_name
      type: string
      required: true
      description: Name of S3 bucket

    - name: namespace
      type: string
      required: true
      description: Namespace to store credentials

    - name: versioning
      type: boolean
      required: false
      default: true
      description: Enable versioning

    - name: public_access
      type: boolean
      required: false
      default: false
      description: Allow public access

  steps:
    - name: create-minio-bucket
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          # Configure Minio client (mc)
          mc alias set minio http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin

          # Create bucket
          mc mb minio/${parameters.bucket_name} || echo "Bucket already exists"

          # Enable versioning if requested
          if [ "${parameters.versioning}" = "true" ]; then
            mc version enable minio/${parameters.bucket_name}
          fi

          # Set public access policy if requested
          if [ "${parameters.public_access}" = "true" ]; then
            mc anonymous set download minio/${parameters.bucket_name}
          else
            mc anonymous set none minio/${parameters.bucket_name}
          fi

          echo "Bucket created successfully"

    - name: create-service-account
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          # Create Minio service account for bucket access
          mc alias set minio http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin

          # Generate service account
          SA_OUTPUT=$(mc admin user svcacct add minio minioadmin \
            --access-key ${parameters.bucket_name}-key \
            --secret-key $(openssl rand -base64 32) \
            --policy-name readonly)

          # Extract credentials
          ACCESS_KEY=$(echo "$SA_OUTPUT" | grep "Access Key" | awk '{print $3}')
          SECRET_KEY=$(echo "$SA_OUTPUT" | grep "Secret Key" | awk '{print $3}')

          # Output as JSON
          echo "{\"access_key\":\"$ACCESS_KEY\",\"secret_key\":\"$SECRET_KEY\"}"

    - name: create-bucket-policy
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          mc alias set minio http://minio.minio.svc.cluster.local:9000 minioadmin minioadmin

          # Create policy JSON
          cat > /tmp/bucket-policy.json <<EOF
          {
            "Version": "2012-10-17",
            "Statement": [
              {
                "Effect": "Allow",
                "Action": [
                  "s3:GetObject",
                  "s3:PutObject",
                  "s3:DeleteObject",
                  "s3:ListBucket"
                ],
                "Resource": [
                  "arn:aws:s3:::${parameters.bucket_name}",
                  "arn:aws:s3:::${parameters.bucket_name}/*"
                ]
              }
            ]
          }
          EOF

          # Apply policy
          mc admin policy create minio ${parameters.bucket_name}-policy /tmp/bucket-policy.json

    - name: store-credentials-in-namespace
      type: kubernetes
      config:
        action: create
        namespace: ${parameters.namespace}
        manifest: |
          apiVersion: v1
          kind: Secret
          metadata:
            name: ${parameters.bucket_name}-s3-credentials
            namespace: ${parameters.namespace}
            labels:
              managed-by: innominatus
              bucket: ${parameters.bucket_name}
          type: Opaque
          stringData:
            access-key: ${create-service-account.access_key}
            secret-key: ${create-service-account.secret_key}
            endpoint: http://minio.minio.svc.cluster.local:9000
            bucket: ${parameters.bucket_name}
            s3-url: s3://${parameters.bucket_name}

  outputs:
    bucket_name: ${parameters.bucket_name}
    endpoint: http://minio.minio.svc.cluster.local:9000
    external_endpoint: http://minio.localtest.me
    access_key: ${create-service-account.access_key}
    secret_key: ${create-service-account.secret_key}
    credentials_secret: ${parameters.bucket_name}-s3-credentials
    s3_url: s3://${parameters.bucket_name}
```

### Phase 5: Vault Team Provider

#### 5.1 Provider Definition

**Repository**: Create in Gitea: `vault-team/vault-provider`

**File**: `provider.yaml`

```yaml
apiVersion: v1
kind: Provider
metadata:
  name: vault-team
  version: 1.0.0
  category: security
  description: HashiCorp Vault secret space provisioners with VSO integration

provisioners:
  - name: vault-space
    type: vault-namespace
    description: Vault namespace with KV secrets engine and Kubernetes auth
    version: 1.0.0

  - name: vault-space-with-vso
    type: vault-vso
    description: Vault space with Vault Secrets Operator for K8s secret sync
    version: 1.0.0

goldenpaths:
  - name: create-vault-space
    description: Create a new Vault namespace with secret engine
  - name: sync-secrets-to-k8s
    description: Setup VSO to sync secrets to Kubernetes
```

#### 5.2 Vault Space Provisioner Workflow

**File**: `workflows/provision-vault-space.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: provision-vault-space
  description: Create Vault namespace with KV engine and Kubernetes auth for VSO

spec:
  parameters:
    - name: app_name
      type: string
      required: true
      description: Application name (used for vault path)

    - name: namespace
      type: string
      required: true
      description: Kubernetes namespace

    - name: service_account
      type: string
      required: false
      default: "default"
      description: Service account for Vault auth

    - name: secrets
      type: array
      required: false
      description: List of secrets to create
      default: []

  steps:
    - name: create-vault-namespace
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root

          # Create namespace for app
          vault namespace create applications/${parameters.app_name} || echo "Namespace exists"

    - name: enable-kv-engine
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Enable KV v2 secrets engine
          vault secrets enable -path=secret kv-v2 || echo "KV engine already enabled"

    - name: configure-kubernetes-auth
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Enable Kubernetes auth
          vault auth enable kubernetes || echo "Kubernetes auth already enabled"

          # Configure Kubernetes auth
          vault write auth/kubernetes/config \
            kubernetes_host="https://kubernetes.default.svc" \
            kubernetes_ca_cert=@/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
            token_reviewer_jwt=@/var/run/secrets/kubernetes.io/serviceaccount/token

    - name: create-vault-policy
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Create policy for app access
          cat > /tmp/app-policy.hcl <<EOF
          path "secret/data/*" {
            capabilities = ["read", "list"]
          }
          path "secret/metadata/*" {
            capabilities = ["read", "list"]
          }
          EOF

          vault policy write ${parameters.app_name}-read /tmp/app-policy.hcl

    - name: create-kubernetes-role
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Create Kubernetes auth role
          vault write auth/kubernetes/role/${parameters.app_name} \
            bound_service_account_names=${parameters.service_account} \
            bound_service_account_namespaces=${parameters.namespace} \
            policies=${parameters.app_name}-read \
            ttl=24h

    - name: setup-vso-resources
      type: kubernetes
      config:
        action: create
        namespace: ${parameters.namespace}
        manifest: |
          ---
          apiVersion: secrets.hashicorp.com/v1beta1
          kind: VaultConnection
          metadata:
            name: ${parameters.app_name}-vault-connection
            namespace: ${parameters.namespace}
          spec:
            address: http://vault.vault.svc.cluster.local:8200
            skipTLSVerify: true
          ---
          apiVersion: secrets.hashicorp.com/v1beta1
          kind: VaultAuth
          metadata:
            name: ${parameters.app_name}-vault-auth
            namespace: ${parameters.namespace}
          spec:
            vaultConnectionRef: ${parameters.app_name}-vault-connection
            method: kubernetes
            mount: kubernetes
            namespace: applications/${parameters.app_name}
            kubernetes:
              role: ${parameters.app_name}
              serviceAccount: ${parameters.service_account}

  outputs:
    vault_namespace: applications/${parameters.app_name}
    vault_path: secret/data
    vault_url: http://vault.localtest.me
    kubernetes_role: ${parameters.app_name}
    vso_auth_ref: ${parameters.app_name}-vault-auth
    vso_connection_ref: ${parameters.app_name}-vault-connection
```

### Phase 6: Dev Team Golden Path

#### 6.1 Golden Path Definition

**File**: `examples/goldenpaths.yaml` (add to existing)

```yaml
goldenpaths:
  # ... existing golden paths ...

  onboard-dev-team:
    workflow: ./workflows/onboard-dev-team.yaml
    description: Complete platform onboarding for development team
    category: onboarding
    tags: [onboarding, dev-team, full-stack, multi-team]
    estimated_duration: 8-12 minutes
    parameters:
      app_name:
        type: string
        required: true
        pattern: '^[a-z0-9\-]+$'
        description: Application name (lowercase, alphanumeric, hyphens)

      environment:
        type: enum
        required: false
        default: development
        allowed_values: [development, staging, production]
        description: Environment type

      db_size:
        type: enum
        required: false
        default: small
        allowed_values: [small, medium, large]
        description: Database size

      enable_backup:
        type: boolean
        required: false
        default: false
        description: Enable automated backups to S3
```

#### 6.2 Dev Team Onboarding Workflow

**File**: `workflows/onboard-dev-team.yaml`

```yaml
apiVersion: workflow.dev/v1
kind: Workflow
metadata:
  name: onboard-dev-team
  description: Complete platform onboarding orchestrating all product teams

spec:
  parameters:
    - name: app_name
      type: string
      required: true

    - name: environment
      type: string
      required: false
      default: development

    - name: db_size
      type: string
      required: false
      default: small

    - name: enable_backup
      type: boolean
      required: false
      default: false

  variables:
    namespace_name: ${parameters.app_name}-${parameters.environment}
    team_id: dev-team

  steps:
    # Step 1: Container Team - Create Namespace
    - name: provision-namespace
      type: workflow
      workflow: container-team/provision-namespace
      parameters:
        namespace_name: ${variables.namespace_name}
        resource_quota:
          cpu: "4"
          memory: "8Gi"
          pods: "20"
        labels:
          app: ${parameters.app_name}
          environment: ${parameters.environment}
          managed-by: innominatus

    # Step 2: Vault Team - Create Vault Space
    - name: provision-vault-space
      type: workflow
      workflow: vault-team/provision-vault-space
      parameters:
        app_name: ${parameters.app_name}
        namespace: ${variables.namespace_name}
        service_account: ${parameters.app_name}-sa
      dependsOn:
        - provision-namespace

    # Step 3: Database Team - Create PostgreSQL Database
    - name: provision-database
      type: workflow
      workflow: database-team/provision-postgres
      parameters:
        db_name: ${parameters.app_name}
        namespace: ${variables.namespace_name}
        team_id: ${variables.team_id}
        size: ${parameters.db_size}
        replicas: 2
        version: "15"
      dependsOn:
        - provision-namespace

    # Step 4: Storage Team - Create S3 Bucket
    - name: provision-storage
      type: workflow
      workflow: storage-team/provision-s3-bucket
      parameters:
        bucket_name: ${parameters.app_name}-${parameters.environment}
        namespace: ${variables.namespace_name}
        versioning: true
        public_access: false
      dependsOn:
        - provision-namespace

    # Step 5: Store DB Credentials in Vault
    - name: store-db-credentials-in-vault
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Store database credentials in Vault
          vault kv put secret/database \
            username="${provision-database.username}" \
            password="${provision-database.password}" \
            host="${provision-database.host}" \
            port="${provision-database.port}" \
            database="${provision-database.database_name}" \
            connection_string="${provision-database.connection_string}"
      dependsOn:
        - provision-database
        - provision-vault-space

    # Step 6: Store S3 Credentials in Vault
    - name: store-s3-credentials-in-vault
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          export VAULT_ADDR=http://vault.localtest.me
          export VAULT_TOKEN=root
          export VAULT_NAMESPACE=applications/${parameters.app_name}

          # Store S3 credentials in Vault
          vault kv put secret/storage \
            access_key="${provision-storage.access_key}" \
            secret_key="${provision-storage.secret_key}" \
            bucket="${provision-storage.bucket_name}" \
            endpoint="${provision-storage.endpoint}" \
            s3_url="${provision-storage.s3_url}"
      dependsOn:
        - provision-storage
        - provision-vault-space

    # Step 7: Setup VSO to sync DB credentials to K8s Secret
    - name: sync-db-secret-to-k8s
      type: kubernetes
      config:
        action: create
        namespace: ${variables.namespace_name}
        manifest: |
          apiVersion: secrets.hashicorp.com/v1beta1
          kind: VaultStaticSecret
          metadata:
            name: ${parameters.app_name}-database-credentials
            namespace: ${variables.namespace_name}
          spec:
            vaultAuthRef: ${provision-vault-space.vso_auth_ref}
            mount: secret
            type: kv-v2
            path: database
            refreshAfter: 30s
            destination:
              name: database-credentials
              create: true
              labels:
                managed-by: innominatus
                type: database
      dependsOn:
        - store-db-credentials-in-vault

    # Step 8: Create Git Repository for App
    - name: create-git-repo
      type: gitea-repo
      config:
        repoName: ${parameters.app_name}
        owner: dev-team
        description: Application repository for ${parameters.app_name}
        private: false
        autoInit: true

    # Step 9: Create Kubernetes Manifests
    - name: create-k8s-manifests
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          mkdir -p /tmp/${parameters.app_name}/k8s

          # Create ServiceAccount
          cat > /tmp/${parameters.app_name}/k8s/serviceaccount.yaml <<EOF
          apiVersion: v1
          kind: ServiceAccount
          metadata:
            name: ${parameters.app_name}-sa
            namespace: ${variables.namespace_name}
          EOF

          # Create Deployment
          cat > /tmp/${parameters.app_name}/k8s/deployment.yaml <<EOF
          apiVersion: apps/v1
          kind: Deployment
          metadata:
            name: ${parameters.app_name}
            namespace: ${variables.namespace_name}
            labels:
              app: ${parameters.app_name}
              environment: ${parameters.environment}
          spec:
            replicas: 2
            selector:
              matchLabels:
                app: ${parameters.app_name}
            template:
              metadata:
                labels:
                  app: ${parameters.app_name}
              spec:
                serviceAccountName: ${parameters.app_name}-sa
                containers:
                - name: app
                  image: nginx:latest  # Replace with actual app image
                  ports:
                  - containerPort: 8080
                  env:
                  # Database connection from Vault secret
                  - name: DATABASE_URL
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: connection_string
                  - name: DB_HOST
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: host
                  - name: DB_PORT
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: port
                  - name: DB_NAME
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: database
                  - name: DB_USER
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: username
                  - name: DB_PASSWORD
                    valueFrom:
                      secretKeyRef:
                        name: database-credentials
                        key: password

                  # S3 storage configuration
                  - name: S3_ENDPOINT
                    value: "${provision-storage.endpoint}"
                  - name: S3_BUCKET
                    value: "${provision-storage.bucket_name}"
                  - name: S3_ACCESS_KEY
                    value: "${provision-storage.access_key}"
                  - name: S3_SECRET_KEY
                    value: "${provision-storage.secret_key}"
                  - name: S3_URL
                    value: "${provision-storage.s3_url}"

                  resources:
                    requests:
                      cpu: 100m
                      memory: 128Mi
                    limits:
                      cpu: 500m
                      memory: 512Mi
          EOF

          # Create Service
          cat > /tmp/${parameters.app_name}/k8s/service.yaml <<EOF
          apiVersion: v1
          kind: Service
          metadata:
            name: ${parameters.app_name}
            namespace: ${variables.namespace_name}
          spec:
            selector:
              app: ${parameters.app_name}
            ports:
            - port: 80
              targetPort: 8080
            type: ClusterIP
          EOF

          echo "Manifests created successfully"
      dependsOn:
        - create-git-repo

    # Step 10: Commit Manifests to Git
    - name: commit-manifests
      type: git-commit-manifests
      config:
        repoName: ${parameters.app_name}
        owner: dev-team
        sourcePath: /tmp/${parameters.app_name}/k8s
        targetPath: k8s
        commitMessage: "feat: initial Kubernetes manifests"
      dependsOn:
        - create-k8s-manifests

    # Step 11: Container Team - Create ArgoCD Application
    - name: provision-argocd-app
      type: workflow
      workflow: container-team/provision-argocd-app
      parameters:
        app_name: ${parameters.app_name}
        namespace: ${variables.namespace_name}
        repo_url: http://gitea-http.gitea.svc.cluster.local:3000/dev-team/${parameters.app_name}
        path: k8s
        sync_policy: auto
      dependsOn:
        - commit-manifests

    # Step 12: Wait for Application to be Healthy
    - name: wait-for-app-health
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          echo "Waiting for application to be healthy..."
          kubectl wait --for=condition=Available \
            deployment/${parameters.app_name} \
            -n ${variables.namespace_name} \
            --timeout=600s

          echo "Application is healthy!"
      dependsOn:
        - provision-argocd-app

    # Step 13: Verify Secret Sync
    - name: verify-secret-sync
      type: policy
      config:
        command: |
          #!/bin/bash
          set -e

          echo "Verifying secret sync from Vault to Kubernetes..."

          # Check if secret exists
          kubectl get secret database-credentials -n ${variables.namespace_name}

          # Verify secret has correct keys
          kubectl get secret database-credentials -n ${variables.namespace_name} -o json | \
            jq -e '.data | has("username") and has("password") and has("connection_string")'

          echo "Secret sync verified successfully!"
      dependsOn:
        - wait-for-app-health

  outputs:
    app_name: ${parameters.app_name}
    namespace: ${variables.namespace_name}
    environment: ${parameters.environment}

    # Database outputs
    database_host: ${provision-database.host}
    database_port: ${provision-database.port}
    database_name: ${provision-database.database_name}
    database_secret: database-credentials

    # Storage outputs
    s3_bucket: ${provision-storage.bucket_name}
    s3_endpoint: ${provision-storage.endpoint}

    # Vault outputs
    vault_namespace: ${provision-vault-space.vault_namespace}

    # Git outputs
    git_repo_url: http://gitea.localtest.me/dev-team/${parameters.app_name}

    # ArgoCD outputs
    argocd_app_url: ${provision-argocd-app.app_url}

    # Status
    deployment_status: healthy
    secret_sync_status: active
```

### Phase 7: Dev Team Score Specification

#### 7.1 Score Spec for Dev Team App

**File**: `examples/dev-team-app.yaml`

```yaml
apiVersion: score.dev/v1b1

metadata:
  name: my-awesome-app

containers:
  main:
    image: nginx:latest  # Replace with actual application image

    # Environment variables using resource outputs
    variables:
      # Database connection (from Vault-synced secret)
      DATABASE_URL: ${resources.database.outputs.connection_string}
      DB_HOST: ${resources.database.outputs.host}
      DB_PORT: ${resources.database.outputs.port}
      DB_NAME: ${resources.database.outputs.database_name}

      # S3 Storage connection
      S3_ENDPOINT: ${resources.storage.outputs.endpoint}
      S3_BUCKET: ${resources.storage.outputs.bucket_name}
      S3_URL: ${resources.storage.outputs.s3_url}

      # Application config
      APP_NAME: my-awesome-app
      ENVIRONMENT: ${environment.type}

# Resource declarations (provisioned by product teams)
resources:
  # Vault team provides secret space
  vault:
    type: vault-space
    params:
      sync_to_namespace: true
      auto_rotate: false

  # Database team provides PostgreSQL
  database:
    type: postgres-db
    params:
      size: small
      replicas: 2
      version: "15"

  # Storage team provides S3 bucket
  storage:
    type: s3-bucket
    params:
      versioning: true
      public_access: false

  # Container team provides namespace and ArgoCD app
  platform:
    type: argocd-app
    params:
      sync_policy: auto

# Environment configuration
environment:
  type: development
  ttl: 168h  # 7 days for dev environment
```

## Demo Execution

### Setup Phase

```bash
# 1. Ensure demo environment is running with PostgreSQL operator
./innominatus-ctl demo-time

# 2. Verify all services are healthy
./innominatus-ctl demo-status

# Expected output should show:
# ✅ Gitea: Running
# ✅ ArgoCD: Running
# ✅ Vault: Running
# ✅ Minio: Running
# ✅ PostgreSQL Operator: Running
```

### Product Teams Setup

```bash
# 3. Setup product team providers in Gitea repositories
# This script creates organizations, repositories, and pushes provider content
./scripts/setup-demo-providers.sh

# Expected output:
# 🚀 Setting up Product Team Providers in Gitea
# ==============================================
#
# Step 1: Creating Gitea organizations
# -----------------------------------
# 📁 Creating organization: container-team
#    ✅ Organization ready
# 📁 Creating organization: database-team
#    ✅ Organization ready
# 📁 Creating organization: storage-team
#    ✅ Organization ready
# 📁 Creating organization: vault-team
#    ✅ Organization ready
#
# Step 2: Creating Git repositories
# -------------------------------
# 📦 Creating repository: container-team/container-team-provider
#    ✅ Repository ready
# 📦 Creating repository: database-team/database-team-provider
#    ✅ Repository ready
# 📦 Creating repository: storage-team/storage-team-provider
#    ✅ Repository ready
# 📦 Creating repository: vault-team/vault-team-provider
#    ✅ Repository ready
#
# Step 3: Pushing provider content
# ------------------------------
# 📤 Pushing container-team provider content to Git
#    ✅ Content pushed to Git
# 📤 Pushing database-team provider content to Git
#    ✅ Content pushed to Git
# 📤 Pushing storage-team provider content to Git
#    ✅ Content pushed to Git
# 📤 Pushing vault-team provider content to Git
#    ✅ Content pushed to Git
#
# 🎉 Setup Complete!
#
# Product team providers are now available in Gitea:
#   • http://gitea.localtest.me/container-team/container-team-provider
#   • http://gitea.localtest.me/database-team/database-team-provider
#   • http://gitea.localtest.me/storage-team/storage-team-provider
#   • http://gitea.localtest.me/vault-team/vault-team-provider

# 4. (Optional) Create dev-team organization for application repositories
./innominatus-ctl run team-setup --team dev-team
```

### Configure Innominatus Admin

```bash
# 5. Create or update admin-config.yaml to register Git-based product team providers
cat > admin-config.yaml <<EOF
# Innominatus Admin Configuration
# This file registers all product team providers from Git repositories

providers:
  - name: container-team
    type: git
    category: platform
    description: Container platform provisioners from container-team
    repository: http://gitea-http.gitea.svc.cluster.local:3000/container-team/container-team-provider
    ref: main
    enabled: true

  - name: database-team
    type: git
    category: data
    description: Database provisioners from database-team
    repository: http://gitea-http.gitea.svc.cluster.local:3000/database-team/database-team-provider
    ref: main
    enabled: true

  - name: storage-team
    type: git
    category: storage
    description: Object storage provisioners from storage-team
    repository: http://gitea-http.gitea.svc.cluster.local:3000/storage-team/storage-team-provider
    ref: main
    enabled: true

  - name: vault-team
    type: git
    category: security
    description: Secret management provisioners from vault-team
    repository: http://gitea-http.gitea.svc.cluster.local:3000/vault-team/vault-team-provider
    ref: main
    enabled: true
EOF

# 6. Restart innominatus to reload providers from Git
pkill innominatus
./innominatus &

# Wait for server to start
sleep 3

# 7. Verify providers are loaded from Git repositories
./innominatus-ctl list-providers

# Expected output should show all 4 product teams with their provisioners:
# container-team:
#   - namespace (kubernetes-namespace)
#   - argocd-app (argocd-application)
# database-team:
#   - postgres-db (postgresql)
# storage-team:
#   - s3-bucket (s3)
# vault-team:
#   - vault-space (vault-namespace)
```

### Dev Team Onboarding (The Main Demo)

```bash
# 9. Dev team onboards their application using the golden path
./innominatus-ctl run onboard-dev-team examples/dev-team-app.yaml \
  --param app_name=my-awesome-app \
  --param environment=development \
  --param db_size=small

# This single command orchestrates:
# - container-team: Creates namespace
# - vault-team: Creates Vault space with VSO
# - database-team: Provisions PostgreSQL database
# - storage-team: Creates S3 bucket
# - Stores credentials in Vault
# - Syncs DB credentials to K8s secret via VSO
# - Creates Git repository
# - Commits K8s manifests
# - container-team: Creates ArgoCD application
# - Deploys application with env vars configured
```

### Verification

```bash
# 10. Check workflow execution
./innominatus-ctl workflow list
./innominatus-ctl workflow detail <workflow-id>
./innominatus-ctl workflow logs <workflow-id>

# 11. Verify namespace and resources
kubectl get namespaces | grep my-awesome-app
kubectl get all -n my-awesome-app-development

# 12. Verify PostgreSQL database
kubectl get postgresql -n my-awesome-app-development
kubectl get pods -n my-awesome-app-development | grep postgres

# 13. Verify Vault secret sync
kubectl get vaultstaticsecret -n my-awesome-app-development
kubectl get secret database-credentials -n my-awesome-app-development
kubectl get secret database-credentials -n my-awesome-app-development -o yaml

# 14. Verify S3 bucket
mc alias set minio http://minio.localtest.me minioadmin minioadmin
mc ls minio/ | grep my-awesome-app

# 15. Verify ArgoCD application
kubectl get application -n argocd | grep my-awesome-app
# Or visit: http://argocd.localtest.me

# 16. Check application pod environment variables
POD=$(kubectl get pod -n my-awesome-app-development -l app=my-awesome-app -o jsonpath='{.items[0].metadata.name}')
kubectl exec -n my-awesome-app-development $POD -- env | grep -E "DATABASE|S3"

# Expected output:
# DATABASE_URL=postgresql://...
# DB_HOST=dev-team-my-awesome-app.my-awesome-app-development.svc.cluster.local
# DB_PORT=5432
# DB_NAME=my-awesome-app
# S3_ENDPOINT=http://minio.minio.svc.cluster.local:9000
# S3_BUCKET=my-awesome-app-development
# S3_URL=s3://my-awesome-app-development

# 17. Verify database password is synced from Vault
kubectl get secret database-credentials -n my-awesome-app-development \
  -o jsonpath='{.data.password}' | base64 -d
# This should show the actual database password from Vault
```

## Stakeholder Presentation Flow

### Act 1: Product Teams Setup (5 minutes)

**Narrative**: "We have 4 product teams, each responsible for a different infrastructure component."

1. Show Gitea organizations and repositories for each team
2. Demonstrate one provider configuration (e.g., storage-team)
3. Show how workflows are defined in Git

**Key Points**:
- Each team owns their infrastructure code
- Teams version their providers in Git
- Changes are tracked and auditable

### Act 2: Platform Configuration (2 minutes)

**Narrative**: "The platform team registers these providers in the central configuration."

1. Show `admin-config.yaml` with provider registrations
2. Run `list-providers` to show all available provisioners

**Key Points**:
- Central registry of available services
- Platform team controls which versions are active
- Providers can be enabled/disabled dynamically

### Act 3: Dev Team Onboarding (10 minutes)

**Narrative**: "A dev team wants to deploy their application. They describe what they need in a Score specification."

1. Show `examples/dev-team-app.yaml`
2. Explain resource declarations
3. Execute the golden path: `./innominatus-ctl run onboard-dev-team ...`
4. Watch real-time progress in terminal

**Key Points**:
- Dev team doesn't need to know how infrastructure is provisioned
- Single command orchestrates all product teams
- Golden path enforces best practices

### Act 4: Verification (3 minutes)

**Narrative**: "Let's verify everything was provisioned correctly."

1. Show namespace with all resources
2. Display pod with environment variables
3. Show Vault-synced database secret
4. Open ArgoCD UI to show GitOps deployment

**Key Points**:
- Database credentials automatically synced via Vault
- Environment variables configured correctly
- Application deployed via GitOps
- Full audit trail in workflow logs

## Success Criteria & Acceptance Criteria

At the end of the demo, stakeholders should see:

### 1. ✅ **Namespace created** by container-team

**Business Outcome:** Development team has an isolated Kubernetes namespace with resource quotas

**Given** the onboard-dev-team workflow has completed successfully
**When** checking Kubernetes namespaces
**Then** namespace exists with proper labels and resource quotas applied

**Technical verification:**
```bash
kubectl get namespace my-awesome-app-development
kubectl get namespace my-awesome-app-development -o jsonpath='{.metadata.labels}'
kubectl get resourcequota -n my-awesome-app-development
```

**Expected results:**
- Namespace `my-awesome-app-development` exists with status `Active`
- Labels include: `managed-by: innominatus`, `app: my-awesome-app`, `environment: development`
- ResourceQuota exists with CPU (4 cores), Memory (8Gi), and Pod limits (20 pods)
- NetworkPolicy `default-deny-ingress` exists

---

### 2. ✅ **PostgreSQL database running** provisioned by database-team

**Business Outcome:** Application has a managed PostgreSQL database with credentials

**Given** the database provisioning step has completed
**When** checking PostgreSQL cluster status
**Then** database cluster is running with correct configuration

**Technical verification:**
```bash
kubectl get postgresql -n my-awesome-app-development
kubectl get pods -n my-awesome-app-development | grep postgres
kubectl get secret -n my-awesome-app-development | grep credentials
```

**Expected results:**
- PostgreSQL cluster `dev-team-my-awesome-app` exists with status `Running`
- PostgreSQL pods (2 replicas) are in `Running` state
- Database credentials secret exists: `dev-team-my-awesome-app.my-awesome-app-app.credentials`
- Database is accessible at: `dev-team-my-awesome-app.my-awesome-app-development.svc.cluster.local:5432`

**Connectivity test:**
```bash
DB_SECRET="dev-team-my-awesome-app.my-awesome-app-app.credentials"
DB_PASSWORD=$(kubectl get secret $DB_SECRET -n my-awesome-app-development -o jsonpath='{.data.password}' | base64 -d)
kubectl run -it --rm psql-test --image=postgres:15 --restart=Never -n my-awesome-app-development -- \
  psql postgresql://my_awesome_app_app:$DB_PASSWORD@dev-team-my-awesome-app.my-awesome-app-development.svc.cluster.local:5432/my-awesome-app -c '\l'
```

---

### 3. ✅ **S3 bucket created** by storage-team

**Business Outcome:** Application has object storage with access credentials

**Given** the storage provisioning step has completed
**When** checking Minio buckets
**Then** bucket exists with correct access policies

**Technical verification:**
```bash
mc alias set minio http://minio.localtest.me minioadmin minioadmin
mc ls minio/ | grep my-awesome-app-development
kubectl get secret -n my-awesome-app-development | grep s3-credentials
```

**Expected results:**
- S3 bucket `my-awesome-app-development` exists in Minio
- Bucket has versioning enabled
- Secret `my-awesome-app-development-s3-credentials` exists in namespace
- Secret contains keys: `access-key`, `secret-key`, `endpoint`, `bucket`, `s3-url`

---

### 4. ✅ **Vault namespace configured** by vault-team with VSO

**Business Outcome:** Application secrets are managed in Vault and synced to Kubernetes

**Given** the vault provisioning and secret sync steps have completed
**When** checking Vault and VSO resources
**Then** Vault namespace exists and secrets are synced to Kubernetes

**Technical verification:**
```bash
# Check Vault namespace
export VAULT_ADDR=http://vault.localtest.me
export VAULT_TOKEN=root
vault namespace list | grep my-awesome-app

# Check VSO resources
kubectl get vaultconnection -n my-awesome-app-development
kubectl get vaultauth -n my-awesome-app-development
kubectl get vaultstaticsecret -n my-awesome-app-development
```

**Expected results:**
- Vault namespace `applications/my-awesome-app` exists
- VaultConnection `my-awesome-app-vault-connection` exists
- VaultAuth `my-awesome-app-vault-auth` exists with Kubernetes auth method
- VaultStaticSecret `my-awesome-app-database-credentials` exists with sync status

---

### 5. ✅ **Kubernetes secret synced from Vault**

**Business Outcome:** Database credentials are automatically synced from Vault to Kubernetes secret

**Given** VSO has completed database credential sync
**When** checking the synced Kubernetes secret
**Then** secret exists with all required database connection fields

**Technical verification:**
```bash
kubectl get secret database-credentials -n my-awesome-app-development
kubectl get secret database-credentials -n my-awesome-app-development -o yaml
kubectl get secret database-credentials -n my-awesome-app-development -o jsonpath='{.data}' | jq 'keys'
```

**Expected results:**
- Secret `database-credentials` exists in namespace
- Secret type is `Opaque`
- Secret contains keys: `username`, `password`, `host`, `port`, `database`, `connection_string`
- Label `managed-by: innominatus` is present
- Secret data is base64-encoded and can be decoded:
```bash
kubectl get secret database-credentials -n my-awesome-app-development -o jsonpath='{.data.password}' | base64 -d
```

---

### 6. ✅ **Application pod running** with correct environment variables

**Business Outcome:** Application pod is running with database and S3 credentials configured as environment variables

**Given** ArgoCD has deployed the application
**When** checking application pods and environment variables
**Then** pods are running with correct environment configuration

**Technical verification:**
```bash
# Check pod status
kubectl get pods -n my-awesome-app-development -l app=my-awesome-app

# Get pod name
POD=$(kubectl get pod -n my-awesome-app-development -l app=my-awesome-app -o jsonpath='{.items[0].metadata.name}')

# Check environment variables
kubectl exec -n my-awesome-app-development $POD -- env | sort | grep -E "DATABASE|DB_|S3_"
```

**Expected results:**
- At least 1 pod with label `app=my-awesome-app` is in `Running` state
- Pod has `Ready 1/1` status
- Pod uses ServiceAccount `my-awesome-app-sa`

**Environment variables must include:**
- `DATABASE_URL` - Full PostgreSQL connection string from Vault secret
- `DB_HOST` - Database host (from Vault secret)
- `DB_PORT` - Database port (from Vault secret, should be `5432`)
- `DB_NAME` - Database name (from Vault secret)
- `DB_USER` - Database username (from Vault secret)
- `DB_PASSWORD` - Database password (from Vault secret)
- `S3_ENDPOINT` - Minio endpoint (`http://minio.minio.svc.cluster.local:9000`)
- `S3_BUCKET` - Bucket name (`my-awesome-app-development`)
- `S3_ACCESS_KEY` - S3 access key
- `S3_SECRET_KEY` - S3 secret key
- `S3_URL` - S3 URL (`s3://my-awesome-app-development`)

**Verify environment variables are correctly set:**
```bash
POD=$(kubectl get pod -n my-awesome-app-development -l app=my-awesome-app -o jsonpath='{.items[0].metadata.name}')

# Check DATABASE_URL format
kubectl exec -n my-awesome-app-development $POD -- sh -c 'echo $DATABASE_URL' | grep -E "^postgresql://.+:.+@.+:5432/.+$"

# Check S3_ENDPOINT
kubectl exec -n my-awesome-app-development $POD -- sh -c 'echo $S3_ENDPOINT' | grep "minio.minio.svc.cluster.local:9000"

# Check S3_BUCKET
kubectl exec -n my-awesome-app-development $POD -- sh -c 'echo $S3_BUCKET' | grep "my-awesome-app-development"
```

---

### 7. ✅ **ArgoCD application** deploying from Git

**Business Outcome:** Application is deployed via GitOps with ArgoCD syncing from Git repository

**Given** the ArgoCD application has been created
**When** checking ArgoCD application status
**Then** application is synced and healthy

**Technical verification:**
```bash
# Check ArgoCD application
kubectl get application -n argocd my-awesome-app
kubectl get application -n argocd my-awesome-app -o jsonpath='{.status.sync.status}'
kubectl get application -n argocd my-awesome-app -o jsonpath='{.status.health.status}'

# Check in ArgoCD UI
open http://argocd.localtest.me/applications/my-awesome-app
```

**Expected results:**
- Application `my-awesome-app` exists in ArgoCD namespace
- Sync status is `Synced` (not `OutOfSync`)
- Health status is `Healthy` (not `Degraded` or `Missing`)
- Application source repository: `http://gitea-http.gitea.svc.cluster.local:3000/dev-team/my-awesome-app`
- Application source path: `k8s`
- Target revision: `HEAD`
- Destination namespace: `my-awesome-app-development`
- Auto-sync enabled with prune and self-heal

**Verify Git repository:**
```bash
# Check Git repository exists
curl -s http://gitea.localtest.me/api/v1/repos/dev-team/my-awesome-app | jq '.name'

# Verify k8s manifests exist in repo
curl -s http://gitea.localtest.me/api/v1/repos/dev-team/my-awesome-app/contents/k8s | jq '.[].name'
# Should show: deployment.yaml, service.yaml, serviceaccount.yaml
```

---

### 8. ✅ **Complete audit trail** in workflow execution logs

**Business Outcome:** Platform team can audit entire deployment process with detailed logs

**Given** the onboard-dev-team workflow has completed
**When** checking workflow execution history
**Then** complete audit trail is available with all steps logged

**Technical verification:**
```bash
# List workflow executions
./innominatus-ctl workflow list

# Get workflow details
WORKFLOW_ID=$(./innominatus-ctl workflow list --format json | jq -r '.[0].id')
./innominatus-ctl workflow detail $WORKFLOW_ID

# View workflow logs
./innominatus-ctl workflow logs $WORKFLOW_ID
```

**Expected results:**
- Workflow execution record exists in database
- Workflow status is `completed` (not `failed` or `running`)
- All 13 steps show status `completed`
- Workflow outputs include: namespace, database credentials, S3 bucket info, Vault namespace, Git repo URL, ArgoCD app URL
- Logs contain detailed output from each step including:
  - Namespace creation
  - Vault space provisioning
  - Database provisioning
  - S3 bucket creation
  - Secret storage in Vault
  - VSO secret sync
  - Git repository creation
  - Manifest generation and commit
  - ArgoCD application creation
  - Health check verification

## Next Steps

After the demo, you can extend this with:

1. **Add monitoring**: Integrate Grafana dashboards for each app
2. **Add observability**: Configure traces and logs
3. **Add compliance**: Implement approval workflows for production
4. **Add cost tracking**: Tag resources with team/app for chargebacks
5. **Add self-service portal**: Build Web UI for dev teams to request resources

## Troubleshooting

### PostgreSQL Operator Issues

```bash
# Check operator status
kubectl get pods -n postgres-operator

# View operator logs
kubectl logs -n postgres-operator -l app.kubernetes.io/name=postgres-operator
```

### Vault Secret Sync Issues

```bash
# Check VSO status
kubectl get pods -n vault-secrets-operator-system

# Check VaultStaticSecret status
kubectl describe vaultstaticsecret -n <namespace>

# Check Vault auth
kubectl logs -n <namespace> <pod-name>
```

### ArgoCD Sync Issues

```bash
# Check application status
kubectl get application -n argocd <app-name> -o yaml

# Sync manually
kubectl patch application <app-name> -n argocd \
  -p '{"operation":{"initiatedBy":{"username":"admin"},"sync":{"revision":"HEAD"}}}' \
  --type=merge
```

## Cleanup

```bash
# Remove dev team application
./innominatus-ctl undeploy-app my-awesome-app

# Cleanup entire demo environment
./innominatus-ctl demo-nuke
```

---

**Document Version**: 1.0.0
**Last Updated**: 2025-10-28
**Author**: Platform Team
**Status**: Implementation Ready
