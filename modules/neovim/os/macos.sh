#!/usr/bin/env bash
# neovim/os/macos.sh - Install Neovim on macOS

if command -v nvim &>/dev/null; then
    log_info "Neovim is already installed on macOS"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Neovim via: brew install neovim"
    return 0
fi

log_info "Installing Neovim via Homebrew..."
brew install neovim
log_success "Neovim installed on macOS"
