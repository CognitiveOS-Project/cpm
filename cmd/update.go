package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/resolver"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a patch to the latest version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !patch.IsInstalled(name) {
			return fmt.Errorf("ERROR:U001: patch %q is not installed", name)
		}

		current, err := patch.ReadManifest(name)
		if err != nil {
			return fmt.Errorf("ERROR:U002: read current manifest: %w", err)
		}

		if !yesMode {
			fmt.Printf("Update %s from v%s to latest? [y/N]: ", name, current.Version)
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" && confirm != "yes" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		regURL := resolveRegistry()
		if regURL != "" {
			result, err := resolver.Resolve(name, regURL)
			if err != nil {
				return fmt.Errorf("ERROR:U003: resolve %q: %w", name, err)
			}

			if result.Manifest.Version == current.Version {
				fmt.Printf("%s v%s is already the latest version\n", name, current.Version)
				os.RemoveAll(result.DataDir)
				return nil
			}

			trashDir := filepath.Join(patch.PatchesDir, ".trash", name)
			installPath := patch.Dir(name)
			_ = os.RemoveAll(trashDir)

			if err := os.Rename(installPath, trashDir); err != nil {
				os.RemoveAll(result.DataDir)
				return fmt.Errorf("ERROR:U004: swap: %w", err)
			}
			if err := os.Rename(result.DataDir, installPath); err != nil {
				_ = os.Rename(trashDir, installPath)
				os.RemoveAll(result.DataDir)
				return fmt.Errorf("ERROR:U005: swap: %w", err)
			}
			os.RemoveAll(trashDir)

			log.Info("Updated %s: %s → %s", name, current.Version, result.Manifest.Version)
			fmt.Printf("✓ Updated %s: %s → %s\n", name, current.Version, result.Manifest.Version)
			return nil
		}

		return fmt.Errorf("ERROR:U006: no registry configured for update")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
