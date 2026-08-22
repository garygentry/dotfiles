#!/usr/bin/env bash
set -euo pipefail

log_info "Installing tmux..."

pkg_install tmux

# Install TPM (Tmux Plugin Manager) — generic, lets any config use `set -g @plugin`.
if [ ! -d ~/.tmux/plugins/tpm ]; then
    git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
    log_info "Installed TPM (Tmux Plugin Manager)"
else
    log_info "TPM already installed, skipping"
fi

# The generic engine ships no tmux.conf. Deploy your own from a content overlay
# (see the dotfiles-gwg tmux config module for an example: themed config + plugins).
log_success "tmux installed"
