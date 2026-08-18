package git

import (
	"os"
	"strings"
)

// Repository represents a Git repository rooted at Path.
type Repository struct {
	Path string
	r    runner
}

// OpenRepository locates the Git repository containing dir (or the current
// working directory if dir is empty) and returns a handle to it. It returns
// ErrNotAGitRepository if dir is not inside a Git work tree.
func OpenRepository(dir string) (*Repository, error) {
	return openRepository(dir, defaultRunner)
}

func openRepository(dir string, r runner) (*Repository, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dir = wd
	}

	out, err := r.run(dir, nil, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, ErrNotAGitRepository
	}

	return &Repository{Path: strings.TrimSpace(out), r: r}, nil
}

// Branch returns the current branch name. It returns an empty string (with
// no error) when the repository is in a detached HEAD state.
func (repo *Repository) Branch() (string, error) {
	out, err := repo.r.run(repo.Path, nil, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(out)
	if branch == "HEAD" {
		return "", nil
	}
	return branch, nil
}

// Commit describes a single commit's author metadata.
type Commit struct {
	Author string
	Email  string
}

// LastCommit returns the author name and email of the HEAD commit. It
// returns ErrNotFound if the repository has no commits yet.
func (repo *Repository) LastCommit() (*Commit, error) {
	out, err := repo.r.run(repo.Path, nil, "log", "-1", "--pretty=format:%an%n%ae")
	if err != nil {
		return nil, ErrNotFound
	}
	lines := strings.SplitN(out, "\n", 2)
	if len(lines) < 2 {
		return nil, ErrNotFound
	}
	return &Commit{Author: lines[0], Email: lines[1]}, nil
}

// SetLocalIdentity sets user.name and user.email in the repository's local
// config (.git/config).
func (repo *Repository) SetLocalIdentity(name, email string) error {
	if _, err := repo.r.run(repo.Path, nil, "config", "--local", "user.name", name); err != nil {
		return err
	}
	if _, err := repo.r.run(repo.Path, nil, "config", "--local", "user.email", email); err != nil {
		return err
	}
	return nil
}
