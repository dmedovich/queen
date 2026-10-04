# Path from v0.9.0 to v1.0

Queen targets a PostgreSQL-first stable release. The criteria below define what must be checked before claiming v1.0 stability; the other database drivers retain their documented, driver-specific guarantees.

## v0.9.0 release gate

- [ ] Review the [v0.9.0 changelog](../CHANGELOG.md) against the tagged diff, especially the module path change and the new migration order checks.
- [ ] Run formatting, lint, vet, race tests, and the PostgreSQL integration suite on the exact release commit. Confirm the remaining driver integration suites pass in CI.
- [ ] Build an application using `github.com/dmedovich/queen@v0.9.0` without a local `replace` directive after publishing the tag. Confirm the generated migrator builds and `verify-registry` passes.
- [ ] Exercise the documented CI workflow against a disposable PostgreSQL database, including `check --rollback-test`, deployment `up`, and `check --ci --json`.
- [ ] Record the exact commit and tag used for the release and link the changelog in the GitHub release notes.

## v1.0 acceptance criteria

### Public contract and upgrades

- [ ] Decide and document the supported Go and PostgreSQL versions. Run PostgreSQL integration tests on every version claimed to be supported.
- [ ] Freeze the exported library API and CLI command/flag behavior. Review defaults, errors, status values, checksum rules, and migration ordering; document intentional compatibility promises.
- [ ] Verify an existing Queen database created by `v0.8.0` upgrades under the new module path without replaying migrations or changing stored checksums. Test an applied history with the old table layout as well as a fresh install.
- [ ] Verify a real consumer can upgrade from the `v0.9.0` module to the v1.0 release with no `replace` directive and no manual database history edits.

### PostgreSQL behavior

- [ ] Test concurrent migrators, lock timeout/cancellation, and connection loss with a real PostgreSQL server. Assert that migration SQL and its history row agree after failures.
- [ ] Test `CREATE INDEX CONCURRENTLY` success, failure, interruption, and explicit recovery. Document how an operator verifies the index and chooses `recover --state`.
- [ ] Test schema-qualified history tables, existing table upgrades, transaction and statement timeouts, and privilege-limited deployment roles.
- [ ] Run a rehearsal of importing Goose SQL and adopting its existing history on a database snapshot. Compare applied versions and schema before any production cutover.

### Release operations

- [ ] Keep the source registry check, disposable rollback test, and production `validate`/`plan`/`up`/`check --ci` path in a consuming application's deployment pipeline.
- [ ] Publish a concise compatibility policy and operational recovery procedure. Confirm the README examples build with the published module version.
- [ ] Tag and publish v1.0 only after the release commit passes the agreed CI matrix and the PostgreSQL upgrade rehearsals.

This is a stability and evidence gate, not a requirement to add more CLI commands or drivers before v1.0.
