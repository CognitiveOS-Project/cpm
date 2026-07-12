package archive

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type SystemDependency struct {
	Name        string `json:"name"`
	Manager     string `json:"manager"`
	Stage       string `json:"stage"` // build, boot, install, runtime
	Required    bool   `json:"required"`
	Version     string `json:"version,omitempty"` // semver or "latest"
	Description string `json:"description,omitempty"`
}

type HardwareDependencies struct {
	Packages []SystemDependency `json:"packages,omitempty"`
}

type Manifest struct {
	Name                string             `json:"name"`
	Version             string             `json:"version"`
	Description         string             `json:"description"`
	Author              string             `json:"author,omitempty"`
	License             string             `json:"license,omitempty"`
	Source              *SourceInfo        `json:"source,omitempty"`
	Dependencies        map[string]string  `json:"dependencies,omitempty"`
	HardwareRequirements *HardwareReq      `json:"hardware_requirements,omitempty"`
	HardwareDependencies *HardwareDependencies `json:"hardware_dependencies,omitempty"`
	Brain               *BrainConfig       `json:"brain,omitempty"`
	Runtime             *RuntimeConfig     `json:"runtime,omitempty"`
	Training            *TrainingConfig    `json:"training,omitempty"`
	Checksum            *ChecksumInfo      `json:"checksum,omitempty"`
}

type ChecksumInfo struct {
	SHA256 string `json:"sha256,omitempty"`
}

type SourceInfo struct {
	Repository string `json:"repository"`
	Issues     string `json:"issues"`
	IssuesAPI  string `json:"issues_api,omitempty"`
}

type HardwareReq struct {
	OS             []string `json:"os,omitempty"`
	Arch           []string `json:"arch,omitempty"`
	MinRAMMB       int      `json:"min_ram_mb,omitempty"`
	MinStorageMB   int      `json:"min_storage_mb,omitempty"`
	NPURequired    bool     `json:"npu_required,omitempty"`
	RecommendedNPU string   `json:"recommended_npu,omitempty"`
}

type BrainConfig struct {
	BaseModel string        `json:"base_model,omitempty"`
	Adapter   string        `json:"adapter,omitempty"`
	RawModel  *ModelConfig  `json:"raw_model,omitempty"`
	WideModel *ModelConfig  `json:"wide_model,omitempty"`
}

type ModelConfig struct {
	BaseModel  string              `json:"base_model,omitempty"`
	Adapter    string              `json:"adapter,omitempty"`
	Weights    *WeightsConfig      `json:"weights,omitempty"`
	Parameters *BrainParameters    `json:"parameters,omitempty"`
	Routing    []RoutingHint       `json:"routing,omitempty"`
}

type WeightsConfig struct {
	Remote *RemoteWeights `json:"remote,omitempty"`
}

type RemoteWeights struct {
	Source   string `json:"source,omitempty"`
	ModelID  string `json:"model_id,omitempty"`
	URL      string `json:"url,omitempty"`
	Filename string `json:"filename,omitempty"`
	Format   string `json:"format,omitempty"`
	Quant    string `json:"quant,omitempty"`
	SHA256   string `json:"sha256,omitempty"`
}

type BrainParameters struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumCtx      int     `json:"num_ctx,omitempty"`
}

type RoutingHint struct {
	Capability string `json:"capability,omitempty"`
	Priority   int    `json:"priority,omitempty"`
}

type RuntimeConfig struct {
	SystemPrompt string       `json:"system_prompt,omitempty"`
	ToolsRoot    string       `json:"tools_root,omitempty"`
	MCPServers   []MCPServer  `json:"mcp_servers,omitempty"`
	Background   bool         `json:"background,omitempty"`
	Capabilities []string     `json:"capabilities,omitempty"`
}

type TrainingConfig struct {
	Tool             string            `json:"tool,omitempty"`
	Hyperparameters  *TrainingParams   `json:"hyperparameters,omitempty"`
	DataRequirements *TrainingDataReqs `json:"data_requirements,omitempty"`
	OutputPath       string            `json:"output_path,omitempty"`
}

type TrainingParams struct {
	Rank         int     `json:"rank,omitempty"`
	Alpha        float64 `json:"alpha,omitempty"`
	Epochs       int     `json:"epochs,omitempty"`
	LearningRate float64 `json:"learning_rate,omitempty"`
}

type TrainingDataReqs struct {
	MinSamples int `json:"min_samples,omitempty"`
}

type MCPServer struct {
	Name      string            `json:"name"`
	Command   string            `json:"command"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Transport string            `json:"transport"`
}

func ReadManifest(r io.Reader) (*Manifest, error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gzr.Close()
	
	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if filepath.Clean(hdr.Name) == "cognitive.json" {
			var m Manifest
			if err := json.NewDecoder(tr).Decode(&m); err != nil {
				return nil, fmt.Errorf("parse cognitive.json: %w", err)
			}
			return &m, nil
		}
	}
	return nil, fmt.Errorf("cognitive.json not found in archive")
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", path, err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal manifest %s: %w", path, err)
	}
	return &m, nil
}

func Extract(r io.Reader, dest string) error {

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}

		name := filepath.Clean(hdr.Name)
		if !filepath.IsLocal(name) {
			continue
		}
		destPath := filepath.Join(dest, name)
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)

		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(destPath, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			_, _ = io.Copy(f, tr)
			f.Close()
		}
	}
	return nil
}
