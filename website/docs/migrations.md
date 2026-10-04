---
id: migrations
title: Migration Format
---

# Migration Format

Each migration is a `queen.M` value:

```go
q.MustAdd(queen.M{
    Version: "001",
    Name:    "create_users",
    UpSQL:   `CREATE TABLE users (id SERIAL PRIMARY KEY);`,
    DownSQL: `DROP TABLE users;`,
})
```

## Fields

| Field | Purpose |
| --- | --- |
| `Version` | Unique migration version. Common choices are `001`, `002`, or timestamp versions. |
| `Name` | Lowercase name using letters, digits, and underscores. |
| `UpSQL` | SQL applied during `Up`. |
| `DownSQL` | SQL applied during rollback. |
| `UpFunc` | Go function applied during `Up`. |
| `DownFunc` | Go function applied during rollback. |
| `ManualChecksum` | Required discipline for Go-function migrations. Bump when function logic changes. |
| `IsolationLevel` | Optional per-migration transaction isolation level. |
| `SQLLockTimeout` | PostgreSQL time spent waiting for a table or row lock. |
| `StatementTimeout` | PostgreSQL SQL execution limit for this migration. |
| `NonTransactional` | Opt into one PostgreSQL SQL command outside a transaction. |

Validation rules:

- `Version` is required and may contain letters, digits, dots, dashes, and underscores.
- `Name` is required, must be at most 63 characters, and uses lowercase letters, digits, and underscores.
- Each migration needs `UpSQL` or `UpFunc`.
- Duplicate versions are rejected at registration.

## SQL migrations

```go
queen.M{
    Version: "002",
    Name:    "add_user_index",
    UpSQL:   `CREATE INDEX users_email_idx ON users (email);`,
    DownSQL: `DROP INDEX users_email_idx;`,
}
```

SQL migrations are checksummed from `UpSQL` and `DownSQL`. If an applied SQL migration changes later, Queen reports it as modified and blocks migration commands until you handle the drift.

Whitespace is normalized before hashing, so basic SQL formatting changes do not cause a checksum mismatch.

## Go-function migrations

```go
queen.M{
    Version:        "003",
    Name:           "normalize_emails",
    ManualChecksum: "normalize-emails-v1",
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `UPDATE users SET email = LOWER(TRIM(email))`)
        return err
    },
}
```

Go functions cannot be hashed from source at runtime. Use `ManualChecksum` as a stable version marker and bump it whenever the function changes.

If a Go-function migration has no SQL and no `ManualChecksum`, Queen stores a no-checksum marker. For production migrations, prefer setting `ManualChecksum` so changes are explicit in review.

## Mixed migrations

You can combine SQL and Go:

```go
queen.M{
    Version: "004",
    Name:    "create_profiles_and_backfill",
    UpSQL:   `CREATE TABLE profiles (user_id BIGINT PRIMARY KEY, display_name TEXT);`,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO profiles (user_id, display_name)
            SELECT id, name FROM users
        `)
        return err
    },
    DownFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `DELETE FROM profiles`)
        return err
    },
    DownSQL: `DROP TABLE profiles;`,
}
```

Queen runs mixed migrations in this order:

- `Up`: `UpSQL`, then `UpFunc`
- `Down`: `DownFunc`, then `DownSQL`

That lets rollback cleanup data before removing schema objects.

## Naming

Use sequential padded versions when you want readable review order:

```yaml
naming:
  pattern: sequential-padded
  padding: 3
  enforce: true
```

Queen also supports sequential and semver naming policies.

## PostgreSQL statement limits and concurrent indexes

`SQLLockTimeout` maps to PostgreSQL `lock_timeout`; it differs from `Config.LockTimeout`, which limits waiting for Queen's advisory lock. `StatementTimeout` maps to PostgreSQL `statement_timeout`. Both settings are local to a transactional migration when nonzero.

For SQL such as `CREATE INDEX CONCURRENTLY`, use one command per non-transactional migration:

```go
queen.M{
    Version:          "006",
    Name:             "users_email_index",
    UpSQL:            `CREATE INDEX CONCURRENTLY users_email_idx ON users (email)`,
    DownSQL:          `DROP INDEX CONCURRENTLY users_email_idx`,
    NonTransactional: true,
}
```

Queen records an `applying` or `rolling_back` marker around the SQL command. If it fails or the process stops, later migration operations stop with `ErrIncompleteMigration`. Inspect the PostgreSQL schema and then use [`recover`](cli-reference.md#recover) with the matching state. Recovery changes only the migration record; it never runs SQL.

## Isolation level

Set a per-migration isolation level when one migration needs stronger or weaker transaction behavior:

```go
queen.M{
    Version:        "005",
    Name:           "critical_backfill",
    ManualChecksum: "critical-backfill-v1",
    IsolationLevel: sql.LevelSerializable,
    UpFunc: func(ctx context.Context, tx *sql.Tx) error {
        _, err := tx.ExecContext(ctx, `UPDATE users SET migrated = TRUE`)
        return err
    },
}
```

Driver support depends on the database. PostgreSQL is the reference path for transactional behavior.

## Rollback policy

Queen does not require every migration to have rollback SQL or a rollback function, but commands that roll back a migration will fail if rollback is missing. For production migrations, decide explicitly:

- provide `DownSQL` or `DownFunc` for reversible schema changes;
- document irreversible migrations in review and avoid workflows that depend on automatic rollback;
- use `check --rollback-test` against disposable databases when you want to verify rollback behavior.
