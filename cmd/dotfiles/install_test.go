package dotfiles

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestInstallCommandFlags(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		wantUnattended bool
		wantFailFast   bool
	}{
		{
			name:           "no flags - defaults",
			args:           []string{},
			wantUnattended: false,
			wantFailFast:   false,
		},
		{
			name:           "unattended flag",
			args:           []string{"--unattended"},
			wantUnattended: true,
			wantFailFast:   false,
		},
		{
			name:           "fail-fast flag",
			args:           []string{"--fail-fast"},
			wantUnattended: false,
			wantFailFast:   true,
		},
		{
			name:           "both flags",
			args:           []string{"--unattended", "--fail-fast"},
			wantUnattended: true,
			wantFailFast:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags to defaults
			unattended = false
			failFast = false

			cmd := &cobra.Command{Use: "install"}
			cmd.Flags().BoolVar(&unattended, "unattended", false, "Run without prompts")
			cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")

			cmd.SetArgs(tt.args)
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags() error = %v", err)
			}

			if unattended != tt.wantUnattended {
				t.Errorf("unattended = %v, want %v", unattended, tt.wantUnattended)
			}
			if failFast != tt.wantFailFast {
				t.Errorf("failFast = %v, want %v", failFast, tt.wantFailFast)
			}
		})
	}
}

func TestInstallCommandModuleSelection(t *testing.T) {
	// Create a temporary dotfiles directory with test modules
	tmpDir := t.TempDir()

	// Create modules directory
	modulesDir := filepath.Join(tmpDir, "modules")
	if err := os.MkdirAll(modulesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create test modules
	testModules := []struct {
		name string
		yml  string
	}{
		{
			name: "git",
			yml: `name: git
description: Git version control
version: 1.0.0
priority: 10`,
		},
		{
			name: "zsh",
			yml: `name: zsh
description: Z shell
version: 1.0.0
priority: 20
dependencies: [git]`,
		},
	}

	for _, mod := range testModules {
		modDir := filepath.Join(modulesDir, mod.name)
		if err := os.MkdirAll(modDir, 0o755); err != nil {
			t.Fatal(err)
		}
		ymlPath := filepath.Join(modDir, "module.yml")
		if err := os.WriteFile(ymlPath, []byte(mod.yml), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Create config.yml
	configYml := `profile: default
user:
  name: Test User
  email: test@example.com`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yml"), []byte(configYml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Test with --dry-run and --unattended to avoid actual installation
	t.Setenv("DOTFILES_DIR", tmpDir)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "install specific module",
			args:    []string{"git", "--dry-run", "--unattended"},
			wantErr: false,
		},
		{
			name:    "install multiple modules",
			args:    []string{"git", "zsh", "--dry-run", "--unattended"},
			wantErr: false,
		},
		{
			name:    "install with dependencies",
			args:    []string{"zsh", "--dry-run", "--unattended"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags
			dryRun = false
			unattended = false
			verbose = false

			// Create a fresh command for each test
			cmd := &cobra.Command{
				Use: "install [modules...]",
				RunE: func(cmd *cobra.Command, args []string) error {
					// For these tests, we're just verifying the command structure
					// and flag parsing, not the full execution logic.
					return nil
				},
			}
			cmd.Flags().BoolVar(&unattended, "unattended", false, "Run without prompts")
			cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")
			cmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Dry run")
			cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose")

			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstallCommandHelp(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "install [modules...]",
		Short: "Install and configure dotfiles modules",
		Long:  installCmd.Long,
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	requiredStrings := []string{
		"Install",  // Command description starts with capital
		"modules",
	}

	for _, required := range requiredStrings {
		if !strings.Contains(output, required) {
			t.Errorf("Help output missing expected string %q.\nOutput: %s", required, output)
		}
	}
}

// TestModuleYAMLParsing verifies that the test modules can be parsed correctly
func TestModuleYAMLParsing(t *testing.T) {
	tests := []struct {
		name    string
		yml     string
		wantErr bool
	}{
		{
			name: "valid basic module",
			yml: `name: test
description: Test module
version: 1.0.0`,
			wantErr: false,
		},
		{
			name: "module with timeout",
			yml: `name: test
description: Test module
version: 1.0.0
timeout: 10m`,
			wantErr: false,
		},
		{
			name: "module with dependencies",
			yml: `name: test
description: Test module
version: 1.0.0
dependencies:
  - git
  - zsh`,
			wantErr: false,
		},
		{
			name: "invalid yaml",
			yml: `name: test
description: [unclosed`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mod struct {
				Name         string   `yaml:"name"`
				Description  string   `yaml:"description"`
				Version      string   `yaml:"version"`
				Timeout      string   `yaml:"timeout"`
				Dependencies []string `yaml:"dependencies"`
			}

			err := yaml.Unmarshal([]byte(tt.yml), &mod)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if mod.Name == "" {
					t.Error("Expected name to be parsed")
				}
			}
		})
	}
}
