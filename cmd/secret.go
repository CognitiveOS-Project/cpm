package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/config"
	"github.com/spf13/cobra"
)

var secretScope string
var secretForce bool
var secretReveal bool

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets (API keys, tokens, credentials)",
}

var secretAddCmd = &cobra.Command{
	Use:   "add <name> <value>",
	Short: "Add or update a secret",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		value := args[1]

		s, err := config.LoadSecrets(secretScope, patchNameForScope())
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}

		if !secretForce {
			if _, exists := s.Secrets[name]; exists {
				return fmt.Errorf("secret %q already exists in %s scope, use --force to overwrite", name, secretScope)
			}
		}

		s.Secrets[name] = value

		if err := config.SaveSecrets(secretScope, patchNameForScope(), s); err != nil {
			return fmt.Errorf("save secrets: %w", err)
		}

		if secretForce {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated secret %q in %s scope\n", name, secretScope)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Added secret %q to %s scope\n", name, secretScope)
		}
		return nil
	},
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List secrets (merged view by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		var global, patch *config.Secrets
		var err error

		global, err = config.LoadSecrets("global", "")
		if err != nil {
			return fmt.Errorf("load global secrets: %w", err)
		}

		patchName := patchNameForScope()
		if patchName != "" {
			patch, err = config.LoadSecrets("patch", patchName)
			if err != nil {
				return fmt.Errorf("load patch secrets: %w", err)
			}
		}

		merged := config.MergeSecrets(global, patch)
		out := cmd.OutOrStdout()

		if len(merged.Secrets) == 0 {
			fmt.Fprintln(out, "No secrets configured.")
			fmt.Fprintf(out, "\nAdd one: cpm secret add <name> <value> --scope %s\n", secretScope)
			return nil
		}

		// Show patch secrets
		if patch != nil && len(patch.Secrets) > 0 {
			scopeTitle := secretScope
			if len(scopeTitle) > 0 {
				scopeTitle = strings.ToUpper(scopeTitle[:1]) + scopeTitle[1:]
			}
			fmt.Fprintf(out, "%s:\n", scopeTitle)
			for k, v := range patch.Secrets {
				if secretReveal {
					fmt.Fprintf(out, "  %s\t%s\n", k, v)
				} else {
					fmt.Fprintf(out, "  %s\t****%s\n", k, lastN(v, 4))
				}
			}
		}

		// Show global secrets (only if not filtered to patch scope)
		if global != nil && len(global.Secrets) > 0 && secretScope != "patch" {
			if patch != nil && len(patch.Secrets) > 0 {
				fmt.Fprintln(out)
			}
			fmt.Fprintln(out, "Global:")
			for k, v := range global.Secrets {
				// Skip if overridden by patch
				if patch != nil {
					if _, ok := patch.Secrets[k]; ok {
						continue
					}
				}
				if secretReveal {
					fmt.Fprintf(out, "  %s\t%s\n", k, v)
				} else {
					fmt.Fprintf(out, "  %s\t****%s\n", k, lastN(v, 4))
				}
			}
		}

		return nil
	},
}

var secretRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a secret",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := config.RemoveSecret(secretScope, patchNameForScope(), name); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed secret %q from %s scope\n", name, secretScope)
		return nil
	},
}

var secretGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Print a secret value to stdout",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		// Try patch scope first, then global
		s, err := config.LoadSecrets("patch", patchNameForScope())
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}

		if val, ok := s.Secrets[name]; ok {
			fmt.Fprint(cmd.OutOrStdout(), val)
			return nil
		}

		s, err = config.LoadSecrets("global", "")
		if err != nil {
			return fmt.Errorf("load secrets: %w", err)
		}

		if val, ok := s.Secrets[name]; ok {
			fmt.Fprint(cmd.OutOrStdout(), val)
			return nil
		}

		return fmt.Errorf("secret %q not found in any scope", name)
	},
}

func patchNameForScope() string {
	if secretScope == "global" {
		return ""
	}
	// Try to read manifest name from CWD
	m, err := archive.LoadManifest("cognitive.json")
	if err != nil || m.Name == "" {
		// Fall back to directory name
		dir, _ := os.Getwd()
		return filepath.Base(dir)
	}
	return m.Name
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

func init() {
	secretCmd.PersistentFlags().StringVar(&secretScope, "scope", "patch", "Scope: patch or global")
	secretCmd.PersistentFlags().BoolVar(&secretForce, "force", false, "Overwrite existing secret")

	secretAddCmd.Flags().BoolVar(&secretForce, "force", false, "Overwrite existing secret")
	secretListCmd.Flags().BoolVar(&secretReveal, "reveal", false, "Show secret values instead of masking")

	secretCmd.AddCommand(secretAddCmd)
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretRemoveCmd)
	secretCmd.AddCommand(secretGetCmd)

	rootCmd.AddCommand(secretCmd)
}
