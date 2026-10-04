package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/dmedovich/queen"
)

func (app *App) recoverCmd() *cobra.Command {
	var state string
	var verified bool
	cmd := &cobra.Command{
		Use:   "recover VERSION",
		Short: "Resolve an interrupted non-transactional migration after inspecting the database",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if state != queen.RecoveryApplied && state != queen.RecoveryNotApplied {
				return fmt.Errorf("--state must be %q or %q", queen.RecoveryApplied, queen.RecoveryNotApplied)
			}
			if !verified {
				return fmt.Errorf("inspect the database schema, then pass --verified to confirm it matches --state %s", state)
			}
			if err := app.checkConfirmation(fmt.Sprintf("recover migration %s as %s", args[0], state)); err != nil {
				return err
			}
			return app.runWithQueen(cmd, func(ctx context.Context, q *queen.Queen) error {
				if err := q.ResolveIncomplete(ctx, args[0], state); err != nil {
					return err
				}
				fmt.Printf("Resolved migration %s as %s\n", args[0], state)
				return nil
			})
		},
	}
	cmd.Flags().StringVar(&state, "state", "", "Verified database state: applied or not-applied")
	cmd.Flags().BoolVar(&verified, "verified", false, "Confirm that the database was inspected and matches --state")
	return cmd
}
