package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareRegistration(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantAuto bool
		wantText string
		wantErr  bool
	}{
		{"standard", "package migrations\nimport \"github.com/dmedovich/queen\"\nfunc Register(q *queen.Queen) {\n\tRegister001(q)\n}\n", true, "q.MustAdd(Migration002AddEmail)", false},
		{"renamed parameter", "package migrations\nimport q \"github.com/dmedovich/queen\"\nfunc Register(registry *q.Queen) { Register001(registry) }\n", true, "registry.MustAdd(Migration002AddEmail)", false},
		{"other Register", "package migrations\nfunc Register(q *Other) {}\n", false, "", false},
		{"custom registry", "package migrations\nfunc RegisterAll(q *Queen) {}\n", false, "", false},
		{"invalid source", "package migrations\nfunc Register( {\n", false, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "migrations.go")
			if err := os.WriteFile(path, []byte(tt.source), 0644); err != nil {
				t.Fatal(err)
			}
			updated, auto, err := prepareRegistration(path, "Migration002AddEmail")
			if (err != nil) != tt.wantErr {
				t.Fatalf("prepareRegistration error = %v, wantErr %v", err, tt.wantErr)
			}
			if auto != tt.wantAuto {
				t.Fatalf("auto = %v, want %v", auto, tt.wantAuto)
			}
			if tt.wantText != "" && !strings.Contains(string(updated), tt.wantText) {
				t.Fatalf("updated registry missing %q:\n%s", tt.wantText, updated)
			}
		})
	}
}

func TestPrepareRegistrationMissingFile(t *testing.T) {
	updated, auto, err := prepareRegistration(filepath.Join(t.TempDir(), "migrations.go"), "Migration002")
	if err != nil || auto || updated != nil {
		t.Fatalf("missing registry: updated=%s auto=%v err=%v", updated, auto, err)
	}
}
