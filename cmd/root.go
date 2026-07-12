package cmd

import (
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/spf13/cobra"
)

var (
	registryURL string
	verbose     bool
	yesMode     bool
	noAudit     bool
	rootDir     string
)

var rootCmd = &cobra.Command{
	Use:   "cpm",
	Short: "Cognitive Package Manager",
	Long: `cpm installs, removes, and manages .cgp cognitive patches
for the CognitiveOS ecosystem.`,
	SilenceUsage: true,
}

func Execute() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&registryURL, "registry", "", "Override default registry URL")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Detailed output")
	rootCmd.PersistentFlags().BoolVar(&yesMode, "yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&noAudit, "no-audit", false, "Skip hardware audit")
	rootCmd.PersistentFlags().StringVar(&rootDir, "root", "/", "Target root directory for cross-compilation")

	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if rootDir != "/" {
			// Adjust patch directory to be relative to the specified root
			patch.PatchesDir = filepath.Join(rootDir, "cognitiveos", "patches")
		}
	}
}
