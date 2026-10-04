package cli

import (
	"strings"
	"testing"

	"github.com/dmedovich/queen"
)

func goosePlanMigrations() []queen.Migration {
	return []queen.Migration{
		{Version: "001", Name: "first", UpSQL: "SELECT 1;"},
		{Version: "002", Name: "second", UpSQL: "SELECT 2;"},
		{Version: "003", Name: "third", UpSQL: "SELECT 3;"},
	}
}

func TestBuildGooseAdoptionPlan(t *testing.T) {
	migrations := goosePlanMigrations()
	events := []gooseEvent{{ID: 1, Version: 0, Applied: true}, {ID: 2, Version: 1, Applied: true}, {ID: 3, Version: 2, Applied: true}}
	plan, err := buildGooseAdoptionPlan("goose_db_version", "queen_migrations", events, nil, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.ToAdopt) != 2 || plan.ToAdopt[0].Version != "001" || plan.ToAdopt[1].Version != "002" {
		t.Fatalf("versions to adopt = %+v", plan.ToAdopt)
	}
	if len(plan.Fingerprint) != 64 {
		t.Fatalf("fingerprint = %q", plan.Fingerprint)
	}
	changed := append([]gooseEvent(nil), events...)
	changed[2].ID = 4
	other, err := buildGooseAdoptionPlan("goose_db_version", "queen_migrations", changed, nil, migrations)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Fingerprint == other.Fingerprint {
		t.Fatal("changed Goose history did not invalidate the plan")
	}
	current := []queenHistoryRow{
		{Version: "002", Name: "second", Checksum: migrations[1].Checksum(), Status: "success"},
		{Version: "001", Name: "first", Checksum: migrations[0].Checksum(), Status: "success"},
	}
	adopted, err := buildGooseAdoptionPlan("goose_db_version", "queen_migrations", events, current, migrations)
	if err != nil || !adopted.AlreadyDone {
		t.Fatalf("already adopted plan = %+v, err = %v", adopted, err)
	}
}

func TestBuildGooseAdoptionPlanRejectsMismatch(t *testing.T) {
	base := []gooseEvent{{ID: 1, Version: 0, Applied: true}, {ID: 2, Version: 1, Applied: true}}
	withManualChecksum := goosePlanMigrations()
	withManualChecksum[0].ManualChecksum = "unchecked"
	for _, tc := range []struct {
		name       string
		events     []gooseEvent
		current    []queenHistoryRow
		migrations []queen.Migration
		want       string
	}{
		{"gap", []gooseEvent{{ID: 1, Version: 0, Applied: true}, {ID: 2, Version: 2, Applied: true}}, nil, goosePlanMigrations(), "gap"},
		{"unknown Goose version", []gooseEvent{{ID: 1, Version: 0, Applied: true}, {ID: 2, Version: 4, Applied: true}}, nil, goosePlanMigrations(), "missing from Queen"},
		{"rolled back latest event", []gooseEvent{{ID: 1, Version: 0, Applied: true}, {ID: 2, Version: 1, Applied: true}, {ID: 3, Version: 1, Applied: false}}, nil, goosePlanMigrations(), "no applied"},
		{"partial Queen history", base, []queenHistoryRow{{Version: "003", Status: "success"}}, goosePlanMigrations(), "queen history differs"},
		{"numeric alias", base, nil, append(goosePlanMigrations(), queen.Migration{Version: "1", Name: "duplicate", UpSQL: "SELECT 4;"}), "same Goose version"},
		{"manual checksum", base, nil, withManualChecksum, "computed checksum"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildGooseAdoptionPlan("goose_db_version", "queen_migrations", tc.events, tc.current, tc.migrations)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestQuotePostgresTable(t *testing.T) {
	got, err := quotePostgresTable(`app"schema.goose"history`)
	if err != nil || got != `"app""schema"."goose""history"` {
		t.Fatalf("quoted table = %q, err = %v", got, err)
	}
	for _, name := range []string{"", "a..b", "a.", ".a", "a.b.c"} {
		if _, err := quotePostgresTable(name); err == nil {
			t.Fatalf("invalid table %q was accepted", name)
		}
	}
}
