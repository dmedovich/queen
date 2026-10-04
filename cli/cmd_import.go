package cli

import (
	"github.com/dmedovich/queen/cli/commands/importcmd"
	"github.com/spf13/cobra"
)

func (app *App) importCmd() *cobra.Command {
	return importcmd.New()
}
