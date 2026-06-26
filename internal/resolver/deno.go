package resolver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type denoModule struct {
	Name     string `json:"name"`
	Latest   string `json:"latest"`
	Versions []struct {
		Version string `json:"version"`
	} `json:"versions,omitempty"`
}

type jsrPackage struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Dist    struct {
		Tarball string `json:"tarball"`
	} `json:"dist,omitempty"`
}

func resolveDeno(target string) (*Result, error) {
	target = strings.TrimPrefix(target, "deno:")

	version := "latest"
	parts := strings.SplitN(target, "@", 2)
	pkgName := parts[0]
	if len(parts) == 2 {
		version = parts[1]
	}

	// Try JSR first for scoped packages
	if strings.Count(pkgName, "/") == 1 {
		result, err := resolveJSR(pkgName, version)
		if err == nil {
			return result, nil
		}
	}

	// Fallback to deno.land/x/
	return resolveDenoLand(pkgName, version)
}

func resolveDenoLand(pkgName, version string) (*Result, error) {
	url := fmt.Sprintf("https://apiland.deno.dev/v2/modules/%s", pkgName)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch deno module %s: %w", pkgName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deno module %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var mod denoModule
	if err := json.NewDecoder(resp.Body).Decode(&mod); err != nil {
		return nil, fmt.Errorf("parse deno response: %w", err)
	}

	ver := mod.Latest
	if version != "latest" {
		ver = version
	}

	tarballURL := fmt.Sprintf("https://apiland.deno.dev/v2/modules/%s/versions/%s/tarball", pkgName, ver)
	archivePath, sum, err := downloadArchive(tarballURL)
	if err != nil {
		return nil, fmt.Errorf("download deno module: %w", err)
	}

	return normalizeArchive(archivePath, sum)
}

func resolveJSR(pkgName, version string) (*Result, error) {
	scope, name := splitJSRPackage(pkgName)
	npmName := fmt.Sprintf("@jsr/%s__%s", scope, name)
	url := fmt.Sprintf("https://registry.npmjs.org/%s", npmName)
	if version != "latest" {
		url += "/" + version
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch JSR package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JSR package %s: HTTP %d", pkgName, resp.StatusCode)
	}

	var pkg jsrPackage
	if err := json.NewDecoder(resp.Body).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("parse JSR response: %w", err)
	}

	if pkg.Dist.Tarball == "" {
		return nil, fmt.Errorf("JSR package %s has no tarball", pkgName)
	}

	archivePath, sum, err := downloadArchive(pkg.Dist.Tarball)
	if err != nil {
		return nil, fmt.Errorf("download JSR tarball: %w", err)
	}

	return normalizeArchive(archivePath, sum)
}

func splitJSRPackage(pkgName string) (string, string) {
	parts := strings.SplitN(pkgName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return pkgName, pkgName
}
