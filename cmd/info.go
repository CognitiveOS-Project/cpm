package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show patch details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
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
	},
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
	rootCmd.AddCommand(infoCmd)
}
