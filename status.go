package queen

import "time"

type Status int

const (
	StatusPending Status = iota
	StatusApplied
	StatusModified
	StatusIncomplete
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusApplied:
		return "applied"
	case StatusModified:
		return "modified"
	case StatusIncomplete:
		return "incomplete"
	default:
		return "unknown"
	}
}

// MigrationStatus represents the current state of a migration.
type MigrationStatus struct {
	Version     string
	Name        string
	Status      Status
	AppliedAt   *time.Time
	Checksum    string
	HasRollback bool
	Destructive bool
}

type MigrationType string

const (
	MigrationTypeSQL    MigrationType = "sql"
	MigrationTypeGoFunc MigrationType = "go-func"
	MigrationTypeMixed  MigrationType = "mixed"
)

// MigrationPlan represents a migration execution plan.
type MigrationPlan struct {
	Version       string        `json:"version"`
	Name          string        `json:"name"`
	Direction     string        `json:"direction"`
	Status        string        `json:"status"`
	Type          MigrationType `json:"type"`
	SQL           string        `json:"sql,omitempty"`
	HasRollback   bool          `json:"has_rollback"`
	IsDestructive bool          `json:"is_destructive"`
	Checksum      string        `json:"checksum"`
	Warnings      []string      `json:"warnings,omitempty"`
}
