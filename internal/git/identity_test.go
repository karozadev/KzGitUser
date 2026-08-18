package git

import (
	"testing"
)

func TestResolveIdentity_Unset(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "" || id.Email != "" {
		t.Fatalf("expected empty identity, got %+v", id)
	}
	if id.NameScope != ScopeUnknown || id.EmailScope != ScopeUnknown {
		t.Fatalf("expected ScopeUnknown, got name=%s email=%s", id.NameScope, id.EmailScope)
	}
}

func TestResolveIdentity_Global(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)
	runGit(t, dir, "config", "--global", "user.name", "Global User")
	runGit(t, dir, "config", "--global", "user.email", "global@example.com")

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "Global User" || id.Email != "global@example.com" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.NameScope != ScopeGlobal || id.EmailScope != ScopeGlobal {
		t.Fatalf("expected ScopeGlobal, got name=%s email=%s", id.NameScope, id.EmailScope)
	}
}

func TestResolveIdentity_LocalOverridesGlobal(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)
	runGit(t, dir, "config", "--global", "user.name", "Global User")
	runGit(t, dir, "config", "--global", "user.email", "global@example.com")
	runGit(t, dir, "config", "--local", "user.name", "Local User")
	runGit(t, dir, "config", "--local", "user.email", "local@company.com")

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "Local User" || id.Email != "local@company.com" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.NameScope != ScopeLocal || id.EmailScope != ScopeLocal {
		t.Fatalf("expected ScopeLocal, got name=%s email=%s", id.NameScope, id.EmailScope)
	}
}

func TestResolveIdentity_EnvironmentOverridesAll(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)
	runGit(t, dir, "config", "--local", "user.name", "Local User")
	runGit(t, dir, "config", "--local", "user.email", "local@company.com")

	t.Setenv("GIT_AUTHOR_NAME", "Env User")
	t.Setenv("GIT_AUTHOR_EMAIL", "env@example.com")

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "Env User" || id.Email != "env@example.com" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.NameScope != ScopeEnvironment || id.EmailScope != ScopeEnvironment {
		t.Fatalf("expected ScopeEnvironment, got name=%s email=%s", id.NameScope, id.EmailScope)
	}
}

func TestResolveIdentity_CommitterEnvFallback(t *testing.T) {
	isolatedEnv(t)
	dir := initRepo(t)

	t.Setenv("GIT_COMMITTER_NAME", "Committer User")
	t.Setenv("GIT_COMMITTER_EMAIL", "committer@example.com")

	id, err := ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.Name != "Committer User" || id.Email != "committer@example.com" {
		t.Fatalf("unexpected identity: %+v", id)
	}
	if id.NameScope != ScopeEnvironment {
		t.Fatalf("expected ScopeEnvironment, got %s", id.NameScope)
	}
}

func TestAvailable(t *testing.T) {
	if !Available() {
		t.Fatal("expected git to be available on PATH for tests")
	}
}
