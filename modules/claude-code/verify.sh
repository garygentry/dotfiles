#!/usr/bin/env bash
# claude-code/verify.sh - Verify Claude Code installation

if command -v claude &>/dev/null; then
    _claude_version="$(claude --version 2>/dev/null)"
    log_success "Claude Code is installed: ${_claude_version}"
else
    log_error "Claude Code is not installed"
    return 1
fi
