---
id: driver-clickhouse
title: ClickHouse Driver
---

# ClickHouse Driver

ClickHouse support is best suited for controlled, serialized migration runs.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/clickhouse"
    _ "github.com/ClickHouse/clickhouse-go/v2"
)

db, err := sql.Open("clickhouse", os.Getenv("CLICKHOUSE_DSN"))
if err != nil {
    return err
}

driver, err := clickhouse.New(db)
if err != nil {
    return err
}

q := queen.New(driver)
```

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | `ReplacingMergeTree` table ordered by version. |
| Lock table | Separate `<table>_lock` `ReplacingMergeTree` table with TTL cleanup. |
| Identifier quoting | Double quotes. |
| Placeholders | `?`. |
| Lock owner | Generated owner ID per driver instance. |
| Locking | Table-backed check/insert through `base.AcquireTableLock`. |
| Unlock | `ALTER TABLE ... DELETE WHERE lock_key = ? AND owner_id = ?`. |
| Execution | Uses the shared base transaction wrapper, but ClickHouse does not provide PostgreSQL-style transactions. |
| Atomic record | No. |

## Guarantees

- Queen creates a migration metadata table and a lock table.
- Expired lock rows are cleaned with a driver-specific cleanup query.
- The lock path has timeout/retry behavior through the shared table-lock helper.

## Limitations

- ClickHouse has no true general-purpose transaction model for migrations.
- The lock is best-effort. `ReplacingMergeTree` and mutations are eventually reconciled.
- `ALTER TABLE ... DELETE` is asynchronous in ClickHouse, so unlock visibility can lag.
- Keep deployment migrators serialized outside the database.

## Operational advice

- Run exactly one migration job per ClickHouse environment.
- Prefer idempotent DDL where possible.
- Use `plan` and `doctor` before applying migrations.
