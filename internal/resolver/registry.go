package resolver

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/registry"
)

func resolveRegistry(target, registryURL string) (*Result, error) {
	rc := registry.New(registryURL)

	name, version := target, ""
	if i := strings.LastIndex(target, "@"); i > 0 {
		name = target[:i]
		version = target[i+1:]
	}

	meta, err := rc.GetMetadata(name, version)
	if err != nil {
		return nil, fmt.Errorf("registry lookup %s: %w", target, err)
	}

	if meta.Status != "" && meta.Status != "active" {
		return nil, fmt.Errorf("package %s is %s (not active)", target, meta.Status)
	}

	opts := registry.DownloadOptions{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if meta.DownloadURL != "" {
		archivePath, sum, err := downloadArchive(meta.DownloadURL)
		if err != nil {
			return nil, fmt.Errorf("download %s: %w", target, err)
		}

		if meta.ChecksumSHA256 != "" && sum != meta.ChecksumSHA256 {
			return nil, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", target, meta.ChecksumSHA256, sum)
		}

		return normalizeArchive(archivePath, sum)
	}

	body, err := rc.Download(target, meta.Version, opts)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", target, err)
	}
	defer body.Close()

	archivePath, sum, err := downloadFromReader(body, target)
	if err != nil {
		return nil, fmt.Errorf("save archive: %w", err)
	}

	if meta.ChecksumSHA256 != "" && sum != meta.ChecksumSHA256 {
		return nil, fmt.Errorf("checksum mismatch for %s: expected %s, got %s", target, meta.ChecksumSHA256, sum)
	}

	return normalizeArchive(archivePath, sum)
}
