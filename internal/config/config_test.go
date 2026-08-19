package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	cases := map[string]bool{
		"john@company.com":   true,
		"john.doe@sub.co.uk": true,
		"":                   false,
		"not-an-email":       false,
		"john@":              false,
		"@company.com":       false,
		"john company.com":   false,
		"john@company":       false,
	}
	for email, want := range cases {
		if got := ValidateEmail(email); got != want {
			t.Errorf("ValidateEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func newTestConfig(t *testing.T) *Config {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	return cfg
}

func TestAddAndGet(t *testing.T) {
	cfg := newTestConfig(t)

	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	p, err := cfg.Get("work")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Name != "John Doe" || p.Email != "john@company.com" {
		t.Fatalf("unexpected profile: %+v", p)
	}
}

func TestAdd_Duplicate(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	err := cfg.Add("work", "Jane Doe", "jane@company.com")
	if !errors.Is(err, ErrProfileExists) {
		t.Fatalf("expected ErrProfileExists, got %v", err)
	}
}

func TestAdd_InvalidEmail(t *testing.T) {
	cfg := newTestConfig(t)
	err := cfg.Add("work", "John Doe", "not-an-email")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestAdd_EmptyName(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Add("", "John Doe", "john@company.com"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName for empty profile name, got %v", err)
	}
	if err := cfg.Add("work", "", "john@company.com"); !errors.Is(err, ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName for empty display name, got %v", err)
	}
}

func TestGet_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	_, err := cfg.Get("missing")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	cfg := newTestConfig(t)
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Update("work", "John D.", "john.d@company.com"); err != nil {
		t.Fatalf("Update: %v", err)
	}
	p, _ := cfg.Get("work")
	if p.Name != "John D." || p.Email != "john.d@company.com" {
		t.Fatalf("unexpected profile after update: %+v", p)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	err := cfg.Update("missing", "X", "x@example.com")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestUpdate_InvalidEmail(t *testing.T) {
	cfg := newTestConfig(t)
	_ = cfg.Add("work", "John Doe", "john@company.com")
	err := cfg.Update("work", "", "not-an-email")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	cfg := newTestConfig(t)
	_ = cfg.Add("work", "John Doe", "john@company.com")

	if err := cfg.Remove("work"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := cfg.Get("work"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected profile to be gone, got err=%v", err)
	}
}

func TestRemove_NotFound(t *testing.T) {
	cfg := newTestConfig(t)
	err := cfg.Remove("missing")
	if !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("expected ErrProfileNotFound, got %v", err)
	}
}

func TestList_SortedAlphabetically(t *testing.T) {
	cfg := newTestConfig(t)
	_ = cfg.Add("personal", "John Doe", "john@gmail.com")
	_ = cfg.Add("work-corp", "John Doe", "john@company.com")
	_ = cfg.Add("freelance", "John Doe", "john@freelance.com")

	profiles := cfg.List()
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}
	want := []string{"freelance", "personal", "work-corp"}
	for i, name := range want {
		if profiles[i].Name != name {
			t.Errorf("profiles[%d].Name = %q, want %q", i, profiles[i].Name, name)
		}
	}
}

func TestList_Empty(t *testing.T) {
	cfg := newTestConfig(t)
	if profiles := cfg.List(); len(profiles) != 0 {
		t.Fatalf("expected no profiles, got %d", len(profiles))
	}
}

func TestPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if cfg.Path() != path {
		t.Fatalf("Path() = %q, want %q", cfg.Path(), path)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "config.json")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom (reload): %v", err)
	}
	p, err := reloaded.Get("work")
	if err != nil {
		t.Fatalf("Get after reload: %v", err)
	}
	if p.Name != "John Doe" || p.Email != "john@company.com" {
		t.Fatalf("unexpected profile after reload: %+v", p)
	}
}

func TestLoadFrom_MissingFileReturnsEmptyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.json")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if len(cfg.List()) != 0 {
		t.Fatalf("expected empty config, got %d profiles", len(cfg.List()))
	}
}

func TestLoadFrom_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := writeFile(path, "{not valid json"); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected an error for invalid JSON")
	}
}

func TestLoad_FallsBackToLegacyPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	legacy := filepath.Join(home, ".kzgit.json")
	if err := writeFile(legacy, `{"profiles":{"legacy":{"name":"Legacy User","email":"legacy@example.com"}}}`); err != nil {
		t.Fatalf("writeFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	p, err := cfg.Get("legacy")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Email != "legacy@example.com" {
		t.Fatalf("unexpected email: %s", p.Email)
	}
}

func TestDefaultPath_NoHomeDir(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("XDG_CONFIG_HOME")

	if _, err := DefaultPath(); err == nil {
		t.Fatal("expected an error when no home directory can be determined")
	}
}

func TestFallbackPath_NoHomeDir(t *testing.T) {
	t.Setenv("HOME", "")
	_ = os.Unsetenv("HOME")

	if _, err := FallbackPath(); err == nil {
		t.Fatal("expected an error when no home directory can be determined")
	}
}

func TestSave_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where Save() needs to create a directory, so
	// MkdirAll fails with "not a directory".
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}

	cfg := &Config{Profiles: map[string]Profile{}, path: filepath.Join(blocker, "sub", "config.json")}
	if err := cfg.Save(); err == nil {
		t.Fatal("expected Save to fail when its directory path is blocked by a file")
	}
}

func TestSave_WriteFileError(t *testing.T) {
	dir := t.TempDir()
	// Make the target config path itself a directory, so the final
	// os.WriteFile fails.
	target := filepath.Join(dir, "config.json")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("seed directory at config path: %v", err)
	}

	cfg := &Config{Profiles: map[string]Profile{}, path: target}
	if err := cfg.Save(); err == nil {
		t.Fatal("expected Save to fail when the config path is a directory")
	}
}

func TestLoad_PrefersDefaultOverLegacy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	xdg := filepath.Join(home, ".config")
	t.Setenv("XDG_CONFIG_HOME", xdg)

	legacy := filepath.Join(home, ".kzgit.json")
	if err := writeFile(legacy, `{"profiles":{"legacy":{"name":"Legacy","email":"legacy@example.com"}}}`); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	primary := filepath.Join(xdg, "kzgit", "config.json")
	if err := writeFile(primary, `{"profiles":{"primary":{"name":"Primary","email":"primary@example.com"}}}`); err != nil {
		t.Fatalf("writeFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := cfg.Get("primary"); err != nil {
		t.Fatalf("expected primary profile to be loaded: %v", err)
	}
	if _, err := cfg.Get("legacy"); err == nil {
		t.Fatal("did not expect legacy profile to be loaded when primary config exists")
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}
