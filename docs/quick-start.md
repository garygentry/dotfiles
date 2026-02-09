# Quick Start

Get up and running with the dotfiles management system in minutes.

## Installation

```bash
curl -sfL https://raw.githubusercontent.com/USER/dotfiles/main/bootstrap.sh | bash
```

## Common Tasks

### List Available Modules

```bash
dotfiles list
```

Example output:
```
Available Modules:

Name         Description                          OS           Status
──────────── ──────────────────────────────────── ──────────── ────────────────
1password    Install and configure 1Password CLI  all          not installed
ssh          Configure SSH keys and settings      all          installed
git          Configure Git with SSH signing       all          installed
zsh          Install and configure Zsh + Zinit    all          installed
neovim       Install Neovim and symlink config    all          not installed
```

### Install All Modules

```bash
dotfiles install
```

### Install Specific Modules

```bash
dotfiles install git zsh neovim
```

### Install with a Profile

```bash
dotfiles install --profile minimal
```

### Dry Run (Preview Changes)

```bash
dotfiles install --dry-run
```

This shows what would be installed without making any changes.

### Unattended Installation

```bash
dotfiles install --unattended
```

Uses default answers for all prompts. Useful for automation.

### Verbose Output

```bash
dotfiles install -v
```

Shows detailed information about what's happening.

## Understanding the Execution Plan

When you run `dotfiles install`, you'll see an execution plan:

```
Execution Plan:
  1. 1password (v1.0.0) - Install and configure 1Password CLI
  2. ssh (v1.0.0) - Configure SSH keys and settings
  3. git (v1.0.0) - Configure Git with SSH signing
  4. zsh (v1.0.0) - Install and configure Zsh + Zinit
  5. neovim (v1.0.0) - Install Neovim and symlink config

OS: ubuntu 22.04 (amd64)
Package Manager: apt
```

Modules are listed in dependency order. For example, `ssh` depends on `1password`, so `1password` runs first.

## Working with Profiles

Profiles let you install predefined sets of modules.

### Available Profiles

- **developer** - Full development environment (all modules)
- **minimal** - Lightweight setup (git, zsh)
- **test** - Testing configuration

### Set Default Profile

Edit `config.yml`:

```yaml
profile: minimal
```

### Create Custom Profile

Create `profiles/custom.yml`:

```yaml
modules:
  - git
  - neovim
```

Then install:

```bash
dotfiles install --profile custom
```

## Configuration

### Edit Main Configuration

```bash
vim ~/.dotfiles/config.yml
```

Example configuration:

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
  zsh:
    theme: powerlevel10k
```

### Module-Specific Settings

Each module can have custom settings under the `modules` key. These are available in module scripts and templates.

## Interactive Prompts

During installation, modules may ask questions:

```
? Which SSH key type would you like to use?
  > ed25519 (recommended)
    rsa
```

Use arrow keys to select and press Enter. In `--unattended` mode, defaults are used automatically.

## Checking Installation Status

View the status of installed modules:

```bash
dotfiles list
```

The **Status** column shows:
- `installed` - Module is installed
- `not installed` - Module not yet installed
- `failed` - Last installation failed

## Troubleshooting

### View Verbose Output

```bash
dotfiles install -v
```

### Check State Files

State is stored in `~/.dotfiles/.state/`:

```bash
cat ~/.dotfiles/.state/git.json
```

Example:
```json
{
  "name": "git",
  "version": "1.0.0",
  "status": "installed",
  "installed_at": "2024-02-09T10:30:00Z",
  "os": "ubuntu"
}
```

### Reset a Module

To reinstall a module, remove its state file:

```bash
rm ~/.dotfiles/.state/git.json
dotfiles install git
```

## Next Steps

- [Configuration Guide](configuration.md) - Detailed configuration options
- [Modules Overview](modules.md) - Learn about each module
- [Creating Modules](creating-modules.md) - Build your own modules
- [CLI Reference](cli-reference.md) - Complete command documentation
