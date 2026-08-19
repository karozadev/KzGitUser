package scan

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
}

func markAsRepo(t *testing.T, dir string) {
	t.Helper()
	mkdirAll(t, filepath.Join(dir, ".git"))
}

func TestRepos_FindsTopLevelRepo(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "project")
	markAsRepo(t, repoDir)

	got := Repos(root, 4)
	if len(got) != 1 || got[0] != repoDir {
		t.Fatalf("Repos() = %v, want [%s]", got, repoDir)
	}
}

func TestRepos_FindsNestedRepos(t *testing.T) {
	root := t.TempDir()
	repoA := filepath.Join(root, "org", "repo-a")
	repoB := filepath.Join(root, "org", "sub", "repo-b")
	markAsRepo(t, repoA)
	markAsRepo(t, repoB)

	got := Repos(root, 4)
	if len(got) != 2 {
		t.Fatalf("expected 2 repos, got %v", got)
	}
	if got[0] != repoA || got[1] != repoB {
		t.Fatalf("Repos() = %v, want [%s %s]", got, repoA, repoB)
	}
}

func TestRepos_DoesNotDescendIntoFoundRepo(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "project")
	markAsRepo(t, repoDir)
	// A nested ".git"-looking directory inside the repo's working tree
	// should never be reported as a second, separate repository.
	nested := filepath.Join(repoDir, "vendor", "some-lib")
	markAsRepo(t, nested)

	got := Repos(root, 4)
	if len(got) != 1 || got[0] != repoDir {
		t.Fatalf("expected only the top-level repo to be found, got %v", got)
	}
}

func TestRepos_SkipsHiddenDirectories(t *testing.T) {
	root := t.TempDir()
	hidden := filepath.Join(root, ".cache", "project")
	markAsRepo(t, hidden)

	got := Repos(root, 4)
	if len(got) != 0 {
		t.Fatalf("expected hidden directories to be skipped, got %v", got)
	}
}

func TestRepos_NoRepositoriesFound(t *testing.T) {
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "not-a-repo", "subdir"))

	got := Repos(root, 4)
	if len(got) != 0 {
		t.Fatalf("expected no repos, got %v", got)
	}
}

func TestRepos_UnreadableRootIsIgnoredSilently(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")

	got := Repos(root, 4)
	if len(got) != 0 {
		t.Fatalf("expected an empty slice for an unreadable root, got %v", got)
	}
}

func TestRepos_DefaultsWorkerCount(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "project")
	markAsRepo(t, repoDir)

	// maxWorkers <= 0 should fall back to DefaultMaxWorkers rather than
	// deadlock or panic.
	got := Repos(root, 0)
	if len(got) != 1 || got[0] != repoDir {
		t.Fatalf("Repos() with maxWorkers=0 = %v, want [%s]", got, repoDir)
	}
}
