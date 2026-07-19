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
	Long: `Create a .cgp skeleton with a cognitive.json manifest.

Templates:
  default        Basic skill with system prompt and tools dir
  prompt-only    Minimal skill — just a system prompt, no tools
  mcp-bridge     MCP server bridge wrapping an external tool
  gguf-model     GGUF model distributed via CognitiveOS
  firmware       Self-contained firmware for the Raw Model
  full           Full-featured skill with everything`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "my-skill"
		if len(args) > 0 {
			dir = args[0]
		}

		if _, err := os.Stat(dir); err == nil {
			return fmt.Errorf("ERROR:INIT001: directory %q already exists", dir)
		}

		switch initTemplate {
		case "prompt-only":
			return initPromptOnly(dir)
		case "mcp-bridge":
			return initMCPBridge(dir)
		case "gguf-model":
			return initGGUFModel(dir)
		case "firmware":
			return initFirmware(dir)
		case "full":
			return initFull(dir)
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

func initPromptOnly(dir string) error {
	if err := os.MkdirAll(filepath.Join(dir, "prompts"), 0755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	manifest := map[string]interface{}{
		"name":        filepath.Base(dir),
		"version":     "0.1.0",
		"description": "Prompt-only skill — no tools",
		"author":      "",
		"license":     "MIT",
		"runtime": map[string]interface{}{
			"system_prompt": "prompts/system.md",
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		return err
	}

	systemPrompt := `# System Prompt

You are a prompt-only CognitiveOS skill. You influence the AI's behavior
but do not provide any tools or MCP servers.
`
	if err := os.WriteFile(filepath.Join(dir, "prompts", "system.md"), []byte(systemPrompt), 0644); err != nil {
		return fmt.Errorf("write system.md: %w", err)
	}

	fmt.Printf("✓ Created prompt-only skeleton in %s/\n", dir)
	return nil
}

func initMCPBridge(dir string) error {
	dirs := []string{
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
		"description": "MCP server bridge",
		"author":      "",
		"license":     "MIT",
		"source": map[string]interface{}{
			"repository": "",
			"issues":     "",
		},
		"hardware_requirements": map[string]interface{}{
			"min_ram_mb":     256,
			"min_storage_mb": 10,
		},
		"runtime": map[string]interface{}{
			"tools_root": "tools",
			"mcp_servers": []map[string]interface{}{
				{
					"name":      filepath.Base(dir),
					"command":   "tools/mcp-server",
					"args":      []string{},
					"transport": "stdio",
				},
			},
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		return err
	}

	fmt.Printf("✓ Created mcp-bridge skeleton in %s/\n", dir)
	fmt.Printf("  Next: place your MCP server binary at tools/mcp-server\n")
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

func initFirmware(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	manifest := map[string]interface{}{
		"name":        filepath.Base(dir),
		"version":     "1.0.0",
		"description": "Raw Model firmware",
		"author":      "",
		"license":     "MIT",
		"source": map[string]interface{}{
			"repository": "",
			"issues":     "",
		},
		"hardware_requirements": map[string]interface{}{
			"min_ram_mb":     256,
			"min_storage_mb": 100,
			"npu_required":   false,
		},
		"checksum": map[string]interface{}{
			"sha256": "",
		},
		"brain": map[string]interface{}{
			"raw_model": map[string]interface{}{
				"weights": map[string]interface{}{
					"remote": map[string]interface{}{
						"source":   "huggingface",
						"model_id": "CognitiveOS/raw-model",
						"filename": "raw-model.gguf",
						"format":   "gguf",
					},
				},
				"parameters": map[string]interface{}{
					"temperature": 0.1,
					"num_ctx":     4096,
				},
			},
		},
		"runtime": map[string]interface{}{
			"capabilities": []string{"model.firmware", "model.raw"},
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		return err
	}

	fmt.Printf("✓ Created firmware skeleton in %s/\n", dir)
	fmt.Printf("  Next: edit cognitive.json — set model_id, filename\n")
	return nil
}

func initFull(dir string) error {
	dirs := []string{
		filepath.Join(dir, "prompts", "templates"),
		filepath.Join(dir, "tools"),
		filepath.Join(dir, "weights"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}

	manifest := map[string]interface{}{
		"name":        filepath.Base(dir),
		"version":     "0.1.0",
		"description": "Full-featured CognitiveOS skill",
		"author":      "",
		"license":     "MIT",
		"source": map[string]interface{}{
			"repository": "",
			"issues":     "",
		},
		"dependencies": map[string]string{},
		"hardware_requirements": map[string]interface{}{
			"min_ram_mb":     1024,
			"min_storage_mb": 100,
			"npu_required":   false,
		},
		"checksum": map[string]interface{}{
			"sha256": "",
		},
		"brain": map[string]interface{}{
			"wide_model": map[string]interface{}{
				"base_model": "",
				"parameters": map[string]interface{}{
					"temperature": 0.7,
					"num_ctx":     8192,
				},
			},
		},
		"runtime": map[string]interface{}{
			"system_prompt": "prompts/system.md",
			"tools_root":    "tools",
			"mcp_servers":   []map[string]interface{}{},
			"capabilities":  []string{},
		},
	}

	if err := writeManifest(dir, manifest); err != nil {
		return err
	}

	systemPrompt := `# System Prompt

You are a full-featured CognitiveOS skill with tools, prompts, templates,
and optional model weights.
`
	if err := os.WriteFile(filepath.Join(dir, "prompts", "system.md"), []byte(systemPrompt), 0644); err != nil {
		return fmt.Errorf("write system.md: %w", err)
	}

	fmt.Printf("✓ Created full skeleton in %s/\n", dir)
	fmt.Println("  Structure:")
	fmt.Println("    cognitive.json     — manifest (edit me)")
	fmt.Println("    prompts/system.md  — system prompt")
	fmt.Println("    prompts/templates/ — prompt templates")
	fmt.Println("    tools/             — MCP servers and scripts")
	fmt.Println("    weights/           — model weights (optional)")
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
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Skeleton template (prompt-only, mcp-bridge, gguf-model, firmware, full)")
	rootCmd.AddCommand(initCmd)
}
