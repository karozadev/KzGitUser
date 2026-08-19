package cmd

import (
	"strings"
	"testing"
)

func TestWhoamiReport_NoRepository(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()

	out, err := whoamiReport(dir)
	if err != nil {
		t.Fatalf("whoamiReport: %v", err)
	}
	if !strings.Contains(out, "Not inside a Git repository") {
		t.Fatalf("expected 'Not inside a Git repository' message, got:\n%s", out)
	}
}

func TestWhoamiReport_NoIdentitySet(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	run(t, dir, "init", "-q")

	out, err := whoamiReport(dir)
	if err != nil {
		t.Fatalf("whoamiReport: %v", err)
	}
	if !strings.Contains(out, "Name   : Not set") || !strings.Contains(out, "Email  : Not set") {
		t.Fatalf("expected Name/Email to be 'Not set', got:\n%s", out)
	}
	if !strings.Contains(out, "No commits yet") {
		t.Fatalf("expected 'No commits yet', got:\n%s", out)
	}
}

func TestWhoamiReport_FullIdentity(t *testing.T) {
	isolatedEnv(t)
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	run(t, dir, "config", "--local", "user.name", "Jane Doe")
	run(t, dir, "config", "--local", "user.email", "jane@company.com")

	out, err := whoamiReport(dir)
	if err != nil {
		t.Fatalf("whoamiReport: %v", err)
	}
	for _, want := range []string{
		"Name   : Jane Doe",
		"Email  : jane@company.com",
		"Source : Local Repository",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestWhoamiCommand_RootAlias(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.name", "Jane Doe")
	run(t, dir, "config", "--local", "user.email", "jane@company.com")

	out, err := execCmd(t, "whoami")
	if err != nil {
		t.Fatalf("execCmd: %v", err)
	}
	if !strings.Contains(out, "jane@company.com") {
		t.Fatalf("expected whoami output, got:\n%s", out)
	}
}

func TestWhoamiCommand_Explicit(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	run(t, dir, "config", "--local", "user.name", "Jane Doe")
	run(t, dir, "config", "--local", "user.email", "jane@company.com")

	out, err := execCmd(t, "whoami")
	if err != nil {
		t.Fatalf("execCmd: %v", err)
	}
	if !strings.Contains(out, "jane@company.com") {
		t.Fatalf("expected whoami output, got:\n%s", out)
	}
}
