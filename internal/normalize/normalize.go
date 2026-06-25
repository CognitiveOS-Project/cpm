package normalize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

type Result struct {
	Manifest *archive.Manifest
	DataDir  string
}

func Archive(path string) (*Result, error) {
	dir, err := os.MkdirTemp("", "cpm-normalize-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	if err := archive.Extract(f, dir); err != nil {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("extract: %w", err)
	}

	m, err := detectManifest(dir)
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}

	return &Result{Manifest: m, DataDir: dir}, nil
}

func detectManifest(dir string) (*archive.Manifest, error) {
	cogPath := filepath.Join(dir, "cognitive.json")
	if data, err := os.ReadFile(cogPath); err == nil {
		var m archive.Manifest
		if err := json.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("parse cognitive.json: %w", err)
		}
		return &m, nil
	}

	pkgPath := filepath.Join(dir, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		return parsePackageJSON(data)
	}

	return nil, fmt.Errorf("no manifest found: expected cognitive.json or package.json")
}

type packageJSON struct {
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description,omitempty"`
	Author          string                 `json:"author,omitempty"`
	License         string                 `json:"license,omitempty"`
	CognitiveOS     map[string]interface{} `json:"cognitive_os"`
}

func parsePackageJSON(data []byte) (*archive.Manifest, error) {
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}

	if pkg.Name == "" || pkg.Version == "" {
		return nil, fmt.Errorf("package.json missing required fields: name, version")
	}

	m := &archive.Manifest{
		Name:        pkg.Name,
		Version:     pkg.Version,
		Description: pkg.Description,
		Author:      pkg.Author,
		License:     pkg.License,
	}

	if pkg.CognitiveOS != nil {
		if v, ok := pkg.CognitiveOS["runtime"]; ok {
			data, _ := json.Marshal(v)
			var rt archive.RuntimeConfig
			if err := json.Unmarshal(data, &rt); err == nil {
				m.Runtime = &rt
			}
		}
		if v, ok := pkg.CognitiveOS["hardware_requirements"]; ok {
			data, _ := json.Marshal(v)
			var hr archive.HardwareReq
			if err := json.Unmarshal(data, &hr); err == nil {
				m.HardwareRequirements = &hr
			}
		}
		if v, ok := pkg.CognitiveOS["source"]; ok {
			data, _ := json.Marshal(v)
			var src archive.SourceInfo
			if err := json.Unmarshal(data, &src); err == nil {
				m.Source = &src
			}
		}
		if v, ok := pkg.CognitiveOS["dependencies"]; ok {
			switch d := v.(type) {
			case map[string]interface{}:
				deps := make(map[string]string)
				for k, val := range d {
					if s, ok := val.(string); ok {
						deps[k] = s
					}
				}
				m.Dependencies = deps
			}
		}
	}

	return m, nil
}
