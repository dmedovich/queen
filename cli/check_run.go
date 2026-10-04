package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/dmedovich/queen"
)

type checkOptions struct {
	ci           bool
	noPending    bool
	noGaps       bool
	rollbackTest bool
	output       io.Writer
}

type checkSummary struct {
	passed   int
	failed   int
	exitCode int
	output   io.Writer
	writeErr error
}

func runPipelineChecks(ctx context.Context, q *queen.Queen, opts checkOptions) checkSummary {
	summary := checkSummary{output: opts.output}
	if summary.output == nil {
		summary.output = os.Stdout
	}

	summary.write("Validating migrations... ")
	if err := q.Validate(ctx); err != nil {
		summary.write("FAIL\n  %v\n", err)
		summary.fail(3)
	} else {
		summary.write("OK\n")
		summary.pass()
	}

	summary.write("Checking for gaps... ")
	gaps, err := q.DetectGaps(ctx)
	if err != nil {
		summary.write("FAIL\n  %v\n", err)
		summary.fail(3)
	} else if len(gaps) > 0 {
		summary.write("WARNING (%d gaps found)\n", len(gaps))
		if opts.noGaps || opts.ci {
			summary.fail(4)
		} else {
			summary.pass()
		}
	} else {
		summary.write("OK\n")
		summary.pass()
	}

	if opts.noPending || opts.ci {
		checkPendingMigrations(ctx, q, &summary)
	}

	if opts.rollbackTest {
		checkRollbackCycle(ctx, q, &summary)
	}

	summary.write("Database connectivity... ")
	if _, err := q.Driver().GetApplied(ctx); err != nil {
		summary.write("FAIL\n  %v\n", err)
		summary.fail(3)
	} else {
		summary.write("OK\n")
		summary.pass()
	}

	return summary
}

func checkRollbackCycle(ctx context.Context, q *queen.Queen, summary *checkSummary) {
	summary.write("Testing rollback cycle... ")

	statuses, err := q.Status(ctx)
	if err != nil {
		summary.write("FAIL\n  %v\n", err)
		summary.fail(6)
		return
	}

	appliedCount := 0
	for _, s := range statuses {
		if s.Status == queen.StatusApplied || s.Status == queen.StatusModified {
			appliedCount++
		}
	}
	if appliedCount > 0 {
		summary.write("FAIL\n  rollback-test requires a clean test database; found %d applied migration(s)\n", appliedCount)
		summary.fail(6)
		return
	}

	if len(statuses) == 0 {
		summary.write("FAIL\n  no migrations registered\n")
		summary.fail(6)
		return
	}

	if err := q.Up(ctx); err != nil {
		summary.write("FAIL\n  apply failed: %v\n", err)
		summary.fail(6)
		return
	}
	if err := q.Reset(ctx); err != nil {
		summary.write("FAIL\n  rollback failed: %v\n", err)
		summary.fail(6)
		return
	}
	if err := q.Up(ctx); err != nil {
		summary.write("FAIL\n  reapply failed: %v\n", err)
		summary.fail(6)
		return
	}

	summary.write("OK\n")
	summary.pass()
}

func checkPendingMigrations(ctx context.Context, q *queen.Queen, summary *checkSummary) {
	summary.write("Checking for pending migrations... ")
	statuses, err := q.Status(ctx)
	if err != nil {
		summary.write("FAIL\n  %v\n", err)
		summary.fail(3)
		return
	}

	pendingCount := 0
	for _, s := range statuses {
		if s.Status == queen.StatusPending {
			pendingCount++
		}
	}
	if pendingCount > 0 {
		summary.write("FAIL (%d pending)\n", pendingCount)
		summary.fail(5)
		return
	}

	summary.write("OK\n")
	summary.pass()
}

func (s *checkSummary) write(format string, args ...any) {
	if s.writeErr != nil {
		return
	}
	out := s.output
	if out == nil {
		out = os.Stdout
	}
	_, s.writeErr = fmt.Fprintf(out, format, args...)
}

func (s *checkSummary) pass() {
	s.passed++
}

func (s *checkSummary) fail(exitCode int) {
	s.failed++
	if s.exitCode == 0 {
		s.exitCode = exitCode
	}
}
