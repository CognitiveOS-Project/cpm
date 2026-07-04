package archive

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReadManifest(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	manifest := Manifest{
		Name:        "test-patch",
		Version:     "1.0.0",
		Description: "A test patch",
		Author:      "Test",
		License:     "MIT",
	}

	b, _ := json.Marshal(manifest)
	_ = tw.WriteHeader(&tar.Header{
		Name: "cognitive.json",
		Mode: 0644,
		Size: int64(len(b)),
	})
	_, _ = tw.Write(b)
	tw.Close()
	gz.Close()

	m, err := ReadManifest(&buf)
	if err != nil {
		t.Fatalf("ReadManifest failed: %v", err)
	}
	if m.Name != "test-patch" {
		t.Fatalf("expected test-patch, got %s", m.Name)
	}
}

func TestReadManifest_WithDotSlash(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	manifest := Manifest{
		Name:        "dot-slash-patch",
		Version:     "0.1.0",
		Description: "Testing ./ prefix",
	}

	b, _ := json.Marshal(manifest)
	_ = tw.WriteHeader(&tar.Header{
		Name: "./cognitive.json",
		Mode: 0644,
		Size: int64(len(b)),
	})
	_, _ = tw.Write(b)
	tw.Close()
	gz.Close()

	m, err := ReadManifest(&buf)
	if err != nil {
		t.Fatalf("ReadManifest with ./ prefix failed: %v", err)
	}
	if m.Name != "dot-slash-patch" {
		t.Fatalf("expected dot-slash-patch, got %s", m.Name)
	}
}

func TestExtract(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	content := []byte("hello world")
	_ = tw.WriteHeader(&tar.Header{
		Name: "tools/test.sh",
		Mode: 0755,
		Size: int64(len(content)),
	})
	_, _ = tw.Write(content)
	tw.Close()
	gz.Close()

	if err := Extract(&buf, dir); err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "tools", "test.sh"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(data) != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", string(data))
	}
}

func TestExtract_TraversalProtection(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	_ = tw.WriteHeader(&tar.Header{
		Name: "../../etc/passwd",
		Mode: 0644,
		Size: 0,
	})
	tw.Close()
	gz.Close()

	if err := Extract(&buf, dir); err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	// Verify the file was NOT written outside dir
	if _, err := os.Stat(filepath.Join(dir, "..", "..", "etc", "passwd")); err == nil {
		t.Fatal("path traversal succeeded — security vulnerability")
	}
}
