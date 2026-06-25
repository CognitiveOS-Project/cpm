package resolver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/checksum"
)

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func resolveGHR(target string) (*Result, error) {
	target = strings.TrimPrefix(target, "ghr:")

	parts := strings.SplitN(target, "@", 2)
	repo := parts[0]
	tag := ""
	if len(parts) == 2 {
		tag = parts[1]
	}

	if tag != "" {
		return resolveGHRAsset(repo, tag)
	}

	// No tag — fetch latest release
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("latest release: HTTP %d", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	return downloadGHRAsset(repo, &rel)
}

func resolveGHRAsset(repo, tag string) (*Result, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch release %s: %w", tag, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release %s: HTTP %d", tag, resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("parse release: %w", err)
	}

	return downloadGHRAsset(repo, &rel)
}

func downloadGHRAsset(repo string, rel *release) (*Result, error) {
	var cgpAsset *asset
	for _, a := range rel.Assets {
		if strings.HasSuffix(a.Name, ".cgp") || strings.HasSuffix(a.Name, ".tar.gz") || strings.HasSuffix(a.Name, ".tgz") {
			cgpAsset = &a
			break
		}
	}
	if cgpAsset == nil {
		return nil, fmt.Errorf("no .cgp/.tar.gz asset in release %s of %s", rel.TagName, repo)
	}

	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("cpm-ghr-%s-%s", sanitize(repo), cgpAsset.Name))
	f, err := os.Create(archivePath)
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	req, _ := http.NewRequest("GET", cgpAsset.BrowserDownloadURL, nil)
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		f.Close()
		os.Remove(archivePath)
		return nil, fmt.Errorf("download asset: %w", err)
	}
	defer resp.Body.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(archivePath)
		return nil, fmt.Errorf("write asset: %w", err)
	}
	f.Close()

	sum, err := checksum.OfFile(archivePath)
	if err != nil {
		return nil, err
	}

	return normalizeArchive(archivePath, sum)
}
