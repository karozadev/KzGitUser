package tui

import (
	"strings"
	"testing"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
)

func TestProfilesRefresh_Empty(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.profilesView.refresh()

	if app.profilesView.list.GetItemCount() != 1 {
		t.Fatalf("expected a single placeholder item, got %d", app.profilesView.list.GetItemCount())
	}
	if !strings.Contains(app.profilesView.statusBar.GetText(true), "No profiles saved") {
		t.Fatal("expected empty-profiles status message")
	}
}

func TestProfilesRefresh_WithProfiles(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("personal", "John Doe", "john@gmail.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.profilesView.refresh()

	if app.profilesView.list.GetItemCount() != 2 {
		t.Fatalf("expected 2 profile items, got %d", app.profilesView.list.GetItemCount())
	}
	if len(app.profilesView.profiles) != 2 {
		t.Fatalf("expected 2 tracked profiles, got %d", len(app.profilesView.profiles))
	}
}

func TestGetSelectedProfile(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.profilesView.refresh()

	if got := app.profilesView.getSelectedProfile(); got != "work" {
		t.Fatalf("getSelectedProfile() = %q, want %q", got, "work")
	}
}

func TestGetSelectedProfile_Empty(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.profilesView.refresh()

	if got := app.profilesView.getSelectedProfile(); got != "" {
		t.Fatalf("expected empty selection with no profiles, got %q", got)
	}
}

func TestSwitchProfile_NotARepo(t *testing.T) {
	isolatedEnv(t)
	t.Chdir(t.TempDir())

	app := newTestApp()
	app.profilesView.switchProfile("work")
	if !strings.Contains(app.profilesView.statusBar.GetText(true), "Not a git repository") {
		t.Fatal("expected not-a-repo status message")
	}
}

func TestSwitchProfile_UnknownProfile(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	app := newTestApp()
	app.profilesView.switchProfile("missing")
	if !strings.Contains(app.profilesView.statusBar.GetText(true), `"missing" does not exist`) {
		t.Fatalf("expected unknown-profile message, got: %s", app.profilesView.statusBar.GetText(true))
	}
}

func TestSwitchProfile_Success(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)

	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.profilesView.switchProfile("work")
	if !strings.Contains(app.profilesView.statusBar.GetText(true), "Switched to profile") {
		t.Fatalf("expected success message, got: %s", app.profilesView.statusBar.GetText(true))
	}

	_ = dir
}

func TestDeleteProfile_Success(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.profilesView.refresh()
	app.profilesView.deleteProfile("work")

	// deleteProfile shows a confirmation via showResult, but immediately
	// calls refresh() afterwards, which overwrites the status bar with
	// the (now-empty) list's own message — so the confirmation text
	// itself isn't observable here. What we can verify is the actual
	// effect: the profile is gone from both the list and the saved config.
	if app.profilesView.list.GetItemCount() != 1 {
		t.Fatalf("expected refresh() to leave the placeholder item after delete, got %d items", app.profilesView.list.GetItemCount())
	}
	reloaded, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := reloaded.Get("work"); err == nil {
		t.Fatal("expected the 'work' profile to be removed from the saved config")
	}
}

func TestDeleteProfile_NotFound(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.profilesView.deleteProfile("missing")
	if !strings.Contains(app.profilesView.statusBar.GetText(true), "Error deleting profile") {
		t.Fatalf("expected error message, got: %s", app.profilesView.statusBar.GetText(true))
	}
}
