---
id: custom-drivers
title: Custom Drivers
---

# Custom Drivers

Database drivers implement `queen.Driver`:

```go
type Driver interface {
    Init(ctx context.Context) error
    GetApplied(ctx context.Context) ([]Applied, error)
    Record(ctx context.Context, m *Migration, meta *MigrationMetadata) error
    Remove(ctx context.Context, version string) error
    Lock(ctx context.Context, timeout time.Duration) error
    Unlock(ctx context.Context) error
    Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error
    Close() error
}
```

`Init` prepares history storage. `GetApplied`, `Record`, and `Remove` read and update it. `Lock` and `Unlock` coordinate concurrent migrators. `Exec` runs migration code using the driver's transaction semantics and requested isolation level where supported. `Close` releases resources.

Implement `TransactionalRecorder` if the database can write a history row inside the same transaction as the migration body. Its `RecordTx` and `RemoveTx` methods let SQL and history commit or roll back together. Optional `MigrationExecutor` supports per-migration SQL timeouts; `NonTransactionalExecutor` supports single-command SQL outside a transaction with recovery markers. `SQLDBProvider` exposes the underlying `*sql.DB` for diagnostics.

Use the [PostgreSQL driver](drivers/postgres.md) as the reference for connection-pinned locking and transactional recording. Describe any weaker guarantees in your driver's documentation so deploy jobs can account for them.
