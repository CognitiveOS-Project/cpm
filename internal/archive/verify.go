package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/schema"
)

var validManagers = map[string]bool{
	"apk": true, "npm": true, "bun": true, "pip": true,
	"cargo": true, "go": true, "git": true,
}

var validStages = map[string]bool{
	"build": true, "boot": true, "install": true, "runtime": true,
}

func validateManifest(m *Manifest, fileExists func(string) bool) error {
	doc := map[string]interface{}{
		"name":        m.Name,
		"version":     m.Version,
		"description": m.Description,
	}
	if m.Author != "" {
		doc["author"] = m.Author
	}
	if m.License != "" {
		doc["license"] = m.License
	}
	if m.HardwareRequirements != nil {
		doc["hardware_requirements"] = m.HardwareRequirements
	}
	if m.HardwareDependencies != nil {
		doc["hardware_dependencies"] = m.HardwareDependencies
	}
	if m.Brain != nil {
		doc["brain"] = m.Brain
	}
	if m.Runtime != nil {
		doc["runtime"] = m.Runtime
	}
	if m.Dependencies != nil {
		doc["dependencies"] = m.Dependencies
	}

	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("ERROR:V006: schema violation: %w", err)
	}

	if m.HardwareRequirements != nil {
		if m.HardwareRequirements.MinRAMMB > 1048576 {
			return fmt.Errorf("ERROR:V010: hardware_requirements.min_ram_mb exceeds maximum (got %d, max 1048576)", m.HardwareRequirements.MinRAMMB)
		}
		if m.HardwareRequirements.MinStorageMB > 1073741824 {
			return fmt.Errorf("ERROR:V010: hardware_requirements.min_storage_mb exceeds maximum (got %d, max 1073741824)", m.HardwareRequirements.MinStorageMB)
		}
	}

	if m.HardwareDependencies != nil {
		for _, pkg := range m.HardwareDependencies.Packages {
			if pkg.Name == "" {
				return fmt.Errorf("ERROR:V011: hardware_dependencies.packages entry missing name")
			}
			if !validManagers[pkg.Manager] {
				return fmt.Errorf("ERROR:V011: invalid manager %q for package %s", pkg.Manager, pkg.Name)
			}
			if !validStages[pkg.Stage] {
				return fmt.Errorf("ERROR:V011: invalid stage %q for package %s", pkg.Stage, pkg.Name)
			}
		}
	}

	if m.Runtime != nil {
		if m.Runtime.SystemPrompt != "" {
			if !fileExists(m.Runtime.SystemPrompt) {
				return fmt.Errorf("ERROR:V007: missing file: %s", m.Runtime.SystemPrompt)
			}
		}
		for _, srv := range m.Runtime.MCPServers {
			cmdPath := srv.Command
			if !filepath.IsAbs(cmdPath) {
				cmdPath = filepath.Join("tools", cmdPath)
			}
			if !fileExists(cmdPath) {
				return fmt.Errorf("ERROR:V008: missing MCP server binary: %s", srv.Command)
			}
		}
	}

	if m.Brain != nil {
		if m.Brain.Adapter != "" {
			if !fileExists(m.Brain.Adapter) {
				return fmt.Errorf("ERROR:V009: missing adapter file: %s", m.Brain.Adapter)
			}
		}
	}

	return nil
}

func checkExecutables(dir string) error {
	toolsDir := filepath.Join(dir, "tools")
	info, err := os.Stat(toolsDir)
	if err != nil {
		return nil
	}
	if !info.IsDir() {
		return nil
	}

	return filepath.Walk(toolsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("ERROR:V012: open %s: %w", path, err)
		}
		defer f.Close()

		buf := make([]byte, 4)
		n, err := f.Read(buf)
		if err != nil || n < 2 {
			return fmt.Errorf("ERROR:V012: %s: not a valid executable (too small or unreadable)", path)
		}

		if string(buf[:2]) == "#!" {
			return nil
		}
		if n >= 4 && buf[0] == 0x7f && buf[1] == 'E' && buf[2] == 'L' && buf[3] == 'F' {
			return nil
		}

		return fmt.Errorf("ERROR:V012: %s: not a valid executable (no shebang or ELF header)", path)
	})
}

func Verify(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("ERROR:V001: open: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("ERROR:V002: invalid gzip: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	foundManifest := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ERROR:V003: invalid tar: %w", err)
		}
		if filepath.Clean(hdr.Name) == "cognitive.json" {
			foundManifest = true
			break
		}
	}
	if !foundManifest {
		return fmt.Errorf("ERROR:V005: cognitive.json not found in archive")
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("ERROR:V001: seek: %w", err)
	}
	gzr2, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("ERROR:V002: invalid gzip: %w", err)
	}
	defer gzr2.Close()

	tmpDir, err := os.MkdirTemp("", "cpm-verify-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tr2 := tar.NewReader(gzr2)
	for {
		hdr, err := tr2.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("ERROR:V003: invalid tar: %w", err)
		}

		name := filepath.Clean(hdr.Name)
		if !filepath.IsLocal(name) {
			continue
		}
		destPath := filepath.Join(tmpDir, name)
		_ = os.MkdirAll(filepath.Dir(destPath), 0755)

		switch hdr.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(destPath, os.FileMode(hdr.Mode))
		case tar.TypeReg:
			out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", name, err)
			}
			_, _ = io.Copy(out, tr2)
			out.Close()
		}
	}

	m, err := LoadManifest(filepath.Join(tmpDir, "cognitive.json"))
	if err != nil {
		return fmt.Errorf("ERROR:V004: %w", err)
	}

	fileExists := func(p string) bool {
		_, err := os.Stat(filepath.Join(tmpDir, p))
		return err == nil
	}

	if err := validateManifest(m, fileExists); err != nil {
		return err
	}

	if err := checkExecutables(tmpDir); err != nil {
		return err
	}

	fmt.Printf("✓ %s is valid (%s v%s)\n", filepath.Base(path), m.Name, m.Version)
	return nil
}

func VerifyExtracted(dir string) error {
	m, err := LoadManifest(filepath.Join(dir, "cognitive.json"))
	if err != nil {
		return fmt.Errorf("ERROR:V005: %w", err)
	}

	fileExists := func(p string) bool {
		_, err := os.Stat(filepath.Join(dir, p))
		return err == nil
	}

	if err := validateManifest(m, fileExists); err != nil {
		return err
	}

	if err := checkExecutables(dir); err != nil {
		return err
	}

	return nil
}
