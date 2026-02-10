package dotfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/garygentry/dotfiles/internal/sysinfo"
	"github.com/garygentry/dotfiles/internal/ui"
	"github.com/spf13/cobra"
)

var (
	newPriority int
	newDepends  []string
	newOS       []string
)

var newCmd = &cobra.Command{
	Use:   "new <module-name>",
	Short: "Generate a new module skeleton",
	Long: `New creates a module skeleton with standard structure and templates.
The generated module includes module.yml, install.sh, and README.md with
TODOs and best practices to help you get started quickly.

Module names must be lowercase, alphanumeric, and may contain hyphens.

Example:
  dotfiles new tmux --priority 35 --depends git,zsh --os linux,macos`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		u := ui.New(verbose)
		moduleName := args[0]

		// Validate module name
		if !isValidModuleName(moduleName) {
			return fmt.Errorf("invalid module name %q: must be lowercase alphanumeric with hyphens only", moduleName)
		}

		sys, err := sysinfo.Detect()
		if err != nil {
			return fmt.Errorf("system detection: %w", err)
		}

		moduleDir := filepath.Join(sys.DotfilesDir, "modules", moduleName)

		// Check if module already exists
		if _, err := os.Stat(moduleDir); err == nil {
			return fmt.Errorf("module %q already exists at %s", moduleName, moduleDir)
		}

		u.Info(fmt.Sprintf("Creating module: %s", moduleName))

		// Create module directory structure
		if err := os.MkdirAll(moduleDir, 0o755); err != nil {
			return fmt.Errorf("creating module directory: %w", err)
		}

		osDir := filepath.Join(moduleDir, "os")
		if err := os.MkdirAll(osDir, 0o755); err != nil {
			return fmt.Errorf("creating os directory: %w", err)
		}

		filesDir := filepath.Join(moduleDir, "files")
		if err := os.MkdirAll(filesDir, 0o755); err != nil {
			return fmt.Errorf("creating files directory: %w", err)
		}

		// Generate module.yml
		moduleYml := generateModuleYml(moduleName, newPriority, newDepends, newOS)
		if err := os.WriteFile(filepath.Join(moduleDir, "module.yml"), []byte(moduleYml), 0o644); err != nil {
			return fmt.Errorf("writing module.yml: %w", err)
		}
		u.Success("Created module.yml")

		// Generate install.sh
		installSh := generateInstallScript(moduleName)
		if err := os.WriteFile(filepath.Join(moduleDir, "install.sh"), []byte(installSh), 0o755); err != nil {
			return fmt.Errorf("writing install.sh: %w", err)
		}
		u.Success("Created install.sh")

		// Generate verify.sh
		verifySh := generateVerifyScript(moduleName)
		if err := os.WriteFile(filepath.Join(moduleDir, "verify.sh"), []byte(verifySh), 0o755); err != nil {
			return fmt.Errorf("writing verify.sh: %w", err)
		}
		u.Success("Created verify.sh")

		// Generate OS-specific scripts
		for _, osName := range []string{"macos", "ubuntu", "arch"} {
			osScript := generateOSScript(moduleName, osName)
			osPath := filepath.Join(osDir, osName+".sh")
			if err := os.WriteFile(osPath, []byte(osScript), 0o755); err != nil {
				return fmt.Errorf("writing %s: %w", osName+".sh", err)
			}
		}
		u.Success("Created OS-specific scripts")

		// Generate README.md
		readme := generateReadme(moduleName, newPriority, newDepends)
		if err := os.WriteFile(filepath.Join(moduleDir, "README.md"), []byte(readme), 0o644); err != nil {
			return fmt.Errorf("writing README.md: %w", err)
		}
		u.Success("Created README.md")

		// Generate example config file
		exampleConfig := generateExampleConfig(moduleName)
		if err := os.WriteFile(filepath.Join(filesDir, "example.conf"), []byte(exampleConfig), 0o644); err != nil {
			return fmt.Errorf("writing example.conf: %w", err)
		}
		u.Success("Created example config file")

		u.Info("")
		u.Success(fmt.Sprintf("Module %q created successfully!", moduleName))
		u.Info(fmt.Sprintf("Location: %s", moduleDir))
		u.Info("")
		u.Info("Next steps:")
		u.Info(fmt.Sprintf("  1. Edit %s/module.yml to configure your module", moduleName))
		u.Info(fmt.Sprintf("  2. Implement installation logic in %s/install.sh", moduleName))
		u.Info(fmt.Sprintf("  3. Test with: dotfiles install %s --dry-run", moduleName))

		return nil
	},
}

func init() {
	newCmd.Flags().IntVar(&newPriority, "priority", 50, "Module priority (lower runs first)")
	newCmd.Flags().StringSliceVar(&newDepends, "depends", []string{}, "Comma-separated list of dependencies")
	newCmd.Flags().StringSliceVar(&newOS, "os", []string{}, "Comma-separated list of supported OSes (empty = all)")
	rootCmd.AddCommand(newCmd)
}

// isValidModuleName checks if the module name is valid (lowercase alphanumeric with hyphens)
func isValidModuleName(name string) bool {
	if name == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-z0-9]+(-[a-z0-9]+)*$`, name)
	return matched
}

// generateModuleYml creates the module.yml content
func generateModuleYml(name string, priority int, depends, os []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("name: %s\n", name))
	sb.WriteString(fmt.Sprintf("description: TODO: Add description for %s\n", name))
	sb.WriteString("version: 1.0.0\n")
	sb.WriteString(fmt.Sprintf("priority: %d\n", priority))

	if len(depends) > 0 {
		sb.WriteString("dependencies:\n")
		for _, dep := range depends {
			sb.WriteString(fmt.Sprintf("  - %s\n", dep))
		}
	} else {
		sb.WriteString("dependencies: []\n")
	}

	if len(os) > 0 {
		sb.WriteString("os:\n")
		for _, osName := range os {
			sb.WriteString(fmt.Sprintf("  - %s\n", osName))
		}
	} else {
		sb.WriteString("os: []  # Empty means all platforms\n")
	}

	sb.WriteString("requires: []  # TODO: Add system requirements (commands that must exist)\n")
	sb.WriteString("tags:\n")
	sb.WriteString("  - TODO\n")
	sb.WriteString("\n")
	sb.WriteString("# Uncomment to set a custom timeout (default: 5m)\n")
	sb.WriteString("# timeout: 10m\n")
	sb.WriteString("\n")
	sb.WriteString("# Uncomment to deploy config files\n")
	sb.WriteString("# files:\n")
	sb.WriteString("#   - source: files/config.conf\n")
	sb.WriteString("#     dest: ~/.config/" + name + "/config.conf\n")
	sb.WriteString("#     type: symlink  # or 'copy' or 'template'\n")

	return sb.String()
}

// generateInstallScript creates the install.sh content
func generateInstallScript(name string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# %s/install.sh - Install and configure %s

# This script is sourced by the dotfiles CLI, so helpers.sh is already loaded.
# Available functions: log_info, log_success, log_error, log_warn, pkg_install, etc.
# Available env vars: DOTFILES_OS, DOTFILES_ARCH, DOTFILES_PKG_MGR, DOTFILES_HOME, etc.

set -euo pipefail

log_info "Installing %s..."

# TODO: Implement installation logic here
# Examples:
#   pkg_install %s
#   curl -L https://example.com/install.sh | bash
#   git clone https://github.com/example/%s ~/.%s

# TODO: Create necessary directories
# mkdir -p "$DOTFILES_HOME/.config/%s"

# TODO: Set up configuration
# More complex setup can go in OS-specific scripts (os/macos.sh, etc.)

log_success "%s installed successfully"
`, name, name, name, name, name, name, name, name)
}

// generateVerifyScript creates the verify.sh content
func generateVerifyScript(name string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# %s/verify.sh - Verify %s installation

set -euo pipefail

log_info "Verifying %s installation..."

# TODO: Add verification checks
# Examples:
#   command -v %s >/dev/null 2>&1 || { log_error "%s not found in PATH"; exit 1; }
#   [ -f "$DOTFILES_HOME/.config/%s/config" ] || { log_error "Config file not found"; exit 1; }

log_success "%s verification passed"
`, name, name, name, name, name, name, name)
}

// generateOSScript creates OS-specific script content
func generateOSScript(name, osName string) string {
	return fmt.Sprintf(`#!/usr/bin/env bash
# %s/os/%s.sh - %s-specific setup for %s

set -euo pipefail

# TODO: Implement %s-specific installation logic
# This script runs BEFORE install.sh
# Use this for OS-specific package installation, etc.

log_info "Running %s-specific setup for %s..."

# Example: Install via package manager
# case "$DOTFILES_PKG_MGR" in
#     apt)
#         sudo apt-get update
#         sudo apt-get install -y %s
#         ;;
#     pacman)
#         sudo pacman -S --noconfirm %s
#         ;;
#     brew)
#         brew install %s
#         ;;
# esac
`, name, osName, osName, name, osName, osName, name, name, name, name)
}

// generateReadme creates the README.md content
func generateReadme(name string, priority int, depends []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s Module\n\n", name))
	sb.WriteString("TODO: Add a brief description of what this module does.\n\n")
	sb.WriteString("## Features\n\n")
	sb.WriteString("- TODO: List key features\n")
	sb.WriteString("- TODO: Mention important configuration\n")
	sb.WriteString("- TODO: Highlight integrations\n\n")
	sb.WriteString("## Prerequisites\n\n")

	if len(depends) > 0 {
		sb.WriteString("This module depends on:\n")
		for _, dep := range depends {
			sb.WriteString(fmt.Sprintf("- `%s`\n", dep))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("No dependencies.\n\n")
	}

	sb.WriteString("## Installation\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("dotfiles install %s\n", name))
	sb.WriteString("```\n\n")
	sb.WriteString("## Configuration\n\n")
	sb.WriteString("TODO: Describe configuration options and how to customize.\n\n")
	sb.WriteString("## Platform Support\n\n")
	sb.WriteString("- macOS\n")
	sb.WriteString("- Ubuntu\n")
	sb.WriteString("- Arch Linux\n\n")
	sb.WriteString("## Notes\n\n")
	sb.WriteString("TODO: Add any important notes, gotchas, or tips.\n")

	return sb.String()
}

// generateExampleConfig creates an example configuration file
func generateExampleConfig(name string) string {
	return fmt.Sprintf(`# Example configuration file for %s
# TODO: Add example configuration options

# This file can be:
# - Symlinked to the target location (type: symlink)
# - Copied to the target location (type: copy)
# - Rendered as a Go template (type: template)

# For templates, you can use:
# - {{ .User.name }} - User's name from config
# - {{ .User.email }} - User's email
# - {{ .OS }} - Operating system
# - {{ .Home }} - Home directory
# - etc.
`, name)
}
