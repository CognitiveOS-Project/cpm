package resolver

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/checksum"
	"github.com/CognitiveOS-Project/cpm/internal/normalize"
)

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			return r
		}
		return '_'
	}, s)
}

func downloadArchive(url string) (string, string, error) {
	name := sanitize(filepath.Base(url))
	if name == "" || name == "." {
		name = "archive"
	}
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("cpm-dl-%s", name))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(archivePath)
	if err != nil {
		return "", "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(archivePath)
		return "", "", fmt.Errorf("write archive: %w", err)
	}
	f.Close()

	sum, err := checksum.OfFile(archivePath)
	if err != nil {
		os.Remove(archivePath)
		return "", "", err
	}

	return archivePath, sum, nil
}

func downloadFromReader(r io.ReadCloser, nameHint string) (string, string, error) {
	defer r.Close()
	name := sanitize(nameHint)
	if name == "" || name == "." {
		name = "archive"
	}
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("cpm-dl-%s", name))

	f, err := os.Create(archivePath)
	if err != nil {
		return "", "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(archivePath)
		return "", "", fmt.Errorf("write archive: %w", err)
	}
	f.Close()

	sum, err := checksum.OfFile(archivePath)
	if err != nil {
		os.Remove(archivePath)
		return "", "", err
	}

	return archivePath, sum, nil
}

func normalizeArchive(archivePath, checksum string) (*Result, error) {
	nr, err := normalize.Archive(archivePath)
	if err != nil {
		os.Remove(archivePath)
		return nil, err
	}

	return &Result{
		Manifest:    nr.Manifest,
		ArchivePath: archivePath,
		DataDir:     nr.DataDir,
		Checksum:    checksum,
	}, nil
}
