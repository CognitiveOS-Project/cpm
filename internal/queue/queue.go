package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

// QueueEntry represents a parsed queue file.
type QueueEntry struct {
	Filename    string                     `json:"-"`
	PatchName   string                     `json:"patch_name"`
	PatchVersion string                   `json:"patch_version"`
	Dependency  archive.SystemDependency `json:"dependency"`
	RegisteredAt time.Time                `json:"registered_at"`
}

// QueueDir returns the queue root directory.
func QueueDir(root string) string {
	return filepath.Join(root, "cognitiveos", "lib", "cpm", "queue")
}

// Register writes a queue file for one dependency.
func Register(root, patchName, patchVersion string, dep archive.SystemDependency) error {
	stageDir := filepath.Join(QueueDir(root), dep.Stage)
	if err := os.MkdirAll(stageDir, 0755); err != nil {
		return fmt.Errorf("create stage dir %s: %w", stageDir, err)
	}

	// Sanitize name and version for filename
	sPatchName := sanitize(patchName)
	sPatchVersion := sanitize(patchVersion)
	sDepName := sanitize(dep.Name)
	
	version := dep.Version
	if version == "" {
		version = "latest"
	}
	sDepVersion := sanitize(version)

	filename := fmt.Sprintf("%s-%s_%s-%s.json", sPatchName, sPatchVersion, sDepName, sDepVersion)
	filePath := filepath.Join(stageDir, filename)

	entry := QueueEntry{
		PatchName:    patchName,
		PatchVersion: patchVersion,
		Dependency:   dep,
		RegisteredAt: time.Now(),
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write queue file %s: %w", filePath, err)
	}

	return nil
}

// ListByStage returns all queue entries for a given stage.
func ListByStage(root, stage string) ([]QueueEntry, error) {
	stageDir := filepath.Join(QueueDir(root), stage)
	files, err := os.ReadDir(stageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read stage dir %s: %w", stageDir, err)
	}

	var entries []QueueEntry
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(stageDir, f.Name()))
		if err != nil {
			continue // skip corrupt files
		}

		var entry QueueEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		entry.Filename = f.Name()
		entries = append(entries, entry)
	}

	return entries, nil
}

// MarkInstalled removes a queue file after successful installation.
func MarkInstalled(root, stage, filename string) error {
	filePath := filepath.Join(QueueDir(root), stage, filename)
	if err := os.Remove(filePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("remove queue file %s: %w", filePath, err)
		}
	}
	return nil
}

// RemoveByPatch removes all registration records for a specific patch.
func RemoveByPatch(root, patchName, patchVersion string) error {
	stages := []string{"build", "boot", "install", "runtime"}
	for _, stage := range stages {
		entries, err := ListByStage(root, stage)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.PatchName == patchName && entry.PatchVersion == patchVersion {
				_ = MarkInstalled(root, stage, entry.Filename)
			}
		}
	}
	return nil
}

func sanitize(s string) string {
	// Replace characters that are invalid or problematic in filenames
	// This is a simple replacement; in a real system, we might use a more robust sanitization
	r := strings.NewReplacer("/", "-", ".", "_", " ", "_", ":", "-")
	return r.Replace(s)
}
