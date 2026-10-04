<p align="center">
  <img src="assets/queen_logo.png" alt="Queen" width="200">
</p>

<h1 align="center">Queen</h1>

<p align="center">
  Lightweight database migration library for Go.<br>
  Define migrations in Go, not SQL files.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/dmedovich/queen"><img src="https://pkg.go.dev/badge/github.com/dmedovich/queen.svg" alt="Go Reference"></a>
  <a href="https://github.com/dmedovich/queen/actions/workflows/test.yml"><img src="https://github.com/dmedovich/queen/actions/workflows/test.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/dmedovich/queen/actions/workflows/integration-tests.yml"><img src="https://github.com/dmedovich/queen/actions/workflows/integration-tests.yml/badge.svg" alt="Integration Tests"></a>
  <a href="https://goreportcard.com/report/github.com/dmedovich/queen"><img src="https://goreportcard.com/badge/github.com/dmedovich/queen" alt="Go Report Card"></a>
  <a href="https://github.com/dmedovich/queen/releases"><img src="https://img.shields.io/github/v/release/dmedovich/queen" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="License"></a>
</p>

## Features

- **Migrations as Go code** — type-safe structs validated at compile time, no separate `.sql` files to ship.
- **SQL or Go functions** — pure SQL for schema, Go functions for data transformations. SQL and Go parts can be combined in one migration.
- **PostgreSQL-first production path** — PostgreSQL is the reference driver for advisory locking, transactional migrations, and atomic migration records.
- **6 databases** — PostgreSQL, MySQL, SQLite, ClickHouse, CockroachDB, MS SQL Server, with driver-specific guarantees.
- **Embeddable CLI + TUI** — wire the CLI into your own binary alongside your migrations (see [CLI](#cli)).
- **Migration locking** — database-backed locking with configurable timeout; see per-driver guarantees below.
- **Checksum validation** — SHA-256 over SQL detects modifications to already-applied migrations and blocks `up`, `down`, and `reset` before they run.
- **Gap detection** — finds missing, skipped, or unregistered migrations.
- **Dry run / plan / explain** — preview what will run before touching the database.
- **Naming patterns** — optional enforcement for sequential, padded sequential, or semver migration versions.
- **Configurable isolation** — set transaction isolation level globally or per migration.
- **Rich metadata** — records who applied each migration, when, on which host, in which environment, and how long it took.
- **Migration toolkit** — `squash`, `baseline`, and `import` commands to consolidate history or onboard an existing database, including goose SQL migrations.
- **Migration tap** — stream every migration's SQL, duration, rows, and errors to a pluggable sink. Live TUI view via `queen up --tap`.
- **Rollback testing** — `queen check --rollback-test` runs an opt-in `up -> reset -> up` cycle against a clean test database.

## Installation

```bash
go get github.com/dmedovich/queen
```

Requires Go 1.26.3+.

## Release Status And API Stability

Queen is being hardened for a PostgreSQL-first production release. The public API is small and intended to stay simple, but pre-1.0 releases may still make breaking changes when they improve safety, correctness, or performance.

Current stability policy:

- PostgreSQL behavior is the reference contract.
- Compile-time compatibility is preferred, but unsafe behavior may be changed even if some callers relied on it.
- Returned migration/config data should be treated as snapshots. Mutating it must not be used to reconfigure a running `Queen`.
- Non-PostgreSQL drivers are supported with the caveats documented in [Locking And Transaction Guarantees](#locking-and-transaction-guarantees).

Current known limitations:

- SQLite locking coordinates Queen instances only within the current process. Do not rely on it to coordinate multiple migrator processes.
- ClickHouse locking is best-effort and should be operationally serialized by your deployment system.
- CockroachDB retries `40001` serialization failures up to three times. Go-function migration callbacks can therefore run more than once and must be retry-safe.
- MySQL and ClickHouse record migration metadata separately from the migration body. PostgreSQL, CockroachDB, SQLite, and MSSQL record it in the same transaction.

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "log"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/postgres"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    db, err := sql.Open("pgx", "postgres://localhost/myapp?sslmode=disable")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    q := queen.New(postgres.New(db))
    defer q.Close()

    q.MustAdd(queen.M{
        Version: "001",
        Name:    "create_users_table",
        UpSQL: `
            CREATE TABLE users (
                id SERIAL PRIMARY KEY,
                email VARCHAR(255) NOT NULL UNIQUE,
                name VARCHAR(255),
                created_at TIMESTAMP DEFAULT NOW()
            )
        `,
        DownSQL: `DROP TABLE users`,
    })

    if err := q.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Go Function Migrations

When a change is more than schema, register a Go function instead of (or alongside) SQL. For transactional drivers such as PostgreSQL, SQL and Go-function migrations run inside the same transaction:

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_emails",
    ManualChecksum: "v1", // bump when the function logic changes
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx,
            `UPDATE users SET email = LOWER(TRIM(email))`)
        return err
    },
})
```

`UpSQL` and `UpFunc` can be combined in a single migration. Queen runs `UpSQL` before `UpFunc`. For rollback, `DownFunc` runs before `DownSQL`, so custom cleanup can happen before schema is dropped.

## CLI

Because migrations are Go code, Queen's CLI is shipped as a library you embed in your own `main.go` together with your migrations. This way the CLI always knows about the exact set of migrations your application ships with — no separate registry, no file scanning.

The CLI exists for operational workflows around the same migration registry you use in code: CI checks, release plans, production confirmation prompts, gap detection, baselining, squashing, goose import, and human inspection through the TUI. If your application only needs "run all pending migrations on startup", the library API is enough.

```go
// cmd/migrate/main.go
package main

import (
    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/cli"
)

func main() {
    cli.Run(func(q *queen.Queen) {
        q.MustAdd(queen.M{Version: "001", Name: "create_users", UpSQL: `...`, DownSQL: `...`})
        q.MustAdd(queen.M{Version: "002", Name: "add_index",    UpSQL: `...`, DownSQL: `...`})
    })
}
```

Then run any command against your project:

```bash
go run ./cmd/migrate up --driver postgres --dsn "postgres://localhost/myapp?sslmode=disable"
go run ./cmd/migrate status
go run ./cmd/migrate plan
```

Available commands: `up`, `down`, `reset`, `goto`, `status`, `log`, `plan`, `explain`, `validate`, `check`, `gap`, `diff`, `doctor`, `create`, `init`, `squash`, `baseline`, `import`, `tui`.

Configuration can also come from a `.queen.yaml` file (`--use-config`), with per-environment settings (`--env production`).

Example config files are included in the repository:

- `.queen.yaml.example` — environment config for the embedded CLI.
- `.queenignore.example` — ignored migration gap format.

### Recommended project layout

Keep migrations in a normal Go package and keep the CLI entrypoint thin:

```text
myapp/
  cmd/
    migrate/
      main.go
  migrations/
    migrations.go
    001_create_users.go
    002_add_user_slug.go
  internal/
  go.mod
```

`migrations/migrations.go` owns the registry:

```go
package migrations

import "github.com/dmedovich/queen"

func Register(q *queen.Queen) {
    Register001CreateUsers(q)
    Register002AddUserSlug(q)
}
```

Each migration file registers one focused change:

```go
package migrations

import "github.com/dmedovich/queen"

func Register001CreateUsers(q *queen.Queen) {
    q.MustAdd(queen.M{
        Version: "001",
        Name:    "create_users",
        UpSQL:   `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL UNIQUE);`,
        DownSQL: `DROP TABLE users;`,
    })
}
```

After `queen init`, `queen create add_email` creates the next migration file and adds it to `Register` in `migrations/migrations.go`. Projects with a custom registry receive a manual registration instruction. Run `verify-registry` in CI to catch a missing registration regardless of how the migration file was created.

Your app can import the same `migrations` package if you run migrations from application startup, and `cmd/migrate` can import it for release tooling.

### CLI toolkit

Onboard an existing database by marking registered migrations as already applied without executing their SQL:

```bash
queen baseline --at 010 --driver sqlite --dsn ./app.db
queen baseline --version 010 --dry-run
```

`baseline` writes migration-table records with `action=baseline`. Use it only when the database schema already matches the migrations you are marking.

Consolidate SQL migration history into a new generated migration file:

```bash
queen squash 001,002,003 --into initial_schema --dry-run
queen squash --from 001 --to 010 --into initial_schema
```

`squash` currently supports registered SQL migrations with SQL rollbacks. It creates the new squashed file and leaves old migration files in place for review.

Import goose SQL migrations into Queen's Go migration format:

```bash
queen import ./db/migrations --from goose --output migrations --dry-run
queen import ./db/migrations --from goose
```

Replace `queen` with your embedded migrator binary or `go run ./cmd/migrate` if you do not install a binary named `queen`.

Queen preserves the version prefix from each goose filename, including timestamp versions such as `20240524054622_create_users.sql`. The importer supports Goose SQL `Up`, optional `Down`, `StatementBegin`/`StatementEnd`, and single-command `NO TRANSACTION` migrations. Queen executes `NO TRANSACTION` migrations through its PostgreSQL driver. It rejects `ENVSUB`, malformed annotations, duplicate versions, and multi-command `NO TRANSACTION` files, since silently converting those would change execution semantics. Goose Go migrations are not converted automatically. Import writes files with exclusive create semantics and fails if a generated file already exists.

**Import converts files only.** For a database already migrated with Goose, stop all Goose migrators, register the generated Queen files in your migrator binary, and preview the history transfer:

```sh
queen adopt-goose --driver postgres --dsn "$DATABASE_URL" --goose-table goose_db_version
# Inspect the schema and version list printed by the preview, then use its plan fingerprint:
queen adopt-goose --driver postgres --dsn "$DATABASE_URL" --goose-table goose_db_version \
  --apply --plan PLAN_FINGERPRINT --verified-schema
queen status --driver postgres --dsn "$DATABASE_URL"
```

`adopt-goose` defaults to a read-only preview. Apply requires the exact fingerprint from a prior preview and `--verified-schema`; it rereads both history tables under locks and writes all Queen records in one transaction. It never executes migration SQL or modifies Goose's table. It accepts a contiguous prefix of applied Goose versions backed by registered SQL-only Queen migrations with computed checksums, and an empty Queen table (or an already identical adoption). It rejects gaps, unknown versions, mismatched Queen records, or a changed plan. Use `--goose-table schema.goose_db_version` if Goose used another schema. Goose stores numeric version IDs, so Queen versions such as `001` are matched to Goose version `1`.

## Dry Run

Inspect what `Up` or `Down` will do without applying anything:

```go
plans, _ := q.DryRun(ctx, queen.DirectionUp, 0)
for _, p := range plans {
    fmt.Printf("%s %s [%s] destructive=%v warnings=%v\n",
        p.Version, p.Name, p.Type, p.IsDestructive, p.Warnings)
}
```

## Checksums

SQL migrations are checksummed from their `UpSQL` and `DownSQL`. If an already-applied SQL migration is edited, Queen reports it as `modified` in `Status()` and `Explain()`, `Validate()` returns `ErrChecksumMismatch`, and `Up`, `Down`, and `Reset` fail before executing any migration work.

Go-function migrations cannot be hashed from source code at runtime. For those migrations, set `ManualChecksum` and bump it whenever the function logic changes:

```go
q.MustAdd(queen.M{
    Version:        "003",
    Name:           "backfill_user_slugs",
    ManualChecksum: "backfill-user-slugs-v2",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `UPDATE users SET slug = LOWER(name)`)
        return err
    },
})
```

## Migration Tap

Observe each migration as it runs — SQL text, duration, rows affected, and errors — without bolting on a separate proxy. Install a `tap.Sink`:

```go
import "github.com/dmedovich/queen/tap"

sink := tap.NewJSONSink(os.Stdout) // one JSON line per event
q := queen.New(postgres.New(db), queen.WithTap(sink))
```

For each migration the sink receives a `start` event, one `exec` event per captured SQL statement, and an `end` event with the total duration and any error.

SQL migrations emit a single `exec` event with the full SQL. Inside a Go-function migration, wrap the transaction to capture each statement:

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "backfill",
    ManualChecksum: "v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        t := tap.ObserveTx(ctx, tx) // no-op when tap is disabled
        if _, err := t.ExecContext(ctx, `UPDATE users SET email = LOWER(email)`); err != nil {
            return err
        }
        _, err := t.ExecContext(ctx, `UPDATE users SET name = TRIM(name)`)
        return err
    },
})
```

Built-in sinks: `NopSink`, `FuncSink`, `MultiSink`, `ChannelSink` (non-blocking, drops on overflow), `JSONSink`, `RecorderSink` (in-memory, for tests). Implement `tap.Sink` for custom destinations.

`tap.ObserveTx` does not create a transaction; Queen already passes your Go migration the active `*sql.Tx`. It only wraps that transaction so tap can observe `ExecContext`, `QueryContext`, `QueryRowContext`, and prepared statements. The older `tap.Tx` helper is still available as a deprecated compatibility alias.

Wrap any sink with `tap.NewAnalyzerSink` to add sql-tap-style diagnostics:

```go
sink := tap.NewAnalyzerSink(
    tap.NewJSONSink(os.Stdout),
    tap.DefaultAnalyzerConfig(), // slow >=100ms, N+1: 5 repeated SELECTs in 1s
)
q := queen.New(postgres.New(db), queen.WithTap(sink))
```

Analyzed `exec` events include:

- `operation` — first SQL keyword, such as `select`, `insert`, `create`
- `sql_template` — normalized SQL for grouping and N+1 detection
- `bound_sql` — SQL with positional args rendered for inspection
- `slow`, `n_plus_1`, `n_plus_1_count`, `n_plus_1_alert`
- `index` — statement number within the migration

The tap package also includes helpers for programmatic inspection:

```go
events := recorder.Events()

filter, _ := tap.ParseFilter("op:select d>100ms slow")
for _, e := range events {
    if filter.Match(e) {
        fmt.Println(e.BoundSQL)
    }
}

summary := tap.Summarize(events)
top := tap.TopQueries(events, "total", 10)
_ = summary
_ = top

_ = tap.WriteMarkdown(os.Stdout, events)
_ = tap.WriteJSONL(os.Stdout, events)
```

A live TUI view is wired into the CLI:

```bash
queen up --tap
```

The live view enables the analyzer by default. Tune it with
`--tap-slow-threshold` and `--tap-nplus1-threshold`.

The full-screen TUI is available as a separate command:

```bash
queen tui --driver postgres --dsn "$DATABASE_URL"
```

It shows migration status, gaps, details, SQL preview for SQL migrations, and tap/explain panels for inspected operations. Go-function migrations do not have static SQL preview; their executed SQL appears in tap when the function uses `tap.ObserveTx`.

## Migration Metadata

For each applied migration record, Queen can persist execution metadata alongside version, name, checksum, and timestamp. The built-in drivers support these fields:

- `applied_by` — current OS user when available
- `duration_ms` — migration execution time in milliseconds
- `hostname` — current machine hostname when available
- `environment` — value of `QUEEN_ENV`
- `action` — operation type such as `apply` or `mark-applied`
- `status` — operation result such as `success`
- `error_message` — optional error details when recorded by the driver flow

This table represents the current applied state of migrations. It is not a full append-only audit log of every migration event.

Use `Status()` when you need the current state in code, and `Driver().GetApplied()` when you want the persisted applied records including metadata.

## Driver Interface

Custom database drivers implement the `queen.Driver` interface:

```go
type Driver interface {
    Init(ctx context.Context) error
    GetApplied(ctx context.Context) ([]Applied, error)
    Record(ctx context.Context, m *Migration, meta *MigrationMetadata) error
    Remove(ctx context.Context, version string) error
    Lock(ctx context.Context, timeout time.Duration) error
    Unlock(ctx context.Context) error
    Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error
    Close() error
}
```

In practice:

- `Init` prepares the migration tracking tables or any driver-specific state
- `GetApplied` returns the persisted applied migration records
- `Record` persists a successful applied migration together with optional metadata
- `Remove` removes an applied migration record after rollback
- `Lock` and `Unlock` provide the driver's migration lock semantics
- `Exec` runs migration code using the driver's transactional semantics and requested isolation level where supported
- `Close` releases driver resources

If you are implementing a custom driver, use the PostgreSQL driver as the reference behavior for locking, metadata persistence, and transaction execution. Drivers that can write migration records inside the same transaction as the migration body can implement `TransactionalRecorder`.

## Locking And Transaction Guarantees

When multiple instances of your app or CI run migrations concurrently, Queen asks the active driver to acquire a database-backed lock:

```go
q := queen.NewWithConfig(driver, &queen.Config{
    TableName:   "queen_migrations",
    LockTimeout: 10 * time.Minute,
})
```

Set `SkipLock: true` for single-instance setups or local development.

Current release guarantees are intentionally Postgres-first:

| Database    | Locking guarantee | Migration transaction | Migration record atomic with body |
|-------------|-------------------|-----------------------|-----------------------------------|
| PostgreSQL  | Production-ready advisory lock pinned to the migration connection | Yes by default; opt-in non-transactional SQL runs outside a transaction | Yes by default; non-transactional SQL uses recovery markers instead |
| MySQL       | `GET_LOCK` pinned to one connection; the connection is always discarded after release | Depends on MySQL DDL implicit-commit semantics | No |
| SQLite      | Process-local lock across Queen instances; not a distributed or cross-process lock | Yes for transactional statements | Yes |
| ClickHouse  | Best-effort table lock with synchronous cleanup; deployment-level serialization is still required | No true transaction support | No |
| CockroachDB | Table-backed ownership lock | Yes; `40001` serialization failures are retried up to three times | Yes |
| MSSQL       | Session-owned application lock pinned to one connection | Yes for transactional statements | Yes |

Use PostgreSQL for production environments that require concurrent migrator safety and atomic migration bookkeeping.

For PostgreSQL production use:

- Use the `pgx` stdlib driver (`github.com/jackc/pgx/v5/stdlib`).
- Keep locking enabled. `SkipLock` is only for controlled single-runner situations.
- Prefer SQL migrations or Go functions that use the provided `*sql.Tx`.
- Keep `ManualChecksum` stable for Go-function migrations and bump it when the function logic changes.
- Let Queen record migrations through the PostgreSQL driver so the migration body and migration record commit atomically.

The PostgreSQL migration table can be placed in an existing schema with `postgres.NewWithTableName(db, "schema.queen_migrations")` or CLI `--table schema.queen_migrations`. Queen quotes each identifier component separately; create the schema before running migrations.

By default, `Up`, `Down`, `Reset`, and `Validate` reject applied versions missing from the registered migrations. `Up` and `Validate` also reject a pending version older than an already applied version. This prevents a stale application instance from changing an unfamiliar migration history. During a deliberate rolling deploy, set `Config.AllowUnknownApplied` (CLI `--allow-unknown-applied`) if old instances must coexist with newer migration records. Set `Config.AllowOutOfOrder` (CLI `--allow-out-of-order`) only when applying an older version is intentional.

If your application already uses native `pgxpool.Pool`, use the pool adapter:

```go
pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    log.Fatal(err)
}
defer pool.Close()

q := queen.New(postgres.NewFromPool(pool))
```

Queen uses pgx's `database/sql` adapter under the hood for this path. Closing `Queen` does not close the caller-owned `pgxpool.Pool`.

`Down(ctx, n <= 0)` rolls back exactly one latest applied migration. This is intentional for compatibility; use `Reset(ctx)` when you want to roll back everything.

Queen also validates applied checksums before `Down` and `Reset`, not just before `Up`. If the code for an applied migration has drifted, fix the drift or intentionally update the recorded state before rolling back.

### PostgreSQL statement limits and concurrent indexes

Set limits on a migration that might wait for table locks or run for a long time:

```go
q.MustAdd(queen.M{
    Version:          "004",
    Name:             "add_lookup_column",
    UpSQL:            `ALTER TABLE users ADD COLUMN lookup_key TEXT`,
    DownSQL:          `ALTER TABLE users DROP COLUMN lookup_key`,
    SQLLockTimeout:   5 * time.Second,
    StatementTimeout: 2 * time.Minute,
})
```

`SQLLockTimeout` maps to PostgreSQL `lock_timeout` and is separate from `Config.LockTimeout`, which limits waiting for Queen's advisory lock. `StatementTimeout` maps to PostgreSQL `statement_timeout`. Both default to zero, leaving the connection's configured values in effect. PostgreSQL applies these settings locally to the migration transaction.

For commands such as `CREATE INDEX CONCURRENTLY`, opt into non-transactional SQL:

```go
q.MustAdd(queen.M{
    Version:          "005",
    Name:             "index_users_email",
    UpSQL:            `CREATE INDEX CONCURRENTLY users_email_idx ON users (email)`,
    DownSQL:          `DROP INDEX CONCURRENTLY users_email_idx`,
    NonTransactional: true,
    SQLLockTimeout:   5 * time.Second,
    StatementTimeout: 10 * time.Minute,
})
```

Use one SQL command per non-transactional migration. This mode is PostgreSQL-only and does not accept Go callbacks. Queen records `status=applying` before the up command and `status=rolling_back` before the down command. It marks a successful up as `success` or removes the record after a successful down. If a command or metadata update fails, the non-success status remains and later `Up`, `Down`, `Reset`, and `Validate` stop with `ErrIncompleteMigration`. `Status` displays `incomplete`.

After an interruption, run `queen doctor` and inspect PostgreSQL to determine whether the SQL command took effect. Then resolve the record explicitly:

```sh
queen recover 001 --driver postgres --dsn "$DATABASE_URL" --state applied --verified
# Or, only when the command's effects are absent:
queen recover 001 --driver postgres --dsn "$DATABASE_URL" --state not-applied --verified
```

`recover` changes only the migration record; it never runs SQL. It requires the original migration to remain registered with its original checksum. A `not-applied` resolution permits a later `up` retry. The `--verified` flag confirms that you inspected the database and chose the matching state.

Before retrying, inspect the database object and the row in `queen_migrations`. For an interrupted up: after confirming the SQL completed, set its status to `success`; if it did not complete, clean up any partial object and delete the row before retrying. For an interrupted down: delete the row if the rollback completed, or restore `status=success` if it did not. For example, a failed concurrent index build can leave an invalid index that needs to be dropped before retrying. Non-transactional SQL and its history record cannot commit atomically.

## CI/CD

Queen migrations are Go code, so the release artifact for migrations is a small Go binary, not a directory of SQL files consumed by a global CLI.

Build the migrator once in CI and use the same binary in the test and deployment jobs. Set `QUEEN_DSN` from the job's database secret (and `QUEEN_DRIVER=postgres`), so the connection string does not appear in the command arguments.

```bash
go test ./...
go build -o queen-migrate ./cmd/migrate
./queen-migrate verify-registry --dir migrations
QUEEN_DSN="$TEST_DATABASE_URL" ./queen-migrate check --rollback-test --no-gaps
```

The rollback test requires a clean, disposable test database. It applies all migrations, rolls them back with `Reset`, then applies them again.

In the deployment job, use the binary built in CI with `QUEEN_DSN` set to the target database:

```bash
./queen-migrate validate
./queen-migrate plan
./queen-migrate up --yes
./queen-migrate check --ci --no-gaps
./queen-migrate status
```

`check --ci` fails while any registered migration is pending, so run it after `up`. The pre-deploy `validate` checks the existing history against the release binary and still permits new migrations.

`verify-registry` needs only the Go source directory and the built binary; it never connects to the database. It compares literal `Version` values in `queen.M`/`queen.Migration` declarations with the versions registered in that binary, catching forgotten registration calls and stale binaries. It also requires `ManualChecksum` for Go-function migrations. Run it while sources are present in CI. Keep migration versions as string literals; dynamically computed versions cannot be verified.

For a complete PostgreSQL GitHub Actions example, see [CI/CD guide](docs/ci-cd.md).

This works well as:

- a dedicated Kubernetes Job before rolling out the app;
- a CI/CD deploy step with database credentials scoped only to the migration job;
- a release image command such as `/app/migrate up --yes`;
- a local developer command through `go run ./cmd/migrate ...`.

Keep exactly one migrator job active per environment. PostgreSQL advisory locking protects against accidental concurrency, but the deployment system should still model migrations as a single explicit step.

## Supported Databases

| Database    | Driver                      | SQL Driver                               | Integration target |
|-------------|-----------------------------|------------------------------------------|--------------------|
| PostgreSQL  | `queen/drivers/postgres`    | `github.com/jackc/pgx/v5/stdlib`         | PostgreSQL 15      |
| MySQL       | `queen/drivers/mysql`       | `github.com/go-sql-driver/mysql`         | MySQL 8.0          |
| SQLite      | `queen/drivers/sqlite`      | `github.com/mattn/go-sqlite3`            | Driver bundled SQLite |
| ClickHouse  | `queen/drivers/clickhouse`  | `github.com/ClickHouse/clickhouse-go/v2` | Latest container image |
| CockroachDB | `queen/drivers/cockroachdb` | `github.com/jackc/pgx/v5/stdlib`         | Latest container image |
| MSSQL       | `queen/drivers/mssql`       | `github.com/microsoft/go-mssqldb`        | SQL Server 2022    |

The table lists what Queen's integration tests exercise today; it is not a formal minimum-version matrix for every database. PostgreSQL 15 is the primary release target.

Run the same driver-specific suites used by CI with:

```bash
make test-postgres
make test-mysql
make test-sqlite
make test-cockroachdb
make test-clickhouse
make test-mssql
```

## Documentation

Deployment examples: [CI/CD guide](docs/ci-cd.md).

## License

Apache License 2.0
