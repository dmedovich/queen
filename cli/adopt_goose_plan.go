package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/dmedovich/queen"
)

type gooseEvent struct {
	ID      int64 `json:"id"`
	Version int64 `json:"version"`
	Applied bool  `json:"applied"`
}

type queenHistoryRow struct {
	Version  string `json:"version"`
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
	Status   string `json:"status"`
}

type gooseAdoptionPlan struct {
	SourceTable  string            `json:"source_table"`
	TargetTable  string            `json:"target_table"`
	Events       []gooseEvent      `json:"events"`
	Registered   []queenHistoryRow `json:"registered"`
	CurrentQueen []queenHistoryRow `json:"current_queen"`
	ToAdopt      []queen.Migration `json:"-"`
	AlreadyDone  bool              `json:"already_done"`
	Fingerprint  string            `json:"-"`
}

func buildGooseAdoptionPlan(sourceTable, targetTable string, events []gooseEvent, current []queenHistoryRow, registered []queen.Migration) (gooseAdoptionPlan, error) {
	plan := gooseAdoptionPlan{SourceTable: sourceTable, TargetTable: targetTable, Events: events, CurrentQueen: current}
	if len(registered) == 0 {
		return plan, fmt.Errorf("no Queen migrations are registered")
	}
	if len(events) == 0 {
		return plan, fmt.Errorf("goose history is empty; no applied migration history to adopt")
	}
	byNumeric := make(map[int64]queen.Migration, len(registered))
	numericVersions := make([]int64, 0, len(registered))
	for _, migration := range registered {
		if err := migration.Validate(); err != nil {
			return plan, fmt.Errorf("registered migration %s is invalid: %w", migration.Version, err)
		}
		version, err := strconv.ParseInt(migration.Version, 10, 64)
		if err != nil || version <= 0 {
			return plan, fmt.Errorf("queen version %q cannot map to a positive Goose integer version", migration.Version)
		}
		if previous, exists := byNumeric[version]; exists {
			return plan, fmt.Errorf("queen versions %q and %q map to the same Goose version %d", previous.Version, migration.Version, version)
		}
		byNumeric[version] = migration
		numericVersions = append(numericVersions, version)
	}
	sort.Slice(numericVersions, func(i, j int) bool { return numericVersions[i] < numericVersions[j] })
	for _, version := range numericVersions {
		migration := byNumeric[version]
		plan.Registered = append(plan.Registered, queenHistoryRow{Version: migration.Version, Name: migration.Name, Checksum: migration.Checksum()})
	}
	latest := make(map[int64]bool)
	var previousID int64
	for i, event := range events {
		if event.Version < 0 || (i > 0 && event.ID <= previousID) {
			return plan, fmt.Errorf("goose history contains invalid or unordered rows")
		}
		previousID = event.ID
		if event.Version == 0 {
			continue
		}
		if _, exists := byNumeric[event.Version]; !exists {
			return plan, fmt.Errorf("goose history contains version %d missing from Queen registration", event.Version)
		}
		latest[event.Version] = event.Applied
	}
	seenPending := false
	for _, version := range numericVersions {
		if !latest[version] {
			seenPending = true
			continue
		}
		if seenPending {
			return plan, fmt.Errorf("goose applied history has a gap before version %d", version)
		}
		migration := byNumeric[version]
		if migration.UpSQL == "" || migration.UpFunc != nil || migration.DownFunc != nil || migration.ManualChecksum != "" {
			return plan, fmt.Errorf("applied Goose version %d must be a SQL-only Queen migration with its computed checksum", version)
		}
		plan.ToAdopt = append(plan.ToAdopt, migration)
	}
	if len(plan.ToAdopt) == 0 {
		return plan, fmt.Errorf("goose history has no applied migration versions")
	}
	sort.Slice(plan.CurrentQueen, func(i, j int) bool { return plan.CurrentQueen[i].Version < plan.CurrentQueen[j].Version })
	if len(current) != 0 {
		if len(current) != len(plan.ToAdopt) {
			return plan, fmt.Errorf("queen history already contains %d records; expected an empty table or the exact %d adopted records", len(current), len(plan.ToAdopt))
		}
		byVersion := make(map[string]queenHistoryRow, len(current))
		for _, entry := range current {
			byVersion[entry.Version] = entry
		}
		for _, migration := range plan.ToAdopt {
			entry := byVersion[migration.Version]
			if entry.Version != migration.Version || entry.Name != migration.Name || entry.Checksum != migration.Checksum() || entry.Status != "success" {
				return plan, fmt.Errorf("queen history differs from Goose adoption at version %s", migration.Version)
			}
		}
		plan.AlreadyDone = true
	}
	payload, err := json.Marshal(plan)
	if err != nil {
		return plan, err
	}
	hash := sha256.Sum256(payload)
	plan.Fingerprint = hex.EncodeToString(hash[:])
	return plan, nil
}

func quotePostgresTable(name string) (string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 0 || len(parts) > 2 {
		return "", fmt.Errorf("invalid table %q: use table or schema.table", name)
	}
	for i, part := range parts {
		if part == "" || strings.TrimSpace(part) != part || strings.ContainsRune(part, '\x00') {
			return "", fmt.Errorf("invalid table %q: use table or schema.table", name)
		}
		parts[i] = `"` + strings.ReplaceAll(part, `"`, `""`) + `"`
	}
	return strings.Join(parts, "."), nil
}
