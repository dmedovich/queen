package cli

import (
	"github.com/spf13/cobra"
	"github.com/dmedovich/queen/cli/commands/importcmd"
)

func (app *App) importCmd() *cobra.Command {
	return importcmd.New()
}
