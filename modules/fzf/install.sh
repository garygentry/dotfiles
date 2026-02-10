#!/usr/bin/env bash
set -euo pipefail

log_info "Installing fzf..."

if command -v fzf >/dev/null 2>&1; then
    log_info "fzf already installed"
    exit 0
fi

# Install fzf
git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf
~/.fzf/install --key-bindings --completion --no-update-rc

log_success "fzf installed with shell integration"
