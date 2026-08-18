package cmd

import (
	"fmt"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	"github.com/spf13/cobra"
)

func newProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage saved Git identity profiles",
		RunE:  runProfilesList,
	}

	cmd.AddCommand(newProfilesAddCmd())
	cmd.AddCommand(newProfilesListCmd())
	cmd.AddCommand(newProfilesRemoveCmd())
	return cmd
}

func newProfilesAddCmd() *cobra.Command {
	var name, email string

	cmd := &cobra.Command{
		Use:   "add <profile>",
		Short: "Add a new profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesAdd(cmd, args[0], name, email)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name for this profile (required)")
	cmd.Flags().StringVar(&email, "email", "", "email address for this profile (required)")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("email")

	return cmd
}

func newProfilesListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		RunE:  runProfilesList,
	}
}

func newProfilesRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <profile>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a saved profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProfilesRemove(cmd, args[0])
		},
	}
}

func runProfilesAdd(cmd *cobra.Command, name, displayName, email string) error {
	cfg, err := kzconfig.Load()
	if err != nil {
		return err
	}

	if err := cfg.Add(name, displayName, email); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q added.\n", name)
	return nil
}

func runProfilesList(cmd *cobra.Command, _ []string) error {
	cfg, err := kzconfig.Load()
	if err != nil {
		return err
	}

	profiles := cfg.List()
	out := cmd.OutOrStdout()

	_, _ = fmt.Fprintf(out, "Profiles\n%s\n", separator)
	if len(profiles) == 0 {
		_, _ = fmt.Fprintln(out, "No profiles saved yet. Add one with: kzgit profiles add <name> --name \"...\" --email \"...\"")
		return nil
	}

	maxLen := 0
	for _, p := range profiles {
		if len(p.Name) > maxLen {
			maxLen = len(p.Name)
		}
	}
	for _, p := range profiles {
		_, _ = fmt.Fprintf(out, "%-*s    %s\n", maxLen, p.Name, p.Email)
	}
	return nil
}

func runProfilesRemove(cmd *cobra.Command, name string) error {
	cfg, err := kzconfig.Load()
	if err != nil {
		return err
	}

	if err := cfg.Remove(name); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Profile %q removed.\n", name)
	return nil
}
