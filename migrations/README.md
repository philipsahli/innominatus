# Database Migrations

## Overview

innominatus uses PostgreSQL for workflow persistence, audit trails, and API key management. Database migrations are automatically executed on server startup to ensure the schema is up-to-date.

## Migration System

**Location:** `migrations/` directory

**Format:** SQL files with numeric prefixes (e.g., `001_create_graph_tables.sql`)

**Execution:**
- Migrations run automatically when the server starts
- Executed in numerical order
- Idempotent: Safe to run multiple times (uses `IF NOT EXISTS`)
- Tracked in `schema_migrations` table

**Server Logs:**
```
Running migration: 001_create_graph_tables.sql
Successfully executed migration: 001_create_graph_tables.sql
...
Successfully executed 5 migration(s)
```

---

## Migration Files

### 001_create_graph_tables.sql
**Purpose:** Create tables for resource graph visualization

**Tables:**
- `graph_nodes` - Resource nodes in the dependency graph
- `graph_edges` - Relationships between resources
- `graph_snapshots` - Historical graph snapshots

### 002_create_application_tables.sql
**Purpose:** Create tables for application and workflow tracking

**Tables:**
- `applications` - Deployed applications
- `workflow_executions` - Workflow execution history
- `workflow_step_executions` - Individual step execution details
- `resource_instances` - Provisioned resource instances

### 003_create_sessions_table.sql
**Purpose:** Create session management table

**Tables:**
- `sessions` - User sessions (cookie-based and OIDC)

**Columns:**
- `id` - Session ID (UUID)
- `username` - User identifier
- `created_at` - Session creation timestamp
- `expires_at` - Session expiration timestamp

### 004_rename_graph_tables.sql
**Purpose:** Rename graph tables for consistency

**Changes:**
- `graph_nodes` → `resource_nodes`
- `graph_edges` → `resource_edges`
- `graph_snapshots` → `resource_graphs`

### 005_create_api_keys_table.sql ⭐ **OIDC Integration**
**Purpose:** Create API keys table for OIDC-authenticated users

**Table:** `user_api_keys`

**Schema:**
```sql
CREATE TABLE IF NOT EXISTS user_api_keys (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    key_hash VARCHAR(64) NOT NULL UNIQUE,
    key_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    CONSTRAINT unique_username_keyname UNIQUE(username, key_name)
);
```

**Indexes:**
- `idx_user_api_keys_username` - Fast lookup by username
- `idx_user_api_keys_hash` - Fast authentication by key hash
- `idx_user_api_keys_expires` - Efficient expiry cleanup

**Security:**
- API keys stored as SHA-256 hashes (never plaintext)
- Unique constraint prevents duplicate key names per user
- `key_hash` is globally unique

**Usage:**
- OIDC users: API keys stored in this table
- Local users: API keys stored in `users.yaml` file
- System automatically detects user type and uses correct storage

**See:** [OIDC Authentication Guide](../docs/OIDC_AUTHENTICATION.md) for complete documentation.

---

## Dual-Source API Key Architecture

innominatus supports two types of users with different API key storage:

| User Type | Source | API Key Storage | Detection |
|-----------|--------|-----------------|-----------|
| **Local Users** | `users.yaml` file | File-based (in YAML) | User exists in users.yaml |
| **OIDC Users** | Keycloak/OIDC | Database (`user_api_keys` table) | User NOT in users.yaml |

### Automatic User Type Detection

When a user tries to generate an API key:

1. Check if user exists in `users.yaml`
2. **If found** → Local user → Store API key in `users.yaml`
3. **If not found** → OIDC user → Store API key in database

### API Key Authentication Flow

When authenticating with an API key:

1. Try to authenticate as local user (check `users.yaml`)
2. If failed, try to authenticate as OIDC user (check database)
3. If database lookup succeeds:
   - Update `last_used_at` timestamp
   - Return user object with username, team, role

### Example: OIDC User API Key

```sql
-- OIDC user "demo-user" generates an API key named "cli-key"
INSERT INTO user_api_keys (
    username,
    key_hash,
    key_name,
    expires_at
) VALUES (
    'demo-user',
    'a1b2c3d4...sha256-hash',  -- SHA-256 hash of the actual key
    'cli-key',
    NOW() + INTERVAL '90 days'
);

-- API key authentication
SELECT username, last_used_at, expires_at
FROM user_api_keys
WHERE key_hash = 'a1b2c3d4...sha256-hash'
  AND expires_at > NOW();

-- Update last used timestamp
UPDATE user_api_keys
SET last_used_at = NOW()
WHERE key_hash = 'a1b2c3d4...sha256-hash';
```

---

## Database Configuration

### Environment Variables

```bash
# Required for database persistence
export DB_HOST="localhost"           # Default: localhost
export DB_PORT="5432"                # Default: 5432
export DB_USER="postgres"            # Default: postgres
export DB_PASSWORD="your-password"   # No default (required)
export DB_NAME="idp_orchestrator"    # Default: idp_orchestrator
export DB_SSLMODE="disable"          # Default: disable (use "require" in production)
```

### Database Initialization

```bash
# Create database (run once)
createdb -h localhost -U postgres idp_orchestrator

# Grant permissions
psql -h localhost -U postgres -c \
  "GRANT ALL PRIVILEGES ON DATABASE idp_orchestrator TO postgres;"
```

### Production Setup

```bash
# Use SSL in production
export DB_SSLMODE="require"

# Use dedicated service account
export DB_USER="orchestrator_service"
export DB_PASSWORD=$(cat /secrets/db-password)

# Connection pooling (configured in server code)
# - Max open connections: 25
# - Max idle connections: 10
# - Connection max lifetime: 15m
```

---

## Manual Migration Management

Migrations run automatically on startup, but you can also run them manually:

### View Migration Status

```sql
-- Connect to database
psql -h localhost -U postgres -d idp_orchestrator

-- Check schema_migrations table
SELECT version, applied_at FROM schema_migrations ORDER BY version;

-- Example output:
-- version | applied_at
-- --------|---------------------------
-- 001     | 2025-10-06 10:00:00+00
-- 002     | 2025-10-06 10:00:00+00
-- 003     | 2025-10-06 10:00:01+00
-- 004     | 2025-10-06 10:00:01+00
-- 005     | 2025-10-06 10:00:01+00
```

### Run Specific Migration

```bash
# Run migration manually
psql -h localhost -U postgres -d idp_orchestrator \
  -f migrations/005_create_api_keys_table.sql
```

### Rollback Migration (Manual)

```sql
-- Rollback API keys table (if needed)
DROP TABLE IF EXISTS user_api_keys;
DELETE FROM schema_migrations WHERE version = '005';
```

⚠️ **Warning:** Rolling back migrations can cause data loss. Only do this in development environments.

---

## API Key Management Queries

### List All API Keys for a User

```sql
SELECT
    key_name,
    created_at,
    last_used_at,
    expires_at,
    CASE
        WHEN expires_at < NOW() THEN 'expired'
        WHEN last_used_at IS NULL THEN 'never_used'
        ELSE 'active'
    END AS status
FROM user_api_keys
WHERE username = 'demo-user'
ORDER BY created_at DESC;
```

### Find Expired API Keys

```sql
SELECT username, key_name, expires_at
FROM user_api_keys
WHERE expires_at < NOW()
ORDER BY expires_at DESC;
```

### Cleanup Old Expired Keys

```sql
-- Remove API keys expired more than 30 days ago
DELETE FROM user_api_keys
WHERE expires_at < NOW() - INTERVAL '30 days';
```

### API Key Usage Statistics

```sql
-- Most active API keys
SELECT
    username,
    key_name,
    COUNT(*) as usage_count,
    MAX(last_used_at) as last_used
FROM user_api_keys
WHERE last_used_at IS NOT NULL
GROUP BY username, key_name
ORDER BY usage_count DESC
LIMIT 10;
```

---

## Troubleshooting

### Migration Failed

**Symptom:** Server fails to start with migration error

**Solution:**
```bash
# Check server logs
# Look for: "Failed to run migration: XXX"

# Connect to database and check schema
psql -h localhost -U postgres -d idp_orchestrator

# Verify table exists
\dt

# Check migration history
SELECT * FROM schema_migrations;

# Manual fix if needed (CAREFUL!)
# Run the failed migration manually
psql -h localhost -U postgres -d idp_orchestrator \
  -f migrations/XXX_failed_migration.sql
```

### API Keys Not Working

**Symptom:** "Invalid API key" error

**Solution:**
```bash
# Verify table exists
psql -h localhost -U postgres -d idp_orchestrator -c "\d user_api_keys"

# Check if user has API keys
psql -h localhost -U postgres -d idp_orchestrator -c \
  "SELECT * FROM user_api_keys WHERE username = 'demo-user';"

# Verify key hash
# (API key should be SHA-256 hashed, 64 hex characters)
echo -n "your-api-key" | sha256sum
```

### Database Connection Failed

**Symptom:** "Database not connected" error

**Solution:**
```bash
# Check environment variables
echo $DB_HOST
echo $DB_NAME
echo $DB_USER

# Test database connection
psql -h $DB_HOST -U $DB_USER -d $DB_NAME -c "SELECT 1;"

# Check server can reach database
# Look for: "Database connected successfully"
```

---

## Best Practices

### API Key Management

1. **Set Reasonable Expiry:** 30-90 days recommended
2. **Use Descriptive Names:** "ci-pipeline", "local-dev", "prod-cli"
3. **Rotate Keys Regularly:** Generate new key before old one expires
4. **Revoke Unused Keys:** Remove keys that haven't been used in 90+ days
5. **Monitor Usage:** Track `last_used_at` timestamps

### Database Maintenance

1. **Regular Backups:**
   ```bash
   pg_dump -h $DB_HOST -U $DB_USER -d idp_orchestrator \
     -F c -f backup-$(date +%Y%m%d).dump
   ```

2. **Cleanup Expired Keys (weekly cron job):**
   ```sql
   DELETE FROM user_api_keys
   WHERE expires_at < NOW() - INTERVAL '30 days';
   ```

3. **Monitor Table Size:**
   ```sql
   SELECT pg_size_pretty(pg_total_relation_size('user_api_keys'));
   ```

4. **Vacuum Regularly:**
   ```sql
   VACUUM ANALYZE user_api_keys;
   ```

---

## References

- [OIDC Authentication Guide](../docs/OIDC_AUTHENTICATION.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Database Migration Best Practices](https://www.postgresql.org/docs/current/ddl-schemas.html)

---

**Last Updated:** 2025-10-06
**Migration Version:** 005
