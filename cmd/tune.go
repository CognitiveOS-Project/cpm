package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/daemon"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/spf13/cobra"
)

var (
	tuneBackground bool
	tuneEpochs     int
	tuneFinalize   bool
	tuneQuantize   string
)

var tuneCmd = &cobra.Command{
	Use:   "tune <name>",
	Short: "Fine-tune an installed patch using local interaction data",
	Long: `Trigger local fine-tuning for a specific package.
This uses the training configuration in the package manifest to invoke 
the designated training tool via the system daemon.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgName := args[0]

		// 1. Verify package is installed
		if !patch.IsInstalled(pkgName) {
			return fmt.Errorf("ERROR:T101: patch %s not installed", pkgName)
		}

		// 2. Read training config from manifest
		m, err := patch.ReadManifest(pkgName)
		if err != nil {
			return fmt.Errorf("ERROR:T102: failed to read manifest: %w", err)
		}

		if m.Training == nil {
			return fmt.Errorf("ERROR:T103: package %s does not support fine-tuning (no training config)", pkgName)
		}

		// 3. Trigger tuning or finalization
		if tuneFinalize {
			// Finalize/Quantize
			payload := map[string]interface{}{
				"package":   pkgName,
				"quantize":  tuneQuantize,
				"finalize":  true,
			}
			if err := daemon.SendMessage("cpm_tune", payload); err != nil {
				return fmt.Errorf("ERROR:T104: failed to trigger finalization: %w", err)
			}
			fmt.Printf("✓ Finalization requested for %s (quant: %s)\n", pkgName, tuneQuantize)
		} else {
			// Standard Tuning
			// Readiness check: check interaction log size
			logPath := filepath.Join("/cognitiveos/data/interactions", pkgName+".jsonl")
			samples := 0
			if data, err := os.ReadFile(logPath); err == nil {
				// Simple count of newlines as proxy for samples
				for i := 0; i < len(data); i++ {
					if data[i] == '\n' {
						samples++
					}
				}
			}

			if m.Training.DataRequirements != nil && samples < m.Training.DataRequirements.MinSamples {
				return fmt.Errorf("ERROR:T105: insufficient data for tuning (%d/%d samples)", 
					samples, m.Training.DataRequirements.MinSamples)
			}

			payload := map[string]interface{}{
				"package":    pkgName,
				"background": tuneBackground,
				"epochs":     tuneEpochs,
				"finalize":   false,
			}
			if err := daemon.SendMessage("cpm_tune", payload); err != nil {
				return fmt.Errorf("ERROR:T106: failed to trigger tuning: %w", err)
			}
			fmt.Printf("✓ Tuning triggered for %s (background: %v, epochs: %d)\n", 
				pkgName, tuneBackground, tuneEpochs)
		}

		return nil
	},
}

func init() {
	fs := tuneCmd.Flags()
	fs.BoolVar(&tuneBackground, "background", false, "Run tuning in the background")
	fs.IntVar(&tuneEpochs, "epochs", 1, "Number of training epochs")
	fs.BoolVar(&tuneFinalize, "finalize", false, "Finalize and quantize the adapter")
	fs.StringVar(&tuneQuantize, "quantize", "", "Quantization level for finalization (e.g. Q4_K_M)")
	rootCmd.AddCommand(tuneCmd)
}
