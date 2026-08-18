package cmd

import (
	"strings"
	"testing"

	kzgit "github.com/karoza/kz-git-user/internal/git"
)

func TestSwitch_Success(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)

	_, err := execCmd(t, "profiles", "add", "work", "--name", "John Doe", "--email", "john@company.com")
	if err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	out, err := execCmd(t, "switch", "work")
	if err != nil {
		t.Fatalf("switch: %v", err)
	}
	if !strings.Contains(out, "work") || !strings.Contains(out, "john@company.com") {
		t.Fatalf("unexpected switch output: %s", out)
	}

	identity, err := kzgit.ResolveIdentity(dir)
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if identity.Name != "John Doe" || identity.Email != "john@company.com" {
		t.Fatalf("identity not applied: %+v", identity)
	}
	if identity.NameScope != kzgit.ScopeLocal {
		t.Fatalf("expected local scope, got %s", identity.NameScope)
	}
}

func TestSwitch_UnknownProfile(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	_, err := execCmd(t, "switch", "missing")
	if err == nil {
		t.Fatal("expected an error switching to an unknown profile")
	}
}

func TestSwitch_NotAGitRepository(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	t.Chdir(dir)

	_, err := execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john@company.com")
	if err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	_, err = execCmd(t, "switch", "work")
	if err == nil {
		t.Fatal("expected an error switching outside of a git repository")
	}
}
