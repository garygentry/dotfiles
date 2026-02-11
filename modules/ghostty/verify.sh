#!/usr/bin/env bash
# ghostty/verify.sh - Verify Ghostty installation

if command -v ghostty &>/dev/null; then
    _ghostty_version="$(ghostty --version 2>/dev/null)"
    log_success "Ghostty is installed: ${_ghostty_version}"
else
    log_error "Ghostty is not installed"
    return 1
fi
