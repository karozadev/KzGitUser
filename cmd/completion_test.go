package cmd

import (
	"testing"

	kzconfig "github.com/karoza/kz-git-user/internal/config"
	"github.com/spf13/cobra"
)

func TestCompleteProfileNames_Empty(t *testing.T) {
	isolatedEnv(t)

	names, directive := completeProfileNames(nil, nil, "")
	if len(names) != 0 {
		t.Fatalf("expected no suggestions with no saved profiles, got %v", names)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}

func TestCompleteProfileNames_WithProfiles(t *testing.T) {
	isolatedEnv(t)
	cfg, err := kzconfig.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Add("work", "John Doe", "john@company.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Add("personal", "John Doe", "john@gmail.com"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	names, directive := completeProfileNames(nil, nil, "")
	if len(names) != 2 || names[0] != "personal" || names[1] != "work" {
		t.Fatalf("expected sorted profile names [personal work], got %v", names)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}

func TestCompleteProfileNames_ArgAlreadyGiven(t *testing.T) {
	isolatedEnv(t)
	names, directive := completeProfileNames(nil, []string{"work"}, "")
	if names != nil {
		t.Fatalf("expected no suggestions once a profile arg is already given, got %v", names)
	}
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}

func TestSwitchAndRemove_HaveCompletion(t *testing.T) {
	if newSwitchCmd().ValidArgsFunction == nil {
		t.Fatal("expected 'switch' to register a ValidArgsFunction")
	}
	if newProfilesRemoveCmd().ValidArgsFunction == nil {
		t.Fatal("expected 'profiles remove' to register a ValidArgsFunction")
	}
}
