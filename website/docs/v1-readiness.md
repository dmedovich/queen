---
id: v1-readiness
title: v1.0 Release Readiness
---

# v1.0 release readiness

Queen's v1.0 contract is PostgreSQL-first. The checks here distinguish repository work from evidence that must come from the release commit and a consuming application.

## Prepared in the repository

- [x] Publish v0.9.0 at the new module path and verify that a separate consumer resolves it without `replace`.
- [x] Document the [v1.0 compatibility contract](compatibility.md), Go 1.26.3 baseline, and PostgreSQL 15/16 support matrix.
- [x] Cover the v0.8 history table layout upgrade and ensure applied migrations are not replayed or rechecksummed.
- [x] Cover PostgreSQL concurrent migrators, advisory lock timeout, cancellation, connection loss, SQL lock and statement timeouts, schema-qualified history, and a deployment role without schema `CREATE` privilege.
- [x] Cover successful, failed, and interrupted `CREATE INDEX CONCURRENTLY`, incomplete migration detection, explicit recovery, and retry. Document [operator recovery](drivers/postgres.md#limitations).
- [x] Cover Goose import and history adoption with integration tests and document [cutover verification](goose-import.md).
- [x] Keep the [CI/CD workflow](ci-cd.md) for registry verification, disposable rollback tests, and production `validate`/`plan`/`up`/`check --ci`.
- [x] Move the documentation site into this repository, build it in CI, and generate versioned release pages from `CHANGELOG.md` when a release is published.
- [x] Write the `v1.0.0` release notes before tagging and keep README limited to an introduction and example.

## Before publishing the tag

- [ ] Run the release commit through formatting, lint, vet, race tests, documentation build, and all database integration jobs. Confirm the PostgreSQL 15 and 16 jobs both pass.
- [ ] Rehearse the consuming application's exact migration binary against a disposable PostgreSQL database: `verify-registry`, `check --rollback-test`, then `validate`, `plan`, `up`, and `check --ci --json`.
- [ ] If migrating an existing Goose database, rehearse import and adoption on a recent database snapshot. Compare applied versions and schema before the production cutover.
- [ ] Confirm GitHub Pages uses the GitHub Actions source and that the documentation workflow can deploy the release page.
- [ ] Review the v1.0 notes against the final commit, then tag and publish only after the release commit passes CI.

After publication, verify that a consumer upgrades from `github.com/dmedovich/queen@v0.9.0` to `@v1.0.0` without `replace` or manual history edits. This final public-module check requires the published tag.
