# Changelog

## v1.0.0

Queen 1.0 is the first stable release of the PostgreSQL-first migration library and embedded CLI.

### Upgrade notes

- From v0.9.0, upgrade the dependency to `github.com/dmedovich/queen@v1.0.0`. The module path and Queen migration history format do not change; applied migrations do not need to be replayed.
- The v1.0 support baseline is Go 1.26.3 or newer and PostgreSQL 15 or 16. PostgreSQL is the reference production driver. MySQL, SQLite, ClickHouse, CockroachDB, and SQL Server retain their [driver-specific guarantees](https://dmedovich.github.io/queen/docs/support-matrix).
- Keep previously applied migration versions and checksums unchanged. If moving from Goose, import its SQL files and [adopt its applied history](https://dmedovich.github.io/queen/docs/goose-import) before running Queen `up`.

### Stable release contract

- PostgreSQL transactional migration SQL and its history record commit or roll back together. Advisory locking uses a pinned connection. Schema-qualified history tables, per-migration SQL timeouts, and a `pgxpool` adapter are supported.
- `NonTransactional` migrations support commands such as `CREATE INDEX CONCURRENTLY`. Interrupted commands leave a recovery marker; inspect the database object before using `recover`.
- The embedded CLI supports source registry verification, disposable rollback tests, deployment plans, Goose history adoption, and machine-readable CI checks. See the [CI/CD guide](https://dmedovich.github.io/queen/docs/ci-cd) and [compatibility contract](https://dmedovich.github.io/queen/docs/compatibility).

### Changes since v0.9.0

- Move the documentation site into this repository and reduce README to an introduction and example. Documentation builds generate versioned release pages from this changelog; publishing a GitHub release deploys the site through GitHub Actions.
- Add PostgreSQL 15 and 16 integration jobs, with coverage for connection loss, SQL timeouts, interrupted concurrent index recovery, history-table upgrades, and restricted deployment roles.
- Return context cancellation promptly when an already-cancelled context reaches the PostgreSQL lock method.

## v0.9.0

### Upgrade notes

- The Go module moved from `github.com/yaop-labs/queen` to `github.com/dmedovich/queen`. Update application imports and the module requirement together. The `v0.8.0` tag belongs to the old module path; use the new path with `v0.9.0`.
- `Up`, `Down`, `Reset`, and `Validate` now reject applied versions absent from the local registry. `Up` and `Validate` also reject a pending version older than an applied version. Rolling deployments that intentionally need the previous behavior can set `Config.AllowUnknownApplied` and, separately, `Config.AllowOutOfOrder` (or the corresponding CLI flags).
- PostgreSQL is the reference production target. Other drivers retain the guarantees and limitations listed in the [support matrix](https://dmedovich.github.io/queen/docs/support-matrix).

### Added and improved

- PostgreSQL migrations use a connection pinned to the advisory lock. Table initialization is serialized, and transactional migration SQL and its history record commit together. The migration table may be schema qualified.
- Per-migration PostgreSQL `SQLLockTimeout` and `StatementTimeout` settings, plus opt-in single-command `NonTransactional` SQL for operations such as `CREATE INDEX CONCURRENTLY`. Interrupted non-transactional migrations keep a recovery marker; `doctor` and `recover` support explicit inspection and resolution.
- Goose SQL import handles optional `Down`, `StatementBegin`/`StatementEnd`, and supported `NO TRANSACTION` files. `adopt-goose` previews an existing PostgreSQL Goose history and transfers a verified, matching applied prefix to Queen in one transaction. Unsupported Goose constructs are rejected during import.
- `verify-registry` checks that the built migrator contains the versions declared in migration source files. The generated project registers files created by `queen create` automatically. `check --json` and the CI/CD guide support building, testing, and deploying the same migrator binary.
- CI covers formatting, linting, vetting, race tests, and database integration suites.
