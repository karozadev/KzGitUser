package tui

import (
	"strings"
	"testing"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
)

func TestExecuteInput_Empty(t *testing.T) {
	app := newTestApp()
	c := app.commandsView
	c.input.SetText("")
	c.executeInput()
	if len(c.history) != 0 {
		t.Fatalf("expected no history entry for empty input, got %v", c.history)
	}
}

func TestExecuteInput_UnknownCommand(t *testing.T) {
	app := newTestApp()
	c := app.commandsView
	c.input.SetText("/bogus")
	c.executeInput()
	if !strings.Contains(c.output.GetText(true), "Unknown command: bogus") {
		t.Fatalf("expected unknown command message, got:\n%s", c.output.GetText(true))
	}
	if c.input.GetText() != "" {
		t.Fatalf("expected input to be cleared after execution")
	}
}

func TestExecuteInput_KnownCommand(t *testing.T) {
	app := newTestApp()
	c := app.commandsView
	c.input.SetText("/help")
	c.executeInput()
	if !strings.Contains(c.output.GetText(true), "Available Commands") {
		t.Fatalf("expected help output, got:\n%s", c.output.GetText(true))
	}
	if len(c.history) != 1 || c.history[0] != "/help" {
		t.Fatalf("expected /help to be recorded in history, got %v", c.history)
	}
}

func TestExecuteInput_WithArgs(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	app := newTestApp()
	c := app.commandsView
	c.input.SetText("/switch missing-profile")
	c.executeInput()
	if !strings.Contains(c.output.GetText(true), `"missing-profile" does not exist`) {
		t.Fatalf("expected missing profile message, got:\n%s", c.output.GetText(true))
	}
}

func TestShowSuggestions(t *testing.T) {
	app := newTestApp()
	c := app.commandsView

	c.input.SetText("/who")
	c.showSuggestions("/who")
	if c.input.GetText() != "/whoami" {
		t.Fatalf("expected autocomplete to /whoami, got %q", c.input.GetText())
	}

	c.input.SetText("/s")
	c.showSuggestions("/s")
	// "switch" and "status" both match the "s" prefix, so no unique
	// suggestion should be applied.
	if c.input.GetText() != "/s" {
		t.Fatalf("expected no autocomplete for an ambiguous prefix, got %q", c.input.GetText())
	}

	c.input.SetText("plain text")
	c.showSuggestions("plain text")
	if c.input.GetText() != "plain text" {
		t.Fatalf("expected non-slash text to be left alone, got %q", c.input.GetText())
	}
}

func TestCmdHelp(t *testing.T) {
	app := newTestApp()
	app.commandsView.cmdHelp("")
	if !strings.Contains(app.commandsView.output.GetText(true), "Available Commands") {
		t.Fatal("expected help text in output")
	}
}

func TestCmdProfiles_Empty(t *testing.T) {
	isolatedEnv(t)
	app := newTestApp()
	app.commandsView.cmdProfiles("")
	if !strings.Contains(app.commandsView.output.GetText(true), "No profiles saved yet") {
		t.Fatalf("expected empty-profiles message, got:\n%s", app.commandsView.output.GetText(true))
	}
}

func TestCmdProfiles_WithProfiles(t *testing.T) {
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
	app.commandsView.cmdProfiles("")
	out := app.commandsView.output.GetText(true)
	if !strings.Contains(out, "john@company.com") {
		t.Fatalf("expected profile in output, got:\n%s", out)
	}
}

func TestCmdWhoami(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	runGit(t, dir, "config", "--local", "user.name", "Jane Doe")
	runGit(t, dir, "config", "--local", "user.email", "jane@company.com")

	app := newTestApp()
	app.commandsView.cmdWhoami("")
	out := app.commandsView.output.GetText(true)
	if !strings.Contains(out, "jane@company.com") {
		t.Fatalf("expected identity in output, got:\n%s", out)
	}
}

func TestCmdSwitch_NoArgs(t *testing.T) {
	app := newTestApp()
	app.commandsView.cmdSwitch("")
	if !strings.Contains(app.commandsView.output.GetText(true), "Usage: /switch") {
		t.Fatal("expected usage message for empty args")
	}
}

func TestCmdSwitch_NotARepo(t *testing.T) {
	isolatedEnv(t)
	t.Chdir(t.TempDir())

	app := newTestApp()
	app.commandsView.cmdSwitch("work")
	if !strings.Contains(app.commandsView.output.GetText(true), "Not a git repository") {
		t.Fatal("expected not-a-repo message")
	}
}

func TestCmdSwitch_Success(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

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
	app.commandsView.cmdSwitch("work")
	out := app.commandsView.output.GetText(true)
	if !strings.Contains(out, "Switched to profile") || !strings.Contains(out, "john@company.com") {
		t.Fatalf("expected success message, got:\n%s", out)
	}
}

func TestCmdCheck_NoEmail(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	app := newTestApp()
	app.commandsView.cmdCheck("")
	if !strings.Contains(app.commandsView.output.GetText(true), "user.email is not set") {
		t.Fatal("expected missing-email message")
	}
}

func TestCmdCheck_DomainRules(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	runGit(t, dir, "config", "--local", "user.email", "john@gmail.com")

	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Rules.AllowedDomains = []string{"company.com"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	app := newTestApp()
	app.commandsView.cmdCheck("")
	out := app.commandsView.output.GetText(true)
	if !strings.Contains(out, "email domain is not allowed") {
		t.Fatalf("expected domain rejection, got:\n%s", out)
	}
}

func TestCmdStatus_NotARepo(t *testing.T) {
	isolatedEnv(t)
	t.Chdir(t.TempDir())

	app := newTestApp()
	app.commandsView.cmdStatus("")
	if !strings.Contains(app.commandsView.output.GetText(true), "Not a git repository") {
		t.Fatal("expected not-a-repo message")
	}
}

func TestCmdStatus_NoCommits(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	app := newTestApp()
	app.commandsView.cmdStatus("")
	if !strings.Contains(app.commandsView.output.GetText(true), "No commits yet") {
		t.Fatal("expected no-commits message")
	}
}

func TestCmdQuit(t *testing.T) {
	app := newTestApp()
	// Application.Stop() is a safe no-op when the screen was never
	// started, so this just verifies no panic occurs.
	app.commandsView.cmdQuit("")
}
