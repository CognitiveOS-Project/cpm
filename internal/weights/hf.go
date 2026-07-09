package weights

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type hfModel struct {
	ID       string      `json:"id"`
	ModelID  string      `json:"modelId"`
	Siblings []hfSibling `json:"siblings"`
}

type hfSibling struct {
	RFilename string `json:"rfilename"`
	Size      int64  `json:"size"`
}

type HFProvider struct {
	client *http.Client
}

func NewHFProvider() *HFProvider {
	return &HFProvider{client: http.DefaultClient}
}

func (p *HFProvider) Name() string { return "hf" }

func (p *HFProvider) Search(ctx context.Context, query string, limit int, format Format) ([]Candidate, error) {
	libraryFilter := "gguf"
	if format == FormatSafeTensors {
		libraryFilter = "safetensors"
	}
	apiURL := fmt.Sprintf(
		"https://huggingface.co/api/models?library=%s&search=%s&sort=downloads&direction=-1&limit=%d&full=true",
		libraryFilter,
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("hf: creating request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hf: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hf: %s — %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var models []hfModel
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("hf: decoding response: %w", err)
	}

	var candidates []Candidate

	for _, model := range models {
		modelID := model.ModelID
		if modelID == "" {
			modelID = model.ID
		}

		for _, file := range model.Siblings {
			name := strings.ToLower(file.RFilename)
			if !strings.HasSuffix(name, ".gguf") && !strings.HasSuffix(name, ".safetensors") {
				continue
			}

			candidates = append(candidates, Candidate{
				ModelID:     modelID,
				Filename:    file.RFilename,
				DownloadURL: fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", modelID, file.RFilename),
				SizeBytes:   file.Size,
			})
		}
	}

	return candidates, nil
}
