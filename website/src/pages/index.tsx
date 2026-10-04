import React from 'react';
import clsx from 'clsx';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useBaseUrl from '@docusaurus/useBaseUrl';

import styles from './index.module.css';

export default function Home(): React.ReactNode {
  const logoUrl = useBaseUrl('/img/queen_logo.png');

  return (
    <Layout
      title="Queen"
      description="Go database migrations embedded in your application">
      <header
        className="hero"
        style={{'--queen-logo-url': `url("${logoUrl}")`} as React.CSSProperties}>
        <div className="container">
          <h1 className="hero__title">Queen</h1>
          <p className="hero__subtitle">
            Go-first database migrations with an embedded CLI, production
            checks, multi-database drivers, and honest operational docs.
          </p>
          <div className="heroActions">
            <Link className="button button--primary button--lg" to="/docs/quick-start">
              Quick Start
            </Link>
            <Link className="button button--secondary button--lg" to="/docs/library">
              Use as a Library
            </Link>
            <Link className="button button--secondary button--lg" to="/docs/support-matrix">
              Support Matrix
            </Link>
          </div>
        </div>
      </header>

      <main>
        <section className={clsx('container', styles.section)}>
          <h2>Wiki Map</h2>
          <div className="quickGrid">
            <Link className="quickTile" to="/docs/migrations">
              <h3>Migration Format</h3>
              <p>SQL, Go functions, checksums, rollbacks, and naming.</p>
            </Link>
            <Link className="quickTile" to="/docs/cli">
              <h3>Embedded CLI</h3>
              <p>Wire commands into your own migrator binary.</p>
            </Link>
            <Link className="quickTile" to="/docs/examples">
              <h3>Examples</h3>
              <p>Common patterns for schema, data, rollback, and CI.</p>
            </Link>
            <Link className="quickTile" to="/docs/database-examples">
              <h3>Database Examples</h3>
              <p>SQL and Go-function migrations for each supported driver.</p>
            </Link>
            <Link className="quickTile" to="/docs/architecture">
              <h3>Architecture</h3>
              <p>How registry, driver, lock, execution, and records fit together.</p>
            </Link>
            <Link className="quickTile" to="/docs/drivers">
              <h3>Drivers</h3>
              <p>Locking and transaction guarantees by database.</p>
            </Link>
            <Link className="quickTile" to="/docs/known-limitations">
              <h3>Limitations</h3>
              <p>The current edge list for import, drivers, rollback, and TUI scope.</p>
            </Link>
          </div>
        </section>
      </main>
    </Layout>
  );
}
