#!/usr/bin/env bash
# zoxide/install.sh - Install zoxide (smarter cd)

if command -v zoxide &>/dev/null; then
    log_info "zoxide is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install zoxide"
    return 0
fi

log_info "Installing zoxide..."
pkg_install zoxide || {
    # Fallback: official install script
    log_info "Package manager install failed, trying official script..."
    curl -sS https://raw.githubusercontent.com/ajeetdsouza/zoxide/main/install.sh | bash
}
log_success "zoxide installed"
