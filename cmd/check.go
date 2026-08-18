package cmd

import (
	"errors"
	"fmt"
	"strings"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/spf13/cobra"
)

// errCheckFailed signals that `kzgit check` ran successfully but the
// identity it inspected did not pass validation. Execute() treats it
// specially: the check report has already been printed, so no extra
// "Error:" line is added, but the process still exits non-zero.
var errCheckFailed = errors.New("identity check failed")

func newCheckCmd() *cobra.Command {
	var domains []string

	cmd := &cobra.Command{
		Use:   "check",
		Short: "Verify that the current Git identity is valid",
		Long: `Verify that the current Git identity is valid: that user.email is set
and, when allowed domains are configured (via --domain or the "rules" in
the KzGitUser config), that the email belongs to one of them.

Exits with status 0 when the identity is valid, and 1 otherwise, making it
suitable for use in a Git hook or CI pipeline.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCheck(cmd, domains)
		},
	}

	cmd.Flags().StringSliceVar(&domains, "domain", nil, "allowed email domain(s) for this check (repeatable, overrides configured rules)")
	return cmd
}

func runCheck(cmd *cobra.Command, checkDomains []string) error {
	out := cmd.OutOrStdout()

	if !kzgit.Available() {
		_, _ = fmt.Fprintln(out, "✗ git is not installed or not found in PATH")
		return errCheckFailed
	}

	identity, err := kzgit.ResolveIdentity(".")
	if err != nil {
		return err
	}

	ok := true

	if identity.Email == "" {
		_, _ = fmt.Fprintln(out, "✗ user.email is not set")
		ok = false
	} else {
		_, _ = fmt.Fprintf(out, "✓ user.email is set (%s)\n", identity.Email)
	}

	domains := checkDomains
	if len(domains) == 0 {
		cfg, err := kzconfig.Load()
		if err == nil {
			domains = cfg.Rules.AllowedDomains
		}
	}

	if len(domains) > 0 {
		if identity.Email != "" && emailMatchesAnyDomain(identity.Email, domains) {
			_, _ = fmt.Fprintf(out, "✓ email domain is allowed (%s)\n", strings.Join(domains, ", "))
		} else if identity.Email != "" {
			_, _ = fmt.Fprintf(out, "✗ email domain is not allowed (expected one of: %s)\n", strings.Join(domains, ", "))
			ok = false
		}
	}

	if !ok {
		_, _ = fmt.Fprintln(out, "\nIdentity check failed.")
		return errCheckFailed
	}

	_, _ = fmt.Fprintln(out, "\nIdentity check passed.")
	return nil
}

func emailMatchesAnyDomain(email string, domains []string) bool {
	at := strings.LastIndex(email, "@")
	if at == -1 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range domains {
		if strings.ToLower(strings.TrimSpace(d)) == domain {
			return true
		}
	}
	return false
}
