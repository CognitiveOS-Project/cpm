package cmd

import (
	"fmt"
	"io"

	"github.com/CognitiveOS-Project/cpm/internal/dep"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed patch",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !patch.IsInstalled(name) {
			return fmt.Errorf("ERROR:RM001: patch %q is not installed", name)
		}

		deps := dep.CheckDependents(name)
		if len(deps) > 0 {
			return fmt.Errorf("ERROR:RM002: cannot remove %q: %v depends on it", name, deps)
		}

		if !yesMode {
			fmt.Printf("Remove %s? [y/N]: ", name)
			var confirm string
			if _, err := fmt.Scanln(&confirm); err != nil && err != io.EOF {
				fmt.Printf("Error reading confirmation: %v\n", err)
				return nil
			}
			if confirm != "y" && confirm != "Y" && confirm != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		if err := patch.Remove(name); err != nil {
			return fmt.Errorf("ERROR:RM003: remove: %w", err)
		}
		log.Info("Removed %s", name)
		fmt.Printf("✓ Removed %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
