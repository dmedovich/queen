package cli

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dmedovich/queen"
)

func inspectGooseAdoption(ctx context.Context, tx *sql.Tx, sourceTable, targetTable, sourceRef, targetRef string, registered []queen.Migration) (gooseAdoptionPlan, error) {
	var sourceOID, targetOID sql.NullInt64
	if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1)::oid, to_regclass($2)::oid", sourceRef, targetRef).Scan(&sourceOID, &targetOID); err != nil {
		return gooseAdoptionPlan{}, fmt.Errorf("locate migration history tables: %w", err)
	}
	if !sourceOID.Valid {
		return gooseAdoptionPlan{}, fmt.Errorf("goose history table %s does not exist", sourceTable)
	}
	if targetOID.Valid && sourceOID.Int64 == targetOID.Int64 {
		return gooseAdoptionPlan{}, fmt.Errorf("goose and Queen history resolve to the same table")
	}
	events, err := readGooseEvents(ctx, tx, sourceRef)
	if err != nil {
		return gooseAdoptionPlan{}, err
	}
	var current []queenHistoryRow
	if targetOID.Valid {
		current, err = readQueenHistory(ctx, tx, targetRef)
		if err != nil {
			return gooseAdoptionPlan{}, err
		}
	}
	return buildGooseAdoptionPlan(sourceTable, targetTable, events, current, registered)
}

func readGooseEvents(ctx context.Context, tx *sql.Tx, tableRef string) ([]gooseEvent, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT id, version_id, is_applied FROM %s ORDER BY id", tableRef))
	if err != nil {
		return nil, fmt.Errorf("read Goose history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var events []gooseEvent
	for rows.Next() {
		var event gooseEvent
		if err := rows.Scan(&event.ID, &event.Version, &event.Applied); err != nil {
			return nil, fmt.Errorf("read Goose history row: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read Goose history: %w", err)
	}
	return events, nil
}

func readQueenHistory(ctx context.Context, tx *sql.Tx, tableRef string) ([]queenHistoryRow, error) {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf("SELECT version, name, checksum, COALESCE(status, '') FROM %s", tableRef))
	if err != nil {
		return nil, fmt.Errorf("read Queen history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var history []queenHistoryRow
	for rows.Next() {
		var entry queenHistoryRow
		if err := rows.Scan(&entry.Version, &entry.Name, &entry.Checksum, &entry.Status); err != nil {
			return nil, fmt.Errorf("read Queen history row: %w", err)
		}
		history = append(history, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read Queen history: %w", err)
	}
	return history, nil
}

func printGooseAdoptionPlan(plan gooseAdoptionPlan) {
	fmt.Printf("Goose history: %s\nQueen history: %s\n", plan.SourceTable, plan.TargetTable)
	if plan.AlreadyDone {
		fmt.Printf("Queen already contains the same %d applied migration(s).\n", len(plan.ToAdopt))
	} else {
		fmt.Printf("Would record %d applied Goose migration(s) in Queen:\n", len(plan.ToAdopt))
		for _, migration := range plan.ToAdopt {
			fmt.Printf("  %s - %s\n", migration.Version, migration.Name)
		}
	}
	fmt.Printf("Plan: %s\n", plan.Fingerprint)
	fmt.Printf("After checking the schema and stopping Goose migrators, run: queen adopt-goose --apply --plan %s --verified-schema\n", plan.Fingerprint)
}
