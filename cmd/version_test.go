package cmd

import (
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	isolatedEnv(t)

	out, err := execCmd(t, "version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if !strings.Contains(out, "kzgit version") {
		t.Fatalf("unexpected version output: %s", out)
	}
}
