---
id: drivers
title: Drivers
---

# Drivers

Queen supports several databases, but the production contract is intentionally PostgreSQL-first.

Start with [Support Matrix](support-matrix.md) when comparing driver guarantees. Current limitations to keep in mind:

- SQLite locking is for local/single-process use, not distributed coordination.
- ClickHouse locking is best-effort; keep deployment migrators serialized.
- CockroachDB retries `40001` serialization errors up to three times; Go callbacks must be retry-safe.
- MySQL and ClickHouse record migration metadata separately from the body.

The detailed edge list lives in [Known Limitations](known-limitations.md).

| Database | Driver package | Locking | Transaction behavior | Migration record atomic with body |
| --- | --- | --- | --- | --- |
| PostgreSQL | `drivers/postgres` | Production-ready advisory lock pinned to one connection | Yes by default; opt-in non-transactional commands | Yes by default; recovery markers for non-transactional commands |
| MySQL | `drivers/mysql` | `GET_LOCK` based | Depends on MySQL DDL semantics | No |
| SQLite | `drivers/sqlite` | Local/single-process use recommended | Yes for transactional statements | Yes |
| ClickHouse | `drivers/clickhouse` | Best-effort table guard | No true transaction support | No |
| CockroachDB | `drivers/cockroachdb` | Table row with expiry | Serializable transactions with bounded retries | Yes |
| MSSQL | `drivers/mssql` | Application lock | Yes for transactional statements | Yes |

## PostgreSQL

Use PostgreSQL when you need the strongest Queen behavior today:

- advisory lock pinned to one connection;
- SQL and Go-function migration body inside a transaction;
- migration record written in the same transaction as the migration body;
- pgxpool adapter support.

```go
q := queen.New(postgres.New(db))
```

Detailed page: [PostgreSQL Driver](drivers/postgres.md).

## MySQL

MySQL uses `GET_LOCK` on a pinned connection and normal `database/sql` transaction execution. DDL rollback behavior follows MySQL rules.

Detailed page: [MySQL Driver](drivers/mysql.md).

## SQLite

SQLite is useful for tests, local tools, and single-process applications. Do not treat the current SQLite lock path as distributed coordination between multiple migrator processes.

Detailed page: [SQLite Driver](drivers/sqlite.md).

## ClickHouse

ClickHouse does not provide the same transaction model as PostgreSQL. Treat its migration lock as a best-effort guard and keep migrations operationally serialized.

Detailed page: [ClickHouse Driver](drivers/clickhouse.md).

## CockroachDB

CockroachDB retries `40001` serialization failures up to three times. Go-function callbacks can run more than once and must be retry-safe; an error after the retry limit still needs operator review.

Detailed page: [CockroachDB Driver](drivers/cockroachdb.md).

## MS SQL Server

MSSQL uses `sp_getapplock` / `sp_releaseapplock` on a pinned session connection.

Detailed page: [MS SQL Server Driver](drivers/mssql.md).
