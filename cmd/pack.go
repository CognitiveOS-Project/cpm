package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
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
	Use:   "pack [--bin <path>] [--manifest <path>]",
	Short: "Package a binary or manifest into a .cgp archive",
	Long: `Create a .cgp (Cognitive Patch) archive from a compiled binary and/or a cognitive.json manifest.
If --bin is provided, binaries are copied to tools/ in the archive.
If --bin is omitted, only the manifest (and any files it referenced) is packaged.
Manifest is resolved via --manifest flag or auto-detected from CWD/parent/bin directories.
After packing, the archive is automatically verified for integrity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tmpDir, err := os.MkdirTemp("", "cpm-pack-*")
		if err != nil {
			return fmt.Errorf("failed to create tmp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)

		if packBin != "" {
			toolsDir := filepath.Join(tmpDir, "tools")
			if err := os.MkdirAll(toolsDir, 0755); err != nil {
				return fmt.Errorf("failed to create tools dir: %w", err)
			}

			info, err := os.Stat(packBin)
			if err != nil {
				return fmt.Errorf("failed to stat bin: %w", err)
			}

			if info.IsDir() {
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
				destBin := filepath.Join(toolsDir, filepath.Base(packBin))
				if err := copyFile(packBin, destBin); err != nil {
					return fmt.Errorf("failed to copy binary: %w", err)
				}
				if err := os.Chmod(destBin, 0755); err != nil {
					return fmt.Errorf("failed to chmod binary: %w", err)
				}
			}
		}

		manifest, err := loadAndMergeManifests(packBin)
		if err != nil {
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

		typedManifest, err := archive.LoadManifest(filepath.Join(tmpDir, "cognitive.json"))
		if err != nil {
			return fmt.Errorf("failed to parse manifest: %w", err)
		}

		if typedManifest.Name == "" {
			return fmt.Errorf("manifest missing required field: name")
		}
		if typedManifest.Version == "" {
			return fmt.Errorf("manifest missing required field: version")
		}

		outputName := fmt.Sprintf("%s-%s", typedManifest.Name, typedManifest.Version)
		if typedManifest.HardwareRequirements != nil {
			osVal := typedManifest.HardwareRequirements.OS
			archVal := typedManifest.HardwareRequirements.Arch
			if len(osVal) > 0 && len(archVal) > 0 {
				outputName = fmt.Sprintf("%s-%s-%s-%s", typedManifest.Name, typedManifest.Version, osVal[0], archVal[0])
			}
		}
		if outputName == fmt.Sprintf("%s-%s", typedManifest.Name, typedManifest.Version) {
			if typedManifest.HardwareRequirements == nil {
				outputName += "-universal"
			}
		}
		outputFile := outputName + ".cgp"

		if err := copyManifestFiles(tmpDir, typedManifest); err != nil {
			return fmt.Errorf("failed to include manifest files: %w", err)
		}

		if err := createCgp(tmpDir, outputFile); err != nil {
			return fmt.Errorf("failed to create archive: %w", err)
		}

		if err := archive.Verify(outputFile); err != nil {
			_ = os.Remove(outputFile)
			return fmt.Errorf("verification failed: %w", err)
		}

		msg := fmt.Sprintf("✓ Packaged %s as %s", packBin, outputFile)
		if packBin == "" {
			msg = fmt.Sprintf("✓ Packaged manifest as %s", outputFile)
		}
		fmt.Println(msg)
		return nil
	},

}

func collectManifestRefs(m *archive.Manifest) []string {
	var refs []string

	if m.Runtime != nil {
		if m.Runtime.SystemPrompt != "" {
			refs = append(refs, m.Runtime.SystemPrompt)
		}
		for _, srv := range m.Runtime.MCPServers {
			if srv.Command != "" && !filepath.IsAbs(srv.Command) {
				refs = append(refs, filepath.Join("tools", srv.Command))
			}
		}
	}
	if m.Brain != nil {
		if m.Brain.Adapter != "" {
			refs = append(refs, m.Brain.Adapter)
		}
		if m.Brain.RawModel != nil && m.Brain.RawModel.Weights != nil && m.Brain.RawModel.Weights.Remote != nil {
			if f := m.Brain.RawModel.Weights.Remote.Filename; f != "" {
				refs = append(refs, filepath.Join("weights", f))
			}
		}
		if m.Brain.WideModel != nil && m.Brain.WideModel.Weights != nil && m.Brain.WideModel.Weights.Remote != nil {
			if f := m.Brain.WideModel.Weights.Remote.Filename; f != "" {
				refs = append(refs, filepath.Join("weights", f))
			}
		}
	}

	seen := map[string]bool{}
	var unique []string
	for _, r := range refs {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	return unique
}

func copyManifestFiles(tmpDir string, m *archive.Manifest) error {
	refs := collectManifestRefs(m)
	cwd, _ := os.Getwd()

	for _, ref := range refs {
		if filepath.IsAbs(ref) {
			continue
		}

		src := filepath.Join(cwd, ref)
		dst := filepath.Join(tmpDir, ref)

		if _, err := os.Stat(src); err != nil {
			fmt.Printf("  Warning: referenced file %s not found, skipping\n", ref)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", ref, err)
		}
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", ref, err)
		}
	}

	return nil
}

func loadAndMergeManifests(binPath string) (map[string]interface{}, error) {
	cwd, _ := os.Getwd()
	searchPaths := []string{
		filepath.Join(cwd, "cognitive.json"),
	}
	if binPath != "" {
		searchPaths = append(searchPaths, filepath.Join(filepath.Dir(binPath), "cognitive.json"))
		searchPaths = append(searchPaths, filepath.Join(binPath, "cognitive.json"))
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
	fs.StringVar(&packOS, "os", "", "Target OS")
	fs.StringVar(&packArch, "arch", "", "Target Architecture")
	fs.StringVar(&packDescription, "description", "", "Package description")
	fs.StringVar(&packManifest, "manifest", "", "Path to cognitive.json manifest file")
	rootCmd.AddCommand(packCmd)
}
