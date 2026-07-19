package cmd

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"github.com/spf13/cobra"
)

var (
	searchLicense    string
	searchMinRAM     int
	searchPage       int
	searchCapability string
	searchExact      bool
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the registry for patches",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		if len(query) < 2 {
			return fmt.Errorf("ERROR:S001: query too short (min 2 characters)")
		}

		regURL := resolveRegistry()
		if regURL == "" {
			return fmt.Errorf("ERROR:S002: no registry configured")
		}

		rc := registry.New(regURL)
		opts := registry.SearchOptions{
			License:     searchLicense,
			MinRAM:      searchMinRAM,
			Page:        searchPage,
			PerPage:     20,
			Capability:  searchCapability,
			Exact:       searchExact,
		}
		results, err := rc.Search(query, opts)
		if err != nil {
			return fmt.Errorf("ERROR:S003: search failed: %w", err)
		}

		if len(results.Results) == 0 {
			fmt.Println("No results found")
			return nil
		}

		fmt.Printf("Found %d matches:\n", results.Total)
		for _, r := range results.Results {
			fmt.Printf("  %-20s %-8s %s\n", r.Name, r.Version, r.Description)
		}
		return nil
	},
}

func init() {
	fs := searchCmd.Flags()
	fs.StringVar(&searchLicense, "license", "", "Filter by SPDX license")
	fs.IntVar(&searchMinRAM, "min-ram", 0, "Minimum RAM in MB")
	fs.IntVar(&searchPage, "page", 1, "Page number")
	fs.StringVar(&searchCapability, "capability", "", "Filter by capability (e.g. model.llm, display.render)")
	fs.BoolVar(&searchExact, "exact", false, "Exact name match")
	_ = searchCmd.RegisterFlagCompletionFunc("license", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"MIT", "Apache-2.0", "GPL-3.0", "BSL-1.0"}, cobra.ShellCompDirectiveDefault
	})
	rootCmd.AddCommand(searchCmd)
}
