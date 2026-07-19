package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPackCmd(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		setup     func(tmpDir string) error
		wantErr   bool
		wantFile  string
		checkFile string
	}{
		{
			name: "pack single binary",
			args: []string{"--bin", "testbin"},
			setup: func(tmpDir string) error {
				_ = os.WriteFile(filepath.Join(tmpDir, "testbin"), []byte("#!/bin/sh\necho 1\n"), 0755)
				return writeTestManifest(tmpDir, map[string]interface{}{
					"name": "test-pack", "version": "1.0.0",
				})
			},
			wantFile: "test-pack-1.0.0-universal.cgp",
		},
		{
			name: "pack binary directory",
			args: []string{"--bin", "bins"},
			setup: func(tmpDir string) error {
				_ = os.MkdirAll(filepath.Join(tmpDir, "bins"), 0755)
				_ = os.WriteFile(filepath.Join(tmpDir, "bins", "tool1"), []byte("#!/bin/sh\necho 1\n"), 0755)
				_ = os.WriteFile(filepath.Join(tmpDir, "bins", "tool2"), []byte("#!/bin/sh\necho 2\n"), 0755)
				return writeTestManifest(tmpDir, map[string]interface{}{
					"name": "test-pack-dir", "version": "1.0.0",
				})
			},
			wantFile: "test-pack-dir-1.0.0-universal.cgp",
		},
		{
			name: "pack with manifest flag",
			args: []string{"--manifest", "custom.json"},
			setup: func(tmpDir string) error {
				return os.WriteFile(filepath.Join(tmpDir, "custom.json"), []byte(`{"name": "custom-pack", "version": "2.0.0"}`), 0644)
			},
			wantFile: "custom-pack-2.0.0-universal.cgp",
		},
		{
			name: "pack minimal",
			args: []string{"--name", "min-pack", "--version", "0.1.0"},
			setup: func(tmpDir string) error {
				return nil
			},
			wantFile: "min-pack-0.1.0-universal.cgp",
		},
		{
			name: "pack with os and arch",
			args: []string{"--name", "os-pack", "--version", "1.0.0", "--os", "linux", "--arch", "amd64"},
			setup: func(tmpDir string) error {
				return nil
			},
			wantFile: "os-pack-1.0.0-linux-amd64.cgp",
		},
		{
			name: "pack missing manifest and name",
			args: []string{"--version", "1.0.0"},
			setup: func(tmpDir string) error {
				return nil
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "cpm-pack-test-*")
			if err != nil {
				t.Fatalf("failed to create tmp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			if tc.setup != nil {
				if err := tc.setup(tmpDir); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			oldWd, _ := os.Getwd()
			_ = os.Chdir(tmpDir)
			defer func() { _ = os.Chdir(oldWd) }()

			// Reset flags
			packBin = ""
			packName = ""
			packVersion = ""
			packOS = ""
			packArch = ""
			packDescription = ""
			packManifest = ""

			// Setup flags from args
			for i := 0; i < len(tc.args); i++ {
				switch tc.args[i] {
				case "--bin":
					packBin = tc.args[i+1]
					i++
				case "--name":
					packName = tc.args[i+1]
					i++
				case "--version":
					packVersion = tc.args[i+1]
					i++
				case "--os":
					packOS = tc.args[i+1]
					i++
				case "--arch":
					packArch = tc.args[i+1]
					i++
				case "--manifest":
					packManifest = tc.args[i+1]
					i++
				}
			}

			err = packCmd.RunE(packCmd, tc.args)
			if (err != nil) != tc.wantErr {
				t.Errorf("packCmd.RunE() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr && tc.wantFile != "" {
				if _, err := os.Stat(filepath.Join(tmpDir, tc.wantFile)); os.IsNotExist(err) {
					t.Errorf("expected file %s not found", tc.wantFile)
				}
			}
		})
	}
}

func writeTestManifest(dir string, m map[string]interface{}) error {
	data, _ := json.Marshal(m)
	return os.WriteFile(filepath.Join(dir, "cognitive.json"), data, 0644)
}
