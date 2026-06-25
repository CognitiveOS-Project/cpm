package resolver

import (
	"fmt"
)

func resolveURL(target string) (*Result, error) {
	archivePath, sum, err := downloadArchive(target)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", target, err)
	}

	return normalizeArchive(archivePath, sum)
}
