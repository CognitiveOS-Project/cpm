package cmd

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/queue"
	"github.com/spf13/cobra"
)

var registerDepsCmd = &cobra.Command{
	Use:   "register-dependencies <package>",
	Short: "Register system-level dependencies for an installed patch in the queue",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		packageName := args[0]
		
		// Resolve root from global flag
		root := "/"
		if rootDir != "/" {
			root = rootDir
		}

		// 1. Read manifest from installed patch
		installPath := patch.Dir(packageName)
		// If root is specified, adjust the path
		if root != "/" {
			installPath = fmt.Sprintf("%s/%s", root, installPath)
		}
		
		manifest, err := archive.LoadManifest(installPath)
		if err != nil {
			return fmt.Errorf("could not load manifest for %s: %w", packageName, err)
		}

		if manifest.HardwareDependencies == nil || len(manifest.HardwareDependencies.Packages) == 0 {
			log.Info("No system dependencies to register for %s", packageName)
			return nil
		}

		// 2. Register each package
		count := 0
		for _, dep := range manifest.HardwareDependencies.Packages {
			if err := queue.Register(root, manifest.Name, manifest.Version, dep); err != nil {
				return fmt.Errorf("failed to register dependency %s: %w", dep.Name, err)
			}
			count++
		}

		log.Info("Registered %d system dependencies for %s", count, packageName)
		fmt.Printf("✓ Registered %d system dependencies for %s\n", count, packageName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registerDepsCmd)
}
