package tui

import (
	"strings"
	"testing"
)

func TestDashboardRefresh_NoRepository(t *testing.T) {
	isolatedEnv(t)
	t.Chdir(t.TempDir())

	app := newTestApp()
	app.dashboard.refresh()

	if !strings.Contains(app.dashboard.fields["repoPath"].GetText(true), "Not inside a Git repository") {
		t.Fatalf("expected not-a-repo message, got: %s", app.dashboard.fields["repoPath"].GetText(true))
	}
	if !strings.Contains(app.dashboard.fields["name"].GetText(true), "Not set") {
		t.Fatalf("expected 'Not set' for name, got: %s", app.dashboard.fields["name"].GetText(true))
	}
}

func TestDashboardRefresh_WithIdentityAndCommit(t *testing.T) {
	isolatedEnv(t)
	dir := initRepoAndChdir(t)
	runGit(t, dir, "config", "--local", "user.name", "Jane Doe")
	runGit(t, dir, "config", "--local", "user.email", "jane@company.com")
	runGit(t, dir, "checkout", "-q", "-b", "main")

	// Create a commit so LastCommit() succeeds.
	writeFileAndCommit(t, dir, "file.txt", "Jane Doe", "jane@company.com")

	app := newTestApp()
	app.dashboard.refresh()

	if !strings.Contains(app.dashboard.fields["email"].GetText(true), "jane@company.com") {
		t.Fatalf("expected identity email, got: %s", app.dashboard.fields["email"].GetText(true))
	}
	if !strings.Contains(app.dashboard.fields["branch"].GetText(true), "main") {
		t.Fatalf("expected branch name, got: %s", app.dashboard.fields["branch"].GetText(true))
	}
	if !strings.Contains(app.dashboard.fields["commitAuthor"].GetText(true), "Jane Doe") {
		t.Fatalf("expected commit author, got: %s", app.dashboard.fields["commitAuthor"].GetText(true))
	}
}

func TestDashboardRefresh_NoCommitsYet(t *testing.T) {
	isolatedEnv(t)
	initRepoAndChdir(t)

	app := newTestApp()
	app.dashboard.refresh()

	if !strings.Contains(app.dashboard.fields["commitAuthor"].GetText(true), "No commits yet") {
		t.Fatalf("expected no-commits message, got: %s", app.dashboard.fields["commitAuthor"].GetText(true))
	}
}

func writeFileAndCommit(t *testing.T, dir, name, author, email string) {
	t.Helper()
	runGit(t, dir, "commit", "--allow-empty", "-q", "-m", "test",
		"--author", author+" <"+email+">")
}
