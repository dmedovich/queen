---
id: config-files
title: Config Files
---

# Config Files

Queen can work entirely from CLI flags, but application repositories often keep two optional files:

- `.queen.yaml` for environment configuration;
- `.queenignore` for intentional migration gaps.

The Queen repository includes example copies:

- `.queen.yaml.example`
- `.queenignore.example`

The site's [`examples/config`](https://github.com/dmedovich/queen/tree/main/website/examples/config) directory also includes copy-paste samples.

## `.queen.yaml`

Use `.queen.yaml` when you want stable environment names instead of repeating `--driver`, `--dsn`, `--table`, and `--timeout`.

```yaml title=".queen.yaml"
config_locked: false

naming:
  pattern: sequential-padded
  padding: 3
  enforce: true

development:
  driver: postgres
  dsn: "postgres://localhost/myapp?sslmode=disable"
  table: queen_migrations
  lock_timeout: 5m

test:
  driver: sqlite
  dsn: "./test.db?_journal_mode=WAL"
  table: queen_migrations
  lock_timeout: 1m

production:
  driver: postgres
  dsn: "${DATABASE_URL}"
  table: queen_migrations
  lock_timeout: 30m
  require_confirmation: true
  require_explicit_unlock: true
```

Run with:

```bash
go run ./cmd/migrate status --use-config --env development
go run ./cmd/migrate up --use-config --env production --unlock-production --yes
```

Notes:

- Environment names are top-level YAML keys.
- `config_locked: true` makes the CLI refuse to load the file until it is unlocked.
- `require_confirmation` enables prompts.
- `require_explicit_unlock` requires `--unlock-production`.
- DSN passwords are redacted in confirmation prompts.

## `.queenignore`

`.queenignore` is for intentional migration gaps. It is read by gap detection and doctor checks.

```text title=".queenignore"
# Format: version # reason

003 # removed before the first production release
20240524054622 # legacy goose migration intentionally skipped
```

Prefer the CLI when editing it:

```bash
go run ./cmd/migrate gap ignore 003 --reason "removed before first release"
go run ./cmd/migrate gap list-ignored
go run ./cmd/migrate gap unignore 003
```

Do not use `.queenignore` to hide checksum drift or failed migrations. It is only for gaps you intentionally accept.
