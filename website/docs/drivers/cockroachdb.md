---
id: driver-cockroachdb
title: CockroachDB Driver
---

# CockroachDB Driver

CockroachDB support follows PostgreSQL-like SQL syntax, with CockroachDB's serializable transaction behavior.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/cockroachdb"
    _ "github.com/jackc/pgx/v5/stdlib"
)

db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}

driver, err := cockroachdb.New(db)
if err != nil {
    return err
}

q := queen.New(driver)
```

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | SQL table with metadata columns. |
| Lock table | Separate `<table>_lock` table with `lock_key` primary key and `expires_at`. |
| Identifier quoting | Double quotes. |
| Placeholders | `$1`, `$2`, ... |
| Lock owner | Generated owner ID per driver instance. |
| Locking | Table-backed lock through cleanup, check, and insert. |
| Unlock | Deletes the lock row for the current owner. |
| Execution | `database/sql` transaction through the shared base driver. |
| Atomic record | Yes. `RecordTx` and `RemoveTx` run with the migration body. |

## Guarantees

- Lock rows expire and are cleaned during acquisition attempts.
- Wrapped `sql.ErrNoRows` is treated as no active lock.
- Queen uses natural status ordering before planning operations.

## Limitations

- CockroachDB can return retryable serialization errors (`40001`).
- The driver retries `40001` serialization failures up to three times; Go callbacks must be safe to run again.
- Migration metadata is written in the same transaction as the migration body.

## Operational advice

- Run one migrator job per environment.
- If retries are exhausted, inspect the migration record and schema before rerunning the migrator.
- Prefer PostgreSQL for the first production target when you need Queen's strongest guarantees today.
