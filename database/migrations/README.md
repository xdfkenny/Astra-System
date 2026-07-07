# Astra Database Migrations

This directory contains ordered PostgreSQL migration scripts for the Astra-Service platform.

## Files

| File | Purpose |
|------|---------|
| `0001_init.sql` | Core schema: enums, tables, indexes, constraints, foreign keys, `updated_at` triggers, soft-delete partial indexes, and the partitioned `audit_logs` and `analytics_events` tables. |
| `0002_outbox_relay.sql` | Outbox relay helper functions (`publish_outbox_batch`, `prune_published_outbox`). The `outbox_events` table is created in `0001_init.sql`; this file repeats it idempotently with `IF NOT EXISTS` so it can also stand alone. |
| `0003_seed_permissions.sql` | Global permissions and system tenant roles (Admin, Manager, Operator, Viewer). |
| `0004_partitioning.sql` | Default partitions and monthly partitions for `audit_logs` and `analytics_events` (current month + next 12 months), plus partition maintenance helpers. |

## Naming conventions

- Prefix migrations with a zero-padded four-digit number: `0001`, `0002`, etc.
- Use a descriptive, snake_case suffix: `_init.sql`, `_outbox_relay.sql`.
- Each file must contain clearly marked `UP` and `DOWN` sections.
- Migrations must be idempotent where possible (`IF NOT EXISTS`, `ON CONFLICT DO NOTHING`).

## Applying migrations

### With `psql` (simplest)

Run the files in numeric order against the target database:

```bash
psql -U astra -d astra_service -f database/migrations/0001_init.sql
psql -U astra -d astra_service -f database/migrations/0002_outbox_relay.sql
psql -U astra -d astra_service -f database/migrations/0003_seed_permissions.sql
psql -U astra -d astra_service -f database/migrations/0004_partitioning.sql
```

Wrap the set in a transaction if the tool supports it, or run each file inside a transaction block:

```bash
psql -U astra -d astra_service -1 -f database/migrations/0001_init.sql
```

> The `-1` flag runs the file as a single transaction.

### With `pg-migrate` (Node.js)

If the project uses `node-pg-migrate`, place each migration under the configured migrations directory and run:

```bash
npx node-pg-migrate up
```

To migrate to a specific version:

```bash
npx node-pg-migrate up 4
```

### With Atlas

Apply all migrations in order:

```bash
atlas schema apply \
  --url "postgres://astra:password@localhost:5432/astra_service" \
  --to "file://database/migrations" \
  --dev-url "postgres://astra:password@localhost:5432/astra_dev"
```

Or use Atlas's versioned migration workflow after the files are imported as Atlas migrations.

## Rollback procedure

Migrations are rolled back in reverse order. Each file has a `DOWN` section that drops the objects it creates.

### `psql` rollback

```bash
psql -U astra -d astra_service -f database/migrations/0004_partitioning.sql   # run DOWN manually
psql -U astra -d astra_service -f database/migrations/0003_seed_permissions.sql
psql -U astra -d astra_service -f database/migrations/0002_outbox_relay.sql
psql -U astra -d astra_service -f database/migrations/0001_init.sql
```

> Most migration tools run `DOWN` automatically when reverting; with plain `psql` you must extract or run the `DOWN` section yourself.

### `pg-migrate` rollback

```bash
npx node-pg-migrate down
```

To revert a specific number of migrations:

```bash
npx node-pg-migrate down 1
```

### Atlas rollback

```bash
atlas schema apply \
  --url "postgres://astra:password@localhost:5432/astra_service" \
  --to "file://database/migrations?version=3" \
  --dev-url "postgres://astra:password@localhost:5432/astra_dev"
```

## Partition maintenance

After `0004_partitioning.sql` is applied, add future partitions before the current range ends:

```sql
-- Create partitions for the next 6 months.
SELECT create_monthly_partitions(6);
```

To create a single partition explicitly:

```sql
SELECT create_future_audit_partition('2028-01-01'::date);
SELECT create_future_analytics_partition('2028-01-01'::date);
```

Schedule a monthly job (e.g., pg_cron) to call `create_monthly_partitions(3)` so new partitions are always ready before data arrives.

## Validation

Before applying to production, syntax-check the files. If a PostgreSQL server is available:

```bash
# Parse-check only (does not execute).
psql -U astra -d astra_service --single-transaction -f database/migrations/0001_init.sql
```

Or use a local ephemeral database:

```bash
createdb astra_migrate_test
psql -d astra_migrate_test -f database/migrations/0001_init.sql
psql -d astra_migrate_test -f database/migrations/0002_outbox_relay.sql
psql -d astra_migrate_test -f database/migrations/0003_seed_permissions.sql
psql -d astra_migrate_test -f database/migrations/0004_partitioning.sql
```

## Source of truth

The TypeScript Drizzle schemas in `../schemas/drizzle.ts` and the Go structs in `../schemas/go_structs.go` describe the final schema state. Migrations should always be regenerated or updated when those files change.
