package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	registryURL string
	verbose     bool
	yesMode     bool
	noAudit     bool
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
}
