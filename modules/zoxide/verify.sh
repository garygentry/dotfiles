#!/usr/bin/env bash
# zoxide/verify.sh - Verify zoxide installation

if command -v zoxide &>/dev/null; then
    _zoxide_version="$(zoxide --version 2>/dev/null)"
    log_success "zoxide is installed: ${_zoxide_version}"
else
    log_error "zoxide is not installed"
    return 1
fi
