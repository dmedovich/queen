---
id: goose-import
title: Import Goose Migrations
---

# Import Goose Migrations

Queen can convert Goose SQL migrations into Go migration files. Conversion does not transfer the database's applied migration history.

```bash
go run ./cmd/migrate import ./db/migrations --from goose --output migrations --dry-run
go run ./cmd/migrate import ./db/migrations --from goose --output migrations
```

The importer supports `.sql` files with goose sections:

```sql
-- +goose Up
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;
```

Queen preserves the version prefix from the goose filename, including timestamp versions such as:

```text
20240524054622_create_users.sql
```

The converted output is a Go file that registers a `queen.M` with `UpSQL` and optional `DownSQL`. The importer also supports `StatementBegin`/`StatementEnd` and one-command `NO TRANSACTION` migrations for PostgreSQL. It rejects `ENVSUB`, malformed annotations, duplicate versions, and multi-command `NO TRANSACTION` files because converting them silently would change execution semantics.

Import does not overwrite generated files. If `migrations.go` or a target migration file already exists, the command fails and leaves the existing file intact.

Goose Go migrations are not converted automatically. Port them manually to `UpFunc` and `DownFunc`, and add `ManualChecksum`.

## Adopt an existing Goose database

Stop all Goose migrators. Register the generated Queen files in your migrator binary, then preview the database history transfer:

```bash
go run ./cmd/migrate adopt-goose --driver postgres --dsn "$DATABASE_URL" --goose-table goose_db_version
```

Inspect the printed version list and verify that the schema actually matches the applied Goose migrations. Apply only the exact previewed plan:

```bash
go run ./cmd/migrate adopt-goose --driver postgres --dsn "$DATABASE_URL" --goose-table goose_db_version \
  --apply --plan FINGERPRINT_FROM_PREVIEW --verified-schema --yes
```

The preview does not modify the database. Apply rereads both history tables under locks and writes Queen records in one transaction. It never executes migration SQL or changes Goose's table. The command requires a contiguous prefix of applied Goose versions backed by registered SQL-only Queen migrations with computed checksums. It rejects gaps, unknown versions, mismatched existing Queen records, and a changed plan. Goose numeric version `1` can match Queen version `001`.

After adoption, run `validate` and `status` before allowing Queen to apply later migrations.
