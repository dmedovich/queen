---
id: cli
title: Embedded CLI
---

# Embedded CLI

Queen migrations are Go code, so the CLI is a package you embed in your own binary. This keeps the migration list tied to the release artifact that runs it.

Use the CLI when you need operational workflows around the library API: CI checks, release plans, production confirmations, gap detection, baselining, squashing, goose import, and TUI inspection. If your app only needs "apply all pending migrations on startup", the library API is enough.

## Minimal command

```go title="cmd/migrate/main.go"
package main

import (
    "github.com/dmedovich/queen/cli"
    "myapp/migrations"

    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    cli.Run(migrations.Register)
}
```

## Common commands

```bash
go run ./cmd/migrate status --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate plan --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate validate --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --driver postgres --dsn "$DATABASE_URL" --yes
go run ./cmd/migrate down --steps 1 --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate doctor --driver postgres --dsn "$DATABASE_URL"
```

For every command and flag, see [CLI Reference](cli-reference.md). For end-to-end recipes, see [Workflows](workflows.md).

## Command groups

| Group | Commands | Why they exist |
| --- | --- | --- |
| Migration execution | `up`, `down`, `reset`, `goto` | Move the database between registered migration versions. |
| Inspection | `status`, `version`, `log`, `plan`, `explain`, `diff`, `validate`, `doctor`, `check`, `verify-registry` | Understand what is applied, what will run, and whether it is safe. |
| Maintenance | `gap`, `baseline`, `squash`, `import`, `adopt-goose`, `recover` | Repair or reshape migration history. |
| Project setup | `init`, `create` | Scaffold a migrator and new migration files. |
| Interactive | `tui`, `up --tap` | Inspect migration state and live SQL in a terminal UI. |

## Configuration file

Queen can load `.queen.yaml` with `--use-config`:

```yaml title=".queen.yaml"
config_locked: false

naming:
  pattern: sequential-padded
  padding: 3
  enforce: true

development:
  driver: postgres
  dsn: "postgres://localhost/myapp?sslmode=disable"
  table: queen_migrations
  lock_timeout: 5m

production:
  driver: postgres
  dsn: "${DATABASE_URL}"
  table: queen_migrations
  lock_timeout: 30m
  require_confirmation: true
  require_explicit_unlock: true
```

Run with:

```bash
go run ./cmd/migrate status --use-config --env development
go run ./cmd/migrate up --use-config --env production --unlock-production
```

Production confirmation redacts DSN passwords before printing.

The repository ships `.queen.yaml.example` and `.queenignore.example`. See [Config Files](config-files.md) for the full format.

## Global flags

These flags are accepted by the root command:

| Flag | Purpose |
| --- | --- |
| `--driver` | Queen driver: `postgres`, `mysql`, `sqlite`, `clickhouse`, `cockroachdb`, or `mssql`. |
| `--dsn` | Database connection string. |
| `--table` | Migration table name. Defaults to `queen_migrations`. |
| `--timeout` | Migration lock timeout, for example `30m` or `1h`. |
| `--use-config` | Load `.queen.yaml`. |
| `--env` | Environment name from `.queen.yaml`. |
| `--unlock-production` | Required when an environment has `require_explicit_unlock: true`. |
| `--yes` | Skip prompts for CI/CD. |
| `--json` | JSON output for commands that implement structured output. |
| `--verbose` | Reserved for verbose command output. |
| `--allow-unknown-applied` | Allow applied versions absent from this binary during an intentional rolling deployment. |
| `--allow-out-of-order` | Allow an older pending version after a newer applied version. |

`QUEEN_DRIVER`, `QUEEN_DSN`, and `QUEEN_TABLE` are also read when matching flags are empty.

## TUI

```bash
go run ./cmd/migrate tui --driver postgres --dsn "$DATABASE_URL"
```

The TUI shows status, gaps, migration details, SQL preview for SQL migrations, and tap output for inspected runs.

The TUI is not required for production deploys. It is a local/debugging cockpit for seeing migration state, inspecting plans, and watching tap events when a migration is hard to reason about from logs alone.
