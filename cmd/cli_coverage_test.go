package cmd

import (
	"strings"
	"testing"
)

func TestInstallCommandExists(t *testing.T) {
	if installCmd == nil {
		t.Fatal("installCmd should not be nil")
	}
	if installCmd.Use != "install <path|name>" {
		t.Fatalf("expected 'install <path|name>' use, got %s", installCmd.Use)
	}
}

func TestListCommandExists(t *testing.T) {
	if listCmd == nil {
		t.Fatal("listCmd should not be nil")
	}
	if listCmd.Use != "list" {
		t.Fatalf("expected 'list' use, got %s", listCmd.Use)
	}
}

func TestSearchCommandExists(t *testing.T) {
	if searchCmd == nil {
		t.Fatal("searchCmd should not be nil")
	}
	if searchCmd.Use != "search <query>" {
		t.Fatalf("expected 'search <query>' use, got %s", searchCmd.Use)
	}
}

func TestRemoveCommandExists(t *testing.T) {
	if removeCmd == nil {
		t.Fatal("removeCmd should not be nil")
	}
	if removeCmd.Use != "remove <name>" {
		t.Fatalf("expected 'remove <name>' use, got %s", removeCmd.Use)
	}
}

func TestDownloadWeightsCommandExists(t *testing.T) {
	if downloadWeightsCmd == nil {
		t.Fatal("downloadWeightsCmd should not be nil")
	}
	if downloadWeightsCmd.Use != "download-weights <model-name>" {
		t.Fatalf("expected 'download-weights <model-name>' use, got %s", downloadWeightsCmd.Use)
	}
}

func TestInstallDependenciesCommandExists(t *testing.T) {
	if installDepsCmd == nil {
		t.Fatal("installDepsCmd should not be nil")
	}
	if installDepsCmd.Use != "install-dependencies" {
		t.Fatalf("expected 'install-dependencies' use, got %s", installDepsCmd.Use)
	}
}

func TestRegisterDependenciesCommandExists(t *testing.T) {
	if registerDepsCmd == nil {
		t.Fatal("registerDepsCmd should not be nil")
	}
	if registerDepsCmd.Use != "register-dependencies <package>" {
		t.Fatalf("expected 'register-dependencies <package>' use, got %s", registerDepsCmd.Use)
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}
	cmds := rootCmd.Commands()
	if len(cmds) == 0 {
		t.Fatal("rootCmd should have subcommands")
	}
}

func TestCommandNames(t *testing.T) {
	expected := []string{"init", "install", "remove", "list", "info", "verify",
		"search", "update", "publish", "download-weights", "auth", "secret",
		"pack", "install-dependencies", "register-dependencies"}
	found := make(map[string]bool)
	for _, c := range rootCmd.Commands() {
		found[c.Name()] = true
	}
	for _, name := range expected {
		if !found[name] {
			t.Fatalf("expected command %s to exist", name)
		}
	}
}

func TestSearchCommandValidation(t *testing.T) {
	cmd := searchCmd
	if cmd == nil {
		t.Fatal("searchCmd should not be nil")
	}
	if !strings.Contains(strings.ToLower(cmd.Short), "search") {
		t.Fatalf("search command short description should mention search, got: %s", cmd.Short)
	}
}
