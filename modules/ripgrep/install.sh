#!/usr/bin/env bash
set -euo pipefail

log_info "Installing ripgrep..."

if command -v rg >/dev/null 2>&1; then
    log_info "ripgrep is already installed"
    exit 0
fi

# Install via package manager
pkg_install ripgrep

log_success "ripgrep installed successfully"
