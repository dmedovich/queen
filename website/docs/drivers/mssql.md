---
id: driver-mssql
title: MS SQL Server Driver
---

# MS SQL Server Driver

The MSSQL driver supports Queen's core flow using SQL Server application locks.

## Packages

```go
import (
    "database/sql"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/mssql"
    _ "github.com/microsoft/go-mssqldb"
)

db, err := sql.Open("sqlserver", os.Getenv("MSSQL_DSN"))
if err != nil {
    return err
}

q := queen.New(mssql.New(db))
```

## How it works

| Area | Implementation |
| --- | --- |
| Migration table | Created with `IF OBJECT_ID(..., 'U') IS NULL`. |
| Identifier quoting | Brackets. |
| Placeholders | `@p1`, `@p2`, ... |
| Locking | `sp_getapplock` with `LockOwner = 'Session'` on a pinned `*sql.Conn`. |
| Unlock | `sp_releaseapplock` on the same connection. |
| Lock name | `queen_lock_` plus migration table name. |
| Execution | `database/sql` transaction through the shared base driver. |
| Atomic record | Yes. `RecordTx` and `RemoveTx` run with the migration body. |

## Guarantees

- Lock connection lifecycle is mutex-guarded.
- Nested lock attempts on the same driver fail deterministically.
- Table names used in `OBJECT_ID(N'...')` string literals escape single quotes.

## Limitations

- Transactional migration body and metadata record commit together.
- SQL Server DDL transaction behavior depends on the statement and database settings.
- This path is supported but less heavily exercised than PostgreSQL.

## DSN

```text
sqlserver://user:pass@localhost:1433?database=myapp
```
