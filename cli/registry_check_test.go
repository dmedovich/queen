package cli

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmedovich/queen"
)

func writeMigrationSource(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRegistry(t *testing.T) {
	const source = `package migrations
import q "github.com/dmedovich/queen"
var First = q.M{Version: "001", Name: "first", UpSQL: "SELECT 1"}
`
	tests := []struct {
		name       string
		files      map[string]string
		registered []queen.Migration
		wantError  string
	}{
		{"matching", map[string]string{"001_first.go": source}, []queen.Migration{{Version: "001"}}, ""},
		{"forgotten registration", map[string]string{"001_first.go": source}, nil, "unregistered source migrations: [001"},
		{"stale binary", map[string]string{"001_first.go": source}, []queen.Migration{{Version: "001"}, {Version: "002"}}, "registered without a source declaration: [002]"},
		{"missing Go checksum", map[string]string{"001_first.go": source}, []queen.Migration{{Version: "001", UpFunc: func(_ context.Context, _ *sql.Tx) error { return nil }}}, "requires ManualChecksum"},
		{"duplicate declaration", map[string]string{"001_first.go": source, "002_duplicate.go": source}, []queen.Migration{{Version: "001"}}, "declared in both"},
		{"dynamic version", map[string]string{"001_first.go": `package migrations
import "github.com/dmedovich/queen"
var Version = "001"
var First = queen.M{Version: Version, Name: "first", UpSQL: "SELECT 1"}
`}, []queen.Migration{{Version: "001"}}, "requires a literal Version"},
		{"no declarations", map[string]string{"migrations.go": "package migrations\n"}, nil, "no Queen migration declarations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, content := range tt.files {
				writeMigrationSource(t, dir, name, content)
			}
			err := checkRegistry(dir, tt.registered)
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("checkRegistry: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("checkRegistry error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestVerifyRegistryCommandNeedsNoDatabase(t *testing.T) {
	dir := t.TempDir()
	writeMigrationSource(t, dir, "001_first.go", `package migrations
import "github.com/dmedovich/queen"
var First = queen.M{Version: "001", Name: "first", UpSQL: "SELECT 1"}
`)
	app := newApp(func(q *queen.Queen) {
		q.MustAdd(queen.M{Version: "001", Name: "first", UpSQL: "SELECT 1"})
	}, nil)
	cmd := app.verifyRegistryCmd()
	cmd.SetArgs([]string{"--dir", dir})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
