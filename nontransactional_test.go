package queen

import (
	"context"
	"errors"
	"testing"
)

type nonTransactionalTestDriver struct {
	testDriver
	execErr   error
	execCalls int
}

func (d *nonTransactionalTestDriver) Record(ctx context.Context, m *Migration, meta *MigrationMetadata) error {
	if err := d.testDriver.Record(ctx, m, meta); err != nil {
		return err
	}
	return d.SetMigrationStatus(ctx, m.Version, meta.Status, meta.DurationMS)
}

func (d *nonTransactionalTestDriver) ExecNonTransactional(context.Context, *Migration, string) error {
	d.execCalls++
	return d.execErr
}

func (d *nonTransactionalTestDriver) SetMigrationStatus(_ context.Context, version, status string, _ int64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	a := d.applied[version]
	a.Status = status
	d.applied[version] = a
	return nil
}

func TestNonTransactionalMigrationLeavesRecoveryMarkerOnFailure(t *testing.T) {
	driver := &nonTransactionalTestDriver{execErr: errors.New("statement failed")}
	q := New(driver)
	q.MustAdd(Migration{Version: "001", Name: "build_index", UpSQL: "CREATE INDEX CONCURRENTLY idx ON t (id)", NonTransactional: true})

	if err := q.Up(context.Background()); err == nil {
		t.Fatal("Up() succeeded despite SQL failure")
	}
	statuses, err := q.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].Status != StatusIncomplete {
		t.Fatalf("statuses = %+v, want incomplete", statuses)
	}
	if err := q.Up(context.Background()); !errors.Is(err, ErrIncompleteMigration) {
		t.Fatalf("second Up() error = %v, want ErrIncompleteMigration", err)
	}
	if driver.execCalls != 1 {
		t.Fatalf("SQL executed %d times, want once", driver.execCalls)
	}
}

func TestNonTransactionalMigrationCanApplyAndRollback(t *testing.T) {
	driver := &nonTransactionalTestDriver{}
	q := New(driver)
	q.MustAdd(Migration{
		Version: "001", Name: "build_index", NonTransactional: true,
		UpSQL: "CREATE INDEX CONCURRENTLY idx ON t (id)", DownSQL: "DROP INDEX CONCURRENTLY idx",
	})
	if err := q.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := q.Down(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if driver.execCalls != 2 {
		t.Fatalf("SQL executed %d times, want twice", driver.execCalls)
	}
}

func TestResolveIncompleteRequiresVerifiedRecord(t *testing.T) {
	driver := &nonTransactionalTestDriver{execErr: errors.New("statement failed")}
	q := New(driver)
	m := Migration{Version: "001", Name: "build_index", UpSQL: "CREATE INDEX CONCURRENTLY idx ON t (id)", NonTransactional: true}
	q.MustAdd(m)
	ctx := context.Background()
	if err := q.Up(ctx); err == nil {
		t.Fatal("expected interrupted migration")
	}
	if err := q.ResolveIncomplete(ctx, "001", "unknown"); err == nil {
		t.Fatal("invalid resolution was accepted")
	}
	if err := q.ResolveIncomplete(ctx, "001", RecoveryApplied); err != nil {
		t.Fatal(err)
	}
	if err := q.ResolveIncomplete(ctx, "001", RecoveryNotApplied); !errors.Is(err, ErrIncompleteMigration) {
		t.Fatalf("resolved record could be changed again: %v", err)
	}
	if err := q.Up(ctx); err != nil {
		t.Fatalf("resolved migration should no longer block Up: %v", err)
	}
}

func TestResolveIncompleteNotAppliedAllowsRetry(t *testing.T) {
	driver := &nonTransactionalTestDriver{execErr: errors.New("statement failed")}
	q := New(driver)
	q.MustAdd(Migration{Version: "001", Name: "build_index", UpSQL: "CREATE INDEX CONCURRENTLY idx ON t (id)", NonTransactional: true})
	ctx := context.Background()
	if err := q.Up(ctx); err == nil {
		t.Fatal("expected interrupted migration")
	}
	if err := q.ResolveIncomplete(ctx, "001", RecoveryNotApplied); err != nil {
		t.Fatal(err)
	}
	driver.execErr = nil
	if err := q.Up(ctx); err != nil {
		t.Fatalf("retry after recovery: %v", err)
	}
	if driver.execCalls != 2 {
		t.Fatalf("SQL executed %d times, want 2", driver.execCalls)
	}
}

func TestResolveIncompleteRejectsChangedMigration(t *testing.T) {
	driver := &nonTransactionalTestDriver{execErr: errors.New("statement failed")}
	q := New(driver)
	q.MustAdd(Migration{Version: "001", Name: "build_index", UpSQL: "CREATE INDEX CONCURRENTLY idx ON t (id)", NonTransactional: true})
	ctx := context.Background()
	if err := q.Up(ctx); err == nil {
		t.Fatal("expected interrupted migration")
	}
	driver.mu.Lock()
	entry := driver.applied["001"]
	entry.Checksum = "changed"
	driver.applied["001"] = entry
	driver.mu.Unlock()
	if err := q.ResolveIncomplete(ctx, "001", RecoveryApplied); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("ResolveIncomplete error = %v, want ErrChecksumMismatch", err)
	}
	if driver.applied["001"].Status != statusApplying {
		t.Fatal("recovery changed a record with mismatched checksum")
	}
}
