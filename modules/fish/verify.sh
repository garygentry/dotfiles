#!/usr/bin/env bash
# fish/verify.sh - Verify Fish shell installation

if command -v fish &>/dev/null; then
    _fish_version="$(fish --version 2>/dev/null)"
    log_success "Fish shell is installed: ${_fish_version}"
else
    log_error "Fish shell is not installed"
    return 1
fi
