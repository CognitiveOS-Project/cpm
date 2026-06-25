package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/audit"
	"github.com/CognitiveOS-Project/cpm/internal/check"
	"github.com/CognitiveOS-Project/cpm/internal/config"
	"github.com/CognitiveOS-Project/cpm/internal/log"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/CognitiveOS-Project/cpm/internal/schema"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <path|name>",
	Short: "Install a .cgp cognitive patch",
	Long: `Install a patch from a local .cgp file or resolve from registry.

Examples:
  cpm install ./email-manager.cgp
  cpm install email-manager`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]

		if patch.IsInstalled(target) {
			log.Error("Patch %q is already installed", target)
			return fmt.Errorf("already installed")
		}

		// Determine source
		var m *archive.Manifest
		var dataPath string

		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			f, err := os.Open(target)
			if err != nil {
				return fmt.Errorf("open %s: %w", target, err)
			}
			defer f.Close()

			m, err = archive.ReadManifest(f)
			if err != nil {
				return fmt.Errorf("read archive: %w", err)
			}
			dataPath = target
		} else {
			regURL := resolveRegistry()
			if regURL == "" {
				return fmt.Errorf("no registry configured")
			}
			rc := registry.New(regURL)

			meta, err := rc.GetMetadata(target, "")
			if err != nil {
				return fmt.Errorf("resolve %q from registry: %w", target, err)
			}

			cacheDir := cacheDir()
			_ = os.MkdirAll(cacheDir, 0755)
			dataPath = filepath.Join(cacheDir, meta.Name+"-"+meta.Version+".cgp")

			if _, err := os.Stat(dataPath); err != nil {
				tmpPath := dataPath + ".tmp"
				body, err := rc.Download(meta.Name, meta.Version)
				if err != nil {
					return fmt.Errorf("download: %w", err)
				}

				f, err := os.Create(tmpPath)
				if err != nil {
					body.Close()
					return fmt.Errorf("create temp: %w", err)
				}
				if _, err := io.Copy(f, body); err != nil {
					body.Close()
					f.Close()
					os.Remove(tmpPath)
					return fmt.Errorf("write temp: %w", err)
				}
				body.Close()
				f.Close()

				if err := os.Rename(tmpPath, dataPath); err != nil {
					os.Remove(tmpPath)
					return fmt.Errorf("rename: %w", err)
				}
			}

			f, err := os.Open(dataPath)
			if err != nil {
				return fmt.Errorf("open cached: %w", err)
			}
			m, err = archive.ReadManifest(f)
			f.Close()
			if err != nil {
				return fmt.Errorf("read manifest: %w", err)
			}
		}

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

		// Extract
		installPath := patch.Dir(m.Name)
		if err := os.MkdirAll(installPath, 0755); err != nil {
			return fmt.Errorf("create install dir: %w", err)
		}

		f, err := os.Open(dataPath)
		if err != nil {
			return fmt.Errorf("open archive: %w", err)
		}
		defer f.Close()

		if err := archive.Extract(f, installPath); err != nil {
			os.RemoveAll(installPath)
			return fmt.Errorf("extract: %w", err)
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
		return registryURL
	}
	cfg, err := config.Load(config.RegistriesPath())
	if err != nil {
		return "https://registry.cognitive-os.org/v1"
	}
	return cfg.DefaultRegistry
}

func init() {
	rootCmd.AddCommand(installCmd)
}
