# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2026-02-11

### ⚠️ Breaking Changes

- **starship module**: Removed `zsh` dependency. Starship is correctly modeled as a cross-shell prompt that works with any shell (fish, bash, zsh, etc.). If you were relying on starship to auto-install zsh, explicitly add zsh to your profile or installation command.

### Added

- **Smart Prompt Behavior**: Only explicitly selected modules show interactive prompts. Auto-included dependencies use sensible defaults.
  - When you run `dotfiles install gh`, only `gh` shows prompts
  - Dependencies (`git`, `ssh`) use their default values automatically
  - Reduces friction and confusion during installation

- **`--prompt-dependencies` flag**: Force interactive prompts for all modules, including auto-included dependencies
  - Use when you want to configure dependency modules during installation
  - Example: `dotfiles install gh --prompt-dependencies` will prompt for gh, git, and ssh configurations

- **`show_when` prompt field**: Module authors can now control when prompts are shown
  - `explicit_install`: Only show when module is explicitly selected (default)
  - `always`: Always show, even for auto-included dependencies
  - `interactive`: Always show in interactive mode
  - Added to zsh module prompts (framework, plugins, theme)

### Changed

- **Prompt behavior**: Modules installed as dependencies (not explicitly requested) now use default values for all prompts instead of showing interactive prompts
- **Dependency tracking**: ExecutionPlan now tracks which modules were explicitly requested vs auto-included
- **Verbose logging**: With `-v` flag, you can see which defaults are being used for auto-included modules

### Technical Details

- Added `ExplicitlyRequested map[string]bool` to `ExecutionPlan` in resolver
- Added `ExplicitModules` and `PromptDependencies` fields to `RunConfig`
- Implemented `shouldShowPrompt()` logic in runner to filter prompts based on context
- Updated `handlePrompts()` to respect explicit vs auto-included distinction

### Migration Guide

**If you use starship with fish/bash:**
- No action needed! Starship will install without pulling in zsh

**If you relied on starship to install zsh:**
- Add `zsh` explicitly to your profile or install command
- Before: `dotfiles install starship` (installed both starship and zsh)
- After: `dotfiles install starship zsh` (explicit selection)

**If you want to configure auto-included dependencies:**
- Use the new `--prompt-dependencies` flag
- Example: `dotfiles install neovim --prompt-dependencies` will prompt for neovim, git, and ssh

## [1.1.0] - 2026-02-10

### Added
- Comprehensive idempotence system with update detection
- Post-install notes system for modules
- 12 new modules: docker, rust, awscli, azure-cli, gcloud, claude-code, gemini-cli, ghostty, tmux, zellij, zoxide, btop
- Oh My Zsh as alternative plugin framework for zsh module

### Changed
- Improved git-delta module to handle sudo prompts
- Enhanced state tracking with operation history

### Fixed
- Fixed git-delta sudo prompt issues

## [1.0.0] - 2026-02-09

### Added
- Initial release with core dotfiles management system
- Go-based CLI with shell module execution
- Dependency resolution using Kahn's algorithm
- State tracking and idempotent operations
- Template rendering with Go templates
- 1Password secrets integration
- Core modules: 1password, ssh, git, zsh, neovim, fonts, fzf, ripgrep, lazygit, gh, fish, starship, python, golang, nodejs
- Profile system for module sets
- Interactive and unattended modes
- Comprehensive test suite

[2.0.0]: https://github.com/garygentry/dotfiles/compare/v1.1.0...v2.0.0
[1.1.0]: https://github.com/garygentry/dotfiles/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/garygentry/dotfiles/releases/tag/v1.0.0
