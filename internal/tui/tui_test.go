package tui

import (
	"testing"

	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/rivo/tview"
)

func TestNewApp(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp returned nil")
	}
	if app.tviewApp == nil {
		t.Fatal("tviewApp is nil")
	}
	if app.pages == nil {
		t.Fatal("pages is nil")
	}
	if app.dashboard == nil {
		t.Fatal("dashboard is nil")
	}
	if app.profilesView == nil {
		t.Fatal("profilesView is nil")
	}
	if app.commandsView == nil {
		t.Fatal("commandsView is nil")
	}
	if app.footer == nil {
		t.Fatal("footer is nil")
	}
}

func TestNewHeaderView(t *testing.T) {
	tv := newHeaderView("Test Title", "Test Subtitle")
	if tv == nil {
		t.Fatal("newHeaderView returned nil")
	}
}

func TestNewSeparator(t *testing.T) {
	tv := newSeparator()
	if tv == nil {
		t.Fatal("newSeparator returned nil")
	}
}

func TestNewInfoRow(t *testing.T) {
	tv := newInfoRow("Label", "Value")
	if tv == nil {
		t.Fatal("newInfoRow returned nil")
	}
}

func TestNewFooterBar(t *testing.T) {
	tv := newFooterBar()
	if tv == nil {
		t.Fatal("newFooterBar returned nil")
	}
}

func TestJoinHints(t *testing.T) {
	hints := []string{
		newKeyBinding("Tab", "Next"),
		newKeyBinding("/", "Command"),
	}
	result := joinHints(hints)
	if result == "" {
		t.Fatal("joinHints returned empty string")
	}
}

func TestEmailMatchesAnyDomain(t *testing.T) {
	tests := []struct {
		email   string
		domains []string
		want    bool
	}{
		{"user@company.com", []string{"company.com"}, true},
		{"user@other.com", []string{"company.com"}, false},
		{"user@COMPANY.COM", []string{"company.com"}, true},
		{"invalid-email", []string{"company.com"}, false},
		{"user@", []string{"company.com"}, false},
		{"user@company.com", []string{}, false},
	}

	for _, tt := range tests {
		got := emailMatchesAnyDomain(tt.email, tt.domains)
		if got != tt.want {
			t.Errorf("emailMatchesAnyDomain(%q, %v) = %v, want %v", tt.email, tt.domains, got, tt.want)
		}
	}
}

func TestIdentitySource(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{"John", "john@company.com", "Environment Variable"},
		{"John", "", "Environment Variable"},
		{"", "john@company.com", "Environment Variable"},
		{"", "", "Not set"},
	}

	for _, tt := range tests {
		identity := &kzgit.Identity{
			Name:       tt.name,
			Email:      tt.email,
			NameScope:  "Environment Variable",
			EmailScope: "Environment Variable",
		}
		got := identitySource(identity)
		if got != tt.expected {
			t.Errorf("identitySource(%+v) = %q, want %q", tt, got, tt.expected)
		}
	}
}

func TestDashboardBuild(t *testing.T) {
	app := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}
	d := newDashboard(app)
	if d == nil {
		t.Fatal("newDashboard returned nil")
	}
	if d.flex == nil {
		t.Fatal("dashboard flex is nil")
	}
	if len(d.fields) == 0 {
		t.Fatal("dashboard has no fields")
	}
}

func TestProfilesViewBuild(t *testing.T) {
	app := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}
	p := newProfilesView(app)
	if p == nil {
		t.Fatal("newProfilesView returned nil")
	}
	if p.flex == nil {
		t.Fatal("profilesView flex is nil")
	}
	if p.list == nil {
		t.Fatal("profilesView list is nil")
	}
}

func TestCommandInputBuild(t *testing.T) {
	app := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}
	c := newCommandInput(app)
	if c == nil {
		t.Fatal("newCommandInput returned nil")
	}
	if c.flex == nil {
		t.Fatal("commandInput flex is nil")
	}
	if c.input == nil {
		t.Fatal("commandInput input is nil")
	}
	if c.output == nil {
		t.Fatal("commandInput output is nil")
	}
	if len(c.commands) == 0 {
		t.Fatal("commandInput has no commands")
	}
}

func TestCommandInputCommands(t *testing.T) {
	app := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}
	c := newCommandInput(app)

	expectedCommands := []string{"help", "profiles", "whoami", "switch", "check", "status", "quit", "exit"}
	for _, cmd := range expectedCommands {
		if _, ok := c.commands[cmd]; !ok {
			t.Errorf("command %q not registered", cmd)
		}
	}
}
