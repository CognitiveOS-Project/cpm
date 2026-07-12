package cmd

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/manager"
	"github.com/CognitiveOS-Project/cpm/internal/queue"
	"github.com/spf13/cobra"
)

var installDepsCmd = &cobra.Command{
	Use:   "install-dependencies",
	Short: "Install pending system dependencies for a specific lifecycle stage",
	RunE: func(cmd *cobra.Command, args []string) error {
		stage, _ := cmd.Flags().GetString("stage")
		if stage == "" {
			return fmt.Errorf("missing required flag --stage")
		}

		root := "/"
		if rootDir != "/" {
			root = rootDir
		}

		// 1. List queue files for the stage
		entries, err := queue.ListByStage(root, stage)
		if err != nil {
			return fmt.Errorf("failed to list queue for stage %s: %w", stage, err)
		}

		if len(entries) == 0 {
			log.Info("No pending dependencies for stage %s", stage)
			return nil
		}

		// 2. Process each entry
		installedCount := 0
		for _, entry := range entries {
			log.Info("Installing %s for %s...", entry.Dependency.Name, entry.PatchName)
			if err := manager.Install(root, entry.Dependency); err != nil {
				if entry.Dependency.Required {
					return fmt.Errorf("critical dependency %s failed: %w", entry.Dependency.Name, err)
				}
				log.Warn("Optional dependency %s failed: %v", entry.Dependency.Name, err)
				// Mark as installed anyway to avoid infinite retry loop for optional deps
				_ = queue.MarkInstalled(root, stage, entry.Filename)
				continue
			}
			
			if err := queue.MarkInstalled(root, stage, entry.Filename); err != nil {
				return fmt.Errorf("failed to mark %s as installed: %w", entry.Dependency.Name, err)
			}
			installedCount++
		}

		log.Info("Installed %d/%d dependencies for stage %s", installedCount, len(entries), stage)
		fmt.Printf("✓ Installed %d/%d dependencies for stage %s\n", installedCount, len(entries), stage)
		return nil
	},
}

func init() {
	installDepsCmd.Flags().String("stage", "", "Lifecycle stage to process (build, boot, install, runtime)")
	rootCmd.AddCommand(installDepsCmd)
}
