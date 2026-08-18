package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version, commit and date are set at build time via -ldflags, e.g.:
//
//	go build -ldflags "-X github.com/karoza/kz-git-user/cmd.version=1.0.0" .
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the kzgit version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "kzgit version %s (commit %s, built %s)\n", version, commit, date)
			return nil
		},
	}
}
