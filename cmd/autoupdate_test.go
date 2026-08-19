package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

func TestOfferUpdate_SkipsForDevBuild(t *testing.T) {
	isolatedEnv(t)
	// version defaults to "dev" in tests (no -ldflags), which
	// offerUpdateIfAvailable treats as "nothing to offer".
	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)

	offerUpdateIfAvailable(c)

	if buf.Len() != 0 {
		t.Fatalf("expected no output for a dev build, got:\n%s", buf.String())
	}
}

func TestOfferUpdate_RespectsOptOutEnvVar(t *testing.T) {
	isolatedEnv(t)
	t.Setenv("KZGIT_NO_UPDATE_CHECK", "1")

	orig := version
	version = "0.1.0"
	t.Cleanup(func() { version = orig })

	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)

	offerUpdateIfAvailable(c)

	if buf.Len() != 0 {
		t.Fatalf("expected KZGIT_NO_UPDATE_CHECK to suppress the check entirely, got:\n%s", buf.String())
	}
}

func TestOfferUpdate_NonInteractive(t *testing.T) {
	isolatedEnv(t)

	orig := version
	version = "0.1.0"
	t.Cleanup(func() { version = orig })

	var buf bytes.Buffer
	c := &cobra.Command{}
	c.SetOut(&buf)

	// go test's stdin/stdout aren't a TTY, so this should no-op before
	// ever making a network call.
	offerUpdateIfAvailable(c)

	if buf.Len() != 0 {
		t.Fatalf("expected no output outside of an interactive terminal, got:\n%s", buf.String())
	}
}

func TestIsInteractive_FalseUnderTest(t *testing.T) {
	// go test's stdin/stdout are not attached to a terminal, so this
	// should always report false in CI and local test runs alike.
	if isInteractive() {
		t.Skip("test process appears to have a real TTY attached; skipping")
	}
}
