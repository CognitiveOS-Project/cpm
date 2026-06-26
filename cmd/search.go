package cmd

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var (
	searchLicense string
	searchMinRAM  int
	searchPage    int
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search across all registries for patches",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		if len(query) < 2 {
			return fmt.Errorf("query too short (min 2 characters)")
		}

		regs := resolveConfig()
		entries := regs.All()

		seen := map[string]bool{}
		total := 0

		for _, entry := range entries {
			rc := registry.New(entry.URL)
			results, err := rc.Search(query, searchPage, 20)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "  [%s] search failed: %v\n", entry.Name, err)
				continue
			}

			for _, r := range results.Results {
				key := r.Name + "@" + r.Version
				if seen[key] {
					continue
				}
				seen[key] = true
				if total == 0 {
					fmt.Printf("Found matches across %d registries:\n", len(entries))
				}
				fmt.Printf("  %-20s %-8s %s  [%s]\n", r.Name, r.Version, r.Description, entry.Name)
				total++
			}
		}

		if total == 0 {
			fmt.Println("No results found")
		}
		return nil
	},
}

func init() {
	fs := searchCmd.Flags()
	fs.StringVar(&searchLicense, "license", "", "Filter by SPDX license")
	fs.IntVar(&searchMinRAM, "min-ram", 0, "Minimum RAM in MB")
	fs.IntVar(&searchPage, "page", 1, "Page number")
	searchCmd.RegisterFlagCompletionFunc("license", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"MIT", "Apache-2.0", "GPL-3.0", "BSL-1.0"}, cobra.ShellCompDirectiveDefault
	})
	rootCmd.AddCommand(searchCmd)
}
