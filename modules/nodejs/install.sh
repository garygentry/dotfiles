#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Node.js..."

if command -v node >/dev/null 2>&1; then
    log_info "Node.js already installed: $(node --version)"
    exit 0
fi

# Install Node.js
pkg_install nodejs npm

log_success "Node.js installed"
