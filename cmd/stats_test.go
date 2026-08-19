package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// commitAt creates an empty commit authored by authorEmail at the given
// time (in UTC), independent of the ambient environment's timezone.
func commitAt(t *testing.T, dir string, when time.Time, authorEmail string) {
	t.Helper()
	iso := when.UTC().Format("2006-01-02T15:04:05+00:00")
	cmd := exec.Command("git", "commit", "-q", "--allow-empty", "-m", "commit at "+iso)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL="+authorEmail,
		"GIT_AUTHOR_DATE="+iso,
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL="+authorEmail,
		"GIT_COMMITTER_DATE="+iso,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestStats_ActiveIdentity(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.name", "Jane Doe")
	run(t, dir, "config", "--local", "user.email", "jane@company.com")

	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -1), "jane@company.com")

	out, err := execCmd(t, "stats", "--path", dir, "--days", "7")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(out, "jane@company.com") {
		t.Fatalf("expected active identity's email in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Total:   1 commits") {
		t.Fatalf("expected 1 commit counted, got:\n%s", out)
	}
}

func TestStats_ActiveIdentity_NoEmail(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	_, err := execCmd(t, "stats")
	if err == nil {
		t.Fatal("expected an error when there is no active identity")
	}
}

func TestStats_NamedProfile(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -2), "work@company.com")

	if _, err := execCmd(t, "profiles", "add", "work", "--name", "Work", "--email", "work@company.com"); err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	out, err := execCmd(t, "stats", "--profile", "work", "--path", dir, "--days", "7")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(out, "Profile: Work <work@company.com>") {
		t.Fatalf("expected named profile header, got:\n%s", out)
	}
	if !strings.Contains(out, "Total:   1 commits") {
		t.Fatalf("expected 1 commit counted, got:\n%s", out)
	}
}

func TestStats_UnknownProfile(t *testing.T) {
	isolatedEnv(t)
	_, err := execCmd(t, "stats", "--profile", "does-not-exist")
	if err == nil {
		t.Fatal("expected an error for an unknown profile")
	}
}

func TestStats_Compare(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -1), "work@company.com")
	commitAt(t, dir, now.AddDate(0, 0, -1), "personal@gmail.com")
	commitAt(t, dir, now.AddDate(0, 0, -1), "personal@gmail.com")

	if _, err := execCmd(t, "profiles", "add", "work", "--name", "Work", "--email", "work@company.com"); err != nil {
		t.Fatalf("profiles add work: %v", err)
	}
	if _, err := execCmd(t, "profiles", "add", "personal", "--name", "Personal", "--email", "personal@gmail.com"); err != nil {
		t.Fatalf("profiles add personal: %v", err)
	}

	out, err := execCmd(t, "stats", "--compare", "--path", dir, "--days", "7")
	if err != nil {
		t.Fatalf("stats --compare: %v", err)
	}
	if !strings.Contains(out, "work@company.com") || !strings.Contains(out, "personal@gmail.com") {
		t.Fatalf("expected both profiles in comparison output, got:\n%s", out)
	}
}

func TestStats_Compare_NoProfiles(t *testing.T) {
	isolatedEnv(t)
	_, err := execCmd(t, "stats", "--compare")
	if err == nil {
		t.Fatal("expected an error when --compare has no saved profiles")
	}
}

func TestStats_CompareAndProfileMutuallyExclusive(t *testing.T) {
	isolatedEnv(t)
	_, err := execCmd(t, "stats", "--compare", "--profile", "work")
	if err == nil {
		t.Fatal("expected an error when --compare and --profile are combined")
	}
}

func TestStats_DaysOutOfRange(t *testing.T) {
	isolatedEnv(t)
	if _, err := execCmd(t, "stats", "--days", "0"); err == nil {
		t.Fatal("expected an error for --days=0")
	}
	if _, err := execCmd(t, "stats", "--days", "366"); err == nil {
		t.Fatal("expected an error for --days=366")
	}
}

func TestStats_NoRepositoriesFound(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	if _, err := execCmd(t, "profiles", "add", "work", "--name", "Work", "--email", "work@company.com"); err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	out, err := execCmd(t, "stats", "--profile", "work", "--path", dir)
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(out, "No Git repositories found") {
		t.Fatalf("expected a no-repositories message, got:\n%s", out)
	}
}

func TestStats_ScansNestedRepos(t *testing.T) {
	isolatedEnv(t)
	root := t.TempDir()
	repoA := filepath.Join(root, "org", "repo-a")
	repoB := filepath.Join(root, "org", "repo-b")
	if err := os.MkdirAll(repoA, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.MkdirAll(repoB, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	run(t, repoA, "init", "-q")
	run(t, repoB, "init", "-q")

	now := time.Now().UTC()
	commitAt(t, repoA, now.AddDate(0, 0, -1), "work@company.com")
	commitAt(t, repoB, now.AddDate(0, 0, -1), "work@company.com")

	if _, err := execCmd(t, "profiles", "add", "work", "--name", "Work", "--email", "work@company.com"); err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	out, err := execCmd(t, "stats", "--profile", "work", "--path", root, "--days", "7")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if !strings.Contains(out, "Total:   2 commits") {
		t.Fatalf("expected commits from both nested repos to be aggregated, got:\n%s", out)
	}
}

func TestStats_GitNotAvailable(t *testing.T) {
	isolatedEnv(t)
	t.Setenv("PATH", "")

	_, err := execCmd(t, "stats", "--profile", "work")
	if err == nil {
		t.Fatal("expected an error when git is not available")
	}
}

func TestNewStatsCmd_FlagDefaults(t *testing.T) {
	cmd := newStatsCmd()

	if f := cmd.Flag("days"); f == nil || f.DefValue != "30" {
		t.Fatalf("expected --days to default to 30, got %+v", f)
	}
	if f := cmd.Flag("path"); f == nil || f.DefValue != "." {
		t.Fatalf("expected --path to default to \".\", got %+v", f)
	}
	if f := cmd.Flag("compare"); f == nil || f.DefValue != "false" {
		t.Fatalf("expected --compare to default to false, got %+v", f)
	}
}
