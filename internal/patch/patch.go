package patch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

var PatchesDir = "/cognitiveos/patches"

func init() {
	if d := os.Getenv("CPM_PATCHES_DIR"); d != "" {
		PatchesDir = d
	}
}

type Installed struct {
	Manifest *archive.Manifest
	Path     string
}

func List() ([]Installed, error) {
	entries, err := os.ReadDir(PatchesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read patches dir: %w", err)
	}

	var patches []Installed
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		m, err := ReadManifest(e.Name())
		if err != nil {
			continue
		}
		patches = append(patches, Installed{
			Manifest: m,
			Path:     filepath.Join(PatchesDir, e.Name()),
		})
	}
	return patches, nil
}

func ReadManifest(name string) (*archive.Manifest, error) {
	path := filepath.Join(PatchesDir, name, "cognitive.json")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var m archive.Manifest
	if err := json.NewDecoder(f).Decode(&m); err != nil {
		return nil, fmt.Errorf("parse %s/cognitive.json: %w", name, err)
	}
	return &m, nil
}

func IsInstalled(name string) bool {
	_, err := os.Stat(filepath.Join(PatchesDir, name))
	return err == nil
}

func Dir(name string) string {
	return filepath.Join(PatchesDir, name)
}

func Remove(name string) error {
	return os.RemoveAll(filepath.Join(PatchesDir, name))
}
