// Package git provides read/write access to the Git identity (user.name,
// user.email) and basic repository metadata by shelling out to the system
// git binary. It intentionally avoids reimplementing Git's own config
// resolution logic.
package git

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

// ErrNotAGitRepository is returned when an operation requires a Git
// repository but the current directory is not inside one.
var ErrNotAGitRepository = errors.New("not a git repository")

// ErrNotFound is returned when a requested git config key has no value.
var ErrNotFound = errors.New("git config key not found")

// Scope identifies the level a piece of Git configuration was read from or
// should be written to.
type Scope string

const (
	// ScopeEnvironment means the value came from an environment variable
	// such as GIT_AUTHOR_NAME/GIT_AUTHOR_EMAIL, which overrides any config
	// file for the purpose of the next commit.
	ScopeEnvironment Scope = "Environment Variable"
	// ScopeLocal is the repository-local config (.git/config).
	ScopeLocal Scope = "Local Repository"
	// ScopeGlobal is the per-user config (~/.gitconfig or XDG equivalent).
	ScopeGlobal Scope = "Global"
	// ScopeSystem is the machine-wide config (/etc/gitconfig).
	ScopeSystem Scope = "System"
	// ScopeUnknown means no source could be determined.
	ScopeUnknown Scope = "Not set"
)

// runner executes external commands. It exists so tests can substitute a
// fake implementation without touching the real git binary or filesystem.
type runner interface {
	run(dir string, env []string, args ...string) (stdout string, err error)
}

type execRunner struct{}

func (execRunner) run(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = env
	}
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", &CommandError{Args: args, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
	return out.String(), nil
}

// CommandError wraps a failed invocation of the git binary with its stderr
// output for clearer error messages.
type CommandError struct {
	Args   []string
	Stderr string
	Err    error
}

func (e *CommandError) Error() string {
	if e.Stderr != "" {
		return "git " + strings.Join(e.Args, " ") + ": " + e.Stderr
	}
	return "git " + strings.Join(e.Args, " ") + ": " + e.Err.Error()
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

var defaultRunner runner = execRunner{}
