package manager

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

// CmdExecutor abstracts command execution for testability.
type CmdExecutor interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) ([]byte, error)
	CombinedOutput(name string, args ...string) ([]byte, error)
}

type realExecutor struct{}

func (e realExecutor) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (e realExecutor) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (e realExecutor) CombinedOutput(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Executor is the default command executor. Can be replaced in tests.
var Executor CmdExecutor = realExecutor{}

// Install installs a system dependency using the appropriate package manager.
func Install(root string, dep archive.SystemDependency) error {
	var name string
	var args []string

	switch dep.Manager {
	case "apk":
		args = []string{"add"}
		if root != "/" {
			args = append(args, "--root", root)
		}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s=%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		name = "apk"

	case "npm":
		args = []string{"install", "-g"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s@%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		name = "npm"

	case "bun":
		args = []string{"install", "-g"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s@%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		name = "bun"

	case "pip":
		args = []string{"install"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s==%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		name = "pip"

	case "cargo":
		args = []string{"install"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			args = append(args, "--version", dep.Version)
		}
		args = append(args, pkg)
		name = "cargo"

	case "go":
		pkg := dep.Name
		version := "latest"
		if dep.Version != "" {
			version = "v" + dep.Version
		}
		args = []string{"install", fmt.Sprintf("%s@%s", pkg, version)}
		name = "go"

	case "git":
		repoPath := fmt.Sprintf("/cognitiveos/lib/cpm/externals/%s", dep.Name)
		if root != "/" {
			repoPath = fmt.Sprintf("%s/cognitiveos/lib/cpm/externals/%s", root, dep.Name)
		}
		args = []string{"clone"}
		if dep.Version != "" && dep.Version != "latest" {
			args = append(args, "-b", dep.Version)
		}
		args = append(args, dep.Name, repoPath)
		name = "git"

	default:
		return fmt.Errorf("unsupported package manager: %s", dep.Manager)
	}

	output, err := Executor.CombinedOutput(name, args...)
	if err != nil {
		return fmt.Errorf("install %s failed: %w (output: %s)", dep.Name, err, string(output))
	}

	return nil
}

// IsInstalled checks if a package is already installed.
func IsInstalled(root string, dep archive.SystemDependency) (bool, error) {
	switch dep.Manager {
	case "apk":
		args := []string{"info", "-e", dep.Name}
		if root != "/" {
			args = append([]string{"--root", root}, args...)
		}
		err := Executor.Run("apk", args...)
		if err == nil {
			return true, nil
		}
		// If the error is not an ExitError, something went wrong with the execution
		if _, ok := err.(*exec.ExitError); !ok {
			return false, err
		}
		return false, nil

	default:
		return false, nil
	}
}

// ResolveVersion queries the package manager for the latest version.
func ResolveVersion(root string, dep archive.SystemDependency) (string, error) {
	switch dep.Manager {
	case "apk":
		args := []string{"search", "-e", dep.Name}
		if root != "/" {
			args = append([]string{"--root", root}, args...)
		}
		output, err := Executor.Output("apk", args...)
		if err != nil {
			return "", fmt.Errorf("failed to query apk: %w", err)
		}

		line := strings.TrimSpace(string(output))
		parts := strings.Split(line, "-")
		if len(parts) >= 2 {
			return parts[len(parts)-2], nil
		}
		return "latest", nil

	default:
		return "latest", nil
	}
}
