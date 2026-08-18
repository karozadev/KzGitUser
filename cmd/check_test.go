package cmd

import (
	"strings"
	"testing"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
)

func TestCheck_PassesWithEmailSet(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.email", "john@company.com")
	run(t, dir, "config", "--local", "user.name", "John Doe")

	out, err := execCmd(t, "check")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !strings.Contains(out, "Identity check passed") {
		t.Fatalf("expected pass message, got:\n%s", out)
	}
}

func TestCheck_FailsWithoutEmail(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	out, err := execCmd(t, "check")
	if err == nil {
		t.Fatal("expected check to fail when user.email is unset")
	}
	if !strings.Contains(out, "user.email is not set") {
		t.Fatalf("expected explanatory message, got:\n%s", out)
	}
}

func TestCheck_DomainFlag_Allowed(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.email", "john@company.com")

	_, err := execCmd(t, "check", "--domain", "company.com")
	if err != nil {
		t.Fatalf("expected check to pass for an allowed domain: %v", err)
	}
}

func TestCheck_DomainFlag_Rejected(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.email", "john@gmail.com")

	out, err := execCmd(t, "check", "--domain", "company.com")
	if err == nil {
		t.Fatal("expected check to fail for a disallowed domain")
	}
	if !strings.Contains(out, "email domain is not allowed") {
		t.Fatalf("expected domain rejection message, got:\n%s", out)
	}
}

func TestCheck_NoRulesConfigured_PassesOnEmailAlone(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.email", "john@gmail.com")

	if _, err := execCmd(t, "check"); err != nil {
		t.Fatalf("expected check to pass without configured rules: %v", err)
	}
}

func TestCheck_UsesConfiguredRules(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.email", "john@gmail.com")

	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Rules.AllowedDomains = []string{"company.com"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, err := execCmd(t, "check")
	if err == nil {
		t.Fatal("expected check to fail based on configured rules")
	}
	if !strings.Contains(out, "email domain is not allowed") {
		t.Fatalf("expected domain rejection message, got:\n%s", out)
	}
}
