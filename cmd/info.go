package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/patch"
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
			return fmt.Errorf("patch %q not found: %w", name, err)
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

		if m.Runtime != nil {
			for _, s := range m.Runtime.MCPServers {
				fmt.Printf("MCP servers:    %s (%s)\n", s.Name, s.Transport)
			}
			if m.Runtime.Background {
				fmt.Println("Background:     yes")
			}
		}

		installedPath := filepath.Join(patch.PatchesDir, name)
		if _, err := os.Stat(installedPath); err == nil {
			fmt.Printf("Installed:      %s\n", installedPath)
			fmt.Printf("Size:           %d MB\n", dirSize(installedPath)/(1024*1024))
		}
		return nil
	},
}

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
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
