# Plan: Create `gwg-unattended` Profile

## Context
Create a new profile for unattended installs with a curated set of modules: starship, tmux, docker, fzf, neovim, btop, git, claude-code, and nodejs.

## Changes

**Create** `profiles/gwg-unattended.yml` with the requested modules listed under `modules:`. The dependency resolver (Kahn's algorithm in `internal/module/resolve.go`) will automatically pull in transitive dependencies (`ssh` via `git`, `ripgrep` via `fzf`), so only the explicitly requested modules need to be listed.

```yaml
modules:
  - git
  - starship
  - tmux
  - docker
  - fzf
  - neovim
  - btop
  - claude-code
  - nodejs
```

`git` is listed first since other modules depend on it; remaining modules in logical order.

## Verification
```bash
make build && bin/dotfiles install --profile gwg-unattended --unattended --dry-run
```
