package queen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// testDriver is a minimal driver implementation for testing.
type testDriver struct {
	mu      sync.Mutex
	applied map[string]Applied
}

func (d *testDriver) Init(ctx context.Context) error { return nil }
func (d *testDriver) GetApplied(ctx context.Context) ([]Applied, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	applied := make([]Applied, 0, len(d.applied))
	for _, a := range d.applied {
		applied = append(applied, a)
	}
	return applied, nil
}
func (d *testDriver) Record(ctx context.Context, m *Migration, meta *MigrationMetadata) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.applied == nil {
		d.applied = make(map[string]Applied)
	}
	d.applied[m.Version] = Applied{
		Version:   m.Version,
		Name:      m.Name,
		AppliedAt: time.Now(),
		Checksum:  m.Checksum(),
	}
	return nil
}
func (d *testDriver) Remove(ctx context.Context, version string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.applied, version)
	return nil
}
func (d *testDriver) Lock(ctx context.Context, timeout time.Duration) error { return nil }
func (d *testDriver) Unlock(ctx context.Context) error                      { return nil }
func (d *testDriver) Exec(ctx context.Context, isolationLevel sql.IsolationLevel, fn func(*sql.Tx) error) error {
	return fn(nil)
}
func (d *testDriver) Close() error { return nil }

func TestGetDriverName(t *testing.T) {
	driver := &testDriver{}
	q := New(driver)

	driverName := q.getDriverName()
	// Should extract "queen" from "*queen.testDriver"
	if driverName != "queen" {
		t.Errorf("getDriverName() = %q, want %q", driverName, "queen")
	}
}

func TestGetDriverNameNilDriver(t *testing.T) {
	q := &Queen{driver: nil}
	driverName := q.getDriverName()
	if driverName != driverNameUnknown {
		t.Errorf("getDriverName() with nil driver = %q, want %q", driverName, driverNameUnknown)
	}
}

func TestStrictMigrationHistory(t *testing.T) {
	ctx := context.Background()
	newMigration := func(version string) Migration {
		return Migration{Version: version, Name: "migration_" + version, UpFunc: func(context.Context, *sql.Tx) error { return nil }}
	}
	t.Run("unknown applied version blocks changes by default", func(t *testing.T) {
		driver := &testDriver{applied: map[string]Applied{"002": {Version: "002", Status: "success"}}}
		q := New(driver)
		q.MustAdd(newMigration("001"))
		if err := q.Up(ctx); !errors.Is(err, ErrUnknownApplied) {
			t.Fatalf("Up error = %v, want ErrUnknownApplied", err)
		}
		allowed := NewWithConfig(driver, &Config{AllowUnknownApplied: true, AllowOutOfOrder: true})
		allowed.MustAdd(newMigration("001"))
		if err := allowed.Up(ctx); err != nil {
			t.Fatalf("explicitly allowed rolling deploy: %v", err)
		}
	})
	t.Run("older pending version requires explicit opt out", func(t *testing.T) {
		newer := newMigration("002")
		driver := &testDriver{applied: map[string]Applied{"002": {Version: "002", Name: newer.Name, Checksum: newer.Checksum(), Status: "success"}}}
		q := New(driver)
		q.MustAdd(newMigration("001"))
		q.MustAdd(newer)
		if err := q.Validate(ctx); !errors.Is(err, ErrOutOfOrderMigration) {
			t.Fatalf("Validate error = %v, want ErrOutOfOrderMigration", err)
		}
		if err := q.Up(ctx); !errors.Is(err, ErrOutOfOrderMigration) {
			t.Fatalf("Up error = %v, want ErrOutOfOrderMigration", err)
		}
		allowed := NewWithConfig(driver, &Config{AllowOutOfOrder: true})
		allowed.MustAdd(newMigration("001"))
		allowed.MustAdd(newer)
		if err := allowed.Up(ctx); err != nil {
			t.Fatalf("explicitly allowed out-of-order migration: %v", err)
		}
	})
}

func TestQueenConcurrentRegistrationAndReaders(t *testing.T) {
	q := New(&testDriver{})
	ctx := context.Background()

	const migrations = 128
	const readers = 16

	var wg sync.WaitGroup
	for i := range migrations {
		wg.Go(func() {
			err := q.Add(Migration{
				Version: fmt.Sprintf("%03d", i+1),
				Name:    fmt.Sprintf("migration_%03d", i+1),
				UpFunc:  func(context.Context, *sql.Tx) error { return nil },
			})
			if err != nil {
				t.Errorf("Add() failed: %v", err)
			}
		})
	}

	for range readers {
		wg.Go(func() {
			for j := range migrations {
				if _, err := q.Status(ctx); err != nil {
					t.Errorf("Status() failed: %v", err)
				}
				if err := q.Validate(ctx); err != nil && !errors.Is(err, ErrNoMigrations) {
					t.Errorf("Validate() failed: %v", err)
				}
				if _, err := q.DetectGaps(ctx); err != nil {
					t.Errorf("DetectGaps() failed: %v", err)
				}
				_ = q.FindMigration(fmt.Sprintf("%03d", (j%migrations)+1))
				q.SetTap(nil)
			}
		})
	}

	wg.Wait()

	statuses, err := q.Status(ctx)
	if err != nil {
		t.Fatalf("Status() after concurrent access: %v", err)
	}
	if len(statuses) != migrations {
		t.Fatalf("registered migrations = %d, want %d", len(statuses), migrations)
	}
	for i := range migrations {
		version := fmt.Sprintf("%03d", i+1)
		if q.FindMigration(version) == nil {
			t.Fatalf("FindMigration(%q) = nil after concurrent registration", version)
		}
	}
}

type unlockDeadlineDriver struct {
	testDriver

	sawDeadline bool
}

type unlockErrorDriver struct {
	testDriver
	err error
}

type lockFirstTestDriver struct {
	testDriver
	events []string
}

func (*lockFirstTestDriver) LockBeforeInit() {}
func (d *lockFirstTestDriver) Lock(context.Context, time.Duration) error {
	d.events = append(d.events, "lock")
	return nil
}
func (d *lockFirstTestDriver) Init(context.Context) error {
	d.events = append(d.events, "init")
	return nil
}

func TestUpLocksBeforeInitWhenDriverRequiresIt(t *testing.T) {
	driver := &lockFirstTestDriver{}
	q := New(driver)
	q.MustAdd(Migration{
		Version: "001", Name: "lock_before_init",
		UpFunc: func(context.Context, *sql.Tx) error { return nil },
	})
	if err := q.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(driver.events) != 2 || driver.events[0] != "lock" || driver.events[1] != "init" {
		t.Fatalf("events = %v, want [lock init]", driver.events)
	}
}

func (d *unlockErrorDriver) Unlock(context.Context) error { return d.err }

func TestUpReportsUnlockFailure(t *testing.T) {
	unlockErr := errors.New("connection lost during unlock")
	driver := &unlockErrorDriver{err: unlockErr}
	q := New(driver)
	q.MustAdd(Migration{
		Version: "001", Name: "unlock_error",
		UpFunc: func(context.Context, *sql.Tx) error { return nil },
	})
	if err := q.Up(context.Background()); !errors.Is(err, unlockErr) {
		t.Fatalf("Up() error = %v, want unlock error", err)
	}
}

func (d *unlockDeadlineDriver) Unlock(ctx context.Context) error {
	_, d.sawDeadline = ctx.Deadline()
	return nil
}

func TestLockUnlockUsesBoundedContext(t *testing.T) {
	driver := &unlockDeadlineDriver{}
	q := NewWithConfig(driver, &Config{
		TableName:   "queen_migrations",
		LockTimeout: time.Minute,
	})

	unlock, err := q.lock(context.Background())
	if err != nil {
		t.Fatalf("lock() failed: %v", err)
	}
	if err := unlock(); err != nil {
		t.Fatalf("unlock() failed: %v", err)
	}

	if !driver.sawDeadline {
		t.Fatal("Unlock() did not receive a bounded context")
	}
}

func TestNewWithConfigCopiesMutableConfig(t *testing.T) {
	t.Parallel()

	naming := &NamingConfig{
		Pattern: NamingPatternSequentialPadded,
		Padding: 3,
		Enforce: true,
	}
	config := &Config{
		TableName:   "queen_migrations",
		LockTimeout: time.Minute,
		Naming:      naming,
	}

	q := NewWithConfig(&testDriver{}, config)
	config.TableName = "mutated_table"
	config.LockTimeout = time.Hour
	naming.Padding = 9
	naming.Enforce = false

	if q.config == config {
		t.Fatal("NewWithConfig retained caller-owned Config pointer")
	}
	if q.config.Naming == naming {
		t.Fatal("NewWithConfig retained caller-owned NamingConfig pointer")
	}
	if q.config.TableName != "queen_migrations" {
		t.Fatalf("table name changed after caller mutation: %q", q.config.TableName)
	}
	if q.config.LockTimeout != time.Minute {
		t.Fatalf("lock timeout changed after caller mutation: %v", q.config.LockTimeout)
	}
	if q.config.Naming.Padding != 3 || !q.config.Naming.Enforce {
		t.Fatalf("naming config changed after caller mutation: %+v", q.config.Naming)
	}
}

func TestFindMigrationReturnsDefensiveCopy(t *testing.T) {
	t.Parallel()

	q := New(&testDriver{})
	q.MustAdd(Migration{
		Version: "001",
		Name:    "create_users",
		UpSQL:   "CREATE TABLE users (id INTEGER)",
	})

	found := q.FindMigration("001")
	if found == nil {
		t.Fatal("FindMigration() = nil")
	}

	found.Name = "mutated"
	found.UpSQL = "DROP TABLE users"

	foundAgain := q.FindMigration("001")
	if foundAgain == nil {
		t.Fatal("FindMigration() after mutation = nil")
	}
	if foundAgain.Name != "create_users" {
		t.Fatalf("FindMigration exposed internal migration name: %q", foundAgain.Name)
	}
	if foundAgain.UpSQL != "CREATE TABLE users (id INTEGER)" {
		t.Fatalf("FindMigration exposed internal migration SQL: %q", foundAgain.UpSQL)
	}
}
