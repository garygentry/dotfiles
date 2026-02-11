#!/usr/bin/env bash
# zellij/install.sh - Install Zellij terminal multiplexer

if command -v zellij &>/dev/null; then
    log_info "Zellij is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Zellij"
    return 0
fi

log_info "Installing Zellij..."
pkg_install zellij || {
    # Fallback: install via cargo if available
    if command -v cargo &>/dev/null; then
        log_info "Package manager install failed, trying cargo..."
        cargo install zellij
    else
        log_error "Failed to install Zellij (no package manager or cargo available)"
        return 1
    fi
}
log_success "Zellij installed"
