# Database Migrations

## Overview

Database migrations are managed via raw SQL files in `database/migrations/`. The TypeScript ORM (Drizzle) is used for schema definitions and type generation, but migrations are hand-written SQL.

## Migration Files

| File | Description | Applied |
|------|-------------|---------|
| `0001_init.sql` | Full initial schema (all 22 tables, enums, indexes) | Yes |
| `0002_outbox_relay.sql` | Outbox relay processing and retry logic | Yes |
| `0003_seed_permissions.sql` | RBAC permissions seed data | Yes |
| `0004_partitioning.sql` | Monthly partitioning for audit_logs and analytics_events | Yes |
| `0005_webauthn_sessions.sql` | WebAuthn session and credential storage | Yes |

## Migration Workflow

### Applying Migrations

1. Connect to target database
2. Apply migrations sequentially (tracked in `_migrations` table)
3. Verify schema with Drizzle type generation

```bash
# Using PostgreSQL client
psql $DATABASE_URL -f database/migrations/0001_init.sql
psql $DATABASE_URL -f database/migrations/0002_outbox_relay.sql
# ... etc
```

### Creating New Migrations

1. Create new SQL file: `database/migrations/0006_description.sql`
2. Update `database/schemas/drizzle.ts` to reflect schema changes
3. Update `database/schemas/go_structs.go` for Go mirror
4. Test migration locally
5. Commit migration file

## Schema Sync

The TypeScript Drizzle schema and Go structs must stay in sync:

- `database/schemas/drizzle.ts` - 1416 lines, full Drizzle ORM schema with relations
- `database/schemas/go_structs.go` - 595 lines, Go struct mirror for services
- Tests validate consistency between both representations

## Rollback Strategy

Migrations are designed to be **forward-only**. Rollbacks are handled by:
1. Writing a new migration that reverses the change
2. Testing rollback on staging
3. Applying during maintenance window

## Partition Management

`audit_logs` and `analytics_events` use PostgreSQL partitioning:

```sql
CREATE TABLE audit_logs (
    id UUID DEFAULT gen_random_uuid(),
    event_type audit_event_type NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ...
) PARTITION BY RANGE (created_at);

-- Monthly partitions created via cron/trigger
CREATE TABLE audit_logs_2026_07 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
```

New partitions are created automatically by a database function or scheduled job.
