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

type Manifest struct {
	Name                string             `json:"name"`
	Version             string             `json:"version"`
	Description         string             `json:"description"`
	Author              string             `json:"author,omitempty"`
	License             string             `json:"license,omitempty"`
	Dependencies        map[string]string  `json:"dependencies,omitempty"`
	HardwareRequirements *HardwareReq      `json:"hardware_requirements,omitempty"`
	Brain               *BrainConfig       `json:"brain,omitempty"`
	Runtime             *RuntimeConfig     `json:"runtime,omitempty"`
}

type HardwareReq struct {
	MinRAMMB     int    `json:"min_ram_mb,omitempty"`
	MinStorageMB int    `json:"min_storage_mb,omitempty"`
	NPURequired  bool   `json:"npu_required,omitempty"`
}

type BrainConfig struct {
	BaseModel string `json:"base_model,omitempty"`
	Adapter   string `json:"adapter,omitempty"`
}

type RuntimeConfig struct {
	SystemPrompt string       `json:"system_prompt,omitempty"`
	ToolsRoot    string       `json:"tools_root,omitempty"`
	MCPServers   []MCPServer  `json:"mcp_servers,omitempty"`
	Background   bool         `json:"background,omitempty"`
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
