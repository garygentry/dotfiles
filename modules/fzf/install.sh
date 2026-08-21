#!/usr/bin/env bash
set -euo pipefail

log_info "Installing fzf..."

if command -v fzf >/dev/null 2>&1; then
    log_info "fzf already installed"
    exit 0
fi

# Try package manager first, fall back to a git checkout.
if pkg_install fzf; then
    log_success "fzf installed via package manager"
else
    log_info "Package manager install failed, installing from git..."
    if [[ -d "$HOME/.fzf/.git" ]]; then
        git -C "$HOME/.fzf" pull --ff-only || true
    else
        rm -rf "$HOME/.fzf"
        git clone --depth 1 https://github.com/junegunn/fzf.git "$HOME/.fzf"
    fi
    "$HOME/.fzf/install" --all --no-update-rc
    log_success "fzf installed with shell integration"
fi
