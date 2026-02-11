#!/usr/bin/env bash
# ghostty/install.sh - Install Ghostty terminal emulator

if command -v ghostty &>/dev/null; then
    log_info "Ghostty is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Ghostty"
    return 0
fi

log_info "Installing Ghostty..."

case "${DOTFILES_PKG_MGR}" in
    apt)
        sudo snap install ghostty
        ;;
    *)
        pkg_install ghostty
        ;;
esac

log_success "Ghostty installed"
