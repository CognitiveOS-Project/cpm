package manager

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

type mockExecutor struct {
	runFunc            func(name string, args ...string) error
	outputFunc         func(name string, args ...string) ([]byte, error)
	combinedOutputFunc func(name string, args ...string) ([]byte, error)
}

func (m *mockExecutor) Run(name string, args ...string) error {
	if m.runFunc != nil {
		return m.runFunc(name, args...)
	}
	return nil
}

func (m *mockExecutor) Output(name string, args ...string) ([]byte, error) {
	if m.outputFunc != nil {
		return m.outputFunc(name, args...)
	}
	return []byte(""), nil
}

func (m *mockExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	if m.combinedOutputFunc != nil {
		return m.combinedOutputFunc(name, args...)
	}
	return []byte(""), nil
}

func TestInstall(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		dep      archive.SystemDependency
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{
			name: "apk standard",
			root: "/",
			dep: archive.SystemDependency{
				Name: "zlib", Manager: "apk", Version: "latest",
			},
			wantCmd:  "apk",
			wantArgs: []string{"add", "zlib"},
			wantErr:  false,
		},
		{
			name: "apk custom root and version",
			root: "/mnt/target",
			dep: archive.SystemDependency{
				Name: "zlib", Manager: "apk", Version: "1.2.11",
			},
			wantCmd:  "apk",
			wantArgs: []string{"add", "--root", "/mnt/target", "zlib=1.2.11"},
			wantErr:  false,
		},
		{
			name: "npm",
			root: "/",
			dep: archive.SystemDependency{
				Name: "typescript", Manager: "npm", Version: "5.0.0",
			},
			wantCmd:  "npm",
			wantArgs: []string{"install", "-g", "typescript@5.0.0"},
			wantErr:  false,
		},
		{
			name: "pip",
			root: "/",
			dep: archive.SystemDependency{
				Name: "requests", Manager: "pip", Version: "2.28.0",
			},
			wantCmd:  "pip",
			wantArgs: []string{"install", "requests==2.28.0"},
			wantErr:  false,
		},
		{
			name: "cargo",
			root: "/",
			dep: archive.SystemDependency{
				Name: "ripgrep", Manager: "cargo", Version: "13.0.0",
			},
			wantCmd:  "cargo",
			wantArgs: []string{"install", "--version", "13.0.0", "ripgrep"},
			wantErr:  false,
		},
		{
			name: "go",
			root: "/",
			dep: archive.SystemDependency{
				Name: "github.com/example/tool", Manager: "go", Version: "1.0.0",
			},
			wantCmd:  "go",
			wantArgs: []string{"install", "github.com/example/tool@v1.0.0"},
			wantErr:  false,
		},
		{
			name: "git",
			root: "/mnt/target",
			dep: archive.SystemDependency{
				Name: "github.com/example/repo", Manager: "git", Version: "main",
			},
			wantCmd:  "git",
			wantArgs: []string{"clone", "-b", "main", "github.com/example/repo", "/mnt/target/cognitiveos/lib/cpm/externals/github.com/example/repo"},
			wantErr:  false,
		},
		{
			name: "unsupported",
			root: "/",
			dep: archive.SystemDependency{
				Name: "test", Manager: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockExecutor{
				combinedOutputFunc: func(name string, args ...string) ([]byte, error) {
					if name != tc.wantCmd {
						t.Errorf("expected cmd %s, got %s", tc.wantCmd, name)
					}
					if len(args) != len(tc.wantArgs) {
						t.Errorf("expected %d args, got %d", len(tc.wantArgs), len(args))
					}
					for i, arg := range args {
						if i < len(tc.wantArgs) && arg != tc.wantArgs[i] {
							t.Errorf("arg[%d] expected %s, got %s", i, tc.wantArgs[i], arg)
						}
					}
					return []byte("success"), nil
				},
			}
			Executor = mock
			err := Install(tc.root, tc.dep)
			if (err != nil) != tc.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestIsInstalled(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		dep      archive.SystemDependency
		runFunc  func(name string, args ...string) error
		want     bool
		wantErr  bool
	}{
		{
			name: "apk installed",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "apk"},
			runFunc: func(name string, args ...string) error {
				return nil
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "apk not installed",
			root: "/",
			dep:  archive.SystemDependency{Name: "fake", Manager: "apk"},
			runFunc: func(name string, args ...string) error {
				// Return a real ExitError by running a command that fails
				return exec.Command("sh", "-c", "exit 1").Run()
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "apk error",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "apk"},
			runFunc: func(name string, args ...string) error {
				return errors.New("some internal error")
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "other manager",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "npm"},
			want:    false,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockExecutor{
				runFunc: tc.runFunc,
			}
			Executor = mock
			got, err := IsInstalled(tc.root, tc.dep)
			if (err != nil) != tc.wantErr {
				t.Errorf("IsInstalled() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("IsInstalled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name     string
		root     string
		dep      archive.SystemDependency
		output   string
		want     string
		wantErr  bool
	}{
		{
			name: "apk success",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "apk"},
			output: "zlib-1.2.11-r0",
			want: "1.2.11",
			wantErr: false,
		},
		{
			name: "apk not found",
			root: "/",
			dep:  archive.SystemDependency{Name: "fake", Manager: "apk"},
			output: "",
			want: "latest",
			wantErr: false,
		},
		{
			name: "apk error",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "apk"},
			output: "",
			wantErr: true,
		},
		{
			name: "other manager",
			root: "/",
			dep:  archive.SystemDependency{Name: "zlib", Manager: "npm"},
			want: "latest",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockExecutor{
				outputFunc: func(name string, args ...string) ([]byte, error) {
					if tc.wantErr {
						return nil, fmt.Errorf("error")
					}
					return []byte(tc.output), nil
				},
			}
			Executor = mock
			got, err := ResolveVersion(tc.root, tc.dep)
			if (err != nil) != tc.wantErr {
				t.Errorf("ResolveVersion() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("ResolveVersion() = %v, want %v", got, tc.want)
			}
		})
	}
}
