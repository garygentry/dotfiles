#!/usr/bin/env bash
# btop/verify.sh - Verify btop installation

if command -v btop &>/dev/null; then
    log_success "btop is installed"
else
    log_error "btop is not installed"
    return 1
fi
