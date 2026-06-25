package normalize

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func writeTestArchive(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "pkg.tar.gz")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0644,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	tw.Close()
	gw.Close()
	return path
}

func TestArchiveCognitiveJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeTestArchive(t, dir, map[string]string{
		"cognitive.json": `{"name":"test-pkg","version":"1.0.0","description":"test"}`,
	})

	result, err := Archive(path)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}
	if result.Manifest.Name != "test-pkg" {
		t.Fatalf("expected test-pkg, got %s", result.Manifest.Name)
	}
	if result.Manifest.Version != "1.0.0" {
		t.Fatalf("expected 1.0.0, got %s", result.Manifest.Version)
	}
	os.RemoveAll(result.DataDir)
}

func TestArchivePackageJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeTestArchive(t, dir, map[string]string{
		"package.json": `{
			"name":"my-npm-pkg",
			"version":"2.0.0",
			"description":"npm package",
			"author":"test",
			"cognitive_os": {
				"runtime": "nodejs:18",
				"dependencies": {"hello":"^1.0.0"}
			}
		}`,
	})

	result, err := Archive(path)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}
	if result.Manifest.Name != "my-npm-pkg" {
		t.Fatalf("expected my-npm-pkg, got %s", result.Manifest.Name)
	}
	if result.Manifest.Dependencies["hello"] != "^1.0.0" {
		t.Fatalf("expected dep hello:^1.0.0, got %s", result.Manifest.Dependencies["hello"])
	}
	os.RemoveAll(result.DataDir)
}

func TestArchiveNoManifest(t *testing.T) {
	dir := t.TempDir()
	path := writeTestArchive(t, dir, map[string]string{
		"readme.md": "# hello",
	})

	_, err := Archive(path)
	if err == nil {
		t.Fatal("expected error for no manifest")
	}
}
