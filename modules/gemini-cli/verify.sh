#!/usr/bin/env bash
# gemini-cli/verify.sh - Verify Gemini CLI installation

if command -v gemini &>/dev/null; then
    _gemini_version="$(gemini --version 2>/dev/null)"
    log_success "Gemini CLI is installed: ${_gemini_version}"
else
    log_error "Gemini CLI is not installed"
    return 1
fi
