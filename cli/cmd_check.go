package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (app *App) checkCmd() *cobra.Command {
	var (
		ci           bool
		noPending    bool
		noGaps       bool
		rollbackTest bool
	)

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Quick validation for CI/CD pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := commandContext(cmd)
			q, err := app.setupQueen(ctx)
			if err != nil {
				if app.config.JSON {
					if encodeErr := json.NewEncoder(os.Stdout).Encode(checkJSONResult{Passed: 0, Failed: 1, ExitCode: 2, Output: fmt.Sprintf("Configuration error: %v", err)}); encodeErr != nil {
						return encodeErr
					}
					return &exitCodeError{code: 2, quiet: true, cause: err}
				}
				return &exitCodeError{code: 2, cause: fmt.Errorf("configuration error: %w", err)}
			}
			defer func() { _ = q.Close() }()

			var output bytes.Buffer
			options := checkOptions{
				ci:           ci,
				noPending:    noPending,
				noGaps:       noGaps,
				rollbackTest: rollbackTest,
			}
			if app.config.JSON {
				options.output = &output
			}
			summary := runPipelineChecks(ctx, q, options)

			if app.config.JSON {
				if err := json.NewEncoder(os.Stdout).Encode(checkJSONResult{Passed: summary.passed, Failed: summary.failed, ExitCode: summary.exitCode, Output: output.String()}); err != nil {
					return err
				}
			} else {
				outputCheckSummary(summary)
			}
			if summary.failed > 0 {
				return &exitCodeError{code: summary.exitCode, quiet: true}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&ci, "ci", false, "CI mode (strict validation, fails on warnings)")
	cmd.Flags().BoolVar(&noPending, "no-pending", false, "Fail if pending migrations exist")
	cmd.Flags().BoolVar(&noGaps, "no-gaps", false, "Fail if gaps are detected")
	cmd.Flags().BoolVar(&rollbackTest, "rollback-test", false, "Run an up/reset/up rollback cycle against a clean test database")

	return cmd
}

type checkJSONResult struct {
	Passed   int    `json:"passed"`
	Failed   int    `json:"failed"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

func outputCheckSummary(summary checkSummary) {
	fmt.Println()
	if summary.failed > 0 {
		fmt.Printf("Checks: %d passed, %d failed\n", summary.passed, summary.failed)
		return
	}

	total := summary.passed + summary.failed
	fmt.Printf("All checks passed (%d/%d)\n", summary.passed, total)
}
