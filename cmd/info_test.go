package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeInfoTestManifest(t *testing.T, dir string, m map[string]interface{}) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "cognitive.json")
	data, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInfoJSONFromPath(t *testing.T) {
	dir := t.TempDir()
	writeInfoTestManifest(t, dir, map[string]interface{}{
		"name":        "test-pkg",
		"version":     "1.0.0",
		"description": "A test package",
		"author":      "Tester",
	})

	path := filepath.Join(dir, "cognitive.json")

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	infoJSON = true
	infoManifest = path
	err := infoCmd.RunE(nil, []string{"ignored"})
	w.Close()
	os.Stdout = old
	infoJSON = false
	infoManifest = ""

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var output map[string]interface{}
	json.NewDecoder(r).Decode(&output)

	if output["name"] != "test-pkg" {
		t.Errorf("expected name test-pkg, got %v", output["name"])
	}
	if output["version"] != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %v", output["version"])
	}
	if output["description"] != "A test package" {
		t.Errorf("expected description, got %v", output["description"])
	}
	if output["author"] != "Tester" {
		t.Errorf("expected author Tester, got %v", output["author"])
	}
}

func TestInfoJSONFromCWD(t *testing.T) {
	dir := t.TempDir()
	writeInfoTestManifest(t, dir, map[string]interface{}{
		"name":    "cwd-pkg",
		"version": "0.1.0",
	})

	// Save and restore CWD
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	old := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	infoJSON = true
	infoManifest = ""
	err := infoCmd.RunE(nil, []string{"ignored"})
	w.Close()
	os.Stdout = old
	infoJSON = false

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestInfoJSONFilename(t *testing.T) {
	dir := t.TempDir()
	writeInfoTestManifest(t, dir, map[string]interface{}{
		"name":    "my-skill",
		"version": "0.1.0",
	})

	path := filepath.Join(dir, "cognitive.json")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	infoJSON = true
	infoManifest = path
	err := infoCmd.RunE(nil, []string{"ignored"})
	w.Close()
	os.Stdout = old
	infoJSON = false

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var output map[string]interface{}
	json.NewDecoder(r).Decode(&output)

	if output["filename"] != "my-skill-0.1.0-universal.cgp" {
		t.Errorf("expected universal filename, got %v", output["filename"])
	}
}

func TestInfoJSONFilenameWithOS(t *testing.T) {
	dir := t.TempDir()
	writeInfoTestManifest(t, dir, map[string]interface{}{
		"name":    "native-pkg",
		"version": "2.0.0",
		"hardware_requirements": map[string]interface{}{
			"os":   []string{"linux"},
			"arch": []string{"arm64"},
		},
	})

	path := filepath.Join(dir, "cognitive.json")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	infoJSON = true
	infoManifest = path
	err := infoCmd.RunE(nil, []string{"ignored"})
	w.Close()
	os.Stdout = old
	infoJSON = false

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var output map[string]interface{}
	json.NewDecoder(r).Decode(&output)

	if output["filename"] != "native-pkg-2.0.0-linux-arm64.cgp" {
		t.Errorf("expected linux-arm64 filename, got %v", output["filename"])
	}
}

func TestInfoPlainRequiresName(t *testing.T) {
	// Plain-text mode without arg should error (cobra validates Args)
	err := infoCmd.Args(infoCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing name arg")
	}
}
