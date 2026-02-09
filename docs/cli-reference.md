# CLI Reference

Complete reference for the dotfiles command-line interface.

## Global Flags

These flags are available for all commands:

```
--help, -h       Show help for command
--verbose, -v    Enable verbose output
--dry-run        Show what would be done without making changes
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
  "installed_at": "2024-02-09T10:30:00Z",
  "updated_at": "2024-02-09T10:30:00Z",
  "os": "ubuntu",
  "error": ""
}
```

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
