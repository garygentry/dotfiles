# Creating Modules

This guide walks you through creating a new module for the dotfiles management system.

## Overview

A module is a self-contained unit that installs and configures a specific tool or application. Modules consist of:

- **module.yml** - Metadata and configuration
- **install.sh** - Main installation logic
- **os/*.sh** - OS-specific setup (optional)
- **verify.sh** - Post-installation verification (optional)
- **files/** - Configuration files to deploy (optional)

## Quick Start

### 1. Create Module Directory

```bash
mkdir -p modules/mymodule
cd modules/mymodule
```

### 2. Create module.yml

```yaml
name: mymodule
description: Install and configure My Tool
version: 1.0.0
priority: 100
dependencies: []
os: []  # Empty means all platforms
requires: []
tags:
  - development
  - tools

files:
  - source: files/config.conf
    dest: ~/.config/mymodule/config.conf
    type: symlink

prompts:
  - key: theme
    message: "Which theme would you like to use?"
    type: choice
    options:
      - dark
      - light
    default: dark
```

### 3. Create install.sh

```bash
#!/usr/bin/env bash
set -euo pipefail

# Log what we're doing
log_info "Installing mymodule..."

# Install the package
pkg_install mymodule

# Create config directory
mkdir -p ~/.config/mymodule

# Get user's choice from prompt
THEME="${DOTFILES_PROMPT_THEME:-dark}"
log_info "Configuring with theme: $THEME"

log_success "mymodule installed successfully"
```

### 4. Make Scripts Executable

```bash
chmod +x install.sh
```

### 5. Test Your Module

```bash
# From the dotfiles root directory
./bin/dotfiles install mymodule --dry-run

# Actually install
./bin/dotfiles install mymodule
```

## Module Structure

### Complete Example

```
modules/mymodule/
├── module.yml          # Module metadata
├── install.sh          # Main installation script
├── verify.sh           # Verification script (optional)
├── os/                 # OS-specific scripts (optional)
│   ├── macos.sh
│   ├── ubuntu.sh
│   └── arch.sh
└── files/              # Configuration files (optional)
    ├── config.conf
    ├── theme.conf.tmpl # Template file
    └── aliases.sh
```

## module.yml Schema

### Required Fields

```yaml
name: mymodule              # Module identifier (lowercase, alphanumeric, hyphens)
description: "Brief description"  # Short description
version: 1.0.0             # Semantic version
```

### Optional Fields

```yaml
priority: 100              # Execution order (default: 100, lower runs first)
dependencies:              # Other modules required first
  - git
  - zsh
os:                        # Supported platforms (empty = all)
  - macos
  - ubuntu
  - arch
requires:                  # System requirements (commands that must exist)
  - git
  - curl
tags:                      # Categorization
  - development
  - shell
timeout: 10m               # Script timeout (default: 5m)
                          # Accepts: "10s", "5m", "1h", etc.
```

### Files

Files to deploy from the module directory to the system:

```yaml
files:
  - source: files/config.conf    # Path relative to module directory
    dest: ~/.config/app/config   # Destination (~ expands to home)
    type: symlink                # "symlink", "copy", or "template"

  - source: files/theme.tmpl
    dest: ~/.config/app/theme
    type: template               # Rendered as Go template
```

**File Types:**
- **symlink** - Creates symbolic link (default)
- **copy** - Copies file preserving permissions
- **template** - Renders as Go template before writing

### Prompts

Interactive questions to ask during installation:

```yaml
prompts:
  # Text input
  - key: api_key
    message: "Enter your API key:"
    type: input
    default: ""

  # Yes/No confirmation
  - key: enable_feature
    message: "Enable advanced features?"
    type: confirm
    default: "true"

  # Multiple choice
  - key: theme
    message: "Select a theme:"
    type: choice
    options:
      - dark
      - light
      - auto
    default: dark
```

Answers are available as environment variables in scripts: `$DOTFILES_PROMPT_KEY_NAME` (uppercase).

## Writing Scripts

### install.sh

Main installation logic. This is required.

```bash
#!/usr/bin/env bash
set -euo pipefail

# Helpers are automatically available
log_info "Installing mymodule..."

# Check if already installed
if pkg_installed mymodule; then
    log_info "mymodule already installed, skipping..."
    exit 0
fi

# Install package
pkg_install mymodule

# Additional setup
mkdir -p ~/.config/mymodule

# Success!
log_success "mymodule installed"
```

### OS-Specific Scripts (os/*.sh)

Optional scripts for platform-specific setup:

```bash
# os/macos.sh
#!/usr/bin/env bash
set -euo pipefail

log_info "Running macOS-specific setup..."

# macOS-specific commands
defaults write com.myapp theme -string "dark"
```

```bash
# os/ubuntu.sh
#!/usr/bin/env bash
set -euo pipefail

log_info "Running Ubuntu-specific setup..."

# Add PPA or configure apt
sudo add-apt-repository -y ppa:myapp/ppa
```

```bash
# os/arch.sh
#!/usr/bin/env bash
set -euo pipefail

log_info "Running Arch-specific setup..."

# AUR installation or Arch-specific config
```

### verify.sh

Optional post-installation verification:

```bash
#!/usr/bin/env bash
set -euo pipefail

log_info "Verifying mymodule installation..."

# Check command exists
if ! command -v mymodule &>/dev/null; then
    log_error "mymodule command not found"
    exit 1
fi

# Check config file
if [[ ! -f ~/.config/mymodule/config.conf ]]; then
    log_error "Config file not found"
    exit 1
fi

log_success "Verification passed"
```

## Available Helper Functions

All scripts have access to helper functions from `lib/helpers.sh`:

### Logging

```bash
log_info "Informational message"
log_warn "Warning message"
log_error "Error message"
log_success "Success message"
```

### OS Detection

```bash
if is_macos; then
    echo "Running on macOS"
elif is_ubuntu; then
    echo "Running on Ubuntu"
elif is_arch; then
    echo "Running on Arch Linux"
fi
```

### Environment Checks

```bash
if has_sudo; then
    sudo apt install package
fi

if is_interactive; then
    # Show interactive prompts
fi

if is_dry_run; then
    log_info "[dry-run] Would install package"
    exit 0
fi
```

### Package Management

```bash
# Check if installed
if pkg_installed git; then
    echo "Git is installed"
fi

# Install packages (skips if already installed)
pkg_install git curl wget

# Works across brew (macOS), apt (Ubuntu), pacman (Arch)
```

### File Operations

```bash
# Create symlink (backs up existing file)
link_file "$DOTFILES_MODULE_DIR/files/config" ~/.config/app/config

# Copy file (backs up existing file)
copy_file "$DOTFILES_MODULE_DIR/files/script.sh" ~/bin/script.sh
```

### Templates and Secrets

```bash
# Render template (calls back to Go CLI)
render_template "$DOTFILES_MODULE_DIR/files/config.tmpl" ~/.config/app/config

# Get secret from 1Password (calls back to Go CLI)
API_KEY=$(get_secret "op://vault/item/field")
```

### Interactive Prompts

```bash
# These respect --unattended flag automatically

# Text input
NAME=$(prompt_input "Enter your name:" "John Doe")

# Confirmation
if prompt_confirm "Enable feature?" "true"; then
    echo "Feature enabled"
fi

# Choice
THEME=$(prompt_choice "Select theme:" "dark" "light" "auto")
```

## Environment Variables

Your scripts receive these environment variables:

### System Information

```bash
$DOTFILES_OS          # Operating system: darwin, ubuntu, arch
$DOTFILES_ARCH        # Architecture: amd64, arm64
$DOTFILES_PKG_MGR     # Package manager: brew, apt, pacman
$DOTFILES_HAS_SUDO    # "true" or "false"
```

### Paths

```bash
$DOTFILES_HOME        # User's home directory
$DOTFILES_DIR         # Dotfiles repository path
$DOTFILES_BIN         # Path to dotfiles CLI binary
$DOTFILES_MODULE_DIR  # Current module directory
$DOTFILES_MODULE_NAME # Current module name
```

### Execution Context

```bash
$DOTFILES_INTERACTIVE # "true" if interactive terminal
$DOTFILES_DRY_RUN     # "true" in --dry-run mode
$DOTFILES_VERBOSE     # "true" in verbose mode
```

### User Configuration

```bash
$DOTFILES_USER_NAME   # From config.yml
$DOTFILES_USER_EMAIL  # From config.yml
$DOTFILES_USER_GITHUB_USER  # From config.yml
```

### Prompt Answers

Prompt answers are available as `DOTFILES_PROMPT_<KEY>` (uppercase):

```yaml
# module.yml
prompts:
  - key: theme
    message: "Select theme:"
    type: choice
    options: [dark, light]
    default: dark
```

```bash
# install.sh
THEME="${DOTFILES_PROMPT_THEME}"  # "dark" or "light"
```

## Working with Templates

Templates use Go's `text/template` syntax.

### Template Context

Templates have access to:

```go
.User.Name          // User's full name
.User.Email         // User's email
.User.GithubUser    // GitHub username
.OS                 // Operating system
.Arch               // Architecture
.Home               // Home directory
.DotfilesDir        // Dotfiles repository path
.Module.theme       // Module settings from config.yml
.Module.api_key     // Prompt answers
.Secrets.api_key    // Secrets from 1Password
.Env.PATH           // Environment variables
```

### Example Template

```
# files/gitconfig.tmpl
[user]
    name = {{ .User.Name }}
    email = {{ .User.Email }}

[github]
    user = {{ .User.GithubUser }}

{{- if eq .OS "darwin" }}
[credential]
    helper = osxkeychain
{{- else }}
[credential]
    helper = cache --timeout=900
{{- end }}

[init]
    defaultBranch = {{ .Module.default_branch | default "main" }}
```

### Template Functions

```go
{{ env "HOME" }}                    // Get environment variable
{{ default "value" .Module.key }}   // Default value if empty
{{ .User.Name | upper }}            // Uppercase
{{ .User.Name | lower }}            // Lowercase
{{ contains "substring" .OS }}      // String contains
{{ join "," .Module.features }}     // Join slice
{{ .Module.name | trimSpace }}      // Trim whitespace
```

## Dependencies

Declare dependencies to ensure modules run in the correct order:

```yaml
name: mymodule
dependencies:
  - git      # Git must be installed first
  - zsh      # Zsh must be installed first
```

Dependencies are:
- **Transitive** - If you depend on `git` and `git` depends on `ssh`, you automatically depend on `ssh` too
- **Resolved automatically** - The system uses topological sorting to determine execution order
- **Cycle-detected** - Circular dependencies are rejected with a clear error message

## Priority

Modules with the same dependencies run in priority order:

```yaml
priority: 50  # Lower numbers run first
```

**Default**: 100

**Examples:**
- 1password: 10 (runs very early, no dependencies)
- ssh: 20 (depends on 1password)
- git: 30 (depends on ssh)
- zsh: 40 (depends on git)
- neovim: 50 (depends on git)

Within the same priority level, modules are sorted alphabetically by name.

## Platform Support

### Limit to Specific Platforms

```yaml
os:
  - macos
  - ubuntu
```

If `os` is empty or omitted, the module runs on all platforms.

### Platform-Specific Logic

Use OS-specific scripts:

```
modules/mymodule/
├── install.sh       # Runs on all platforms
├── os/
│   ├── macos.sh    # Only runs on macOS
│   ├── ubuntu.sh   # Only runs on Ubuntu
│   └── arch.sh     # Only runs on Arch
```

Or use conditionals in install.sh:

```bash
if is_macos; then
    brew install mypackage
elif is_ubuntu; then
    sudo apt install mypackage
elif is_arch; then
    sudo pacman -S mypackage
fi
```

## Testing Your Module

### Dry Run

```bash
dotfiles install mymodule --dry-run -v
```

Shows what would happen without making changes.

### Verbose Output

```bash
dotfiles install mymodule -v
```

Shows detailed execution information.

### Reset State

To test reinstallation:

```bash
rm ~/.dotfiles/.state/mymodule.json
dotfiles install mymodule
```

### Integration Tests

Add tests in `test/integration/test_install.sh`:

```bash
# --- Test: MyModule verification ---
echo ""
echo "--- Test: MyModule verification ---"
assert_command_exists "mymodule is installed" "mymodule"
assert_file_exists "config exists" "$HOME/.config/mymodule/config.conf"
```

## Best Practices

### 1. Idempotency

Scripts should be safe to run multiple times:

```bash
# Good: Check before installing
if ! pkg_installed mypackage; then
    pkg_install mypackage
fi

# Bad: Always tries to install
pkg_install mypackage
```

### 2. Error Handling

Use `set -euo pipefail` to fail fast:

```bash
#!/usr/bin/env bash
set -euo pipefail  # Exit on error, undefined vars, pipe failures
```

### 3. Backup Files

The helpers automatically backup files:

```bash
# Automatically backs up ~/.zshrc if it exists
link_file "$DOTFILES_MODULE_DIR/files/zshrc" ~/.zshrc
```

Backups are created with `.backup-TIMESTAMP` suffix.

### 4. Minimal Dependencies

Only declare direct dependencies:

```bash
# Good
dependencies:
  - git

# Bad: zsh already depends on git, transitive deps are automatic
dependencies:
  - git
  - ssh  # ssh is a dependency of git
```

### 5. Clear Logging

Use descriptive log messages:

```bash
log_info "Installing neovim package..."
log_info "Creating ~/.config/nvim directory..."
log_info "Installing packer.nvim plugin manager..."
log_success "Neovim configured successfully"
```

### 6. Respect Flags

Check dry-run and other flags:

```bash
if is_dry_run; then
    log_info "[dry-run] Would install mypackage"
    exit 0
fi
```

### 7. Document Prompts

Provide clear prompt messages and sensible defaults:

```yaml
prompts:
  - key: theme
    message: "Which color theme would you like? (dark recommended for most terminals)"
    type: choice
    options: [dark, light, auto]
    default: dark
```

## Examples

See existing modules for reference:
- [ssh module](../modules/ssh/) - Simple module with templates and secrets
- [git module](../modules/git/) - Module with OS-specific scripts
- [zsh module](../modules/zsh/) - Complex module with external dependencies (Zinit)
- [neovim module](../modules/neovim/) - Minimal module with symlinks

## Next Steps

- [Module Reference](module-reference.md) - Complete module.yml schema
- [Shell Helpers](shell-helpers.md) - All available helper functions
- [Templates](templates.md) - Advanced template usage
- [Testing](ci-cd.md) - Add integration tests for your module
