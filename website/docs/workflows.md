---
id: workflows
title: Workflows
---

# Workflows

This page shows how the pieces fit together in real projects.

## New project

`init` is an embedded CLI command, so a new project needs a small bootstrap entry point before it has `cmd/migrate`. From the root of a Go module, create a temporary file:

```go title="queen-bootstrap.go"
package main

import (
    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/cli"
)

func main() {
    cli.Run(func(*queen.Queen) {})
}
```

Run it once to generate the permanent migrator and migration registry:

```bash
go get github.com/dmedovich/queen
go run ./queen-bootstrap.go init --driver postgres --with-config
rm ./queen-bootstrap.go
go mod tidy
```

Then edit:

- `cmd/migrate/main.go` to use your real module import path;
- `.queen.yaml` or your deploy flags;
- generated migration files under `migrations/`.

## Daily development

Create a migration:

```bash
go run ./cmd/migrate create add_user_slug --type sql
```

`create` registers the migration in the conventional generated registry. Review the file, then inspect the plan:

```bash
go run ./cmd/migrate validate --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate plan --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --driver postgres --dsn "$DATABASE_URL"
```

If the migration contains Go code, set `ManualChecksum` and bump it when the function changes.

## CI gate

Build and verify the release binary while sources are available:

```bash
go build -o migrate ./cmd/migrate
./migrate verify-registry --dir migrations
```

For an empty disposable test database:

```bash
QUEEN_DRIVER=postgres QUEEN_DSN="$TEST_DATABASE_URL" ./migrate check --rollback-test --no-gaps
```

## Production deploy

Build the migrator once:

```bash
go build -o migrate ./cmd/migrate
```

Run preflight checks:

```bash
./migrate validate --driver postgres --dsn "$DATABASE_URL"
./migrate plan --driver postgres --dsn "$DATABASE_URL"
```

Apply:

```bash
./migrate up --driver postgres --dsn "$DATABASE_URL" --yes
./migrate check --ci --no-gaps --json --driver postgres --dsn "$DATABASE_URL"
./migrate status --driver postgres --dsn "$DATABASE_URL"
```

For `.queen.yaml` production environments:

```bash
./migrate up --use-config --env production --unlock-production --yes
```

Keep one intentional migrator job active per environment.

## Existing database

If the database schema already exists and you want Queen to start tracking it:

```bash
go run ./cmd/migrate baseline --at 010 --dry-run --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate baseline --at 010 --driver postgres --dsn "$DATABASE_URL"
```

Baseline records migrations as applied without executing SQL. Use it only after checking that the schema actually matches those migrations.

## Import from goose

Preview first. Use a new output directory because the importer generates its own registry and will not overwrite the one created by `init`:

```bash
go run ./cmd/migrate import ./db/migrations --from goose --output imported_migrations --dry-run
```

Then import:

```bash
go run ./cmd/migrate import ./db/migrations --from goose --output imported_migrations
```

Review the generated Go files and merge their registrations into the application's migrator before committing. If Goose already migrated the database, use `adopt-goose` to preview and transfer its applied history before running Queen `up`; see [Goose import](goose-import.md).

## Clean up old migration history

Squash registered SQL migrations:

```bash
go run ./cmd/migrate squash 001,002,003 --into initial_schema --dry-run --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate squash 001,002,003 --into initial_schema --driver postgres --dsn "$DATABASE_URL"
```

Commit the generated migration after review. Do not delete old migrations from a live project until your release process accounts for databases that may still need them.

## Investigate a production issue

Start with non-mutating commands:

```bash
go run ./cmd/migrate status --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate doctor --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate gap detect --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate explain 010 --driver postgres --dsn "$DATABASE_URL"
```

For local inspection, use the TUI:

```bash
go run ./cmd/migrate tui --driver postgres --dsn "$DATABASE_URL"
```

For a live migration that needs SQL visibility:

```bash
go run ./cmd/migrate up --tap --driver postgres --dsn "$DATABASE_URL"
```
