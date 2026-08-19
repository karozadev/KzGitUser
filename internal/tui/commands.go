package tui

import (
	"fmt"
	"strings"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type commandInput struct {
	flex       *tview.Flex
	app        *App
	input      *tview.InputField
	output     *tview.TextView
	history    []string
	historyIdx int
	commands   map[string]func(args string)
}

func newCommandInput(app *App) *commandInput {
	c := &commandInput{
		app:      app,
		history:  make([]string, 0),
		commands: make(map[string]func(args string)),
	}
	c.flex = c.build()
	c.registerCommands()
	return c
}

func (c *commandInput) build() *tview.Flex {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBackgroundColor(colorBackground)

	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetBackgroundColor(colorBackground)
	fmt.Fprintf(header, "[#6C9EEB::b] Commands ")
	root.AddItem(header, 1, 0, false)
	root.AddItem(newSeparator(), 1, 0, false)

	c.output = tview.NewTextView()
	c.output.SetDynamicColors(true)
	c.output.SetBackgroundColor(colorBackground)
	c.output.SetScrollable(true)
	c.output.SetRegions(true)
	root.AddItem(c.output, 0, 1, false)

	c.input = tview.NewInputField()
	c.input.SetBackgroundColor(colorBackground)
	c.input.SetFieldBackgroundColor(colorHighlight)
	c.input.SetFieldTextColor(colorForeground)
	c.input.SetPlaceholderTextColor(colorMuted)
	c.input.SetLabel("[#6C9EEB]> ")
	c.input.SetPlaceholder("Type a command... (press / to focus)")
	c.input.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEnter {
			c.executeInput()
		}
	})
	c.input.SetChangedFunc(func(text string) {
		c.showSuggestions(text)
	})
	root.AddItem(c.input, 1, 0, true)

	return root
}

func (c *commandInput) registerCommands() {
	c.commands["help"] = c.cmdHelp
	c.commands["profiles"] = c.cmdProfiles
	c.commands["whoami"] = c.cmdWhoami
	c.commands["switch"] = c.cmdSwitch
	c.commands["check"] = c.cmdCheck
	c.commands["status"] = c.cmdStatus
	c.commands["quit"] = c.cmdQuit
	c.commands["exit"] = c.cmdQuit
}

func (c *commandInput) executeInput() {
	text := strings.TrimSpace(c.input.GetText())
	if text == "" {
		return
	}

	c.history = append(c.history, text)
	c.historyIdx = len(c.history)

	if text[0] == '/' {
		text = text[1:]
	}

	parts := strings.SplitN(text, " ", 2)
	cmdName := parts[0]
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	if cmd, ok := c.commands[cmdName]; ok {
		c.output.Write([]byte(fmt.Sprintf("[#6C9EEB]> %s\n", c.input.GetText())))
		cmd(args)
	} else {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]Unknown command: %s\n", cmdName)))
		c.output.Write([]byte("[#6C7086]Type /help to see available commands\n"))
	}

	c.input.SetText("")
	c.output.ScrollToEnd()
}

func (c *commandInput) showSuggestions(text string) {
	if len(text) == 0 || text[0] != '/' {
		return
	}

	query := strings.ToLower(text[1:])
	matches := make([]string, 0)
	for cmd := range c.commands {
		if strings.HasPrefix(cmd, query) {
			matches = append(matches, "/"+cmd)
		}
	}

	if len(matches) == 1 && matches[0] != text {
		c.input.SetText(matches[0])
	}
}

func (c *commandInput) cmdHelp(_ string) {
	help := `
[#6C9EEB::b]Available Commands:
[white]/help       [#6C7086]Show this help message
[white]/profiles   [#6C7086]List and manage Git profiles
[white]/whoami     [#6C7086]Show current Git identity
[white]/switch     [#6C7086]Switch to a profile (usage: /switch <name>)
[white]/check      [#6C7086]Check current Git configuration
[white]/status     [#6C7086]Show repository status
[white]/quit       [#6C7086]Exit KzGitUser
`
	c.output.Write([]byte(help))
}

func (c *commandInput) cmdProfiles(_ string) {
	cfg, err := kzconfig.Load()
	if err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]Error loading config: %s\n", err.Error())))
		return
	}

	profiles := cfg.List()
	if len(profiles) == 0 {
		c.output.Write([]byte("[#6C7086]No profiles saved yet.\n"))
		return
	}

	c.output.Write([]byte("[#6C9EEB::b]Saved Profiles:\n"))
	for _, p := range profiles {
		c.output.Write([]byte(fmt.Sprintf("  [white]%s [#6C7086]- %s\n", p.Name, p.Email)))
	}
}

func (c *commandInput) cmdWhoami(_ string) {
	if !kzgit.Available() {
		c.output.Write([]byte("[#E06C75]✗ Git is not installed or not found in PATH\n"))
		return
	}

	identity, err := kzgit.ResolveIdentity(".")
	if err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]Error: %s\n", err.Error())))
		return
	}

	nameVal := identity.Name
	if nameVal == "" {
		nameVal = "Not set"
	}
	emailVal := identity.Email
	if emailVal == "" {
		emailVal = "Not set"
	}

	c.output.Write([]byte("[#6C9EEB::b]Current Identity:\n"))
	c.output.Write([]byte(fmt.Sprintf("  [white]Name   : %s\n", nameVal)))
	c.output.Write([]byte(fmt.Sprintf("  [white]Email  : %s\n", emailVal)))
	c.output.Write([]byte(fmt.Sprintf("  [white]Source : %s\n", string(identitySource(identity)))))
}

func (c *commandInput) cmdSwitch(args string) {
	name := strings.TrimSpace(args)
	if name == "" {
		c.output.Write([]byte("[#E06C75]Usage: /switch <profile-name>\n"))
		c.output.Write([]byte("[#6C7086]Run /profiles to see available profiles\n"))
		return
	}

	if !kzgit.Available() {
		c.output.Write([]byte("[#E06C75]✗ Git is not installed or not found in PATH\n"))
		return
	}

	repo, err := kzgit.OpenRepository(".")
	if err != nil {
		c.output.Write([]byte("[#E06C75]✗ Not a git repository\n"))
		return
	}

	cfg, err := kzconfig.Load()
	if err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]Error loading config: %s\n", err.Error())))
		return
	}

	profile, err := cfg.Get(name)
	if err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]✗ Profile %q does not exist\n", name)))
		c.output.Write([]byte("[#6C7086]Available profiles:\n"))
		for _, p := range cfg.List() {
			c.output.Write([]byte(fmt.Sprintf("  [white]%s\n", p.Name)))
		}
		return
	}

	if err := repo.SetLocalIdentity(profile.Name, profile.Email); err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]✗ Error applying profile: %s\n", err.Error())))
		return
	}

	c.output.Write([]byte(fmt.Sprintf("[#7EC87E]✓ Switched to profile %q (%s <%s>)\n", name, profile.Name, profile.Email)))
}

func (c *commandInput) cmdCheck(_ string) {
	if !kzgit.Available() {
		c.output.Write([]byte("[#E06C75]✗ Git is not installed or not found in PATH\n"))
		return
	}

	identity, err := kzgit.ResolveIdentity(".")
	if err != nil {
		c.output.Write([]byte(fmt.Sprintf("[#E06C75]Error: %s\n", err.Error())))
		return
	}

	if identity.Email == "" {
		c.output.Write([]byte("[#E06C75]✗ user.email is not set\n"))
	} else {
		c.output.Write([]byte(fmt.Sprintf("[#7EC87E]✓ user.email is set (%s)\n", identity.Email)))
	}

	cfg, err := kzconfig.Load()
	if err == nil && len(cfg.Rules.AllowedDomains) > 0 {
		if identity.Email != "" && emailMatchesAnyDomain(identity.Email, cfg.Rules.AllowedDomains) {
			c.output.Write([]byte(fmt.Sprintf("[#7EC87E]✓ email domain is allowed (%s)\n", strings.Join(cfg.Rules.AllowedDomains, ", "))))
		} else if identity.Email != "" {
			c.output.Write([]byte(fmt.Sprintf("[#E06C75]✗ email domain is not allowed (expected: %s)\n", strings.Join(cfg.Rules.AllowedDomains, ", "))))
		}
	}
}

func (c *commandInput) cmdStatus(_ string) {
	repo, err := kzgit.OpenRepository(".")
	if err != nil {
		c.output.Write([]byte("[#E06C75]✗ Not a git repository\n"))
		return
	}

	branch, err := repo.Branch()
	if err != nil || branch == "" {
		branch = "detached HEAD"
	}

	commit, err := repo.LastCommit()
	if err != nil {
		c.output.Write([]byte("[#6C9EEB::b]Repository Status:\n"))
		c.output.Write([]byte(fmt.Sprintf("  [white]Path   : %s\n", repo.Path)))
		c.output.Write([]byte(fmt.Sprintf("  [white]Branch : %s\n", branch)))
		c.output.Write([]byte("[#6C7086]No commits yet.\n"))
		return
	}

	c.output.Write([]byte("[#6C9EEB::b]Repository Status:\n"))
	c.output.Write([]byte(fmt.Sprintf("  [white]Path   : %s\n", repo.Path)))
	c.output.Write([]byte(fmt.Sprintf("  [white]Branch : %s\n", branch)))
	c.output.Write([]byte(fmt.Sprintf("  [white]Author : %s\n", commit.Author)))
	c.output.Write([]byte(fmt.Sprintf("  [white]Email  : %s\n", commit.Email)))
}

func (c *commandInput) cmdQuit(_ string) {
	c.app.stop()
}

func emailMatchesAnyDomain(email string, domains []string) bool {
	at := strings.LastIndex(email, "@")
	if at == -1 || at == len(email)-1 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range domains {
		if strings.ToLower(strings.TrimSpace(d)) == domain {
			return true
		}
	}
	return false
}
