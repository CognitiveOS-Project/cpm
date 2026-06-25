package resolver

import (
	"fmt"
	"strings"
)

func resolveGit(target, provider string) (*Result, error) {
	ref := "HEAD"
	target = strings.TrimPrefix(target, provider+":")

	parts := strings.SplitN(target, "@", 2)
	if len(parts) == 2 {
		target = parts[0]
		ref = parts[1]
	}

	var apiURL string
	switch provider {
	case "github.com":
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/tarball/%s", target, ref)
	case "gitlab.com":
		apiURL = fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/repository/archive.tar.gz?ref=%s",
			strings.ReplaceAll(target, "/", "%2F"), ref)
	case "bitbucket.org":
		apiURL = fmt.Sprintf("https://bitbucket.org/%s/get/%s.tar.gz", target, ref)
	default:
		return nil, fmt.Errorf("unsupported git provider: %s", provider)
	}

	archivePath, sum, err := downloadArchive(apiURL)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", provider, err)
	}

	return normalizeArchive(archivePath, sum)
}
