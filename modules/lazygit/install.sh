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
    LAZYGIT_ARCH="Linux_x86_64"
    [[ "${DOTFILES_ARCH:-}" == "arm64" ]] && LAZYGIT_ARCH="Linux_arm64"
    download_file \
        "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_${LAZYGIT_ARCH}.tar.gz" \
        "/tmp/lazygit.tar.gz"
    tar xf /tmp/lazygit.tar.gz -C /tmp lazygit
    sudo_cmd install /tmp/lazygit /usr/local/bin
}

log_success "lazygit installed"
