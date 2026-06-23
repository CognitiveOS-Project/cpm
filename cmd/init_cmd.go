package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [<dir>]",
	Short: "Create a .cgp skeleton directory",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "my-skill"
		if len(args) > 0 {
			dir = args[0]
		}

		if _, err := os.Stat(dir); err == nil {
			return fmt.Errorf("directory %q already exists", dir)
		}

		dirs := []string{
			filepath.Join(dir, "prompts", "templates"),
			filepath.Join(dir, "tools"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("create %s: %w", d, err)
			}
		}

		manifest := map[string]interface{}{
			"name":        filepath.Base(dir),
			"version":     "0.1.0",
			"description": "Describe what this patch does",
			"author":      "",
			"license":     "MIT",
			"hardware_requirements": map[string]interface{}{
				"min_ram_mb":     512,
				"min_storage_mb": 50,
			},
			"runtime": map[string]interface{}{
				"system_prompt": "prompts/system.md",
				"tools_root":    "tools",
			},
		}

		mf, err := os.Create(filepath.Join(dir, "cognitive.json"))
		if err != nil {
			return fmt.Errorf("create cognitive.json: %w", err)
		}
		defer mf.Close()

		enc := json.NewEncoder(mf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(manifest); err != nil {
			return fmt.Errorf("write cognitive.json: %w", err)
		}

		systemPrompt := `# System Prompt

You are a CognitiveOS skill. When loaded, your behavior is defined here.
`
		if err := os.WriteFile(filepath.Join(dir, "prompts", "system.md"), []byte(systemPrompt), 0644); err != nil {
			return fmt.Errorf("write system.md: %w", err)
		}

		fmt.Printf("✓ Created .cgp skeleton in %s/\n", dir)
		fmt.Printf("  Next: edit %s and add your tools/prompts\n", filepath.Join(dir, "cognitive.json"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
