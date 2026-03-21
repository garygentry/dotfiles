# Plan: tmux Status Bar Cleanup

## Context

The current tmux status bar has the session name and hostname on the right side, with nothing on the left. The user wants the machine hostname in a styled pill on the left (matching the window tab aesthetic), the session name removed from the right, and the window list truly centered so it doesn't shift when dynamic content (clock, etc.) changes length.

tmux 3.2a is installed, which supports `status-justify absolute-centre` — this anchors the window list to the terminal midpoint regardless of left/right content width.

## File to Modify

`modules/tmux/tmux.conf`

## Changes

### 1. Move host to left, remove session+host from right (lines 106–107)

```tmux
# Before
set -g @catppuccin_status_modules_right "session host date_time"
set -g @catppuccin_status_modules_left ""

# After
set -g @catppuccin_status_modules_right "date_time"
set -g @catppuccin_status_modules_left "host"
```

### 2. Replace `@catppuccin_status_justify` with native `status-justify absolute-centre` (line 109)

`@catppuccin_status_justify` is not a recognized catppuccin option in the installed version. Replace with the native tmux setting that actually works and uses absolute centering:

```tmux
# Before
set -g @catppuccin_status_justify "centre"

# After
set -g status-justify absolute-centre
```

`absolute-centre` (tmux ≥ 3.2) centers the window list relative to terminal width, not the gap between left/right — so it never shifts when the clock ticks or path changes.

### 3. Remove dead commented-out load line (line 112)

```tmux
# Remove this line entirely:
# run ~/.config/tmux/plugins/catppuccin/tmux/catppuccin.tmux
```

TPM already loads the plugin via the `@plugin` declaration; this line is a leftover from manual installation instructions.

## Verification

After applying changes, reload config in a live tmux session:

```
<prefix> r
```

Confirm:
- Hostname appears in a styled pill on the left
- Session name is gone from the right; only the clock remains
- Window tabs stay visually centered when the clock updates (watch for a minute or resize the terminal to confirm)
