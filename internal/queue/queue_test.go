package queue

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

func TestQueue(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cpm-queue-test-*")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	root := tmpDir
	patchName := "test-patch"
	patchVersion := "1.0.0"
	dep := archive.SystemDependency{
		Name:     "test-dep",
		Version:  "2.0.0",
		Manager:  "apk",
		Stage:    "runtime",
		Required: true,
	}

	t.Run("Register", func(t *testing.T) {
		err := Register(root, patchName, patchVersion, dep)
		if err != nil {
			t.Fatalf("Register failed: %v", err)
		}

		// Verify file exists
		expectedFile := filepath.Join(QueueDir(root), dep.Stage, "test-patch-1_0_0_test-dep-2_0_0.json")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("expected file %s not found", expectedFile)
		}
	})

	t.Run("ListByStage", func(t *testing.T) {
		entries, err := ListByStage(root, "runtime")
		if err != nil {
			t.Fatalf("ListByStage failed: %v", err)
		}

		if len(entries) != 1 {
			t.Errorf("expected 1 entry, got %d", len(entries))
		}

		if entries[0].Dependency.Name != dep.Name {
			t.Errorf("expected dependency %s, got %s", dep.Name, entries[0].Dependency.Name)
		}
	})

	t.Run("MarkInstalled", func(t *testing.T) {
		entries, _ := ListByStage(root, "runtime")
		filename := entries[0].Filename

		err := MarkInstalled(root, "runtime", filename)
		if err != nil {
			t.Fatalf("MarkInstalled failed: %v", err)
		}

		entries, _ = ListByStage(root, "runtime")
		if len(entries) != 0 {
			t.Errorf("expected 0 entries after MarkInstalled, got %d", len(entries))
		}
	})

	t.Run("RemoveByPatch", func(t *testing.T) {
		// Register again
		Register(root, patchName, patchVersion, dep)
		
		err := RemoveByPatch(root, patchName, patchVersion)
		if err != nil {
			t.Fatalf("RemoveByPatch failed: %v", err)
		}

		entries, _ := ListByStage(root, "runtime")
		if len(entries) != 0 {
			t.Errorf("expected 0 entries after RemoveByPatch, got %d", len(entries))
		}
	})
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test/patch", "test-patch"},
		{"test.patch", "test_patch"},
		{"test patch", "test_patch"},
		{"test:patch", "test-patch"},
		{"clean-name", "clean-name"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := sanitize(tc.input)
			if got != tc.expected {
				t.Errorf("sanitize(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}
