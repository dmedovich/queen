---
id: driver-sqlite
title: SQLite Driver
---

# SQLite Driver

SQLite is useful for tests, local tooling, and single-process applications.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/sqlite"
    _ "github.com/mattn/go-sqlite3"
)

db, err := sql.Open("sqlite3", "./app.db?_journal_mode=WAL")
if err != nil {
    return err
}

q := queen.New(sqlite.New(db))
```

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | SQLite table with `WITHOUT ROWID`. |
| Identifier quoting | Double quotes. |
| Placeholders | `?`. |
| Timestamp parsing | Queen parses SQLite text timestamps. |
| Locking | Sets `busy_timeout`, sets `locking_mode=EXCLUSIVE`, runs `BEGIN IMMEDIATE`, then commits. |
| Execution | `database/sql` transaction through the shared base driver. |
| Atomic record | Yes. `RecordTx` and `RemoveTx` run with the migration body. |

## Guarantees

- SQL statements supported by SQLite transactions are run inside a transaction.
- The driver works well for in-memory and file-backed local databases.

## Limitations

- The current lock path does not hold a pinned lock transaction for the full migration duration.
- Do not use SQLite locking as distributed coordination between multiple migrator processes.
- Transactional migration body and metadata record commit together.
- Some SQLite DDL behavior depends on SQLite version and pragma settings.

## Recommended local DSN

```text
./app.db?_journal_mode=WAL&_busy_timeout=5000
```
