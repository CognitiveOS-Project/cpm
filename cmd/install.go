package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/audit"
	"github.com/CognitiveOS-Project/cpm/internal/check"
	"github.com/CognitiveOS-Project/cpm/internal/config"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/manager"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/queue"
	"github.com/CognitiveOS-Project/cpm/internal/resolver"
	"github.com/CognitiveOS-Project/cpm/internal/schema"
	"github.com/CognitiveOS-Project/cpm/internal/weights"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <path|name>",
	Short: "Install a .cgp cognitive patch",
	Long: `Install a cognitive patch from any source using the universal protocol resolver.

Sources include:
  - Local .cgp file:     cpm install ./email-manager.cgp
  - Registry name:       cpm install email-manager
  - GitHub repo:         cpm install github.com/user/repo@v1.0.0
  - GitHub Release:      cpm install ghr:user/repo@v1.0.0
  - npm package:         cpm install npm:@scope/name
  - Bun package:         cpm install bun:name
  - Deno module:         cpm install deno:@scope/name
  - Direct URL:          cpm install https://example.com/pkg.cgp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		if patch.IsInstalled(target) {
			log.Error("Patch %q is already installed", target)
			return fmt.Errorf("ERROR:I001: patch %q is already installed", target)
		}

		regURL := resolveRegistry()
		return installWithDeps(target, regURL)
	},
}

func installWithDeps(target, regURL string) error {
	result, err := resolver.Resolve(target, regURL)
	if err != nil {
		return fmt.Errorf("resolve %q: %w", target, err)
	}

	if err := archive.VerifyExtracted(result.DataDir); err != nil {
		_ = os.RemoveAll(result.DataDir)
		return fmt.Errorf("verify: %w", err)
	}

	m := result.Manifest

	doc := buildSchemaDoc(m)
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("ERROR:I002: schema validation: %w", err)
	}

	if len(m.Dependencies) > 0 {
		regURL := resolveRegistry()
		for depName := range m.Dependencies {
			if patch.IsInstalled(depName) {
				continue
			}
			log.Info("Installing dependency %s for %s", depName, m.Name)
			if err := installWithDeps(depName, regURL); err != nil {
				return fmt.Errorf("ERROR:I003: dependency %s: %w", depName, err)
			}
		}
	}

	// Step 3.5: Register and Install system dependencies
	if m.HardwareDependencies != nil && len(m.HardwareDependencies.Packages) > 0 {
		// 1. Register all system dependencies in the queue
		for _, dep := range m.HardwareDependencies.Packages {
			if err := queue.Register("/", m.Name, m.Version, dep); err != nil {
				return fmt.Errorf("ERROR:I010: register dependency %s: %w", dep.Name, err)
			}
		}
		log.Info("Registered %d system dependencies for %s", len(m.HardwareDependencies.Packages), m.Name)

		// 2. Immediately install dependencies for the 'install' stage
		if err := installDependenciesForStage("/", "install"); err != nil {
			// Rollback: remove registered queue files for this patch
			_ = queue.RemoveByPatch("/", m.Name, m.Version)
			return fmt.Errorf("ERROR:I011: install system dependencies: %w", err)
		}
	}

	if !noAudit {
		res, err := audit.Run()
		if err != nil {
			log.Warn("Hardware audit failed: %v", err)
		} else if err := audit.Check(m.HardwareRequirements, res); err != nil {
			return fmt.Errorf("ERROR:I004: hardware: %w", err)
		}
	}

	if m.Source != nil {
		if m.Source.Issues != "" {
			if err := check.IssuesReachable(m.Source.Issues); err != nil {
				log.Warn("Source issues URL: %v", err)
			}
		}

		bugResult, err := check.CheckForBugs(m.Source)
		if err != nil {
			return fmt.Errorf("ERROR:I005: bug check: %w", err)
		}
		if bugResult.HasBugs {
			log.Audit("known_bugs", map[string]interface{}{
				"name":  m.Name,
				"count": bugResult.Count,
				"urls":  bugResult.URLs,
			})
			return fmt.Errorf("ERROR:I006: refusing to install %q — %d open bug(s) found", m.Name, bugResult.Count)
		}
	}

	if err := downloadRemoteWeights(result.DataDir); err != nil {
		return fmt.Errorf("ERROR:I007: download weights: %w", err)
	}

	installPath := patch.Dir(m.Name)
	_ = os.RemoveAll(installPath)
	if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
		_ = os.RemoveAll(result.DataDir)
		return fmt.Errorf("ERROR:I008: create parent dir: %w", err)
	}

	if err := os.Rename(result.DataDir, installPath); err != nil {
		if err := copyDir(result.DataDir, installPath); err != nil {
			_ = os.RemoveAll(installPath)
			_ = os.RemoveAll(result.DataDir)
			return fmt.Errorf("ERROR:I009: extract: %w", err)
		}
		_ = os.RemoveAll(result.DataDir)
	}

	if result.Checksum != "" {
		log.Audit("checksum", map[string]interface{}{
			"name":     m.Name,
			"version":  m.Version,
			"checksum": result.Checksum,
		})
	}

	log.Info("Installed %s v%s", m.Name, m.Version)
	fmt.Printf("✓ Installed %s v%s\n", m.Name, m.Version)
	return nil
}

func buildSchemaDoc(m *archive.Manifest) map[string]interface{} {
	doc := map[string]interface{}{
		"name":        m.Name,
		"version":     m.Version,
		"description": m.Description,
	}
	if m.Author != "" {
		doc["author"] = m.Author
	}
	if m.License != "" {
		doc["license"] = m.License
	}
	return doc
}

func resolveRegistry() string {
	if registryURL != "" {
		if isURL(registryURL) {
			return registryURL
		}
		cfg, err := config.Load(config.RegistriesPath())
		if err != nil {
			return defaultPrimary()
		}
		url, err := cfg.Resolve(registryURL)
		if err != nil {
			return defaultPrimary()
		}
		return url
	}
	cfg, err := config.Load(config.RegistriesPath())
	if err != nil {
		return defaultPrimary()
	}
	return cfg.Official.Primary
}

func installDependenciesForStage(root, stage string) error {
	entries, err := queue.ListByStage(root, stage)
	if err != nil {
		return fmt.Errorf("list queue for stage %s: %w", stage, err)
	}

	for _, entry := range entries {
		log.Info("Installing system dependency %s for %s...", entry.Dependency.Name, entry.PatchName)
		if err := manager.Install(root, entry.Dependency); err != nil {
			if entry.Dependency.Required {
				return fmt.Errorf("required dependency %s failed: %w", entry.Dependency.Name, err)
			}
			log.Warn("Optional dependency %s failed: %v", entry.Dependency.Name, err)
		}
		if err := queue.MarkInstalled(root, stage, entry.Filename); err != nil {
			return fmt.Errorf("mark installed %s: %w", entry.Dependency.Name, err)
		}
	}
	return nil
}

func isURL(s string) bool {
	return len(s) > 4 && (s[:4] == "http" || s[:5] == "https")
}

func defaultPrimary() string {
	return "https://registry-us-all-distros-official.registry.cognitive-os.org/v1"
}

func downloadRemoteWeights(dataDir string) error {
	raw, err := os.ReadFile(filepath.Join(dataDir, "cognitive.json"))
	if err != nil {
		return nil
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil
	}

	brain, _ := doc["brain"].(map[string]interface{})
	if brain == nil {
		return nil
	}

	var downloadErr error

	downloadIfRemote := func(kind weights.Kind, kindKey string) {
		modelCfg, _ := brain[kindKey].(map[string]interface{})
		if modelCfg == nil {
			return
		}
		weightsCfg, _ := modelCfg["weights"].(map[string]interface{})
		if weightsCfg == nil {
			return
		}
		remote, _ := weightsCfg["remote"].(map[string]interface{})
		if remote == nil {
			return
		}

		source, _ := remote["source"].(string)
		if source != "huggingface" {
			return
		}

		modelID, _ := remote["model_id"].(string)
		if modelID == "" {
			downloadErr = fmt.Errorf("%s.weights.remote.model_id is required", kindKey)
			return
		}

		filename, _ := remote["filename"].(string)
		expectedSHA256, _ := remote["sha256"].(string)
		if expectedSHA256 == "" {
			if checksum, ok := doc["checksum"].(map[string]interface{}); ok {
				expectedSHA256, _ = checksum["sha256"].(string)
			}
		}

		ctx := context.Background()
		prov := weights.NewHFProvider()

		candidates, err := prov.Search(ctx, modelID, 3, weights.FormatGGUF)
		if err != nil {
			downloadErr = fmt.Errorf("search %s: %w", modelID, err)
			return
		}

		var match *weights.Candidate
		for i := range candidates {
			if filename != "" && candidates[i].Filename == filename {
				match = &candidates[i]
				break
			}
		}
		if match == nil {
			for i := range candidates {
				if strings.Contains(strings.ToLower(candidates[i].Filename), strings.ToLower(filename)) {
					match = &candidates[i]
					break
				}
			}
		}
		if match == nil && len(candidates) > 0 {
			match = &candidates[0]
		}
		if match == nil {
			downloadErr = fmt.Errorf("no matching file found for %s", modelID)
			return
		}

		dest := resolveDest(kind, match)

		if kind == weights.KindRaw {
			if _, err := os.Stat(dest); err == nil {
				log.Info("Raw model already exists at %s — skipping", dest)
				return
			}
		}

		log.Info("Downloading weights from %s", match.DownloadURL)
		if err := weights.Download(ctx, match.DownloadURL, dest, expectedSHA256, weights.TextProgress); err != nil {
			downloadErr = fmt.Errorf("download %s: %w", modelID, err)
			return
		}
		log.Info("Downloaded weights to %s", dest)
	}

	downloadIfRemote(weights.KindRaw, "raw_model")
	downloadIfRemote(weights.KindWide, "wide_model")

	return downloadErr
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

func init() {
	rootCmd.AddCommand(installCmd)
}
