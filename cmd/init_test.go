package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		template  string
		wantErr   bool
		checkFiles []string
	}{
		{
			name: "default template",
			args: []string{"my-skill"},
			checkFiles: []string{
				"my-skill/cognitive.json",
				"my-skill/prompts/system.md",
				"my-skill/prompts/templates/",
				"my-skill/tools/",
			},
		},
		{
			name: "prompt-only template",
			args: []string{"prompt-skill"},
			template: "prompt-only",
			checkFiles: []string{
				"prompt-skill/cognitive.json",
				"prompt-skill/prompts/system.md",
			},
		},
		{
			name: "mcp-bridge template",
			args: []string{"bridge-skill"},
			template: "mcp-bridge",
			checkFiles: []string{
				"bridge-skill/cognitive.json",
				"bridge-skill/tools/",
			},
		},
		{
			name: "gguf-model template",
			args: []string{"model-skill"},
			template: "gguf-model",
			checkFiles: []string{
				"model-skill/cognitive.json",
			},
		},
		{
			name: "firmware template",
			args: []string{"firmware-skill"},
			template: "firmware",
			checkFiles: []string{
				"firmware-skill/cognitive.json",
			},
		},
		{
			name: "full template",
			args: []string{"full-skill"},
			template: "full",
			checkFiles: []string{
				"full-skill/cognitive.json",
				"full-skill/prompts/system.md",
				"full-skill/prompts/templates/",
				"full-skill/tools/",
				"full-skill/weights/",
			},
		},
		{
			name: "directory already exists",
			args: []string{"existing-dir"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "cpm-init-test-*")
			if err != nil {
				t.Fatalf("failed to create tmp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Change working directory to tmpDir
			oldWd, _ := os.Getwd()
			_ = os.Chdir(tmpDir)
			defer func() { _ = os.Chdir(oldWd) }()

			// If testing "directory already exists", create it first
			if tc.name == "directory already exists" {
				_ = os.Mkdir("existing-dir", 0755)
			}

			// Setup the command
			cmd := initCmd
			initTemplate = tc.template

			// We need to set up the flags for the command
			// Since initCmd is already added to rootCmd, we just set the variable
			
			// Use a helper to execute the command since it's a cobra.Command
			// We'll call RunE directly to avoid os.Exit or printing to stdout
			
			// To use RunE, we need to simulate the Cobra context
			runErr := initCmd.RunE(cmd, tc.args)


			if (runErr != nil) != tc.wantErr {
				t.Errorf("initCmd.RunE() error = %v, wantErr %v", runErr, tc.wantErr)
			}

			if !tc.wantErr {
				for _, file := range tc.checkFiles {
					if _, err := os.Stat(filepath.Join(tmpDir, file)); os.IsNotExist(err) {
						t.Errorf("expected file/dir %s not found", file)
					}
				}
			}
		})
	}
}
