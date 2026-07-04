package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type npmPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

func resolveNPM(target string) (*Result, error) {
	target = strings.TrimPrefix(target, "npm:")
	target = strings.TrimPrefix(target, "bun:")

	version := "latest"
	pkgName := target

	if strings.HasPrefix(target, "@") {
		// @scope/name@version
		if idx := strings.LastIndex(target, "@"); idx > 1 {
			pkgName = target[:idx]
			version = target[idx+1:]
		}
	} else {
		if idx := strings.Index(target, "@"); idx > 0 {
			pkgName = target[:idx]
			version = target[idx+1:]
		}
	}

	url := fmt.Sprintf("https://registry.npmjs.org/%s/%s", pkgName, version)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch npm package %s: %w", pkgName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("npm package %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var pkg npmPackage
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("parse npm response: %w", err)
	}

	if pkg.Dist.Tarball == "" {
		return nil, fmt.Errorf("npm package %s has no tarball", pkgName)
	}

	archivePath, sum, err := downloadArchive(pkg.Dist.Tarball)
	if err != nil {
		return nil, fmt.Errorf("download tarball: %w", err)
	}

	return normalizeArchive(archivePath, sum)
}
