---
id: troubleshooting
title: Troubleshooting
---

# Troubleshooting

## Checksum mismatch

Queen detected that an applied migration's stored checksum no longer matches the registered migration. This can follow a SQL edit or a changed `ManualChecksum` for a Go-function migration.

What to do:

- restore the original migration if the change was accidental;
- create a new migration for the intended change;
- only update recorded state manually if you have audited the database and understand the drift.

## No rollback defined

`Down` and `Reset` require `DownSQL` or `DownFunc`.

For irreversible migrations, document the reason in code and avoid workflows that depend on automatic rollback.

## Incomplete PostgreSQL migration

An interrupted `NonTransactional` command leaves `status=applying` or `status=rolling_back`. `Up`, `Down`, `Reset`, and `Validate` stop with `ErrIncompleteMigration`.

Run `doctor`, inspect the migration record and database object, then use [`recover`](cli-reference.md#recover) with `--state applied` or `--state not-applied` and `--verified`. Recovery only changes history metadata. Remove a partial or invalid object before retrying an incomplete up migration. The [PostgreSQL guide](drivers/postgres.md#limitations) explains the choice for interrupted up and down commands.

## Unknown or out-of-order migration

`ErrUnknownApplied` means the database contains an applied version absent from this binary's registry. Check that the release binary includes the full migration history and run `verify-registry` on its sources. For an intentional rolling deployment where old and new binaries overlap, `--allow-unknown-applied` can permit the old binary to inspect the newer database state.

`ErrOutOfOrderMigration` means a pending version sorts before an already applied one. Check the registry and version names. Use `--allow-out-of-order` only if applying that older version is part of the deployment plan.

## Gap detected

Run:

```bash
go run ./cmd/migrate gap detect --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate gap analyze --driver postgres --dsn "$DATABASE_URL"
```

If a numbering gap is intentional, ignore it:

```bash
go run ./cmd/migrate gap ignore 003 --reason "removed before release"
```

If an application gap exists, inspect carefully before using `gap fill`.

The file format is documented in [Config Files](config-files.md#queenignore).

## Production prompt blocks deploy

Production environments can require:

```bash
--unlock-production
```

Use `--yes` only in trusted CI/CD contexts where the environment and DSN are already controlled.
