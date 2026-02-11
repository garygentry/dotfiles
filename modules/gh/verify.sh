#!/usr/bin/env bash
# gh/verify.sh - Verify GitHub CLI installation

if command -v gh &>/dev/null; then
    _gh_version="$(gh --version 2>/dev/null | head -1)"
    log_success "GitHub CLI is installed: ${_gh_version}"
else
    log_error "GitHub CLI is not installed"
    return 1
fi
