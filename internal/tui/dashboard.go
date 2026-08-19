package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	kzgit "github.com/karoza/kz-git-user/internal/git"
	"github.com/rivo/tview"
)

type dashboard struct {
	flex   *tview.Flex
	app    *App
	fields map[string]*tview.TextView
}

func newDashboard(app *App) *dashboard {
	d := &dashboard{
		app:    app,
		fields: make(map[string]*tview.TextView),
	}
	d.flex = d.build()
	return d
}

func (d *dashboard) build() *tview.Flex {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBackgroundColor(colorBackground)

	title := newHeaderView("KzGitUser", "Git Identity Manager")
	root.AddItem(title, 2, 0, false)
	root.AddItem(newSeparator(), 1, 0, false)

	identitySection := d.buildIdentitySection()
	root.AddItem(identitySection, 6, 0, false)

	root.AddItem(newSeparator(), 1, 0, false)

	repoSection := d.buildRepoSection()
	root.AddItem(repoSection, 7, 0, false)

	root.AddItem(newSeparator(), 1, 0, false)

	lastCommitSection := d.buildLastCommitSection()
	root.AddItem(lastCommitSection, 4, 0, false)

	root.AddItem(tview.NewTextView().SetBackgroundColor(colorBackground), 1, 0, false)

	return root
}

func (d *dashboard) buildIdentitySection() *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(colorBackground)

	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetBackgroundColor(colorBackground)
	fmt.Fprintf(header, "[#6C9EEB::b] Current Git Profile ")
	flex.AddItem(header, 1, 0, false)
	flex.AddItem(newSeparator(), 1, 0, false)

	nameRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	nameLabel := newInfoRow("Name", "")
	nameRow.AddItem(nameLabel, 0, 1, false)
	d.fields["name"] = nameLabel
	flex.AddItem(nameRow, 1, 0, false)

	emailRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	emailLabel := newInfoRow("Email", "")
	emailRow.AddItem(emailLabel, 0, 1, false)
	d.fields["email"] = emailLabel
	flex.AddItem(emailRow, 1, 0, false)

	sourceRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	sourceLabel := newInfoRow("Source", "")
	sourceRow.AddItem(sourceLabel, 0, 1, false)
	d.fields["source"] = sourceLabel
	flex.AddItem(sourceRow, 1, 0, false)

	return flex
}

func (d *dashboard) buildRepoSection() *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(colorBackground)

	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetBackgroundColor(colorBackground)
	fmt.Fprintf(header, "[#6C9EEB::b] Repository ")
	flex.AddItem(header, 1, 0, false)
	flex.AddItem(newSeparator(), 1, 0, false)

	repoPathRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	repoPathLabel := newInfoRow("Path", "")
	repoPathRow.AddItem(repoPathLabel, 0, 1, false)
	d.fields["repoPath"] = repoPathLabel
	flex.AddItem(repoPathRow, 1, 0, false)

	repoNameRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	repoNameLabel := newInfoRow("Repository", "")
	repoNameRow.AddItem(repoNameLabel, 0, 1, false)
	d.fields["repoName"] = repoNameLabel
	flex.AddItem(repoNameRow, 1, 0, false)

	branchRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	branchLabel := newInfoRow("Branch", "")
	branchRow.AddItem(branchLabel, 0, 1, false)
	d.fields["branch"] = branchLabel
	flex.AddItem(branchRow, 1, 0, false)

	statusRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	statusLabel := newInfoRow("Status", "")
	statusRow.AddItem(statusLabel, 0, 1, false)
	d.fields["status"] = statusLabel
	flex.AddItem(statusRow, 1, 0, false)

	return flex
}

func (d *dashboard) buildLastCommitSection() *tview.Flex {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBackgroundColor(colorBackground)

	header := tview.NewTextView()
	header.SetDynamicColors(true)
	header.SetBackgroundColor(colorBackground)
	fmt.Fprintf(header, "[#6C9EEB::b] Last Commit ")
	flex.AddItem(header, 1, 0, false)
	flex.AddItem(newSeparator(), 1, 0, false)

	commitAuthorRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	commitAuthorLabel := newInfoRow("Author", "")
	commitAuthorRow.AddItem(commitAuthorLabel, 0, 1, false)
	d.fields["commitAuthor"] = commitAuthorLabel
	flex.AddItem(commitAuthorRow, 1, 0, false)

	commitEmailRow := tview.NewFlex().SetDirection(tview.FlexColumn)
	commitEmailLabel := newInfoRow("Email", "")
	commitEmailRow.AddItem(commitEmailLabel, 0, 1, false)
	d.fields["commitEmail"] = commitEmailLabel
	flex.AddItem(commitEmailRow, 1, 0, false)

	return flex
}

func (d *dashboard) refresh() {
	dir := "."

	identity, err := kzgit.ResolveIdentity(dir)
	if err != nil {
		d.fields["name"].SetText("[#E06C75]Error resolving identity")
		d.fields["email"].SetText(err.Error())
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
	sourceVal := string(identitySource(identity))

	d.fields["name"].SetText(fmt.Sprintf("[#6C7086]Name       : [white]%s", nameVal))
	d.fields["email"].SetText(fmt.Sprintf("[#6C7086]Email      : [white]%s", emailVal))
	d.fields["source"].SetText(fmt.Sprintf("[#6C7086]Source     : [white]%s", sourceVal))

	repo, err := kzgit.OpenRepository(dir)
	if err != nil {
		d.fields["repoPath"].SetText("[#6C7086]Path       : [colorForeground]Not inside a Git repository")
		d.fields["repoName"].SetText("[#6C7086]Repository : [colorForeground]-")
		d.fields["branch"].SetText("[#6C7086]Branch     : [colorForeground]-")
		d.fields["status"].SetText("[#6C7086]Status     : [colorForeground]-")
		d.fields["commitAuthor"].SetText("[#6C7086]Author     : [colorForeground]No commits")
		d.fields["commitEmail"].SetText("[#6C7086]Email      : [colorForeground]-")
		return
	}

	repoName := filepath.Base(repo.Path)
	d.fields["repoPath"].SetText(fmt.Sprintf("[#6C7086]Path       : [white]%s", repo.Path))
	d.fields["repoName"].SetText(fmt.Sprintf("[#6C7086]Repository : [white]%s", repoName))

	branch, err := repo.Branch()
	if err != nil || branch == "" {
		d.fields["branch"].SetText("[#6C7086]Branch     : [colorForeground]detached HEAD")
	} else {
		d.fields["branch"].SetText(fmt.Sprintf("[#6C7086]Branch     : [white]%s", branch))
	}

	d.fields["status"].SetText("[#6C7086]Status     : [white]clean")

	commit, err := repo.LastCommit()
	if err != nil {
		d.fields["commitAuthor"].SetText("[#6C7086]Author     : [colorForeground]No commits yet")
		d.fields["commitEmail"].SetText("[#6C7086]Email      : [colorForeground]-")
	} else {
		d.fields["commitAuthor"].SetText(fmt.Sprintf("[#6C7086]Author     : [white]%s", commit.Author))
		d.fields["commitEmail"].SetText(fmt.Sprintf("[#6C7086]Email      : [white]%s", commit.Email))
	}
}

func identitySource(identity *kzgit.Identity) string {
	if identity.Email != "" {
		return string(identity.EmailScope)
	}
	if identity.Name != "" {
		return string(identity.NameScope)
	}
	return string(kzgit.ScopeUnknown)
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
