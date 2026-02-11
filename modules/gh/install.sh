#!/usr/bin/env bash
# gh/install.sh - Install GitHub CLI

if command -v gh &>/dev/null; then
    log_info "GitHub CLI is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install GitHub CLI"
    return 0
fi

log_info "Installing GitHub CLI..."

case "${DOTFILES_PKG_MGR}" in
    pacman)
        pkg_install github-cli
        ;;
    *)
        pkg_install gh
        ;;
esac

log_success "GitHub CLI installed"
