package cmd

import (
	kzconfig "github.com/karoza/kz-git-user/internal/config"
	"github.com/spf13/cobra"
)

// completeProfileNames provides shell completion for commands that take a
// saved profile name as their sole argument (switch, profiles remove).
func completeProfileNames(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cfg, err := kzconfig.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	names := make([]string, 0, len(cfg.Profiles))
	for _, p := range cfg.List() {
		names = append(names, p.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
