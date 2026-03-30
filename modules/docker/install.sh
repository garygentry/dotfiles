#!/usr/bin/env bash
set -euo pipefail

log_info "Installing Docker..."

if command -v docker >/dev/null 2>&1; then
    log_info "Docker already installed"
else
    # Package name varies by distro: docker.io on Ubuntu/Debian, docker on Arch/macOS
    if is_ubuntu; then
        pkg_install docker.io
    else
        pkg_install docker
    fi
fi

# Add user to docker group if the group exists
if getent group docker >/dev/null 2>&1; then
    if ! groups "$USER" 2>/dev/null | grep -q docker; then
        sudo_cmd usermod -aG docker "$USER"
        log_warn "You may need to log out and back in for docker group membership to take effect"
    fi
else
    log_warn "docker group does not exist — docker may need to be started first"
fi

log_success "Docker installed"
