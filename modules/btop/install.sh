#!/usr/bin/env bash
# btop/install.sh - Install btop system monitor

if command -v btop &>/dev/null; then
    log_info "btop is already installed"
    return 0
fi

if is_dry_run; then
    log_info "[dry-run] Would install btop"
    return 0
fi

log_info "Installing btop..."
pkg_install btop
log_success "btop installed"
