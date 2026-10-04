---
id: cli-reference
title: CLI Reference
---

# CLI Reference

Examples use `go run ./cmd/migrate`, but the same commands work with a built migrator binary.

```bash
go build -o migrate ./cmd/migrate
./migrate status --driver postgres --dsn "$DATABASE_URL"
```

## Global options

| Flag | Default | Description |
| --- | --- | --- |
| `--driver` | empty | Database driver: `postgres`, `mysql`, `sqlite`, `clickhouse`, `cockroachdb`, or `mssql`. |
| `--dsn` | empty | Database connection string. |
| `--table` | `queen_migrations` | Migration metadata table. |
| `--timeout` | driver default | Lock timeout, such as `30m` or `1h`. |
| `--use-config` | `false` | Load `.queen.yaml`. |
| `--env` | empty | Select an environment from `.queen.yaml`. |
| `--unlock-production` | `false` | Allow environments that require explicit unlock. |
| `--yes` | `false` | Skip prompts. Use in CI/CD only when inputs are controlled. |
| `--json` | `false` | Emit JSON for commands that support structured output. |
| `--verbose` | `false` | Reserved for verbose output. |
| `--allow-unknown-applied` | `false` | Allow applied versions absent from this binary during an intentional rolling deployment. |
| `--allow-out-of-order` | `false` | Allow an older pending version to run after a newer applied version. |

Environment fallback:

| Variable | Used for |
| --- | --- |
| `QUEEN_DRIVER` | `--driver` |
| `QUEEN_DSN` | `--dsn` |
| `QUEEN_TABLE` | `--table` |

For `.queen.yaml` and `.queenignore` examples, see [Config Files](config-files.md).

## `up`

Applies pending migrations.

```bash
go run ./cmd/migrate up --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --steps 2 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --to 010 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --tap --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--steps` | `0` | Number of pending migrations to apply. `0` means all. |
| `--to` | empty | Apply pending migrations through the given version. Cannot be used with `--steps`. |
| `--tap` | `false` | Show a live tap TUI while migrations run. |
| `--tap-slow-threshold` | `100ms` | Marks tap SQL events as slow. `0` disables slow markers. |
| `--tap-nplus1-threshold` | `5` | Repeated SELECT threshold for N+1 detection. `0` disables it. |

## `down`

Rolls back applied migrations.

```bash
go run ./cmd/migrate down --steps 1 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate down --to 010 --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--steps` | `0` | Number of migrations to roll back. If no target is given, `0` behaves as one rollback. |
| `--to` | empty | Roll back to a target version, keeping that version applied. Cannot be used with `--steps`. |

## `reset`

Rolls back all applied migrations.

```bash
go run ./cmd/migrate reset --driver postgres --dsn "$DATABASE_URL"
```

`reset` is destructive and asks for confirmation unless `--yes` is used.

## `goto`

Moves to a specific registered version. If the target is pending, Queen migrates up. If it is already applied, Queen rolls back newer applied migrations.

```bash
go run ./cmd/migrate goto 010 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate goto latest --driver postgres --dsn "$DATABASE_URL"
```

## `status`

Shows every registered migration with status, applied timestamp, checksum, and rollback availability.

```bash
go run ./cmd/migrate status --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate status --json --driver postgres --dsn "$DATABASE_URL"
```

## `version`

Prints the latest applied migration version.

```bash
go run ./cmd/migrate version --driver postgres --dsn "$DATABASE_URL"
```

## `log`

Shows persisted applied migration records.

```bash
go run ./cmd/migrate log --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate log --last 5 --reverse --with-meta --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--last` | `0` | Show the last N records. |
| `--since` | empty | Filter by date, `YYYY-MM-DD`. |
| `--with-duration` | `false` | Include execution duration. |
| `--with-meta` | `false` | Include metadata such as applied user, hostname, and environment. |
| `--reverse` | `false` | Show newest first. |

## `plan`

Shows a dry-run execution plan.

```bash
go run ./cmd/migrate plan --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate plan --direction down --limit 2 --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--direction` | `up` | `up` for pending migrations, `down` for rollback plan. |
| `--limit` | `0` | Maximum number of migrations to show. `0` means all. |

## `explain`

Shows one migration's plan details: status, type, direction, checksum, warnings, and SQL when available.

```bash
go run ./cmd/migrate explain 010 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate explain 010 --json --driver postgres --dsn "$DATABASE_URL"
```

Go-function source code is not shown; tap can show SQL executed by functions that use `tap.ObserveTx`.

## `diff`

Compares two versions and lists migrations between them.

```bash
go run ./cmd/migrate diff current latest --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate diff 003 +2 --driver postgres --dsn "$DATABASE_URL"
```

Accepted version keywords:

| Value | Meaning |
| --- | --- |
| `current` | Latest applied migration. |
| `latest` | Latest registered migration. |
| `+N` / `-N` | Relative target for the second argument. |

Flag:

| Flag | Default | Description |
| --- | --- | --- |
| `--show-sql` | `false` | Currently prints a note because `MigrationStatus` does not carry SQL. Use `explain` for SQL. |

## `validate`

Validates registered migrations and applied checksums.

```bash
go run ./cmd/migrate validate --driver postgres --dsn "$DATABASE_URL"
```

Use this before a deploy when you want a simple pass/fail check without the fuller CI behavior of `check`.

## `doctor`

Runs diagnostics.

```bash
go run ./cmd/migrate doctor --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate doctor --deep --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate doctor --gaps --fix --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--deep` | `false` | Run slower schema SQL analysis. |
| `--gaps` | `false` | Only check migration gaps. |
| `--fix` | `false` | Print suggested repair commands for detected issues. |

## `check`

CI-oriented validation. It prints a compact sequence of checks and exits non-zero on failure.

```bash
go run ./cmd/migrate check --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate check --ci --no-gaps --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate check --rollback-test --driver postgres --dsn "$TEST_DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--ci` | `false` | Strict mode. Fails on warnings and pending migrations. |
| `--no-pending` | `false` | Fail if pending migrations exist. |
| `--no-gaps` | `false` | Fail if gaps are detected. |
| `--rollback-test` | `false` | Runs `up -> reset -> up` against a clean test database. |

With `--json`, `check` emits one object with `passed`, `failed`, `exit_code`, and `output`, and exits with that code on failure. Run `--ci` after applying migrations; it fails while any remain pending.

`--rollback-test` refuses to run when the target database already has applied migrations.

## `gap`

Detects and manages migration gaps.

```bash
go run ./cmd/migrate gap detect --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate gap analyze --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate gap fill 003 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate gap ignore 003 --reason "removed before release"
go run ./cmd/migrate gap list-ignored
go run ./cmd/migrate gap unignore 003
```

Subcommands:

| Command | Purpose |
| --- | --- |
| `gap detect` | Lists numbering and application gaps. Supports `--json`. |
| `gap analyze` | Prints a human-readable impact summary. |
| `gap fill [versions...]` | Applies missing application-gap migrations. |
| `gap fill --mark-applied` | Records gaps as applied without executing SQL. Use only when manually applied. |
| `gap ignore VERSION --reason ...` | Writes the version into `.queenignore`. |
| `gap list-ignored` | Lists ignored gaps. |
| `gap unignore VERSION` | Removes an ignored gap. |

## `baseline`

Marks existing schema migrations as applied without executing them.

```bash
go run ./cmd/migrate baseline --at 010 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate baseline --version 010 --dry-run --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--name` | `baseline` | Label stored in baseline metadata. |
| `--at` | latest registered version | Mark pending migrations through this version as applied. |
| `--version` | empty | Alias for `--at`. |
| `--dry-run` | `false` | Preview records without writing them. |

Use baseline only when the database schema already matches the migrations being marked.

## `squash`

Combines registered SQL migrations into a new generated SQL migration file.

```bash
go run ./cmd/migrate squash 001,002,003 --into initial_schema --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate squash --from 001 --to 010 --into initial_schema --dry-run --driver postgres --dsn "$DATABASE_URL"
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--into` | required | Name for the generated migration. |
| `--from` | empty | Start version for range squash. |
| `--to` | empty | End version for range squash. |
| `--dry-run` | `false` | Preview without writing. |

Current behavior: only SQL migrations with SQL rollback can be squashed safely.

## `import`

Imports migrations from another tool. Current importer: goose SQL migrations.

```bash
go run ./cmd/migrate import ./db/migrations --from goose --output imported_migrations --dry-run
go run ./cmd/migrate import ./db/migrations --from goose --output imported_migrations
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--from` | required | Source tool. Currently `goose`. |
| `--output` | `migrations` | Output directory. |
| `--dry-run` | `false` | Preview conversion without writing files. |

The importer supports goose `.sql` files with `-- +goose Up` and `-- +goose Down`. It preserves timestamp versions, generates its own `migrations.go`, and fails instead of overwriting an existing registry. Use an unused output directory, then integrate the generated files into the final migrator.

See [Goose import](goose-import.md) for supported annotations and transferring history from an existing Goose database.

## `adopt-goose`

Previews or adopts existing PostgreSQL Goose history. The preview is read-only; apply requires its exact fingerprint and confirmation that the schema matches the applied history.

```bash
go run ./cmd/migrate adopt-goose --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate adopt-goose --driver postgres --dsn "$DATABASE_URL" \
  --apply --plan FINGERPRINT_FROM_PREVIEW --verified-schema --yes
```

Flags: `--goose-table` (default `goose_db_version`), `--apply`, `--plan`, and `--verified-schema`. See [Goose import](goose-import.md#adopt-an-existing-goose-database) for the required sequence.

## `recover`

Resolves an interrupted PostgreSQL non-transactional migration after an operator has inspected its SQL effects. The command changes only history metadata.

```bash
go run ./cmd/migrate recover 006 --state applied --verified --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate recover 006 --state not-applied --verified --driver postgres --dsn "$DATABASE_URL"
```

`--state` accepts `applied` or `not-applied`; `--verified` confirms the schema was inspected.

## `verify-registry`

Checks that migrations declared with literal versions in the source directory are registered in the built migrator. It does not connect to a database.

```bash
./migrate verify-registry --dir migrations
```

Go-function migrations need `ManualChecksum` for this check.

## `create`

Creates a new migration file in `migrations/`.

```bash
go run ./cmd/migrate create create_users --type sql
go run ./cmd/migrate create backfill_profiles --type go
```

Flag:

| Flag | Default | Description |
| --- | --- | --- |
| `--type` | `sql` | `sql` or `go`. |

Current behavior: `create` writes to `migrations/`, adds the new migration to the conventional generated registry, and derives the next version from filenames and `.queen.yaml` naming config when present. Custom registries still need manual registration; run `verify-registry` in CI.

Limitation: semver naming config exists, but `create` does not expose a manual `--version` flag yet. See [Known Limitations](known-limitations.md#create-and-naming).

## `init`

Scaffolds a Queen migrator. In a new project, run it through the temporary entry point shown in [New project](workflows.md#new-project); `cmd/migrate` does not exist until `init` creates it.

```bash
go run ./queen-bootstrap.go init --driver postgres --with-config
go run ./queen-bootstrap.go init --interactive
go run ./queen-bootstrap.go init --migrations-dir db/migrations
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--driver` | `postgres` | Driver for generated config examples. |
| `--with-config` | `false` | Create `.queen.yaml`. |
| `--interactive` | `false` | Launch the interactive setup wizard. |
| `--migrations-dir` | `migrations` | Directory for generated migration package. |

`init` refuses to overwrite files that already exist.

## `tui`

Launches the terminal UI.

```bash
go run ./cmd/migrate tui --driver postgres --dsn "$DATABASE_URL"
```

Use it for local inspection, not as the main production deployment path.
