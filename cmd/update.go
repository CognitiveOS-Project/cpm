package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a patch to the latest version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !patch.IsInstalled(name) {
			return fmt.Errorf("patch %q is not installed", name)
		}

		current, err := patch.ReadManifest(name)
		if err != nil {
			return fmt.Errorf("read current manifest: %w", err)
		}

		regURL := resolveRegistry()
		if regURL == "" {
			return fmt.Errorf("no registry configured")
		}

		rc := registry.New(regURL)
		meta, err := rc.GetMetadata(name, "")
		if err != nil {
			return fmt.Errorf("resolve from registry: %w", err)
		}

		if meta.Version == current.Version {
			fmt.Printf("%s v%s is already the latest version\n", name, current.Version)
			return nil
		}

		// Download to staging
		cacheDir := cacheDir()
		_ = os.MkdirAll(cacheDir, 0755)
		cachePath := filepath.Join(cacheDir, name+"-"+meta.Version+".cgp")

		if _, err := os.Stat(cachePath); os.IsNotExist(err) {
			body, err := rc.Download(meta.Name, meta.Version)
			if err != nil {
				return fmt.Errorf("download: %w", err)
			}
			defer body.Close()
			f, err := os.Create(cachePath)
			if err != nil {
				return fmt.Errorf("create cache: %w", err)
			}
			defer f.Close()
			_, _ = io.Copy(f, body)
		}

		// Extract to staging
		stagingDir := filepath.Join(patch.PatchesDir, ".staging", name)
		_ = os.RemoveAll(stagingDir)

		f, err := os.Open(cachePath)
		if err != nil {
			return fmt.Errorf("open cache: %w", err)
		}
		defer f.Close()

		if err := archive.Extract(f, stagingDir); err != nil {
			os.RemoveAll(stagingDir)
			return fmt.Errorf("extract staging: %w", err)
		}

		// Swap
		trashDir := filepath.Join(patch.PatchesDir, ".trash", name)
		_ = os.RemoveAll(trashDir)

		if err := os.Rename(patch.Dir(name), trashDir); err != nil {
			os.RemoveAll(stagingDir)
			return fmt.Errorf("swap: %w", err)
		}
		if err := os.Rename(stagingDir, patch.Dir(name)); err != nil {
			_ = os.Rename(trashDir, patch.Dir(name))
			os.RemoveAll(stagingDir)
			return fmt.Errorf("swap: %w", err)
		}
		os.RemoveAll(trashDir)

		log.Info("Updated %s: %s → %s", name, current.Version, meta.Version)
		fmt.Printf("✓ Updated %s: %s → %s\n", name, current.Version, meta.Version)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
