#!/usr/bin/env bash
set -euo pipefail

log_info "Installing lazygit..."

if command -v lazygit >/dev/null 2>&1; then
    log_info "lazygit already installed"
    exit 0
fi

pkg_install lazygit || {
    # Fallback to GitHub release
    LAZYGIT_VERSION=$(curl -s "https://api.github.com/repos/jesseduffield/lazygit/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')
    curl -Lo /tmp/lazygit.tar.gz "https://github.com/jesseduffield/lazygit/releases/latest/download/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz"
    tar xf /tmp/lazygit.tar.gz -C /tmp lazygit
    sudo install /tmp/lazygit /usr/local/bin
}

log_success "lazygit installed"
