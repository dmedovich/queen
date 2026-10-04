---
id: metadata
title: Migration Metadata
---

# Migration Metadata

Queen stores each currently applied migration's version, name, checksum, and application time. Built-in drivers also support:

| Field | Meaning |
| --- | --- |
| `applied_by` | Current OS user when available. |
| `duration_ms` | Migration execution time in milliseconds. |
| `hostname` | Host where the migrator ran, when available. |
| `environment` | Value of `QUEEN_ENV`. |
| `action` | Operation such as `apply` or `mark-applied`. |
| `status` | Current result, normally `success`; PostgreSQL non-transactional migrations can temporarily be `applying` or `rolling_back`. |
| `error_message` | Optional error details when recorded by the driver flow. |

The history table represents **current applied state**, not an append-only log of every attempt. Rollback removes a successful migration record. Use `Queen.Status(ctx)` for the current state in code, or `Queen.Driver().GetApplied(ctx)` when you need persisted records with metadata.

The PostgreSQL recovery markers are described under [non-transactional migrations](migrations.md#postgresql-statement-limits-and-concurrent-indexes).
