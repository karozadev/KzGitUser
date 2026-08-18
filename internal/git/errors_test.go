package git

import (
	"errors"
	"testing"
)

func TestCommandError(t *testing.T) {
	inner := errors.New("boom")
	err := &CommandError{Args: []string{"config", "--get", "user.name"}, Stderr: "fatal: bad config", Err: inner}

	if got := err.Error(); got != "git config --get user.name: fatal: bad config" {
		t.Fatalf("unexpected Error(): %s", got)
	}
	if !errors.Is(err, inner) {
		t.Fatal("expected Unwrap to expose the inner error")
	}
}

func TestCommandError_NoStderr(t *testing.T) {
	inner := errors.New("boom")
	err := &CommandError{Args: []string{"status"}, Err: inner}
	if got := err.Error(); got != "git status: boom" {
		t.Fatalf("unexpected Error(): %s", got)
	}
}
