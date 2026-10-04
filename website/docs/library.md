---
id: library
title: Use as a Library
---

# Use Queen as a Library

Use the library API when migrations should run from your application startup, tests, or an internal deploy tool.

## Create a Queen instance

```go
db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}
defer db.Close()

q := queen.New(postgres.New(db))
defer q.Close()
```

`queen.New` accepts options:

```go
q := queen.New(
    postgres.New(db),
    queen.WithLogger(logger),
    queen.WithTap(tap.NewJSONSink(os.Stdout)),
)
```

## Configure behavior

```go
q := queen.NewWithConfig(postgres.New(db), &queen.Config{
    TableName:   "queen_migrations",
    LockTimeout: 10 * time.Minute,
    SkipLock:    false,
})
```

Use `SkipLock: true` only for controlled single-runner cases. For production PostgreSQL, keep locking enabled.

Config fields:

| Field | Default | Purpose |
| --- | --- | --- |
| `TableName` | `queen_migrations` | Migration metadata table. |
| `LockTimeout` | `30m` | Driver lock acquisition timeout. |
| `SkipLock` | `false` | Skip driver lock acquisition. |
| `Naming` | `nil` | Optional version naming policy. |
| `IsolationLevel` | `sql.LevelDefault` | Default transaction isolation level. |
| `AllowUnknownApplied` | `false` | Permit applied database versions absent from this binary's registry during an intentional rolling deployment. |
| `AllowOutOfOrder` | `false` | Permit applying an older pending version after a newer version. |

## Register migrations

```go
func Register(q *queen.Queen) {
    q.MustAdd(queen.M{
        Version: "001",
        Name:    "create_users",
        UpSQL:   `CREATE TABLE users (id SERIAL PRIMARY KEY, email TEXT NOT NULL);`,
        DownSQL: `DROP TABLE users;`,
    })
}
```

`MustAdd` is convenient during program startup. Use `Add` if you want to return registration errors instead of panicking.

## Apply and rollback

```go
err := q.Up(ctx)          // apply all pending migrations
err = q.UpSteps(ctx, 2)   // apply up to 2 pending migrations
err = q.Down(ctx, 1)      // rollback one migration
err = q.Reset(ctx)        // rollback all applied migrations
```

`Down(ctx, n <= 0)` rolls back exactly one latest migration. Use `Reset(ctx)` for "everything".

## Inspect before running

```go
plans, err := q.DryRun(ctx, queen.DirectionUp, 0)
if err != nil {
    return err
}

for _, p := range plans {
    fmt.Printf("%s %s destructive=%v\n", p.Version, p.Name, p.IsDestructive)
}
```

`Validate(ctx)` checks registration validity, applied checksums, incomplete migrations, unknown applied versions, and migration order before you run migrations. By default, `Up`, `Down`, and `Reset` also stop on unsafe applied history.

## Runtime API

| Method | Purpose |
| --- | --- |
| `Add(m queen.M) error` | Register one migration. |
| `MustAdd(m queen.M)` | Register one migration and panic on invalid input. |
| `Up(ctx)` | Apply all pending migrations. |
| `UpSteps(ctx, n)` | Apply up to `n` pending migrations; `n <= 0` means all. |
| `Down(ctx, n)` | Roll back `n` latest migrations; `n <= 0` means one. |
| `Reset(ctx)` | Roll back all applied migrations. |
| `Status(ctx)` | Return status for all registered migrations. |
| `Validate(ctx)` | Validate registration and checksum drift. |
| `DryRun(ctx, direction, limit)` | Return migration plans without executing. |
| `Explain(ctx, version)` | Return detailed plan for one migration. |
| `FindMigration(version)` | Return a defensive copy of one registered migration. |
| `RegisteredMigrations()` | Return copies of registered migrations in version order. |
| `ResolveIncomplete(ctx, version, state)` | Resolve an interrupted non-transactional migration after inspecting the database. |
| `Driver()` | Return the underlying driver. |
| `SQLDB()` | Return the underlying `*sql.DB` when the driver exposes it. |
| `Close()` | Close driver resources. |

Long-running public operations on a single `Queen` instance are serialized internally. For production deploys, still run one intentional migrator job per environment.

## Status values

`Status(ctx)` reports:

| Status | Meaning |
| --- | --- |
| `pending` | Registered in code, not recorded in the database. |
| `applied` | Registered and recorded with matching checksum. |
| `modified` | Applied, but the registered checksum differs from the stored checksum. |
| `incomplete` | A non-transactional migration needs manual inspection and recovery. |

`modified` and `incomplete` block migration execution until you resolve the problem.

## pgxpool adapter

If your app already owns a `pgxpool.Pool`, use the adapter:

```go
pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
if err != nil {
    return err
}
defer pool.Close()

q := queen.New(postgres.NewFromPool(pool))
defer q.Close()
```

Closing Queen closes the `database/sql` adapter handle, not the caller-owned pool.

## Testing helper

Queen includes a small test helper:

```go
func TestMigrations(t *testing.T) {
    driver := setupTestDriver(t)
    q := queen.NewTest(t, driver)

    migrations.Register(q.Queen)
    q.TestUpDown()
}
```

Useful helper methods include `MustUp`, `MustDown`, `MustReset`, `MustValidate`, `TestUpDown`, and `TestRollback`.
