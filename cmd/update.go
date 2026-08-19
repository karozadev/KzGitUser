package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	kzupdate "github.com/karoza/kz-git-user/internal/update"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var (
		yes       bool
		checkOnly bool
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Check for and install a newer kzgit release",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, yes, checkOnly)
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "install the update without prompting for confirmation")
	cmd.Flags().BoolVar(&checkOnly, "check", false, "only check whether a new version is available; don't install it")

	return cmd
}

func runUpdate(cmd *cobra.Command, yes, checkOnly bool) error {
	out := cmd.OutOrStdout()

	current := effectiveVersion()

	_, _ = fmt.Fprintln(out, "Checking for updates...")
	info, err := kzupdate.ForceCheck(current)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if !info.Available {
		_, _ = fmt.Fprintf(out, "You're already on the latest version (v%s).\n", info.Latest)
		return nil
	}

	_, _ = fmt.Fprintf(out, "A new version of kzgit is available: v%s (current: %s)\n", info.Latest, versionLabel(current))
	printReleaseNotes(out, info.ReleaseNotes)

	if checkOnly {
		_, _ = fmt.Fprintln(out, "Run 'kzgit update' to install it.")
		return nil
	}

	if !yes {
		confirmed, err := confirm(cmd, "Update now?")
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(out, "Update skipped.")
			return nil
		}
	}

	_, _ = fmt.Fprintf(out, "Downloading kzgit v%s...\n", info.Latest)
	if err := kzupdate.Install(info.Latest); err != nil {
		return fmt.Errorf("installing update: %w", err)
	}

	_, _ = fmt.Fprintf(out, "Updated to kzgit v%s.\n", info.Latest)
	return nil
}

// effectiveVersion returns the running kzgit version for update comparisons,
// treating an unreleased "dev" build as always eligible to update.
func effectiveVersion() string {
	if version == "dev" {
		return "0.0.0"
	}
	return version
}

func versionLabel(current string) string {
	if version == "dev" {
		return "dev"
	}
	return "v" + current
}

// printReleaseNotes prints a release's changelog (as published in its
// GitHub release body) indented under a "Changelog:" heading. It is a
// no-op when notes is empty, e.g. for a release with no generated notes.
func printReleaseNotes(out io.Writer, notes string) {
	if notes == "" {
		return
	}
	_, _ = fmt.Fprintln(out, "\nChangelog:")
	for _, line := range strings.Split(notes, "\n") {
		_, _ = fmt.Fprintf(out, "  %s\n", line)
	}
}

// confirm prints a yes/no prompt and reads a single line from cmd's input,
// defaulting to "yes" on an empty answer (pressing Enter).
func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s [Y/n] ", prompt)
	scanner := bufio.NewScanner(cmd.InOrStdin())
	if !scanner.Scan() {
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "" || answer == "y" || answer == "yes", nil
}
