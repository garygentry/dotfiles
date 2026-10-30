package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/garygentry/dotfiles/internal/config"
	"github.com/garygentry/dotfiles/internal/secrets"
	"github.com/spf13/cobra"
)

var secretRef string

var getSecretCmd = &cobra.Command{
	Use:    "get-secret",
	Short:  "Retrieve a secret from the configured provider",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// The root command silences errors, and this runs inside module scripts
		// (via get_secret), so say why it failed on stderr or the script just sees
		// a bare non-zero exit.
		err := runGetSecret()
		if err != nil {
			fmt.Fprintf(os.Stderr, "dotfiles get-secret: %v\n", err)
		}
		return err
	},
}

func runGetSecret() error {
	// Determine the dotfiles directory.
	dotfilesDir := os.Getenv("DOTFILES_DIR")
	if dotfilesDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("determining home directory: %w", err)
		}
		dotfilesDir = filepath.Join(home, ".dotfiles")
	}

	// Load configuration to find the secrets provider settings.
	cfg, err := config.Load(dotfilesDir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider := secrets.NewProvider(cfg.Secrets.Provider, cfg.Secrets.Account)

	if !provider.Available() {
		return fmt.Errorf("secrets provider %q is not available (prerequisites not installed)", provider.Name())
	}

	if !provider.IsAuthenticated() {
		return fmt.Errorf("secrets provider %q is not signed in; sign in (1Password: eval \"$(op signin)\") and re-run", provider.Name())
	}

	value, err := provider.GetSecret(secretRef)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(os.Stdout, value)
	return err
}

func init() {
	getSecretCmd.Flags().StringVar(&secretRef, "ref", "", "Secret reference (e.g. op://vault/item/field)")
	_ = getSecretCmd.MarkFlagRequired("ref")
	rootCmd.AddCommand(getSecretCmd)
}
