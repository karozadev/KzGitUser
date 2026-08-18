package cmd

import (
	"strings"
	"testing"
)

func TestProfilesAddListRemove(t *testing.T) {
	isolatedEnv(t)

	out, err := execCmd(t, "profiles", "add", "work", "--name", "John Doe", "--email", "john@company.com")
	if err != nil {
		t.Fatalf("profiles add: %v", err)
	}
	if !strings.Contains(out, `"work" added`) {
		t.Fatalf("unexpected add output: %s", out)
	}

	out, err = execCmd(t, "profiles", "list")
	if err != nil {
		t.Fatalf("profiles list: %v", err)
	}
	if !strings.Contains(out, "work") || !strings.Contains(out, "john@company.com") {
		t.Fatalf("expected profile in list output, got:\n%s", out)
	}

	out, err = execCmd(t, "profiles", "remove", "work")
	if err != nil {
		t.Fatalf("profiles remove: %v", err)
	}
	if !strings.Contains(out, `"work" removed`) {
		t.Fatalf("unexpected remove output: %s", out)
	}

	out, err = execCmd(t, "profiles", "list")
	if err != nil {
		t.Fatalf("profiles list: %v", err)
	}
	if !strings.Contains(out, "No profiles saved yet") {
		t.Fatalf("expected empty profiles message, got:\n%s", out)
	}
}

func TestProfilesAdd_MissingRequiredFlags(t *testing.T) {
	isolatedEnv(t)

	_, err := execCmd(t, "profiles", "add", "work")
	if err == nil {
		t.Fatal("expected an error when --name/--email are missing")
	}
}

func TestProfilesAdd_InvalidEmail(t *testing.T) {
	isolatedEnv(t)

	_, err := execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "not-an-email")
	if err == nil {
		t.Fatal("expected an error for invalid email")
	}
}

func TestProfilesAdd_Duplicate(t *testing.T) {
	isolatedEnv(t)

	_, err := execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john@company.com")
	if err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err = execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john2@company.com")
	if err == nil {
		t.Fatal("expected an error when adding a duplicate profile")
	}
}

func TestProfilesRemove_NotFound(t *testing.T) {
	isolatedEnv(t)

	_, err := execCmd(t, "profiles", "remove", "missing")
	if err == nil {
		t.Fatal("expected an error when removing a non-existent profile")
	}
}

func TestProfilesCommand_DefaultsToList(t *testing.T) {
	isolatedEnv(t)
	_, _ = execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john@company.com")

	out, err := execCmd(t, "profiles")
	if err != nil {
		t.Fatalf("profiles: %v", err)
	}
	if !strings.Contains(out, "work") {
		t.Fatalf("expected bare 'profiles' to list profiles, got:\n%s", out)
	}
}
