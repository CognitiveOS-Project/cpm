package manager

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

// Install installs a system dependency using the appropriate package manager.
func Install(root string, dep archive.SystemDependency) error {
	var cmd *exec.Cmd

	switch dep.Manager {
	case "apk":
		// For Alpine, use --root to install into target directory if provided
		args := []string{"add"}
		if root != "/" {
			args = append(args, "--root", root)
		}
		
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s=%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		cmd = exec.Command("apk", args...)

	case "npm":
		args := []string{"install", "-g"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s@%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		cmd = exec.Command("npm", args...)

	case "bun":
		args := []string{"install", "-g"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s@%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		cmd = exec.Command("bun", args...)

	case "pip":
		args := []string{"install"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			pkg = fmt.Sprintf("%s==%s", dep.Name, dep.Version)
		}
		args = append(args, pkg)
		cmd = exec.Command("pip", args...)

	case "cargo":
		args := []string{"install"}
		pkg := dep.Name
		if dep.Version != "" && dep.Version != "latest" {
			args = append(args, "--version", dep.Version)
		}
		args = append(args, pkg)
		cmd = exec.Command("cargo", args...)

	case "go":
		pkg := dep.Name
		version := "latest"
		if dep.Version != "" {
			version = "v" + dep.Version
		}
		cmd = exec.Command("go", "install", fmt.Sprintf("%s@%s", pkg, version))

	case "git":
		// For git, we clone the repo into /cognitiveos/lib/cpm/externals/<name>
		// This is a simplified implementation
		repoPath := fmt.Sprintf("/cognitiveos/lib/cpm/externals/%s", dep.Name)
		if root != "/" {
			repoPath = fmt.Sprintf("%s/cognitiveos/lib/cpm/externals/%s", root, dep.Name)
		}
		
		args := []string{"clone"}
		if dep.Version != "" && dep.Version != "latest" {
			args = append(args, "-b", dep.Version)
		}
		args = append(args, dep.Name, repoPath)
		cmd = exec.Command("git", args...)

	default:
		return fmt.Errorf("unsupported package manager: %s", dep.Manager)
	}

	if cmd == nil {
		return fmt.Errorf("failed to construct install command for %s", dep.Name)
	}

	output, err := cmd.CombinedOutput()
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
		cmd := exec.Command("apk", args...)
		err := cmd.Run()
		if err == nil {
			return true, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return false, nil
		}
		return false, err

	default:
		// For other managers, assume not installed or use a simpler check
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
		cmd := exec.Command("apk", args...)
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to query apk: %w", err)
		}
		
		// apk search output: "package-version-arch"
		line := strings.TrimSpace(string(output))
		parts := strings.Split(line, "-")
		if len(parts) >= 2 {
			// This is a very rough version extraction
			return parts[len(parts)-2], nil
		}
		return "latest", nil

	default:
		return "latest", nil
	}
}
