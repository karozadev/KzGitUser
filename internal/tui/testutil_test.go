package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rivo/tview"
)

// isolatedEnv redirects Git and KzGitUser config to a fresh temp directory
// so tests never read or write the real developer's configuration.
func isolatedEnv(t *testing.T) {
	t.Helper()

	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(dir, "gitconfig-global"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(dir, "gitconfig-system"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	for _, key := range []string{
		"GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL",
		"EMAIL",
	} {
		t.Setenv(key, "")
		_ = os.Unsetenv(key)
	}
}

// initRepoAndChdir creates a fresh Git repository and changes the test's
// working directory into it (restored automatically by t.Chdir).
func initRepoAndChdir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	t.Chdir(dir)
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// newTestApp builds an App without ever starting its event loop, matching
// the pattern already used by TestDashboardBuild etc.
func newTestApp() *App {
	a := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}
	a.dashboard = newDashboard(a)
	a.profilesView = newProfilesView(a)
	a.commandsView = newCommandInput(a)
	a.footer = newFooterBar()
	a.pages.AddPage("dashboard", a.dashboard.flex, true, true)
	a.pages.AddPage("profiles", a.profilesView.flex, true, false)
	a.pages.AddPage("commands", a.commandsView.flex, true, false)
	a.currentPage = "dashboard"
	return a
}
