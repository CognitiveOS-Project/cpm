package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CognitiveOS-Project/cpm/internal/auth"
	"github.com/CognitiveOS-Project/cpm/internal/config"
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

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Store SSH key locally and verify with registry",
	Long: `Store your SSH key path in local config and verify it is registered
with the configured registry.

Examples:
  cpm auth login
  cpm auth login --key ~/.ssh/my_key`,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyPath := authKeyPath
		if keyPath == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("ERROR:A004: cannot find home directory: %w", err)
			}
			keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		}

		signer, err := auth.LoadPrivateKey(keyPath)
		if err != nil {
			return fmt.Errorf("ERROR:A005: load private key: %w", err)
		}

		fingerprint := auth.PublicKeyFingerprint(signer)

		cfg := &config.AuthConfig{
			KeyPath:     keyPath,
			Fingerprint: fingerprint,
		}

		rc := getRegistryClient()
		status, err := rc.CheckAuthStatus(fingerprint)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not verify key with registry: %v\n", err)
			cfg.Registered = false
		} else {
			cfg.Registered = status.Registered
			cfg.RegisteredAt = status.RegisteredAt
		}

		if err := config.SaveAuth(cfg); err != nil {
			return fmt.Errorf("ERROR:A006: save auth config: %w", err)
		}

		fmt.Printf("Logged in\n")
		fmt.Printf("  Key:        %s\n", keyPath)
		fmt.Printf("  Fingerprint: %s\n", fingerprint)
		if cfg.Registered {
			fmt.Printf("  Status:     registered\n")
			fmt.Printf("  Registered: %s\n", cfg.RegisteredAt)
		} else {
			fmt.Printf("  Status:     not registered (run: cpm auth register)\n")
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear local authentication state",
	Long: `Remove the stored SSH key path from local config.

This does not affect the server — your key remains registered.
It only removes the local reference.

Examples:
  cpm auth logout`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.RemoveAuth(); err != nil {
			return fmt.Errorf("ERROR:A007: logout: %w", err)
		}
		fmt.Println("Logged out")
		return nil
	},
}

func init() {
	authRegisterCmd.Flags().StringVar(&authKeyPath, "key", "", "SSH public key path (default: ~/.ssh/id_ed25519.pub)")
	authLoginCmd.Flags().StringVar(&authKeyPath, "key", "", "SSH private key path (default: ~/.ssh/id_ed25519)")
	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}
