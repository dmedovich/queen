package cli

import (
	"fmt"

	"github.com/dmedovich/queen"
	"github.com/spf13/cobra"
)

func (app *App) verifyRegistryCmd() *cobra.Command {
	var dir string
	cmd := &cobra.Command{
		Use:   "verify-registry",
		Short: "Check that Go migration sources are linked into this binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			q := queen.New(nil)
			app.registerFunc(q)
			registered := q.RegisteredMigrations()
			if err := checkRegistry(dir, registered); err != nil {
				return err
			}
			fmt.Printf("Registry matches %d migration source(s)\n", len(registered))
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "migrations", "Directory containing Go migration sources")
	return cmd
}
