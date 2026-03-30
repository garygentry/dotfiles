#!/usr/bin/env bash
# claude-code/verify.sh - Verify Claude Code installation

_claude_bin=""
if command -v claude &>/dev/null; then
    _claude_bin="claude"
elif [[ -x "${HOME}/.local/bin/claude" ]]; then
    _claude_bin="${HOME}/.local/bin/claude"
fi

if [[ -n "$_claude_bin" ]]; then
    _claude_version="$("$_claude_bin" --version 2>/dev/null)"
    log_success "Claude Code is installed: ${_claude_version}"
else
    log_error "Claude Code is not installed"
    return 1
fi
