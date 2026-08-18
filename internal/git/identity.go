package git

import (
	"os"
	"os/exec"
	"strings"
)

// Identity is the effective Git identity that would be used for the next
// commit made in a given directory, along with where each field came from.
type Identity struct {
	Name       string
	Email      string
	NameScope  Scope
	EmailScope Scope
}

// configScopes lists the config levels checked, in Git's own precedence
// order (most specific first). Environment variables are checked separately
// beforehand since they override every config level.
var configScopes = []struct {
	scope Scope
	flag  string
}{
	{ScopeLocal, "--local"},
	{ScopeGlobal, "--global"},
	{ScopeSystem, "--system"},
}

// ResolveIdentity determines the Git identity that would be used for a
// commit made in dir, following Git's own precedence: environment
// variables, then local, global, and system configuration.
func ResolveIdentity(dir string) (*Identity, error) {
	return resolveIdentity(dir, defaultRunner)
}

func resolveIdentity(dir string, r runner) (*Identity, error) {
	name, nameScope := resolveField(dir, r, "user.name", "GIT_AUTHOR_NAME")
	email, emailScope := resolveField(dir, r, "user.email", "GIT_AUTHOR_EMAIL")

	return &Identity{
		Name:       name,
		Email:      email,
		NameScope:  nameScope,
		EmailScope: emailScope,
	}, nil
}

func resolveField(dir string, r runner, configKey, envKey string) (string, Scope) {
	if v, ok := envValue(envKey); ok {
		return v, ScopeEnvironment
	}

	for _, s := range configScopes {
		v, found := configGet(dir, r, s.flag, configKey)
		if found {
			return v, s.scope
		}
	}

	return "", ScopeUnknown
}

// envValue checks the author env var first, falling back to the matching
// committer var, mirroring which one Git itself would honor for the
// author/committer fields respectively.
func envValue(envKey string) (string, bool) {
	if v := os.Getenv(envKey); v != "" {
		return v, true
	}
	committerKey := strings.Replace(envKey, "AUTHOR", "COMMITTER", 1)
	if committerKey != envKey {
		if v := os.Getenv(committerKey); v != "" {
			return v, true
		}
	}
	return "", false
}

func configGet(dir string, r runner, scopeFlag, key string) (string, bool) {
	out, err := r.run(dir, nil, "config", scopeFlag, "--get", key)
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(out)
	if value == "" {
		return "", false
	}
	return value, true
}

// Available reports whether the git binary can be located on PATH, used to
// give a clear error message instead of a raw exec failure.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
