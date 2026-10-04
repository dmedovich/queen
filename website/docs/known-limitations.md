---
id: known-limitations
title: Known Limitations
---

# Known Limitations

This page is intentionally direct. Queen should be pleasant to use, but migration tools also need clear edges.

## Current driver limits

| Area | Current behavior | What to do |
| --- | --- | --- |
| PostgreSQL | Reference path; strongest locking and transaction semantics. | Prefer it for production when you can. |
| MySQL | Uses `GET_LOCK`, but DDL rollback follows MySQL rules. | Test rollback scripts on the same MySQL flavor/version you deploy. |
| SQLite | Locking is local and not a distributed deployment lock. | Use for tests, local tools, and single-process applications. |
| ClickHouse | Locking is best-effort and table based. ClickHouse has no PostgreSQL-like transaction model. | Serialize deploys outside Queen. |
| CockroachDB | Serializable conflicts can produce `40001` errors; Queen retries up to three times. | Make Go callbacks retry-safe and inspect state if retries are exhausted. |
| MSSQL | Uses application locks, but DDL behavior follows SQL Server rules. | Treat rollback testing as required before production use. |

## `queen import`

`queen import` currently imports supported Goose SQL migrations. It writes Queen Go migrations and fails instead of overwriting existing generated files. For an already migrated database, use [Goose history adoption](goose-import.md#adopt-an-existing-goose-database) before `up`.

Known limits:

- it rejects Goose Go migrations, `ENVSUB`, and unsupported `NO TRANSACTION` files;
- it does not prove imported SQL is semantically correct;
- it preserves versions, so odd historical naming stays odd;
- imported migrations should be reviewed before merging.

Run import in dry-run mode first:

```bash
go run ./cmd/migrate import ./db/migrations --from goose --output imported_migrations --dry-run
```

## `create` and naming

`create` writes into `migrations/` and supports:

```bash
go run ./cmd/migrate create create_users --type sql
go run ./cmd/migrate create backfill_profiles --type go
```

The command derives the next version from existing migration filenames and `.queen.yaml` naming config. Sequential and padded sequential naming are the practical paths today.

Current limitation: semver naming exists in the naming config, but `create` does not expose a manual `--version` flag yet. Create semver migrations by writing the file yourself or using a custom project template.

## Go-function checksums

Queen cannot hash Go function source at runtime. A function migration should set `ManualChecksum` and bump it when the function behavior changes:

```go
queen.M{
    Version:        "003",
    Name:           "normalize_emails",
    ManualChecksum: "normalize-emails-v2",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `UPDATE users SET email = LOWER(TRIM(email))`)
        return err
    },
}
```

## Rollback is explicit

Queen does not invent rollback SQL. A rollback command fails when a migration has no `DownSQL` or `DownFunc`.

That is intentional. Some production migrations are irreversible, and pretending otherwise is worse than making the limitation visible.

## TUI scope

The TUI is an inspection and debugging surface. It is useful for local diagnosis, viewing status, reading plans, and watching tap events.

Do not make the TUI the main production deployment path. Use the CLI commands directly in CI/CD.
