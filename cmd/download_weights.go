package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/weights"
	"github.com/spf13/cobra"
)

var (
	downloadProvider string
	downloadKind     string
	downloadFormat   string
	downloadOutput   string
	downloadDryRun   bool
)

var downloadWeightsCmd = &cobra.Command{
	Use:   "download-weights <model-name>",
	Short: "Download model weights from a provider",
	Long: `Download model weights (GGUF, safetensors) from a weight provider.

Provider: hf (Hugging Face Hub) — searches public GGUF models sorted by downloads.
  cpm download-weights --provider hf --kind wide --type gguf google/gemma-4-2b
  cpm download-weights --provider hf --kind raw --type gguf CognitiveOS/raw-model

Files are placed at:
  --kind raw  → /cognitiveos/models/raw/raw-model-<name>.gguf (skip if exists)
  --kind wide → /cognitiveos/models/wide/active/<filename>.gguf
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelName := args[0]

		var kind weights.Kind
		switch downloadKind {
		case "raw":
			kind = weights.KindRaw
		case "wide":
			kind = weights.KindWide
		default:
			return fmt.Errorf("invalid --kind: %q (must be raw or wide)", downloadKind)
		}

		var format weights.Format
		switch downloadFormat {
		case "gguf":
			format = weights.FormatGGUF
		case "safetensors":
			format = weights.FormatSafeTensors
		default:
			return fmt.Errorf("invalid --type: %q (must be gguf or safetensors)", downloadFormat)
		}

		var prov weights.Provider
		switch downloadProvider {
		case "hf":
			prov = weights.NewHFProvider()
		default:
			return fmt.Errorf("invalid --provider: %q (must be hf)", downloadProvider)
		}

		ctx := context.Background()

		candidates, err := prov.Search(ctx, modelName, 5)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		formatExt := "." + format.String()

		var match *weights.Candidate
		for i := range candidates {
			if strings.HasSuffix(strings.ToLower(candidates[i].Filename), formatExt) {
				match = &candidates[i]
				break
			}
		}
		if match == nil {
			for i := range candidates {
				if strings.Contains(strings.ToLower(candidates[i].Filename), strings.ToLower(modelName)) {
					match = &candidates[i]
					break
				}
			}
		}
		if match == nil && len(candidates) > 0 {
			match = &candidates[0]
		}
		if match == nil {
			return fmt.Errorf("no %s files found for %q", formatExt, modelName)
		}

		dest := downloadOutput
		if dest == "" {
			dest = resolveDest(kind, match)
		}
		fmt.Printf("Found: %s\n", match.DownloadURL)
		fmt.Printf("File:  %s\n", match.Filename)
		fmt.Printf("Dest:  %s\n", dest)

		if downloadDryRun {
			return nil
		}

		if kind == weights.KindRaw {
			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("File already exists at %s — skipping (use --kind wide to re-download)\n", dest)
				return nil
			}
		}

		fmt.Println("Downloading...")
		if err := weights.Download(ctx, match.DownloadURL, dest, match.SHA256, weights.TextProgress); err != nil {
			return fmt.Errorf("download: %w", err)
		}
		fmt.Printf("✓ Downloaded to %s\n", dest)
		return nil
	},
}

func resolveDest(kind weights.Kind, c *weights.Candidate) string {
	dir := weights.ModelDir(kind)
	if kind == weights.KindRaw {
		base := strings.TrimSuffix(c.Filename, filepath.Ext(c.Filename))
		name := fmt.Sprintf("raw-model-%s%s", base, filepath.Ext(c.Filename))
		return filepath.Join(dir, name)
	}
	return filepath.Join(dir, c.Filename)
}

func init() {
	fs := downloadWeightsCmd.Flags()
	fs.StringVar(&downloadProvider, "provider", "hf", "Weight provider (hf)")
	fs.StringVar(&downloadKind, "kind", "wide", "Model kind (raw or wide)")
	fs.StringVar(&downloadFormat, "type", "gguf", "File format (gguf or safetensors)")
	fs.StringVar(&downloadOutput, "output", "", "Custom output path (overrides default)")
	fs.BoolVar(&downloadDryRun, "dry-run", false, "Show what would be downloaded without downloading")
	rootCmd.AddCommand(downloadWeightsCmd)
}
