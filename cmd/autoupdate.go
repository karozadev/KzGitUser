package cmd

import (
	"fmt"
	"os"

	kzupdate "github.com/karoza/kz-git-user/internal/update"
	"github.com/spf13/cobra"
)

// offerUpdateIfAvailable is the "shell startup"-style check: on an
// interactive terminal, it looks for a newer kzgit release (using a cached,
// rate-limited check so it doesn't hit the network on every invocation) and,
// if one exists, offers to install it right away. It never blocks or fails
// non-interactive usage: any error, or the absence of a TTY, is a silent
// no-op so hooks, CI and scripted use of kzgit are unaffected.
func offerUpdateIfAvailable(cmd *cobra.Command) {
	if os.Getenv("KZGIT_NO_UPDATE_CHECK") != "" {
		return
	}
	if version == "dev" {
		return
	}
	if !isInteractive() {
		return
	}

	info, err := kzupdate.Check(version)
	if err != nil || !info.Available {
		return
	}

	out := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(out, "\nA new version of kzgit is available: v%s (you have v%s).\n", info.Latest, info.Current)

	confirmed, err := confirm(cmd, "Update now?")
	if err != nil {
		return
	}
	if !confirmed {
		_, _ = fmt.Fprintln(out, "Run 'kzgit update' anytime to upgrade.")
		return
	}

	_, _ = fmt.Fprintf(out, "Downloading kzgit v%s...\n", info.Latest)
	if err := kzupdate.Install(info.Latest); err != nil {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Update failed: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(out, "Updated to kzgit v%s. This takes effect the next time you run kzgit.\n", info.Latest)
}

// isInteractive reports whether both stdin and stdout are attached to a
// terminal, so the update prompt only ever appears for a human at a
// keyboard and never in a pipe, hook, or CI job.
func isInteractive() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil || stdinInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil || stdoutInfo.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	return true
}
