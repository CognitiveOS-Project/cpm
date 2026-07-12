package cmd

import (
	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify <path>",
	Short: "Verify a .cgp archive integrity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return archive.Verify(args[0])
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
}
