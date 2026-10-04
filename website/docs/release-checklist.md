---
id: release-checklist
title: Release Checklist
---

# Release Checklist

This page is for maintainers releasing Queen or updating the documentation. Application teams can use the same checklist as inspiration for their own migration release process.

## Before a Queen release

On the exact release commit, require formatting, `go vet ./...`, `golangci-lint run ./...`, and the race-tested unit suite:

```bash
go test -race ./...
```

Require the repository's integration jobs to pass, including the PostgreSQL 15 and 16 matrix and the other driver suites. The PostgreSQL job runs `make test-postgres`, which also tests the CLI and Goose import against isolated databases.

Build the documentation:

```bash
npm ci --prefix website
npm run build --prefix website
```

Check dependency advisories for the docs site:

```bash
npm audit --prefix website --omit=dev --audit-level=high
```

## Documentation checks

- Verify [CLI Reference](cli-reference.md) against Cobra command definitions in `cli/`.
- Verify [Support Matrix](support-matrix.md) against driver implementation details.
- Keep [Known Limitations](known-limitations.md) honest when a bug is fixed or a caveat is removed.
- Rebuild the docs after sidebar, route, or page id changes.
- Add a `## vX.Y.Z` section to `CHANGELOG.md` before tagging. The site build creates a page for each release; publication fails if the published tag has no matching notes.
- Keep `.queen.yaml.example` and `.queenignore.example` aligned with [Config Files](config-files.md).

## Migration compatibility checks

- Run `queen import` tests after changing import code.
- Try `import --dry-run` on representative goose migrations before advertising new importer support.
- Confirm that generated files are not overwritten by import, init, or create flows.
- Confirm that examples still compile when copied into a normal Go module.

## Release notes checklist

Call out:

- driver behavior changes;
- CLI command or flag changes;
- migration format changes;
- checksum behavior changes;
- importer compatibility changes;
- known limitations that were fixed or newly discovered.

The goal is simple: a user should not need to read source code to understand whether an upgrade changes migration safety.
