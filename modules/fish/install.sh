#!/usr/bin/env bash
# fish/install.sh - Install Fish shell

if command -v fish &>/dev/null; then
    log_info "Fish shell is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install Fish shell"
    return 0
fi

log_info "Installing Fish shell..."
pkg_install fish

log_success "Fish shell installed"
