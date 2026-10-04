---
id: quick-start
title: Quick Start
---

# Quick Start

Install Queen:

```bash
go get github.com/dmedovich/queen
```

Queen currently requires Go `1.26.3+`.

## Run one migration

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
        Name:    "create_users",
        UpSQL: `
            CREATE TABLE users (
                id SERIAL PRIMARY KEY,
                email TEXT NOT NULL UNIQUE,
                created_at TIMESTAMP NOT NULL DEFAULT NOW()
            );
        `,
        DownSQL: `DROP TABLE users;`,
    })

    if err := q.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Check status

```go
statuses, err := q.Status(context.Background())
if err != nil {
    log.Fatal(err)
}

for _, s := range statuses {
    log.Printf("%s %s %s", s.Version, s.Name, s.Status)
}
```

## Add a migrator command

After registering migrations in a `myapp/migrations` package, create `cmd/migrate/main.go`:

```go
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

Run it:

```bash
go run ./cmd/migrate status --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate plan --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --driver postgres --dsn "$DATABASE_URL" --yes
```

To scaffold both packages automatically, follow [New project](workflows.md#new-project).

Next reads:

- [Migration Format](migrations.md) for SQL, Go-function, and mixed migrations.
- [Database Examples](database-examples.md) for driver-specific SQL.
- [Support Matrix](support-matrix.md) before choosing a non-PostgreSQL production driver.
