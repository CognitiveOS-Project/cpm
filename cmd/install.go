package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/audit"
	"github.com/CognitiveOS-Project/cpm/internal/check"
	"github.com/CognitiveOS-Project/cpm/internal/config"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
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
			return fmt.Errorf("already installed")
		}

		// Determine source via universal protocol resolver
		regURL := resolveRegistry()
		result, err := resolver.Resolve(target, regURL)
		if err != nil {
			return fmt.Errorf("resolve %q: %w", target, err)
		}

		m := result.Manifest

		// Validate schema
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
		if err := schema.Validate(doc); err != nil {
			return fmt.Errorf("validation: %w", err)
		}

		// Hardware audit
		if !noAudit {
			res, err := audit.Run()
			if err != nil {
				log.Warn("Hardware audit failed: %v", err)
			} else if err := audit.Check(m.HardwareRequirements, res); err != nil {
				return fmt.Errorf("hardware: %w", err)
			}
		}

		// Source validation — check issues URL reachability
		if m.Source != nil {
			if m.Source.Issues != "" {
				if err := check.IssuesReachable(m.Source.Issues); err != nil {
					log.Warn("Source issues URL: %v", err)
				}
			}

			result, err := check.CheckForBugs(m.Source)
			if err != nil {
				return fmt.Errorf("bug check: %w", err)
			}
			if result.HasBugs {
				log.Audit("known_bugs", map[string]interface{}{
					"name":  m.Name,
					"count": result.Count,
					"urls":  result.URLs,
				})
				return fmt.Errorf("refusing to install %q — %d open bug(s) found", m.Name, result.Count)
			}
		}

		// Remote weight download — check manifest for weights.remote.source
		if err := downloadRemoteWeights(result.DataDir); err != nil {
			return fmt.Errorf("download weights: %w", err)
		}

		// Move extracted data to install path
		installPath := patch.Dir(m.Name)
		_ = os.RemoveAll(installPath)
		if err := os.MkdirAll(filepath.Dir(installPath), 0755); err != nil {
			_ = os.RemoveAll(result.DataDir)
			return fmt.Errorf("create parent dir: %w", err)
		}

		if err := os.Rename(result.DataDir, installPath); err != nil {
			// Fallback: copy across filesystems
			if err := copyDir(result.DataDir, installPath); err != nil {
				_ = os.RemoveAll(installPath)
				_ = os.RemoveAll(result.DataDir)
				return fmt.Errorf("extract: %w", err)
			}
			_ = os.RemoveAll(result.DataDir)
		}

		// Log checksum for audit trail
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
	},
}

func cacheDir() string {
	if d := os.Getenv("CPM_CACHE_DIR"); d != "" {
		return d
	}
	return "/cognitiveos/data/cache/downloads"
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

func isURL(s string) bool {
	return len(s) > 4 && (s[:4] == "http" || s[:5] == "https")
}

func defaultPrimary() string {
	return "https://registry-us-all-distros-official.cognitive-os.org/v1"
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

		candidates, err := prov.Search(ctx, modelID, 3)
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
