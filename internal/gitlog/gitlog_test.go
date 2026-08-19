package gitlog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, nil, "init", "-q")
	runGit(t, dir, nil, "config", "user.name", "Test User")
	runGit(t, dir, nil, "config", "user.email", "test@example.com")
	return dir
}

func runGit(t *testing.T, dir string, extraEnv []string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if extraEnv != nil {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

// commitAt creates an (empty, allow-empty) commit authored by authorEmail
// at the given time, in a fixed UTC offset so date comparisons in tests
// don't depend on the host's local timezone.
func commitAt(t *testing.T, dir string, when time.Time, authorEmail string) {
	t.Helper()
	iso := when.UTC().Format("2006-01-02T15:04:05+00:00")
	runGit(t, dir, []string{
		"GIT_AUTHOR_NAME=Test User",
		"GIT_AUTHOR_EMAIL=" + authorEmail,
		"GIT_AUTHOR_DATE=" + iso,
		"GIT_COMMITTER_NAME=Test User",
		"GIT_COMMITTER_EMAIL=" + authorEmail,
		"GIT_COMMITTER_DATE=" + iso,
		"TZ=UTC",
	}, "commit", "-q", "--allow-empty", "-m", "commit at "+iso)
}

func TestScan_FindsCommitsWithinWindow(t *testing.T) {
	dir := initRepo(t)
	now := time.Now().UTC()
	// Commit oldest-first, like a real repository's history: git's
	// --since traversal assumes commit dates increase as HEAD advances,
	// and stops early once it hits an old commit, so an out-of-order
	// synthetic history would make it miss later, in-window commits.
	commitAt(t, dir, now.AddDate(0, 0, -40), "work@company.com") // outside a 30-day window
	commitAt(t, dir, now.AddDate(0, 0, -2), "work@company.com")

	entries := Scan([]string{dir}, now.AddDate(0, 0, -30), 4)
	if len(entries) != 1 {
		t.Fatalf("expected 1 commit within the window, got %d: %+v", len(entries), entries)
	}
	if entries[0].Email != "work@company.com" {
		t.Fatalf("unexpected email: %s", entries[0].Email)
	}
}

func TestScan_CapturesMultipleAuthors(t *testing.T) {
	dir := initRepo(t)
	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -1), "work@company.com")
	commitAt(t, dir, now.AddDate(0, 0, -1), "personal@gmail.com")

	entries := Scan([]string{dir}, now.AddDate(0, 0, -30), 4)
	if len(entries) != 2 {
		t.Fatalf("expected 2 commits, got %d: %+v", len(entries), entries)
	}
}

func TestScan_MultipleRepos(t *testing.T) {
	dirA := initRepo(t)
	dirB := initRepo(t)
	now := time.Now().UTC()
	commitAt(t, dirA, now.AddDate(0, 0, -1), "work@company.com")
	commitAt(t, dirB, now.AddDate(0, 0, -1), "work@company.com")
	commitAt(t, dirB, now.AddDate(0, 0, -1), "work@company.com")

	entries := Scan([]string{dirA, dirB}, now.AddDate(0, 0, -30), 4)
	if len(entries) != 3 {
		t.Fatalf("expected 3 commits across both repos, got %d", len(entries))
	}
}

func TestScan_SkipsInvalidRepoSilently(t *testing.T) {
	notARepo := t.TempDir()
	dir := initRepo(t)
	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -1), "work@company.com")

	entries := Scan([]string{notARepo, dir}, now.AddDate(0, 0, -30), 4)
	if len(entries) != 1 {
		t.Fatalf("expected the invalid repo to be silently skipped, got %d entries", len(entries))
	}
}

func TestScan_EmptyRepoList(t *testing.T) {
	entries := Scan(nil, time.Now(), 4)
	if entries != nil {
		t.Fatalf("expected nil entries for an empty repo list, got %v", entries)
	}
}

func TestScan_DefaultsWorkerCount(t *testing.T) {
	dir := initRepo(t)
	now := time.Now().UTC()
	commitAt(t, dir, now.AddDate(0, 0, -1), "work@company.com")

	entries := Scan([]string{dir}, now.AddDate(0, 0, -30), 0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 commit with maxWorkers=0, got %d", len(entries))
	}
}

func TestCountsByEmail(t *testing.T) {
	entries := []Entry{
		{Repo: "a", Email: "Work@Company.com", Date: "2024-01-01"},
		{Repo: "a", Email: "work@company.com", Date: "2024-01-01"},
		{Repo: "a", Email: "work@company.com", Date: "2024-01-02"},
		{Repo: "b", Email: "personal@gmail.com", Date: "2024-01-01"},
	}

	got := CountsByEmail(entries)

	if got["work@company.com"]["2024-01-01"] != 2 {
		t.Fatalf("expected email matching to be case-insensitive and counts to accumulate, got %+v", got)
	}
	if got["work@company.com"]["2024-01-02"] != 1 {
		t.Fatalf("unexpected count for 2024-01-02: %+v", got)
	}
	if got["personal@gmail.com"]["2024-01-01"] != 1 {
		t.Fatalf("unexpected personal count: %+v", got)
	}
}

func TestLogRepo_InvalidPath(t *testing.T) {
	_, err := logRepo(filepath.Join(t.TempDir(), "does-not-exist"), time.Now())
	if err == nil {
		t.Fatal("expected an error for a nonexistent path")
	}
}

func ExampleCountsByEmail() {
	entries := []Entry{{Email: "a@b.com", Date: "2024-01-01"}}
	counts := CountsByEmail(entries)
	fmt.Println(counts["a@b.com"]["2024-01-01"])
	// Output: 1
}
