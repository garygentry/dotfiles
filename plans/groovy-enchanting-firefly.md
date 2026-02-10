# Plan: Add Starship Prompt Module

## Context
Adding a new module to install and configure [Starship](https://starship.rs/), a cross-shell prompt. It will be installed via Starship's official installer script and included in the developer profile.

## Files to Create

### 1. `modules/starship/module.yml`
- `name: starship`, priority 45 (after zsh at 40, before neovim at 50)
- `dependencies: [zsh]` — needs shell configured first
- `os: [macos, ubuntu, arch]`
- `requires: [curl]` — needed for the official installer
- `files:` — symlink `starship.toml` to `~/.config/starship.toml`
- `tags: [shell, prompt]`

### 2. `modules/starship/install.sh`
- Check if `starship` is already installed (`command -v starship`)
- If not, install via official installer: `curl -sS https://starship.rs/install.sh | sh -s -- -y`
- Respect `is_dry_run` checks
- Ensure `~/.config` directory exists

### 3. `modules/starship/starship.toml`
- Minimal custom config: format string, directory truncation, git branch/status, language indicators (Go, Python, Node, Rust), command duration, newline

### 4. `modules/starship/verify.sh`
- Verify `starship` binary is on PATH

## Files to Modify

### 5. `modules/zsh/zshrc` (line ~108, before `.zshrc.local` source)
- Add Starship init block:
  ```
  # Starship prompt
  if command -v starship &>/dev/null; then
      eval "$(starship init zsh)"
  fi
  ```
- Guarded with `command -v` so zshrc still works without starship installed

### 6. `profiles/developer.yml`
- Add `starship` to the modules list (after `zsh`, before `neovim`)

## Verification
- `export PATH="/usr/local/go/bin:$PATH" && go build ./...` — confirm project still compiles
- `dotfiles install --dry-run starship` — verify module is discovered and plan is correct
