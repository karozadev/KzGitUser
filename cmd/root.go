// Package cmd implements the kzgit command-line interface.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kzgit",
		Short: "Visualize, verify and switch your Git identity",
		Long: `KzGitUser (kzgit) helps you visualize, verify and easily switch the
active Git identity (user.name and user.email) in a repository, so you
never accidentally commit with the wrong professional or personal email.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runWhoami,
	}

	root.AddCommand(newWhoamiCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newProfilesCmd())
	root.AddCommand(newSwitchCmd())
	root.AddCommand(newCheckCmd())

	return root
}

// Execute builds and runs the kzgit command tree, printing any error to
// stderr and returning the process exit code.
func Execute() int {
	cmd := newRootCmd()
	if err := cmd.Execute(); err != nil {
		if !errors.Is(err, errCheckFailed) {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return 1
	}
	return 0
}
