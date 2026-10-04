---
id: support-matrix
title: Support Matrix
---

# Support Matrix

Queen has one API across drivers, but databases do not provide the same locking, transaction, and DDL behavior. Use this page before picking a production target.

## Driver capabilities

| Database | Driver package | Lock type | Migration body transaction | Migration record atomic with body | Rollback support | Production fit |
| --- | --- | --- | --- | --- | --- | --- |
| PostgreSQL | `drivers/postgres` | Advisory lock on a pinned connection | Yes by default | Yes by default; non-transactional SQL uses recovery markers | Yes, when migration has `DownSQL` or `DownFunc` | Reference production path |
| MySQL | `drivers/mysql` | `GET_LOCK` on a pinned connection | Yes for transactional statements | No | Depends on MySQL DDL semantics | Usable with DDL caveats |
| SQLite | `drivers/sqlite` | Local SQLite write coordination | Yes for transactional statements | Yes | Yes for transactional statements | Local, tests, single-process tools |
| ClickHouse | `drivers/clickhouse` | Best-effort table guard with expiry | No true transaction model | No | Limited by ClickHouse DDL behavior | Operationally serialized deployments |
| CockroachDB | `drivers/cockroachdb` | Table row with expiry | Serializable transactions with bounded `40001` retries | Yes | Yes for transactional statements; callbacks must be retry-safe | Needs retry-aware operations |
| MSSQL | `drivers/mssql` | `sp_getapplock` on a pinned session | Yes for transactional statements | Yes | Depends on SQL Server DDL semantics | Usable with DDL caveats |

## Feature matrix

| Feature | PostgreSQL | MySQL | SQLite | ClickHouse | CockroachDB | MSSQL |
| --- | --- | --- | --- | --- | --- | --- |
| SQL migrations | Yes | Yes | Yes | Yes | Yes | Yes |
| Go-function migrations | Yes | Yes | Yes | Yes | Yes | Yes |
| Mixed SQL + Go migrations | Yes | Yes | Yes | Yes | Yes | Yes |
| Automatic checksum for SQL | Yes | Yes | Yes | Yes | Yes | Yes |
| Manual checksum for functions | Yes | Yes | Yes | Yes | Yes | Yes |
| Bounded lock timeout | Yes | Yes | Limited by SQLite behavior | Best effort | Yes | Yes |
| Multi-process lock suitable for production | Yes | Yes | No | Best effort | Yes, with retry care | Yes |
| Migration body and record in one transaction | Yes by default | No | Yes | No | Yes | Yes |
| Pool adapter | `pgxpool` | No | No | No | No | No |

## How to read "atomic record"

"Migration record atomic with body" means Queen commits both the migration work and the row in the migration table in the same database transaction. PostgreSQL, SQLite, CockroachDB, and MSSQL implement transactional recording. PostgreSQL's opt-in non-transactional mode uses recovery markers because those SQL commands cannot share a transaction with the record.

For MySQL and ClickHouse, a migration can succeed and the record write can fail as a separate step. Check `status`, `log`, and your schema before retrying.

## Recommended choices

- Use PostgreSQL for the strongest current Queen contract.
- Use SQLite for local development, tests, and embedded tools.
- Use MySQL and MSSQL when your team already understands each database's DDL transaction behavior.
- Use CockroachDB with retry-aware deployment scripts.
- Use ClickHouse only with one migrator process at a time.
