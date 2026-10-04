package cli

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitializeProjectGeneratedFilesAreUsable(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := oldWd
	if filepath.Base(repoRoot) == "cli" {
		repoRoot = filepath.Dir(repoRoot)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}

	if err := initializeProject(DriverPostgres, true, "migrations"); err != nil {
		t.Fatalf("initializeProject: %v", err)
	}

	mainPath := filepath.Join("cmd", "migrate", "main.go")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read generated main.go: %v", err)
	}
	for _, forbidden := range []string{`"database/sql"`, `"fmt"`, `"log"`} {
		if strings.Contains(string(mainContent), forbidden) {
			t.Fatalf("generated main.go contains unused import %s", forbidden)
		}
	}

	if err := os.WriteFile("go.mod", []byte(`module yourmodule

go 1.26.3

replace github.com/dmedovich/queen => `+repoRoot+`

require github.com/dmedovich/queen v0.0.0
`), 0644); err != nil {
		t.Fatalf("write temp go.mod: %v", err)
	}

	runGeneratedProjectCommand(t, "mod", "tidy")
	runGeneratedProjectCommand(t, "test", "./...")
	runGeneratedProjectCommand(t, "build", "-o", "migrate", "./cmd/migrate")
	check := exec.Command("./migrate", "check", "--json")
	check.Env = append(os.Environ(), "QUEEN_DRIVER=", "QUEEN_DSN=")
	checkOutput, checkErr := check.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(checkErr, &exitErr) || exitErr.ExitCode() != 2 {
		t.Fatalf("check process exit = %v, want code 2; output=%s", checkErr, checkOutput)
	}
	var checkResult checkJSONResult
	if err := json.Unmarshal(checkOutput, &checkResult); err != nil || checkResult.ExitCode != 2 {
		t.Fatalf("check process JSON = %+v, err=%v; output=%s", checkResult, err, checkOutput)
	}
	runGeneratedProjectCommand(t, "run", "./cmd/migrate", "verify-registry")
	runGeneratedProjectCommand(t, "run", "./cmd/migrate", "create", "add_email")
	runGeneratedProjectCommand(t, "run", "./cmd/migrate", "verify-registry")
	if err := os.WriteFile(filepath.Join("migrations", "003_unregistered.go"), []byte(generateSQLTemplate("003", "unregistered", "Migration003Unregistered")), 0644); err != nil {
		t.Fatal(err)
	}
	verify := exec.Command("go", "run", "./cmd/migrate", "verify-registry")
	verify.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	output, err := verify.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "unregistered source migrations: [003") {
		t.Fatalf("verify-registry should reject an unregistered migration: err=%v output=%s", err, output)
	}

	app := &App{
		config: &Config{
			UseConfig:        true,
			Env:              "production",
			Table:            DefaultTableName,
			UnlockProduction: true,
		},
	}
	if err := app.loadConfigFile(); err != nil {
		t.Fatalf("generated .queen.yaml must load production env: %v", err)
	}
	if app.config.Driver != DriverPostgres {
		t.Fatalf("loaded driver = %q, want %q", app.config.Driver, DriverPostgres)
	}
	if app.config.DSN != "${DATABASE_URL}" {
		t.Fatalf("loaded DSN = %q, want ${DATABASE_URL}", app.config.DSN)
	}
	if !app.requiresConfirmation() {
		t.Fatal("production env from generated config must require confirmation")
	}
}

func runGeneratedProjectCommand(t *testing.T, args ...string) {
	t.Helper()

	cmd := exec.Command("go", args...)
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated project command go %q failed: %v\n%s", args, err, out)
	}
}

func TestInitializeProjectDoesNotOverwriteExistingFiles(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll("migrations", 0755); err != nil {
		t.Fatal(err)
	}
	existingPath := filepath.Join("migrations", "migrations.go")
	if err := os.WriteFile(existingPath, []byte("sentinel"), 0644); err != nil {
		t.Fatal(err)
	}

	err = initializeProject(DriverPostgres, true, "migrations")
	if err == nil {
		t.Fatal("initializeProject succeeded despite existing migrations.go")
	}
	if !strings.Contains(err.Error(), "file already exists") {
		t.Fatalf("initializeProject error = %v, want file already exists", err)
	}

	content, err := os.ReadFile(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "sentinel" {
		t.Fatalf("existing file was overwritten: %q", content)
	}
}
