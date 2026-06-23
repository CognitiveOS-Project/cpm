package registry

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type SearchResult struct {
	Results []PatchSummary `json:"results"`
	Total   int            `json:"total"`
	Page    int            `json:"page"`
}

type PatchSummary struct {
	Name                string `json:"name"`
	Version             string `json:"version"`
	Description         string `json:"description"`
	License             string `json:"license"`
	ChecksumSHA256      string `json:"checksum_sha256"`
}

type PatchMetadata struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	ChecksumSHA256 string `json:"checksum_sha256"`
}

type Client struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Search(query string, page, perPage int) (*SearchResult, error) {
	u, _ := url.Parse(c.BaseURL + "/search")
	q := u.Query()
	q.Set("q", query)
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	u.RawQuery = q.Encode()

	resp, err := c.HTTP.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(body))
	}

	var sr SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &sr, nil
}

func (c *Client) GetMetadata(name, version string) (*PatchMetadata, error) {
	u := c.BaseURL + "/patches/" + name
	if version != "" {
		u += "/" + version
	}

	resp, err := c.HTTP.Get(u)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(body))
	}

	var pm PatchMetadata
	if err := json.NewDecoder(resp.Body).Decode(&pm); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &pm, nil
}

func (c *Client) Download(name, version string) (io.ReadCloser, error) {
	u := c.BaseURL + "/patches/" + name + "/" + version + "/download"
	resp, err := c.HTTP.Get(u)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(body))
	}
	return resp.Body, nil
}
