#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Go..."

if command -v go >/dev/null 2>&1; then
    log_info "Go already installed: $(go version)"
    exit 0
fi

# Install Go
pkg_install golang || pkg_install go

# Set up GOPATH
mkdir -p ~/go/{bin,src,pkg}

log_success "Go installed"
