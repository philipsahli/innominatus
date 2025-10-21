# High Availability Architecture Analysis

**Date:** 2025-10-06
**Current Status:** 🔴 **NOT HA-READY**
**Target:** Support 3+ replicas with full resilience
**Analyst:** Claude Code AI Assistant

---

## Executive Summary

innominatus currently **cannot run in High Availability (HA) mode** with multiple replicas due to fundamental architectural limitations. The async queue implementation (Gap #61 - recently resolved) uses **in-memory Go channels**, which makes it incompatible with horizontal scaling.

### Critical Blockers

| Component | Current State | HA Blocker | Impact |
|-----------|---------------|------------|--------|
| **Async Queue** | In-memory Go channels | ❌ Tasks lost on pod restart | 🔴 **CRITICAL** |
| **Metrics** | In-memory counters | ❌ Per-replica metrics only | 🟡 **HIGH** |
| **Active Tasks** | Local map | ❌ No cross-replica visibility | 🟡 **HIGH** |
| **Workspaces** | Local filesystem | ❌ Not shared across pods | 🔴 **CRITICAL** |
| **Sessions** | In-memory + PostgreSQL | ⚠️ Partially HA-ready | 🟡 **MEDIUM** |
| **Login Tracking** | In-memory map | ❌ Rate limiting per-replica | 🟡 **MEDIUM** |
| **Database** | Single PostgreSQL | ⚠️ Single point of failure | 🔴 **CRITICAL** |

### What Needs to Change

To support 3+ replicas, the architecture must migrate from:
- **In-memory state** → **Distributed state** (Redis, PostgreSQL)
- **Local filesystem** → **Shared storage** (PVC ReadWriteMany, S3)
- **Single database** → **PostgreSQL HA** (Primary-Replica, Patroni)
- **Process-local queue** → **Distributed queue** (Redis, PostgreSQL-backed)

**Estimated Effort:** 6-8 weeks for full HA implementation

---

## Table of Contents

1. [Current Architecture Analysis](#1-current-architecture-analysis)
2. [HA Blockers by Component](#2-ha-blockers-by-component)
3. [Distributed Queue Architecture](#3-distributed-queue-architecture)
4. [Database HA Strategy](#4-database-ha-strategy)
5. [Shared Storage Strategy](#5-shared-storage-strategy)
6. [Session & Authentication](#6-session--authentication)
7. [Metrics & Observability](#7-metrics--observability)
8. [Implementation Roadmap](#8-implementation-roadmap)
9. [Trade-offs & Considerations](#9-trade-offs--considerations)
10. [Example 3-Replica Deployment](#10-example-3-replica-deployment)

---

## 1. Current Architecture Analysis

### 1.1 Single-Replica Architecture (Current)

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Pod                         │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  innominatus Server (Single Replica)                   │ │
│  │                                                          │ │
│  │  ┌──────────────────────────────────┐                  │ │
│  │  │  In-Memory Queue                 │                  │ │
│  │  │  - Go channel (buffered 100)     │                  │ │
│  │  │  - 5 worker goroutines           │                  │ │
│  │  │  - activeTasks map               │                  │ │
│  │  │  - metricsCollector (in-memory)  │                  │ │
│  │  └──────────────────────────────────┘                  │ │
│  │                                                          │ │
│  │  ┌──────────────────────────────────┐                  │ │
│  │  │  Local Filesystem                │                  │ │
│  │  │  - /workspaces (RWO PVC)         │                  │ │
│  │  │  - Terraform state               │                  │ │
│  │  │  - Ansible playbooks             │                  │ │
│  │  └──────────────────────────────────┘                  │ │
│  │                                                          │ │
│  │  ┌──────────────────────────────────┐                  │ │
│  │  │  In-Memory State                 │                  │ │
│  │  │  - loginAttempts map             │                  │ │
│  │  │  - memoryWorkflows map           │                  │ │
│  │  └──────────────────────────────────┘                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
                ┌──────────────────┐
                │   PostgreSQL     │
                │  (Single Node)   │
                │  - queue_tasks   │
                │  - workflows     │
                │  - sessions      │
                └──────────────────┘
```

### 1.2 Problems with Multi-Replica Deployment

**Scenario:** Deploy 3 replicas with current architecture

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Replica 1  │  │  Replica 2  │  │  Replica 3  │
│  Queue: []  │  │  Queue: []  │  │  Queue: []  │  ❌ Separate queues
│  Tasks: 2   │  │  Tasks: 1   │  │  Tasks: 0   │  ❌ No shared state
│  Metrics: X │  │  Metrics: Y │  │  Metrics: Z │  ❌ Fragmented metrics
└─────────────┘  └─────────────┘  └─────────────┘
       │                │                │
       └────────────────┴────────────────┘
                        │
                        ▼
                ┌──────────────┐
                │  PostgreSQL  │
                │   (Shared)   │
                └──────────────┘
```

**Critical Issues:**

1. **Task Distribution:** No mechanism to distribute tasks across replicas
2. **Duplicate Execution:** Same task could be picked up by multiple workers
3. **Lost Tasks:** Tasks in-memory channels lost on pod restart
4. **Fragmented Metrics:** Each replica has separate counters
5. **No Visibility:** Can't see active tasks across all replicas
6. **Workspace Isolation:** Each replica has separate filesystem

---

## 2. HA Blockers by Component

### 2.1 Async Queue (`internal/queue/queue.go`)

**Current Implementation:**
```go
type Queue struct {
    tasks           chan *WorkflowTask      // ❌ In-memory channel
    workers         int
    executor        WorkflowExecutor
    db              *database.Database
    logger          *logging.ZerologAdapter
    wg              sync.WaitGroup
    ctx             context.Context
    cancel          context.CancelFunc
    mu              sync.RWMutex
    activeTasks     map[string]*WorkflowTask // ❌ Process-local map
    taskStatusChan  chan taskStatusUpdate    // ❌ In-memory channel
    metricsCollector *MetricsCollector       // ❌ In-memory counters
}
```

**HA Blockers:**

| Issue | Current Behavior | HA Impact |
|-------|------------------|-----------|
| **In-memory queue** | `chan *WorkflowTask` | Tasks lost on pod restart |
| **Local active tasks** | `map[string]*WorkflowTask` | No cross-replica visibility |
| **Process-bound workers** | 5 goroutines per pod | Workers can't share work |
| **In-memory metrics** | Local counters | Metrics fragmented across replicas |
| **No task locking** | No coordination | Duplicate task execution |

**Code References:**
- `queue.go:76` - Channel creation: `make(chan *WorkflowTask, 100)`
- `queue.go:83` - Active tasks map: `make(map[string]*WorkflowTask)`
- `queue.go:140` - Enqueue to channel: `q.tasks <- task`
- `queue.go:186` - Worker reads from channel: `task, ok := <-q.tasks`

**Why This Fails in HA:**
- **Pod A** enqueues task → stored in **Pod A's** channel
- **Pod A** crashes → task lost (never persisted as "pending")
- **Pod B** can't access **Pod A's** channel
- **Pod C** can't see active tasks in **Pod A** or **Pod B**

### 2.2 Metrics Collector (`internal/queue/queue.go:62-69`)

**Current Implementation:**
```go
type MetricsCollector struct {
    mu                sync.RWMutex
    tasksEnqueued     int64  // ❌ Per-replica counter
    tasksCompleted    int64  // ❌ Per-replica counter
    tasksFailed       int64  // ❌ Per-replica counter
    totalQueueTime    time.Duration  // ❌ Per-replica
    totalExecutionTime time.Duration // ❌ Per-replica
}
```

**HA Impact:**
- `/api/queue/stats` returns **different values** per replica
- Load balancer distributes requests → **inconsistent metrics**
- No global view of queue health

**Example Problem:**
```bash
# Request hits Replica 1
curl /api/queue/stats
# {"tasks_enqueued": 100, "tasks_completed": 95}

# Request hits Replica 2
curl /api/queue/stats
# {"tasks_enqueued": 50, "tasks_completed": 48}

# Which one is correct? Neither - need aggregated view!
```

### 2.3 Workspace Storage (`internal/workflow/executor.go`)

**Current Implementation:**
```go
// Workspaces are created on local filesystem
workspaceDir := fmt.Sprintf("./workspaces/%s-%s", appName, environment)
os.MkdirAll(workspaceDir, 0755)
```

**Helm Chart Configuration:**
```yaml
# values.yaml:218-223
persistence:
  enabled: true
  storageClass: ""
  accessMode: ReadWriteOnce  # ❌ Only one pod can mount
  size: 10Gi
```

**HA Blocker:**
- **ReadWriteOnce (RWO):** PVC can only mount to **one pod**
- Multi-replica deployment → **each pod gets separate PVC**
- Terraform state, Ansible artifacts → **not shared**
- Task executed on **Replica A** → workspace only on **Replica A**

**Why This Matters:**
- **Scenario 1:** Task starts on Pod A, creates Terraform workspace. Pod A crashes. Task retried on Pod B → **no access to workspace**.
- **Scenario 2:** User queries workflow status from Pod C → **can't see logs/artifacts from Pod A**.

**Required Change:**
- **ReadWriteMany (RWX):** Allows multiple pods to mount same volume
- **Supported by:** NFS, CephFS, Azure Files, AWS EFS
- **Not supported by:** Local storage, EBS, GCE PD

### 2.4 Session Management (`internal/server/handlers.go:129-130`)

**Current Implementation:**
```go
type Server struct {
    // ...
    sessionManager    auth.ISessionManager
    loginAttempts     map[string][]time.Time  // ❌ In-memory rate limiting
    loginMutex        sync.Mutex
}
```

**HA Status:** ⚠️ **Partially HA-Ready**

**What Works:**
- Sessions stored in PostgreSQL → shared across replicas
- API key authentication works across replicas

**What Doesn't Work:**
- `loginAttempts` map is **per-replica**
- Attacker can bypass rate limiting by hitting different replicas
- Each replica allows **N** attempts independently

**Example Attack:**
```bash
# Hit Replica 1: 5 failed attempts
for i in {1..5}; do curl /login -d "user=admin&pass=wrong"; done
# Replica 1: "Too many attempts"

# Hit Replica 2: 5 more attempts (separate counter)
for i in {1..5}; do curl /login -d "user=admin&pass=wrong"; done
# Replica 2: Allows another 5 attempts!

# Attacker gets 5 × N attempts (N = number of replicas)
```

### 2.5 Database Layer (`internal/database/database.go`)

**Current State:** ✅ **Database client is stateless** (HA-ready)

**HA Concerns:**
- **Connection pooling:** Not explicitly configured
- **Single PostgreSQL:** Database itself is single point of failure
- **Lock contention:** Multiple replicas updating `queue_tasks` table
- **Transaction isolation:** Potential race conditions

**Required Configuration:**
```go
// Connection pool settings for HA
config.MaxConns = 25              // Max connections per replica
config.MinConns = 5               // Min idle connections
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 5 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

**PostgreSQL HA Options:**
1. **Patroni** - Automatic failover with etcd/Consul
2. **Stolon** - PostgreSQL HA with Kubernetes integration
3. **CloudNativePG** - Kubernetes-native PostgreSQL operator
4. **Bitnami PostgreSQL HA Chart** - Primary-replica setup

---

## 3. Distributed Queue Architecture

### 3.1 Recommended Approach: Database-Backed Queue

**Why Database-Backed?**
- ✅ Leverages existing PostgreSQL infrastructure
- ✅ ACID transactions prevent duplicate processing
- ✅ No additional infrastructure (Redis, RabbitMQ)
- ✅ Tasks survive pod restarts
- ✅ Simple to implement and maintain

**Architecture:**

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Replica 1  │  │  Replica 2  │  │  Replica 3  │
│             │  │             │  │             │
│  Workers:   │  │  Workers:   │  │  Workers:   │
│  - Poll DB  │  │  - Poll DB  │  │  - Poll DB  │
│  - Lock row │  │  - Lock row │  │  - Lock row │
│  - Execute  │  │  - Execute  │  │  - Execute  │
│  - Update   │  │  - Update   │  │  - Update   │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┴────────────────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │       PostgreSQL              │
        │  ┌─────────────────────────┐  │
        │  │   queue_tasks           │  │
        │  ├─────────────────────────┤  │
        │  │ id | status | locked_by │  │
        │  │ 1  | pending| NULL      │  │ ← All replicas compete
        │  │ 2  | running| replica-1 │  │ ← Locked by Replica 1
        │  │ 3  | running| replica-2 │  │ ← Locked by Replica 2
        │  │ 4  | pending| NULL      │  │ ← Available
        │  └─────────────────────────┘  │
        └───────────────────────────────┘
```

### 3.2 Database Schema Changes

**Add columns to `queue_tasks` table:**

```sql
-- Migration: 010_add_distributed_queue_fields.sql
ALTER TABLE queue_tasks
ADD COLUMN locked_by VARCHAR(255),           -- Pod name that owns this task
ADD COLUMN locked_at TIMESTAMP,              -- When task was locked
ADD COLUMN lock_expires_at TIMESTAMP,        -- Lock expiration (for stale locks)
ADD COLUMN heartbeat_at TIMESTAMP,           -- Last heartbeat from worker
ADD COLUMN retry_count INTEGER DEFAULT 0,    -- Number of retry attempts
ADD COLUMN max_retries INTEGER DEFAULT 3,    -- Max retry attempts
ADD COLUMN priority INTEGER DEFAULT 5;       -- Task priority (1=highest)

-- Index for efficient polling
CREATE INDEX idx_queue_tasks_polling ON queue_tasks(status, priority DESC, enqueued_at ASC)
WHERE status = 'pending' AND (locked_by IS NULL OR lock_expires_at < NOW());

-- Index for stale lock cleanup
CREATE INDEX idx_queue_tasks_stale_locks ON queue_tasks(locked_by, lock_expires_at)
WHERE status = 'running' AND lock_expires_at < NOW();
```

### 3.3 Worker Implementation (Polling Pattern)

**New Queue Implementation:**

```go
// internal/queue/distributed_queue.go
type DistributedQueue struct {
    db              *database.Database
    workers         int
    executor        WorkflowExecutor
    podName         string            // Kubernetes pod name
    pollInterval    time.Duration     // How often to poll database
    lockDuration    time.Duration     // How long to hold lock
    ctx             context.Context
    cancel          context.CancelFunc
    wg              sync.WaitGroup
}

func NewDistributedQueue(workers int, executor WorkflowExecutor, db *database.Database) *DistributedQueue {
    ctx, cancel := context.WithCancel(context.Background())

    // Get pod name from environment (set by Kubernetes)
    podName := os.Getenv("HOSTNAME") // e.g., "innominatus-6d8f9b7c-abcd"
    if podName == "" {
        podName = fmt.Sprintf("server-%d", time.Now().Unix())
    }

    return &DistributedQueue{
        db:           db,
        workers:      workers,
        executor:     executor,
        podName:      podName,
        pollInterval: 1 * time.Second,  // Poll every second
        lockDuration: 5 * time.Minute,  // Hold lock for 5 minutes
        ctx:          ctx,
        cancel:       cancel,
    }
}

// Start worker pool that polls database
func (q *DistributedQueue) Start() {
    for i := 0; i < q.workers; i++ {
        q.wg.Add(1)
        go q.worker(i)
    }

    // Start stale lock cleanup goroutine
    q.wg.Add(1)
    go q.cleanupStaleLocks()
}

// Worker polls database for pending tasks
func (q *DistributedQueue) worker(id int) {
    defer q.wg.Done()

    ticker := time.NewTicker(q.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-q.ctx.Done():
            return
        case <-ticker.C:
            // Try to claim a task
            task, err := q.claimTask()
            if err != nil {
                continue // No task available or error
            }

            // Execute task with heartbeat
            q.executeWithHeartbeat(id, task)
        }
    }
}

// Claim a pending task using SELECT FOR UPDATE SKIP LOCKED
func (q *DistributedQueue) claimTask() (*WorkflowTask, error) {
    tx, err := q.db.DB().Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()

    // Find and lock next available task (atomic operation)
    query := `
        UPDATE queue_tasks
        SET status = 'running',
            locked_by = $1,
            locked_at = NOW(),
            lock_expires_at = NOW() + INTERVAL '5 minutes',
            started_at = NOW()
        WHERE id = (
            SELECT id FROM queue_tasks
            WHERE status = 'pending'
              AND (locked_by IS NULL OR lock_expires_at < NOW())
              AND retry_count < max_retries
            ORDER BY priority DESC, enqueued_at ASC
            FOR UPDATE SKIP LOCKED  -- Key: Skip rows locked by other transactions
            LIMIT 1
        )
        RETURNING task_id, app_name, workflow_name, workflow_spec, metadata
    `

    var task WorkflowTask
    err = tx.QueryRow(query, q.podName).Scan(
        &task.ID,
        &task.AppName,
        &task.WorkflowName,
        &task.WorkflowJSON,
        &task.MetadataJSON,
    )

    if err != nil {
        return nil, err // No task available
    }

    // Unmarshal JSON fields
    json.Unmarshal(task.WorkflowJSON, &task.Workflow)
    json.Unmarshal(task.MetadataJSON, &task.Metadata)

    if err := tx.Commit(); err != nil {
        return nil, err
    }

    return &task, nil
}

// Execute task and send heartbeats to prevent stale lock
func (q *DistributedQueue) executeWithHeartbeat(workerID int, task *WorkflowTask) {
    // Start heartbeat goroutine
    heartbeatCtx, heartbeatCancel := context.WithCancel(q.ctx)
    defer heartbeatCancel()

    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-heartbeatCtx.Done():
                return
            case <-ticker.C:
                // Update heartbeat and extend lock
                q.db.DB().Exec(`
                    UPDATE queue_tasks
                    SET heartbeat_at = NOW(),
                        lock_expires_at = NOW() + INTERVAL '5 minutes'
                    WHERE task_id = $1 AND locked_by = $2
                `, task.ID, q.podName)
            }
        }
    }()

    // Execute workflow
    err := q.executor.ExecuteWorkflowWithName(task.AppName, task.WorkflowName, task.Workflow)

    // Update task status
    if err != nil {
        q.handleTaskFailure(task, err)
    } else {
        q.handleTaskSuccess(task)
    }
}

// Handle task failure with retry logic
func (q *DistributedQueue) handleTaskFailure(task *WorkflowTask, taskErr error) {
    query := `
        UPDATE queue_tasks
        SET status = CASE
                WHEN retry_count + 1 >= max_retries THEN 'failed'
                ELSE 'pending'
            END,
            retry_count = retry_count + 1,
            error_message = $1,
            locked_by = NULL,
            locked_at = NULL,
            lock_expires_at = NULL,
            updated_at = NOW(),
            completed_at = CASE
                WHEN retry_count + 1 >= max_retries THEN NOW()
                ELSE NULL
            END
        WHERE task_id = $2 AND locked_by = $3
    `

    q.db.DB().Exec(query, taskErr.Error(), task.ID, q.podName)
}

// Handle task success
func (q *DistributedQueue) handleTaskSuccess(task *WorkflowTask) {
    query := `
        UPDATE queue_tasks
        SET status = 'completed',
            locked_by = NULL,
            locked_at = NULL,
            lock_expires_at = NULL,
            completed_at = NOW(),
            updated_at = NOW()
        WHERE task_id = $1 AND locked_by = $2
    `

    q.db.DB().Exec(query, task.ID, q.podName)
}

// Cleanup stale locks (tasks where worker crashed without updating)
func (q *DistributedQueue) cleanupStaleLocks() {
    defer q.wg.Done()

    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-q.ctx.Done():
            return
        case <-ticker.C:
            // Reset stale locks to pending
            query := `
                UPDATE queue_tasks
                SET status = 'pending',
                    locked_by = NULL,
                    locked_at = NULL,
                    lock_expires_at = NULL,
                    retry_count = retry_count + 1
                WHERE status = 'running'
                  AND lock_expires_at < NOW()
                  AND retry_count < max_retries
            `

            result, _ := q.db.DB().Exec(query)
            affected, _ := result.RowsAffected()

            if affected > 0 {
                q.logger.WarnWithFields("Cleaned up stale locks", map[string]interface{}{
                    "tasks_reset": affected,
                })
            }
        }
    }
}
```

### 3.4 Key HA Features

**1. SELECT FOR UPDATE SKIP LOCKED:**
- PostgreSQL-specific (also in MySQL 8.0+)
- **Atomic** task claiming - only **one** replica can claim a task
- `SKIP LOCKED` - if row is locked by another transaction, **skip it** and try next row
- Prevents duplicate task execution

**2. Lock Expiration:**
- Tasks have `lock_expires_at` timestamp
- If worker crashes, lock expires after 5 minutes
- Stale lock cleanup resets task to `pending`
- Prevents tasks from being stuck forever

**3. Heartbeat Mechanism:**
- Worker sends heartbeat every 30 seconds
- Updates `heartbeat_at` and extends `lock_expires_at`
- Proves worker is still alive
- If heartbeat stops → lock expires → task reclaimed

**4. Retry Logic:**
- Tasks track `retry_count` and `max_retries`
- Failed tasks automatically retry (up to 3 times)
- After max retries → marked as `failed`
- Transient failures (network, timeout) handled automatically

**5. Priority Queuing:**
- Tasks have `priority` field (1-10, 1=highest)
- Workers process high-priority tasks first
- `ORDER BY priority DESC, enqueued_at ASC`

---

## 4. Database HA Strategy

### 4.1 Current Database Deployment

**Helm Chart (values.yaml:154-173):**
```yaml
postgresql:
  enabled: true
  auth:
    username: innominatus
    password: "changeme"
    database: idp_orchestrator
  primary:
    persistence:
      enabled: true
      size: 8Gi
```

**Problem:** Single PostgreSQL pod - **SPOF** (Single Point of Failure)

### 4.2 Recommended: PostgreSQL HA with Patroni

**Why Patroni?**
- ✅ Automatic failover (< 30 seconds)
- ✅ Kubernetes-native (uses etcd/Consul for consensus)
- ✅ Read replicas for load distribution
- ✅ Backup/restore integration
- ✅ Battle-tested in production

**Architecture:**

```
┌────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                  │
│                                                        │
│  ┌──────────────┐   ┌──────────────┐   ┌───────────┐ │
│  │  PostgreSQL  │   │  PostgreSQL  │   │  etcd/    │ │
│  │   Primary    │──▶│   Replica 1  │   │  Consul   │ │
│  │ (Read/Write) │   │  (Read-Only) │   │ (Leader   │ │
│  └──────┬───────┘   └──────────────┘   │ Election) │ │
│         │                               └───────────┘ │
│         │           ┌──────────────┐                  │
│         └──────────▶│  PostgreSQL  │                  │
│                     │   Replica 2  │                  │
│                     │  (Read-Only) │                  │
│                     └──────────────┘                  │
│                                                        │
│  ┌─────────────────────────────────────────────────┐  │
│  │  Patroni Leader Election                        │  │
│  │  - Monitors primary health                      │  │
│  │  - Promotes replica on failure                  │  │
│  │  - Updates Service endpoints                    │  │
│  └─────────────────────────────────────────────────┘  │
│                                                        │
│  ┌──────────────┐          ┌──────────────┐          │
│  │  Service:    │          │  Service:    │          │
│  │  postgres    │          │  postgres-ro │          │
│  │  (Primary)   │          │  (Replicas)  │          │
│  └──────────────┘          └──────────────┘          │
└────────────────────────────────────────────────────────┘
                  ▲                      ▲
                  │                      │
         ┌────────┴────────┐    ┌────────┴────────┐
         │  innominatus    │    │  innominatus    │
         │  (Write Ops)    │    │  (Read Ops)     │
         └─────────────────┘    └─────────────────┘
```

**Helm Chart Configuration:**

```yaml
# Use Bitnami PostgreSQL HA chart
postgresql:
  enabled: false  # Disable single-node PostgreSQL

postgresql-ha:
  enabled: true
  postgresql:
    replicaCount: 3
    username: innominatus
    password: "changeme"
    database: idp_orchestrator
    repmgrUsername: repmgr
    repmgrPassword: "repmgrpass"

  persistence:
    enabled: true
    size: 10Gi

  pgpool:
    replicaCount: 3
    adminUsername: admin
    adminPassword: "adminpass"

  # Enable automatic failover
  metrics:
    enabled: true

  # Backup configuration
  backup:
    enabled: true
    schedule: "0 2 * * *"  # Daily at 2am
    retention: 7  # Keep 7 days of backups
```

**Connection String Updates:**

```go
// internal/database/database.go

func NewDatabase() (*Database, error) {
    // Read-write connection (primary)
    writeHost := os.Getenv("DB_WRITE_HOST")  // postgres-postgresql-ha-pgpool

    // Read-only connection (replicas)
    readHost := os.Getenv("DB_READ_HOST")    // postgres-postgresql-ha-pgpool-replicas

    // For writes (INSERT, UPDATE, DELETE)
    writeConnString := fmt.Sprintf(
        "host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        writeHost, user, password, dbname,
    )

    // For reads (SELECT) - can use replicas
    readConnString := fmt.Sprintf(
        "host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
        readHost, user, password, dbname,
    )

    // Create separate connection pools
    writePool, _ := pgxpool.Connect(context.Background(), writeConnString)
    readPool, _ := pgxpool.Connect(context.Background(), readConnString)

    return &Database{
        writePool: writePool,
        readPool:  readPool,
    }, nil
}
```

**Benefits:**
- **Automatic failover:** Primary failure → replica promoted in < 30s
- **Read scaling:** Offload SELECT queries to replicas
- **Zero downtime maintenance:** Rolling updates without service interruption
- **Data durability:** Replication ensures no data loss

---

## 5. Shared Storage Strategy

### 5.1 Problem: Local Workspaces

**Current Implementation:**
```go
// internal/workflow/executor.go
workspaceDir := fmt.Sprintf("./workspaces/%s-%s", appName, environment)
```

**Each replica has separate workspace:**
```
Replica 1: /workspaces/my-app-production/
Replica 2: /workspaces/my-app-production/  (different PVC)
Replica 3: /workspaces/my-app-production/  (different PVC)
```

**Problem:**
- Terraform state on Replica 1 → not visible to Replica 2
- Task starts on Replica 1, crashes → retried on Replica 2 → **state mismatch**

### 5.2 Solution: Shared Storage

**Option 1: ReadWriteMany (RWX) PersistentVolumeClaim**

**Update Helm Chart:**
```yaml
# values.yaml
persistence:
  enabled: true
  storageClass: "nfs"  # or "azurefile", "efs-sc"
  accessMode: ReadWriteMany  # ✅ Changed from ReadWriteOnce
  size: 50Gi
```

**Supported Storage Classes:**
- **NFS:** Most common, works everywhere
- **Azure Files:** Azure Kubernetes Service (AKS)
- **AWS EFS:** Amazon Elastic Kubernetes Service (EKS)
- **Google Filestore:** Google Kubernetes Engine (GKE)
- **CephFS:** Self-hosted Ceph cluster

**Deployment Template Update:**
```yaml
# charts/innominatus/templates/deployment.yaml
spec:
  replicas: {{ .Values.replicaCount }}  # Can now be 3+
  template:
    spec:
      containers:
      - name: innominatus
        volumeMounts:
        - name: workspaces
          mountPath: /app/workspaces  # ✅ Shared across all pods
      volumes:
      - name: workspaces
        persistentVolumeClaim:
          claimName: {{ include "innominatus.fullname" . }}-workspaces
```

**Option 2: Object Storage (S3-Compatible)**

**Why S3?**
- ✅ Cloud-native, highly available
- ✅ No need for RWX storage class
- ✅ Scales infinitely
- ✅ Works with Minio (self-hosted), AWS S3, GCS, Azure Blob

**Implementation:**
```go
// internal/storage/storage.go
type WorkspaceStorage interface {
    CreateWorkspace(appName, environment string) (string, error)
    WriteFile(workspaceID, filename string, data []byte) error
    ReadFile(workspaceID, filename string) ([]byte, error)
    DeleteWorkspace(workspaceID string) error
}

// S3-backed implementation
type S3Storage struct {
    client *s3.Client
    bucket string
}

func (s *S3Storage) CreateWorkspace(appName, environment string) (string, error) {
    workspaceID := fmt.Sprintf("%s-%s-%d", appName, environment, time.Now().Unix())
    prefix := fmt.Sprintf("workspaces/%s/", workspaceID)

    // Create marker file (S3 doesn't have "directories")
    _, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(prefix + ".workspace"),
        Body:   strings.NewReader(""),
    })

    return workspaceID, err
}

func (s *S3Storage) WriteFile(workspaceID, filename string, data []byte) error {
    key := fmt.Sprintf("workspaces/%s/%s", workspaceID, filename)

    _, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   bytes.NewReader(data),
    })

    return err
}
```

**Helm Chart S3 Configuration:**
```yaml
# values.yaml
storage:
  type: s3  # or "filesystem"
  s3:
    endpoint: "http://minio.minio-system.svc.cluster.local:9000"
    bucket: "innominatus-workspaces"
    accessKey: "minioadmin"
    secretKey: "minioadmin"
    region: "us-east-1"
```

**Recommendation:**
- **Development/Testing:** Use NFS (simpler)
- **Production:** Use S3/EFS/Azure Files (more reliable)

---

## 6. Session & Authentication

### 6.1 Current State

**✅ What Works:**
- Sessions stored in PostgreSQL → HA-ready
- API keys stored in PostgreSQL → HA-ready
- OIDC tokens validated against Keycloak → stateless

**❌ What Doesn't Work:**
```go
// internal/server/handlers.go:129-130
loginAttempts     map[string][]time.Time  // ❌ Per-replica
loginMutex        sync.Mutex
```

**Problem:**
- Rate limiting is **per-replica**
- Attacker can distribute requests across replicas
- Each replica independently tracks attempts

### 6.2 Solution: Redis-Backed Rate Limiting

**Why Redis?**
- ✅ Shared state across replicas
- ✅ TTL support (automatic cleanup)
- ✅ Atomic operations (INCR, EXPIRE)
- ✅ High performance (< 1ms latency)

**Implementation:**

```go
// internal/auth/rate_limiter.go
type RateLimiter struct {
    redis        *redis.Client
    maxAttempts  int
    windowMinutes int
}

func NewRateLimiter(redisAddr string) *RateLimiter {
    return &RateLimiter{
        redis: redis.NewClient(&redis.Options{
            Addr: redisAddr,
        }),
        maxAttempts:  5,
        windowMinutes: 15,
    }
}

func (rl *RateLimiter) CheckLoginAttempts(username string) (bool, error) {
    key := fmt.Sprintf("login_attempts:%s", username)

    // Increment attempt counter
    count, err := rl.redis.Incr(context.TODO(), key).Result()
    if err != nil {
        return false, err
    }

    // Set expiration on first attempt
    if count == 1 {
        rl.redis.Expire(context.TODO(), key, time.Duration(rl.windowMinutes)*time.Minute)
    }

    // Check if exceeded
    if count > int64(rl.maxAttempts) {
        return false, nil  // Too many attempts
    }

    return true, nil  // Allowed
}

func (rl *RateLimiter) ResetAttempts(username string) error {
    key := fmt.Sprintf("login_attempts:%s", username)
    return rl.redis.Del(context.TODO(), key).Err()
}
```

**Helm Chart Redis Configuration:**

```yaml
# values.yaml
redis:
  enabled: true
  architecture: standalone  # or "replication" for HA
  auth:
    enabled: true
    password: "redis-password"
  master:
    persistence:
      enabled: false  # Rate limiting data is ephemeral
```

**Alternative: Database-Backed Rate Limiting** (if avoiding Redis)

```sql
-- Create rate limiting table
CREATE TABLE login_attempts (
    username VARCHAR(255) PRIMARY KEY,
    attempt_count INTEGER DEFAULT 0,
    window_start TIMESTAMP DEFAULT NOW(),
    locked_until TIMESTAMP
);

-- Cleanup old entries
CREATE INDEX idx_login_attempts_cleanup ON login_attempts(window_start)
WHERE window_start < NOW() - INTERVAL '15 minutes';
```

---

## 7. Metrics & Observability

### 7.1 Current Metrics (In-Memory)

**Problem:**
```go
type MetricsCollector struct {
    tasksEnqueued     int64  // ❌ Per-replica counter
    tasksCompleted    int64  // ❌ Per-replica counter
    tasksFailed       int64  // ❌ Per-replica counter
}
```

**Impact:**
- `/api/queue/stats` returns **different values** per replica
- Prometheus scrapes each replica separately
- No global queue metrics

### 7.2 Solution: Database-Backed Metrics

**Aggregate from `queue_tasks` table:**

```go
// internal/queue/distributed_queue.go
func (q *DistributedQueue) GetQueueStats() map[string]interface{} {
    // Query database for global metrics
    query := `
        SELECT
            COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
            COUNT(*) FILTER (WHERE status = 'running') as running_count,
            COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
            COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
            COUNT(DISTINCT locked_by) FILTER (WHERE status = 'running') as active_workers,
            AVG(EXTRACT(EPOCH FROM (started_at - enqueued_at)) * 1000)
                FILTER (WHERE started_at IS NOT NULL) as avg_queue_time_ms,
            AVG(EXTRACT(EPOCH FROM (completed_at - started_at)) * 1000)
                FILTER (WHERE completed_at IS NOT NULL) as avg_execution_time_ms
        FROM queue_tasks
        WHERE created_at > NOW() - INTERVAL '24 hours'
    `

    var stats struct {
        Pending         int64
        Running         int64
        Completed       int64
        Failed          int64
        ActiveWorkers   int64
        AvgQueueTimeMs  float64
        AvgExecTimeMs   float64
    }

    q.db.DB().QueryRow(query).Scan(
        &stats.Pending,
        &stats.Running,
        &stats.Completed,
        &stats.Failed,
        &stats.ActiveWorkers,
        &stats.AvgQueueTimeMs,
        &stats.AvgExecTimeMs,
    )

    return map[string]interface{}{
        "queue_size":              stats.Pending,
        "active_tasks":            stats.Running,
        "tasks_completed":         stats.Completed,
        "tasks_failed":            stats.Failed,
        "active_workers":          stats.ActiveWorkers,
        "workers":                 q.workers * q.getReplicaCount(),
        "avg_queue_time_ms":       int64(stats.AvgQueueTimeMs),
        "avg_execution_time_ms":   int64(stats.AvgExecTimeMs),
    }
}

func (q *DistributedQueue) getReplicaCount() int {
    // Query distinct locked_by values to count active replicas
    query := `
        SELECT COUNT(DISTINCT locked_by)
        FROM queue_tasks
        WHERE heartbeat_at > NOW() - INTERVAL '2 minutes'
    `

    var count int
    q.db.DB().QueryRow(query).Scan(&count)
    return count
}
```

### 7.3 Prometheus Metrics (Federated)

**Expose per-replica metrics for Prometheus:**

```go
// internal/metrics/metrics.go
var (
    queueTasksTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "innominatus_queue_tasks_total",
            Help: "Total number of queue tasks by status",
        },
        []string{"status", "replica"},
    )

    queueDurationSeconds = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "innominatus_queue_duration_seconds",
            Help: "Queue task duration",
            Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
        },
        []string{"status", "replica"},
    )
)

func RecordTaskCompletion(status string, duration time.Duration) {
    replica := os.Getenv("HOSTNAME")
    queueTasksTotal.WithLabelValues(status, replica).Inc()
    queueDurationSeconds.WithLabelValues(status, replica).Observe(duration.Seconds())
}
```

**Prometheus will aggregate across replicas:**
```promql
# Total completed tasks (all replicas)
sum(innominatus_queue_tasks_total{status="completed"})

# Average task duration (all replicas)
avg(innominatus_queue_duration_seconds)

# Per-replica breakdown
innominatus_queue_tasks_total{status="completed", replica=~"innominatus-.*"}
```

---

## 8. Implementation Roadmap

### Phase 1: Database-Backed Queue (2-3 weeks)

**Week 1: Schema & Polling**
- ✅ Create migration `010_add_distributed_queue_fields.sql`
- ✅ Implement `DistributedQueue` with polling pattern
- ✅ Add `SELECT FOR UPDATE SKIP LOCKED` query
- ✅ Test with 3 replicas locally (Docker Compose)

**Week 2: Heartbeat & Retry**
- ✅ Implement heartbeat mechanism
- ✅ Add stale lock cleanup
- ✅ Implement retry logic with exponential backoff
- ✅ Add priority queuing

**Week 3: Integration & Testing**
- ✅ Replace in-memory queue with distributed queue
- ✅ Update server initialization
- ✅ Add integration tests (3 replicas + PostgreSQL)
- ✅ Load testing (100 concurrent tasks)

**Deliverables:**
- `internal/queue/distributed_queue.go` (new)
- `internal/database/migrations/010_add_distributed_queue_fields.sql` (new)
- Updated `internal/server/handlers.go`
- Test suite with 3-replica scenarios

---

### Phase 2: Shared Storage (1-2 weeks)

**Week 4: NFS/RWX Setup**
- ✅ Update Helm chart for ReadWriteMany PVC
- ✅ Test with NFS storage class
- ✅ Verify workspace sharing across replicas
- ✅ Update documentation

**Alternative: S3 Integration (2 weeks)**
- ✅ Implement `WorkspaceStorage` interface
- ✅ Add S3-backed storage implementation
- ✅ Update workflow executor to use abstraction
- ✅ Test with Minio/AWS S3

**Deliverables:**
- Updated `charts/innominatus/values.yaml`
- `internal/storage/storage.go` (if using S3)
- Updated workspace management in workflow executor

---

### Phase 3: Database HA (1-2 weeks)

**Week 5-6: PostgreSQL HA**
- ✅ Deploy Bitnami PostgreSQL HA chart
- ✅ Configure read/write connection pools
- ✅ Update database layer for read replicas
- ✅ Test failover scenarios
- ✅ Configure automated backups

**Deliverables:**
- Updated `charts/innominatus/values.yaml`
- Read/write connection pool in `internal/database/database.go`
- Failover testing documentation

---

### Phase 4: Rate Limiting & Metrics (1 week)

**Week 7: Distributed Rate Limiting**
- ✅ Deploy Redis (or use database-backed)
- ✅ Implement distributed rate limiter
- ✅ Update login handler
- ✅ Test rate limiting across replicas

**Deliverables:**
- `internal/auth/rate_limiter.go` (new)
- Updated Helm chart with Redis dependency

---

### Phase 5: Production Deployment (1 week)

**Week 8: Deployment & Validation**
- ✅ Deploy 3-replica configuration to staging
- ✅ Run chaos engineering tests (kill pods)
- ✅ Monitor metrics and logs
- ✅ Performance benchmarking
- ✅ Update production deployment guide

**Deliverables:**
- Production Helm values file
- Chaos testing results
- Updated `docs/platform-team-guide/kubernetes-deployment.md`

---

**Total Timeline:** 6-8 weeks
**Estimated Effort:** 1-2 engineers

---

## 9. Trade-offs & Considerations

### 9.1 Database-Backed Queue vs. Redis Queue

| Factor | Database-Backed | Redis Queue |
|--------|-----------------|-------------|
| **Infrastructure** | Uses existing PostgreSQL | Requires Redis cluster |
| **Performance** | ~100-500 tasks/sec | ~10,000 tasks/sec |
| **Complexity** | Low (SQL queries) | Medium (Redis data structures) |
| **Persistence** | ✅ ACID guarantees | ⚠️ Optional (RDB/AOF) |
| **Operations** | Same as database | Separate monitoring/backups |
| **Cost** | Included | Additional infrastructure |
| **HA** | Follows database HA | Requires Redis Sentinel/Cluster |

**Recommendation:**
- **Start with database-backed** (simpler, leverages existing infrastructure)
- **Migrate to Redis** if throughput becomes bottleneck (> 500 tasks/sec)

### 9.2 Polling Interval Trade-off

**Shorter interval (e.g., 100ms):**
- ✅ Lower latency (tasks start faster)
- ❌ Higher database load (more SELECT queries)

**Longer interval (e.g., 5s):**
- ✅ Lower database load
- ❌ Higher latency (tasks wait longer)

**Recommendation:** Start with **1 second**, adjust based on workload.

### 9.3 Shared Storage Performance

**NFS Performance Concerns:**
- Network latency for file I/O
- Not ideal for high-frequency writes (e.g., logs)
- **Solution:** Use local cache + periodic sync to NFS

**S3 Performance:**
- Higher latency than filesystem (50-100ms per request)
- Not suitable for Terraform state locking (use Terraform backend)
- **Solution:** Use S3 for artifacts, PostgreSQL for state locks

### 9.4 Cost Implications

**3-Replica Deployment:**
- 3× compute resources (CPU/memory)
- Shared storage costs (NFS: $0.30/GB/month, S3: $0.023/GB/month)
- PostgreSQL HA: 3× database storage
- Redis (if used): Additional memory/storage

**Estimated Cost Increase:** 2.5-3× over single-replica

---

## 10. Example 3-Replica Deployment

### 10.1 Helm Values (Production)

```yaml
# values-production.yaml

# 3 replicas for HA
replicaCount: 3

image:
  repository: ghcr.io/philipsahli/innominatus
  tag: "v0.2.0"

# Resource limits per replica
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 250m
    memory: 256Mi

# Horizontal Pod Autoscaler
autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

# Pod Disruption Budget (ensure at least 2 replicas during updates)
podDisruptionBudget:
  enabled: true
  minAvailable: 2

# Affinity: Spread pods across nodes
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app.kubernetes.io/name: innominatus
        topologyKey: kubernetes.io/hostname

# Disable bundled PostgreSQL (use external)
postgresql:
  enabled: false

# Use Bitnami PostgreSQL HA
postgresql-ha:
  enabled: true
  postgresql:
    replicaCount: 3
    username: innominatus
    password: "changeme-secure-password"
    database: idp_orchestrator
    postgresPassword: "postgres-secure-password"
  persistence:
    enabled: true
    size: 20Gi
    storageClass: "standard"
  pgpool:
    replicaCount: 3

# Shared workspace storage (ReadWriteMany)
persistence:
  enabled: true
  storageClass: "nfs-client"  # or "azurefile", "efs-sc"
  accessMode: ReadWriteMany
  size: 100Gi

# Redis for rate limiting
redis:
  enabled: true
  architecture: replication
  auth:
    enabled: true
    password: "redis-secure-password"
  master:
    persistence:
      enabled: false
  replica:
    replicaCount: 2

# Environment variables
env:
  RUNNING_IN_KUBERNETES: "true"
  PORT: "8081"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"

  # Database connection (pgpool)
  DB_HOST: "innominatus-postgresql-ha-pgpool"
  DB_PORT: "5432"
  DB_USER: "innominatus"
  DB_NAME: "idp_orchestrator"

  # Redis connection
  REDIS_ENABLED: "true"
  REDIS_HOST: "innominatus-redis-master"
  REDIS_PORT: "6379"

  # Queue configuration
  QUEUE_WORKERS: "5"  # 5 workers per replica = 15 total
  QUEUE_POLL_INTERVAL: "1s"
  QUEUE_LOCK_DURATION: "5m"

# Ingress for external access
ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
  hosts:
    - host: innominatus.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: innominatus-tls
      hosts:
        - innominatus.example.com

# Service Monitor for Prometheus
serviceMonitor:
  enabled: true
  interval: 30s
  scrapeTimeout: 10s
```

### 10.2 Deployment Commands

```bash
# Add Bitnami Helm repository
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Create namespace
kubectl create namespace innominatus-system

# Install innominatus with 3 replicas + HA
helm install innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --values values-production.yaml \
  --wait \
  --timeout 10m

# Verify deployment
kubectl get pods -n innominatus-system
# Expected output:
# innominatus-6d8f9b7c-abcd   1/1  Running  0  5m
# innominatus-6d8f9b7c-efgh   1/1  Running  0  5m
# innominatus-6d8f9b7c-ijkl   1/1  Running  0  5m

# Check PostgreSQL HA
kubectl get pods -n innominatus-system -l app.kubernetes.io/name=postgresql-ha
# Expected output:
# innominatus-postgresql-ha-postgresql-0  1/1  Running  0  5m
# innominatus-postgresql-ha-postgresql-1  1/1  Running  0  5m
# innominatus-postgresql-ha-postgresql-2  1/1  Running  0  5m

# Test distributed queue
kubectl exec -it innominatus-6d8f9b7c-abcd -n innominatus-system -- \
  curl localhost:8081/api/queue/stats
# Should show aggregated stats across all replicas
```

### 10.3 Chaos Testing

**Kill a pod and verify recovery:**

```bash
# Delete one replica
kubectl delete pod innominatus-6d8f9b7c-abcd -n innominatus-system

# Kubernetes automatically creates new pod
kubectl get pods -n innominatus-system -l app.kubernetes.io/name=innominatus
# innominatus-6d8f9b7c-mnop   1/1  Running  0  10s  ← New pod

# Verify queue still processes tasks
kubectl logs -n innominatus-system innominatus-6d8f9b7c-mnop | grep "Worker started"
# Worker 0 started
# Worker 1 started
# ...

# Check database for stale locks cleaned up
kubectl exec -it innominatus-postgresql-ha-postgresql-0 -n innominatus-system -- \
  psql -U innominatus -d idp_orchestrator -c \
  "SELECT COUNT(*) FROM queue_tasks WHERE status='running' AND lock_expires_at < NOW();"
# count: 0 (no stale locks)
```

### 10.4 Rolling Update

```bash
# Update image tag
helm upgrade innominatus ./charts/innominatus \
  --namespace innominatus-system \
  --values values-production.yaml \
  --set image.tag=v0.3.0 \
  --wait

# Kubernetes performs rolling update:
# 1. Creates new pod (v0.3.0)
# 2. Waits for readiness
# 3. Terminates old pod (v0.2.0)
# 4. Repeats for all replicas
# Result: Zero downtime
```

---

## Conclusion

innominatus **cannot currently run in HA mode** due to in-memory state dependencies. To support 3+ replicas:

### Critical Changes Required

1. **Distributed Queue** (database-backed with polling)
2. **Shared Storage** (RWX PVC or S3)
3. **Database HA** (PostgreSQL Primary-Replica)
4. **Distributed Rate Limiting** (Redis or database)

### Implementation Priority

1. 🔴 **Phase 1:** Distributed Queue (2-3 weeks) - **BLOCKING**
2. 🔴 **Phase 2:** Shared Storage (1-2 weeks) - **BLOCKING**
3. 🟡 **Phase 3:** Database HA (1-2 weeks) - **HIGH PRIORITY**
4. 🟡 **Phase 4:** Rate Limiting (1 week) - **MEDIUM PRIORITY**

### Estimated Effort

- **Total Timeline:** 6-8 weeks
- **Engineers:** 1-2 full-time
- **Cost Increase:** 2.5-3× infrastructure costs

### Recommendation

**Start with Phase 1 (Distributed Queue)** - this is the most critical blocker. The database-backed queue approach is:
- ✅ Simpler than Redis/RabbitMQ
- ✅ Leverages existing PostgreSQL
- ✅ Production-ready with proper testing
- ✅ Can migrate to Redis later if needed

---

**Document Version:** 1.0
**Next Review:** After Phase 1 implementation
**Contact:** Platform Team
