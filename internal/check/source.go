package check

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func IssuesReachable(issuesURL string) error {
	resp, err := httpClient.Get(issuesURL)
	if err != nil {
		return fmt.Errorf("issues URL unreachable: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("issues URL returned HTTP %d", resp.StatusCode)
	}
	return nil
}

type BugCheckResult struct {
	HasBugs bool
	Count   int
	URLs    []string
}

func CheckForBugs(source *archive.SourceInfo) (*BugCheckResult, error) {
	if source == nil {
		return &BugCheckResult{}, nil
	}

	apiURL := deriveIssuesAPI(source)
	if apiURL == "" {
		return nil, fmt.Errorf("unknown git provider and no issues_api set")
	}

	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("issues API unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("issues API returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read issues response: %w", err)
	}

	return parseBugResponse(body, source.Repository)
}

func deriveIssuesAPI(source *archive.SourceInfo) string {
	if source.IssuesAPI != "" {
		return source.IssuesAPI
	}

	repoURL, err := url.Parse(source.Repository)
	if err != nil {
		return ""
	}

	switch repoURL.Host {
	case "github.com":
		parts := strings.Split(strings.Trim(repoURL.Path, "/"), "/")
		if len(parts) < 2 {
			return ""
		}
		return fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?labels=bug&state=open&per_page=1", parts[0], parts[1])

	case "gitlab.com":
		parts := strings.Split(strings.Trim(repoURL.Path, "/"), "/")
		if len(parts) < 2 {
			return ""
		}
		return fmt.Sprintf("https://gitlab.com/api/v4/projects/%s%%2F%s/issues?labels=bug&state=opened&per_page=1", parts[0], parts[1])

	case "bitbucket.org":
		parts := strings.Split(strings.Trim(repoURL.Path, "/"), "/")
		if len(parts) < 2 {
			return ""
		}
		return fmt.Sprintf("https://api.bitbucket.org/2.0/repositories/%s/%s/issues?q=state%%3D%%22new%%22+AND+kind%%3D%%22bug%%22", parts[0], parts[1])

	default:
		return ""
	}
}

func parseBugResponse(body []byte, repoURL string) (*BugCheckResult, error) {
	u, _ := url.Parse(repoURL)
	if u == nil {
		return &BugCheckResult{}, nil
	}

	host := u.Host
	var count int
	var urls []string

	switch host {
	case "github.com", "gitlab.com":
		var issues []struct {
			HTMLURL string `json:"html_url"`
		}
		if err := json.Unmarshal(body, &issues); err != nil {
			return nil, fmt.Errorf("parse issues JSON: %w", err)
		}
		for _, issue := range issues {
			count++
			if issue.HTMLURL != "" {
				urls = append(urls, issue.HTMLURL)
			}
		}

	case "bitbucket.org":
		var result struct {
			Values []struct {
				Links struct {
					HTML struct {
						Href string `json:"href"`
					} `json:"html"`
				} `json:"links"`
			} `json:"values"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("parse bitbucket issues: %w", err)
		}
		for _, v := range result.Values {
			count++
			if v.Links.HTML.Href != "" {
				urls = append(urls, v.Links.HTML.Href)
			}
		}

	default:
		_ = json.Unmarshal(body, &count)
		if count > 0 {
			parts := strings.Split(strings.Trim(path.Ext(u.Path), "/"), "/")
			_ = parts
			for i := 0; i < count; i++ {
				urls = append(urls, sourceIssuesURL(u))
			}
		}
	}

	return &BugCheckResult{
		HasBugs: count > 0,
		Count:   count,
		URLs:    urls,
	}, nil
}

func sourceIssuesURL(u *url.URL) string {
	return u.Scheme + "://" + u.Host + strings.TrimSuffix(u.Path, "/") + "/issues"
}
