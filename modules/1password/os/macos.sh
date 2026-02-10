#!/usr/bin/env bash
# 1password/os/macos.sh - Install 1Password CLI on macOS via Homebrew

if command -v op &>/dev/null; then
    log_info "1Password CLI already installed on macOS"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install 1Password CLI via: brew install --cask 1password-cli"
    return 0
fi

log_info "Installing 1Password CLI via Homebrew..."
brew install --cask 1password-cli
log_success "1Password CLI installed on macOS"
