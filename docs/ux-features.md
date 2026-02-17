# UX Features Guide

This guide covers the modern terminal user experience features introduced in version 2.1.0.

## Overview

The dotfiles CLI provides a polished, professional terminal experience with:
- **Grid-based module selection** - Compact, navigable interface
- **Progress tracking** - Real-time progress bars with time estimates
- **Smart output handling** - Compact by default, detailed on errors
- **Catppuccin Mocha theming** - Beautiful, consistent color scheme

## Grid-Based Module Selection

### What It Looks Like

When you run `dotfiles install` without specifying modules, you'll see:

```
┌─ Select modules to install ──────────────────────────────────────┐
│                                                                   │
│  [x] 1password      [x] git            [ ] neovim                 │
│  [ ] golang         [x] docker         [ ] python                 │
│  [x] fish           [ ] tmux           [ ] zsh                    │
│  [ ] aws            [x] kubernetes     [ ] terraform              │
│  ...                                                               │
│                                                                   │
│  Navigate: ↑/↓/←/→  Toggle: Space  Select All: A  Continue: Enter │
│  Preview: git - Configure git with SSH signing and useful defaults│
└───────────────────────────────────────────────────────────────────┘
```

### Key Features

- **Grid Layout**: Displays 3-5 columns based on terminal width
- **Live Preview**: Bottom pane shows full description of highlighted module
- **Pre-selection**: Modules from your profile are pre-checked
- **Keyboard Navigation**: Intuitive controls for fast selection

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up one row |
| `↓` / `j` | Move cursor down one row |
| `←` / `h` | Move cursor left one column |
| `→` / `l` | Move cursor right one column |
| `Space` | Toggle selection of current module |
| `A` | Select all modules |
| `N` | Select none (clear all selections) |
| `Enter` | Confirm selection and continue |
| `Esc` / `Q` | Cancel and exit |

### Benefits

- **Space Efficient**: 28+ modules fit in ~10 lines (64% reduction)
- **Better Overview**: See all modules at once without scrolling
- **Faster Selection**: Grid layout + keyboard shortcuts = speed
- **Clear Feedback**: Live preview shows exactly what each module does

## Progress Tracking

### Real-Time Progress Bar

During installation, you'll see:

```
┌─────────────────────────────────────────────────────────────┐
│ Installing 10 modules  ████████████░░░░░░░░░░  6/10 (60%)  │
│ Current: docker • Elapsed: 1m23s • Est. remaining: ~1m10s  │
└─────────────────────────────────────────────────────────────┘
```

### What It Shows

- **Progress Bar**: Visual indicator of completion (filled vs. unfilled blocks)
- **Fraction**: Current module / total modules (e.g., "6/10")
- **Percentage**: Completion percentage (e.g., "60%")
- **Current Module**: Name of module currently being installed
- **Elapsed Time**: How long the installation has been running
- **Estimated Remaining**: Predicted time to completion (based on average module duration)

### Completion Summary

When all modules finish:

```
┌─────────────────────────────────────────────────────────────┐
│ ✓ Installation complete  ██████████████████  10/10 (100%)  │
│ Success: 9 • Failed: 1 • Skipped: 2 • Time: 2m35s           │
└─────────────────────────────────────────────────────────────┘

✓ git (2s)
✓ docker (12s)
✓ neovim (5s)
✗ golang (failed: download timeout)
⊘ python (skipped: already up-to-date)
```

### Benefits

- **Clear Progress**: Always know how far along you are
- **Time Awareness**: Estimates help you plan ("~2 minutes remaining")
- **Quick Summary**: See results at a glance
- **Per-Module Timing**: Identify slow modules

## Smart Output Handling

### Compact Mode (Default)

**During execution:**
```
⠋ Installing docker...  [5s]
```

**On success:**
```
✓ Executed install.sh (12s)
```

**On failure (auto-expanded):**
```
✗ Failed docker: install script error: exit status 1 [8s]

  ╭─ Script output (install.sh) ──────────────────────╮
  │ E: Unable to locate package docker-ce             │
  │ Reading package lists... Done                     │
  │ Building dependency tree... Done                  │
  ╰────────────────────────────────────────────────────╯
```

### Verbose Mode (`--verbose` or `-v`)

Streams all output in real-time:

```
[DEBUG] Running script: install.sh
• Installing Docker prerequisites...
• Adding Docker GPG key...
• Adding Docker repository...
• Installing docker-ce...
✓ Docker installed successfully
```

### Output Modes Comparison

| Mode | When to Use | Output Style | Best For |
|------|-------------|--------------|----------|
| **Compact** (default) | Normal installation | Spinners + summaries | Clean, minimal output |
| **Verbose** (`-v`) | Debugging, troubleshooting | Full streaming | Understanding what's happening |
| **Non-TTY** (auto) | CI/CD, scripts, pipes | Plain text | Log aggregation, automation |

### Smart Features

#### 1. Sudo Detection

Scripts using `sudo` automatically stream output to preserve password prompts:

```bash
# Script contains: sudo apt-get install ...
# Result: Output streams live (no buffering)
```

#### 2. Auto-Expanding Errors

Failed scripts automatically show their output:

```
✗ Failed installation

  ╭─ Script output (last 30 lines) ───────────────╮
  │ Error: Connection timeout                     │
  │ Failed to download package                    │
  ╰────────────────────────────────────────────────╯
```

#### 3. Pattern Recognition

Extracts operations from script output:

```bash
# Script: log_info "Installing packages..."
# CLI shows: • Installing packages...
```

Recognized patterns:
- `• Message` - Information (from `log_info`)
- `✓ Message` - Success (from `log_success`)
- `Installing ...` - Package installation (from `pkg_install`)

### Benefits

- **Reduced Noise**: 60-75% less output (200+ lines → 50-80 lines)
- **Immediate Error Visibility**: Failures show context automatically
- **Preserved Interactivity**: Sudo prompts still work
- **Debug Capability**: Verbose mode available when needed

## Non-TTY Mode (CI/CD)

### Automatic Detection

When stdin is not a terminal (e.g., piped input), the CLI automatically uses plain text output:

```bash
# Piped execution
curl ... | bash

# CI/CD environments
dotfiles install --unattended
```

### Plain Text Output

```
[INFO] Detecting system...
[OK] System: ubuntu/amd64 (pkg: apt)
[MULTISELECT] Select modules to install (using defaults: git, zsh)
[PROGRESS] 1/2: git
[OK] Executed install.sh
[PROGRESS] 2/2: zsh
[OK] Executed install.sh
[SUMMARY] Completed in 1m23s: 2 succeeded, 0 failed, 0 skipped
```

### Features

- **No ANSI codes**: Plain text suitable for log files
- **No interactive elements**: Grid selector, progress bars disabled
- **Default selections**: Uses profile or specified modules
- **Parseable output**: Structured for log aggregation

## Color Scheme

All UI elements use the **Catppuccin Mocha** color palette:

| Element | Color | Use Case |
|---------|-------|----------|
| Blue (`#89b4fa`) | Informational | Titles, borders, info messages |
| Green (`#a6e3a1`) | Success | Checkmarks, success messages, progress bars |
| Red (`#f38ba8`) | Errors | Error messages, failure indicators |
| Yellow (`#f9e2af`) | Warnings | Warning messages, skip indicators |
| Mauve (`#cba6f7`) | Highlights | Spinner, cursor, selected items |
| Text (`#cdd6f4`) | Primary text | Main content |
| Subtext (`#a6adc8`) | Secondary text | Descriptions, timestamps |
| Overlay (`#6c7086`) | Muted | Help text, borders |

## Accessibility

- **Keyboard-only**: All features fully accessible via keyboard
- **No mouse required**: Grid selector, prompts work without mouse
- **Screen reader friendly**: Non-TTY mode provides clean text output
- **Terminal width aware**: Grid adapts from 80 to 200+ columns

## Performance

- **Minimal overhead**: UI rendering is fast (< 1ms per update)
- **I/O bound**: Execution time dominated by script execution
- **Small memory footprint**: Output buffering typically < 1MB per script
- **No external dependencies**: All UI built with standard libraries

## Troubleshooting

### Grid selector shows garbled output

**Problem**: Terminal doesn't support ANSI escape codes

**Solution**: Use explicit module names:
```bash
dotfiles install git zsh neovim
```

### Progress bar flickers

**Problem**: Terminal refresh rate mismatch

**Solution**: This is cosmetic and doesn't affect functionality. The progress bar still works correctly.

### Want to see all output

**Problem**: Compact mode hides script output

**Solution**: Use verbose mode:
```bash
dotfiles install -v
```

### Output buffering breaks sudo prompts

**Problem**: Can't enter sudo password

**Solution**: The CLI automatically detects `sudo` in scripts and streams output. If detection fails:
```bash
dotfiles install -v  # Force streaming mode
```

### CI/CD shows ANSI codes

**Problem**: Colors appear as raw escape codes in logs

**Solution**: The CLI auto-detects non-TTY. If it fails:
```bash
dotfiles install --unattended 2>&1 | tee install.log
```

## Examples

### Interactive Installation

```bash
# Standard interactive flow
dotfiles install

# 1. Grid selector appears
# 2. Select modules with arrow keys + space
# 3. Press Enter to confirm
# 4. Watch progress bar
# 5. Review completion summary
```

### Automated Installation

```bash
# CI/CD: Use defaults, plain text output
dotfiles install --unattended

# Specific modules with verbose output
dotfiles install git zsh -v

# Dry run to preview
dotfiles install --dry-run
```

### Debugging Failed Module

```bash
# 1. Install fails, error auto-expands
# 2. Review error output
# 3. Re-run with verbose mode:
dotfiles install failed-module -v

# 4. See full script output
# 5. Identify and fix issue
```

## See Also

- [CLI Reference](cli-reference.md) - Complete command documentation
- [Quick Start](quick-start.md) - Getting started guide
- [Troubleshooting](troubleshooting.md) - Common issues and solutions
