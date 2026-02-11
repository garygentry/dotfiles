#!/usr/bin/env bash
# zellij/verify.sh - Verify Zellij installation

if command -v zellij &>/dev/null; then
    _zellij_version="$(zellij --version 2>/dev/null)"
    log_success "Zellij is installed: ${_zellij_version}"
else
    log_error "Zellij is not installed"
    return 1
fi
