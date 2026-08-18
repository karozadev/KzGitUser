package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/spf13/cobra"
)

func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the Git identity active in the current repository",
		RunE:  runWhoami,
	}
}

func runWhoami(cmd *cobra.Command, _ []string) error {
	out, err := whoamiReport(".")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprint(cmd.OutOrStdout(), out)
	return nil
}

const separator = "────────────────────────────"

func whoamiReport(dir string) (string, error) {
	var b strings.Builder

	if !kzgit.Available() {
		return "", errors.New("git is not installed or not found in PATH")
	}

	identity, err := kzgit.ResolveIdentity(dir)
	if err != nil {
		return "", err
	}

	writeSection(&b, "Git Identity")
	writeField(&b, "Name", valueOrPlaceholder(identity.Name, "Not set"))
	writeField(&b, "Email", valueOrPlaceholder(identity.Email, "Not set"))
	writeField(&b, "Source", string(identitySource(identity)))
	b.WriteString("\n")

	repo, err := kzgit.OpenRepository(dir)
	if err != nil {
		writeSection(&b, "Repository")
		b.WriteString("Not inside a Git repository.\n")
		return b.String(), nil
	}

	writeSection(&b, "Repository")
	writeField(&b, "Path", repo.Path)

	branch, err := repo.Branch()
	if err != nil || branch == "" {
		writeField(&b, "Branch", "Not available (detached HEAD or no commits yet)")
	} else {
		writeField(&b, "Branch", branch)
	}
	b.WriteString("\n")

	writeSection(&b, "Last Commit")
	commit, err := repo.LastCommit()
	if err != nil {
		b.WriteString("No commits yet.\n")
		return b.String(), nil
	}
	writeField(&b, "Author", commit.Author)
	writeField(&b, "Email", commit.Email)

	return b.String(), nil
}

// identitySource picks a single scope to display for the pair, preferring
// the email's scope since that is the field kzgit cares most about, but
// falling back to the name's scope if the email is unset.
func identitySource(identity *kzgit.Identity) kzgit.Scope {
	if identity.Email != "" {
		return identity.EmailScope
	}
	if identity.Name != "" {
		return identity.NameScope
	}
	return kzgit.ScopeUnknown
}

func valueOrPlaceholder(value, placeholder string) string {
	if value == "" {
		return placeholder
	}
	return value
}

func writeSection(w io.Writer, title string) {
	_, _ = fmt.Fprintf(w, "%s\n%s\n", title, separator)
}

func writeField(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "%-7s: %s\n", label, value)
}
