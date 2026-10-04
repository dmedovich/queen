package importcmd

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGooseSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		input             string
		wantUp            string
		wantDown          string
		wantNoTransaction bool
		wantErr           string
	}{
		{
			name: "standard goose migration",
			input: `-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL
);

-- +goose Down
DROP TABLE users;`,
			wantUp:   "CREATE TABLE users (\n    id SERIAL PRIMARY KEY,\n    email VARCHAR(255) NOT NULL\n);",
			wantDown: "DROP TABLE users;",
		},
		{
			name: "up only, no down",
			input: `-- +goose Up
ALTER TABLE users ADD COLUMN name VARCHAR(255);`,
			wantUp:   "ALTER TABLE users ADD COLUMN name VARCHAR(255);",
			wantDown: "",
		},
		{
			name: "multiple statements in up",
			input: `-- +goose Up
CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT);
CREATE INDEX idx_posts_title ON posts(title);

-- +goose Down
DROP INDEX idx_posts_title;
DROP TABLE posts;`,
			wantUp:   "CREATE TABLE posts (id SERIAL PRIMARY KEY, title TEXT);\nCREATE INDEX idx_posts_title ON posts(title);",
			wantDown: "DROP INDEX idx_posts_title;\nDROP TABLE posts;",
		},
		{
			name:    "empty file",
			input:   "",
			wantErr: "exactly one Up",
		},
		{
			name: "comments before directives",
			input: `-- This is a comment
-- Another comment

-- +goose Up
CREATE TABLE accounts (id INT);

-- +goose Down
DROP TABLE accounts;`,
			wantUp:   "CREATE TABLE accounts (id INT);",
			wantDown: "DROP TABLE accounts;",
		},
		{
			name: "goose StatementBegin and StatementEnd markers",
			input: `-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION IF EXISTS update_timestamp();`,
			wantUp:   "CREATE OR REPLACE FUNCTION update_timestamp()\nRETURNS TRIGGER AS $$\nBEGIN\n    NEW.updated_at = NOW();\n    RETURN NEW;\nEND;\n$$ LANGUAGE plpgsql;",
			wantDown: "DROP FUNCTION IF EXISTS update_timestamp();",
		},
		{name: "case insensitive annotation", input: "-- +goose up\nSELECT 1;\n-- +goose down\nSELECT 2;", wantUp: "SELECT 1;", wantDown: "SELECT 2;"},
		{name: "nontransactional annotation", input: "-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY idx ON t (id);", wantUp: "CREATE INDEX CONCURRENTLY idx ON t (id);", wantNoTransaction: true},
		{name: "env substitution rejected", input: "-- +goose Up\n-- +goose ENVSUB ON\nSELECT '${NAME}';", wantErr: "ENVSUB"},
		{name: "duplicate up rejected", input: "-- +goose Up\nSELECT 1;\n-- +goose Up\nSELECT 2;", wantErr: "duplicate Up"},
		{name: "unmatched statement marker rejected", input: "-- +goose Up\n-- +goose StatementBegin\nSELECT 1;", wantErr: "balanced statement markers"},
		{name: "empty down rejected", input: "-- +goose Up\nSELECT 1;\n-- +goose Down", wantErr: "down section is empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotUp, gotDown, noTransaction, err := parseGooseSQL(tt.input)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if noTransaction != tt.wantNoTransaction {
				t.Errorf("noTransaction = %t, want %t", noTransaction, tt.wantNoTransaction)
			}
			if gotUp != tt.wantUp {
				t.Errorf("upSQL mismatch\ngot:  %q\nwant: %q", gotUp, tt.wantUp)
			}
			if gotDown != tt.wantDown {
				t.Errorf("downSQL mismatch\ngot:  %q\nwant: %q", gotDown, tt.wantDown)
			}
		})
	}
}

func TestGenerateQueenMigrationFile(t *testing.T) {
	t.Parallel()

	result := generateQueenMigrationFile("001", "create_users", "Register001create_users",
		"CREATE TABLE users (id SERIAL PRIMARY KEY);",
		"DROP TABLE users;", false,
	)

	checks := []struct {
		name    string
		contain string
	}{
		{"package declaration", "package migrations"},
		{"queen import", `"github.com/dmedovich/queen"`},
		{"function name", "func Register001create_users(q *queen.Queen)"},
		{"version", `Version: "001"`},
		{"name", `Name:    "create_users"`},
		{"up SQL", "CREATE TABLE users (id SERIAL PRIMARY KEY);"},
		{"down SQL", "DROP TABLE users;"},
		{"MustAdd call", "q.MustAdd(queen.M{"},
	}

	for _, c := range checks {
		if !strings.Contains(result, c.contain) {
			t.Errorf("generated file missing %s: %q not found in output", c.name, c.contain)
		}
	}
}

func TestGenerateQueenMigrationFile_BacktickEscaping(t *testing.T) {
	t.Parallel()

	result := generateQueenMigrationFile("001", "add_json", "Register001add_json",
		"ALTER TABLE users ADD COLUMN meta JSON DEFAULT '`{}`';",
		"ALTER TABLE users DROP COLUMN meta;", false,
	)

	// Backticks in SQL should be escaped for Go raw string literals
	if strings.Count(result, "` + \"`\" + `") < 1 {
		t.Error("backticks in SQL should be escaped in generated Go code")
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "generated.go", result, 0); err != nil {
		t.Fatalf("generated Go is invalid: %v", err)
	}
}

func TestImportFromGoose_NoTransaction(t *testing.T) {
	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "out")
	content := "-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY idx ON users (id);\n-- +goose Down\nDROP INDEX CONCURRENTLY idx;"
	if err := os.WriteFile(filepath.Join(sourceDir, "001_concurrent_index.sql"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := importFromGoose(sourceDir, outputDir, false); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(outputDir, "001_concurrent_index.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "NonTransactional: true") {
		t.Fatalf("NO TRANSACTION semantics lost:\n%s", body)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "001_concurrent_index.go", body, 0); err != nil {
		t.Fatalf("generated Go is invalid: %v", err)
	}
}

func TestImportFromGoose_RejectsUnsupportedInputWithoutOutput(t *testing.T) {
	for _, tc := range []struct{ name, files, want string }{
		{"envsub", "-- +goose Up\n-- +goose ENVSUB ON\nSELECT '${NAME}';", "ENVSUB"},
		{"multi no transaction", "-- +goose NO TRANSACTION\n-- +goose Up\nSELECT 1; SELECT 2;", "multiple SQL statements"},
		{"empty up", "-- +goose Up\n-- +goose Down\nSELECT 1;", "up section is empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sourceDir := t.TempDir()
			outputDir := filepath.Join(t.TempDir(), "out")
			if err := os.WriteFile(filepath.Join(sourceDir, "001_example.sql"), []byte(tc.files), 0644); err != nil {
				t.Fatal(err)
			}
			err := importFromGoose(sourceDir, outputDir, false)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("import error = %v, want %q", err, tc.want)
			}
			if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
				t.Fatalf("output created after rejected source: %v", err)
			}
		})
	}
}

func TestImportFromGoose_RejectsDuplicateVersions(t *testing.T) {
	sourceDir := t.TempDir()
	for _, name := range []string{"001_first.sql", "1_second.sql"} {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte("-- +goose Up\nSELECT 1;"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := importFromGoose(sourceDir, t.TempDir(), true); err == nil || !strings.Contains(err.Error(), "duplicate goose version") {
		t.Fatalf("duplicate version error = %v", err)
	}
}

func TestImportFromGoose_RejectsGoMigrationInMixedSource(t *testing.T) {
	sourceDir := t.TempDir()
	for name, body := range map[string]string{
		"001_create_users.sql": "-- +goose Up\nSELECT 1;",
		"002_backfill.go":      "package migrations",
	} {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	err := importFromGoose(sourceDir, filepath.Join(t.TempDir(), "out"), false)
	if err == nil || !strings.Contains(err.Error(), "cannot be converted automatically") {
		t.Fatalf("mixed source error = %v", err)
	}
}

func TestGenerateRegistrationFile(t *testing.T) {
	t.Parallel()

	calls := "\tRegister001create_users(q)\n\tRegister002add_posts(q)"
	result := generateRegistrationFile(calls)

	checks := []string{
		"package migrations",
		`"github.com/dmedovich/queen"`,
		"func Register(q *queen.Queen)",
		"Register001create_users(q)",
		"Register002add_posts(q)",
	}

	for _, c := range checks {
		if !strings.Contains(result, c) {
			t.Errorf("registration file missing: %q", c)
		}
	}
}

func TestImportFromGoose_EndToEnd(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "queen_migrations")

	gooseFiles := map[string]string{
		"001_create_users.sql": `-- +goose Up
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- +goose Down
DROP TABLE users;`,

		"002_create_posts.sql": `-- +goose Up
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    title TEXT NOT NULL,
    body TEXT
);

CREATE INDEX idx_posts_user_id ON posts(user_id);

-- +goose Down
DROP INDEX idx_posts_user_id;
DROP TABLE posts;`,

		"003_add_users_name.sql": `-- +goose Up
ALTER TABLE users ADD COLUMN name VARCHAR(255);

-- +goose Down
ALTER TABLE users DROP COLUMN name;`,
	}

	for name, content := range gooseFiles {
		if err := os.WriteFile(filepath.Join(sourceDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", name, err)
		}
	}

	err := importFromGoose(sourceDir, outputDir, false)
	if err != nil {
		t.Fatalf("importFromGoose() error = %v", err)
	}

	expectedFiles := []string{
		"migrations.go",
		"001_create_users.go",
		"002_create_posts.go",
		"003_add_users_name.go",
	}

	for _, name := range expectedFiles {
		path := filepath.Join(outputDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s was not created", name)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("failed to read %s: %v", name, err)
			continue
		}

		if !strings.Contains(string(content), "package migrations") {
			t.Errorf("%s missing package declaration", name)
		}
	}

	usersContent, _ := os.ReadFile(filepath.Join(outputDir, "001_create_users.go"))
	usersStr := string(usersContent)

	if !strings.Contains(usersStr, "CREATE TABLE users") {
		t.Error("001_create_users.go missing up SQL")
	}
	if !strings.Contains(usersStr, "DROP TABLE users") {
		t.Error("001_create_users.go missing down SQL")
	}
	if !strings.Contains(usersStr, `Version: "001"`) {
		t.Error("001_create_users.go missing version")
	}

	regContent, _ := os.ReadFile(filepath.Join(outputDir, "migrations.go"))
	regStr := string(regContent)

	if !strings.Contains(regStr, "func Register(q *queen.Queen)") {
		t.Error("migrations.go missing Register function")
	}

	if strings.Count(regStr, "(q)") < 3 {
		t.Errorf("migrations.go should have 3 registration calls, got %d", strings.Count(regStr, "(q)"))
	}
}

func TestImportFromGoose_DryRun(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "should_not_exist")

	content := `-- +goose Up
CREATE TABLE test (id INT);
-- +goose Down
DROP TABLE test;`
	if err := os.WriteFile(filepath.Join(sourceDir, "001_test.sql"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := importFromGoose(sourceDir, outputDir, true)
	if err != nil {
		t.Fatalf("dry-run should not error, got: %v", err)
	}

	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Error("dry-run should not create output directory")
	}
}

func TestImportFromGoose_PreservesGooseVersion(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "queen_migrations")

	content := `-- +goose Up
CREATE TABLE events (id INT);
-- +goose Down
DROP TABLE events;`
	if err := os.WriteFile(filepath.Join(sourceDir, "20240524054622_create_events.sql"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := importFromGoose(sourceDir, outputDir, false); err != nil {
		t.Fatalf("importFromGoose() error = %v", err)
	}

	generated := filepath.Join(outputDir, "20240524054622_create_events.go")
	body, err := os.ReadFile(generated)
	if err != nil {
		t.Fatalf("read generated migration: %v", err)
	}
	if !strings.Contains(string(body), `Version: "20240524054622"`) {
		t.Fatalf("generated migration did not preserve goose version:\n%s", body)
	}
}

func TestImportFromGoose_DoesNotOverwriteExistingMigration(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "queen_migrations")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	existingPath := filepath.Join(outputDir, "001_test.go")
	existingContent := []byte("package migrations\n\n// keep me\n")
	if err := os.WriteFile(existingPath, existingContent, 0644); err != nil {
		t.Fatal(err)
	}

	content := `-- +goose Up
CREATE TABLE test (id INT);
-- +goose Down
DROP TABLE test;`
	if err := os.WriteFile(filepath.Join(sourceDir, "001_test.sql"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := importFromGoose(sourceDir, outputDir, false)
	if err == nil {
		t.Fatal("expected import to fail when output file already exists")
	}
	if !strings.Contains(err.Error(), "file already exists") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, readErr := os.ReadFile(existingPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(existingContent) {
		t.Fatalf("existing migration was overwritten:\n%s", got)
	}
}

func TestImportFromGoose_DoesNotOverwriteRegistrationFile(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	outputDir := filepath.Join(t.TempDir(), "queen_migrations")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	existingPath := filepath.Join(outputDir, "migrations.go")
	existingContent := []byte("package migrations\n\nfunc Register() {}\n")
	if err := os.WriteFile(existingPath, existingContent, 0644); err != nil {
		t.Fatal(err)
	}

	content := `-- +goose Up
CREATE TABLE test (id INT);
-- +goose Down
DROP TABLE test;`
	if err := os.WriteFile(filepath.Join(sourceDir, "001_test.sql"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := importFromGoose(sourceDir, outputDir, false)
	if err == nil {
		t.Fatal("expected import to fail when migrations.go already exists")
	}
	if !strings.Contains(err.Error(), "file already exists") {
		t.Fatalf("unexpected error: %v", err)
	}

	got, readErr := os.ReadFile(existingPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(existingContent) {
		t.Fatalf("existing registration file was overwritten:\n%s", got)
	}
}

func TestImportFromGoose_NoFiles(t *testing.T) {
	t.Parallel()

	emptyDir := t.TempDir()
	err := importFromGoose(emptyDir, t.TempDir(), false)

	if err == nil {
		t.Fatal("expected error for empty source directory")
	}
	if !strings.Contains(err.Error(), "no goose migration files found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImportFromGoose_InvalidFileName(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(sourceDir, "invalid.sql"), []byte("-- +goose Up\nSELECT 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	err := importFromGoose(sourceDir, t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error when all files are invalid")
	}
	if !strings.Contains(err.Error(), "invalid goose migration filename") {
		t.Errorf("unexpected error: %v", err)
	}
}
