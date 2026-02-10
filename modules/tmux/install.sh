#!/usr/bin/env bash
set -euo pipefail

log_info "Installing tmux..."

pkg_install tmux

# Install TPM (Tmux Plugin Manager)
if [ ! -d ~/.tmux/plugins/tpm ]; then
    git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
    log_info "Installed TPM (Tmux Plugin Manager)"
fi

log_success "tmux installed"
