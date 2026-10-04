---
id: configuration
title: Configuration
---

# Configuration

Configuration exists at two levels:

- library configuration through `queen.Config`;
- CLI configuration through flags, environment variables, or `.queen.yaml`.

Sample files are available in [Config Files](config-files.md).

## Library config

```go
q := queen.NewWithConfig(driver, &queen.Config{
    TableName:      "queen_migrations",
    LockTimeout:    30 * time.Minute,
    SkipLock:       false,
    IsolationLevel: sql.LevelSerializable,
    Naming: &queen.NamingConfig{
        Pattern: queen.NamingPatternSequentialPadded,
        Padding: 3,
        Enforce: true,
    },
})
```

## CLI flags

```bash
go run ./cmd/migrate up \
  --driver postgres \
  --dsn "$DATABASE_URL" \
  --table queen_migrations \
  --timeout 30m \
  --yes
```

Global CLI flags are documented in [CLI Reference](cli-reference.md#global-options).

## Environment-specific config

```yaml
config_locked: false

development:
  driver: sqlite
  dsn: "./app.db?_journal_mode=WAL"
  table: queen_migrations
  lock_timeout: 5m

production:
  driver: postgres
  dsn: "${DATABASE_URL}"
  table: queen_migrations
  lock_timeout: 30m
  require_confirmation: true
  require_explicit_unlock: true
```

Use `config_locked: true` when you want the config file to act as a safety stop instead of an active source. Queen will refuse to load it until the lock is removed.

## Environment variables

The CLI reads these values when the matching flag is not set:

| Variable | Equivalent flag |
| --- | --- |
| `QUEEN_DRIVER` | `--driver` |
| `QUEEN_DSN` | `--dsn` |
| `QUEEN_TABLE` | `--table` |

`QUEEN_ENV` is stored in migration metadata when drivers persist metadata.

## Production prompts

When the selected config environment has:

```yaml
require_confirmation: true
require_explicit_unlock: true
```

Queen requires `--unlock-production` and asks for confirmation before destructive operations. DSN passwords are redacted in prompts.
