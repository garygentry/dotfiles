#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Python..."

if command -v python3 >/dev/null 2>&1; then
    log_info "Python already installed: $(python3 --version)"
    exit 0
fi

# Install Python
pkg_install python3 python3-pip

log_success "Python installed"
