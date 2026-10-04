---
id: ci-cd
title: CI/CD
---

# CI/CD

Queen migrations are compiled into your application's migrator binary. Build it once per release, test that artifact, and use the same binary for deployment.

## Package job

```bash
go test ./...
go build -o queen-migrate ./cmd/migrate
./queen-migrate verify-registry --dir migrations
```

`verify-registry` compares literal migration versions in source with those registered in the binary. It needs the source directory, but no database connection. Go-function migrations need `ManualChecksum` for this check.

Use a disposable PostgreSQL database for a full apply, rollback, and reapply test:

```bash
export QUEEN_DRIVER=postgres
export QUEEN_DSN="$TEST_DATABASE_URL"
./queen-migrate check --rollback-test --no-gaps --json
```

`--rollback-test` requires a clean test database. It executes `up -> reset -> up`, so never point it at a deployed database.

## Deployment job

Make the application rollout depend on one migration job for the environment. Give database credentials only to that job and pass them through `QUEEN_DSN` rather than command arguments.

```bash
export QUEEN_DRIVER=postgres
export QUEEN_DSN="$PRODUCTION_DATABASE_URL"
./queen-migrate validate
./queen-migrate plan
./queen-migrate up --yes
./queen-migrate check --ci --no-gaps --json
./queen-migrate status
```

`check --ci` runs **after** `up`: it fails while migrations remain pending. `validate` and `plan` are the preflight commands. Keep schema changes compatible with old and new application instances during a rolling deployment.

`check --json` emits one object with `passed`, `failed`, `exit_code`, and `output`; the process also exits nonzero on failure. Configuration errors use code `2`, validation and connectivity errors `3`, strict gap failures `4`, pending migrations `5`, and rollback test failures `6`.

PostgreSQL advisory locks protect against accidental concurrent runs, but keep one intentional migration job per environment. The [GitHub Actions example](ci-cd-example.md) shows the complete workflow.
