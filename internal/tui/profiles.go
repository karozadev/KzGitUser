package tui

import (
	"fmt"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type profilesView struct {
	flex      *tview.Flex
	app       *App
	list      *tview.List
	profiles  []kzconfig.NamedProfile
	statusBar *tview.TextView
}

func newProfilesView(app *App) *profilesView {
	p := &profilesView{
		app: app,
	}
	p.flex = p.build()
	return p
}

func (p *profilesView) build() *tview.Flex {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBackgroundColor(colorBackground)

	header := newHeaderView("Profiles", "Manage your Git identity profiles")
	root.AddItem(header, 2, 0, false)
	root.AddItem(newSeparator(), 1, 0, false)

	p.list = tview.NewList()
	p.list.SetBackgroundColor(colorBackground)
	p.list.SetHighlightFullLine(true)
	p.list.ShowSecondaryText(true)
	p.list.SetSelectedBackgroundColor(colorPrimary)
	p.list.SetMainTextStyle(tcell.StyleDefault.Foreground(colorForeground).Bold(true))
	p.list.SetSecondaryTextStyle(tcell.StyleDefault.Foreground(colorMuted))

	root.AddItem(p.list, 0, 1, true)

	p.statusBar = tview.NewTextView()
	p.statusBar.SetDynamicColors(true)
	p.statusBar.SetBackgroundColor(colorBackground)
	p.statusBar.SetText("[#6C7086]Enter: Switch  a: Add  d: Delete  r: Refresh  Esc: Back")
	root.AddItem(p.statusBar, 1, 0, false)

	return root
}

func (p *profilesView) refresh() {
	p.list.Clear()
	p.profiles = nil

	cfg, err := kzconfig.Load()
	if err != nil {
		p.statusBar.SetText(fmt.Sprintf("[#E06C75]Error loading config: %s", err.Error()))
		return
	}

	profiles := cfg.List()
	if len(profiles) == 0 {
		p.list.AddItem("No profiles found", "Press 'a' to add a new profile", 'a', nil)
		p.statusBar.SetText("[#6C7086]No profiles saved. Press 'a' to add one.")
		return
	}

	p.profiles = profiles
	for i, profile := range profiles {
		idx := i
		p.list.AddItem(
			profile.Name,
			profile.Email,
			'r',
			func() {
				p.switchProfile(p.profiles[idx].Name)
			},
		)
	}
	p.statusBar.SetText("[#6C7086]Enter: Switch  a: Add  d: Delete  r: Refresh  Esc: Back")
}

func (p *profilesView) switchProfile(name string) {
	if !kzgit.Available() {
		p.showResult("✗ Git is not installed or not found in PATH", false)
		return
	}

	repo, err := kzgit.OpenRepository(".")
	if err != nil {
		p.showResult("✗ Not a git repository", false)
		return
	}

	cfg, err := kzconfig.Load()
	if err != nil {
		p.showResult(fmt.Sprintf("✗ Error loading config: %s", err.Error()), false)
		return
	}

	profile, err := cfg.Get(name)
	if err != nil {
		p.showResult(fmt.Sprintf("✗ Profile %q does not exist", name), false)
		return
	}

	if err := repo.SetLocalIdentity(profile.Name, profile.Email); err != nil {
		p.showResult(fmt.Sprintf("✗ Error applying profile: %s", err.Error()), false)
		return
	}

	p.showResult(fmt.Sprintf("✓ Switched to profile %q (%s <%s>)", name, profile.Name, profile.Email), true)
}

func (p *profilesView) deleteProfile(name string) {
	cfg, err := kzconfig.Load()
	if err != nil {
		p.showResult(fmt.Sprintf("✗ Error loading config: %s", err.Error()), false)
		return
	}

	if err := cfg.Remove(name); err != nil {
		p.showResult(fmt.Sprintf("✗ Error deleting profile: %s", err.Error()), false)
		return
	}

	if err := cfg.Save(); err != nil {
		p.showResult(fmt.Sprintf("✗ Error saving config: %s", err.Error()), false)
		return
	}

	p.showResult(fmt.Sprintf("✓ Profile %q deleted", name), true)
	p.refresh()
}

func (p *profilesView) showResult(msg string, success bool) {
	if success {
		p.statusBar.SetText(fmt.Sprintf("[#7EC87E]%s", msg))
	} else {
		p.statusBar.SetText(fmt.Sprintf("[#E06C75]%s", msg))
	}
}

func (p *profilesView) getSelectedProfile() string {
	idx := p.list.GetCurrentItem()
	if idx >= 0 && idx < len(p.profiles) {
		return p.profiles[idx].Name
	}
	return ""
}
