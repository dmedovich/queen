package cli

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dmedovich/queen"
)

func TestOutputCheckSummaryCountsOnlyExecutedChecks(t *testing.T) {
	out := captureStdout(t, func() {
		outputCheckSummary(checkSummary{passed: 3})
	})
	if !strings.Contains(out, "All checks passed (3/3)") {
		t.Fatalf("output summary mismatch:\n%s", out)
	}
	if strings.Contains(out, "All checks passed (3/0)") {
		t.Fatalf("output used stale denominator:\n%s", out)
	}
}

func TestCheckJSONReportsPendingWithExitCode(t *testing.T) {
	app := newApp(func(q *queen.Queen) {
		q.MustAdd(queen.M{Version: "001", Name: "create_users", UpSQL: "CREATE TABLE users (id INTEGER PRIMARY KEY)"})
	}, nil)
	app.rootCmd.SetArgs([]string{"check", "--driver", "sqlite", "--dsn", filepath.Join(t.TempDir(), "queen.db"), "--ci", "--json"})
	var runErr error
	output := captureStdout(t, func() { runErr = app.rootCmd.Execute() })
	var exitErr *exitCodeError
	if !errors.As(runErr, &exitErr) || exitErr.code != 5 {
		t.Fatalf("Execute error = %v, want exit code 5", runErr)
	}
	var result checkJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output)
	}
	if result.Passed != 3 || result.Failed != 1 || result.ExitCode != 5 || !strings.Contains(result.Output, "1 pending") {
		t.Fatalf("check JSON result = %+v", result)
	}
}

func TestCheckJSONReportsConfigurationError(t *testing.T) {
	app := newApp(nil, nil)
	app.rootCmd.SetArgs([]string{"check", "--json"})
	var runErr error
	output := captureStdout(t, func() { runErr = app.rootCmd.Execute() })
	var exitErr *exitCodeError
	if !errors.As(runErr, &exitErr) || exitErr.code != 2 {
		t.Fatalf("Execute error = %v, want exit code 2", runErr)
	}
	var result checkJSONResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, output)
	}
	if result.Failed != 1 || result.ExitCode != 2 || !strings.Contains(result.Output, "driver is required") {
		t.Fatalf("check JSON result = %+v", result)
	}
}

func TestOutputCheckSummaryReportsFailures(t *testing.T) {
	out := captureStdout(t, func() {
		outputCheckSummary(checkSummary{passed: 2, failed: 1, exitCode: 3})
	})
	if !strings.Contains(out, "Checks: 2 passed, 1 failed") {
		t.Fatalf("output summary mismatch:\n%s", out)
	}
	if strings.Contains(out, "All checks passed") {
		t.Fatalf("failure output claimed success:\n%s", out)
	}
}
