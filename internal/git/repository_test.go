package git

import (
	"errors"
	"testing"
)

func TestOpenRepository_NotARepo(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()

	_, err := OpenRepository(dir)
	if !errors.Is(err, ErrNotAGitRepository) {
		t.Fatalf("expected ErrNotAGitRepository, got %v", err)
	}
}

func TestOpenRepository_Success(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)

	repo, err := OpenRepository(dir)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	if repo.Path == "" {
		t.Fatal("expected non-empty repository path")
	}
}

func TestBranch(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)
	runGit(t, dir, "checkout", "-q", "-b", "main")
	commitFile(t, dir, "file.txt", "Author", "author@example.com")

	repo, err := OpenRepository(dir)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	branch, err := repo.Branch()
	if err != nil {
		t.Fatalf("Branch: %v", err)
	}
	if branch != "main" {
		t.Fatalf("expected branch 'main', got %q", branch)
	}
}

func TestLastCommit_NoCommits(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)

	repo, err := OpenRepository(dir)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	_, err = repo.LastCommit()
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLastCommit_Success(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)
	commitFile(t, dir, "file.txt", "Jane Doe", "jane@example.com")

	repo, err := OpenRepository(dir)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	commit, err := repo.LastCommit()
	if err != nil {
		t.Fatalf("LastCommit: %v", err)
	}
	if commit.Author != "Jane Doe" || commit.Email != "jane@example.com" {
		t.Fatalf("unexpected commit: %+v", commit)
	}
}

func TestSetLocalIdentity(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)

	repo, err := OpenRepository(dir)
	if err != nil {
		t.Fatalf("OpenRepository: %v", err)
	}
	if err := repo.SetLocalIdentity("Local User", "local@company.com"); err != nil {
		t.Fatalf("SetLocalIdentity: %v", err)
	}

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "Local User" || id.Email != "local@company.com" {
		t.Fatalf("unexpected identity after SetLocalIdentity: %+v", id)
	}
	if id.NameScope != ScopeLocal {
		t.Fatalf("expected ScopeLocal, got %s", id.NameScope)
	}
}
