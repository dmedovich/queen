package queen

import (
	"errors"
	"fmt"
)

var (
	ErrNoMigrations         = errors.New("no migrations registered")
	ErrVersionConflict      = errors.New("version conflict")
	ErrMigrationNotFound    = errors.New("migration not found")
	ErrChecksumMismatch     = errors.New("checksum mismatch")
	ErrLockTimeout          = errors.New("lock timeout")
	ErrNoDriver             = errors.New("driver not initialized")
	ErrInvalidMigration     = errors.New("invalid migration")
	ErrNameTooLong          = errors.New("migration name exceeds 63 characters")
	ErrInvalidMigrationName = errors.New("invalid migration name")
	ErrAlreadyApplied       = errors.New("migration already applied")
	ErrIncompleteMigration  = errors.New("incomplete non-transactional migration; inspect database state before recovery")
	ErrUnknownApplied       = errors.New("database contains a migration not registered in code")
	ErrOutOfOrderMigration  = errors.New("pending migration precedes an applied migration")
)

// MigrationError wraps an error with migration context.
type MigrationError struct {
	Version   string
	Name      string
	Operation string
	Driver    string
	Cause     error
}

func (e *MigrationError) Error() string {
	if e.Driver != "" && e.Operation != "" {
		return fmt.Sprintf("migration %s (%s) failed during %s operation on %s: %v",
			e.Version, e.Name, e.Operation, e.Driver, e.Cause)
	}
	if e.Operation != "" {
		return fmt.Sprintf("migration %s (%s) failed during %s: %v",
			e.Version, e.Name, e.Operation, e.Cause)
	}
	return fmt.Sprintf("migration %s (%s): %v", e.Version, e.Name, e.Cause)
}

func (e *MigrationError) Unwrap() error {
	return e.Cause
}

func newMigrationError(version, name, operation, driver string, err error) error {
	return &MigrationError{
		Version:   version,
		Name:      name,
		Operation: operation,
		Driver:    driver,
		Cause:     err,
	}
}
