package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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
	Name            string   `json:"name"`
	Version         string   `json:"version"`
	Description     string   `json:"description"`
	ChecksumSHA256  string   `json:"checksum_sha256"`
	DownloadURL     string   `json:"download_url"`
	Status          string   `json:"status"`
}

type VersionInfo struct {
	Version string `json:"version"`
	Status  string `json:"status"`
}

type DependencyTree struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies []DependencyTree  `json:"dependencies"`
}

type SearchOptions struct {
	License    string
	MinRAM     int
	Page       int
	PerPage    int
	Capability string
	Exact      bool
}

type Registry interface {
	Search(query string, opts SearchOptions) (*SearchResult, error)
	GetMetadata(name, version string) (*PatchMetadata, error)
	GetVersions(name string) ([]VersionInfo, error)
	GetDependencies(name string) (*DependencyTree, error)
	Unlock(name, version, code string) error
	Download(name, version string, opts DownloadOptions) (io.ReadCloser, error)
	Publish(token string, req PublishRequest) error
	PublishSSH(fingerprint, signature string, req PublishRequest) error
	PublishOfficial(fingerprint, signature string, req PublishRequest, metadataJSON, cgpData []byte) error
	RegisterPublicKey(publicKey string) (*RegisterResponse, error)
	CheckAuthStatus(fingerprint string) (*AuthStatusResponse, error)
	Signup(req SignupRequest) (*SignupResponse, error)
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

func (c *Client) Search(query string, opts SearchOptions) (*SearchResult, error) {
	u, _ := url.Parse(c.BaseURL + "/search")
	q := u.Query()
	q.Set("q", query)
	if opts.Page < 1 {
		opts.Page = 1
	}
	q.Set("page", fmt.Sprintf("%d", opts.Page))
	if opts.PerPage < 1 {
		opts.PerPage = 20
	}
	q.Set("per_page", fmt.Sprintf("%d", opts.PerPage))
	if opts.License != "" {
		q.Set("license", opts.License)
	}
	if opts.MinRAM > 0 {
		q.Set("min_ram_mb", fmt.Sprintf("%d", opts.MinRAM))
	}
	if opts.Capability != "" {
		q.Set("capability", opts.Capability)
	}
	if opts.Exact {
		q.Set("exact", "true")
	}
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

func (c *Client) GetVersions(name string) ([]VersionInfo, error) {
	u := c.BaseURL + "/patches/" + name + "/versions"

	resp, err := c.HTTP.Get(u)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(body))
	}

	var versions []VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return versions, nil
}

func (c *Client) GetDependencies(name string) (*DependencyTree, error) {
	u := c.BaseURL + "/patches/" + name + "/dependencies"

	resp, err := c.HTTP.Get(u)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(body))
	}

	var dt DependencyTree
	if err := json.NewDecoder(resp.Body).Decode(&dt); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &dt, nil
}

func (c *Client) Unlock(name, version, code string) error {
	u := c.BaseURL + "/patches/" + name + "/" + version + "/unlock"

	body := map[string]string{"code": code}
	data, _ := json.Marshal(body)

	resp, err := c.HTTP.Post(u, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unlock: %s", string(respBody))
	}
	return nil
}

type DownloadOptions struct {
	OS   string
	Arch string
}

func (c *Client) Download(name, version string, opts DownloadOptions) (io.ReadCloser, error) {
	u, _ := url.Parse(c.BaseURL + "/patches/" + name + "/" + version + "/download")
	q := u.Query()
	if opts.OS != "" {
		q.Set("os", opts.OS)
	}
	if opts.Arch != "" {
		q.Set("arch", opts.Arch)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "cpm/1.0")
	
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
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

type PublishRequest struct {
	Name             string          `json:"name"`
	Version          string          `json:"version"`
	Description      string          `json:"description"`
	Author           string          `json:"author,omitempty"`
	SourceRepository string          `json:"source_repository,omitempty"`
	SourceIssues     string          `json:"source_issues,omitempty"`
	DownloadURL      string          `json:"download_url,omitempty"`
	SHA256           string          `json:"sha256,omitempty"`
	Tags             []string        `json:"tags,omitempty"`
	Scope            string          `json:"scope,omitempty"`
	Visibility       string          `json:"visibility,omitempty"`
	Manifest         json.RawMessage `json:"manifest,omitempty"`
}

func (c *Client) Publish(token string, req PublishRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/patches", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry: %s", string(respBody))
	}
	return nil
}

type RegisterResponse struct {
	Fingerprint  string `json:"fingerprint"`
	KeyType      string `json:"public_key_type"`
	Comment      string `json:"comment,omitempty"`
	RegisteredAt string `json:"registered_at"`
}

type AuthStatusResponse struct {
	Fingerprint  string `json:"fingerprint"`
	Registered   bool   `json:"registered"`
	RegisteredAt string `json:"registered_at,omitempty"`
}

type SignupRequest struct {
	Profile   json.RawMessage `json:"profile"`
	PublicKey string          `json:"public_key"`
	Signature string          `json:"signature"`
}

type SignupResponse struct {
	MachineID string `json:"machine_id"`
	Status    string `json:"status"`
}

func (c *Client) PublishSSH(fingerprint, signature string, req PublishRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/patches", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-SSH-Fingerprint", fingerprint)
	httpReq.Header.Set("X-SSH-Signature", signature)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry: %s", string(respBody))
	}
	return nil
}

func (c *Client) PublishOfficial(fingerprint, signature string, req PublishRequest, metadataJSON, cgpData []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	mPart, err := mw.CreateFormField("metadata")
	if err != nil {
		return fmt.Errorf("create metadata part: %w", err)
	}
	if _, err := mPart.Write(metadataJSON); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	cPart, err := mw.CreateFormFile("cgp", "package.cgp")
	if err != nil {
		return fmt.Errorf("create cgp part: %w", err)
	}
	if _, err := cPart.Write(cgpData); err != nil {
		return fmt.Errorf("write cgp: %w", err)
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("close multipart: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/patches", &buf)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", mw.FormDataContentType())
	httpReq.Header.Set("X-SSH-Fingerprint", fingerprint)
	httpReq.Header.Set("X-SSH-Signature", signature)

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry: %s", string(respBody))
	}
	return nil
}

func (c *Client) RegisterPublicKey(publicKey string) (*RegisterResponse, error) {
	payload := map[string]string{"public_key": publicKey}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/auth/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(respBody))
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &regResp, nil
}

func (c *Client) CheckAuthStatus(fingerprint string) (*AuthStatusResponse, error) {
	payload := map[string]string{"fingerprint": fingerprint}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	httpReq, err := http.NewRequest("PUT", c.BaseURL+"/auth/status", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(respBody))
	}

	var statusResp AuthStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &statusResp, nil
}

func (c *Client) Signup(req SignupRequest) (*SignupResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/auth/signup", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registry: %s", string(respBody))
	}

	var signupResp SignupResponse
	if err := json.NewDecoder(resp.Body).Decode(&signupResp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &signupResp, nil
}
