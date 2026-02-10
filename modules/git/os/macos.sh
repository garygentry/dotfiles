#!/usr/bin/env bash
# git/os/macos.sh - macOS-specific git setup

# macOS ships with git via Xcode CLI tools, but we may want a newer version
if ! command -v git &>/dev/null; then
    log_info "Git not found, installing via Homebrew..."
    pkg_install git
else
    log_info "Git is already installed on macOS"
fi

# Configure macOS-specific credential helper
if is_dry_run; then
    log_info "[dry-run] Would configure macOS git credential helper"
else
    git config --global credential.helper osxkeychain
    log_info "Configured macOS credential helper (osxkeychain)"
fi
