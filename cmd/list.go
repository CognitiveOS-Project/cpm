package cmd

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed patches",
	RunE: func(cmd *cobra.Command, args []string) error {
		patches, err := patch.List()
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}
		if len(patches) == 0 {
			fmt.Println("No patches installed")
			return nil
		}
		for _, p := range patches {
			if verbose {
				fmt.Printf("%-20s %-8s %s\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Description)
				fmt.Printf("  path: %s\n", p.Path)
				if p.Manifest.Runtime != nil {
					fmt.Printf("  tools: %d MCP servers\n", len(p.Manifest.Runtime.MCPServers))
				}
				if p.Manifest.HardwareRequirements != nil {
					fmt.Printf("  memory: %d MB\n", p.Manifest.HardwareRequirements.MinRAMMB)
				}
			} else {
				fmt.Printf("%-20s %-8s %s\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Description)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
