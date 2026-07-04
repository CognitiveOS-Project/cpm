package resolver

import (
	"fmt"
	"strings"
)

func resolveRegistry(target, registryURL string) (*Result, error) {
	pkgURL := strings.TrimRight(registryURL, "/") + "/v1/packages/" + target

	archivePath, sum, err := downloadArchive(pkgURL)
	if err != nil {
		return nil, fmt.Errorf("registry lookup %s: %w", target, err)
	}

	return normalizeArchive(archivePath, sum)
}
