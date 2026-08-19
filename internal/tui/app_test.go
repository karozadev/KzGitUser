package tui

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	kzconfig "github.com/karoza/kz-git-user/internal/config"
)

func runeEvent(r rune) *tcell.EventKey {
	return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone)
}

func keyEvent(key tcell.Key) *tcell.EventKey {
	return tcell.NewEventKey(key, 0, tcell.ModNone)
}

func TestShowPage(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	app.showPage("profiles")
	if app.currentPage != "profiles" {
		t.Fatalf("expected currentPage=profiles, got %s", app.currentPage)
	}

	app.showPage("commands")
	if app.currentPage != "commands" {
		t.Fatalf("expected currentPage=commands, got %s", app.currentPage)
	}

	app.showPage("dashboard")
	if app.currentPage != "dashboard" {
		t.Fatalf("expected currentPage=dashboard, got %s", app.currentPage)
	}
}

func TestCyclePages(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	order := []string{"profiles", "commands", "dashboard"}
	for _, want := range order {
		app.cyclePages()
		if app.currentPage != want {
			t.Fatalf("cyclePages: currentPage = %s, want %s", app.currentPage, want)
		}
	}
}

func TestUpdateFooter_PerPage(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	app.currentPage = "dashboard"
	app.updateFooter()
	if !strings.Contains(app.footer.GetText(true), "Profiles") {
		t.Fatalf("expected dashboard hints, got: %s", app.footer.GetText(true))
	}

	app.currentPage = "profiles"
	app.updateFooter()
	if !strings.Contains(app.footer.GetText(true), "Navigate") {
		t.Fatalf("expected profiles hints, got: %s", app.footer.GetText(true))
	}

	app.currentPage = "commands"
	app.updateFooter()
	if !strings.Contains(app.footer.GetText(true), "Execute") {
		t.Fatalf("expected commands hints, got: %s", app.footer.GetText(true))
	}
}

func TestHandleInput_Escape(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.showPage("profiles")

	if got := app.handleInput(keyEvent(tcell.KeyEscape)); got != nil {
		t.Fatal("expected Escape to be consumed (nil returned)")
	}
	if app.currentPage != "dashboard" {
		t.Fatalf("expected Escape to return to dashboard, got %s", app.currentPage)
	}
}

func TestHandleInput_Tab(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	if got := app.handleInput(keyEvent(tcell.KeyTab)); got != nil {
		t.Fatal("expected Tab to be consumed (nil returned)")
	}
	if app.currentPage != "profiles" {
		t.Fatalf("expected Tab to advance to profiles, got %s", app.currentPage)
	}
}

func TestHandleInput_QuitOnlyOutsideCommands(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.currentPage = "commands"

	// On the commands page, 'q' should be passed through (not consumed),
	// so the app isn't quit while the user is typing a command.
	if got := app.handleInput(runeEvent('q')); got == nil {
		t.Fatal("expected 'q' to be passed through on the commands page")
	}

	app.currentPage = "dashboard"
	if got := app.handleInput(runeEvent('q')); got != nil {
		t.Fatal("expected 'q' to be consumed (and quit) outside the commands page")
	}
}

func TestHandleInput_HelpAndSlash(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	app.handleInput(runeEvent('?'))
	if app.currentPage != "commands" {
		t.Fatalf("expected '?' to switch to commands page, got %s", app.currentPage)
	}
	if !strings.Contains(app.commandsView.output.GetText(true), "Available Commands") {
		t.Fatal("expected '?' to trigger the help command")
	}

	app2 := newTestApp()
	app2.handleInput(runeEvent('/'))
	if app2.currentPage != "commands" {
		t.Fatalf("expected '/' to switch to commands page, got %s", app2.currentPage)
	}
	if app2.commandsView.input.GetText() != "/" {
		t.Fatalf("expected '/' to seed the command input, got %q", app2.commandsView.input.GetText())
	}
}

func TestHandleInput_ProfilesPageShortcuts(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("a", "A", "a@example.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Add("b", "B", "b@example.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.showPage("profiles")
	app.profilesView.list.SetCurrentItem(0)

	app.handleInput(runeEvent('j'))
	if app.profilesView.list.GetCurrentItem() != 1 {
		t.Fatalf("expected 'j' to move down, got index %d", app.profilesView.list.GetCurrentItem())
	}

	app.handleInput(runeEvent('k'))
	if app.profilesView.list.GetCurrentItem() != 0 {
		t.Fatalf("expected 'k' to move up, got index %d", app.profilesView.list.GetCurrentItem())
	}

	// 'k' at the top should clamp at 0, not go negative.
	app.handleInput(runeEvent('k'))
	if app.profilesView.list.GetCurrentItem() != 0 {
		t.Fatalf("expected 'k' at top to clamp at 0, got index %d", app.profilesView.list.GetCurrentItem())
	}
}

func TestHandleInput_NumberKeys(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	app.handleInput(runeEvent('2'))
	if app.currentPage != "profiles" {
		t.Fatalf("expected '2' to switch to profiles, got %s", app.currentPage)
	}
	app.handleInput(runeEvent('3'))
	if app.currentPage != "commands" {
		t.Fatalf("expected '3' to switch to commands, got %s", app.currentPage)
	}
	app.handleInput(runeEvent('1'))
	if app.currentPage != "dashboard" {
		t.Fatalf("expected '1' to switch to dashboard, got %s", app.currentPage)
	}
}

func TestHandleInput_PKeyOnlyFromDashboard(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()

	app.handleInput(runeEvent('p'))
	if app.currentPage != "profiles" {
		t.Fatalf("expected 'p' from dashboard to open profiles, got %s", app.currentPage)
	}
}

func TestHandleInput_AddDeleteRefreshOnlyOnProfilesPage(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.showPage("profiles")

	app.handleInput(runeEvent('a'))
	if !app.pages.HasPage("dialog") {
		t.Fatal("expected 'a' on profiles page to open the add-profile dialog")
	}
	app.pages.RemovePage("dialog")

	app.handleInput(runeEvent('r'))
	// Should not panic; refresh() was called.

	app.handleInput(runeEvent('d'))
	// No profile selected, so showDeleteProfileDialog shows a result
	// message (via App.showResult, which targets the footer) instead of
	// opening the confirmation dialog.
	if !strings.Contains(app.footer.GetText(true), "No profile selected") {
		t.Fatalf("expected 'no profile selected' after 'd' with nothing to delete, got: %s", app.footer.GetText(true))
	}
}

func TestShowAddProfileDialog(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.showAddProfileDialog()
	if !app.pages.HasPage("dialog") {
		t.Fatal("expected showAddProfileDialog to add a 'dialog' page")
	}
}

func TestShowDeleteProfileDialog_NoSelection(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.profilesView.refresh()

	app.showDeleteProfileDialog()
	if app.pages.HasPage("dialog") {
		t.Fatal("expected no dialog to be shown when nothing is selected")
	}
	if !strings.Contains(app.footer.GetText(true), "No profile selected") {
		t.Fatal("expected 'no profile selected' message in the footer")
	}
}

func TestShowDeleteProfileDialog_WithSelection(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("work", "W", "w@example.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.profilesView.refresh()

	app.showDeleteProfileDialog()
	if !app.pages.HasPage("dialog") {
		t.Fatal("expected a delete confirmation dialog to be shown")
	}
}

func TestShowResult(t *testing.T) {
	app := newTestApp()

	app.showResult("all good", true)
	if !strings.Contains(app.footer.GetText(true), "all good") {
		t.Fatalf("expected success message in footer, got: %s", app.footer.GetText(true))
	}

	app.showResult("oops", false)
	if !strings.Contains(app.footer.GetText(true), "oops") {
		t.Fatalf("expected failure message in footer, got: %s", app.footer.GetText(true))
	}
}

func TestStop(t *testing.T) {
	app := newTestApp()
	// Application.Stop() is a documented no-op when the screen was never
	// started (a.screen == nil), so this only verifies no panic occurs.
	app.stop()
}

func TestJoinHints_Empty(t *testing.T) {
	if got := joinHints(nil); got != "" {
		t.Fatalf("joinHints(nil) = %q, want empty string", got)
	}
}
