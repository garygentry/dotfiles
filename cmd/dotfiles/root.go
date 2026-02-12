package dotfiles

import (
	"github.com/spf13/cobra"
)

var (
	verbose    bool
	dryRun     bool
	logJSON    bool
	unattended bool
)

var rootCmd = &cobra.Command{
	Use:   "dotfiles",
	Short: "A flexible, configurable dotfiles management system",
	Long: `Dotfiles manages your system configuration with a Go CLI for orchestration
and shell-based modules for install logic. Supports macOS, Ubuntu, and Arch Linux.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")
	rootCmd.PersistentFlags().BoolVar(&logJSON, "log-json", false, "Output logs in JSON format")
	rootCmd.PersistentFlags().BoolVar(&unattended, "unattended", false, "Run without prompts, using defaults")
}

func Execute() error {
	return rootCmd.Execute()
}
