package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/config"
)

func setupTestSecrets(t *testing.T) string {
	t.Helper()
	patchesDir := t.TempDir()
	os.Setenv("CPM_PATCHES_DIR", filepath.Join(patchesDir, "patches"))
	t.Cleanup(func() { os.Unsetenv("CPM_PATCHES_DIR") })

	homeDir := t.TempDir()
	os.Setenv("HOME", homeDir)
	t.Cleanup(func() { os.Unsetenv("HOME") })

	return patchesDir
}

func TestSecretAddAndGet(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "add", "TEST_KEY", "test-value-123"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "get", "TEST_KEY"})
	buf.Reset()
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("get: %v", err)
	}
	if buf.String() != "test-value-123" {
		t.Errorf("get = %q, want test-value-123", buf.String())
	}
}

func TestSecretAddDuplicateWithoutForce(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "add", "DUP_KEY", "val1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "add", "DUP_KEY", "val2"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for duplicate without --force")
	}
}

func TestSecretAddDuplicateWithForce(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "add", "FORCE_KEY", "val1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "add", "FORCE_KEY", "val2", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("force add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "get", "FORCE_KEY"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.Execute()
	if buf.String() != "val2" {
		t.Errorf("after force: got %q, want val2", buf.String())
	}
}

func TestSecretRemove(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "add", "DEL_KEY", "to-delete"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "remove", "DEL_KEY"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("remove: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "get", "DEL_KEY"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for removed secret")
	}
}

func TestSecretListEmpty(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "list"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("No secrets")) {
		t.Errorf("expected 'No secrets' message, got: %s", buf.String())
	}
}

func TestSecretListWithSecrets(t *testing.T) {
	setupTestSecrets(t)

	rootCmd.SetArgs([]string{"secret", "add", "LIST_KEY", "list-val"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}

	rootCmd.SetArgs([]string{"secret", "list"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("LIST_KEY")) {
		t.Errorf("expected LIST_KEY in output, got: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("****")) {
		t.Errorf("expected masked output, got: %s", buf.String())
	}
}

func TestSecretGlobalScope(t *testing.T) {
	setupTestSecrets(t)

	s := config.NewSecrets()
	s.Secrets["GLOBAL_KEY"] = "global-val"
	config.SaveSecrets("global", "", s)

	rootCmd.SetArgs([]string{"secret", "get", "GLOBAL_KEY"})
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("get global: %v", err)
	}
	if buf.String() != "global-val" {
		t.Errorf("get = %q, want global-val", buf.String())
	}
}
