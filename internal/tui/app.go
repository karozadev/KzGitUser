package tui

import (
	"fmt"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// App is the main TUI application.
type App struct {
	tviewApp     *tview.Application
	pages        *tview.Pages
	dashboard    *dashboard
	profilesView *profilesView
	commandsView *commandInput
	footer       *tview.TextView
	currentPage  string
}

// NewApp creates a new TUI application.
func NewApp() *App {
	a := &App{
		tviewApp: tview.NewApplication(),
		pages:    tview.NewPages(),
	}

	a.dashboard = newDashboard(a)
	a.profilesView = newProfilesView(a)
	a.commandsView = newCommandInput(a)
	a.footer = newFooterBar()

	a.pages.AddPage("dashboard", a.dashboard.flex, true, true)
	a.pages.AddPage("profiles", a.profilesView.flex, true, false)
	a.pages.AddPage("commands", a.commandsView.flex, true, false)

	a.tviewApp.SetRoot(a.buildLayout(), true)
	a.tviewApp.SetInputCapture(a.handleInput)

	a.currentPage = "dashboard"
	return a
}

func (a *App) buildLayout() *tview.Flex {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBackgroundColor(colorBackground)
	root.AddItem(a.pages, 0, 1, true)
	root.AddItem(a.footer, 1, 0, false)
	return root
}

func (a *App) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		a.showPage("dashboard")
		return nil
	case tcell.KeyTab:
		a.cyclePages()
		return nil
	}

	switch event.Rune() {
	case 'q':
		if a.currentPage != "commands" {
			a.stop()
			return nil
		}
	case '?':
		a.showPage("commands")
		a.commandsView.cmdHelp("")
		return nil
	case '/':
		a.showPage("commands")
		a.commandsView.input.SetText("/")
		a.tviewApp.SetFocus(a.commandsView.input)
		return nil
	case 'p':
		if a.currentPage == "dashboard" {
			a.showPage("profiles")
			return nil
		}
	case 'a':
		if a.currentPage == "profiles" {
			a.showAddProfileDialog()
			return nil
		}
	case 'd':
		if a.currentPage == "profiles" {
			a.showDeleteProfileDialog()
			return nil
		}
	case 'r':
		if a.currentPage == "profiles" {
			a.profilesView.refresh()
			return nil
		}
	case 'j':
		if a.currentPage == "profiles" {
			a.profilesView.list.SetCurrentItem(a.profilesView.list.GetCurrentItem() + 1)
			return nil
		}
	case 'k':
		if a.currentPage == "profiles" {
			idx := a.profilesView.list.GetCurrentItem() - 1
			if idx < 0 {
				idx = 0
			}
			a.profilesView.list.SetCurrentItem(idx)
			return nil
		}
	case '1':
		a.showPage("dashboard")
		return nil
	case '2':
		a.showPage("profiles")
		return nil
	case '3':
		a.showPage("commands")
		return nil
	}

	return event
}

func (a *App) showPage(name string) {
	a.currentPage = name
	a.pages.SwitchToPage(name)
	a.updateFooter()

	switch name {
	case "dashboard":
		a.dashboard.refresh()
		a.tviewApp.SetFocus(a.pages)
	case "profiles":
		a.profilesView.refresh()
		a.tviewApp.SetFocus(a.profilesView.list)
	case "commands":
		a.tviewApp.SetFocus(a.commandsView.input)
	}
}

func (a *App) cyclePages() {
	pages := []string{"dashboard", "profiles", "commands"}
	for i, p := range pages {
		if p == a.currentPage {
			next := pages[(i+1)%len(pages)]
			a.showPage(next)
			return
		}
	}
}

func (a *App) updateFooter() {
	var hints []string
	switch a.currentPage {
	case "dashboard":
		hints = []string{
			newKeyBinding("1-3", "Pages"),
			newKeyBinding("p", "Profiles"),
			newKeyBinding("/", "Command"),
			newKeyBinding("q", "Quit"),
		}
	case "profiles":
		hints = []string{
			newKeyBinding("j/k", "Navigate"),
			newKeyBinding("Enter", "Switch"),
			newKeyBinding("a", "Add"),
			newKeyBinding("d", "Delete"),
			newKeyBinding("r", "Refresh"),
			newKeyBinding("Esc", "Back"),
		}
	case "commands":
		hints = []string{
			newKeyBinding("Enter", "Execute"),
			newKeyBinding("Esc", "Back"),
		}
	}
	a.footer.SetText("  " + joinHints(hints))
}

func joinHints(hints []string) string {
	result := ""
	for i, h := range hints {
		if i > 0 {
			result += "  │  "
		}
		result += h
	}
	return result
}

func (a *App) showAddProfileDialog() {
	name := ""
	email := ""

	modal := tview.NewModal().
		SetText("Add new profile").
		AddButtons([]string{"Add", "Cancel"}).
		SetBackgroundColor(colorBackground).
		SetTextColor(colorForeground).
		SetBorderColor(colorPrimary)

	nameInput := tview.NewInputField().
		SetLabel("Name: ").
		SetFieldBackgroundColor(colorHighlight).
		SetFieldTextColor(colorForeground)

	emailInput := tview.NewInputField().
		SetLabel("Email: ").
		SetFieldBackgroundColor(colorHighlight).
		SetFieldTextColor(colorForeground)

	form := tview.NewForm().
		AddFormItem(nameInput).
		AddFormItem(emailInput).
		AddButton("Add", func() {
			name = nameInput.GetText()
			email = emailInput.GetText()
			if name == "" || email == "" {
				a.showResult("✗ Name and email are required", false)
				return
			}

			cfg, err := kzconfig.Load()
			if err != nil {
				a.showResult(fmt.Sprintf("✗ Error: %s", err.Error()), false)
				return
			}

			if err := cfg.Add(name, name, email); err != nil {
				a.showResult(fmt.Sprintf("✗ Error: %s", err.Error()), false)
				return
			}

			if err := cfg.Save(); err != nil {
				a.showResult(fmt.Sprintf("✗ Error: %s", err.Error()), false)
				return
			}

			a.showResult(fmt.Sprintf("✓ Profile %q added", name), true)
			a.profilesView.refresh()
			a.tviewApp.Stop()
		}).
		AddButton("Cancel", func() {
			a.tviewApp.Stop()
		})

	form.SetBackgroundColor(colorBackground)
	form.SetBorderColor(colorPrimary)
	form.SetFieldBackgroundColor(colorHighlight)

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(modal, 3, 0, false).
		AddItem(form, 0, 1, true)

	a.pages.AddPage("dialog", flex, true, true)
	a.tviewApp.SetFocus(form)
}

func (a *App) showDeleteProfileDialog() {
	profile := a.profilesView.getSelectedProfile()
	if profile == "" {
		a.showResult("✗ No profile selected", false)
		return
	}

	modal := tview.NewModal()
	modal.SetText(fmt.Sprintf("Delete profile %q?", profile))
	modal.AddButtons([]string{"Delete", "Cancel"})
	modal.SetBackgroundColor(colorBackground)
	modal.SetTextColor(colorForeground)
	modal.SetBorderColor(colorError)

	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		if buttonLabel == "Delete" {
			a.profilesView.deleteProfile(profile)
		}
		a.pages.RemovePage("dialog")
		a.tviewApp.SetFocus(a.profilesView.list)
	})

	a.pages.AddPage("dialog", modal, true, true)
	a.tviewApp.SetFocus(modal)
}

func (a *App) showResult(msg string, success bool) {
	color := "#E06C75"
	if success {
		color = "#7EC87E"
	}
	a.footer.SetText(fmt.Sprintf("  [%s]%s", color, msg))
}

func (a *App) stop() {
	a.tviewApp.Stop()
}

// Run starts the TUI application.
func (a *App) Run() error {
	a.dashboard.refresh()
	return a.tviewApp.Run()
}
