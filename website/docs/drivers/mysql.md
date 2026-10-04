---
id: driver-mysql
title: MySQL Driver
---

# MySQL Driver

The MySQL driver supports Queen's core migration flow, with MySQL-specific DDL caveats.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/mysql"
    _ "github.com/go-sql-driver/mysql"
)

db, err := sql.Open("mysql", os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}

q := queen.New(mysql.New(db))
```

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | InnoDB table with metadata columns. |
| Identifier quoting | Backticks. |
| Placeholders | `?`. |
| Locking | MySQL `GET_LOCK(lockName, timeoutSeconds)` on a pinned `*sql.Conn`. |
| Unlock | `RELEASE_LOCK(lockName)` on the same connection. |
| Lock name | `queen_lock_` plus migration table name. |
| Execution | `database/sql` transaction through the shared base driver. |
| Atomic record | No. Migration body commits before `Record`. |

## Guarantees

- Locking uses a named MySQL lock and keeps the owning connection until unlock.
- The migration table is created with `ENGINE=InnoDB`.
- Existing metadata columns are checked through `INFORMATION_SCHEMA.COLUMNS`.

## Limitations

- MySQL DDL often causes implicit commits. A migration can contain statements that do not roll back the way PostgreSQL DDL does.
- Migration metadata is not written in the same transaction as the migration body.
- Lock timeout is passed to MySQL in whole seconds.
- This path is supported but less heavily exercised than PostgreSQL.

## DSN

Typical `go-sql-driver/mysql` DSN:

```text
user:pass@tcp(localhost:3306)/myapp?parseTime=true
```
