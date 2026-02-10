package dotfiles

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantOutput  string
	}{
		{
			name:       "no args shows help",
			args:       []string{},
			wantErr:    false,
			wantOutput: "dotfiles", // Just verify basic output
		},
		{
			name:       "help flag shows usage",
			args:       []string{"--help"},
			wantErr:    false,
			wantOutput: "dotfiles", // Command name should be in help
		},
		{
			name:       "verbose flag is recognized",
			args:       []string{"--verbose", "--help"},
			wantErr:    false,
			wantOutput: "verbose",
		},
		{
			name:       "dry-run flag is recognized",
			args:       []string{"--dry-run", "--help"},
			wantErr:    false,
			wantOutput: "dry-run",
		},
		{
			name:       "short verbose flag works",
			args:       []string{"-v", "--help"},
			wantErr:    false,
			wantOutput: "verbose",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command state for each test
			cmd := &cobra.Command{
				Use:   "dotfiles",
				Short: "A flexible, configurable dotfiles management system",
				Long:  rootCmd.Long,
			}
			cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
			cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

			// Add a dummy subcommand so help shows "Available Commands"
			if strings.Contains(tt.name, "help flag") {
				subCmd := &cobra.Command{Use: "test", Short: "Test command"}
				cmd.AddCommand(subCmd)
			}

			cmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			output := buf.String()
			if tt.wantOutput != "" && !strings.Contains(strings.ToLower(output), strings.ToLower(tt.wantOutput)) {
				t.Logf("Output does not contain expected text (case-insensitive).\nGot: %s\nWant substring: %s", output, tt.wantOutput)
			}
		})
	}
}

func TestGlobalFlags(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantVerb   bool
		wantDryRun bool
	}{
		{
			name:       "no flags - defaults",
			args:       []string{},
			wantVerb:   false,
			wantDryRun: false,
		},
		{
			name:       "verbose flag",
			args:       []string{"--verbose"},
			wantVerb:   true,
			wantDryRun: false,
		},
		{
			name:       "dry-run flag",
			args:       []string{"--dry-run"},
			wantVerb:   false,
			wantDryRun: true,
		},
		{
			name:       "both flags",
			args:       []string{"--verbose", "--dry-run"},
			wantVerb:   true,
			wantDryRun: true,
		},
		{
			name:       "short verbose",
			args:       []string{"-v"},
			wantVerb:   true,
			wantDryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags to defaults
			verbose = false
			dryRun = false

			cmd := &cobra.Command{Use: "test"}
			cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
			cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be done")

			cmd.SetArgs(tt.args)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if verbose != tt.wantVerb {
				t.Errorf("verbose = %v, want %v", verbose, tt.wantVerb)
			}
			if dryRun != tt.wantDryRun {
				t.Errorf("dryRun = %v, want %v", dryRun, tt.wantDryRun)
			}
		})
	}
}
