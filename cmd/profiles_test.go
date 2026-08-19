package cmd

import (
	"os"
	"path/filepath"
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

func TestProfilesAdd_SaveError(t *testing.T) {
	isolatedEnv(t)
	xdg := os.Getenv("XDG_CONFIG_HOME")
	// Block the directory Save() needs to create with a plain file.
	if err := os.MkdirAll(xdg, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(xdg, "kzgit"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}

	_, err := execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john@company.com")
	if err == nil {
		t.Fatal("expected an error when the config directory can't be created")
	}
}

func TestProfilesRemove_SaveError(t *testing.T) {
	isolatedEnv(t)
	if _, err := execCmd(t, "profiles", "add", "work", "--name", "John", "--email", "john@company.com"); err != nil {
		t.Fatalf("profiles add: %v", err)
	}

	xdg := os.Getenv("XDG_CONFIG_HOME")
	configPath := filepath.Join(xdg, "kzgit", "config.json")
	// Replace the config file with a directory so the post-removal Save()
	// fails when it tries to write to that path.
	if err := os.Remove(configPath); err != nil {
		t.Fatalf("remove config file: %v", err)
	}
	if err := os.Mkdir(configPath, 0o755); err != nil {
		t.Fatalf("seed directory at config path: %v", err)
	}

	_, err := execCmd(t, "profiles", "remove", "work")
	if err == nil {
		t.Fatal("expected an error when the config path can't be written")
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
