package cmd

import (
	"os"
	"testing"
)

func TestExecute_Success(t *testing.T) {
	isolatedEnv(t)
	origArgs := os.Args
	os.Args = []string{"kzgit", "version"}
	t.Cleanup(func() { os.Args = origArgs })

	if code := Execute(); code != 0 {
		t.Fatalf("Execute() = %d, want 0", code)
	}
}

func TestExecute_Failure(t *testing.T) {
	isolatedEnv(t)
	origArgs := os.Args
	os.Args = []string{"kzgit", "profiles", "remove", "does-not-exist"}
	t.Cleanup(func() { os.Args = origArgs })

	if code := Execute(); code != 1 {
		t.Fatalf("Execute() = %d, want 1", code)
	}
}
