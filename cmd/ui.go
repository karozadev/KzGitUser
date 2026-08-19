package cmd

import (
	"github.com/karoza/kz-git-user/internal/tui"
	"github.com/spf13/cobra"
)

func newUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: "Launch the interactive Terminal User Interface",
		Long: `Launch KzGitUser's interactive Terminal User Interface (TUI).

The TUI provides a visual dashboard for managing your Git identity profiles,
viewing repository information, and switching between profiles.

Keyboard shortcuts:
  1-3       Switch between pages
  j/k       Navigate lists
  Enter     Select/Confirm
  Esc       Back/Close
  /         Command mode
  ?         Help
  q         Quit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := tui.NewApp()
			return app.Run()
		},
	}
}
