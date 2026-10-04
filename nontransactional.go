package queen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/dmedovich/queen/tap"
)

const (
	statusApplying    = "applying"
	statusRollingBack = "rolling_back"
	// RecoveryApplied confirms that an interrupted migration's SQL effects exist.
	RecoveryApplied = "applied"
	// RecoveryNotApplied confirms that an interrupted migration's SQL effects do not exist.
	RecoveryNotApplied = "not-applied"
)

// ResolveIncomplete updates the migration record after an operator has checked
// the database schema. It never runs migration SQL. The version must still be
// registered with the same checksum as the interrupted migration.
func (q *Queen) ResolveIncomplete(ctx context.Context, version, resolution string) (retErr error) {
	if resolution != RecoveryApplied && resolution != RecoveryNotApplied {
		return fmt.Errorf("invalid recovery state %q: use %q or %q", resolution, RecoveryApplied, RecoveryNotApplied)
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	executor, ok := q.driver.(NonTransactionalExecutor)
	if !ok {
		return fmt.Errorf("driver %s does not support non-transactional recovery", q.getDriverName())
	}
	var migration *Migration
	for _, m := range q.migrations {
		if m.Version == version {
			migration = m
			break
		}
	}
	if migration == nil || !migration.NonTransactional {
		return fmt.Errorf("version %q is not a registered non-transactional migration", version)
	}
	unlock, err := q.initAndLock(ctx)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, unlock()) }()
	if err := q.loadApplied(ctx); err != nil {
		return err
	}
	entry, ok := q.applied[version]
	if !ok || (entry.Status != statusApplying && entry.Status != statusRollingBack) {
		return fmt.Errorf("%w: version %q has no interrupted migration record", ErrIncompleteMigration, version)
	}
	if entry.Name != migration.Name || entry.Checksum != migration.Checksum() {
		return fmt.Errorf("%w: version %q no longer matches the registered migration", ErrChecksumMismatch, version)
	}
	if resolution == RecoveryApplied {
		if err := executor.SetMigrationStatus(ctx, version, "success", 0); err != nil {
			return err
		}
		entry.Status = "success"
	} else {
		if err := q.driver.Remove(ctx, version); err != nil {
			return err
		}
		delete(q.applied, version)
	}
	return nil
}

func (q *Queen) execMigration(ctx context.Context, m *Migration, isolation sql.IsolationLevel, fn func(*sql.Tx) error) error {
	if executor, ok := q.driver.(MigrationExecutor); ok {
		return executor.ExecMigration(ctx, m, isolation, fn)
	}
	if m.SQLLockTimeout > 0 || m.StatementTimeout > 0 {
		return fmt.Errorf("driver %s does not support per-migration SQL timeouts", q.getDriverName())
	}
	return q.driver.Exec(ctx, isolation, fn)
}

func (q *Queen) applyNonTransactional(ctx context.Context, m *Migration) error {
	executor, ok := q.driver.(NonTransactionalExecutor)
	if !ok {
		return fmt.Errorf("driver %s does not support non-transactional migrations", q.getDriverName())
	}
	start := time.Now()
	q.emitStart(m, tap.DirectionUp, start)
	meta := q.collectMetadata("apply", statusApplying, 0, nil)
	err := q.driver.Record(ctx, m, meta)
	if err == nil {
		sqlStart := time.Now()
		err = executor.ExecNonTransactional(ctx, m, DirectionUp)
		q.emitExec(m, tap.DirectionUp, sqlStart, time.Since(sqlStart), m.UpSQL, 0, err)
		if err == nil {
			err = executor.SetMigrationStatus(ctx, m.Version, "success", time.Since(start).Milliseconds())
		}
	}
	q.emitEnd(m, tap.DirectionUp, start, time.Since(start), err)
	if err != nil {
		return err
	}
	meta.Status = "success"
	meta.DurationMS = time.Since(start).Milliseconds()
	q.applied[m.Version] = &Applied{
		Version: m.Version, Name: m.Name, AppliedAt: time.Now(),
		Checksum: m.Checksum(), AppliedBy: meta.AppliedBy,
		DurationMS: meta.DurationMS, Hostname: meta.Hostname,
		Environment: meta.Environment, Action: meta.Action, Status: meta.Status,
	}
	return nil
}

func (q *Queen) rollbackNonTransactional(ctx context.Context, m *Migration) error {
	executor, ok := q.driver.(NonTransactionalExecutor)
	if !ok {
		return fmt.Errorf("driver %s does not support non-transactional migrations", q.getDriverName())
	}
	start := time.Now()
	q.emitStart(m, tap.DirectionDown, start)
	err := executor.SetMigrationStatus(ctx, m.Version, statusRollingBack, 0)
	if err == nil {
		sqlStart := time.Now()
		err = executor.ExecNonTransactional(ctx, m, DirectionDown)
		q.emitExec(m, tap.DirectionDown, sqlStart, time.Since(sqlStart), m.DownSQL, 0, err)
		if err == nil {
			err = q.driver.Remove(ctx, m.Version)
		}
	}
	q.emitEnd(m, tap.DirectionDown, start, time.Since(start), err)
	if err != nil {
		return err
	}
	delete(q.applied, m.Version)
	return nil
}
