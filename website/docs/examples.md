---
id: examples
title: Examples
---

# Examples

For database-specific versions of these patterns, see [Database Examples](database-examples.md).

## Create a table

```go
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
```

## Add a nullable column, backfill, then constrain

```go
q.MustAdd(queen.M{
    Version: "002",
    Name:    "add_user_slug",
    UpSQL: `
        ALTER TABLE users ADD COLUMN slug TEXT;
        UPDATE users SET slug = LOWER(REPLACE(email, '@', '-'));
        ALTER TABLE users ALTER COLUMN slug SET NOT NULL;
        CREATE UNIQUE INDEX users_slug_idx ON users (slug);
    `,
    DownSQL: `
        DROP INDEX users_slug_idx;
        ALTER TABLE users DROP COLUMN slug;
    `,
})
```

## Backfill with Go

```go
q.MustAdd(queen.M{
    Version:        "003",
    Name:           "trim_user_emails",
    ManualChecksum: "trim-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `UPDATE users SET email = LOWER(TRIM(email))`)
        return err
    },
})
```

## Observe SQL from a Go function

```go
q.MustAdd(queen.M{
    Version:        "004",
    Name:           "backfill_profiles",
    ManualChecksum: "backfill-profiles-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        t := tap.ObserveTx(ctx, tx)
        _, err := t.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, email FROM users
        `)
        return err
    },
})
```

## Run a release migrator

```bash
go build -o queen-migrate ./cmd/migrate
./queen-migrate check --driver postgres --dsn "$DATABASE_URL" --ci --no-gaps
./queen-migrate plan --driver postgres --dsn "$DATABASE_URL"
./queen-migrate up --driver postgres --dsn "$DATABASE_URL" --yes
```
