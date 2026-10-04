---
id: architecture
title: Architecture
---

# Architecture

Queen is a Go-first migration library with an embedded CLI. The main design choice is that migrations are registered by your application code, not discovered by a global external process.

## Runtime shape

```text
your cmd/migrate binary
  -> migrations.Register(q)
  -> queen.Queen
  -> database driver
  -> migration lock
  -> migration transaction
  -> migration record
```

The migrator binary is built from the same Go module as the application. That gives releases a stable migration set: the binary can only run the migrations that were compiled into it.

## Core pieces

| Piece | Responsibility |
| --- | --- |
| `queen.M` | One migration: version, name, SQL, function hooks, rollback, checksum policy. |
| `queen.Queen` | Registry, validation, planning, execution, rollback, status, checksum checks. |
| Driver | Database-specific SQL, locking, transaction behavior, migration table storage. |
| Embedded CLI | Operational commands around the library API. |
| TUI | Local inspection surface for status, plans, and tap events. |
| TAP | SQL observation layer for Go-function migrations and live debugging. |

## Migration lifecycle

1. Register migrations in order-independent Go code.
2. Queen validates versions, names, duplicates, and checksums.
3. A command asks for a plan: pending migrations for `up`, applied migrations for `down`, or a target for `goto`.
4. The driver acquires its migration lock.
5. Queen executes each migration.
6. Queen writes or removes the migration record.
7. The driver releases the lock.

PostgreSQL, SQLite, CockroachDB, and MSSQL write transactional migration records with the body. MySQL and ClickHouse record metadata separately. PostgreSQL's opt-in non-transactional SQL uses recovery markers; see [Support Matrix](support-matrix.md).

## Why embed the CLI?

The CLI is not a separate product that scans whatever files happen to be on disk. It is a package you put in `cmd/migrate`:

```go
package main

import (
    "github.com/dmedovich/queen/cli"
    "myapp/migrations"
)

func main() {
    cli.Run(migrations.Register)
}
```

That gives you normal operational commands without separating migration code from release code:

- `plan` before a deploy;
- `check --ci --no-gaps` in CI;
- `up --yes` in controlled release jobs;
- `doctor` during incident diagnosis;
- `baseline`, `squash`, and `import` for history maintenance.

If your application only needs "run all pending migrations on startup", call the library API directly.

## Why the TUI exists

The TUI is for humans looking at a live system or a hard migration locally. It helps with:

- scanning applied, pending, and modified migrations;
- viewing plan details;
- inspecting SQL migrations;
- watching tap events for Go-function migrations.

It is not required for CI/CD.

## Migration package shape

A typical project has one package that owns migration registration:

```text
migrations/
  migrations.go
  001_create_users.go
  002_add_user_slug.go
  003_backfill_profiles.go
cmd/
  migrate/
    main.go
```

`migrations.go` exposes one function:

```go
package migrations

import "github.com/dmedovich/queen"

func Register(q *queen.Queen) {
    q.MustAdd(CreateUsers)
    q.MustAdd(AddUserSlug)
    q.MustAdd(BackfillProfiles)
}
```

Each migration file owns one or a small set of related `queen.M` values. That keeps review focused and avoids a single giant migration registry file.
