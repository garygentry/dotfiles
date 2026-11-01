#!/usr/bin/env bash
# git/os/macos.sh - macOS-specific git setup

# macOS ships with git via Xcode CLI tools, but we may want a newer version
if ! command -v git &>/dev/null; then
    log_info "Git not found, installing via Homebrew..."
    pkg_install git
else
    log_info "Git is already installed on macOS"
fi
