---
id: driver-postgres
title: PostgreSQL Driver
---

# PostgreSQL Driver

PostgreSQL is Queen's reference production driver.

The v1.0 test matrix covers PostgreSQL 15 and 16. A deployment role may run migrations against an existing history table without `CREATE` permission on its schema if it has schema `USAGE`, the required privileges on the history table, and the privileges needed by each migration's SQL. Initial creation or upgrade of the history table requires the corresponding DDL privileges; prepare it with an owner role first if deployments use a restricted role.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/postgres"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}

q := queen.New(postgres.New(db))
```

With native `pgxpool.Pool`:

```go
pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}

q := queen.New(postgres.NewFromPool(pool))
```

`Queen.Close()` closes the `database/sql` adapter handle, not the caller-owned pool.

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | `CREATE TABLE IF NOT EXISTS <table>` with metadata columns. |
| Identifier quoting | Double quotes. |
| Placeholders | `$1`, `$2`, ... |
| Locking | `pg_advisory_lock(lockID)` on a pinned `*sql.Conn`. |
| Unlock | `pg_advisory_unlock(lockID)` on the same pinned connection. |
| Lock key | Stable FNV-1a hash of the migration table name. |
| Execution | Transaction on the lock-holding connection by default. |
| Atomic record | Yes for transactional migrations. `RecordTx` and `RemoveTx` share the migration transaction. |
| Pool support | `postgres.NewFromPool` and `NewFromPoolWithTableName`. |

## Guarantees

- One Queen operation on one driver instance cannot overwrite the advisory lock connection.
- PostgreSQL migration body and migration metadata are committed in the same transaction.
- `LockTimeout` bounds advisory lock acquisition.
- `ErrLockTimeout` wraps lock timeout failures.
- Table initialization is serialized under the migration lock, including upgrades of existing history tables.
- Schema-qualified migration tables are supported with `postgres.NewWithTableName(db, "schema.queen_migrations")`.

## Limitations

- Advisory lock IDs are derived from table names. Use distinct table names for distinct migration domains.
- PostgreSQL is the most tested production path; behavior documented here is stronger than other drivers.
- DDL still follows PostgreSQL's own transaction rules.
- Opt-in non-transactional SQL cannot commit atomically with its history row; Queen keeps recovery markers for interrupted commands.

For per-migration SQL timeouts, concurrent indexes, and recovery, see [Migration Format](../migrations.md#postgresql-statement-limits-and-concurrent-indexes).

After an interrupted non-transactional command, run `doctor` and inspect both the migration record and the intended database object. If an interrupted up completed, resolve it as `applied`; if it did not, remove any partial object before resolving it as `not-applied` and retrying. For an interrupted down, choose `not-applied` only if the rollback completed; otherwise choose `applied`. A failed concurrent index creation may leave an invalid index that must be removed before a retry. [`recover`](../cli-reference.md#recover) changes only the history record and requires `--verified`.

For a concurrent index, inspect PostgreSQL's validity flag before choosing the recovery state:

```sql
SELECT indexrelid::regclass AS index_name, indisvalid, indisready
FROM pg_index
WHERE indexrelid = to_regclass('my_index');
```

If the index is absent or invalid after an interrupted up, drop any invalid index with `DROP INDEX CONCURRENTLY my_index` outside a transaction, then use `recover VERSION --state not-applied --verified` and retry. If it is valid and has the intended definition, use `recover VERSION --state applied --verified`.

## Recommended production config

```yaml
production:
  driver: postgres
  dsn: "${DATABASE_URL}"
  table: queen_migrations
  lock_timeout: 30m
  require_confirmation: true
  require_explicit_unlock: true
```
