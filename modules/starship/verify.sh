#!/usr/bin/env bash
# starship/verify.sh - Verify Starship installation

if ! command -v starship &>/dev/null; then
    log_error "Starship is not installed"
    exit 1
fi

log_success "Starship is installed ($(starship --version))"
