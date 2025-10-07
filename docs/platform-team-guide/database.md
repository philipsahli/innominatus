# Database Configuration

Configure PostgreSQL for production use with innominatus.

---

## Requirements

- PostgreSQL 13+
- Database user with `CREATE DATABASE` permissions
- Connection pooling recommended

---

## Database Setup

### 1. Create Database

```bash
createdb -h postgres.production.internal -U postgres idp_orchestrator
```

### 2. Create Service User

```sql
CREATE USER orchestrator_service WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE idp_orchestrator TO orchestrator_service;
```

### 3. Enable SSL (Production)

```bash
export DB_SSLMODE=require
```

---

## Configuration

### Environment Variables

```bash
DB_HOST=postgres.production.internal
DB_PORT=5432
DB_USER=orchestrator_service
DB_PASSWORD=secure_password
DB_NAME=idp_orchestrator
DB_SSLMODE=require
```

### Connection Pooling

```yaml
database:
  maxOpenConnections: 25
  maxIdleConnections: 10
  connectionMaxLifetime: "15m"
  connectionMaxIdleTime: "5m"
```

---

## Migrations

innominatus automatically runs database migrations on startup.

**Migration Files:** `migrations/`

**Tables Created:**
- `workflow_executions` - Workflow tracking
- `workflow_step_executions` - Step-level details
- `applications` - Deployed applications
- `resource_instances` - Provisioned resources
- `sessions` - User sessions
- `user_api_keys` - API keys for OIDC users
- `schema_migrations` - Migration tracking

**See:** [Database Migrations Guide](../migrations/README.md) for details.

---

## Backup & Restore

### Backup

```bash
pg_dump -h $DB_HOST -U $DB_USER -d idp_orchestrator \
  -F c -f orchestrator-backup-$(date +%Y%m%d).dump
```

### Restore

```bash
pg_restore -h $DB_HOST -U $DB_USER -d idp_orchestrator \
  -c orchestrator-backup-20251006.dump
```

### Automated Backup (Cron)

```bash
# Add to crontab
0 2 * * * /usr/local/bin/backup-innominatus-db.sh
```

---

## Monitoring

### Check Database Size

```sql
SELECT pg_size_pretty(pg_database_size('idp_orchestrator'));
```

### Check Connection Count

```sql
SELECT count(*) FROM pg_stat_activity
WHERE datname = 'idp_orchestrator';
```

### View Active Queries

```sql
SELECT pid, usename, application_name, state, query
FROM pg_stat_activity
WHERE datname = 'idp_orchestrator' AND state != 'idle';
```

---

## Troubleshooting

### Connection Refused

Check PostgreSQL is running:

```bash
psql -h $DB_HOST -U $DB_USER -d idp_orchestrator -c "SELECT 1;"
```

### Too Many Connections

Adjust PostgreSQL `max_connections`:

```sql
ALTER SYSTEM SET max_connections = 200;
SELECT pg_reload_conf();
```

### Slow Queries

Enable query logging:

```sql
ALTER DATABASE idp_orchestrator SET log_min_duration_statement = 1000;
```

---

## Production Checklist

- [ ] SSL/TLS enabled (`DB_SSLMODE=require`)
- [ ] Connection pooling configured
- [ ] Backup automated (daily)
- [ ] Monitoring configured (connection count, query performance)
- [ ] Dedicated service account (not superuser)
- [ ] Database firewall rules (only innominatus pods)

---

**See:** [Database Migrations README](../migrations/README.md)
