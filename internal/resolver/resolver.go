package resolver

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/checksum"
	"github.com/CognitiveOS-Project/cpm/internal/normalize"
)

type Result struct {
	Manifest    *archive.Manifest
	ArchivePath string
	DataDir     string
	Checksum    string
}

func Resolve(target string, registryURL string) (*Result, error) {
	if isLocalPath(target) {
		return resolveLocal(target)
	}

	if strings.HasPrefix(target, "github.com/") || strings.HasPrefix(target, "github:") {
		return resolveGit(target, "github.com")
	}

	if strings.HasPrefix(target, "gitlab:") {
		return resolveGit(target, "gitlab.com")
	}

	if strings.HasPrefix(target, "bitbucket:") {
		return resolveGit(target, "bitbucket.org")
	}

	if strings.HasPrefix(target, "ghr:") {
		return resolveGHR(target)
	}

	if strings.HasPrefix(target, "npm:") {
		return resolveNPM(target)
	}

	if strings.HasPrefix(target, "bun:") {
		return resolveNPM(target) // bun uses npm registry with different slug
	}

	if strings.HasPrefix(target, "deno:") {
		return resolveDeno(target)
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		return resolveURL(target)
	}

	if registryURL != "" {
		return resolveRegistry(target, registryURL)
	}

	return nil, fmt.Errorf("unable to resolve %q: no registry URL configured and no protocol handler matched", target)
}

func isLocalPath(target string) bool {
	if strings.HasPrefix(target, "./") || strings.HasPrefix(target, "../") || strings.HasPrefix(target, "/") {
		return true
	}
	if strings.Contains(target, string(filepath.Separator)) {
		info, err := os.Stat(target)
		return err == nil && (info.IsDir() || strings.HasSuffix(target, ".cgp"))
	}
	return false
}

func resolveLocal(target string) (*Result, error) {
	info, err := os.Stat(target)
	if err != nil {
		return nil, fmt.Errorf("local path %q: %w", target, err)
	}

	if info.IsDir() {
		return resolveLocalDir(target)
	}

	if strings.HasSuffix(target, ".cgp") || strings.HasSuffix(target, ".tar.gz") || strings.HasSuffix(target, ".tgz") {
		return resolveLocalArchive(target)
	}

	return nil, fmt.Errorf("local path %q is not a .cgp/.tar.gz archive or directory", target)
}

func resolveLocalDir(target string) (*Result, error) {
	data, err := os.ReadFile(filepath.Join(target, "cognitive.json"))
	if err != nil {
		return nil, fmt.Errorf("directory %q has no cognitive.json: %w", target, err)
	}
	var m archive.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &Result{
		Manifest: &m,
		DataDir:  target,
	}, nil
}

func resolveLocalArchive(target string) (*Result, error) {
	sum, err := checksum.OfFile(target)
	if err != nil {
		return nil, err
	}

	result, err := normalize.Archive(target)
	if err != nil {
		return nil, err
	}

	return &Result{
		Manifest:    result.Manifest,
		ArchivePath: target,
		DataDir:     result.DataDir,
		Checksum:    sum,
	}, nil
}
