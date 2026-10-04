//go:build integration

package importcmd

import (
	"context"
	"database/sql"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dmedovich/queen"
	"github.com/dmedovich/queen/drivers/postgres"
	helpers "github.com/dmedovich/queen/tests/integration"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestGooseImportPostgresRoundTrip(t *testing.T) {
	source := t.TempDir()
	output := filepath.Join(t.TempDir(), "migrations")
	for name, body := range map[string]string{
		"001_create_table.sql":     "-- +goose Up\nCREATE TABLE goose_import_body (id INT);\nINSERT INTO goose_import_body VALUES (1);\n-- +goose Down\nDELETE FROM goose_import_body;\nDROP TABLE goose_import_body;",
		"002_concurrent_index.sql": "-- +goose NO TRANSACTION\n-- +goose Up\nCREATE INDEX CONCURRENTLY goose_import_idx ON goose_import_body (id);\n-- +goose Down\nDROP INDEX CONCURRENTLY goose_import_idx;",
		"003_create_function.sql":  "-- +goose Up\n-- +goose StatementBegin\nCREATE FUNCTION goose_import_fn() RETURNS integer LANGUAGE plpgsql AS $$ BEGIN RETURN 42; END; $$;\n-- +goose StatementEnd\n-- +goose Down\nDROP FUNCTION goose_import_fn();",
	} {
		if err := os.WriteFile(filepath.Join(source, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := importFromGoose(source, output, false); err != nil {
		t.Fatal(err)
	}
	generated, err := filepath.Glob(filepath.Join(output, "*.go"))
	if err != nil || len(generated) != 4 {
		t.Fatalf("generated files = %v, err = %v", generated, err)
	}
	for _, file := range generated {
		if _, err := parser.ParseFile(token.NewFileSet(), file, nil, 0); err != nil {
			t.Fatalf("invalid generated Go in %s: %v", file, err)
		}
	}
	parsed, err := scanGooseMigrations(source, false)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: helpers.PostgresImage(), ExposedPorts: []string{"5432/tcp"},
		Env:        map[string]string{"POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "testdb"},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(90 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true})
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
	db, err := sql.Open("pgx", "postgres://test:test@"+host+":"+port.Port()+"/testdb?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	q := queen.New(postgres.NewWithTableName(db, "queen_goose_import_test"))
	for _, migration := range parsed {
		q.MustAdd(queen.M{Version: migration.version, Name: migration.name, UpSQL: migration.upSQL, DownSQL: migration.downSQL, NonTransactional: migration.noTransaction})
	}
	if err := q.Up(ctx); err != nil {
		t.Fatalf("apply imported migrations: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM goose_import_body").Scan(&count); err != nil || count != 1 {
		t.Fatalf("imported table count = %d, err = %v", count, err)
	}
	var index sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('goose_import_idx')").Scan(&index); err != nil || !index.Valid {
		t.Fatalf("imported index = %v, err = %v", index, err)
	}
	var functionResult int
	if err := db.QueryRowContext(ctx, "SELECT goose_import_fn()").Scan(&functionResult); err != nil || functionResult != 42 {
		t.Fatalf("imported function returned %d, err = %v", functionResult, err)
	}
	if err := q.Down(ctx, 3); err != nil {
		t.Fatalf("rollback imported migrations: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT to_regclass('goose_import_body')").Scan(&index); err != nil || index.Valid {
		t.Fatalf("table after rollback = %v, err = %v", index, err)
	}
}
