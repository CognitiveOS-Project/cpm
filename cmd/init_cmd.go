package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initTemplate string

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

		switch initTemplate {
		case "gguf-model":
			return initGGUFModel(dir)
		default:
			return initDefault(dir)
		}
	},
}

func initDefault(dir string) error {
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

	if err := writeManifest(dir, manifest); err != nil {
		return err
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
}

func initGGUFModel(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	manifest := map[string]interface{}{
		"name":        filepath.Base(dir),
		"version":     "1.0.0",
		"description": "GGUF model distributed via CognitiveOS",
		"author":      "",
		"license":     "MIT",
		"source": map[string]interface{}{
			"repository": "",
			"issues":     "",
		},
		"hardware_requirements": map[string]interface{}{
			"min_ram_mb":     4096,
			"min_storage_mb": 2048,
			"npu_required":   false,
		},
		"checksum": map[string]interface{}{
			"sha256": "",
		},
		"brain": map[string]interface{}{
			"wide_model": map[string]interface{}{
				"base_model": filepath.Base(dir),
				"weights": map[string]interface{}{
					"remote": map[string]interface{}{
						"source":     "huggingface",
						"model_id":   "org/model-name",
						"filename":   "model-name-Q4_K_M.gguf",
						"format":     "gguf",
						"quant":      "Q4_K_M",
						"size_bytes": 0,
					},
				},
				"parameters": map[string]interface{}{
					"temperature": 0.7,
					"num_ctx":     8192,
				},
			},
		},
		"runtime": map[string]interface{}{
			"capabilities": []string{"model.llm", "model.chat"},
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		return err
	}

	fmt.Printf("✓ Created gguf-model skeleton in %s/\n", dir)
	fmt.Printf("  Next: edit cognitive.json — set model_id, filename, quant, size_bytes\n")
	return nil
}

func writeManifest(dir string, manifest map[string]interface{}) error {
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
	return nil
}

func init() {
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Skeleton template (gguf-model)")
	rootCmd.AddCommand(initCmd)
}
