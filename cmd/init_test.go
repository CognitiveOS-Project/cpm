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
				"my-skill/.github/docker/Dockerfile.ci",
				"my-skill/.github/workflows/ci.yml",
				"my-skill/.github/workflows/publish.yml",
			},
		},
		{
			name: "prompt-only template",
			args: []string{"prompt-skill"},
			template: "prompt-only",
			checkFiles: []string{
				"prompt-skill/cognitive.json",
				"prompt-skill/prompts/system.md",
				"prompt-skill/.github/docker/Dockerfile.ci",
				"prompt-skill/.github/workflows/ci.yml",
				"prompt-skill/.github/workflows/publish.yml",
			},
		},
		{
			name: "mcp-bridge template",
			args: []string{"bridge-skill"},
			template: "mcp-bridge",
			checkFiles: []string{
				"bridge-skill/cognitive.json",
				"bridge-skill/tools/",
				"bridge-skill/.github/docker/Dockerfile.ci",
				"bridge-skill/.github/workflows/ci.yml",
				"bridge-skill/.github/workflows/publish.yml",
			},
		},
		{
			name: "gguf-model template",
			args: []string{"model-skill"},
			template: "gguf-model",
			checkFiles: []string{
				"model-skill/cognitive.json",
				"model-skill/.github/docker/Dockerfile.ci",
				"model-skill/.github/workflows/ci.yml",
				"model-skill/.github/workflows/publish.yml",
			},
		},
		{
			name: "firmware template",
			args: []string{"firmware-skill"},
			template: "firmware",
			checkFiles: []string{
				"firmware-skill/cognitive.json",
				"firmware-skill/.github/docker/Dockerfile.ci",
				"firmware-skill/.github/workflows/ci.yml",
				"firmware-skill/.github/workflows/publish.yml",
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
				"full-skill/.github/docker/Dockerfile.ci",
				"full-skill/.github/workflows/ci.yml",
				"full-skill/.github/workflows/publish.yml",
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

			cmd := initCmd
			initTemplate = tc.template

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
