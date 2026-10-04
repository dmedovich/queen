---
id: compatibility
title: Compatibility and Upgrades
---

# Compatibility and Upgrades

Queen's stable contract for v1.0 is PostgreSQL-first. Other drivers remain available with the database-specific guarantees in the [support matrix](support-matrix.md).

## Versions

The v1.0 support baseline is Go 1.26.3 or newer and PostgreSQL 15 or 16. The PostgreSQL integration suite runs against both versions in CI. Release notes record any future change to those minimums or to migration behavior. Other drivers follow the [support matrix](support-matrix.md).

The module path changed at v0.9.0 from `github.com/yaop-labs/queen` to `github.com/dmedovich/queen`. Update Go imports and requirements together. Migration history is stored in the database, so changing the import path does not require replaying migrations.

## Applied history

Queen identifies migrations by version and verifies the stored checksum against the registered migration. Do not edit an applied migration to change its behavior; add a new version. For Go-function migrations, set `ManualChecksum` and bump it when behavior changes.

By default, `Up`, `Down`, `Reset`, and `Validate` stop when the database contains a version absent from the binary's registry. `Up` and `Validate` also reject an older pending version after a newer applied version. During an intentional rolling deployment, `AllowUnknownApplied` (CLI `--allow-unknown-applied`) permits the former case. `AllowOutOfOrder` (CLI `--allow-out-of-order`) permits the latter. Use each override only when the deployment plan requires it.

The library returns copies of registered migrations. Treat config and migration values as startup inputs; changing returned values does not reconfigure a running `Queen`.

## v1.0 compatibility contract

- Public Go packages, exported types, methods, and options follow semantic versioning from v1.0. Additive APIs can appear in minor releases; breaking changes require a new major version.
- The default history table is `queen_migrations`. A version is applied at most once. Versions sort naturally; an older pending version is rejected after a newer version was applied unless `AllowOutOfOrder` is set.
- SQL migration checksums derive from the registered SQL. Go-function behavior cannot be hashed reliably, so use `ManualChecksum` and change it when that behavior changes. Applied checksum drift fails validation.
- Callers can identify Queen's sentinel errors with `errors.Is`, including `ErrLockTimeout`, `ErrChecksumMismatch`, `ErrUnknownApplied`, `ErrOutOfOrderMigration`, and `ErrIncompleteMigration`. Migration-specific failures may wrap a `MigrationError` that supports `errors.As`.
- The CLI command names and documented flags in the [CLI reference](cli-reference.md) are part of the v1.0 interface. Scripts should use exit status and `--json` output where offered; human-readable text is not a machine interface.
- PostgreSQL transactional migration SQL and its history row commit or roll back together. `NonTransactional` SQL has an explicit recovery marker and needs operator verification after interruption. Other drivers provide only the guarantees listed in their support-matrix rows.

Compatibility promises cover documented behavior. They do not make arbitrary migration SQL portable between database engines or guarantee that application database permissions are sufficient for a given migration.

## Operational recovery

PostgreSQL transactional migrations commit the SQL and history row together. Opt-in non-transactional SQL uses an incomplete marker when execution is interrupted. Inspect the database object and use [`recover`](cli-reference.md#recover) to record the matching state; recovery never executes migration SQL.

For a database previously managed by Goose, import files and [adopt the applied history](goose-import.md#adopt-an-existing-goose-database) before running Queen `up`.
