#!/usr/bin/env bash
# btop/verify.sh - Verify btop installation

if command -v btop &>/dev/null; then
    log_success "btop is installed"
elif ! sudo_usable; then
    log_warn "btop not installed — skipped (no usable sudo); not a failure"
else
    log_error "btop is not installed"
    return 1
fi
