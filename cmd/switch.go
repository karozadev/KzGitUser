package cmd

import (
	"fmt"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/spf13/cobra"
)

func newSwitchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "switch <profile>",
		Short: "Apply a saved profile to the current repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(cmd, args[0])
		},
	}
}

func runSwitch(cmd *cobra.Command, name string) error {
	if !kzgit.Available() {
		return fmt.Errorf("git is not installed or not found in PATH")
	}

	repo, err := kzgit.OpenRepository(".")
	if err != nil {
		return fmt.Errorf("not a git repository (or any parent up to the filesystem root)")
	}

	cfg, err := kzconfig.Load()
	if err != nil {
		return err
	}

	profile, err := cfg.Get(name)
	if err != nil {
		return fmt.Errorf("profile %q does not exist. Run 'kzgit profiles list' to see available profiles", name)
	}

	if err := repo.SetLocalIdentity(profile.Name, profile.Email); err != nil {
		return fmt.Errorf("applying profile %q: %w", name, err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Switched to profile %q (%s <%s>) in %s\n", name, profile.Name, profile.Email, repo.Path)
	return nil
}
