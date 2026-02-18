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

When you run this command, you'll see a **grid-based module selector** where you can:
- Use **arrow keys** (↑↓←→) to navigate between modules
- Press **Space** to toggle module selection
- Press **A** to select all modules, **N** to select none
- See a live preview of the currently highlighted module
- Press **Enter** to start installation

During installation, a **progress bar** shows:
- How many modules have been installed (e.g., "3/10 60%")
- Current module being installed
- Elapsed time and estimated time remaining

### Install Specific Modules

```bash
dotfiles install git zsh neovim
```

This skips the interactive selector and directly installs the specified modules.

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

**Verbose mode** streams all script output in real-time, showing:
- Detailed execution logs
- All script output (stdout/stderr)
- Debug information

Use verbose mode when:
- Troubleshooting module failures
- Understanding what scripts are doing
- Debugging configuration issues

**Note**: In compact mode (default), script output is buffered and only shown on errors. Verbose mode disables this buffering and streams everything live.

## Understanding the Installation Flow

### 1. Module Selection (Interactive Mode)

When you run `dotfiles install` without specifying modules, you'll see a grid selector:

```
┌─ Select modules to install ──────────────────────────────────┐
│                                                               │
│  [x] 1password      [x] git            [ ] neovim            │
│  [ ] golang         [x] docker         [ ] python            │
│  [x] fish           [ ] tmux           [ ] zsh               │
│                                                               │
│  Navigate: ↑/↓/←/→  Toggle: Space  Select All: A  Continue: Enter
│  Preview: git - Configure git with SSH signing and defaults  │
└───────────────────────────────────────────────────────────────┘
```

### 2. Execution Plan

After selection, you'll see the execution plan:

```
Execution Plan:
  Install (3):
    1. 1password - Install and configure 1Password CLI
    2. git - Configure Git with SSH signing
    3. docker - Install Docker and Docker Compose

OS: ubuntu 22.04 (amd64)
Package Manager: apt
```

Modules are listed in **dependency order**. For example, `git` depends on `1password`, so `1password` runs first.

### 3. Progress Tracking

During installation, a progress bar shows overall status:

```
┌─────────────────────────────────────────────────────────────┐
│ Installing 3 modules  ████████░░░░  2/3 (67%)              │
│ Current: git • Elapsed: 1m15s • Est. remaining: ~38s        │
└─────────────────────────────────────────────────────────────┘

⠋ Installing git...
```

### 4. Completion Summary

After all modules complete:

```
┌─────────────────────────────────────────────────────────────┐
│ ✓ Installation complete  ██████████████  3/3 (100%)        │
│ Success: 3 • Failed: 0 • Skipped: 0 • Time: 2m5s            │
└─────────────────────────────────────────────────────────────┘
```

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

- [Creating Modules](creating-modules.md) - Build your own modules
- [CLI Reference](cli-reference.md) - Complete command documentation
- [Idempotence](IDEMPOTENCE.md) - How re-runs are handled safely
