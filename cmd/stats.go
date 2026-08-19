package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/karoza/kz-git-user/internal/gitlog"
	"github.com/karoza/kz-git-user/internal/heatmap"
	"github.com/karoza/kz-git-user/internal/scan"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	var (
		profileName string
		days        int
		path        string
		compare     bool
	)

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show a commit heatmap for a Git identity profile",
		Long: `Show a GitHub-style commit contribution heatmap in the terminal, scoped to
a Git identity profile (or the currently active identity), built from
every Git repository found under --path.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runStats(cmd, statsOptions{
				profileName: profileName,
				days:        days,
				path:        path,
				compare:     compare,
			})
		},
	}

	cmd.Flags().StringVar(&profileName, "profile", "", "profile to show stats for (default: the currently active identity)")
	cmd.Flags().IntVar(&days, "days", 30, "number of days to analyze (1-365)")
	cmd.Flags().StringVar(&path, "path", ".", "root directory to scan recursively for Git repositories")
	cmd.Flags().BoolVar(&compare, "compare", false, "show all saved profiles side by side")
	_ = cmd.RegisterFlagCompletionFunc("profile", completeProfileNames)

	return cmd
}

type statsOptions struct {
	profileName string
	days        int
	path        string
	compare     bool
}

// statsProfile is the minimal identity information needed to render a
// heatmap: a display name and the email to bucket commits by.
type statsProfile struct {
	Name  string
	Email string
}

func runStats(cmd *cobra.Command, opts statsOptions) error {
	if opts.days < 1 || opts.days > 365 {
		return fmt.Errorf("--days must be between 1 and 365 (got %d)", opts.days)
	}
	if opts.compare && opts.profileName != "" {
		return fmt.Errorf("--profile and --compare cannot be used together")
	}
	if !kzgit.Available() {
		return fmt.Errorf("git is not installed or not found in PATH")
	}

	profiles, err := resolveStatsProfiles(opts)
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(opts.path)
	if err != nil {
		return fmt.Errorf("resolving --path %q: %w", opts.path, err)
	}

	out := cmd.OutOrStdout()
	repos := scan.Repos(absPath, scan.DefaultMaxWorkers)
	if len(repos) == 0 {
		_, _ = fmt.Fprintf(out, "No Git repositories found under %s\n", absPath)
		return nil
	}

	since := time.Now().AddDate(0, 0, -(opts.days - 1))
	entries := gitlog.Scan(repos, since, gitlog.DefaultMaxWorkers)
	byEmail := gitlog.CountsByEmail(entries)
	noColor := !stdoutIsTerminal() || os.Getenv("NO_COLOR") != ""

	if opts.compare {
		blocks := make([][]string, 0, len(profiles))
		for _, p := range profiles {
			blocks = append(blocks, heatmap.Render(heatmap.Options{
				ProfileName: p.Name,
				Email:       p.Email,
				Days:        opts.days,
				Counts:      byEmail[strings.ToLower(p.Email)],
				NoColor:     noColor,
			}))
		}
		for _, line := range heatmap.SideBySide(blocks, "   ") {
			_, _ = fmt.Fprintln(out, line)
		}
		return nil
	}

	p := profiles[0]
	for _, line := range heatmap.Render(heatmap.Options{
		ProfileName: p.Name,
		Email:       p.Email,
		Days:        opts.days,
		Counts:      byEmail[strings.ToLower(p.Email)],
		NoColor:     noColor,
	}) {
		_, _ = fmt.Fprintln(out, line)
	}
	return nil
}

// resolveStatsProfiles determines which profile(s) to render: every saved
// profile for --compare, the named profile for --profile, or otherwise the
// currently active Git identity.
func resolveStatsProfiles(opts statsOptions) ([]statsProfile, error) {
	if opts.compare {
		cfg, err := kzconfig.Load()
		if err != nil {
			return nil, err
		}
		list := cfg.List()
		if len(list) == 0 {
			return nil, fmt.Errorf("no profiles configured; add one with 'kzgit profiles add'")
		}
		profiles := make([]statsProfile, 0, len(list))
		for _, p := range list {
			profiles = append(profiles, statsProfile{Name: p.Name, Email: p.Email})
		}
		return profiles, nil
	}

	if opts.profileName != "" {
		cfg, err := kzconfig.Load()
		if err != nil {
			return nil, err
		}
		p, err := cfg.Get(opts.profileName)
		if err != nil {
			return nil, fmt.Errorf("profile %q does not exist. Run 'kzgit profiles list' to see available profiles", opts.profileName)
		}
		return []statsProfile{{Name: p.Name, Email: p.Email}}, nil
	}

	identity, err := kzgit.ResolveIdentity(".")
	if err != nil {
		return nil, err
	}
	if identity.Email == "" {
		return nil, fmt.Errorf("no active Git identity found; set one with 'kzgit switch <profile>', configure user.email, or pass --profile")
	}
	name := identity.Name
	if name == "" {
		name = identity.Email
	}
	return []statsProfile{{Name: name, Email: identity.Email}}, nil
}

// stdoutIsTerminal reports whether stdout is attached to a terminal, used
// to decide whether ANSI color codes make sense (never when piped to a
// file or another program).
func stdoutIsTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
