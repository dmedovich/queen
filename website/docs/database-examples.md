---
id: database-examples
title: Database Examples
---

# Database Examples

These examples show the same migration patterns across Queen's supported drivers:

- a clean SQL migration;
- a Go-function migration;
- a mixed migration where schema is SQL and data movement is Go.

Use the examples as starting points, then adjust SQL types and DDL to your database version.

## PostgreSQL

PostgreSQL is Queen's reference production path. It supports transactional DDL for common schema changes and Queen records the migration row in the same transaction as the migration body.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL: `
        CREATE TABLE users (
            id BIGSERIAL PRIMARY KEY,
            email TEXT NOT NULL UNIQUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
    `,
    DownSQL: `DROP TABLE users;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_user_emails",
    ManualChecksum: "normalize-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            UPDATE users
            SET email = LOWER(TRIM(email))
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_profiles",
    UpSQL: `
        CREATE TABLE profiles (
            user_id BIGINT PRIMARY KEY REFERENCES users(id),
            display_name TEXT NOT NULL
        );
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, split_part(email, '@', 1)
            FROM users
        `)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
})
```

## MySQL

MySQL uses `?` placeholders. DDL transaction behavior depends on the statement and MySQL version, so test rollback on the same engine you run in production.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL: `
        CREATE TABLE users (
            id BIGINT AUTO_INCREMENT PRIMARY KEY,
            email VARCHAR(320) NOT NULL UNIQUE,
            created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
    `,
    DownSQL: `DROP TABLE users;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_user_emails",
    ManualChecksum: "normalize-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            UPDATE users
            SET email = LOWER(TRIM(email))
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_profiles",
    UpSQL: `
        CREATE TABLE profiles (
            user_id BIGINT PRIMARY KEY,
            display_name VARCHAR(320) NOT NULL,
            CONSTRAINT profiles_user_id_fk FOREIGN KEY (user_id) REFERENCES users(id)
        );
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, SUBSTRING_INDEX(email, '@', 1)
            FROM users
        `)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
})
```

## SQLite

SQLite is best for local tools, tests, and single-process applications. Use `INTEGER PRIMARY KEY` for rowid-backed IDs.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL: `
        CREATE TABLE users (
            id INTEGER PRIMARY KEY,
            email TEXT NOT NULL UNIQUE,
            created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
    `,
    DownSQL: `DROP TABLE users;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_user_emails",
    ManualChecksum: "normalize-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            UPDATE users
            SET email = LOWER(TRIM(email))
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_profiles",
    UpSQL: `
        CREATE TABLE profiles (
            user_id INTEGER PRIMARY KEY,
            display_name TEXT NOT NULL,
            FOREIGN KEY (user_id) REFERENCES users(id)
        );
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, substr(email, 1, instr(email, '@') - 1)
            FROM users
        `)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
})
```

## ClickHouse

ClickHouse is column-oriented and does not behave like an OLTP database. Prefer append-oriented schema changes and serialize migrator runs outside Queen.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_events",
    UpSQL: `
        CREATE TABLE events (
            id UUID,
            user_id UInt64,
            event_name String,
            created_at DateTime
        )
        ENGINE = MergeTree()
        ORDER BY (created_at, user_id);
    `,
    DownSQL: `DROP TABLE events;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "backfill_event_names",
    ManualChecksum: "backfill-event-names-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            ALTER TABLE events
            UPDATE event_name = 'unknown'
            WHERE event_name = ''
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_events_materialized_view",
    UpSQL: `
        CREATE TABLE event_counts (
            event_name String,
            count UInt64
        )
        ENGINE = SummingMergeTree()
        ORDER BY event_name;
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO event_counts
            SELECT event_name, count()
            FROM events
            GROUP BY event_name
        `)
        return err
    },
    DownSQL: `DROP TABLE event_counts;`,
})
```

## CockroachDB

CockroachDB uses PostgreSQL-like SQL syntax, but serializable conflicts are normal and retryable. Deploy scripts should retry failed migration commands when the error is retryable.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL: `
        CREATE TABLE users (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            email STRING NOT NULL UNIQUE,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
        );
    `,
    DownSQL: `DROP TABLE users;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_user_emails",
    ManualChecksum: "normalize-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            UPDATE users
            SET email = lower(trim(email))
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_profiles",
    UpSQL: `
        CREATE TABLE profiles (
            user_id UUID PRIMARY KEY REFERENCES users(id),
            display_name STRING NOT NULL
        );
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, split_part(email, '@', 1)
            FROM users
        `)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
})
```

## MSSQL

MSSQL uses `@p1`, `@p2`, ... placeholders internally. Write SQL Server DDL and test rollback behavior on the same compatibility level you deploy.

### SQL migration

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL: `
        CREATE TABLE users (
            id BIGINT IDENTITY(1,1) PRIMARY KEY,
            email NVARCHAR(320) NOT NULL UNIQUE,
            created_at DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
        );
    `,
    DownSQL: `DROP TABLE users;`,
})
```

### Function migration

```go
q.MustAdd(queen.M{
    Version:        "002",
    Name:           "normalize_user_emails",
    ManualChecksum: "normalize-user-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            UPDATE users
            SET email = LOWER(LTRIM(RTRIM(email)))
        `)
        return err
    },
})
```

### Mixed migration

```go
q.MustAdd(queen.M{
    Version: "003",
    Name:    "add_profiles",
    UpSQL: `
        CREATE TABLE profiles (
            user_id BIGINT PRIMARY KEY,
            display_name NVARCHAR(320) NOT NULL,
            CONSTRAINT profiles_user_id_fk FOREIGN KEY (user_id) REFERENCES users(id)
        );
    `,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, LEFT(email, CHARINDEX('@', email + '@') - 1)
            FROM users
        `)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
})
```
