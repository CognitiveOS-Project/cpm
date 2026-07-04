package checksum

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOfFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	_ = os.WriteFile(path, []byte("hello world"), 0644)

	sum, err := OfFile(path)
	if err != nil {
		t.Fatalf("OfFile failed: %v", err)
	}
	if sum != "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9" {
		t.Fatalf("unexpected hash: %s", sum)
	}
}

func TestVerify(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	_ = os.WriteFile(path, []byte("data"), 0644)

	sum, _ := OfFile(path)
	if err := Verify(path, sum); err != nil {
		t.Fatalf("Verify should pass: %v", err)
	}

	if err := Verify(path, "badhash"); err == nil {
		t.Fatal("Verify should fail on bad hash")
	}
}
