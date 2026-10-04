---
id: project-structure
title: Project Structure
---

# Project Structure

Queen migrations are Go code, so the cleanest layout is a normal Go package for migrations plus a small CLI entrypoint.

```text
myapp/
  cmd/
    migrate/
      main.go
  migrations/
    migrations.go
    001_create_users.go
    002_add_user_slug.go
    003_backfill_profiles.go
  go.mod
```

## Migration registry

```go title="migrations/migrations.go"
package migrations

import "github.com/dmedovich/queen"

func Register(q *queen.Queen) {
    Register001CreateUsers(q)
    Register002AddUserSlug(q)
    Register003BackfillProfiles(q)
}
```

## SQL migration file

```go title="migrations/001_create_users.go"
package migrations

import "github.com/dmedovich/queen"

func Register001CreateUsers(q *queen.Queen) {
    q.MustAdd(queen.M{
        Version: "001",
        Name:    "create_users",
        UpSQL: `
            CREATE TABLE users (
                id BIGSERIAL PRIMARY KEY,
                email TEXT NOT NULL UNIQUE,
                created_at TIMESTAMP NOT NULL DEFAULT NOW()
            );
        `,
        DownSQL: `DROP TABLE users;`,
    })
}
```

## Go-function migration file

```go title="migrations/003_backfill_profiles.go"
package migrations

import (
    "context"
    "database/sql"

    "github.com/dmedovich/queen"
)

func Register003BackfillProfiles(q *queen.Queen) {
    q.MustAdd(queen.M{
        Version:        "003",
        Name:           "backfill_profiles",
        ManualChecksum: "backfill-profiles-v1",
        UpFunc:         up003BackfillProfiles,
    })
}

func up003BackfillProfiles(ctx context.Context, tx *sql.Tx) error {
    _, err := tx.ExecContext(ctx, `
        INSERT INTO profiles (user_id, display_name)
        SELECT id, email FROM users
    `)
    return err
}
```

## CLI entrypoint

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

This gives you one migration registry reused by:

- local development through `go run ./cmd/migrate ...`;
- CI/CD through a built migrator binary;
- application startup if your app calls Queen directly.

A complete example lives in [`website/examples/basic-migrator`](https://github.com/dmedovich/queen/tree/main/website/examples/basic-migrator). For the runtime model, see [Architecture](architecture.md); for database-specific SQL, see [Database Examples](database-examples.md).
