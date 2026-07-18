package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/auth"
	"github.com/spf13/cobra"
)

var authKeyPath string

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage registry authentication",
}

var authRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register SSH public key with the registry",
	Long: `Register your SSH public key with the configured registry.

This is a one-time operation per key. The server stores only your
public key — no secrets are transmitted.

Examples:
  cpm auth register
  cpm auth register --key ~/.ssh/my_key.pub`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyPath := authKeyPath
		if keyPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("ERROR:A001: cannot find home directory: %w", err)
			}
			keyPath = filepath.Join(home, ".ssh", "id_ed25519.pub")
		}

		pubKey, err := auth.LoadPublicKey(keyPath)
		if err != nil {
			return fmt.Errorf("ERROR:A002: load public key: %w", err)
		}

		rc := getRegistryClient()
		resp, err := rc.RegisterPublicKey(pubKey)
		if err != nil {
			return fmt.Errorf("ERROR:A003: register: %w", err)
		}

		fmt.Printf("Registered SSH key\n")
		fmt.Printf("  Fingerprint: %s\n", resp.Fingerprint)
		fmt.Printf("  Key type:    %s\n", resp.KeyType)
		if resp.Comment != "" {
			fmt.Printf("  Comment:     %s\n", resp.Comment)
		}
		fmt.Printf("  Registered:  %s\n", resp.RegisteredAt)
		return nil
	},
}

func init() {
	authRegisterCmd.Flags().StringVar(&authKeyPath, "key", "", "SSH public key path (default: ~/.ssh/id_ed25519.pub)")
	authCmd.AddCommand(authRegisterCmd)
	rootCmd.AddCommand(authCmd)
}
