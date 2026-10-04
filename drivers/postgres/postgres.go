// Package postgres provides a PostgreSQL driver for Queen migrations.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/dmedovich/queen"
	"github.com/dmedovich/queen/drivers/base"
)

// Driver implements the queen.Driver interface for PostgreSQL.
type Driver struct {
	base.Driver
	lockID   int64
	lockMu   sync.Mutex
	lockConn *sql.Conn
}

// New creates a new PostgreSQL driver.
func New(db *sql.DB) *Driver {
	return NewWithTableName(db, "queen_migrations")
}

// NewWithTableName creates a new PostgreSQL driver with a custom table name.
func NewWithTableName(db *sql.DB, tableName string) *Driver {
	return &Driver{
		Driver: base.Driver{
			DB:        db,
			TableName: tableName,
			Config: base.Config{
				Placeholder:     base.PlaceholderDollar,
				QuoteIdentifier: quotePostgresIdentifier,
				ParseTime:       nil,
			},
		},
		lockID: hashTableName(tableName),
	}
}

// quotePostgresIdentifier accepts a table in the current schema or schema.table.
// Each component is quoted separately so schema-qualified tables resolve correctly.
func quotePostgresIdentifier(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return base.QuoteDoubleQuotes(parts[0]) + "." + base.QuoteDoubleQuotes(parts[1])
	}
	return base.QuoteDoubleQuotes(name)
}

// NewFromPool creates a PostgreSQL driver from a native pgx pool.
//
// The returned driver uses database/sql through pgx's stdlib adapter. Closing
// the driver closes only the adapter handle; it does not close the pgx pool.
func NewFromPool(pool *pgxpool.Pool) *Driver {
	return NewFromPoolWithTableName(pool, "queen_migrations")
}

// NewFromPoolWithTableName creates a PostgreSQL driver from a native pgx pool
// with a custom migration table name.
func NewFromPoolWithTableName(pool *pgxpool.Pool, tableName string) *Driver {
	return NewWithTableName(stdlib.OpenDBFromPool(pool), tableName)
}

// LockBeforeInit tells Queen that PostgreSQL can lock before table setup.
func (*Driver) LockBeforeInit() {}

// Init creates or upgrades the migrations tracking table under the migration lock.
func (d *Driver) Init(ctx context.Context) error {
	parts := strings.Split(d.TableName, ".")
	if len(parts) > 2 || len(d.TableName) == 0 || (len(parts) == 2 && (parts[0] == "" || parts[1] == "")) {
		return fmt.Errorf("invalid PostgreSQL migration table %q: use table or schema.table", d.TableName)
	}
	tableRef := d.Config.QuoteIdentifier(d.TableName)
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			checksum VARCHAR(64) NOT NULL,
			applied_by VARCHAR(255),
			duration_ms BIGINT,
			hostname VARCHAR(255),
			environment VARCHAR(50),
			action VARCHAR(20) DEFAULT 'apply',
			status VARCHAR(20) DEFAULT 'success',
			error_message TEXT
		)
	`, tableRef)

	metadataColumns := []struct{ name, definition string }{
		{"applied_by", "VARCHAR(255)"},
		{"duration_ms", "BIGINT"},
		{"hostname", "VARCHAR(255)"},
		{"environment", "VARCHAR(50)"},
		{"action", "VARCHAR(20) DEFAULT 'apply'"},
		{"status", "VARCHAR(20) DEFAULT 'success'"},
		{"error_message", "TEXT"},
	}

	return d.Exec(ctx, sql.LevelDefault, func(tx *sql.Tx) error {
		// Direct Init calls also serialize with migrations through this key.
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", d.lockID); err != nil {
			return fmt.Errorf("lock migration table initialization: %w", err)
		}
		var existing sql.NullString
		if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1)", tableRef).Scan(&existing); err != nil {
			return fmt.Errorf("find migration table: %w", err)
		}
		if !existing.Valid {
			if _, err := tx.ExecContext(ctx, query); err != nil {
				return fmt.Errorf("create migration table: %w", err)
			}
			return nil
		}

		rows, err := tx.QueryContext(ctx, `SELECT attname FROM pg_attribute
			WHERE attrelid = to_regclass($1) AND attnum > 0 AND NOT attisdropped`, tableRef)
		if err != nil {
			return fmt.Errorf("inspect migration table: %w", err)
		}
		columns := make(map[string]bool)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				_ = rows.Close()
				return fmt.Errorf("inspect migration table: %w", err)
			}
			columns[name] = true
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("inspect migration table: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("inspect migration table: %w", err)
		}
		for _, name := range []string{"version", "name", "applied_at", "checksum"} {
			if !columns[name] {
				return fmt.Errorf("migration table %s is missing required column %s", tableRef, name)
			}
		}
		for _, column := range metadataColumns {
			if columns[column.name] {
				continue
			}
			alter := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, tableRef, d.Config.QuoteIdentifier(column.name), column.definition)
			if _, err := tx.ExecContext(ctx, alter); err != nil {
				return fmt.Errorf("upgrade migration table column %s: %w", column.name, err)
			}
		}
		return nil
	})
}

// Lock acquires an advisory lock to prevent concurrent migrations.
func (d *Driver) Lock(ctx context.Context, timeout time.Duration) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()

	if d.lockConn != nil {
		return fmt.Errorf("postgres driver already holds advisory lock '%d' for table '%s'",
			d.lockID, d.TableName)
	}

	conn, err := d.DB.Conn(ctx)
	if err != nil {
		return err
	}

	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err = conn.ExecContext(lockCtx, "SELECT pg_advisory_lock($1)", d.lockID)
	if err != nil {
		_ = conn.Close()
		if errors.Is(lockCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("%w: failed to acquire advisory lock '%d' for table '%s'",
				queen.ErrLockTimeout, d.lockID, d.TableName)
		}
		return err
	}

	d.lockConn = conn
	return nil
}

// Unlock releases the advisory lock.
func (d *Driver) Unlock(ctx context.Context) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()

	if d.lockConn == nil {
		return nil
	}
	conn := d.lockConn
	defer func() {
		_ = conn.Close()
		d.lockConn = nil
	}()

	var unlocked bool
	err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", d.lockID).Scan(&unlocked)
	if err != nil {
		return fmt.Errorf("failed to release advisory lock '%d' for table '%s': %w",
			d.lockID, d.TableName, err)
	}
	if !unlocked {
		return fmt.Errorf("advisory lock '%d' for table '%s' was not held by the migration session", d.lockID, d.TableName)
	}
	return nil
}

// Exec runs migrations on the connection holding the advisory lock. If that
// session is lost, the migration transaction is lost with it.
func (d *Driver) Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()

	if d.lockConn == nil {
		return d.Driver.Exec(ctx, isolationLevel, fn)
	}

	tx, err := d.lockConn.BeginTx(ctx, &sql.TxOptions{Isolation: isolationLevel})
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// GetApplied reads through the lock connection when a migration holds it.
func (d *Driver) GetApplied(ctx context.Context) ([]queen.Applied, error) {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()
	if d.lockConn != nil {
		return base.GetApplied(ctx, &d.Driver, d.lockConn)
	}
	return d.Driver.GetApplied(ctx)
}

// Record writes through the lock connection when the migration is non-transactional.
func (d *Driver) Record(ctx context.Context, m *queen.Migration, meta *queen.MigrationMetadata) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()
	if d.lockConn != nil {
		return base.RecordOn(ctx, &d.Driver, d.lockConn, m, meta)
	}
	return d.Driver.Record(ctx, m, meta)
}

// Remove writes through the lock connection when the migration is non-transactional.
func (d *Driver) Remove(ctx context.Context, version string) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()
	if d.lockConn != nil {
		return base.RemoveOn(ctx, &d.Driver, d.lockConn, version)
	}
	return d.Driver.Remove(ctx, version)
}

// ExecMigration sets PostgreSQL timeouts for the duration of one transaction.
func (d *Driver) ExecMigration(ctx context.Context, m *queen.Migration, isolation sql.IsolationLevel, fn func(*sql.Tx) error) error {
	return d.Exec(ctx, isolation, func(tx *sql.Tx) error {
		if err := setLocalTimeout(ctx, tx, "lock_timeout", m.SQLLockTimeout); err != nil {
			return err
		}
		if err := setLocalTimeout(ctx, tx, "statement_timeout", m.StatementTimeout); err != nil {
			return err
		}
		return fn(tx)
	})
}

func setLocalTimeout(ctx context.Context, tx *sql.Tx, name string, timeout time.Duration) error {
	if timeout == 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, "SELECT set_config($1, $2, true)", name, fmt.Sprintf("%dms", timeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("set %s: %w", name, err)
	}
	return nil
}

// ExecNonTransactional runs one SQL command on the lock-holding connection.
func (d *Driver) ExecNonTransactional(ctx context.Context, m *queen.Migration, direction string) (retErr error) {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()

	conn := d.lockConn
	if conn == nil {
		var err error
		conn, err = d.DB.Conn(ctx)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, conn.Close()) }()
	}

	if m.SQLLockTimeout > 0 {
		previous, err := setSessionTimeout(ctx, conn, "lock_timeout", m.SQLLockTimeout)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, restoreSessionTimeout(conn, "lock_timeout", previous)) }()
	}
	if m.StatementTimeout > 0 {
		previous, err := setSessionTimeout(ctx, conn, "statement_timeout", m.StatementTimeout)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, restoreSessionTimeout(conn, "statement_timeout", previous)) }()
	}

	query := m.UpSQL
	if direction == queen.DirectionDown {
		query = m.DownSQL
	} else if direction != queen.DirectionUp {
		return fmt.Errorf("invalid migration direction %q", direction)
	}
	if query == "" {
		return queen.ErrInvalidMigration
	}
	_, err := conn.ExecContext(ctx, query)
	return err
}

func setSessionTimeout(ctx context.Context, conn *sql.Conn, name string, timeout time.Duration) (string, error) {
	var previous string
	if err := conn.QueryRowContext(ctx, "SELECT current_setting($1)", name).Scan(&previous); err != nil {
		return "", fmt.Errorf("read %s: %w", name, err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT set_config($1, $2, false)", name, fmt.Sprintf("%dms", timeout.Milliseconds())); err != nil {
		return "", fmt.Errorf("set %s: %w", name, err)
	}
	return previous, nil
}

func restoreSessionTimeout(conn *sql.Conn, name, previous string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := conn.ExecContext(ctx, "SELECT set_config($1, $2, false)", name, previous)
	if err != nil {
		return fmt.Errorf("restore %s: %w", name, err)
	}
	return nil
}

// SetMigrationStatus persists the recovery marker around a non-transactional command.
func (d *Driver) SetMigrationStatus(ctx context.Context, version, status string, durationMS int64) error {
	d.lockMu.Lock()
	defer d.lockMu.Unlock()
	execer := base.Execer(d.DB)
	if d.lockConn != nil {
		execer = d.lockConn
	}
	query := fmt.Sprintf(`UPDATE %s SET status = $1, duration_ms = CASE WHEN $2 > 0 THEN $2 ELSE duration_ms END WHERE version = $3`, d.Config.QuoteIdentifier(d.TableName))
	result, err := execer.ExecContext(ctx, query, status, durationMS, version)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("update migration %s status: affected %d rows, want 1", version, rows)
	}
	return nil
}

// RecordTx records an applied migration in the migration transaction.
func (d *Driver) RecordTx(ctx context.Context, tx *sql.Tx, m *queen.Migration, meta *queen.MigrationMetadata) error {
	return base.RecordTx(ctx, &d.Driver, tx, m, meta)
}

// RemoveTx removes a migration record in the rollback transaction.
func (d *Driver) RemoveTx(ctx context.Context, tx *sql.Tx, version string) error {
	return base.RemoveTx(ctx, &d.Driver, tx, version)
}

// hashTableName creates a stable int64 key from the table name for advisory locks.
func hashTableName(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64())
}
