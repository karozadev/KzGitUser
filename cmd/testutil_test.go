package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
	run(t, dir, "init", "-q")
	t.Chdir(dir)
	return dir
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// execCmd runs the kzgit command tree with the given args and returns
// combined stdout and the resulting error.
func execCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}
