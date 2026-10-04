# Basic Queen Migrator

This example shows the recommended shape for application-owned migrations:

```text
basic-migrator/
  cmd/migrate/main.go
  migrations/migrations.go
  migrations/001_create_users.go
  migrations/002_add_user_slug.go
  migrations/003_backfill_profiles.go
```

This example compiles inside the Queen repository. When copying it into your application, change the migrations import in `cmd/migrate/main.go` to your application's module path and add `github.com/dmedovich/queen` as a dependency.

```bash
go run ./cmd/migrate status --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate plan --driver postgres --dsn "$DATABASE_URL"
go run ./cmd/migrate up --driver postgres --dsn "$DATABASE_URL" --yes
```
