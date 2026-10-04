<p align="center"><img src="assets/queen_logo.png" alt="Queen" width="180"></p>

# Queen

Queen is a Go library for database migrations defined in code. It runs SQL or Go functions, tracks applied versions and checksums, and provides an embeddable CLI for migration jobs. PostgreSQL is the reference production driver; MySQL, SQLite, ClickHouse, CockroachDB, and SQL Server have driver-specific guarantees.

## Install

```bash
go get github.com/dmedovich/queen
```

Requires Go 1.26.3 or newer.

## Example

```go
package main

import (
    "context"
    "database/sql"
    "log"
    "os"

    "github.com/dmedovich/queen"
    "github.com/dmedovich/queen/drivers/postgres"
    _ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    q := queen.New(postgres.New(db))
    defer q.Close()
    q.MustAdd(queen.M{
        Version: "001",
        Name: "create_users",
        UpSQL: `CREATE TABLE users (id BIGSERIAL PRIMARY KEY, email TEXT NOT NULL UNIQUE)`,
        DownSQL: `DROP TABLE users`,
    })

    if err := q.Up(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

For CI/CD, compile your migration registry into a dedicated binary and run it before the application rollout. The [quick start](https://dmedovich.github.io/queen/docs/quick-start) and [CI/CD guide](https://dmedovich.github.io/queen/docs/ci-cd) show the setup.

## Documentation

The [documentation site](https://dmedovich.github.io/queen/) covers the API, CLI, PostgreSQL behavior, Goose import, driver guarantees, and release notes. Its source is in [website](website/). See the [changelog](CHANGELOG.md) for release changes.

## License

Apache-2.0. See [LICENSE](LICENSE).
