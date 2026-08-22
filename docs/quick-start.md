---
title: "Quick Start"
---

# Quick Start

Get up and running with the dotfiles management system in minutes.

## Installation

```bash
curl -sfL https://raw.githubusercontent.com/garygentry/dotfiles/main/bootstrap.sh | bash
```

## Common Tasks

### List Available Modules

```bash
dotfiles list
```

Example output:
```
Name         Description                          OS                 Status
-----------  -----------------------------------  -----------------  -------------
1password    Install and configure 1Password CLI  macos,ubuntu,arch  not installed
ssh          Configure SSH keys and settings      macos,ubuntu,arch  installed
git          Configure git with SSH signing       macos,ubuntu,arch  installed
zsh          Install and configure Zsh            macos,ubuntu,arch  installed
neovim       Install Neovim and symlink config    macos,ubuntu,arch  not installed
```

> The listing above is truncated — `dotfiles list` shows all ~30 modules. When a
> [content overlay](content-overlay.md) contributes modules, an extra **Source** column
> appears tagging each as `built-in`, `override`, or `custom`.

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

Modules are listed in **dependency order**. For example, `git` depends on `ssh`, so `ssh` runs first.

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

- **developer** - Full development environment (a curated module set)
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

The committed `config.yml` ships generic engine defaults (`secrets.provider: noop`, empty
`user.*`). To personalize, prefer a **content overlay** — an optional directory
(`$DOTFILES_CONTENT_DIR`) holding your own `config.yml`/`profiles/`/`modules/` that is
deep-merged over the repo, so the engine stays generic and your identity/secrets live in
your own repo. See the [Content Overlay guide](content-overlay.md) and
[`config.overlay.example.yml`](https://github.com/garygentry/dotfiles/blob/main/config.overlay.example.yml).

Editing `~/.dotfiles/config.yml` directly still works for a single machine:

```bash
vim ~/.dotfiles/config.yml
```

The config shape (defaults shown):

```yaml
profile: minimal           # conservative default (git, zsh); developer is opt-in

secrets:
  provider: noop           # opt into "1password" from your overlay

user:
  name: ""
  email: ""
  github_user: ""

modules:
  ssh:
    key_type: ed25519
    key_source: generate   # generate | agent | 1password | none
  git:
    default_branch: main
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
- [Idempotence](idempotence.md) - How re-runs are handled safely
