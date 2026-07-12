package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	packBin         string
	packName        string
	packVersion     string
	packOS          string
	packArch        string
	packDescription string
	packManifest    string
)

var packCmd = &cobra.Command{
	Use:   "pack --bin <path>",
	Short: "Package a binary into a .cgp archive",
	Long: `Create a .cgp (Cognitive Patch) archive from a compiled binary.
This tool allows using a cognitive.json manifest file for detailed configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if packBin == "" {
			return fmt.Errorf("ERROR:P101: --bin is required")
		}

		// 1. Prepare Temporary Directory
		tmpDir, err := os.MkdirTemp("", "cpm-pack-*")
		if err != nil {
			return fmt.Errorf("failed to create tmp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		// 2. Create Directory Structure
		toolsDir := filepath.Join(tmpDir, "tools")
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			return fmt.Errorf("failed to create tools dir: %w", err)
		}

		// 3. Copy Binary/Binaries
		info, err := os.Stat(packBin)
		if err != nil {
			return fmt.Errorf("failed to stat bin: %w", err)
		}

		if info.IsDir() {
			// Package all binaries in the directory
			entries, err := os.ReadDir(packBin)
			if err != nil {
				return fmt.Errorf("failed to read bin dir: %w", err)
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				src := filepath.Join(packBin, entry.Name())
				dst := filepath.Join(toolsDir, entry.Name())
				if err := copyFile(src, dst); err != nil {
					return fmt.Errorf("failed to copy binary %s: %w", entry.Name(), err)
				}
				if err := os.Chmod(dst, 0755); err != nil {
					return fmt.Errorf("failed to chmod binary %s: %w", entry.Name(), err)
				}
			}
		} else {
			// Package a single binary
			destBin := filepath.Join(toolsDir, filepath.Base(packBin))
			if err := copyFile(packBin, destBin); err != nil {
				return fmt.Errorf("failed to copy binary: %w", err)
			}
			if err := os.Chmod(destBin, 0755); err != nil {
				return fmt.Errorf("failed to chmod binary: %w", err)
			}
		}

		// 4. Resolve and Merge Manifest
		manifest, err := loadAndMergeManifests(packBin)
		if err != nil {
			// Backward compatibility: if no manifest found, try to generate one from flags
			if packName == "" || packVersion == "" {
				return fmt.Errorf("ERROR:P102: no manifest found and --name/--version are required for minimal manifest")
			}

			hwReqs := make(map[string]interface{})
			if packOS != "" {
				hwReqs["os"] = []string{packOS}
			}
			if packArch != "" {
				hwReqs["arch"] = []string{packArch}
			}

			manifest = map[string]interface{}{
				"name":        packName,
				"version":     packVersion,
				"description": packDescription,
			}
			if len(hwReqs) > 0 {
				manifest["hardware_requirements"] = hwReqs
			}
		}

		manifestData, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal manifest: %w", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "cognitive.json"), manifestData, 0644); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}

		// 5. Create .cgp Archive
		name, ok := manifest["name"].(string)
		if !ok {
			return fmt.Errorf("manifest missing required field: name")
		}
		version, ok := manifest["version"].(string)
		if !ok {
			return fmt.Errorf("manifest missing required field: version")
		}

		outputName := fmt.Sprintf("%s-%s", name, version)
		hwReqs, ok := manifest["hardware_requirements"].(map[string]interface{})
		if ok {
			osVal, okOS := hwReqs["os"].([]string)
			archVal, okArch := hwReqs["arch"].([]string)
			if okOS && len(osVal) > 0 && okArch && len(archVal) > 0 {
				outputName = fmt.Sprintf("%s-%s-%s-%s", name, version, osVal[0], archVal[0])
			}
		}
		if outputName == fmt.Sprintf("%s-%s", name, version) {
			// If not specific os/arch, check if we should use -universal
			// For simplicity, if we didn't match os/arch pattern, we use universal if no constraints
			if hwReqs == nil {
				outputName += "-universal"
			}
		}
		outputFile := outputName + ".cgp"

		if err := createCgp(tmpDir, outputFile); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}

		fmt.Printf("✓ Packaged %s as %s\n", packBin, outputFile)
		return nil
	},
}

func loadAndMergeManifests(binPath string) (map[string]interface{}, error) {
	cwd, _ := os.Getwd()
	searchPaths := []string{
		filepath.Join(cwd, "cognitive.json"),
		filepath.Join(filepath.Dir(binPath), "cognitive.json"),
		filepath.Join(binPath, "cognitive.json"),
	}

	merged := make(map[string]interface{})
	found := false

	for _, path := range searchPaths {
		if data, err := os.ReadFile(path); err == nil {
			var m map[string]interface{}
			if err := json.Unmarshal(data, &m); err == nil {
				for k, v := range m {
					merged[k] = v
				}
				found = true
			}
		}
	}

	if packManifest != "" {
		if data, err := os.ReadFile(packManifest); err == nil {
			var m map[string]interface{}
			if err := json.Unmarshal(data, &m); err == nil {
				for k, v := range m {
					merged[k] = v
				}
				found = true
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("no manifest found")
	}

	return merged, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func createCgp(src, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return err
			}
		}
		return nil
	})
}

func init() {
	fs := packCmd.Flags()
	fs.StringVar(&packBin, "bin", "", "Path to the binary to package")
	fs.StringVar(&packName, "name", "", "Package name")
	fs.StringVar(&packVersion, "version", "", "Package version")
	fs.StringVar(&packOS, "os", "linux", "Target OS")
	fs.StringVar(&packArch, "arch", "amd64", "Target Architecture")
	fs.StringVar(&packDescription, "description", "", "Package description")
	fs.StringVar(&packManifest, "manifest", "", "Path to cognitive.json manifest file")
	rootCmd.AddCommand(packCmd)
}
