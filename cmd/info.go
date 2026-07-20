package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var (
	infoJSON     bool
	infoManifest string
)

var infoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show patch details",
	Long: `Show details of an installed patch by name.

With --json, reads a local cognitive.json manifest and outputs machine-parseable
JSON including a computed filename field (matching cpm pack output naming).

Examples:
  cpm info my-skill
  cpm info --json
  cpm info --json --manifest path/to/cognitive.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if infoJSON {
			return infoJSONOutput()
		}
		return infoPlainText(args[0])
	},
}

func infoJSONOutput() error {
	path := infoManifest
	if path == "" {
		path = "cognitive.json"
	}

	m, err := archive.LoadManifest(path)
	if err != nil {
		return fmt.Errorf("ERROR:INFO002: load manifest: %w", err)
	}

	output := map[string]interface{}{
		"name":     m.Name,
		"version":  m.Version,
		"filename": computeFilename(m),
	}
	if m.Description != "" {
		output["description"] = m.Description
	}
	if m.Author != "" {
		output["author"] = m.Author
	}
	if m.License != "" {
		output["license"] = m.License
	}
	if m.Source != nil {
		output["source"] = m.Source
	}
	if m.HardwareRequirements != nil {
		output["hardware_requirements"] = m.HardwareRequirements
	}
	if m.Runtime != nil {
		output["runtime"] = m.Runtime
	}
	if m.Brain != nil {
		output["brain"] = m.Brain
	}
	if m.Checksum != nil && m.Checksum.SHA256 != "" {
		output["checksum"] = m.Checksum
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func computeFilename(m *archive.Manifest) string {
	name := m.Name
	ver := m.Version
	if m.HardwareRequirements != nil {
		osVal := m.HardwareRequirements.OS
		archVal := m.HardwareRequirements.Arch
		if len(osVal) > 0 && len(archVal) > 0 {
			return fmt.Sprintf("%s-%s-%s-%s.cgp", name, ver, osVal[0], archVal[0])
		}
	}
	if m.HardwareRequirements == nil {
		return fmt.Sprintf("%s-%s-universal.cgp", name, ver)
	}
	return fmt.Sprintf("%s-%s.cgp", name, ver)
}

func infoPlainText(name string) error {
	m, err := patch.ReadManifest(name)
	if err != nil {
		return fmt.Errorf("ERROR:INFO001: patch %q not found: %w", name, err)
	}

	fmt.Printf("Name:           %s\n", m.Name)
	fmt.Printf("Version:        %s\n", m.Version)
	fmt.Printf("Author:         %s\n", m.Author)
	fmt.Printf("License:        %s\n", m.License)
	fmt.Printf("Description:    %s\n", m.Description)

	if m.HardwareRequirements != nil {
		hw := m.HardwareRequirements
		fmt.Printf("Hardware:       %d MB RAM, %d MB storage", hw.MinRAMMB, hw.MinStorageMB)
		if hw.NPURequired {
			fmt.Print(", NPU required")
		}
		fmt.Println()
	}

	if len(m.Dependencies) > 0 {
		fmt.Println("Dependencies:")
		for d, v := range m.Dependencies {
			fmt.Printf("  - %s (%s)\n", d, v)
		}
	} else {
		fmt.Println("Dependencies:   (none)")
	}

	if m.Source != nil {
		if m.Source.Repository != "" {
			fmt.Printf("Repository:     %s\n", m.Source.Repository)
		}
		if m.Source.Issues != "" {
			fmt.Printf("Issues:         %s\n", m.Source.Issues)
		}
	}

	if m.Runtime != nil {
		for _, s := range m.Runtime.MCPServers {
			fmt.Printf("MCP servers:    %s (%s)\n", s.Name, s.Transport)
		}
		if m.Runtime.Background {
			fmt.Println("Background:     yes")
		}
		if len(m.Runtime.Capabilities) > 0 {
			fmt.Printf("Capabilities:   %v\n", m.Runtime.Capabilities)
		}
	}

	if m.Checksum != nil && m.Checksum.SHA256 != "" {
		fmt.Printf("SHA-256:        %s\n", m.Checksum.SHA256)
	}

	installedPath := filepath.Join(patch.PatchesDir, name)
	if _, err := os.Stat(installedPath); err == nil {
		fmt.Printf("Installed:      %s\n", installedPath)
		fmt.Printf("Size:           %d MB\n", dirSize(installedPath)/(1024*1024))
	}

	regURL := resolveRegistry()
	if regURL != "" {
		rc := registry.New(regURL)
		meta, err := rc.GetMetadata(name, "")
		if err == nil {
			fmt.Printf("Registry:       %s\n", regURL)
			fmt.Printf("Latest:         %s\n", meta.Version)
			if meta.Status != "" {
				fmt.Printf("Status:         %s\n", meta.Status)
			}
		}
	}

	return nil
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() {
			size += fi.Size()
		}
		return nil
	})
	return size
}

func init() {
	infoCmd.Flags().BoolVar(&infoJSON, "json", false, "Output JSON (reads from --manifest or CWD)")
	infoCmd.Flags().StringVar(&infoManifest, "manifest", "", "Path to cognitive.json (default: CWD/cognitive.json)")
	rootCmd.AddCommand(infoCmd)
}
