# Changelog

## v1.0.0

### Added and improved

- Move the documentation site into the Queen repository, make README a short introduction, and generate release pages from this changelog during documentation builds.
- Run PostgreSQL integration tests against versions 15 and 16 in CI.
- Cover PostgreSQL connection loss, SQL timeouts, interrupted concurrent index recovery, and restricted deployment roles with integration tests. Return context cancellation promptly when an already-cancelled context reaches the PostgreSQL lock method.

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

See [v1.0 readiness](https://dmedovich.github.io/queen/docs/v1-readiness) for the release gates after `v0.9.0`.
