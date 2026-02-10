#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Docker..."

if command -v docker >/dev/null 2>&1; then
    log_info "Docker already installed"
    exit 0
fi

# Install Docker
pkg_install docker || pkg_install docker.io

# Add user to docker group
sudo usermod -aG docker "$USER"

log_warn "You may need to log out and back in for docker group membership to take effect"
log_success "Docker installed"
