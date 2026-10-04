//go:build integration

package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dmedovich/queen"
	helpers "github.com/dmedovich/queen/tests/integration"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresCIPipeline(t *testing.T) {
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: helpers.PostgresImage(), ExposedPorts: []string{"5432/tcp"},
			Env:        map[string]string{"POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "queen_ci"},
			WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		if os.Getenv("QUEEN_REQUIRE_POSTGRES") == "1" {
			t.Fatalf("start PostgreSQL: %v", err)
		}
		t.Skipf("PostgreSQL container unavailable: %v", err)
	}
	defer func() { _ = container.Terminate(ctx) }()
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("postgres://test:test@%s:%s/", host, port.Port())
	testDSN := base + "queen_ci?sslmode=disable"
	db, err := sql.Open("pgx", testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.ExecContext(ctx, "CREATE DATABASE queen_deploy"); err != nil {
		t.Fatal(err)
	}
	prodDSN := base + "queen_deploy?sslmode=disable"

	// CI exercises apply, rollback, and reapply against its disposable database.
	output, err := executeCIMigrator(t, testDSN, "check", "--rollback-test", "--no-gaps", "--json")
	assertCheckJSON(t, output, err, 0)
	if !strings.Contains(output, "Testing rollback cycle... OK") {
		t.Fatalf("rollback check missing from JSON output: %s", output)
	}

	// A release with pending migrations must fail the post-deploy check.
	output, err = executeCIMigrator(t, prodDSN, "check", "--ci", "--json")
	assertCheckJSON(t, output, err, 5)
	for _, args := range [][]string{{"validate"}, {"plan"}, {"up", "--yes"}} {
		if output, err := executeCIMigrator(t, prodDSN, args...); err != nil {
			t.Fatalf("%s: %v\n%s", args[0], err, output)
		}
	}
	output, err = executeCIMigrator(t, prodDSN, "check", "--ci", "--no-gaps", "--json")
	assertCheckJSON(t, output, err, 0)
	if output, err := executeCIMigrator(t, prodDSN, "status"); err != nil || !strings.Contains(output, "create_ci_table") {
		t.Fatalf("status: err=%v output=%s", err, output)
	}
}

func executeCIMigrator(t *testing.T, dsn string, args ...string) (string, error) {
	t.Helper()
	app := newApp(func(q *queen.Queen) {
		q.MustAdd(queen.M{Version: "001", Name: "create_ci_table", UpSQL: "CREATE TABLE ci_table (id BIGINT PRIMARY KEY)", DownSQL: "DROP TABLE ci_table"})
		q.MustAdd(queen.M{Version: "002", Name: "add_ci_name", UpSQL: "ALTER TABLE ci_table ADD COLUMN name TEXT", DownSQL: "ALTER TABLE ci_table DROP COLUMN name"})
	}, nil)
	app.rootCmd.SetArgs(append([]string{"--driver", "postgres", "--dsn", dsn}, args...))
	var runErr error
	output := captureStdout(t, func() { runErr = app.rootCmd.Execute() })
	return output, runErr
}

func assertCheckJSON(t *testing.T, output string, runErr error, wantCode int) {
	t.Helper()
	var result checkJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("check output is not JSON: %v\n%s", err, output)
	}
	if result.ExitCode != wantCode {
		t.Fatalf("check exit_code = %d, want %d; output=%s", result.ExitCode, wantCode, output)
	}
	if wantCode == 0 {
		if runErr != nil || result.Failed != 0 {
			t.Fatalf("successful check: err=%v result=%+v", runErr, result)
		}
		return
	}
	var exitErr *exitCodeError
	if !errors.As(runErr, &exitErr) || exitErr.code != wantCode || result.Failed == 0 {
		t.Fatalf("failed check: err=%v result=%+v", runErr, result)
	}
}
