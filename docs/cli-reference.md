# CLI Reference

Complete reference for the dotfiles command-line interface.

## Global Flags

These flags are available for all commands:

```
--help, -h       Show help for command
--verbose, -v    Enable verbose output
--dry-run        Show what would be done without making changes
--log-json       Output logs in JSON format (for log aggregation)
--unattended     Run without prompts, using defaults (ideal for CI/CD and IaC)
```

## Commands

### dotfiles install

Install modules with dependency resolution.

```bash
dotfiles install [modules...] [flags]
```

**Arguments:**
- `modules` - Optional list of modules to install. If omitted, installs all modules (or those in the configured profile).

**Flags:**
```
--profile string      Use a specific profile (e.g., minimal, developer)
--unattended         Run without prompts, use default answers
--fail-fast          Stop on first module failure (default: continue)
-v, --verbose        Show detailed output including script execution
--dry-run            Preview changes without applying them
```

**Examples:**

```bash
# Install all modules
dotfiles install

# Install specific modules
dotfiles install git zsh neovim

# Use a profile
dotfiles install --profile minimal

# Preview without changes
dotfiles install --dry-run

# Automated installation (no prompts)
dotfiles install --unattended

# Verbose output for debugging
dotfiles install -v

# Stop on first error
dotfiles install --fail-fast
```

**Output:**

```
System Information:
  OS: ubuntu 22.04
  Architecture: amd64
  Package Manager: apt

Execution Plan:
  1. 1password (v1.0.0) - Install and configure 1Password CLI
  2. ssh (v1.0.0) - Configure SSH keys and settings
  3. git (v1.0.0) - Configure Git with SSH signing
  4. zsh (v1.0.0) - Install and configure Zsh + Zinit
  5. neovim (v1.0.0) - Install Neovim and symlink config

? Which SSH key type would you like to use?
  > ed25519 (recommended)
    rsa

✓ 1password - Install and configure 1Password CLI
✓ ssh - Configure SSH keys and settings
✓ git - Configure Git with SSH signing
✓ zsh - Install and configure Zsh + Zinit
✓ neovim - Install Neovim and symlink config

Summary:
  ✓ 5 installed
  ✗ 0 failed
  ⊘ 0 skipped

All modules installed successfully!
```

### dotfiles list

List all available modules with their status.

```bash
dotfiles list [flags]
```

**Flags:**
```
-v, --verbose        Show additional module details
```

**Examples:**

```bash
# List modules
dotfiles list

# Verbose output with tags and dependencies
dotfiles list -v
```

**Output:**

```
Available Modules:

Name         Description                           OS           Status
──────────── ───────────────────────────────────── ──────────── ────────────────
1password    Install and configure 1Password CLI   all          installed
ssh          Configure SSH keys and settings       all          installed
git          Configure Git with SSH signing        all          installed
zsh          Install and configure Zsh + Zinit     all          not installed
neovim       Install Neovim and symlink config     all          failed

5 modules available
```

**Status Values:**
- `installed` - Module is currently installed
- `not installed` - Module has not been installed
- `failed` - Last installation attempt failed

### dotfiles status

Show status of installed modules with detailed information.

```bash
dotfiles status [flags]
```

**Flags:**
```
-v, --verbose        Show operation history and full details
```

**Examples:**

```bash
# Show installed modules
dotfiles status

# Show detailed information including operation history
dotfiles status -v
```

**Output:**

```
Installed Modules:

Name         Version    Status      Installed             OS
──────────── ────────── ─────────── ───────────────────── ──────────
git          1.0.0      installed   2026-02-10 10:30:00   ubuntu
zsh          1.0.0      installed   2026-02-10 10:31:15   ubuntu
neovim       1.0.0      failed      2026-02-10 10:32:00   ubuntu

3 modules installed (1 failed)
```

**Verbose Output:**
Shows operation history for rollback tracking:
- Files deployed (created, modified, symlinked)
- Directories created
- Scripts executed
- Packages installed

### dotfiles uninstall

Uninstall modules and rollback their changes.

```bash
dotfiles uninstall <modules...> [flags]
```

**Arguments:**
- `modules` - One or more modules to uninstall (required)

**Flags:**
```
--force              Skip confirmation prompts and continue on errors
--unattended         Skip confirmation prompts (for automated environments)
--dry-run            Preview rollback plan without executing
-v, --verbose        Show detailed rollback information
```

**Examples:**

```bash
# Uninstall a module
dotfiles uninstall git

# Uninstall multiple modules
dotfiles uninstall git zsh neovim

# Preview what would be uninstalled
dotfiles uninstall git --dry-run

# Unattended mode (auto-confirm)
dotfiles uninstall git --unattended

# Force uninstall (no prompts, continue on errors)
dotfiles uninstall git --force

# Verbose uninstall with detailed output
dotfiles uninstall git -v
```

**Output:**

```
Uninstalling git...

Rollback plan (5 operations):
  1. Remove: /home/user/.gitconfig
  2. Restore /home/user/.bashrc from /home/user/.bashrc.backup
  3. Remove directory: /home/user/.config/git
  4. Package was installed: git (manual removal may be needed)
  5. Script was executed: install.sh (manual cleanup may be needed)

? Proceed with uninstall of git? [y/N]: y

✓ Removed /home/user/.gitconfig
✓ Restored /home/user/.bashrc
✓ Removed directory /home/user/.config/git
ℹ Package git installed (manual removal may be needed)
ℹ Script install.sh executed (manual cleanup may be needed)

✓ Uninstalled git successfully
```

**Rollback Operations:**
- **Created files/symlinks**: Removed
- **Modified files**: Restored from backup (if available)
- **Created directories**: Removed if empty
- **Scripts**: Informational only, not automatically reversed
- **Packages**: Informational only, manual removal needed

**Exit Codes:**
- `0` - All modules uninstalled successfully
- `1` - One or more modules failed to uninstall (unless `--force` used)

### dotfiles new

Generate a new module skeleton with standard structure.

```bash
dotfiles new <module-name> [flags]
```

**Arguments:**
- `module-name` - Name of the module to create (lowercase alphanumeric with hyphens)

**Flags:**
```
--priority int           Module priority (1-100, default: 50)
--depends strings        Comma-separated list of dependencies
--os strings             Comma-separated list of supported OSes (default: all)
--description string     Module description
```

**Examples:**

```bash
# Create a basic module
dotfiles new tmux

# Create with priority and dependencies
dotfiles new my-module --priority 35 --depends git,zsh

# Create with OS restrictions
dotfiles new mac-only --os darwin

# Create with full metadata
dotfiles new advanced \
  --priority 40 \
  --depends git \
  --os ubuntu,arch \
  --description "Advanced configuration module"
```

**Generated Structure:**

```
modules/my-module/
├── module.yml          # Module configuration
├── install.sh          # Installation script
├── verify.sh           # Verification script (optional)
├── os/                 # OS-specific scripts (optional)
│   ├── ubuntu.sh
│   ├── macos.sh
│   └── arch.sh
├── files/              # Template/config files (placeholder)
└── README.md           # Module documentation
```

**Generated module.yml:**
```yaml
name: my-module
version: 1.0.0
description: TODO: Add description
priority: 50
os:
  - all
dependencies: []
requires: []
files: []
prompts: []
tags: []
```

**Next Steps After Generation:**
1. Edit `module.yml` to configure module
2. Implement `install.sh` with installation logic
3. Add configuration files to `files/` directory
4. Update README.md with documentation
5. Test with `dotfiles install my-module --dry-run`

### dotfiles get-secret

Retrieve a secret from the configured secrets provider. This is an internal command typically called by module scripts via the `get_secret` helper function.

```bash
dotfiles get-secret --ref <reference> [flags]
```

**Flags:**
```
--ref string         Secret reference (e.g., op://vault/item/field)
```

**Examples:**

```bash
# Get a secret from 1Password
dotfiles get-secret --ref "op://Private/GitHub/token"

# Typically used in module scripts:
API_KEY=$(get_secret "op://Private/API/key")
```

**Output:**
```
secret-value-here
```

**Error Cases:**
- Secrets provider not configured
- Not authenticated
- Secret not found
- Invalid reference format

### dotfiles render-template

Render a Go template file. This is an internal command typically called by module scripts via the `render_template` helper function.

```bash
dotfiles render-template --src <source> --dest <destination> [flags]
```

**Flags:**
```
--src string         Source template file path
--dest string        Destination file path
```

**Examples:**

```bash
# Render a template
dotfiles render-template --src config.tmpl --dest ~/.config/app/config

# Typically used in module scripts:
render_template "$DOTFILES_MODULE_DIR/files/config.tmpl" ~/.config/app/config
```

**Template Context:**

Templates have access to:
- `.User.Name`, `.User.Email`, `.User.GithubUser`
- `.OS`, `.Arch`, `.Home`, `.DotfilesDir`
- `.Module.*` - Module settings and prompt answers
- `.Secrets.*` - Retrieved secrets
- `.Env.*` - Environment variables

**Template Functions:**
- `env "VAR"` - Get environment variable
- `default "val1" "val2"` - First non-empty value
- `upper`, `lower` - Case conversion
- `contains "substr"` - String contains
- `join ","` - Join slice
- `trimSpace` - Trim whitespace

### dotfiles version

Show version information.

```bash
dotfiles version
```

**Output:**
```
dotfiles version 1.0.0
```

### dotfiles help

Show help for any command.

```bash
dotfiles help [command]
```

**Examples:**

```bash
# Show general help
dotfiles help

# Show help for install command
dotfiles help install
```

## Exit Codes

The CLI uses standard exit codes:

- `0` - Success
- `1` - General error
- `2` - Invalid arguments or flags

**Module Installation:**
- If any module fails during `install` and `--fail-fast` is not set, the command continues and returns `1` at the end
- With `--fail-fast`, returns `1` immediately on first failure

## Environment Variables

These environment variables affect CLI behavior:

### Configuration

```bash
DOTFILES_DIR          # Override dotfiles directory (default: ~/.dotfiles)
DOTFILES_PROFILE      # Override profile from config.yml
```

### Execution Context

```bash
DOTFILES_INTERACTIVE  # Force interactive/non-interactive mode
DOTFILES_DRY_RUN      # Force dry-run mode
DOTFILES_VERBOSE      # Force verbose output
```

### Module Scripts

Module scripts receive many more environment variables. See [Environment Variables](environment-variables.md) for a complete list.

## Configuration Files

### config.yml

Main configuration file at `~/.dotfiles/config.yml`:

```yaml
profile: developer

secrets:
  provider: 1password
  account: my.1password.com

user:
  name: "Your Name"
  email: "your.email@example.com"
  github_user: "yourusername"

modules:
  ssh:
    key_type: ed25519
  git:
    default_branch: main
```

### Profile Files

Profile definitions in `~/.dotfiles/profiles/*.yml`:

```yaml
# profiles/minimal.yml
modules:
  - git
  - zsh
```

### State Files

Module state tracked in `~/.dotfiles/.state/*.json`:

```json
{
  "name": "git",
  "version": "1.0.0",
  "status": "installed",
  "installed_at": "2026-02-09T10:30:00Z",
  "updated_at": "2026-02-09T10:30:05Z",
  "os": "ubuntu",
  "checksum": "",
  "operations": [
    {
      "type": "script_run",
      "action": "executed",
      "path": "/home/user/.dotfiles/modules/git/install.sh",
      "timestamp": "2026-02-09T10:30:01Z"
    },
    {
      "type": "file_deploy",
      "action": "symlinked",
      "path": "/home/user/.gitconfig",
      "timestamp": "2026-02-09T10:30:02Z",
      "metadata": {
        "source": "/home/user/.dotfiles/modules/git/gitconfig",
        "type": "symlink"
      }
    },
    {
      "type": "dir_create",
      "action": "created",
      "path": "/home/user/.config/git",
      "timestamp": "2026-02-09T10:30:03Z"
    }
  ]
}
```

**Operations Field** (added in v1.1.0):
Tracks all operations for rollback capability. See [Rollback Guide](rollback-guide.md) for details.

## Debugging

### Verbose Output

Use `-v` flag to see detailed execution:

```bash
dotfiles install -v
```

Shows:
- System detection details
- Module discovery process
- Dependency resolution steps
- Script output (stdout/stderr)
- File operations
- State changes

### Dry Run

Preview changes without applying them:

```bash
dotfiles install --dry-run
```

Shows:
- Execution plan
- What scripts would run
- What files would be deployed
- No actual changes made

### Check State

View module installation state:

```bash
# List all state files
ls -la ~/.dotfiles/.state/

# View specific module state
cat ~/.dotfiles/.state/git.json | jq .
```

### Reset Module

Remove state to force reinstall:

```bash
rm ~/.dotfiles/.state/module-name.json
dotfiles install module-name
```

## Automation

### CI/CD Usage

Use `--unattended` flag for non-interactive environments:

```bash
# GitHub Actions, Jenkins, etc.
dotfiles install --unattended
```

This:
- Uses default answers for all prompts
- Skips interactive confirmations
- Suitable for automation

### Scripting

The CLI is designed to be scriptable:

```bash
#!/bin/bash
set -euo pipefail

# Bootstrap new system
curl -sfL https://url/to/bootstrap.sh | bash

# Install specific modules
~/.dotfiles/bin/dotfiles install --unattended git zsh neovim

# Verify installation
~/.dotfiles/bin/dotfiles list | grep -q "git.*installed"
```

## Common Workflows

### Initial Setup

```bash
# Run bootstrap
curl -sfL https://url/to/bootstrap.sh | bash

# Configure user settings
vim ~/.dotfiles/config.yml

# Install all modules
dotfiles install
```

### Add Module

```bash
# Install new module
dotfiles install neovim

# Check status
dotfiles list
```

### Update Configuration

```bash
# Edit config
vim ~/.dotfiles/config.yml

# Reinstall to apply changes
rm ~/.dotfiles/.state/module-name.json
dotfiles install module-name
```

### Troubleshooting

```bash
# Run with verbose output
dotfiles install module-name -v

# Check state for errors
cat ~/.dotfiles/.state/module-name.json

# Reset and retry
rm ~/.dotfiles/.state/module-name.json
dotfiles install module-name -v
```

## See Also

- [Quick Start](quick-start.md) - Getting started guide
- [Configuration](configuration.md) - Configuration options
- [Environment Variables](environment-variables.md) - Complete environment variable reference
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
